package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/vortanixapp/panel/internal/api/payments"
	"github.com/vortanixapp/panel/pkg/i18n"
	"github.com/vortanixapp/panel/pkg/notify"
)

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func clipText(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

func freekassaMethodsFrom(cfg map[string]any) []map[string]any {
	out := []map[string]any{}
	var methods []map[string]any
	switch v := cfg["methods_json"].(type) {
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
			out = append(out, map[string]any{"id": id, "name": name})
		}
	}
	return out
}

func (h *Handler) TopupForm(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ctx := r.Context()

	wallets := h.listUserWallets(r, claims.UserID)
	selected := selectWallet(wallets, r.URL.Query().Get("wallet_id"))
	rows := h.loadPaymentProviders(ctx)
	settings := h.loadTenantSettingStrings(ctx)

	currencies := map[string]bool{defaultCurrencyFrom(settings): true}
	for _, wallet := range wallets {
		if c, ok := wallet["currency"].(string); ok && c != "" {
			currencies[strings.ToUpper(c)] = true
		}
	}

	providers := []map[string]any{}
	enabled := []string{}
	freekassaMethods := []map[string]any{}
	needRates := false
	for _, def := range payments.AdminCatalog() {
		row := rows[def.Key]
		if !paymentProviderReady(def, row) {
			continue
		}
		chargeCurrency := def.ChargeCurrency(row.Config, "")
		providers = append(providers, map[string]any{
			"id":          def.Key,
			"code":        def.Key,
			"name":        def.Name,
			"fee_percent": parsePercent(settings[paymentFeeKey(def.Key)]),
			"currency":    chargeCurrency,
			"manual":      def.Manual,
		})
		enabled = append(enabled, def.Key)
		if chargeCurrency != "" {
			for c := range currencies {
				if c != chargeCurrency {
					needRates = true
				}
			}
			currencies[chargeCurrency] = true
		}
		if def.Key == "freekassa" {
			freekassaMethods = freekassaMethodsFrom(row.Config)
		}
	}

	fx := map[string]any{"fee_percent": parsePercent(settings[fxFeeKey]), "rates": nil}
	if needRates {
		if rates, err := payments.CurrentRates(ctx); err == nil {
			codes := make([]string, 0, len(currencies))
			for c := range currencies {
				codes = append(codes, c)
			}
			fx["rates"] = rates.Subset(codes)
		} else {
			log.Printf("курсы валют для формы пополнения не получены: %v", err)
		}
	}

	history := []map[string]any{}
	payRows, err := h.dbOf(ctx).Query(ctx, `
		SELECT id::text, status, amount::float8, currency, created_at::text, provider,
		       COALESCE((meta->>'credited_amount')::float8, amount::float8),
		       COALESCE((meta->>'charge_amount')::float8, 0),
		       COALESCE(meta->>'charge_currency', '')
		FROM core.payments WHERE user_id = $1 ORDER BY created_at DESC LIMIT 20
	`, claims.UserID)
	if err == nil {
		defer payRows.Close()
		for payRows.Next() {
			var id, status, currency, created, provider, chargeCurrency string
			var amount, credited, chargeAmount float64
			if payRows.Scan(&id, &status, &amount, &currency, &created, &provider, &credited, &chargeAmount, &chargeCurrency) == nil {
				history = append(history, map[string]any{
					"id": id, "status": status, "amount": amount, "currency": currency,
					"credited_amount": credited, "created_at": created, "provider": provider,
					"provider_name": payments.Name(provider), "charge_amount": chargeAmount,
					"charge_currency": chargeCurrency,
				})
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"wallets":           wallets,
		"selected_wallet":   selected,
		"payments":          history,
		"providers":         providers,
		"enabled_providers": enabled,
		"freekassa_methods": freekassaMethods,
		"fx":                fx,
	})
}

func (h *Handler) CreateTopup(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ctx := r.Context()
	var body struct {
		Amount          float64 `json:"amount"`
		ProviderID      string  `json:"provider_id"`
		Provider        string  `json:"provider"`
		WalletID        string  `json:"wallet_id"`
		PromoCode       string  `json:"promo_code"`
		PaymentMethodID string  `json:"payment_method_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if body.Amount <= 0 {
		writeError(w, http.StatusBadRequest, "Сумма пополнения должна быть больше нуля")
		return
	}

	code := strings.TrimSpace(body.Provider)
	if code == "" {
		code = strings.TrimSpace(body.ProviderID)
		if _, err := uuid.Parse(code); err == nil {
			_ = h.dbOf(ctx).QueryRow(ctx, `SELECT provider FROM core.payment_providers WHERE id = $1`, code).Scan(&code)
		}
	}
	def, known := payments.Definition(code)
	impl, implErr := payments.Get(code)
	if !known || implErr != nil {
		writeError(w, http.StatusBadRequest, "Способ оплаты недоступен")
		return
	}
	row, err := h.loadPaymentProvider(ctx, code)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	if !paymentProviderReady(def, row) {
		writeError(w, http.StatusBadRequest, "Способ оплаты «"+def.Name+"» сейчас недоступен")
		return
	}

	settings := h.loadTenantSettingStrings(ctx)
	currency := defaultCurrencyFrom(settings)
	var walletID *string
	if body.WalletID != "" {
		var cur string
		if err := h.dbOf(ctx).QueryRow(ctx, `
			SELECT currency FROM core.wallets WHERE id = $1 AND user_id = $2
		`, body.WalletID, claims.UserID).Scan(&cur); err != nil {
			writeError(w, http.StatusBadRequest, "Кошелёк не найден")
			return
		}
		walletID = &body.WalletID
		currency = strings.ToUpper(cur)
	}

	creditedAmount := body.Amount
	promoID := ""
	if body.PromoCode != "" {
		promo, err := payments.ApplyPromo(ctx, h.dbOf(ctx), claims.UserID, body.PromoCode, body.Amount)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		creditedAmount = promo.CreditedAmount
		promoID = promo.PromoID
	}

	chargeCurrency := def.ChargeCurrency(row.Config, currency)
	var rates *payments.Rates
	if chargeCurrency != currency {
		rates, err = payments.CurrentRates(ctx)
		if err != nil {
			log.Printf("курс %s → %s для пополнения не получен: %v", currency, chargeCurrency, err)
			writeError(w, http.StatusServiceUnavailable, "Не удалось получить курс валют для оплаты в "+chargeCurrency+", попробуйте позже")
			return
		}
	}
	quote, err := payments.NewQuote(body.Amount, currency, chargeCurrency,
		parsePercent(settings[paymentFeeKey(code)]), parsePercent(settings[fxFeeKey]), rates)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	meta := map[string]any{
		"promo_code":        body.PromoCode,
		"payment_method_id": body.PaymentMethodID,
		"credited_amount":   creditedAmount,
		"fee_percent":       quote.FeePercent,
		"fx_fee_percent":    quote.FXFeePercent,
		"fx_rate":           quote.Rate,
		"charge_amount":     quote.ChargeAmount,
		"charge_currency":   quote.ChargeCurrency,
	}
	if promoID != "" {
		meta["promo_id"] = promoID
	}
	metaJSON, _ := json.Marshal(meta)

	var paymentID string
	var invoiceNo int64
	if err := h.dbOf(ctx).QueryRow(ctx, `
		INSERT INTO core.payments (user_id, wallet_id, provider, amount, currency, status, meta, promotion_id)
		VALUES ($1, $2, $3, $4, $5, 'pending', $6::jsonb, NULLIF($7, '')::uuid)
		RETURNING id::text, invoice_no
	`, claims.UserID, walletID, code, body.Amount, currency, metaJSON, promoID).Scan(&paymentID, &invoiceNo); err != nil {
		log.Printf("платёж не создан: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create payment")
		return
	}

	var email string
	_ = h.dbOf(ctx).QueryRow(ctx, `SELECT email FROM core.users WHERE id = $1`, claims.UserID).Scan(&email)
	appName := firstNonEmpty(settings["app.name"], settings["panel.name"], "Vortanix")
	receipt := accountingProfileFrom(settings).receipt(email)

	checkoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	checkout, err := impl.CreateCheckout(checkoutCtx, row.Config, payments.CheckoutInput{
		PaymentID:   paymentID,
		InvoiceNo:   invoiceNo,
		Amount:      quote.ChargeAmount,
		Currency:    quote.ChargeCurrency,
		Description: fmt.Sprintf("%s: top-up #%d", appName, invoiceNo),
		ReturnURL:   h.paymentReturnURL(paymentID),
		FailURL:     h.paymentReturnURL(paymentID),
		NotifyURL:   h.paymentNotifyURL(code),
		MethodID:    body.PaymentMethodID,
		Email:       email,
		UserID:      claims.UserID,
		Receipt:     receipt,
	})
	if err != nil {
		log.Printf("платёж %s через %s не создан в кассе: %v", paymentID, code, err)
		failMeta, _ := json.Marshal(map[string]any{"error": clipText(err.Error(), 500)})
		_, _ = h.dbOf(ctx).Exec(ctx, `
			UPDATE core.payments SET status = 'failed', meta = meta || $2::jsonb, updated_at = now() WHERE id = $1
		`, paymentID, failMeta)
		writeError(w, http.StatusBadGateway, "Платёжная система не приняла запрос: "+clipText(err.Error(), 300))
		return
	}

	redirectURL := h.paymentReturnURL(paymentID)
	extra := map[string]any{}
	switch {
	case checkout.Form != nil:
		extra["checkout_form"] = checkout.Form
	case checkout.RedirectURL != "":
		redirectURL = checkout.RedirectURL
		extra["checkout_url"] = redirectURL
	}
	if payments.ReceiptRequested(row.Config) {
		extra["receipt"] = receipt
	}
	status := "processing"
	if checkout.Manual {
		status = "pending"
	}
	extraJSON, _ := json.Marshal(extra)
	_, _ = h.dbOf(ctx).Exec(ctx, `
		UPDATE core.payments
		SET provider_payment_id = COALESCE(NULLIF($2, ''), provider_payment_id),
		    status = $3, meta = meta || $4::jsonb, updated_at = now()
		WHERE id = $1
	`, paymentID, checkout.ProviderPaymentID, status, extraJSON)

	writeJSON(w, http.StatusCreated, map[string]any{
		"payment_id":      paymentID,
		"status":          status,
		"redirect_url":    redirectURL,
		"checkout_form":   checkout.Form,
		"credited_amount": creditedAmount,
		"charge_amount":   quote.ChargeAmount,
		"charge_currency": quote.ChargeCurrency,
	})
}

func (h *Handler) GetPayment(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ctx := r.Context()
	id := chi.URLParam(r, "id")
	if _, err := uuid.Parse(id); err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	var amount float64
	var status, currency, provider, created string
	var invoiceNo int64
	var metaRaw []byte
	err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT amount::float8, status, currency, provider, invoice_no, created_at::text, meta
		FROM core.payments WHERE id = $1 AND user_id = $2
	`, id, claims.UserID).Scan(&amount, &status, &currency, &provider, &invoiceNo, &created, &metaRaw)
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	meta := map[string]any{}
	_ = json.Unmarshal(metaRaw, &meta)

	payment := map[string]any{
		"id":              id,
		"amount":          amount,
		"status":          status,
		"currency":        currency,
		"provider":        provider,
		"provider_name":   payments.Name(provider),
		"invoice_no":      invoiceNo,
		"created_at":      created,
		"charge_amount":   meta["charge_amount"],
		"charge_currency": meta["charge_currency"],
		"fee_percent":     meta["fee_percent"],
		"credited_amount": meta["credited_amount"],
	}
	if status == "pending" || status == "processing" {
		if form, ok := meta["checkout_form"].(map[string]any); ok && form["action"] != nil {
			payment["checkout_form"] = form
		} else if u, ok := meta["checkout_url"].(string); ok && u != "" && !strings.HasSuffix(u, "/payment/"+id) && !strings.Contains(u, "/v1/pay/") {
			payment["checkout_url"] = u
		}
		if def, ok := payments.Definition(provider); ok && def.Manual {
			if row, err := h.loadPaymentProvider(ctx, provider); err == nil {
				instructions := payments.BankInstructions(row.Config)
				instructions["reference"] = fmt.Sprintf("#%d", invoiceNo)
				payment["instructions"] = instructions
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"payment": payment})
}

func (h *Handler) PaymentCheckoutPage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if _, err := uuid.Parse(id); err != nil {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, h.paymentReturnURL(id), http.StatusSeeOther)
}

func (h *Handler) AdminCompletePayment(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ctx := r.Context()
	id := chi.URLParam(r, "id")
	var provider, status string
	if err := h.dbOf(ctx).QueryRow(ctx, `SELECT provider, status FROM core.payments WHERE id = $1`, id).Scan(&provider, &status); err != nil {
		writeError(w, http.StatusNotFound, "payment not found")
		return
	}
	if status != "pending" && status != "processing" {
		writeError(w, http.StatusConflict, "Платёж уже обработан (статус "+status+")")
		return
	}
	if def, ok := payments.Definition(provider); !ok || !def.Manual {
		writeError(w, http.StatusConflict, "Вручную подтверждаются только банковские переводы — остальные платежи зачисляются по уведомлению кассы")
		return
	}
	if err := h.completeTopupPayment(ctx, id, "", payments.Name(provider)); err != nil {
		log.Printf("платёж %s не подтверждён: %v", id, err)
		writeError(w, http.StatusInternalServerError, "failed to complete payment")
		return
	}
	audit(ctx, h.dbOf(ctx), claims.UserID, "payment.confirm", "payment:"+id, map[string]any{"provider": provider})
	h.auditAlert(ctx, claims.UserID, claims.Email, "payment.confirm", "платёж "+id)
	writeJSON(w, http.StatusOK, map[string]string{"status": "completed"})
}

func (h *Handler) AdminCancelPayment(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ctx := r.Context()
	id := chi.URLParam(r, "id")
	var userID, provider string
	err := h.dbOf(ctx).QueryRow(ctx, `
		UPDATE core.payments SET status = 'cancelled', updated_at = now()
		WHERE id = $1 AND status IN ('pending', 'processing')
		RETURNING user_id::text, provider
	`, id).Scan(&userID, &provider)
	if err != nil {
		writeError(w, http.StatusConflict, "Отменить можно только платёж, который ещё не оплачен")
		return
	}
	audit(ctx, h.dbOf(ctx), claims.UserID, "payment.cancel", "payment:"+id, map[string]any{"provider": provider})
	h.notifyUser(ctx, userID, notify.Event{
		Kind:      notify.KindPaymentReceived,
		Title:     i18n.Key("notify.payment_cancelled.title"),
		Body:      i18n.Key("notify.payment_cancelled.body", i18n.Params{"provider": payments.Name(provider)}),
		Action:    h.panelAction("notify.action.balance", "/billing"),
		Meta:      map[string]any{"payment_id": id},
		DedupeKey: "payment.cancelled:" + id,
	})
	writeJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
}
