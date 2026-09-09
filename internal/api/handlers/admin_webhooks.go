package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

var webhookEvents = []struct{ Key, Label string }{
	{"server.created", "Сервер создан"},
	{"server.status", "Сервер сменил статус"},
	{"server.suspended", "Сервер приостановлен за неоплату"},
	{"server.renewed", "Аренда продлена"},
	{"payment.completed", "Платёж прошёл"},
	{"payment.refunded", "Платёж возвращён"},
	{"node.offline", "Нода ушла в офлайн"},
	{"node.maintenance", "Техработы на локации"},
	{"job.failed", "Задача упала"},
}

type webhookBody struct {
	URL         string   `json:"url"`
	Events      []string `json:"events"`
	Description string   `json:"description"`
	Active      *bool    `json:"active"`
}

func (h *Handler) AdminWebhooksList(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	rows, err := h.readerOf(r.Context()).Query(r.Context(), `
		SELECT w.id::text, w.url, w.events, w.active, w.description, w.created_at,
		       (SELECT COUNT(*) FROM core.webhook_deliveries d
		         WHERE d.webhook_id = w.id AND d.status = 'failed'),
		       (SELECT MAX(created_at) FROM core.webhook_deliveries d WHERE d.webhook_id = w.id)
		FROM core.webhooks w
		WHERE w.tenant_id = $1
		ORDER BY w.created_at DESC
	`, claims.TenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()

	list := []map[string]any{}
	for rows.Next() {
		var id, hookURL, description string
		var events []byte
		var active bool
		var createdAt time.Time
		var failed int
		var lastAt *time.Time
		if rows.Scan(&id, &hookURL, &events, &active, &description, &createdAt, &failed, &lastAt) != nil {
			continue
		}
		list = append(list, map[string]any{
			"id": id, "url": hookURL, "events": json.RawMessage(events),
			"active": active, "description": description,
			"created_at":   createdAt.Format(time.RFC3339),
			"failed_count": failed,
			"last_sent_at": timeOrNil(lastAt),
		})
	}

	catalog := []map[string]string{}
	for _, e := range webhookEvents {
		catalog = append(catalog, map[string]string{"key": e.Key, "label": e.Label})
	}
	writeJSON(w, http.StatusOK, map[string]any{"webhooks": list, "events": catalog})
}

func (h *Handler) AdminWebhookCreate(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var body webhookBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	target := strings.TrimSpace(body.URL)
	parsed, err := url.Parse(target)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		writeError(w, http.StatusBadRequest, "нужен полный адрес вида https://example.com/hook")
		return
	}

	known := map[string]bool{}
	for _, e := range webhookEvents {
		known[e.Key] = true
	}
	events := []string{}
	for _, e := range body.Events {
		e = strings.TrimSpace(e)
		if !known[e] {
			writeError(w, http.StatusBadRequest, "неизвестное событие: "+e)
			return
		}
		events = append(events, e)
	}
	if len(events) == 0 {
		writeError(w, http.StatusBadRequest, "выберите хотя бы одно событие")
		return
	}

	raw := make([]byte, 24)
	if _, err := rand.Read(raw); err != nil {
		writeError(w, http.StatusInternalServerError, "не удалось создать секрет")
		return
	}
	secret := hex.EncodeToString(raw)
	eventsJSON, _ := json.Marshal(events)

	var id string
	if h.dbOf(r.Context()).QueryRow(r.Context(), `
		INSERT INTO core.webhooks (tenant_id, url, events, secret, description)
		VALUES ($1, $2, $3::jsonb, $4, $5)
		RETURNING id::text
	`, claims.TenantID, target, eventsJSON, secret, strings.TrimSpace(body.Description)).Scan(&id) != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	audit(r.Context(), h.dbOf(r.Context()), claims.TenantID, claims.UserID, "webhook.create", "webhook:"+id,
		map[string]any{"url": target, "events": events})
	h.auditAlert(r.Context(), claims.TenantID, claims.UserID, claims.Email, "webhook.create", target)

	writeJSON(w, http.StatusCreated, map[string]any{
		"id": id, "url": target, "secret": secret,
	})
}

