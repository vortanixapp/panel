package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var walletCurrencies = []string{"RUB", "USD", "EUR", "UAH"}

func (h *Handler) GetBilling(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	wallets := h.listUserWallets(r, claims.UserID)
	selected := selectWallet(wallets, r.URL.Query().Get("wallet_id"))

	var creditsTotal, debitsTotal float64
	txs := []map[string]any{}
	page := 1
	pageSize := 20
	if p := r.URL.Query().Get("page"); p != "" {
		if n, err := strconv.Atoi(p); err == nil && n > 0 {
			page = n
		}
	}
	if ps := r.URL.Query().Get("page_size"); ps != "" {
		if n, err := strconv.Atoi(ps); err == nil && n > 0 && n <= 100 {
			pageSize = n
		}
	}
	offset := (page - 1) * pageSize
	var txTotal int
	if selected != nil {
		wid := selected["id"].(string)
		_ = h.dbOf(r.Context()).QueryRow(r.Context(), `SELECT COUNT(*) FROM core.transactions WHERE wallet_id = $1`, wid).Scan(&txTotal)
		txRows, _ := h.dbOf(r.Context()).Query(r.Context(), `
			SELECT t.id::text, t.type, t.amount, t.description, t.created_at::text
			FROM core.transactions t
			WHERE t.wallet_id = $1 ORDER BY t.created_at DESC LIMIT $2 OFFSET $3
		`, wid, pageSize, offset)
		if txRows != nil {
			defer txRows.Close()
			for txRows.Next() {
				var id, typ, desc, created string
				var amount float64
				if txRows.Scan(&id, &typ, &amount, &desc, &created) == nil {
					txs = append(txs, map[string]any{"id": id, "type": typ, "amount": amount, "description": desc, "created_at": created})
				}
			}
		}
		_ = h.dbOf(r.Context()).QueryRow(r.Context(), `
			SELECT COALESCE(SUM(amount), 0) FROM core.transactions WHERE wallet_id = $1 AND type = 'credit'
		`, wid).Scan(&creditsTotal)
		_ = h.dbOf(r.Context()).QueryRow(r.Context(), `
			SELECT COALESCE(SUM(ABS(amount)), 0) FROM core.transactions WHERE wallet_id = $1 AND type = 'debit'
		`, wid).Scan(&debitsTotal)
	}

	existing := map[string]bool{}
	for _, w := range wallets {
		if cur, ok := w["currency"].(string); ok {
			existing[strings.ToUpper(cur)] = true
		}
	}
	available := []string{}
	for _, c := range walletCurrencies {
		if !existing[c] {
			available = append(available, c)
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"wallets":              wallets,
		"selected_wallet":      selected,
		"transactions":         txs,
		"credits_total":        creditsTotal,
		"debits_total":         debitsTotal,
		"available_currencies": available,
		"pagination": map[string]any{
			"page":      page,
			"page_size": pageSize,
			"total":     txTotal,
		},
	})
}

func (h *Handler) listUserWallets(r *http.Request, userID string) []map[string]any {
	rows, _ := h.dbOf(r.Context()).Query(r.Context(), `
		SELECT id::text, currency, balance FROM core.wallets
		WHERE user_id = $1 ORDER BY is_default DESC, currency
	`, userID)
	defer func() {
		if rows != nil {
			rows.Close()
		}
	}()
	wallets := []map[string]any{}
	if rows != nil {
		for rows.Next() {
			var id, cur string
			var bal float64
			if rows.Scan(&id, &cur, &bal) == nil {
				wallets = append(wallets, map[string]any{"id": id, "currency": cur, "balance": bal})
			}
		}
	}
	return wallets
}

func selectWallet(wallets []map[string]any, walletID string) map[string]any {
	if walletID != "" {
		for _, w := range wallets {
			if w["id"] == walletID {
				return w
			}
		}
	}
	if len(wallets) > 0 {
		return wallets[0]
	}
	return nil
}

