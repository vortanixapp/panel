package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func (h *Handler) AdminLogs(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok || !isStaffRole(claims.Role) {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	if r.URL.Query().Get("stream") == "1" {
		h.adminLogsStream(w, r)
		return
	}
	limit := 100
	if v := strings.TrimSpace(r.URL.Query().Get("limit")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			if n > 1000 {
				n = 1000
			}
			limit = n
		}
	}
	search := strings.TrimSpace(r.URL.Query().Get("search"))
	action := strings.TrimSpace(r.URL.Query().Get("action"))
	q := `
		SELECT id::text, action, resource, created_at::text
		FROM core.audit_logs
		WHERE true
	`
	args := []any{}
	if action != "" {
		q += ` AND action = $1`
		args = append(args, action)
	}
	if search != "" {
		idx := len(args) + 1
		q += ` AND (resource ILIKE '%' || $` + strconv.Itoa(idx) + ` || '%' OR action ILIKE '%' || $` + strconv.Itoa(idx) + ` || '%')`
		args = append(args, search)
	}
	q += ` ORDER BY created_at DESC LIMIT ` + strconv.Itoa(limit)
	rows, _ := h.dbOf(r.Context()).Query(r.Context(), q, args...)
	defer func() {
		if rows != nil {
			rows.Close()
		}
	}()
	list := []map[string]any{}
	if rows != nil {
		for rows.Next() {
			var id, action, res, created string
			if rows.Scan(&id, &action, &res, &created) == nil {
				list = append(list, map[string]any{"id": id, "action": action, "resource": res, "created_at": created})
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"logs":   list,
		"limit":  limit,
		"search": search,
		"action": action,
	})
}

func (h *Handler) adminLogsStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	if rc := http.NewResponseController(w); rc != nil {
		_ = rc.SetWriteDeadline(time.Time{})
	}

	ctx := r.Context()
	since := strings.TrimSpace(r.URL.Query().Get("since"))

	send := func() bool {
		q := `SELECT id::text, action, resource, created_at::text
		      FROM core.audit_logs WHERE true`
		args := []any{}
		if since != "" {
			q += ` AND created_at > $1::timestamptz`
			args = append(args, since)
		}
		q += ` ORDER BY created_at ASC LIMIT 200`
		rows, err := h.dbOf(ctx).Query(ctx, q, args...)
		if err != nil {
			return false
		}
		defer rows.Close()
		for rows.Next() {
			var id, action, res, created string
			if rows.Scan(&id, &action, &res, &created) != nil {
				continue
			}
			b, _ := json.Marshal(map[string]string{
				"id": id, "action": action, "resource": res, "created_at": created,
			})
			if _, err := w.Write([]byte("data: " + string(b) + "\n\n")); err != nil {
				return false
			}
			since = created
		}
		flusher.Flush()
		return rows.Err() == nil
	}

	if !send() {
		return
	}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	keepalive := time.NewTicker(15 * time.Second)
	defer keepalive.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-keepalive.C:
			if _, err := w.Write([]byte(": keepalive\n\n")); err != nil {
				return
			}
			flusher.Flush()
		case <-ticker.C:
			if !send() {
				return
			}
		}
	}
}
