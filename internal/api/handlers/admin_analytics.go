package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

func analyticsDays(r *http.Request) int {
	days := 30
	if v := strings.TrimSpace(r.URL.Query().Get("days")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 730 {
			days = n
		}
	}
	return days
}

func (h *Handler) AdminAnalytics(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ctx := r.Context()
	days := analyticsDays(r)
	tenant := claims.TenantID

	writeJSON(w, http.StatusOK, map[string]any{
		"days":      days,
		"revenue":   h.analyticsRevenue(ctx, tenant, days),
		"series":    h.analyticsSeries(ctx, tenant, days),
		"sources":   h.analyticsBySource(ctx, tenant, days),
		"recurring": h.analyticsRecurring(ctx, tenant),
		"customers": h.analyticsCustomers(ctx, tenant, days),
		"churn":     h.analyticsChurn(ctx, tenant, days),
		"breakdown": map[string]any{
			"games":     h.analyticsBreakdown(ctx, tenant, "game"),
			"locations": h.analyticsBreakdown(ctx, tenant, "location"),
			"tariffs":   h.analyticsBreakdown(ctx, tenant, "tariff"),
		},
	})
}

func (h *Handler) analyticsRevenue(ctx context.Context, tenantID string, days int) map[string]any {
	var charges, chargesPrev, topups, topupsPrev float64
	var payers int

	_ = h.readerOf(ctx).QueryRow(ctx, `
		SELECT
			COALESCE(SUM(ABS(amount)) FILTER (WHERE created_at >= now() - make_interval(days => $2::int)), 0),
			COALESCE(SUM(ABS(amount)) FILTER (
				WHERE created_at >= now() - make_interval(days => $2::int * 2)
				  AND created_at <  now() - make_interval(days => $2::int)), 0)
		FROM core.transactions
		WHERE tenant_id = $1 AND type = 'debit'
	`, tenantID, days).Scan(&charges, &chargesPrev)

	_ = h.readerOf(ctx).QueryRow(ctx, `
		SELECT
			COALESCE(SUM(amount) FILTER (WHERE created_at >= now() - make_interval(days => $2::int)), 0),
			COALESCE(SUM(amount) FILTER (
				WHERE created_at >= now() - make_interval(days => $2::int * 2)
				  AND created_at <  now() - make_interval(days => $2::int)), 0),
			COUNT(DISTINCT user_id) FILTER (WHERE created_at >= now() - make_interval(days => $2::int))
		FROM core.payments
		WHERE tenant_id = $1 AND status IN ('success', 'succeeded', 'completed', 'paid')
	`, tenantID, days).Scan(&topups, &topupsPrev, &payers)

	arpu := 0.0
	if payers > 0 {
		arpu = charges / float64(payers)
	}
	return map[string]any{
		"charges": charges, "charges_prev": chargesPrev,
		"topups": topups, "topups_prev": topupsPrev,
		"payers": payers, "arpu": arpu,
		"charges_change_percent": changePercent(charges, chargesPrev),
		"topups_change_percent":  changePercent(topups, topupsPrev),
	}
}

func changePercent(now, prev float64) float64 {
	if prev == 0 {
		if now == 0 {
			return 0
		}
		return 100
	}
	return (now - prev) / prev * 100
}

func (h *Handler) analyticsSeries(ctx context.Context, tenantID string, days int) []map[string]any {
	rows, err := h.readerOf(ctx).Query(ctx, `
		WITH d AS (
			SELECT generate_series(
				date_trunc('day', now()) - make_interval(days => $2::int - 1),
				date_trunc('day', now()),
				interval '1 day'
			) AS day
		)
		SELECT d.day::date::text,
		       COALESCE((
		           SELECT SUM(ABS(t.amount)) FROM core.transactions t
		           WHERE t.tenant_id = $1 AND t.type = 'debit'
		             AND t.created_at >= d.day AND t.created_at < d.day + interval '1 day'
		       ), 0),
		       COALESCE((
		           SELECT SUM(p.amount) FROM core.payments p
		           WHERE p.tenant_id = $1
		             AND p.status IN ('success', 'succeeded', 'completed', 'paid')
		             AND p.created_at >= d.day AND p.created_at < d.day + interval '1 day'
		       ), 0),
		       COALESCE((
		           SELECT COUNT(*) FROM core.servers s
		           WHERE s.tenant_id = $1
		             AND s.created_at >= d.day AND s.created_at < d.day + interval '1 day'
		       ), 0)
		FROM d ORDER BY d.day
	`, tenantID, days)
	if err != nil {
		return []map[string]any{}
	}
	defer rows.Close()

	out := []map[string]any{}
	for rows.Next() {
		var day string
		var charges, topups float64
		var newServers int
		if rows.Scan(&day, &charges, &topups, &newServers) == nil {
			out = append(out, map[string]any{
				"date": day, "charges": charges, "topups": topups, "new_servers": newServers,
			})
		}
	}
	return out
}

