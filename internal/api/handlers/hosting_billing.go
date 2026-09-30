package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/vortanixapp/panel/internal/api/hosting"
	"github.com/vortanixapp/panel/internal/api/payments"
)

type hostingPlanPricing struct {
	ID             string
	Name           string
	Monthly        float64
	Periods        []int
	RenewalPeriods []int
	Discounts      map[string]float64
}

type hostingQuote struct {
	Period         int                       `json:"period_days"`
	Currency       string                    `json:"currency"`
	BaseCost       float64                   `json:"base_cost"`
	PeriodPercent  float64                   `json:"period_discount_percent"`
	PeriodDiscount float64                   `json:"period_discount"`
	Subtotal       float64                   `json:"subtotal"`
	FinalCost      float64                   `json:"final_cost"`
	PromoPreview   payments.RentPromoPreview `json:"promo_preview"`
}

func (h *Handler) loadHostingPlanPricing(ctx context.Context, planID string) (hostingPlanPricing, error) {
	var plan hostingPlanPricing
	var periods, renewal, discounts []byte
	err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT id::text, name, price_monthly::float8, rental_periods, renewal_periods, discounts
		FROM core.hosting_plans
		WHERE id = NULLIF($1, '')::uuid AND active = true
	`, planID).Scan(&plan.ID, &plan.Name, &plan.Monthly, &periods, &renewal, &discounts)
	if err != nil {
		return plan, err
	}
	_ = json.Unmarshal(periods, &plan.Periods)
	_ = json.Unmarshal(renewal, &plan.RenewalPeriods)
	plan.Discounts = map[string]float64{}
	_ = json.Unmarshal(discounts, &plan.Discounts)
	return plan, nil
}

func hostingPeriodAllowed(allowed []int, period int) bool {
	if period < 1 || period > 365 {
		return false
	}
	if len(allowed) == 0 {
		return true
	}
	for _, p := range allowed {
		if p == period {
			return true
		}
	}
	return false
}

func (h *Handler) hostingQuote(
	ctx context.Context, userID string, plan hostingPlanPricing, period int, promoCode string,
) (hostingQuote, string) {
	base := math.Round(plan.Monthly*float64(period)/30*100) / 100
	percent := math.Min(math.Max(plan.Discounts[strconv.Itoa(period)], 0), 100)
	periodDiscount := math.Round(base*percent) / 100
	subtotal := math.Max(0, math.Round((base-periodDiscount)*100)/100)

	promo, promoErr := payments.PickPromotion(
		ctx, h.dbOf(ctx), payments.ApplyHosting, userID, promoCode, "", "", "", subtotal,
	)
	preview := payments.ApplyRentDiscount(promo, subtotal)
	final := preview.FinalCost
	if promoErr != "" {
		preview.Valid = false
		message := promoErr
		preview.Error = &message
		final = subtotal
	}
	return hostingQuote{
		Period:         period,
		Currency:       h.defaultCurrency(ctx),
		BaseCost:       base,
		PeriodPercent:  percent,
		PeriodDiscount: periodDiscount,
		Subtotal:       subtotal,
		FinalCost:      final,
		PromoPreview:   preview,
	}, promoErr
}

func (h *Handler) HostingRentQuote(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	q := r.URL.Query()
	plan, err := h.loadHostingPlanPricing(r.Context(), q.Get("plan_id"))
	if err != nil {
		writeError(w, http.StatusNotFound, "Тариф не найден")
		return
	}
	period, _ := strconv.Atoi(q.Get("period"))
	if period == 0 {
		period = 30
	}
	if !hostingPeriodAllowed(plan.Periods, period) {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("Срок %d дн. недоступен для этого тарифа", period))
		return
	}
	quote, _ := h.hostingQuote(r.Context(), claims.UserID, plan, period, q.Get("promo_code"))
	writeJSON(w, http.StatusOK, quote)
}

func (h *Handler) hostingAccountPlan(ctx context.Context, accountID, userID string) (hostingPlanPricing, string, error) {
	var planID, username string
	err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT hosting_plan_id::text, username
		FROM core.hosting_accounts WHERE id = $1 AND user_id = $2
	`, accountID, userID).Scan(&planID, &username)
	if err != nil {
		return hostingPlanPricing{}, "", err
	}
	plan, err := h.loadHostingPlanPricing(ctx, planID)
	return plan, username, err
}

func renewalPeriods(plan hostingPlanPricing) []int {
	if len(plan.RenewalPeriods) > 0 {
		return plan.RenewalPeriods
	}
	return plan.Periods
}

