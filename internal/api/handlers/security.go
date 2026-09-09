package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

const (
	loginFailWindow    = 15 * time.Minute
	loginFailThreshold = 10
	loginAutoBlockFor  = 30 * time.Minute
)

func (h *Handler) recordLoginAttempt(ctx context.Context, r *http.Request, userID, email, reason string, success bool) {
	agent := r.Header.Get("User-Agent")
	if len(agent) > 300 {
		agent = agent[:300]
	}
	_, _ = h.dbOf(ctx).Exec(ctx, `
		INSERT INTO core.login_attempts (email, user_id, ip, user_agent, success, reason)
		VALUES ($1, NULLIF($2, '')::uuid, $3, $4, $5, $6)
	`, strings.ToLower(strings.TrimSpace(email)), userID, clientIP(r), agent, success, reason)
}

func (h *Handler) ipBlocked(ctx context.Context, ip string) (bool, string) {
	if strings.TrimSpace(ip) == "" {
		return false, ""
	}
	var reason string
	err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT COALESCE(reason, '') FROM core.ip_blocks
		WHERE ip = $1 AND (expires_at IS NULL OR expires_at > now())
		LIMIT 1
	`, ip).Scan(&reason)
	if err != nil {
		return false, ""
	}
	return true, reason
}

func (h *Handler) autoBlockIfNeeded(ctx context.Context, ip string) {
	if strings.TrimSpace(ip) == "" {
		return
	}
	var fails int
	if h.dbOf(ctx).QueryRow(ctx, `
		SELECT COUNT(*) FROM core.login_attempts
		WHERE ip = $1 AND success = false AND created_at > now() - $2::interval
	`, ip, loginFailWindow.String()).Scan(&fails) != nil {
		return
	}
	if fails < loginFailThreshold {
		return
	}
	_, _ = h.dbOf(ctx).Exec(ctx, `
		INSERT INTO core.ip_blocks (ip, reason, auto, expires_at)
		VALUES ($1, $2, true, now() + $3::interval)
		ON CONFLICT (ip) DO UPDATE
		SET expires_at = EXCLUDED.expires_at, reason = EXCLUDED.reason
		WHERE core.ip_blocks.auto = true
	`, ip, strconv.Itoa(fails)+" неудачных входов подряд", loginAutoBlockFor.String())
}

func (h *Handler) staffRequires2FA(ctx context.Context) bool {
	return truthySetting(h.tenantSettingString(ctx, "security.staff_2fa_required"))
}

func (h *Handler) AdminLoginAttempts(w http.ResponseWriter, r *http.Request) {
	_, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	onlyFailed := r.URL.Query().Get("failed") == "1"
	search := strings.TrimSpace(r.URL.Query().Get("search"))

	rows, err := h.readerOf(r.Context()).Query(r.Context(), `
		SELECT email, ip, user_agent, success, COALESCE(reason, ''), created_at
		FROM core.login_attempts
		WHERE ($1 = false OR success = false)
		  AND ($2 = '' OR email ILIKE '%' || $2 || '%' OR ip ILIKE '%' || $2 || '%')
		ORDER BY created_at DESC
		LIMIT 200
	`, onlyFailed, search)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()

	list := []map[string]any{}
	for rows.Next() {
		var email, ip, agent, reason string
		var success bool
		var createdAt time.Time
		if rows.Scan(&email, &ip, &agent, &success, &reason, &createdAt) != nil {
			continue
		}
		list = append(list, map[string]any{
			"email": email, "ip": ip, "user_agent": agent,
			"success": success, "reason": reason,
			"created_at": createdAt.Format(time.RFC3339),
		})
	}

	summary := []map[string]any{}
	sumRows, err := h.readerOf(r.Context()).Query(r.Context(), `
		SELECT ip, COUNT(*) AS fails, MAX(created_at)
		FROM core.login_attempts
		WHERE success = false AND created_at > now() - interval '24 hours' AND ip <> ''
		GROUP BY ip
		HAVING COUNT(*) >= 3
		ORDER BY fails DESC
		LIMIT 20
	`)
	if err == nil {
		defer sumRows.Close()
		for sumRows.Next() {
			var ip string
			var fails int
			var last time.Time
			if sumRows.Scan(&ip, &fails, &last) == nil {
				summary = append(summary, map[string]any{
					"ip": ip, "fails": fails, "last_at": last.Format(time.RFC3339),
				})
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{"attempts": list, "suspicious": summary})
}

func (h *Handler) AdminIPBlocksList(w http.ResponseWriter, r *http.Request) {
	_, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	rows, err := h.readerOf(r.Context()).Query(r.Context(), `
		SELECT b.id::text, b.ip, COALESCE(b.reason, ''), b.auto, b.expires_at, b.created_at,
		       COALESCE(u.email, '')
		FROM core.ip_blocks b
		LEFT JOIN core.users u ON u.id = b.created_by
		ORDER BY b.created_at DESC
		LIMIT 200
	`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()

	list := []map[string]any{}
	for rows.Next() {
		var id, ip, reason, email string
		var auto bool
		var expires *time.Time
		var createdAt time.Time
		if rows.Scan(&id, &ip, &reason, &auto, &expires, &createdAt, &email) != nil {
			continue
		}
		list = append(list, map[string]any{
			"id": id, "ip": ip, "reason": reason, "auto": auto,
			"expires_at": timeOrNil(expires),
			"created_at": createdAt.Format(time.RFC3339),
			"created_by": email,
			"active":     expires == nil || expires.After(time.Now()),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"blocks": list})
}

type ipBlockBody struct {
	IP     string `json:"ip"`
	Reason string `json:"reason"`
	Hours  int    `json:"hours"`
}

func (h *Handler) AdminIPBlockCreate(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var body ipBlockBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	ip := strings.TrimSpace(body.IP)
	if ip == "" {
		writeError(w, http.StatusBadRequest, "укажите адрес")
		return
	}
	if ip == clientIP(r) {
		writeError(w, http.StatusConflict, "это ваш собственный адрес")
		return
	}

	var expires any
	if body.Hours > 0 {
		expires = time.Now().Add(time.Duration(body.Hours) * time.Hour)
	}
	var id string
	err := h.dbOf(r.Context()).QueryRow(r.Context(), `
		INSERT INTO core.ip_blocks (ip, reason, auto, expires_at, created_by)
		VALUES ($1, $2, false, $3, $4)
		ON CONFLICT (ip) DO UPDATE
		SET reason = EXCLUDED.reason, auto = false, expires_at = EXCLUDED.expires_at,
		    created_by = EXCLUDED.created_by
		RETURNING id::text
	`, ip, strings.TrimSpace(body.Reason), expires, nullableUUID(claims.UserID)).Scan(&id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	audit(r.Context(), h.dbOf(r.Context()), claims.UserID, "ip.block", "ip:"+ip,
		map[string]any{"reason": body.Reason, "hours": body.Hours})
	h.auditAlert(r.Context(), claims.UserID, claims.Email, "ip.block", ip)
	writeJSON(w, http.StatusCreated, map[string]any{"id": id, "ip": ip})
}

func (h *Handler) AdminIPBlockDelete(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := chi.URLParam(r, "id")
	tag, err := h.dbOf(r.Context()).Exec(r.Context(), `
		DELETE FROM core.ip_blocks WHERE id = $1
	`, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "блокировка не найдена")
		return
	}
	audit(r.Context(), h.dbOf(r.Context()), claims.UserID, "ip.unblock", "ip_block:"+id, nil)
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