func (h *Handler) CreateWallet(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var body struct {
		Currency string `json:"currency"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if body.Currency == "" {
		body.Currency = "RUB"
	}
	body.Currency = strings.ToUpper(body.Currency)
	var id string
	err := h.dbOf(r.Context()).QueryRow(r.Context(), `
		INSERT INTO core.wallets (user_id, currency) VALUES ($1, $2)
		ON CONFLICT (user_id, currency) DO UPDATE SET currency = EXCLUDED.currency
		RETURNING id::text
	`, claims.UserID, body.Currency).Scan(&id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"id": id})
}

func (h *Handler) AdminBilling(w http.ResponseWriter, r *http.Request) {
	_, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	limit := 50
	if v, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && v > 0 {
		limit = v
		if limit > 1000 {
			limit = 1000
		}
	}
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	provider := strings.TrimSpace(r.URL.Query().Get("provider"))
	search := strings.TrimSpace(r.URL.Query().Get("q"))
	days := 0
	if v, err := strconv.Atoi(r.URL.Query().Get("days")); err == nil && v > 0 {
		days = v
	}

	rows, _ := h.dbOf(r.Context()).Query(r.Context(), `
		SELECT p.id::text, p.user_id::text, COALESCE(u.email, ''), p.provider,
		       p.amount::float8, p.currency, p.status,
		       COALESCE(p.provider_payment_id, ''), p.created_at::text,
		       COALESCE(p.refunded_amount, 0)::float8
		FROM core.payments p
		LEFT JOIN core.users u ON u.id = p.user_id
		WHERE ($2 = '' OR p.status = $2)
		  AND ($3 = '' OR p.provider = $3)
		  AND ($4 = 0 OR p.created_at >= now() - make_interval(days => $4))
		  AND ($5 = '' OR u.email ILIKE '%' || $5 || '%'
		       OR COALESCE(p.provider_payment_id, '') ILIKE '%' || $5 || '%')
		ORDER BY p.created_at DESC LIMIT $1
	`, limit, status, provider, days, search)
	defer func() {
		if rows != nil {
			rows.Close()
		}
	}()
	list := []map[string]any{}
	if rows != nil {
		for rows.Next() {
			var id, uid, email, prov, currency, st, extID, created string
			var amount, refunded float64
			if rows.Scan(&id, &uid, &email, &prov, &amount, &currency, &st, &extID, &created, &refunded) == nil {
				list = append(list, map[string]any{
					"id": id, "user_id": uid, "user_email": email, "provider": prov,
					"amount": amount, "currency": currency, "status": st,
					"provider_payment_id": extID, "created_at": created,
					"refunded_amount": refunded,
				})
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"payments": list})
}

func (h *Handler) ListPromotions(w http.ResponseWriter, r *http.Request) {
	_, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	rows, err := h.dbOf(r.Context()).Query(r.Context(), `
		SELECT id::text, COALESCE(title, ''), COALESCE(code, ''),
		       COALESCE(discount_type, ''), discount_value::float8, active,
		       starts_at, ends_at, max_uses, used_count, min_amount::float8,
		       only_new_users, COALESCE(description, ''),
		       COALESCE(applies_to, '[]'::jsonb), bonus_percent::float8, bonus_fixed::float8,
		       COALESCE(filters, '{}'::jsonb)
		FROM core.promotions
		ORDER BY created_at DESC
	`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Не удалось получить список акций: "+err.Error())
		return
	}
	defer rows.Close()
	list := []map[string]any{}
	for rows.Next() {
		var id, title, code, typ, description string
		var val, bonusPercent, bonusFixed float64
		var active, onlyNew bool
		var startsAt, endsAt *time.Time
		var maxUses *int
		var minAmount *float64
		var usedCount int
		var appliesTo, filters []byte
		if rows.Scan(&id, &title, &code, &typ, &val, &active, &startsAt, &endsAt,
			&maxUses, &usedCount, &minAmount, &onlyNew, &description,
			&appliesTo, &bonusPercent, &bonusFixed, &filters) != nil {
			continue
		}
		item := map[string]any{
			"id": id, "title": title, "code": code, "type": typ, "value": val,
			"active": active, "starts_at": isoOrNil(startsAt), "ends_at": isoOrNil(endsAt),
			"max_uses": maxUses, "used_count": usedCount, "min_amount": minAmount,
			"only_new_users": onlyNew, "description": description,
			"applies_to":    jsonOrEmptyArray(appliesTo),
			"bonus_percent": bonusPercent,
			"bonus_fixed":   bonusFixed,
		}
		var f map[string][]string
		if json.Unmarshal(filters, &f) == nil {
			for _, key := range []string{"tariff_ids", "game_ids", "location_ids", "user_ids"} {
				if v, ok := f[key]; ok {
					item[key] = v
				} else {
					item[key] = []string{}
				}
			}
		}
		list = append(list, item)
	}
	writeJSON(w, http.StatusOK, map[string]any{"promotions": list})
}

func (h *Handler) CreatePromotion(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	var body promotionBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	discountType, ok := normalizeDiscountType(body.DiscountType)
	if !ok {
		writeError(w, http.StatusBadRequest, "type must be percent or fixed")
		return
	}
	startsAt, ok := parseDate(body.StartsAt)
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid starts_at")
		return
	}
	endsAt, ok := parseDate(body.EndsAt)
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid ends_at")
		return
	}
	appliesTo, _ := normalizeAppliesTo(body.AppliesTo)
	code := normalizePromoCode(body.Code)
	var id string
	err := h.dbOf(r.Context()).QueryRow(r.Context(), `
		INSERT INTO core.promotions ( title, code, discount_type, discount_value, active,
			starts_at, ends_at, max_uses, min_amount, only_new_users, description,
			applies_to, bonus_percent, bonus_fixed, filters
		)
		VALUES ( COALESCE(NULLIF($1, ''), $2, 'Акция'), $2, $3, COALESCE($4, 0), COALESCE($5, true),
		        $6, $7, $8, $9, COALESCE($10, false), $11,
		        COALESCE($12::jsonb, '[]'::jsonb), COALESCE($13, 0), COALESCE($14, 0),
		        COALESCE($15::jsonb, '{}'::jsonb))
		RETURNING id::text
	`, body.Title, code, discountType, body.Value, body.Active,
		startsAt, endsAt, body.MaxUses, body.MinAmount, body.OnlyNewUsers, body.Description,
		appliesTo, body.BonusPercent, body.BonusFixed, promoFiltersPatch(body)).Scan(&id)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	audit(r.Context(), h.dbOf(r.Context()), claims.UserID, "promotion.create", "promotion:"+id, nil)
	writeJSON(w, http.StatusCreated, map[string]string{"id": id})
}

func (h *Handler) defaultCurrency(ctx context.Context) string {
	cur := strings.ToUpper(strings.TrimSpace(h.tenantSettingString(ctx, "billing.default_currency")))
	for _, c := range walletCurrencies {
		if cur == c {
			return cur
		}
	}
	return walletCurrencies[0]
}

func (h *Handler) ensureDefaultWallet(ctx context.Context, userID string) {
	if _, err := h.dbOf(ctx).Exec(ctx, `
		INSERT INTO core.wallets (user_id, currency, is_default)
		VALUES ($1, $2, true)
		ON CONFLICT (user_id, currency) DO NOTHING
	`, userID, h.defaultCurrency(ctx)); err != nil {
		log.Printf("кошелёк по умолчанию для %s не создан: %v", userID, err)
	}
}