var analyticsSourceLabels = map[string]string{
	"server_rent":      "Аренда серверов",
	"server_renew":     "Продления",
	"server_tariff":    "Смена тарифа",
	"server_resources": "Изменение ресурсов",
	"admin_adjustment": "Корректировки админом",
	"admin_user":       "Корректировки админом",
	"":                 "Без источника (до 24.08.2026)",
}

func (h *Handler) analyticsBySource(ctx context.Context, tenantID string, days int) []map[string]any {
	rows, err := h.readerOf(ctx).Query(ctx, `
		SELECT COALESCE(source_type, ''), COUNT(*), COALESCE(SUM(ABS(amount)), 0)
		FROM core.transactions
		WHERE tenant_id = $1 AND type = 'debit'
		  AND created_at >= now() - make_interval(days => $2::int)
		GROUP BY 1 ORDER BY 3 DESC
	`, tenantID, days)
	if err != nil {
		return []map[string]any{}
	}
	defer rows.Close()

	out := []map[string]any{}
	for rows.Next() {
		var source string
		var count int
		var amount float64
		if rows.Scan(&source, &count, &amount) != nil {
			continue
		}
		label, known := analyticsSourceLabels[source]
		if !known {
			label = source
		}
		out = append(out, map[string]any{
			"source": source, "label": label, "count": count, "amount": amount,
		})
	}
	return out
}

func (h *Handler) analyticsRecurring(ctx context.Context, tenantID string) map[string]any {
	var monthly float64
	var active, expiring int
	_ = h.readerOf(ctx).QueryRow(ctx, `
		SELECT
			COALESCE(SUM(t.price_monthly), 0),
			COUNT(*),
			COUNT(*) FILTER (WHERE s.expires_at IS NOT NULL
			                   AND s.expires_at < now() + interval '7 days')
		FROM core.servers s
		LEFT JOIN core.tariffs t ON t.id = s.tariff_id
		WHERE s.tenant_id = $1
		  AND COALESCE(s.is_blocked, false) = false
		  AND (s.expires_at IS NULL OR s.expires_at > now())
	`, tenantID).Scan(&monthly, &active, &expiring)

	arps := 0.0
	if active > 0 {
		arps = monthly / float64(active)
	}
	return map[string]any{
		"monthly": monthly, "active_servers": active,
		"expiring_7d": expiring, "avg_per_server": arps,
	}
}

func (h *Handler) analyticsCustomers(ctx context.Context, tenantID string, days int) map[string]any {
	var registered, withServer, withPayment, total int
	_ = h.readerOf(ctx).QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE u.created_at >= now() - make_interval(days => $2::int)),
			COUNT(*) FILTER (WHERE u.created_at >= now() - make_interval(days => $2::int)
			                   AND EXISTS (SELECT 1 FROM core.servers s WHERE s.user_id = u.id)),
			COUNT(*) FILTER (WHERE u.created_at >= now() - make_interval(days => $2::int)
			                   AND EXISTS (SELECT 1 FROM core.payments p
			                               WHERE p.user_id = u.id
			                                 AND p.status IN ('success','succeeded','completed','paid'))),
			COUNT(*)
		FROM core.users u
		WHERE u.tenant_id = $1 AND u.role = 'user'
	`, tenantID, days).Scan(&registered, &withServer, &withPayment, &total)

	return map[string]any{
		"total": total, "registered": registered,
		"with_server": withServer, "with_payment": withPayment,
		"server_conversion":  percentOf(withServer, registered),
		"payment_conversion": percentOf(withPayment, registered),
	}
}

func percentOf(part, whole int) float64 {
	if whole == 0 {
		return 0
	}
	return float64(part) / float64(whole) * 100
}

func (h *Handler) analyticsChurn(ctx context.Context, tenantID string, days int) map[string]any {
	var expired, renewed, suspended int
	_ = h.readerOf(ctx).QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE s.expires_at IS NOT NULL
			                   AND s.expires_at < now()
			                   AND s.expires_at >= now() - make_interval(days => $2::int)),
			COUNT(*) FILTER (WHERE EXISTS (
			                     SELECT 1 FROM core.transactions t
			                     WHERE t.tenant_id = s.tenant_id
			                       AND t.source_type = 'server_renew'
			                       AND t.source_id = s.id
			                       AND t.created_at >= now() - make_interval(days => $2::int))),
			COUNT(*) FILTER (WHERE s.suspended_at IS NOT NULL)
		FROM core.servers s
		WHERE s.tenant_id = $1
	`, tenantID, days).Scan(&expired, &renewed, &suspended)

	return map[string]any{
		"expired": expired, "renewed": renewed, "suspended": suspended,
	}
}