func (h *Handler) HostingRenewPreview(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	plan, _, err := h.hostingAccountPlan(r.Context(), chi.URLParam(r, "id"), claims.UserID)
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	var body struct {
		Period    int    `json:"period"`
		PromoCode string `json:"promo_code"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if body.Period <= 0 {
		body.Period = 30
	}
	if !hostingPeriodAllowed(renewalPeriods(plan), body.Period) {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("Срок продления %d дн. недоступен", body.Period))
		return
	}
	quote, _ := h.hostingQuote(r.Context(), claims.UserID, plan, body.Period, body.PromoCode)
	writeJSON(w, http.StatusOK, quote)
}

func (h *Handler) walletCurrencyFor(ctx context.Context, userID, walletID string) (string, bool) {
	if walletID == "" {
		return h.defaultCurrency(ctx), true
	}
	var currency string
	err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT currency FROM core.wallets WHERE id = $1::uuid AND user_id = $2
	`, walletID, userID).Scan(&currency)
	return strings.ToUpper(currency), err == nil
}

func (h *Handler) refundTransaction(ctx context.Context, txID string, amount float64, desc string) {
	if txID == "" || amount <= 0 {
		return
	}
	db := h.dbOf(ctx)
	tx, err := db.Begin(ctx)
	if err != nil {
		return
	}
	defer tx.Rollback(ctx)
	var walletID string
	if tx.QueryRow(ctx, `SELECT wallet_id::text FROM core.transactions WHERE id = $1::uuid`, txID).Scan(&walletID) != nil {
		return
	}
	if _, err := tx.Exec(ctx, `
		UPDATE core.wallets SET balance = balance + $2, updated_at = now() WHERE id = $1::uuid
	`, walletID, amount); err != nil {
		return
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO core.transactions ( wallet_id, type, amount, description, source_type)
		VALUES ($1::uuid, 'credit', $2, $3, 'refund')
	`, walletID, amount, desc); err != nil {
		return
	}
	_ = tx.Commit(ctx)
}

func (h *Handler) HostingRentSubmit(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if h.refuseUnidentified(r.Context(), w, claims.UserID, claims.Role) {
		return
	}
	var body struct {
		PlanID    string `json:"plan_id"`
		Username  string `json:"username"`
		Domain    string `json:"domain"`
		Period    int    `json:"period"`
		WalletID  string `json:"wallet_id"`
		PromoCode string `json:"promo_code"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	username := body.Username
	if username == "" {
		username = body.Domain
	}
	periodDays := 30
	if body.Period > 0 {
		periodDays = body.Period
	}

	ctx := r.Context()
	plan, err := h.loadHostingPlanPricing(ctx, body.PlanID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid plan or failed to create account")
		return
	}
	if !hostingPeriodAllowed(plan.Periods, periodDays) {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("Срок %d дн. недоступен для этого тарифа", periodDays))
		return
	}
	currency, walletOK := h.walletCurrencyFor(ctx, claims.UserID, body.WalletID)
	if !walletOK {
		writeError(w, http.StatusBadRequest, "Кошелёк не найден")
		return
	}
	quote, promoErr := h.hostingQuote(ctx, claims.UserID, plan, periodDays, body.PromoCode)
	if promoErr != "" {
		writeError(w, http.StatusBadRequest, promoErr)
		return
	}

	var id string
	err = h.dbOf(ctx).QueryRow(ctx, `
		INSERT INTO core.hosting_accounts (
			user_id, hosting_server_id, hosting_plan_id, username, primary_domain, expires_at
		)
		SELECT $1, p.hosting_server_id, p.id, $3, NULLIF($4, ''), now() + make_interval(days => $5)
		FROM core.hosting_plans p
		WHERE p.id = NULLIF($2, '')::uuid
		RETURNING id::text
	`, claims.UserID, body.PlanID, username, body.Domain, periodDays).Scan(&id)
	if err != nil || id == "" {
		writeError(w, http.StatusBadRequest, "invalid plan or failed to create account")
		return
	}
	dropAccount := func() {
		_, _ = h.dbOf(ctx).Exec(ctx, `DELETE FROM core.hosting_accounts WHERE id = $1`, id)
	}

	promoID := quote.PromoPreview.PromoID
	promoClaimed := false
	if promoID != "" {
		claimed, _ := payments.ClaimPromoUsage(ctx, h.dbOf(ctx), promoID, claims.UserID)
		if !claimed {
			dropAccount()
			writeError(w, http.StatusConflict, "Промокод уже использован")
			return
		}
		promoClaimed = true
	}
	releasePromo := func() {
		if promoClaimed {
			_ = payments.ReleasePromoUsage(ctx, h.dbOf(ctx), promoID, claims.UserID)
		}
	}

	txID := ""
	if quote.FinalCost > 0 {
		txID, err = h.debitWalletForRentSource(r, claims, body.WalletID, currency,
			quote.FinalCost, "Hosting rent: "+plan.Name, "hosting_rent", id)
		if err != nil {
			dropAccount()
			releasePromo()
			writeError(w, http.StatusPaymentRequired, err.Error())
			return
		}
	}

	var hostingServerID, planPackage string
	_ = h.dbOf(ctx).QueryRow(ctx, `
		SELECT hs.id::text, hp.panel_package_name
		FROM core.hosting_accounts ha
		JOIN core.hosting_servers hs ON hs.id = ha.hosting_server_id
		JOIN core.hosting_plans hp ON hp.id = ha.hosting_plan_id
		WHERE ha.id = $1
	`, id).Scan(&hostingServerID, &planPackage)
	h.provisionHostingAccount(r, id, username, body.Domain, hostingServerID, planPackage)

	var status string
	_ = h.dbOf(ctx).QueryRow(ctx, `SELECT status FROM core.hosting_accounts WHERE id = $1`, id).Scan(&status)
	if status == "error" {
		h.refundTransaction(ctx, txID, quote.FinalCost, "Возврат: не удалось создать хостинг "+plan.Name)
		releasePromo()
		writeError(w, http.StatusBadGateway, "Не удалось создать аккаунт на хостинге, деньги возвращены на баланс")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"id": id})
}

