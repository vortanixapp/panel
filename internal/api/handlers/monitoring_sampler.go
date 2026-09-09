package handlers

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"
)

const (
	monitoringSampleInterval = time.Minute
	monitoringSampleWorkers  = 8
	monitoringFailThreshold  = 3
	monitoringRetentionDays  = 45
)

type samplerTarget struct {
	ServerID string
	TenantID string
	NodeID   string
	GameID   string
	Name     string
	Port     int
	Slots    int
	Status   string
}

func (h *Handler) StartMonitoringSampler(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(monitoringSampleInterval)
		defer ticker.Stop()
		cleanup := time.NewTicker(6 * time.Hour)
		defer cleanup.Stop()

		h.sampleMonitoringOnce(ctx)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				h.sampleMonitoringOnce(ctx)
			case <-cleanup.C:
				h.purgeOnlinePoints(ctx)
			}
		}
	}()
}

func (h *Handler) sampleMonitoringOnce(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, monitoringSampleInterval)
	defer cancel()

	targets, err := h.monitoringSampleTargets(ctx)
	if err != nil {
		log.Printf("monitoring sampler: targets: %v", err)
		return
	}
	if len(targets) == 0 {
		return
	}

	queue := make(chan samplerTarget)
	var wg sync.WaitGroup
	for i := 0; i < monitoringSampleWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for t := range queue {
				h.sampleServer(ctx, t)
			}
		}()
	}
	for _, t := range targets {
		select {
		case <-ctx.Done():
		case queue <- t:
		}
	}
	close(queue)
	wg.Wait()
}

func (h *Handler) monitoringSampleTargets(ctx context.Context) ([]samplerTarget, error) {
	rows, err := h.readerOf(ctx).Query(ctx, `
		SELECT s.id::text, s.tenant_id::text, s.node_id::text, s.game_id, s.name,
		       COALESCE(s.primary_port, 0), COALESCE(s.limits, '{}'::jsonb),
		       COALESCE(s.status, 'stopped'), COALESCE(s.runtime_status, '')
		FROM core.servers s
		WHERE COALESCE(s.provisioning_status, 'pending') = 'ready'
		  AND COALESCE(s.is_blocked, false) = false
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []samplerTarget{}
	for rows.Next() {
		var t samplerTarget
		var limitsRaw []byte
		var status, runtime string
		if rows.Scan(&t.ServerID, &t.TenantID, &t.NodeID, &t.GameID, &t.Name,
			&t.Port, &limitsRaw, &status, &runtime) != nil {
			continue
		}
		limits := map[string]any{}
		_ = json.Unmarshal(limitsRaw, &limits)
		t.Slots = limitsInt(limits, "slots")

		if live, ok := h.cache.GetServerStatus(ctx, t.ServerID); ok {
			t.Status = resolveEffectiveStatus(status, runtime, &live)
		} else {
			t.Status = resolveEffectiveStatus(status, runtime, nil)
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (h *Handler) sampleServer(ctx context.Context, t samplerTarget) {
	sample := onlineSample{}

	intentionallyDown := t.Status == "stopped" || t.Status == "stopping"

	if !intentionallyDown {
		res, err := h.agentCommand(ctx, t.NodeID, t.ServerID, "game_query", map[string]any{
			"game_id": t.GameID,
			"limits":  map[string]any{"slots": t.Slots},
			"port":    t.Port,
		})
		if err == nil && res != nil {
			sample.Up = toString(res["runtime_status"]) == "running"
			sample.Online = intFromAny(res["online_players"])
			sample.MaxPlayers = intFromAny(res["max_players"])
			sample.Ping = intFromAny(res["ping_ms"])
			sample.TPS = floatFromAny(res["tps"])
		}
	}
	if sample.MaxPlayers == 0 {
		sample.MaxPlayers = t.Slots
	}
	if !sample.Up {
		sample.Online, sample.Ping, sample.TPS = 0, 0, 0
	}

	if _, err := h.dbOf(ctx).Exec(ctx, `
		INSERT INTO core.server_online_points
		    (server_id, tenant_id, online, max_players, ping_ms, tps, up)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, t.ServerID, t.TenantID, sample.Online, sample.MaxPlayers,
		sample.Ping, sample.TPS, sample.Up); err != nil {
		return
	}

	if !intentionallyDown {
		h.trackMonitoringIncident(ctx, t, sample.Up)
	}
}

func (h *Handler) trackMonitoringIncident(ctx context.Context, t samplerTarget, up bool) {
	if up {
		_, _ = h.dbOf(ctx).Exec(ctx, `
			UPDATE core.server_incidents
			SET resolved_at = now()
			WHERE server_id = $1 AND auto = true AND resolved_at IS NULL
		`, t.ServerID)
		return
	}

	var recentFails int
	if h.readerOf(ctx).QueryRow(ctx, `
		SELECT COUNT(*) FROM (
		    SELECT up FROM core.server_online_points
		    WHERE server_id = $1
		    ORDER BY ts DESC LIMIT $2
		) recent
		WHERE NOT up
	`, t.ServerID, monitoringFailThreshold).Scan(&recentFails) != nil {
		return
	}
	if recentFails < monitoringFailThreshold {
		return
	}

	var openIncidents int
	if h.readerOf(ctx).QueryRow(ctx, `
		SELECT COUNT(*) FROM core.server_incidents
		WHERE server_id = $1 AND auto = true AND resolved_at IS NULL
	`, t.ServerID).Scan(&openIncidents) != nil || openIncidents > 0 {
		return
	}

	_, _ = h.dbOf(ctx).Exec(ctx, `
		INSERT INTO core.server_incidents
		    (server_id, tenant_id, title, level, body, auto)
		VALUES ($1, $2, $3, 'bad', $4, true)
	`, t.ServerID, t.TenantID, "Сервер не отвечает на запросы",
		"Агент не получил ответ от игрового сервера несколько раз подряд. "+
			"Инцидент закроется автоматически, когда сервер снова начнёт отвечать.")
}

func (h *Handler) purgeOnlinePoints(ctx context.Context) {
	_, _ = h.dbOf(ctx).Exec(ctx, `
		DELETE FROM core.server_online_points
		WHERE ts < now() - ($1::text || ' days')::interval
	`, monitoringRetentionDays)
}
