package handlers

import (
	"net/http"
	"strconv"
	"strings"
)

type searchHit struct {
	Kind     string `json:"kind"`
	ID       string `json:"id"`
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
	URL      string `json:"url"`
}

func (h *Handler) AdminSearch(w http.ResponseWriter, r *http.Request) {
	_, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if len(query) < 2 {
		writeJSON(w, http.StatusOK, map[string]any{"results": []searchHit{}})
		return
	}
	ctx := r.Context()
	results := []searchHit{}

	if rows, err := h.readerOf(ctx).Query(ctx, `
		SELECT u.id::text, u.email, COALESCE(p.first_name, ''), COALESCE(p.last_name, ''), u.role
		FROM core.users u
		LEFT JOIN core.user_profiles p ON p.user_id = u.id
		WHERE (u.email ILIKE '%' || $1 || '%'
		       OR COALESCE(p.first_name, '') ILIKE '%' || $1 || '%'
		       OR COALESCE(p.last_name, '') ILIKE '%' || $1 || '%')
		ORDER BY u.created_at DESC
		LIMIT 5
	`, query); err == nil {
		for rows.Next() {
			var id, email, first, last, role string
			if rows.Scan(&id, &email, &first, &last, &role) == nil {
				name := strings.TrimSpace(first + " " + last)
				results = append(results, searchHit{
					Kind: "user", ID: id, Title: email,
					Subtitle: strings.TrimSpace(name + " · " + roleLabelRU(role)),
					URL:      "/admin/users/" + id,
				})
			}
		}
		rows.Close()
	}

	if rows, err := h.readerOf(ctx).Query(ctx, `
		SELECT s.id::text, s.name, COALESCE(s.ip_address, ''), COALESCE(s.primary_port, 0),
		       s.game_id, COALESCE(u.email, ''), COALESCE(n.name, '')
		FROM core.servers s
		LEFT JOIN core.users u ON u.id = s.user_id
		LEFT JOIN core.nodes n ON n.id = s.node_id
		WHERE (s.name ILIKE '%' || $1 || '%'
		       OR COALESCE(s.ip_address, '') ILIKE '%' || $1 || '%'
		       OR s.id::text ILIKE '%' || $1 || '%'
		       OR COALESCE(u.email, '') ILIKE '%' || $1 || '%')
		ORDER BY s.created_at DESC
		LIMIT 5
	`, query); err == nil {
		for rows.Next() {
			var id, name, ip, game, email, node string
			var port int
			if rows.Scan(&id, &name, &ip, &port, &game, &email, &node) == nil {
				address := ip
				if port > 0 {
					address += ":" + strconv.Itoa(port)
				}
				results = append(results, searchHit{
					Kind: "server", ID: id, Title: name,
					Subtitle: strings.TrimSpace(game + " · " + address + " · " + email + " · " + node),
					URL:      "/admin/servers/" + id,
				})
			}
		}
		rows.Close()
	}

	if rows, err := h.readerOf(ctx).Query(ctx, `
		SELECT p.id::text, p.amount::float8, p.currency, p.status, COALESCE(p.provider, ''),
		       COALESCE(u.email, ''), COALESCE(p.provider_payment_id, '')
		FROM core.payments p
		LEFT JOIN core.users u ON u.id = p.user_id
		WHERE (COALESCE(p.provider_payment_id, '') ILIKE '%' || $1 || '%'
		       OR p.id::text ILIKE '%' || $1 || '%'
		       OR COALESCE(u.email, '') ILIKE '%' || $1 || '%')
		ORDER BY p.created_at DESC
		LIMIT 5
	`, query); err == nil {
		for rows.Next() {
			var id, currency, status, provider, email, external string
			var amount float64
			if rows.Scan(&id, &amount, &currency, &status, &provider, &email, &external) == nil {
				results = append(results, searchHit{
					Kind: "payment", ID: id,
					Title:    formatMoney(amount) + " " + strings.ToUpper(currency),
					Subtitle: strings.TrimSpace(status + " · " + provider + " · " + email + " · " + external),
					URL:      "/admin/billing",
				})
			}
		}
		rows.Close()
	}

	if rows, err := h.readerOf(ctx).Query(ctx, `
		SELECT t.id::text, t.subject, t.status, COALESCE(u.email, '')
		FROM core.support_tickets t
		LEFT JOIN core.users u ON u.id = t.user_id
		WHERE (t.subject ILIKE '%' || $1 || '%' OR COALESCE(u.email, '') ILIKE '%' || $1 || '%')
		ORDER BY t.created_at DESC
		LIMIT 5
	`, query); err == nil {
		for rows.Next() {
			var id, subject, status, email string
			if rows.Scan(&id, &subject, &status, &email) == nil {
				results = append(results, searchHit{
					Kind: "ticket", ID: id, Title: subject,
					Subtitle: strings.TrimSpace(status + " · " + email),
					URL:      "/admin/support/" + id,
				})
			}
		}
		rows.Close()
	}

	if rows, err := h.readerOf(ctx).Query(ctx, `
		SELECT id::text, name, COALESCE(fqdn, ''), status
		FROM core.nodes
		WHERE (name ILIKE '%' || $1 || '%' OR COALESCE(fqdn, '') ILIKE '%' || $1 || '%')
		ORDER BY name
		LIMIT 5
	`, query); err == nil {
		for rows.Next() {
			var id, name, fqdn, status string
			if rows.Scan(&id, &name, &fqdn, &status) == nil {
				results = append(results, searchHit{
					Kind: "location", ID: id, Title: name,
					Subtitle: strings.TrimSpace(fqdn + " · " + status),
					URL:      "/admin/locations/" + id,
				})
			}
		}
		rows.Close()
	}

	writeJSON(w, http.StatusOK, map[string]any{"results": results, "query": query})
}

func roleLabelRU(role string) string {
	switch role {
	case "owner":
		return "владелец"
	case "admin":
		return "администратор"
	case "support":
		return "поддержка"
	default:
		return "клиент"
	}
}
