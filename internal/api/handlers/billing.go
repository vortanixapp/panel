package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/vortanixapp/panel/internal/api/payments"
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
		INSERT INTO core.wallets (user_id, tenant_id, currency) VALUES ($1, $2, $3)
		ON CONFLICT (user_id, currency) DO UPDATE SET currency = EXCLUDED.currency
		RETURNING id::text
	`, claims.UserID, claims.TenantID, body.Currency).Scan(&id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"id": id})
}

func (h *Handler) TopupForm(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	wallets := h.listUserWallets(r, claims.UserID)
	selected := selectWallet(wallets, r.URL.Query().Get("wallet_id"))

	rows, _ := h.dbOf(r.Context()).Query(r.Context(), `
		SELECT id::text, provider, enabled FROM core.payment_providers WHERE tenant_id = $1 AND enabled = true
	`, claims.TenantID)
	providers := []map[string]any{}
	enabled := []string{}
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var id, provider string
			var enabledFlag bool
			if rows.Scan(&id, &provider, &enabledFlag) == nil && enabledFlag {
				providers = append(providers, map[string]any{"id": id, "code": provider, "name": provider})
				enabled = append(enabled, provider)
			}
		}
	}

	payments := []map[string]any{}
	payRows, _ := h.dbOf(r.Context()).Query(r.Context(), `
		SELECT id::text, status, amount, currency, created_at::text
		FROM core.payments WHERE user_id = $1 ORDER BY created_at DESC LIMIT 20
	`, claims.UserID)
	if payRows != nil {
		defer payRows.Close()
		for payRows.Next() {
			var id, status, cur, created string
			var amount float64
			if payRows.Scan(&id, &status, &amount, &cur, &created) == nil {
				payments = append(payments, map[string]any{
					"id": id, "status": status, "amount": amount, "currency": cur,
					"credited_amount": nil, "created_at": created,
				})
			}
		}
	}

	freekassaMethods := []map[string]any{}
	for _, p := range providers {
		if p["code"] == "freekassa" {
			var config []byte
			_ = h.dbOf(r.Context()).QueryRow(r.Context(), `
				SELECT config FROM core.payment_providers WHERE id = $1
			`, p["id"]).Scan(&config)
			if plain, decErr := h.secrets.DecryptJSON(config); decErr == nil && len(plain) > 0 {
				var cfg map[string]any
				if json.Unmarshal(plain, &cfg) == nil {
					if raw, ok := cfg["methods_json"]; ok {
						var methods []map[string]any
						switch v := raw.(type) {
						case string:
							_ = json.Unmarshal([]byte(v), &methods)
						case []any:
							for _, item := range v {
								if m, ok := item.(map[string]any); ok {
									methods = append(methods, m)
								}
							}
						}
						for _, m := range methods {
							if id, ok := m["id"]; ok {
								name, _ := m["name"].(string)
								freekassaMethods = append(freekassaMethods, map[string]any{"id": id, "name": name})
							}
						}
					}
				}
			}
			break
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"wallets":           wallets,
		"selected_wallet":   selected,
		"payments":          payments,
		"providers":         providers,
		"enabled_providers": enabled,
		"freekassa_methods": freekassaMethods,
	})
}

func (h *Handler) CreateTopup(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var body struct {
		Amount          float64 `json:"amount"`
		ProviderID      string  `json:"provider_id"`
		Provider        string  `json:"provider"`
		WalletID        string  `json:"wallet_id"`
		PromoCode       string  `json:"promo_code"`
		PaymentMethodID string  `json:"payment_method_id"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if body.Amount <= 0 {
		writeError(w, http.StatusBadRequest, "amount must be positive")
		return
	}
	provider := body.Provider
	if provider == "" {
		provider = "manual"
	}
	if body.ProviderID != "" {
		_ = h.dbOf(r.Context()).QueryRow(r.Context(), `
			SELECT provider FROM core.payment_providers WHERE id = $1 AND tenant_id = $2
		`, body.ProviderID, claims.TenantID).Scan(&provider)
	}
	currency := "RUB"
	var walletID *string
	if body.WalletID != "" {
		walletID = &body.WalletID
		var cur string
		if h.dbOf(r.Context()).QueryRow(r.Context(), `
			SELECT currency FROM core.wallets WHERE id = $1 AND user_id = $2
		`, body.WalletID, claims.UserID).Scan(&cur) == nil {
			currency = cur
		}
	}

	creditedAmount := body.Amount
	var promoID string
	if body.PromoCode != "" {
		promo, err := payments.ApplyPromo(r.Context(), h.dbOf(r.Context()), claims.TenantID, claims.UserID, body.PromoCode, body.Amount)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		creditedAmount = promo.CreditedAmount
		promoID = promo.PromoID
	}

	metaMap := map[string]any{
		"provider_id":       body.ProviderID,
		"promo_code":        body.PromoCode,
		"payment_method_id": body.PaymentMethodID,
		"credited_amount":   creditedAmount,
	}
	if promoID != "" {
		metaMap["promo_id"] = promoID
	}
	meta, _ := json.Marshal(metaMap)
	var id string
	err := h.dbOf(r.Context()).QueryRow(r.Context(), `
		INSERT INTO core.payments (tenant_id, user_id, wallet_id, provider, amount, currency, status, meta, promotion_id)
		VALUES ($1, $2, $3, $4, $5, $6, 'pending', $7::jsonb, NULLIF($8, '')::uuid)
		RETURNING id::text
	`, claims.TenantID, claims.UserID, walletID, provider, body.Amount, currency, meta, promoID).Scan(&id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create payment")
		return
	}

	panelBase := envOr("PANEL_PUBLIC_URL", h.frontendURL)
	returnURL, failURL := paymentsPanelURLs(panelBase, id)
	redirectURL := returnURL

	if payments.IsSupported(provider) {
		cfg, cfgErr := payments.LoadProviderConfig(r.Context(), h.dbOf(r.Context()), h.secrets, claims.TenantID, provider)
		if cfgErr == nil {
			var userEmail string
			_ = h.dbOf(r.Context()).QueryRow(r.Context(), `SELECT email FROM core.users WHERE id = $1`, claims.UserID).Scan(&userEmail)
			pspURL, pspID, checkoutErr := payments.CreateCheckout(r.Context(), provider, cfg, payments.CheckoutInput{
				TenantID:    claims.TenantID,
				PaymentID:   id,
				Amount:      body.Amount,
				Currency:    currency,
				Description: "Vortanix balance top-up",
				ReturnURL:   returnURL,
				FailURL:     failURL,
				MethodID:    body.PaymentMethodID,
				Email:       userEmail,
			})
			if checkoutErr == nil && pspURL != "" {
				redirectURL = pspURL
				if pspID != "" {
					_, _ = h.dbOf(r.Context()).Exec(r.Context(), `
						UPDATE core.payments SET provider_payment_id = $2, status = 'processing' WHERE id = $1
					`, id, pspID)
				}
			}
		}
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"payment_id":      id,
		"status":          "pending",
		"redirect_url":    redirectURL,
		"credited_amount": creditedAmount,
	})
}

