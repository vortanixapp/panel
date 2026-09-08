package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

// Услуга обращения и уведомления по нему.

type supportService struct {
	Kind  string `json:"kind"`
	ID    string `json:"id"`
	Label string `json:"label"`
}

// userServices — услуги, к которым клиент может привязать обращение.
//
// Игровые серверы и веб-аккаунты лежат в разных таблицах, поэтому собираем их
// в один список здесь: форме нужен один селект, а не два, между которыми надо
// догадаться выбрать.
func (h *Handler) userServices(ctx context.Context, tenantID, userID string) []supportService {
	out := []supportService{}

	rows, err := h.dbOf(ctx).Query(ctx, `
		SELECT 'server', id::text, name FROM core.servers
		WHERE tenant_id = $1 AND user_id = $2
		UNION ALL
		SELECT 'hosting', id::text,
		       COALESCE(NULLIF(primary_domain, ''), username)
		FROM core.hosting_accounts
		WHERE tenant_id = $1 AND user_id = $2
		ORDER BY 1, 3
	`, tenantID, userID)
	if err != nil {
		return out
	}
	defer rows.Close()

	for rows.Next() {
		var s supportService
		if rows.Scan(&s.Kind, &s.ID, &s.Label) == nil {
			out = append(out, s)
		}
	}
	return out
}

// serviceLabels разрешает названия услуг для набора обращений одним запросом.
//
// Название не храним копией в тикете: сервер переименуют — и в поддержке
// останется старое имя, по которому уже не найти, о чём шла речь.
func (h *Handler) serviceLabels(ctx context.Context, tenantID string, ids []string) map[string]string {
	labels := map[string]string{}
	if len(ids) == 0 {
		return labels
	}

	rows, err := h.dbOf(ctx).Query(ctx, `
		SELECT id::text, name FROM core.servers
		WHERE tenant_id = $1 AND id = ANY($2::uuid[])
		UNION ALL
		SELECT id::text, COALESCE(NULLIF(primary_domain, ''), username)
		FROM core.hosting_accounts
		WHERE tenant_id = $1 AND id = ANY($2::uuid[])
	`, tenantID, ids)
	if err != nil {
		return labels
	}
	defer rows.Close()

	for rows.Next() {
		var id, label string
		if rows.Scan(&id, &label) == nil {
			labels[id] = label
		}
	}
	return labels
}

// SetTicketNotify включает и выключает уведомления по обращению.
//
// Владелец обращения — единственный, кто может это менять: чужой тикет
// заглушить нельзя, а сотруднику отключать уведомления клиенту незачем.
func (h *Handler) SetTicketNotify(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := chi.URLParam(r, "id")
	var body struct {
		Notify bool `json:"notify"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	tag, err := h.dbOf(r.Context()).Exec(r.Context(), `
		UPDATE core.support_tickets SET notify = $3, updated_at = now()
		WHERE id = $1::uuid AND tenant_id = $2 AND user_id = $4
	`, id, claims.TenantID, body.Notify, claims.UserID)
	if err != nil || tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "обращение не найдено")
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"notify": body.Notify})
}

// supportSLA — среднее время первого ответа по отделам за неделю.
//
// Считаем по переписке, а не берём из настройки: цифра, выставленная руками,
// обещает клиенту то, чего никто не мерил. Учитываем только обращения, где
// поддержка действительно ответила, — иначе неотвеченные занижали бы среднее,
// просто не попадая в расчёт времени.
func (h *Handler) supportSLA(ctx context.Context, tenantID string) []map[string]any {
	rows, err := h.dbOf(ctx).Query(ctx, `
		SELECT t.category,
		       ROUND(AVG(EXTRACT(epoch FROM first_reply.at - t.created_at)) / 60)::int
		FROM core.support_tickets t
		JOIN LATERAL (
		    SELECT MIN(m.created_at) AS at
		    FROM core.support_messages m
		    WHERE m.ticket_id = t.id AND m.is_staff
		) first_reply ON first_reply.at IS NOT NULL
		WHERE t.tenant_id = $1 AND t.created_at > now() - interval '7 days'
		GROUP BY t.category
		ORDER BY 2
	`, tenantID)
	if err != nil {
		return nil
	}
	defer rows.Close()

	out := []map[string]any{}
	for rows.Next() {
		var category string
		var minutes int
		if rows.Scan(&category, &minutes) != nil {
			continue
		}
		out = append(out, map[string]any{"category": category, "minutes": minutes})
	}
	return out
}

func normalizeServiceKind(kind string) string {
	switch strings.TrimSpace(kind) {
	case "server", "hosting":
		return strings.TrimSpace(kind)
	default:
		return ""
	}
}
