package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/vortanixapp/panel/internal/api/jobwake"
)

const defaultJobStuckMinutes = 30

var jobTypeLabels = map[string]string{
	"provision_server": "Создание сервера",
	"backup_server":    "Резервная копия",
	"node_setup":       "Настройка ноды",
	"daemon_action":    "Действие демона",
	"daemon_pull":      "Обновление образов",
	"send_mailing":     "Рассылка",
	"ftp_account":      "FTP-аккаунт",
}

func jobTypeLabel(t string) string {
	if label, ok := jobTypeLabels[t]; ok {
		return label
	}
	return t
}

func jobQueryInt(raw string, def, min, max int) int {
	v := strings.TrimSpace(raw)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	if n < min {
		return min
	}
	if n > max {
		return max
	}
	return n
}

func jobErrorText(result []byte) string {
	if len(result) == 0 {
		return ""
	}
	var parsed map[string]any
	if json.Unmarshal(result, &parsed) != nil {
		return ""
	}
	if msg, ok := parsed["error"].(string); ok {
		return msg
	}
	return ""
}

func (h *Handler) AdminJobsList(w http.ResponseWriter, r *http.Request) {
	_, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ctx := r.Context()
	query := r.URL.Query()

	status := strings.TrimSpace(strings.ToLower(query.Get("status")))
	jobType := strings.TrimSpace(strings.ToLower(query.Get("type")))
	search := strings.TrimSpace(query.Get("search"))
	limit := jobQueryInt(query.Get("limit"), 50, 1, 500)
	offset := jobQueryInt(query.Get("offset"), 0, 0, 1_000_000)
	stuckMinutes := jobQueryInt(query.Get("stuck_minutes"), defaultJobStuckMinutes, 1, 1440)

	where := " WHERE true"
	args := []any{}
	arg := func(v any) string {
		args = append(args, v)
		return "$" + strconv.Itoa(len(args))
	}

	switch status {
	case "", "all":
	case "stuck":
		where += " AND j.status = 'running' AND j.updated_at < now() - make_interval(mins => " + arg(stuckMinutes) + "::int)"
	case "active":
		where += " AND j.status IN ('pending', 'running')"
	case "pending", "running", "completed", "failed", "cancelled":
		where += " AND j.status = " + arg(status)
	default:
		writeError(w, http.StatusBadRequest, "invalid status")
		return
	}
	if jobType != "" && jobType != "all" {
		where += " AND j.type = " + arg(jobType)
	}
	if search != "" {
		p := arg(search)
		where += " AND (j.id::text ILIKE '%' || " + p + " || '%' OR j.payload::text ILIKE '%' || " + p + " || '%')"
	}

	var total int
	_ = h.readerOf(ctx).QueryRow(ctx, "SELECT COUNT(*) FROM core.jobs j"+where, args...).Scan(&total)

	listArgs := append(append([]any{}, args...), limit, offset)
	rows, err := h.readerOf(ctx).Query(ctx, `
		SELECT j.id::text, j.type, j.status, j.attempts,
		       j.created_at, j.updated_at,
		       j.payload, COALESCE(j.result, '{}'::jsonb),
		       EXTRACT(EPOCH FROM (now() - j.created_at))::bigint,
		       EXTRACT(EPOCH FROM (now() - j.updated_at))::bigint,
		       COALESCE(s.name, ''), COALESCE(n.name, '')
		FROM core.jobs j
		LEFT JOIN core.servers s
		       ON s.id = core.try_uuid(j.payload->>'server_id')
		LEFT JOIN core.nodes n
		       ON n.id = COALESCE(core.try_uuid(j.payload->>'node_id'), s.node_id)
	`+where+`
		ORDER BY j.created_at DESC
		LIMIT $`+strconv.Itoa(len(args)+1)+" OFFSET $"+strconv.Itoa(len(args)+2),
		listArgs...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()

	jobs := []map[string]any{}
	for rows.Next() {
		var id, jType, jStatus, serverName, nodeName string
		var attempts int
		var createdAt, updatedAt time.Time
		var payload, result []byte
		var ageSec, sinceUpdateSec int64
		if rows.Scan(&id, &jType, &jStatus, &attempts, &createdAt, &updatedAt,
			&payload, &result, &ageSec, &sinceUpdateSec, &serverName, &nodeName) != nil {
			continue
		}
		jobs = append(jobs, map[string]any{
			"id":           id,
			"type":         jType,
			"type_label":   jobTypeLabel(jType),
			"status":       jStatus,
			"attempts":     attempts,
			"created_at":   createdAt.Format(time.RFC3339),
			"updated_at":   updatedAt.Format(time.RFC3339),
			"age_seconds":  ageSec,
			"idle_seconds": sinceUpdateSec,
			"stuck":        jStatus == "running" && sinceUpdateSec > int64(stuckMinutes)*60,
			"error":        jobErrorText(result),
			"payload":      json.RawMessage(payload),
			"result":       json.RawMessage(result),
			"server_name":  serverName,
			"node_name":    nodeName,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"jobs":          jobs,
		"total":         total,
		"limit":         limit,
		"offset":        offset,
		"stuck_minutes": stuckMinutes,
		"summary":       h.jobsSummary(ctx, stuckMinutes),
		"types":         h.jobsTypes(ctx),
	})
}

func (h *Handler) jobsSummary(ctx context.Context, stuckMinutes int) map[string]any {
	summary := map[string]any{
		"pending": 0, "running": 0, "completed": 0, "failed": 0, "cancelled": 0,
		"stuck": 0, "oldest_pending_seconds": 0,
	}
	rows, err := h.readerOf(ctx).Query(ctx, `
		SELECT status, COUNT(*) FROM core.jobs GROUP BY status
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var st string
			var count int
			if rows.Scan(&st, &count) == nil {
				summary[st] = count
			}
		}
	}
	var stuck int
	_ = h.readerOf(ctx).QueryRow(ctx, `
		SELECT COUNT(*) FROM core.jobs
		WHERE status = 'running'
		  AND updated_at < now() - make_interval(mins => $1::int)
	`, stuckMinutes).Scan(&stuck)
	summary["stuck"] = stuck

	var oldest int64
	_ = h.readerOf(ctx).QueryRow(ctx, `
		SELECT COALESCE(EXTRACT(EPOCH FROM (now() - MIN(created_at)))::bigint, 0)
		FROM core.jobs WHERE status = 'pending'
	`).Scan(&oldest)
	summary["oldest_pending_seconds"] = oldest
	return summary
}

func (h *Handler) jobsTypes(ctx context.Context) []map[string]any {
	out := []map[string]any{}
	rows, err := h.readerOf(ctx).Query(ctx, `
		SELECT type,
		       COUNT(*),
		       COUNT(*) FILTER (WHERE status = 'pending'),
		       COUNT(*) FILTER (WHERE status = 'failed')
		FROM core.jobs
		GROUP BY type ORDER BY type
	`)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var t string
		var total, pending, failed int
		if rows.Scan(&t, &total, &pending, &failed) == nil {
			out = append(out, map[string]any{
				"type": t, "label": jobTypeLabel(t),
				"total": total, "pending": pending, "failed": failed,
			})
		}
	}
	return out
}

func (h *Handler) AdminJobRetry(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := chi.URLParam(r, "id")
	stuckMinutes := jobQueryInt(r.URL.Query().Get("stuck_minutes"), defaultJobStuckMinutes, 1, 1440)

	var jobType string
	err := h.dbOf(r.Context()).QueryRow(r.Context(), `
		UPDATE core.jobs SET
			status        = 'pending',
			result        = NULL,
			attempts      = attempts + 1,
			last_actor_id = $2
		WHERE id = $1
		  AND (status IN ('failed', 'cancelled')
		       OR (status = 'running' AND updated_at < now() - make_interval(mins => $3::int)))
		RETURNING type
	`, id, nullableUUID(claims.UserID), stuckMinutes).Scan(&jobType)
	if err != nil {
		var status string
		if h.dbOf(r.Context()).QueryRow(r.Context(), `SELECT status FROM core.jobs WHERE id = $1`,
			id).Scan(&status) == nil {
			writeError(w, http.StatusConflict, "job in status "+status+" cannot be retried")
			return
		}
		writeError(w, http.StatusNotFound, "job not found")
		return
	}

	jobwake.Notify(jobType)
	audit(r.Context(), h.dbOf(r.Context()), claims.UserID, "job.retry", "job:"+id,
		map[string]any{"type": jobType})
	writeJSON(w, http.StatusOK, map[string]string{"status": "pending", "type": jobType})
}

func (h *Handler) AdminJobCancel(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := chi.URLParam(r, "id")
	stuckMinutes := jobQueryInt(r.URL.Query().Get("stuck_minutes"), defaultJobStuckMinutes, 1, 1440)
	result, _ := json.Marshal(map[string]string{"error": "отменено администратором"})

	var jobType string
	err := h.dbOf(r.Context()).QueryRow(r.Context(), `
		UPDATE core.jobs SET
			status        = 'cancelled',
			result        = $2::jsonb,
			last_actor_id = $3
		WHERE id = $1
		  AND (status = 'pending'
		       OR (status = 'running' AND updated_at < now() - make_interval(mins => $4::int)))
		RETURNING type
	`, id, result, nullableUUID(claims.UserID), stuckMinutes).Scan(&jobType)
	if err != nil {
		var status string
		if h.dbOf(r.Context()).QueryRow(r.Context(), `SELECT status FROM core.jobs WHERE id = $1`,
			id).Scan(&status) == nil {
			writeError(w, http.StatusConflict, "job in status "+status+" cannot be cancelled")
			return
		}
		writeError(w, http.StatusNotFound, "job not found")
		return
	}
	audit(r.Context(), h.dbOf(r.Context()), claims.UserID, "job.cancel", "job:"+id,
		map[string]any{"type": jobType})
	writeJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
}

func (h *Handler) AdminJobDelete(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := chi.URLParam(r, "id")
	tag, err := h.dbOf(r.Context()).Exec(r.Context(), `
		DELETE FROM core.jobs
		WHERE id = $1 AND status IN ('completed', 'failed', 'cancelled')
	`, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusConflict, "only finished jobs can be deleted")
		return
	}
	audit(r.Context(), h.dbOf(r.Context()), claims.UserID, "job.delete", "job:"+id, nil)
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

type jobsBulkBody struct {
	Type string `json:"type"`
	Days int    `json:"days"`
}

func (h *Handler) AdminJobsRetryFailed(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var body jobsBulkBody
	_ = json.NewDecoder(r.Body).Decode(&body)
	jobType := strings.TrimSpace(strings.ToLower(body.Type))

	rows, err := h.dbOf(r.Context()).Query(r.Context(), `
		UPDATE core.jobs SET
			status        = 'pending',
			result        = NULL,
			attempts      = attempts + 1,
			last_actor_id = $1
		WHERE status = 'failed'
		  AND ($2 = '' OR type = $2)
		RETURNING type
	`, nullableUUID(claims.UserID), jobType)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()

	woken := map[string]bool{}
	count := 0
	for rows.Next() {
		var t string
		if rows.Scan(&t) == nil {
			count++
			woken[t] = true
		}
	}
	for t := range woken {
		jobwake.Notify(t)
	}
	audit(r.Context(), h.dbOf(r.Context()), claims.UserID, "job.retry_failed", "jobs",
		map[string]any{"count": count, "type": jobType})
	writeJSON(w, http.StatusOK, map[string]any{"status": "queued", "count": count})
}

func (h *Handler) AdminJobsCleanup(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var body jobsBulkBody
	_ = json.NewDecoder(r.Body).Decode(&body)
	days := body.Days
	if days <= 0 {
		days = 30
	}
	if days > 3650 {
		days = 3650
	}
	tag, err := h.dbOf(r.Context()).Exec(r.Context(), `
		DELETE FROM core.jobs
		WHERE status IN ('completed', 'cancelled')
		  AND created_at < now() - make_interval(days => $1::int)
	`, days)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	audit(r.Context(), h.dbOf(r.Context()), claims.UserID, "job.cleanup", "jobs",
		map[string]any{"days": days, "count": tag.RowsAffected()})
	writeJSON(w, http.StatusOK, map[string]any{"status": "deleted", "count": tag.RowsAffected()})
}
