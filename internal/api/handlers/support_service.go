package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

type supportService struct {
	Kind  string `json:"kind"`
	ID    string `json:"id"`
	Label string `json:"label"`
}

func (h *Handler) userServices(ctx context.Context, userID string) []supportService {
	out := []supportService{}

	rows, err := h.dbOf(ctx).Query(ctx, `
		SELECT 'server', id::text, name FROM core.servers
		WHERE user_id = $1
		UNION ALL
		SELECT 'hosting', id::text,
		       COALESCE(NULLIF(primary_domain, ''), username)
		FROM core.hosting_accounts
		WHERE user_id = $1
		ORDER BY 1, 3
	`, userID)
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

func (h *Handler) serviceLabels(ctx context.Context, ids []string) map[string]string {
	labels := map[string]string{}
	if len(ids) == 0 {
		return labels
	}

	rows, err := h.dbOf(ctx).Query(ctx, `
		SELECT id::text, name FROM core.servers
		WHERE id = ANY($1::uuid[])
		UNION ALL
		SELECT id::text, COALESCE(NULLIF(primary_domain, ''), username)
		FROM core.hosting_accounts
		WHERE id = ANY($1::uuid[])
	`, ids)
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
		UPDATE core.support_tickets SET notify = $2, updated_at = now()
		WHERE id = $1::uuid AND user_id = $3
	`, id, body.Notify, claims.UserID)
	if err != nil || tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "обращение не найдено")
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"notify": body.Notify})
}

func (h *Handler) supportSLA(ctx context.Context) []map[string]any {
	rows, err := h.dbOf(ctx).Query(ctx, `
		SELECT t.category,
		       ROUND(AVG(EXTRACT(epoch FROM first_reply.at - t.created_at)) / 60)::int
		FROM core.support_tickets t
		JOIN LATERAL (
		    SELECT MIN(m.created_at) AS at
		    FROM core.support_messages m
		    WHERE m.ticket_id = t.id AND m.is_staff
		) first_reply ON first_reply.at IS NOT NULL
		WHERE t.created_at > now() - interval '7 days'
		GROUP BY t.category
		ORDER BY 2
	`)
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