func paymentsPanelURLs(panelBase, paymentID string) (success, fail string) {
	path := "/billing/topup/payment/" + paymentID
	panelBase = strings.TrimSuffix(panelBase, "/")
	if panelBase == "" {
		return path, path
	}
	return panelBase + path, panelBase + path
}

func (h *Handler) GetPayment(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := chi.URLParam(r, "id")
	var amount float64
	var status, cur string
	err := h.dbOf(r.Context()).QueryRow(r.Context(), `
		SELECT amount, status, currency FROM core.payments WHERE id = $1 AND user_id = $2
	`, id, claims.UserID).Scan(&amount, &status, &cur)
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"payment": map[string]any{"id": id, "amount": amount, "status": status, "currency": cur},
	})
}

func (h *Handler) AdminBilling(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
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
		WHERE p.tenant_id = $1
		  AND ($3 = '' OR p.status = $3)
		  AND ($4 = '' OR p.provider = $4)
		  AND ($5 = 0 OR p.created_at >= now() - make_interval(days => $5))
		  AND ($6 = '' OR u.email ILIKE '%' || $6 || '%'
		       OR COALESCE(p.provider_payment_id, '') ILIKE '%' || $6 || '%')
		ORDER BY p.created_at DESC LIMIT $2
	`, claims.TenantID, limit, status, provider, days, search)
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

func (h *Handler) ListPaymentProviders(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	rows, _ := h.dbOf(r.Context()).Query(r.Context(), `SELECT id::text, provider, enabled FROM core.payment_providers WHERE tenant_id = $1`, claims.TenantID)
	defer func() {
		if rows != nil {
			rows.Close()
		}
	}()
	list := []map[string]any{}
	if rows != nil {
		for rows.Next() {
			var id, provider string
			var enabled bool
			if rows.Scan(&id, &provider, &enabled) == nil {
				list = append(list, map[string]any{"id": id, "code": provider, "name": provider, "enabled": enabled})
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"providers": list})
}

func (h *Handler) UpdatePaymentProvider(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	var body map[string]any
	_ = json.NewDecoder(r.Body).Decode(&body)
	var configJSON []byte
	if raw, ok := body["config"]; ok && raw != nil {
		marshalled, err := json.Marshal(raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid config")
			return
		}
		configJSON, err = h.secrets.EncryptJSON(marshalled)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to encrypt provider config")
			return
		}
	}
	_, _ = h.dbOf(r.Context()).Exec(r.Context(), `UPDATE core.payment_providers SET enabled = COALESCE($3, enabled), config = COALESCE($4::jsonb, config) WHERE id = $1 AND tenant_id = $2`,
		chi.URLParam(r, "id"), claims.TenantID, body["enabled"], configJSON)
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (h *Handler) ListPromotions(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	// discount_type отдаём как есть, без подстановки percent: пустой тип
	// означает акцию без скидки — только с бонусом на пополнение.
	rows, err := h.dbOf(r.Context()).Query(r.Context(), `
		SELECT id::text, COALESCE(title, ''), COALESCE(code, ''),
		       COALESCE(discount_type, ''), discount_value::float8, active,
		       starts_at, ends_at, max_uses, used_count, min_amount::float8,
		       only_new_users, COALESCE(description, ''),
		       COALESCE(applies_to, '[]'::jsonb), bonus_percent::float8, bonus_fixed::float8,
		       COALESCE(filters, '{}'::jsonb)
		FROM core.promotions WHERE tenant_id = $1
		ORDER BY created_at DESC
	`, claims.TenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "не удалось получить список акций: "+err.Error())
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
	// Название по умолчанию берём из кода, а если акция безкодовая — из
	// описания: пустой заголовок в списке выглядел бы как пустая строка.
	err := h.dbOf(r.Context()).QueryRow(r.Context(), `
		INSERT INTO core.promotions (
			tenant_id, title, code, discount_type, discount_value, active,
			starts_at, ends_at, max_uses, min_amount, only_new_users, description,
			applies_to, bonus_percent, bonus_fixed, filters
		)
		VALUES ($1, COALESCE(NULLIF($2, ''), $3, 'Акция'), $3, $4, COALESCE($5, 0), COALESCE($6, true),
		        $7, $8, $9, $10, COALESCE($11, false), $12,
		        COALESCE($13::jsonb, '[]'::jsonb), COALESCE($14, 0), COALESCE($15, 0),
		        COALESCE($16::jsonb, '{}'::jsonb))
		RETURNING id::text
	`, claims.TenantID, body.Title, code, discountType, body.Value, body.Active,
		startsAt, endsAt, body.MaxUses, body.MinAmount, body.OnlyNewUsers, body.Description,
		appliesTo, body.BonusPercent, body.BonusFixed, promoFiltersPatch(body)).Scan(&id)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	audit(r.Context(), h.dbOf(r.Context()), claims.TenantID, claims.UserID, "promotion.create", "promotion:"+id, nil)
	writeJSON(w, http.StatusCreated, map[string]string{"id": id})
}
