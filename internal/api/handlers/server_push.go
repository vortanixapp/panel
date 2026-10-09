package handlers

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/vortanixapp/panel/pkg/protocol"
)

const (
	liveServerPrefix  = "srv:"
	livePlayersPrefix = "plr:"
	serverPushGap     = 3 * time.Second
	playersPushGap    = 10 * time.Second
	playersPushWait   = 12 * time.Second
	serverAudienceTTL = time.Minute
	serverPushPrune   = 5 * time.Minute
)

type serverPushPayload struct {
	ServerID   string  `json:"server_id"`
	CPU        float64 `json:"cpu_pct"`
	MemUsedMB  int     `json:"mem_used_mb"`
	MemLimitMB int     `json:"mem_limit_mb"`
	TS         int64   `json:"ts"`
}

type playersPushPayload struct {
	ServerID      string `json:"server_id"`
	Online        int    `json:"online_players"`
	Max           int    `json:"max_players"`
	PlayersOnline any    `json:"players_online"`
	CurrentMap    string `json:"current_map"`
}

type serverAudience struct {
	at  time.Time
	ids []string
}

func metricNumber(m map[string]any, key string) float64 {
	switch v := m[key].(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	}
	return 0
}

func (l *notifyLive) active() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.subs) > 0
}

func (l *notifyLive) connected(userIDs []string) []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	var out []string
	for _, id := range userIDs {
		if len(l.subs[id]) > 0 {
			out = append(out, id)
		}
	}
	return out
}

func (h *Handler) StartServerPush(ctx context.Context) {
	if h.live == nil || h.cache == nil {
		return
	}
	go h.runServerPush(ctx)
}

func (h *Handler) runServerPush(ctx context.Context) {
	events, stop := h.cache.Subscribe(ctx, protocol.TenantEventsChannel())
	defer stop()

	sentAt := map[string]time.Time{}
	playersAt := map[string]time.Time{}
	audiences := map[string]serverAudience{}
	var playersBusy sync.Map
	prune := time.NewTicker(serverPushPrune)
	defer prune.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case now := <-prune.C:
			for id, at := range sentAt {
				if now.Sub(at) > serverPushPrune {
					delete(sentAt, id)
					delete(playersAt, id)
				}
			}
			for id, a := range audiences {
				if now.Sub(a.at) > serverAudienceTTL {
					delete(audiences, id)
				}
			}
		case raw, ok := <-events:
			if !ok {
				return
			}
			var ev protocol.TenantEvent
			if json.Unmarshal([]byte(raw), &ev) != nil || ev.Type != "server.metrics" || ev.ServerID == "" {
				continue
			}
			if !h.live.active() {
				continue
			}
			now := time.Now()
			if now.Sub(sentAt[ev.ServerID]) < serverPushGap {
				continue
			}

			audience, cached := audiences[ev.ServerID]
			if !cached || now.Sub(audience.at) > serverAudienceTTL {
				audience = serverAudience{at: now, ids: h.serverAudienceIDs(ctx, ev.ServerID)}
				audiences[ev.ServerID] = audience
			}
			viewers := h.live.connected(audience.ids)
			if len(viewers) == 0 {
				continue
			}
			sentAt[ev.ServerID] = now

			body, err := json.Marshal(serverPushPayload{
				ServerID:   ev.ServerID,
				CPU:        metricNumber(ev.Metrics, "cpu_pct"),
				MemUsedMB:  int(metricNumber(ev.Metrics, "mem_used_mb")),
				MemLimitMB: int(metricNumber(ev.Metrics, "mem_limit_mb")),
				TS:         now.Unix(),
			})
			if err == nil {
				msg := liveServerPrefix + string(body)
				for _, userID := range viewers {
					h.live.deliver(userID, msg)
				}
			}

			if now.Sub(playersAt[ev.ServerID]) < playersPushGap {
				continue
			}
			if _, busy := playersBusy.LoadOrStore(ev.ServerID, struct{}{}); busy {
				continue
			}
			playersAt[ev.ServerID] = now
			serverID := ev.ServerID
			go func() {
				defer playersBusy.Delete(serverID)
				pctx, cancel := context.WithTimeout(ctx, playersPushWait)
				defer cancel()
				h.pushPlayers(pctx, serverID, viewers)
			}()
		}
	}
}

func (h *Handler) pushPlayers(ctx context.Context, serverID string, viewers []string) {
	var status, runtime, gameID, billingType string
	var port int
	var limitsRaw, configRaw []byte
	err := h.reader().QueryRow(ctx, `
		SELECT COALESCE(s.status, ''), COALESCE(s.runtime_status, ''), COALESCE(s.game_id, ''),
		       COALESCE(s.primary_port, 0), COALESCE(s.limits, '{}'::jsonb),
		       COALESCE(s.config, '{}'::jsonb), COALESCE(t.billing_type, '')
		FROM core.servers s
		LEFT JOIN core.tariffs t ON t.id = s.tariff_id
		WHERE s.id = $1::uuid
	`, serverID).Scan(&status, &runtime, &gameID, &port, &limitsRaw, &configRaw, &billingType)
	if err != nil || gameID == "" || (runtime != "running" && status != "running") {
		return
	}
	nodeID, err := h.serverNodeID(ctx, serverID)
	if err != nil || nodeID == "" {
		return
	}

	limits := map[string]any{}
	_ = json.Unmarshal(limitsRaw, &limits)
	config := map[string]any{}
	_ = json.Unmarshal(configRaw, &config)

	paidSlots := 0
	if billingType == "slots" {
		paidSlots = limitsInt(limits, "slots")
	}
	baseSlots := paidSlots
	if baseSlots <= 0 {
		baseSlots = configuredSlots(gameID, config)
	}
	if baseSlots > 0 && limitsInt(limits, "slots") <= 0 {
		limits["slots"] = baseSlots
	}

	snap := h.sharedAgentStatus(ctx, nodeID, serverID, gameID, limits, port, false)
	result := snap.query
	if snap.queryErr != nil || result == nil {
		return
	}
	if answered, has := result["answered"].(bool); has && !answered {
		return
	}

	maxPlayers := intFromAny(result["max_players"])
	if paidSlots > 0 {
		maxPlayers = paidSlots
	} else if maxPlayers <= 0 {
		maxPlayers = baseSlots
	}
	list := result["players_online"]
	if list == nil {
		list = []any{}
	}
	current, _ := result["current_map"].(string)
	body, err := json.Marshal(playersPushPayload{
		ServerID:      serverID,
		Online:        intFromAny(result["online_players"]),
		Max:           maxPlayers,
		PlayersOnline: list,
		CurrentMap:    current,
	})
	if err != nil {
		return
	}
	msg := livePlayersPrefix + string(body)
	for _, userID := range viewers {
		h.live.deliver(userID, msg)
	}
}

func (h *Handler) serverAudienceIDs(ctx context.Context, serverID string) []string {
	qctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	rows, err := h.reader().Query(qctx, `
		SELECT user_id::text FROM core.servers WHERE id = $1::uuid AND user_id IS NOT NULL
		UNION
		SELECT user_id::text FROM core.server_friends WHERE server_id = $1::uuid
	`, serverID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil && id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}