func (h *Handler) AdminWebhookUpdate(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := chi.URLParam(r, "id")
	var body webhookBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	var eventsJSON any
	if body.Events != nil {
		known := map[string]bool{}
		for _, e := range webhookEvents {
			known[e.Key] = true
		}
		for _, e := range body.Events {
			if !known[strings.TrimSpace(e)] {
				writeError(w, http.StatusBadRequest, "неизвестное событие: "+e)
				return
			}
		}
		raw, _ := json.Marshal(body.Events)
		eventsJSON = raw
	}

	tag, err := h.dbOf(r.Context()).Exec(r.Context(), `
		UPDATE core.webhooks SET
			active      = COALESCE($3, active),
			events      = COALESCE($4::jsonb, events),
			description = COALESCE(NULLIF($5, ''), description)
		WHERE id = $1 AND tenant_id = $2
	`, id, claims.TenantID, body.Active, eventsJSON, strings.TrimSpace(body.Description))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "вебхук не найден")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (h *Handler) AdminWebhookDelete(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := chi.URLParam(r, "id")
	tag, err := h.dbOf(r.Context()).Exec(r.Context(), `DELETE FROM core.webhooks WHERE id = $1 AND tenant_id = $2`,
		id, claims.TenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "вебхук не найден")
		return
	}
	audit(r.Context(), h.dbOf(r.Context()), claims.TenantID, claims.UserID, "webhook.delete", "webhook:"+id, nil)
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) AdminWebhookDeliveries(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := chi.URLParam(r, "id")
	rows, err := h.readerOf(r.Context()).Query(r.Context(), `
		SELECT event, status, attempts, response_code, COALESCE(error, ''), created_at, delivered_at
		FROM core.webhook_deliveries
		WHERE tenant_id = $1 AND webhook_id = $2
		ORDER BY created_at DESC
		LIMIT 50
	`, claims.TenantID, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()

	list := []map[string]any{}
	for rows.Next() {
		var event, status, errText string
		var attempts int
		var code *int
		var createdAt time.Time
		var deliveredAt *time.Time
		if rows.Scan(&event, &status, &attempts, &code, &errText, &createdAt, &deliveredAt) != nil {
			continue
		}
		item := map[string]any{
			"event": event, "status": status, "attempts": attempts,
			"error":        nilIfEmpty(errText),
			"created_at":   createdAt.Format(time.RFC3339),
			"delivered_at": timeOrNil(deliveredAt),
		}
		if code != nil {
			item["response_code"] = *code
		}
		list = append(list, item)
	}
	writeJSON(w, http.StatusOK, map[string]any{"deliveries": list})
}

func (h *Handler) AdminWebhookTest(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := chi.URLParam(r, "id")
	var exists bool
	if h.dbOf(r.Context()).QueryRow(r.Context(), `
		SELECT EXISTS(SELECT 1 FROM core.webhooks WHERE id = $1 AND tenant_id = $2)
	`, id, claims.TenantID).Scan(&exists) != nil || !exists {
		writeError(w, http.StatusNotFound, "вебхук не найден")
		return
	}
	payload, _ := json.Marshal(map[string]any{
		"test": true, "sent_at": time.Now().Format(time.RFC3339),
	})
	if _, err := h.dbOf(r.Context()).Exec(r.Context(), `
		INSERT INTO core.webhook_deliveries (tenant_id, webhook_id, event, payload, next_retry_at)
		VALUES ($1, $2, 'webhook.test', $3::jsonb, now())
	`, claims.TenantID, id, payload); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "queued"})
}

func (h *Handler) emitWebhook(ctx context.Context, tenantID, event string, payload map[string]any) {
	body, err := json.Marshal(payload)
	if err != nil {
		return
	}
	_, _ = h.dbOf(ctx).Exec(ctx, `
		INSERT INTO core.webhook_deliveries (tenant_id, webhook_id, event, payload, next_retry_at)
		SELECT $1, w.id, $2, $3::jsonb, now()
		FROM core.webhooks w
		WHERE w.tenant_id = $1 AND w.active = true
		  AND (jsonb_array_length(w.events) = 0 OR w.events ? $2)
	`, tenantID, event, body)
}
