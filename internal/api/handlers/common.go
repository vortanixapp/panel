package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/vortanix/vortanix/internal/api/paneljwt"
)

type Services struct {
	DB *pgxpool.Pool
}

func tenantClaims(ctx context.Context) (*paneljwt.Claims, bool) {
	claims, ok := ctx.Value(authContextKey).(*paneljwt.Claims)
	return claims, ok
}

func isAdminRole(role string) bool {
	return role == "owner" || role == "admin"
}

func isStaffRole(role string) bool {
	return isAdminRole(role) || role == "support"
}

func requireAdmin(w http.ResponseWriter, claims *paneljwt.Claims) bool {
	if !isAdminRole(claims.Role) {
		writeError(w, http.StatusForbidden, "forbidden")
		return false
	}
	return true
}

func audit(ctx context.Context, db *pgxpool.Pool, tenantID, userID, action, resource string, meta map[string]any) {
	var metaJSON []byte
	if meta != nil {
		metaJSON, _ = json.Marshal(meta)
	}
	_, _ = db.Exec(ctx, `
		INSERT INTO core.audit_logs (tenant_id, user_id, action, resource, meta)
		VALUES ($1, $2, $3, $4, $5::jsonb)
	`, tenantID, nullableUUID(userID), action, resource, metaJSON)
}

func nullableUUID(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
