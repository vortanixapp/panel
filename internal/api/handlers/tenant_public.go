package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

func (h *Handler) resolvePublicTenantID(ctx context.Context, r *http.Request) (tenantID string, ok bool) {
	if token := strings.TrimSpace(r.URL.Query().Get("token")); token != "" {
		if id, found := h.tenantIDByPublicToken(ctx, token); found {
			return id, true
		}
	}

	slug := strings.TrimSpace(r.Header.Get("X-Tenant-Slug"))
	if slug == "" {
		slug = strings.TrimSpace(r.URL.Query().Get("tenant"))
	}
	if slug == "" {
		slug = tenantSlugFromHost(r.Host)
	}
	if slug == "" {
		slug = envOr("DEFAULT_TENANT_SLUG", "dev")
	}

	err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT id::text FROM core.tenants WHERE slug = $1 AND status = 'active'
	`, slug).Scan(&tenantID)
	return tenantID, err == nil
}

func (h *Handler) tenantIDByPublicToken(ctx context.Context, token string) (string, bool) {
	var tenantID string
	var raw []byte
	rows, err := h.dbOf(ctx).Query(ctx, `
		SELECT tenant_id::text, value FROM core.tenant_settings
		WHERE key = 'monitoring.public_token'
	`)
	if err != nil {
		return "", false
	}
	defer rows.Close()
	for rows.Next() {
		if rows.Scan(&tenantID, &raw) != nil {
			continue
		}
		var stored string
		if json.Unmarshal(raw, &stored) == nil && stored == token {
			return tenantID, true
		}
	}
	return "", false
}

func tenantSlugFromHost(host string) string {
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "" {
		return ""
	}
	if i := strings.Index(host, ":"); i >= 0 {
		host = host[:i]
	}
	if host == "localhost" || host == "127.0.0.1" {
		return ""
	}
	parts := strings.Split(host, ".")
	if len(parts) >= 3 {
		return parts[0]
	}
	return ""
}

func lookupServerTenant(ctx context.Context, db *pgxpool.Pool, serverID string) (string, bool) {
	var tenantID string
	err := db.QueryRow(ctx, `SELECT tenant_id::text FROM core.servers WHERE id = $1`, serverID).Scan(&tenantID)
	return tenantID, err == nil
}
