package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/vortanixapp/panel/internal/api/jobwake"
)

type nodeBulkBody struct {
	Action      string `json:"action"`
	Message     string `json:"message"`
	Days        int    `json:"days"`
	OnlyRunning bool   `json:"only_running"`
}

var nodeBulkActions = map[string]string{
	"start":   "Запуск серверов",
	"stop":    "Остановка серверов",
	"restart": "Перезапуск серверов",
	"notify":  "Уведомление владельцам",
	"extend":  "Продление аренды",
}

func (h *Handler) AdminNodeBulkAction(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	nodeID := chi.URLParam(r, "id")
	var body nodeBulkBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	action := strings.TrimSpace(strings.ToLower(body.Action))
	if _, known := nodeBulkActions[action]; !known {
		writeError(w, http.StatusBadRequest, "unknown action")
		return
	}
	if action == "notify" && strings.TrimSpace(body.Message) == "" {
		writeError(w, http.StatusBadRequest, "message required")
		return
	}
	if action == "extend" && (body.Days <= 0 || body.Days > 365) {
		writeError(w, http.StatusBadRequest, "days must be between 1 and 365")
		return
	}
	ctx := r.Context()

	var nodeName string
	if h.dbOf(ctx).QueryRow(ctx, `SELECT name FROM core.nodes WHERE id = $1`,
		nodeID).Scan(&nodeName) != nil {
		writeError(w, http.StatusNotFound, "location not found")
		return
	}

	var affected int
	countQuery := `SELECT COUNT(*)::int FROM core.servers WHERE node_id = $1`
	if body.OnlyRunning {
		countQuery += ` AND status = 'running'`
	}
	_ = h.dbOf(ctx).QueryRow(ctx, countQuery, nodeID).Scan(&affected)
	if affected == 0 {
		writeError(w, http.StatusConflict, "на локации нет подходящих серверов")
		return
	}

	payload, _ := json.Marshal(map[string]any{
		"node_id": nodeID, "action": action,
		"message": strings.TrimSpace(body.Message), "days": body.Days,
		"only_running": body.OnlyRunning,
		"node_name":    nodeName,
	})
	if _, err := h.dbOf(ctx).Exec(ctx, `
		INSERT INTO core.jobs ( type, status, payload)
		VALUES ( 'node_bulk', 'pending', $1::jsonb)
	`, payload); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	jobwake.Notify("node_bulk")
	audit(ctx, h.dbOf(ctx), claims.UserID, "node.bulk."+action, "node:"+nodeID,
		map[string]any{"servers": affected, "days": body.Days})
	h.auditAlert(ctx, claims.UserID, claims.Email, "node.bulk."+action,
		nodeName+", серверов: "+strconv.Itoa(affected))

	writeJSON(w, http.StatusAccepted, map[string]any{
		"status":  "queued",
		"action":  action,
		"servers": affected,
	})
}