func (h *Handler) RenewHostingAccount(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ctx := r.Context()
	accountID := chi.URLParam(r, "id")
	panelID, panelCfg, err := h.hostingPanelRef(r, accountID, claims.UserID)
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	plan, _, err := h.hostingAccountPlan(ctx, accountID, claims.UserID)
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	var body struct {
		Period    int    `json:"period"`
		WalletID  string `json:"wallet_id"`
		PromoCode string `json:"promo_code"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if body.Period <= 0 {
		body.Period = 30
	}
	if !hostingPeriodAllowed(renewalPeriods(plan), body.Period) {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("Срок продления %d дн. недоступен", body.Period))
		return
	}
	currency, walletOK := h.walletCurrencyFor(ctx, claims.UserID, body.WalletID)
	if !walletOK {
		writeError(w, http.StatusBadRequest, "Кошелёк не найден")
		return
	}
	quote, promoErr := h.hostingQuote(ctx, claims.UserID, plan, body.Period, body.PromoCode)
	if promoErr != "" {
		writeError(w, http.StatusBadRequest, promoErr)
		return
	}

	promoID := quote.PromoPreview.PromoID
	promoClaimed := false
	if promoID != "" {
		claimed, _ := payments.ClaimPromoUsage(ctx, h.dbOf(ctx), promoID, claims.UserID)
		if !claimed {
			writeError(w, http.StatusConflict, "Промокод уже использован")
			return
		}
		promoClaimed = true
	}
	txID := ""
	if quote.FinalCost > 0 {
		txID, err = h.debitWalletForRentSource(r, claims, body.WalletID, currency,
			quote.FinalCost, "Hosting renew: "+plan.Name, "hosting_renew", accountID)
		if err != nil {
			if promoClaimed {
				_ = payments.ReleasePromoUsage(ctx, h.dbOf(ctx), promoID, claims.UserID)
			}
			writeError(w, http.StatusPaymentRequired, err.Error())
			return
		}
	}
	rollback := func() {
		h.refundTransaction(ctx, txID, quote.FinalCost, "Возврат: не удалось продлить хостинг "+plan.Name)
		if promoClaimed {
			_ = payments.ReleasePromoUsage(ctx, h.dbOf(ctx), promoID, claims.UserID)
		}
	}

	adapter := hosting.NewAdapter(panelCfg)
	if err := adapter.Renew(ctx, panelID, body.Period); err != nil {
		rollback()
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	var wasSuspended bool
	_ = h.dbOf(ctx).QueryRow(ctx, `
		SELECT suspended_at IS NOT NULL FROM core.hosting_accounts WHERE id = $1
	`, accountID).Scan(&wasSuspended)
	if wasSuspended {
		if err := adapter.Unsuspend(ctx, panelID); err != nil {
			rollback()
			writeError(w, http.StatusBadGateway, "Не удалось снять приостановку на панели: "+err.Error())
			return
		}
	}
	_, _ = h.dbOf(ctx).Exec(ctx, `
		UPDATE core.hosting_accounts
		SET expires_at = GREATEST(COALESCE(expires_at, now()), now()) + make_interval(days => $2),
		    status = 'active',
		    suspended_at = NULL
		WHERE id = $1
	`, accountID, body.Period)
	writeJSON(w, http.StatusOK, map[string]string{"status": "renewed"})
}