func (h *Handler) analyticsBreakdown(ctx context.Context, tenantID, dimension string) []map[string]any {
	var groupExpr, joinExpr string
	switch dimension {
	case "game":
		groupExpr = "COALESCE(g.name, s.game_id)"
		joinExpr = "LEFT JOIN core.games g ON g.tenant_id = s.tenant_id AND g.slug = s.game_id"
	case "location":
		groupExpr = "COALESCE(n.name, '—')"
		joinExpr = "LEFT JOIN core.nodes n ON n.id = s.node_id"
	case "tariff":
		groupExpr = "COALESCE(t.name, 'без тарифа')"
		joinExpr = ""
	default:
		return []map[string]any{}
	}

	query := fmt.Sprintf(`
		SELECT %s AS label, COUNT(*)::int, COALESCE(SUM(t.price_monthly), 0)
		FROM core.servers s
		LEFT JOIN core.tariffs t ON t.id = s.tariff_id
		%s
		WHERE s.tenant_id = $1
		  AND COALESCE(s.is_blocked, false) = false
		  AND (s.expires_at IS NULL OR s.expires_at > now())
		GROUP BY 1
		ORDER BY 3 DESC, 2 DESC
		LIMIT 20
	`, groupExpr, joinExpr)

	rows, err := h.readerOf(ctx).Query(ctx, query, tenantID)
	if err != nil {
		return []map[string]any{}
	}
	defer rows.Close()

	out := []map[string]any{}
	for rows.Next() {
		var label string
		var servers int
		var monthly float64
		if rows.Scan(&label, &servers, &monthly) == nil {
			out = append(out, map[string]any{
				"label": label, "servers": servers, "monthly": monthly,
			})
		}
	}
	return out
}

func (h *Handler) AdminAnalyticsExport(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	days := analyticsDays(r)
	series := h.analyticsSeries(r.Context(), claims.TenantID, days)

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition",
		fmt.Sprintf("attachment; filename=\"vortanix-analytics-%s.csv\"", time.Now().Format("2006-01-02")))
	_, _ = w.Write([]byte("\xEF\xBB\xBF"))
	_, _ = w.Write([]byte("Дата;Списания;Пополнения;Новых серверов\n"))
	for _, row := range series {
		_, _ = fmt.Fprintf(w, "%s;%.2f;%.2f;%d\n",
			row["date"], row["charges"], row["topups"], row["new_servers"])
	}
}

func (h *Handler) AdminServerAnalytics(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	serverID := chi.URLParam(r, "id")
	rows, err := h.readerOf(r.Context()).Query(r.Context(), `
		SELECT COALESCE(source_type, ''), COALESCE(description, ''), ABS(amount), created_at
		FROM core.transactions
		WHERE tenant_id = $1 AND source_id = $2::uuid AND type = 'debit'
		ORDER BY created_at DESC
		LIMIT 100
	`, claims.TenantID, serverID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()

	list := []map[string]any{}
	total := 0.0
	for rows.Next() {
		var source, desc string
		var amount float64
		var createdAt time.Time
		if rows.Scan(&source, &desc, &amount, &createdAt) != nil {
			continue
		}
		total += amount
		list = append(list, map[string]any{
			"source": source, "description": desc, "amount": amount,
			"created_at": createdAt.Format(time.RFC3339),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"charges": list, "total": total})
}
