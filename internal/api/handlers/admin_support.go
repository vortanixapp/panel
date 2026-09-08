package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

var supportStatuses = map[string]bool{
	"open": true, "pending": true, "answered": true, "closed": true,
}

var supportPriorities = map[string]bool{
	"low": true, "normal": true, "high": true, "urgent": true,
}

func (h *Handler) AdminSupportQueue(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	if status != "" && !supportStatuses[status] {
		writeError(w, http.StatusBadRequest, "unknown status")
		return
	}
	// Отдел не сверяем со списком: набор настраивается для каждой панели
	// отдельно, и жёсткая проверка запрещала бы фильтр по своему отделу.
	category := strings.TrimSpace(r.URL.Query().Get("category"))

	rows, err := h.readerOf(r.Context()).Query(r.Context(), `
		SELECT t.id::text, t.subject, t.status, t.priority, t.category,
		       t.user_id::text, COALESCE(u.email, ''),
		       COALESCE(t.assigned_admin_id::text, ''), COALESCE(a.email, ''),
		       t.created_at::text, COALESCE(t.last_message_at, t.created_at)::text,
		       (SELECT COUNT(*) FROM core.support_messages m WHERE m.ticket_id = t.id),
		       COALESCE((
		           SELECT NOT m.is_staff FROM core.support_messages m
		           WHERE m.ticket_id = t.id ORDER BY m.created_at DESC LIMIT 1
		       ), true),
		       -- Сколько обращений у этого клиента всего: оператору важно
		       -- отличать первое обращение от двенадцатого, ответ на них
		       -- разный.
		       (SELECT COUNT(*) FROM core.support_tickets ut
		         WHERE ut.tenant_id = t.tenant_id AND ut.user_id = t.user_id),
		       t.service_kind, COALESCE(t.service_id::text, '')
		FROM core.support_tickets t
		LEFT JOIN core.users u ON u.id = t.user_id
		LEFT JOIN core.users a ON a.id = t.assigned_admin_id
		WHERE t.tenant_id = $1 AND ($2 = '' OR t.status = $2)
		  AND ($3 = '' OR t.category = $3)
		ORDER BY
			CASE t.priority WHEN 'urgent' THEN 0 WHEN 'high' THEN 1 WHEN 'normal' THEN 2 ELSE 3 END,
			COALESCE(t.last_message_at, t.created_at) DESC
		LIMIT 200
	`, claims.TenantID, status, category)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()

	list := []map[string]any{}
	for rows.Next() {
		var id, subject, st, priority, category, userID, email, assignedID, assignedEmail string
		var created, lastMessage string
		var messages, userTickets int
		var awaitingStaff bool
		var serviceKind, serviceID string
		if rows.Scan(&id, &subject, &st, &priority, &category, &userID, &email, &assignedID,
			&assignedEmail, &created, &lastMessage, &messages, &awaitingStaff, &userTickets,
			&serviceKind, &serviceID) != nil {
			continue
		}
		list = append(list, map[string]any{
			"id": id, "subject": subject, "status": st, "priority": priority,
			"category": category,
			"user_id":  userID, "user_email": email,
			"assigned_admin_id": assignedID, "assigned_admin_email": assignedEmail,
			"created_at": created, "last_message_at": lastMessage,
			"messages": messages, "awaiting_staff": awaitingStaff,
			"user_tickets": userTickets,
			"service_kind": serviceKind, "service_id": serviceID,
		})
	}
	serviceIDs := []string{}
	for _, item := range list {
		if sid, _ := item["service_id"].(string); sid != "" {
			serviceIDs = append(serviceIDs, sid)
		}
	}
	if labels := h.serviceLabels(r.Context(), claims.TenantID, serviceIDs); len(labels) > 0 {
		for _, item := range list {
			sid, _ := item["service_id"].(string)
			if label := labels[sid]; label != "" {
				item["service"] = label
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{"tickets": list})
}

func (h *Handler) AdminSupportUpdate(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := chi.URLParam(r, "id")
	var body struct {
		Status     *string `json:"status"`
		Priority   *string `json:"priority"`
		AssignToMe *bool   `json:"assign_to_me"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if body.Status != nil && !supportStatuses[*body.Status] {
		writeError(w, http.StatusBadRequest, "unknown status")
		return
	}
	if body.Priority != nil && !supportPriorities[*body.Priority] {
		writeError(w, http.StatusBadRequest, "unknown priority")
		return
	}

	var assignee any
	assignChanged := body.AssignToMe != nil
	if assignChanged && *body.AssignToMe {
		assignee = claims.UserID
	}

	tag, err := h.dbOf(r.Context()).Exec(r.Context(), `
		UPDATE core.support_tickets SET
			status            = COALESCE($3, status),
			priority          = COALESCE($4, priority),
			assigned_admin_id = CASE WHEN $5::bool THEN $6::uuid ELSE assigned_admin_id END,
			closed_at         = CASE WHEN $3 = 'closed' THEN now()
			                         WHEN $3 IS NOT NULL THEN NULL
			                         ELSE closed_at END,
			updated_at        = now()
		WHERE id = $1 AND tenant_id = $2
	`, id, claims.TenantID, body.Status, body.Priority, assignChanged, assignee)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "ticket not found")
		return
	}
	audit(r.Context(), h.dbOf(r.Context()), claims.TenantID, claims.UserID, "support.update", "ticket:"+id,
		map[string]any{"status": body.Status, "priority": body.Priority})
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}
