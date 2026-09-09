package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ctx := r.Context()

	cacheKey := "panel:dashboard:" + claims.UserID
	var cached map[string]any
	if hit, err := h.cache.GetJSON(ctx, cacheKey, &cached); err == nil && hit {
		writeJSON(w, http.StatusOK, cached)
		return
	}

	ownerFilter := ` AND user_id = $1`
	args := []any{claims.UserID}

	db := h.readerOf(ctx)
	var total, running int
	if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM core.servers WHERE true`+ownerFilter, args...).Scan(&total); err != nil {
		log.Printf("главная кабинета: счёт серверов: %v", err)
		writeError(w, http.StatusInternalServerError, "не удалось прочитать данные кабинета")
		return
	}
	_ = db.QueryRow(ctx, `SELECT COUNT(*) FROM core.servers WHERE status = 'running'`+ownerFilter, args...).Scan(&running)

	var expiringSoon int
	_ = db.QueryRow(ctx, `
		SELECT COUNT(*) FROM core.servers
		WHERE true`+ownerFilter+`
		  AND expires_at IS NOT NULL
		  AND expires_at >= now()
		  AND expires_at < now() + interval '7 days'
	`, args...).Scan(&expiringSoon)

	var nextChargeText string
	var nextExpiry *time.Time
	_ = db.QueryRow(ctx, `
		SELECT expires_at FROM core.servers
		WHERE true`+ownerFilter+` AND expires_at IS NOT NULL
		ORDER BY expires_at ASC LIMIT 1
	`, args...).Scan(&nextExpiry)
	if nextExpiry != nil {
		nextChargeText = nextExpiry.Format("02.01.2006")
	} else {
		nextChargeText = "—"
	}

	wallets := []map[string]any{}
	var balance float64
	balanceCurrency := h.defaultCurrency(ctx)
	if wrows, werr := db.Query(ctx, `
		SELECT currency, COALESCE(balance, 0), COALESCE(is_default, false)
		FROM core.wallets WHERE user_id = $1
		ORDER BY is_default DESC, currency
	`, claims.UserID); werr == nil {
		defer wrows.Close()
		for wrows.Next() {
			var cur string
			var bal float64
			var isDefault bool
			if wrows.Scan(&cur, &bal, &isDefault) != nil {
				continue
			}
			wallets = append(wallets, map[string]any{
				"currency": cur, "balance": bal, "is_default": isDefault,
			})
			if len(wallets) == 1 {
				balance, balanceCurrency = bal, cur
			}
		}
	}

	var openTickets int
	_ = db.QueryRow(ctx, `
		SELECT COUNT(*) FROM core.support_tickets
		WHERE user_id = $1 AND status = 'open'
	`, claims.UserID).Scan(&openTickets)

	recentServers := h.listEnrichedServers(ctx, claims.UserID, false, 6)
	recentTransactions := h.queryRecentTransactions(ctx, claims.UserID)
	news := h.queryRecentNews(ctx)

	resp := map[string]any{
		"balance":                    balance,
		"balance_currency":           balanceCurrency,
		"wallets":                    wallets,
		"total_servers":              total,
		"active_servers":             running,
		"expiring_soon_count":        expiringSoon,
		"open_support_tickets_count": openTickets,
		"next_charge_text":           nextChargeText,
		"recent_servers":             recentServers,
		"recent_transactions":        recentTransactions,
		"news":                       news,
	}
	_ = h.cache.SetJSON(ctx, cacheKey, resp, 5*time.Second)
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) queryRecentTransactions(ctx context.Context, userID string) []map[string]any {
	rows, err := h.readerOf(ctx).Query(ctx, `
		SELECT t.id::text, t.type, t.amount, COALESCE(t.description, ''), t.created_at
		FROM core.transactions t
		JOIN core.wallets w ON w.id = t.wallet_id
		WHERE w.user_id = $1
		ORDER BY t.created_at DESC
		LIMIT 8
	`, userID)
	if err != nil {
		return []map[string]any{}
	}
	defer rows.Close()

	list := []map[string]any{}
	for rows.Next() {
		var id, typ, desc string
		var amount float64
		var createdAt time.Time
		if rows.Scan(&id, &typ, &amount, &desc, &createdAt) == nil {
			list = append(list, map[string]any{
				"id":          id,
				"type":        typ,
				"amount":      amount,
				"description": desc,
				"created_at":  createdAt.Format(time.RFC3339),
			})
		}
	}
	return list
}

func (h *Handler) queryRecentNews(ctx context.Context) []map[string]any {
	rows, err := h.readerOf(ctx).Query(ctx, `
		SELECT id::text, title, slug, COALESCE(excerpt, ''), published_at, meta
		FROM core.news
		WHERE active = true
		ORDER BY published_at DESC NULLS LAST
		LIMIT 3
	`)
	if err != nil {
		return []map[string]any{}
	}
	defer rows.Close()

	list := []map[string]any{}
	for rows.Next() {
		var id, title, slug, excerpt string
		var publishedAt *time.Time
		var meta []byte
		if rows.Scan(&id, &title, &slug, &excerpt, &publishedAt, &meta) != nil {
			continue
		}
		item := map[string]any{
			"id": id, "title": title, "slug": slug, "excerpt": excerpt, "image": nil,
		}
		if publishedAt != nil {
			item["published_at"] = publishedAt.Format(time.RFC3339)
		}
		if len(meta) > 0 {
			var m map[string]any
			if json.Unmarshal(meta, &m) == nil {
				if img, ok := m["image_url"].(string); ok && img != "" {
					item["image"] = img
				} else if img, ok := m["image"].(string); ok && img != "" {
					item["image"] = img
				}
			}
		}
		list = append(list, item)
	}
	return list
}

func derefStr(p *string, fallback string) string {
	if p != nil && *p != "" {
		return *p
	}
	return fallback
}

func (h *Handler) AdminDashboard(w http.ResponseWriter, r *http.Request) {
	_, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ctx := r.Context()
	var servers, nodes, online int
	db := h.readerOf(ctx)
	_ = db.QueryRow(ctx, `SELECT COUNT(*) FROM core.servers`).Scan(&servers)
	_ = db.QueryRow(ctx, `SELECT COUNT(*) FROM core.nodes`).Scan(&nodes)
	_ = db.QueryRow(ctx, `SELECT COUNT(*) FROM core.nodes WHERE status = 'online'`).Scan(&online)
	var users int
	_ = db.QueryRow(ctx, `SELECT COUNT(*) FROM core.users`).Scan(&users)

	var rev24h, rev7d, rev30d float64
	_ = db.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount), 0) FROM core.payments
		WHERE status IN ('success','succeeded','completed')
		  AND created_at >= now() - interval '24 hours'
	`).Scan(&rev24h)
	_ = db.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount), 0) FROM core.payments
		WHERE status IN ('success','succeeded','completed')
		  AND created_at >= now() - interval '7 days'
	`).Scan(&rev7d)
	_ = db.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount), 0) FROM core.payments
		WHERE status IN ('success','succeeded','completed')
		  AND created_at >= now() - interval '30 days'
	`).Scan(&rev30d)

	var provInstalling, provFailed int
	_ = db.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE LOWER(COALESCE(provisioning_status, '')) IN ('installing','provisioning','pending')),
			COUNT(*) FILTER (WHERE LOWER(COALESCE(provisioning_status, '')) = 'failed')
		FROM core.servers
	`).Scan(&provInstalling, &provFailed)

	var expiring, failedSrv int
	_ = db.QueryRow(ctx, `
		SELECT COUNT(*) FROM core.servers
		WHERE expires_at IS NOT NULL
		  AND expires_at >= now()
		  AND expires_at < now() + interval '7 days'
	`).Scan(&expiring)
	_ = db.QueryRow(ctx, `
		SELECT COUNT(*) FROM core.servers
		WHERE (LOWER(COALESCE(provisioning_status, '')) = 'failed'
		       OR (provisioning_error IS NOT NULL AND provisioning_error <> ''))
	`).Scan(&failedSrv)

	writeJSON(w, http.StatusOK, map[string]any{
		"servers":                 servers,
		"nodes":                   nodes,
		"online_nodes":            online,
		"users":                   users,
		"revenue_24h":             rev24h,
		"revenue_7d":              rev7d,
		"revenue_30d":             rev30d,
		"provisioning_installing": provInstalling,
		"provisioning_failed":     provFailed,
		"expiring_count":          expiring,
		"failed_count":            failedSrv,
	})
}
