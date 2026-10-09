package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/vortanixapp/panel/pkg/protocol"
)

const (
	taskSourceJob  = "job"
	taskSourceNode = "node_task"
)

var nodeTaskLabels = map[string]string{
	protocol.ActionNodeInfo:       "Сбор сведений о ноде",
	protocol.ActionAgentLogs:      "Чтение логов агента",
	protocol.ActionAgentRestart:   "Перезапуск агента",
	protocol.ActionContainersList: "Список контейнеров",
	protocol.ActionDiskUsage:      "Анализ диска",
	protocol.ActionCleanupPreview: "Предпросмотр очистки",
	protocol.ActionCleanupApply:   "Очистка ноды",
	protocol.ActionDiagnostics:    "Диагностика ноды",
	protocol.ActionSandboxInstall: "Установка gVisor",
	protocol.ActionAgentUpdate:    "Обновление агента",
}

func nodeTaskLabel(action string) string {
	if label, ok := nodeTaskLabels[action]; ok {
		return label
	}
	return action
}

func normalizeJobStatus(status string) string {
	switch status {
	case "pending":
		return "queued"
	case "completed":
		return "done"
	default:
		return status
	}
}

func normalizeNodeTaskStatus(status string) string {
	switch status {
	case "sent":
		return "running"
	case "expired":
		return "failed"
	default:
		return status
	}
}

func isActiveTaskStatus(status string) bool {
	return status == "queued" || status == "running"
}

