package handlers

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
)

type serverGameSettingsRow struct {
	GameID      string
	Config      map[string]any
	Limits      map[string]any
	BillingType string
	Slots       int
}

func (h *Handler) loadServerGameSettingsRow(ctx context.Context, tenantID, serverID string) (*serverGameSettingsRow, error) {
	var gameID string
	var configRaw, limitsRaw []byte
	var billingType *string
	err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT s.game_id, COALESCE(s.config, '{}'::jsonb), COALESCE(s.limits, '{}'::jsonb),
		       t.billing_type
		FROM core.servers s
		LEFT JOIN core.tariffs t ON t.id = s.tariff_id
		WHERE s.id = $1 AND s.tenant_id = $2
	`, serverID, tenantID).Scan(&gameID, &configRaw, &limitsRaw, &billingType)
	if err != nil {
		return nil, err
	}
	row := &serverGameSettingsRow{GameID: strings.ToLower(gameID)}
	_ = json.Unmarshal(configRaw, &row.Config)
	_ = json.Unmarshal(limitsRaw, &row.Limits)
	if billingType != nil {
		row.BillingType = *billingType
	}
	row.Slots = serverSlotsFromLimits(row.Limits)
	return row, nil
}

func extractStoredGameSettings(config map[string]any, profileKey string) map[string]any {
	if config == nil {
		return map[string]any{}
	}
	gs, ok := config["game_settings"].(map[string]any)
	if !ok || gs == nil {
		return map[string]any{}
	}
	if stored, ok := gs[profileKey].(map[string]any); ok && stored != nil {
		return stored
	}
	return map[string]any{}
}

func (h *Handler) mergeGameSettingsConfig(ctx context.Context, tenantID, serverID, profileKey string, settings map[string]any) error {
	raw, err := json.Marshal(settings)
	if err != nil {
		return err
	}
	_, err = h.dbOf(ctx).Exec(ctx, `
		UPDATE core.servers
		SET config = jsonb_set(
			COALESCE(config, '{}'::jsonb),
			ARRAY['game_settings', $3::text],
			COALESCE(config->'game_settings'->$3::text, '{}'::jsonb) || $4::jsonb,
			true
		)
		WHERE id = $1 AND tenant_id = $2
	`, serverID, tenantID, profileKey, raw)
	return err
}

func (h *Handler) updateServerStartupParamsString(ctx context.Context, tenantID, serverID, params string) error {
	_, err := h.dbOf(ctx).Exec(ctx, `
		UPDATE core.servers
		SET config = config || jsonb_build_object('startup_params', to_jsonb($3::text))
		WHERE id = $1 AND tenant_id = $2
	`, serverID, tenantID, params)
	return err
}

func effectiveStartupParams(config map[string]any) string {
	if config == nil {
		return ""
	}
	v, ok := config["startup_params"]
	if !ok || v == nil {
		return ""
	}
	if s, isString := v.(string); isString {
		return s
	}
	return strings.TrimSpace(toString(v))
}

func serverSlotsFromLimits(limits map[string]any) int {
	if limits == nil {
		return 0
	}
	switch n := limits["slots"].(type) {
	case float64:
		return int(n)
	case int:
		return n
	case json.Number:
		if i, err := n.Int64(); err == nil {
			return int(i)
		}
	case string:
		if i, err := strconv.Atoi(strings.TrimSpace(n)); err == nil {
			return i
		}
	}
	return 0
}

func (h *Handler) agentFilesReadQuiet(ctx context.Context, tenantID, serverID, path string) (string, bool) {
	nodeID, err := h.serverNodeID(ctx, tenantID, serverID)
	if err != nil {
		return "", false
	}
	result, err := h.agentCommand(ctx, nodeID, serverID, "files_read", map[string]any{"path": path})
	if err != nil || !agentResultOK(result) {
		return "", false
	}
	return toString(result["content"]), true
}

func (h *Handler) agentFilesWriteQuiet(ctx context.Context, tenantID, serverID, path, content string) bool {
	nodeID, err := h.serverNodeID(ctx, tenantID, serverID)
	if err != nil {
		return false
	}
	result, err := h.agentCommand(ctx, nodeID, serverID, "files_write", map[string]any{
		"path":    path,
		"content": content,
	})
	return err == nil && agentResultOK(result)
}

func agentResultOK(result map[string]any) bool {
	if result == nil {
		return false
	}
	if okVal, exists := result["ok"]; exists {
		if b, ok := okVal.(bool); ok && !b {
			return false
		}
	}
	return true
}
