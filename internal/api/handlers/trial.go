package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/vortanixapp/panel/internal/api/jobwake"
	"github.com/vortanixapp/panel/pkg/gamecatalog"
	"github.com/vortanixapp/panel/pkg/notify"
	"github.com/vortanixapp/panel/pkg/portalloc"
)

type trialSettings struct {
	Enabled      bool
	Hours        int
	CooldownDays int
	Games        []string
	MemoryMB     int
	DiskMB       int
}

func (h *Handler) trialSettings(ctx context.Context, tenantID string) trialSettings {
	s := trialSettings{
		Hours:        intSetting(h.tenantSettingString(ctx, tenantID, "trial.hours"), 2),
		CooldownDays: intSetting(h.tenantSettingString(ctx, tenantID, "trial.cooldown_days"), 30),
		MemoryMB:     intSetting(h.tenantSettingString(ctx, tenantID, "trial.memory_mb"), 0),
		DiskMB:       intSetting(h.tenantSettingString(ctx, tenantID, "trial.disk_mb"), 0),
	}
	s.Enabled = truthySetting(h.tenantSettingString(ctx, tenantID, "trial.enabled"))
	for _, g := range strings.Split(h.tenantSettingString(ctx, tenantID, "trial.games"), ",") {
		if g = strings.TrimSpace(g); g != "" {
			s.Games = append(s.Games, strings.ToLower(g))
		}
	}
	if s.Hours < 1 {
		s.Hours = 1
	}
	if s.Hours > 168 {
		s.Hours = 168
	}
	return s
}

func intSetting(raw string, def int) int {
	if n, err := strconv.Atoi(strings.TrimSpace(raw)); err == nil && n > 0 {
		return n
	}
	return def
}