type taskFeedItem struct {
	ID          string          `json:"id"`
	Source      string          `json:"source"`
	Kind        string          `json:"kind"`
	Label       string          `json:"label"`
	Status      string          `json:"status"`
	Target      string          `json:"target"`
	Error       string          `json:"error,omitempty"`
	Percent     *int            `json:"percent,omitempty"`
	Message     string          `json:"message,omitempty"`
	Stuck       bool            `json:"stuck,omitempty"`
	Attempts    int             `json:"attempts,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	FinishedAt  *time.Time      `json:"finished_at,omitempty"`
	DurationSec int64           `json:"duration_sec"`
	Input       json.RawMessage `json:"input,omitempty"`
	Output      json.RawMessage `json:"output,omitempty"`
}

func taskDuration(created time.Time, finished *time.Time, status string, updated time.Time) int64 {
	end := time.Now()
	if finished != nil {
		end = *finished
	} else if !isActiveTaskStatus(status) {
		end = updated
	}
	if d := int64(end.Sub(created).Seconds()); d > 0 {
		return d
	}
	return 0
}

func progressFields(raw []byte) (*int, string) {
	if len(raw) == 0 {
		return nil, ""
	}
	var p map[string]any
	if json.Unmarshal(raw, &p) != nil {
		return nil, ""
	}
	var percent *int
	switch v := p["percent"].(type) {
	case float64:
		n := int(v)
		percent = &n
	case string:
		if n := jobQueryInt(v, -1, -1, 100); n >= 0 {
			percent = &n
		}
	}
	msg, _ := p["message"].(string)
	if msg == "" {
		msg, _ = p["stage"].(string)
	}
	return percent, msg
}

func joinTarget(parts ...string) string {
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, " · ")
}

func (h *Handler) feedJobs(ctx context.Context, scope string, limit int, id string) []taskFeedItem {
	where := " WHERE true"
	args := []any{}
	switch {
	case id != "":
		where += " AND j.id::text = $1"
		args = append(args, id)
	case scope == "active":
		where += " AND j.status IN ('pending', 'running')"
	default:
		where += " AND j.status IN ('completed', 'failed', 'cancelled')"
	}
	args = append(args, limit)
	rows, err := h.readerOf(ctx).Query(ctx, `
		SELECT j.id::text, j.type, j.status, j.attempts, j.created_at, j.updated_at,
		       j.payload, COALESCE(j.result, '{}'::jsonb),
		       COALESCE(s.name, ''), COALESCE(n.name, ''),
		       COALESCE(j.payload->>'server_id', '')
		FROM core.jobs j
		LEFT JOIN core.servers s ON s.id = core.try_uuid(j.payload->>'server_id')
		LEFT JOIN core.nodes n ON n.id = COALESCE(core.try_uuid(j.payload->>'node_id'), s.node_id)
	`+where+`
		ORDER BY j.created_at DESC
		LIMIT $`+strconv.Itoa(len(args)), args...)
	if err != nil {
		return nil
	}
	defer rows.Close()

	stuckAfter := time.Duration(defaultJobStuckMinutes) * time.Minute
	items := []taskFeedItem{}
	for rows.Next() {
		var it taskFeedItem
		var status, serverName, nodeName, serverID string
		var payload, result []byte
		if rows.Scan(&it.ID, &it.Kind, &status, &it.Attempts, &it.CreatedAt, &it.UpdatedAt,
			&payload, &result, &serverName, &nodeName, &serverID) != nil {
			continue
		}
		it.Source = taskSourceJob
		it.Label = jobTypeLabel(it.Kind)
		it.Status = normalizeJobStatus(status)
		it.Target = joinTarget(serverName, nodeName)
		it.Error = jobErrorText(result)
		it.Input = payload
		it.Output = result
		if status != "pending" && status != "running" {
			finished := it.UpdatedAt
			it.FinishedAt = &finished
		}
		it.Stuck = status == "running" && time.Since(it.UpdatedAt) > stuckAfter
		it.DurationSec = taskDuration(it.CreatedAt, it.FinishedAt, it.Status, it.UpdatedAt)
		if status == "running" && serverID != "" && it.Kind == "provision_server" {
			if p := h.deriveProvisioningProgress(ctx, serverID, "provisioning", ""); p != nil {
				raw, _ := json.Marshal(p)
				it.Percent, it.Message = progressFields(raw)
				if derived, _ := p["derived"].(bool); derived {
					it.Percent = nil
				}
			}
		}
		items = append(items, it)
	}
	return items
}

func (h *Handler) feedNodeTasks(ctx context.Context, scope string, limit int, id string) []taskFeedItem {
	where := " WHERE true"
	args := []any{}
	switch {
	case id != "":
		where += " AND t.id::text = $1"
		args = append(args, id)
	case scope == "active":
		where += " AND t.status IN ('queued', 'sent', 'running')"
	default:
		where += " AND t.status IN ('done', 'failed', 'expired')"
	}
	args = append(args, limit)
	rows, err := h.readerOf(ctx).Query(ctx, `
		SELECT t.id::text, t.action, t.status, t.params, t.progress, t.result,
		       COALESCE(t.error, ''), t.created_at, t.updated_at, t.finished_at,
		       COALESCE(n.name, ''), COALESCE(t.deadline_at < now(), false)
		FROM core.node_tasks t
		LEFT JOIN core.nodes n ON n.id = t.node_id
	`+where+`
		ORDER BY t.created_at DESC
		LIMIT $`+strconv.Itoa(len(args)), args...)
	if err != nil {
		return nil
	}
	defer rows.Close()

	items := []taskFeedItem{}
	for rows.Next() {
		var it taskFeedItem
		var status, nodeName string
		var params, progress, result []byte
		var overdue bool
		if rows.Scan(&it.ID, &it.Kind, &status, &params, &progress, &result,
			&it.Error, &it.CreatedAt, &it.UpdatedAt, &it.FinishedAt, &nodeName, &overdue) != nil {
			continue
		}
		it.Source = taskSourceNode
		it.Label = nodeTaskLabel(it.Kind)
		it.Status = normalizeNodeTaskStatus(status)
		it.Target = nodeName
		it.Input = params
		it.Output = result
		it.Percent, it.Message = progressFields(progress)
		it.Stuck = isActiveTaskStatus(it.Status) && overdue
		if status == "expired" && it.Error == "" {
			it.Error = "Время ожидания истекло"
		}
		it.DurationSec = taskDuration(it.CreatedAt, it.FinishedAt, it.Status, it.UpdatedAt)
		items = append(items, it)
	}
	return items
}

func (h *Handler) taskFeedCounts(ctx context.Context) map[string]int {
	counts := map[string]int{"active": 0, "queued": 0, "running": 0, "failed_24h": 0}
	var jobsQueued, jobsRunning, jobsFailed int
	_ = h.readerOf(ctx).QueryRow(ctx, `
		SELECT COUNT(*) FILTER (WHERE status = 'pending'),
		       COUNT(*) FILTER (WHERE status = 'running'),
		       COUNT(*) FILTER (WHERE status = 'failed' AND updated_at > now() - interval '24 hours')
		FROM core.jobs
	`).Scan(&jobsQueued, &jobsRunning, &jobsFailed)
	var taskQueued, taskRunning, taskFailed int
	_ = h.readerOf(ctx).QueryRow(ctx, `
		SELECT COUNT(*) FILTER (WHERE status = 'queued'),
		       COUNT(*) FILTER (WHERE status IN ('sent', 'running')),
		       COUNT(*) FILTER (WHERE status IN ('failed', 'expired') AND updated_at > now() - interval '24 hours')
		FROM core.node_tasks
	`).Scan(&taskQueued, &taskRunning, &taskFailed)
	counts["queued"] = jobsQueued + taskQueued
	counts["running"] = jobsRunning + taskRunning
	counts["active"] = counts["queued"] + counts["running"]
	counts["failed_24h"] = jobsFailed + taskFailed
	return counts
}

func (h *Handler) AdminTaskFeed(w http.ResponseWriter, r *http.Request) {
	if _, ok := tenantClaims(r.Context()); !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ctx := r.Context()
	query := r.URL.Query()
	scope := "history"
	if strings.EqualFold(strings.TrimSpace(query.Get("scope")), "active") {
		scope = "active"
	}
	limit := jobQueryInt(query.Get("limit"), 50, 1, 200)

	items := append(h.feedJobs(ctx, scope, limit, ""), h.feedNodeTasks(ctx, scope, limit, "")...)
	sort.SliceStable(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	if len(items) > limit {
		items = items[:limit]
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"tasks":  items,
		"counts": h.taskFeedCounts(ctx),
	})
}

func (h *Handler) taskLogLines(r *http.Request, it taskFeedItem) []string {
	ctx := r.Context()
	if it.Source == taskSourceJob {
		var payload map[string]any
		_ = json.Unmarshal(it.Input, &payload)
		serverID, _ := payload["server_id"].(string)
		nodeID, _ := payload["node_id"].(string)
		switch it.Kind {
		case "provision_server":
			if serverID != "" {
				if lines, err := h.cache.InstallLog(ctx, serverID, 400); err == nil {
					return lines
				}
			}
		case "node_setup":
			if status, ok := h.locationSetupStatus(r, nodeID); ok {
				if text, _ := status["log"].(string); strings.TrimSpace(text) != "" {
					return strings.Split(strings.TrimRight(text, "\n"), "\n")
				}
			}
		}
	}
	return nil
}

func (h *Handler) AdminTaskFeedDetail(w http.ResponseWriter, r *http.Request) {
	if _, ok := tenantClaims(r.Context()); !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	source := chi.URLParam(r, "source")
	id := chi.URLParam(r, "id")

	var found []taskFeedItem
	switch source {
	case taskSourceJob:
		found = h.feedJobs(r.Context(), "", 1, id)
	case taskSourceNode:
		found = h.feedNodeTasks(r.Context(), "", 1, id)
	default:
		writeError(w, http.StatusBadRequest, "unknown task source")
		return
	}
	if len(found) == 0 {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	lines := h.taskLogLines(r, found[0])
	if lines == nil {
		lines = []string{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"task": found[0], "log": lines})
}

func (h *Handler) AdminTaskEvents(w http.ResponseWriter, r *http.Request) {
	if _, ok := tenantClaims(r.Context()); !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ctx := r.Context()
	limit := jobQueryInt(r.URL.Query().Get("limit"), 50, 1, 200)
	rows, err := h.readerOf(ctx).Query(ctx, `
		SELECT e.id, e.kind, e.level, e.data, e.created_at, COALESCE(n.name, '')
		FROM core.node_events e
		LEFT JOIN core.nodes n ON n.id = e.node_id
		ORDER BY e.id DESC
		LIMIT $1
	`, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()

	events := []map[string]any{}
	for rows.Next() {
		var id int64
		var kind, level, node string
		var data []byte
		var created time.Time
		if rows.Scan(&id, &kind, &level, &data, &created, &node) != nil {
			continue
		}
		events = append(events, map[string]any{
			"id": id, "kind": kind, "level": level, "node": node,
			"data": json.RawMessage(data), "created_at": created.UTC().Format(time.RFC3339),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"events": events})
}
