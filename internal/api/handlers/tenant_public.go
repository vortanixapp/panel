package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

func (h *Handler) publicMonitoringToken(ctx context.Context) string {
	var raw []byte
	err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT value FROM core.tenant_settings
		WHERE key = 'monitoring.public_token'
	`).Scan(&raw)
	if err != nil {
		return ""
	}
	var stored string
	if json.Unmarshal(raw, &stored) != nil {
		return ""
	}
	return stored
}

func (h *Handler) publicRequestAllowed(ctx context.Context, r *http.Request) bool {
	stored := h.publicMonitoringToken(ctx)
	if stored == "" {
		return true
	}
	return strings.TrimSpace(r.URL.Query().Get("token")) == stored
}