func truthySetting(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

func (h *Handler) trialAvailableAt(ctx context.Context, tenantID, userID string, cooldownDays int) time.Time {
	var last time.Time
	err := h.readerOf(ctx).QueryRow(ctx, `
		SELECT created_at FROM core.trial_grants
		WHERE tenant_id = $1 AND user_id = $2
		ORDER BY created_at DESC LIMIT 1
	`, tenantID, userID).Scan(&last)
	if err != nil {
		return time.Time{}
	}
	return last.Add(time.Duration(cooldownDays) * 24 * time.Hour)
}

func (h *Handler) TrialStatus(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ctx := r.Context()
	cfg := h.trialSettings(ctx, claims.TenantID)

	available := cfg.Enabled
	var reason string
	var nextAt any
	if !cfg.Enabled {
		reason = "пробный сервер сейчас не выдаётся"
	} else {
		if at := h.trialAvailableAt(ctx, claims.TenantID, claims.UserID, cfg.CooldownDays); !at.IsZero() {
			if at.After(time.Now()) {
				available = false
				reason = "пробный сервер уже выдавался"
				nextAt = at.Format(time.RFC3339)
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"available":     available,
		"reason":        reason,
		"hours":         cfg.Hours,
		"games":         cfg.Games,
		"cooldown_days": cfg.CooldownDays,
		"next_at":       nextAt,
	})
}

type trialCreateBody struct {
	GameID string `json:"game_id"`
	NodeID string `json:"node_id"`
	Name   string `json:"name"`
}

func (h *Handler) TrialCreate(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var body trialCreateBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	ctx := r.Context()
	cfg := h.trialSettings(ctx, claims.TenantID)
	if !cfg.Enabled {
		writeError(w, http.StatusConflict, "пробный сервер сейчас не выдаётся")
		return
	}
	if at := h.trialAvailableAt(ctx, claims.TenantID, claims.UserID, cfg.CooldownDays); at.After(time.Now()) {
		writeError(w, http.StatusConflict,
			"пробный сервер уже выдавался, следующий будет доступен "+at.Format("02.01.2006"))
		return
	}

	gameID, gameOK := resolveGameSlug(ctx, h, claims.TenantID, body.GameID)
	if !gameOK {
		writeError(w, http.StatusBadRequest, "игра не найдена в каталоге")
		return
	}
	if len(cfg.Games) > 0 && !containsFold(cfg.Games, gameID) {
		writeError(w, http.StatusConflict, "на этой игре пробный сервер не выдаётся")
		return
	}

	nodeID := strings.TrimSpace(body.NodeID)
	if nodeID == "" {
		var err error
		nodeID, err = h.pickTrialNode(ctx, claims.TenantID)
		if err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
	}
	if h.nodeInMaintenance(ctx, claims.TenantID, nodeID) {
		writeError(w, http.StatusConflict, "на этой локации идут технические работы")
		return
	}
	if reason := h.nodeCapacityReason(ctx, claims.TenantID, nodeID, gameID); reason != "" {
		writeError(w, http.StatusConflict, reason)
		return
	}

	limits := gamecatalog.DefaultLimits(gameID)
	if cfg.MemoryMB > 0 {
		limits["memory_mb"] = cfg.MemoryMB
	}
	if cfg.DiskMB > 0 {
		limits["disk_mb"] = cfg.DiskMB
	}
	limitsJSON, _ := json.Marshal(limits)

	name := strings.TrimSpace(body.Name)
	if name == "" {
		name = "Пробный " + gameID
	}
	expires := time.Now().Add(time.Duration(cfg.Hours) * time.Hour)

	var serverID string
	err := h.dbOf(ctx).QueryRow(ctx, `
		INSERT INTO core.servers
			(tenant_id, node_id, game_id, name, limits, user_id, provisioning_status, expires_at, is_trial, rental_period_days, config)
		VALUES ($1, $2::uuid, $3, $4, $5::jsonb, $6::uuid, 'provisioning', $7, true, 1,
		        jsonb_build_object('startup_params', $8::text))
		RETURNING id::text
	`, claims.TenantID, nodeID, gameID, name, limitsJSON, claims.UserID, expires,
		h.defaultStartupParams(ctx, claims.TenantID, gameID)).Scan(&serverID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "не удалось создать пробный сервер")
		return
	}

	_, _ = h.dbOf(ctx).Exec(ctx, `
		INSERT INTO core.trial_grants (tenant_id, user_id, server_id, game_id, hours)
		VALUES ($1, $2::uuid, $3::uuid, $4, $5)
	`, claims.TenantID, claims.UserID, serverID, gameID, cfg.Hours)

	_, _ = h.dbOf(ctx).Exec(ctx, `
		UPDATE core.servers s SET ip_address = n.fqdn
		FROM core.nodes n WHERE s.id = $1 AND n.id = s.node_id
	`, serverID)
	if gameID != "test" {
		if _, err := portalloc.Assign(ctx, h.dbOf(ctx), claims.TenantID, nodeID, serverID, gameID); err != nil {
			writeError(w, http.StatusConflict, "на локации нет свободного порта для этой игры")
			return
		}
	}

	_, _ = h.dbOf(ctx).Exec(ctx, `
		INSERT INTO core.jobs (tenant_id, type, status, payload)
		VALUES ($1, 'provision_server', 'pending', $2::jsonb)
	`, claims.TenantID, mustJSON(map[string]string{"server_id": serverID}))
	jobwake.Notify("provision_server")

	h.notifyUser(ctx, claims.TenantID, claims.UserID, notify.Event{
		Kind:  notify.KindServerReady,
		Title: "Пробный сервер запускается",
		Body: fmt.Sprintf("Сервер «%s» работает %d ч. После этого он остановится — продлите аренду, чтобы оставить его себе.",
			name, cfg.Hours),
		Action: h.serverAction("Открыть сервер", serverID, ""),
		Meta:   map[string]any{"server_id": serverID},
	})
	audit(ctx, h.dbOf(ctx), claims.TenantID, claims.UserID, "server.trial", "server:"+serverID,
		map[string]any{"game": gameID, "hours": cfg.Hours})

	writeJSON(w, http.StatusCreated, map[string]any{
		"server_id":  serverID,
		"expires_at": expires.Format(time.RFC3339),
		"hours":      cfg.Hours,
	})
}

func (h *Handler) pickTrialNode(ctx context.Context, tenantID string) (string, error) {
	rows, err := h.readerOf(ctx).Query(ctx, `
		SELECT n.id::text
		FROM core.nodes n
		WHERE n.tenant_id = $1
		  AND COALESCE(n.is_active, n.active, true) = true
		  AND COALESCE(n.maintenance_mode, false) = false
		  AND n.status = 'online'
		ORDER BY (SELECT COUNT(*) FROM core.servers s WHERE s.node_id = n.id) ASC
		LIMIT 5
	`, tenantID)
	if err != nil {
		return "", fmt.Errorf("нет доступных локаций")
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			return id, nil
		}
	}
	return "", fmt.Errorf("нет доступных локаций")
}

func containsFold(list []string, value string) bool {
	for _, item := range list {
		if strings.EqualFold(item, value) {
			return true
		}
	}
	return false
}
