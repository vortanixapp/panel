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
		q += ` AND action = $2`
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
			q += ` AND created_at > $2::timestamptz`
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

const bugReportCategory = "bug"

var bugSeverityPriority = map[string]string{
	"low":      "low",
	"medium":   "normal",
	"high":     "high",
	"critical": "urgent",
}

const (
	bugFieldMaxLen  = 200
	bugEnvMaxFields = 20
)

func clampBugField(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > bugFieldMaxLen {
		return value[:bugFieldMaxLen]
	}
	return value
}

func (h *Handler) AdminBugReportCreate(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok || !isStaffRole(claims.Role) {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	var body struct {
		Title       string            `json:"title"`
		Description string            `json:"description"`
		Severity    string            `json:"severity"`
		Component   string            `json:"component"`
		Node        string            `json:"node"`
		Environment map[string]string `json:"environment"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	title := strings.TrimSpace(body.Title)
	if title == "" {
		writeError(w, http.StatusBadRequest, "title required")
		return
	}
	severity := strings.ToLower(strings.TrimSpace(body.Severity))
	priority, known := bugSeverityPriority[severity]
	if !known {
		severity, priority = "medium", "normal"
	}
	component := clampBugField(body.Component)
	if component == "" {
		component = "panel-ui"
	}
	node := clampBugField(body.Node)
	subject := "[BUG][" + strings.ToUpper(severity) + "][" + component + "] " + title
	desc := strings.TrimSpace(body.Description)
	if desc == "" {
		desc = "Описание не указано"
	}

	meta := map[string]any{
		"kind":      bugReportCategory,
		"severity":  severity,
		"component": component,
	}
	if node != "" {
		meta["node"] = node
	}
	if len(body.Environment) > 0 {
		env := make(map[string]string, len(body.Environment))
		for k, v := range body.Environment {
			if len(env) >= bugEnvMaxFields {
				break
			}
			env[clampBugField(k)] = clampBugField(v)
		}
		meta["environment"] = env
	}
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed encoding report meta")
		return
	}

	ctx := r.Context()
	tx, err := h.dbOf(ctx).Begin(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer tx.Rollback(ctx)

	var ticketID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO core.support_tickets ( user_id, subject, category, priority, meta)
		VALUES ( $1, $2, $3, $4, $5::jsonb)
		RETURNING id::text
	`, claims.UserID, subject, bugReportCategory, priority, metaJSON).Scan(&ticketID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed creating ticket")
		return
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO core.support_messages ( ticket_id, user_id, message, body, is_staff)
		VALUES ( $1::uuid, $2, $3, $3, true)
	`, ticketID, claims.UserID, desc); err != nil {
		writeError(w, http.StatusInternalServerError, "failed saving report body")
		return
	}
	if err := tx.Commit(ctx); err != nil {
		writeError(w, http.StatusInternalServerError, "failed creating ticket")
		return
	}

	audit(ctx, h.dbOf(ctx), claims.UserID, "admin.bug_report.create", ticketID, map[string]any{
		"severity":  severity,
		"component": component,
		"node":      node,
		"priority":  priority,
	})
	writeJSON(w, http.StatusCreated, map[string]any{
		"ok":        true,
		"ticket_id": ticketID,
		"priority":  priority,
		"status":    "created",
	})
}

func bugReportTitle(subject string) string {
	rest := subject
	for i := 0; i < 3; i++ {
		if !strings.HasPrefix(rest, "[") {
			return strings.TrimSpace(rest)
		}
		end := strings.Index(rest, "]")
		if end < 0 {
			return strings.TrimSpace(rest)
		}
		rest = rest[end+1:]
	}
	return strings.TrimSpace(rest)
}

func (h *Handler) AdminBugReportList(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok || !isStaffRole(claims.Role) {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	ctx := r.Context()
	rows, err := h.dbOf(ctx).Query(ctx, `
		SELECT t.id::text, t.subject, t.status, t.priority, t.created_at::text,
		       COALESCE(t.meta->>'severity', ''), COALESCE(t.meta->>'component', ''),
		       t.user_id::text = $1
		FROM core.support_tickets t
		WHERE (t.category = $2 OR t.meta->>'kind' = $2 OR t.subject LIKE '[BUG]%')
		ORDER BY t.created_at DESC
		LIMIT 50
	`, claims.UserID, bugReportCategory)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed loading reports")
		return
	}
	defer rows.Close()

	reports := []map[string]any{}
	mine := 0
	for rows.Next() {
		var id, subject, status, priority, created, severity, component string
		var own bool
		if err := rows.Scan(&id, &subject, &status, &priority, &created, &severity, &component, &own); err != nil {
			writeError(w, http.StatusInternalServerError, "failed loading reports")
			return
		}
		if own {
			mine++
		}
		reports = append(reports, map[string]any{
			"id": id, "title": bugReportTitle(subject), "status": status,
			"priority": priority, "created_at": created,
			"severity": severity, "component": component, "mine": own,
		})
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed loading reports")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"reports": reports, "mine": mine})
}
