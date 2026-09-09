package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/vortanixapp/panel/internal/api/payments"
	"github.com/vortanixapp/panel/pkg/notify"
)

type refundBody struct {
	Amount float64 `json:"amount"`
	Reason string  `json:"reason"`
}

func (h *Handler) AdminRefundPayment(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	paymentID := chi.URLParam(r, "id")
	var body refundBody
	_ = json.NewDecoder(r.Body).Decode(&body)
	ctx := r.Context()

	var provider, currency, status, userID string
	var providerPaymentID *string
	var amount, refunded float64
	err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT provider, currency, status, user_id::text, provider_payment_id,
		       amount, COALESCE(refunded_amount, 0)
		FROM core.payments WHERE id = $1 AND tenant_id = $2
	`, paymentID, claims.TenantID).Scan(
		&provider, &currency, &status, &userID, &providerPaymentID, &amount, &refunded)
	if err != nil {
		writeError(w, http.StatusNotFound, "payment not found")
		return
	}
	if !isPaidStatus(status) {
		writeError(w, http.StatusConflict, "возврат возможен только по оплаченному платежу (статус "+status+")")
		return
	}

	if !payments.RefundSupported(provider) {
		writeError(w, http.StatusConflict,
			"у шлюза "+provider+" нет API возврата — верните платёж в его личном кабинете")
		return
	}

	available := amount - refunded
	if available <= 0 {
		writeError(w, http.StatusConflict, "платёж уже возвращён полностью")
		return
	}
	refundAmount := body.Amount
	if refundAmount <= 0 {
		refundAmount = available
	}
	if refundAmount > available+0.009 {
		writeError(w, http.StatusBadRequest, "сумма больше остатка по платежу")
		return
	}

	walletID, balance, ok := h.walletForUser(ctx, claims.TenantID, userID, currency)
	if !ok {
		writeError(w, http.StatusConflict, "у клиента нет кошелька в валюте платежа")
		return
	}
	if balance+0.009 < refundAmount {
		writeError(w, http.StatusConflict,
			"на балансе клиента меньше суммы возврата — он уже потратил эти деньги; вернуть можно "+formatMoney(balance)+" "+currency)
		return
	}

	cfg, cfgErr := h.tenantProviderConfig(ctx, claims.TenantID, provider)
	if cfgErr != nil {
		writeError(w, http.StatusConflict, cfgErr.Error())
		return
	}
	reference, refErr := payments.Refund(ctx, provider, cfg, strPtr(providerPaymentID),
		refundAmount, currency, strings.TrimSpace(body.Reason))
	if refErr != nil {
		writeError(w, http.StatusBadGateway, refErr.Error())
		return
	}

	tx, err := h.dbOf(ctx).Begin(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		UPDATE core.wallets SET balance = balance - $2, updated_at = now() WHERE id = $1
	`, walletID, refundAmount); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO core.transactions (tenant_id, wallet_id, type, amount, description, source_type, source_id)
		VALUES ($1, $2, 'debit', $3, $4, 'refund', $5::uuid)
	`, claims.TenantID, walletID, refundAmount,
		"Возврат платежа"+reasonSuffix(body.Reason), paymentID); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	newRefunded := refunded + refundAmount
	newStatus := status
	if newRefunded+0.009 >= amount {
		newStatus = "refunded"
	}
	if _, err := tx.Exec(ctx, `
		UPDATE core.payments SET
			refunded_amount  = $2,
			refunded_at      = now(),
			refund_reference = $3,
			refund_reason    = $4,
			status           = $5,
			updated_at       = now()
		WHERE id = $1
	`, paymentID, newRefunded, reference, strings.TrimSpace(body.Reason), newStatus); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	if err := tx.Commit(ctx); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}

	h.notifyUser(ctx, claims.TenantID, userID, notify.Event{
		Kind:  notify.KindPaymentRefunded,
		Title: "Возврат платежа",
		Body: "Возвращено " + formatMoney(refundAmount) + " " + currency + reasonSuffix(body.Reason) +
			". Деньги вернутся тем же способом, которым были внесены.",
		Action: h.panelAction("К платежам", "/billing"),
		Meta:   map[string]any{"payment_id": paymentID, "amount": refundAmount},
	})
	audit(ctx, h.dbOf(ctx), claims.TenantID, claims.UserID, "payment.refund", "payment:"+paymentID,
		map[string]any{"amount": refundAmount, "provider": provider, "reference": reference})
	h.auditAlert(ctx, claims.TenantID, claims.UserID, claims.Email, "payment.refund", "платёж "+paymentID)
	h.emitWebhook(ctx, claims.TenantID, "payment.refunded", map[string]any{
		"payment_id": paymentID, "amount": refundAmount,
		"currency": currency, "provider": provider, "user_id": userID,
	})

	writeJSON(w, http.StatusOK, map[string]any{
		"status":           newStatus,
		"refunded_amount":  newRefunded,
		"refund_reference": reference,
	})
}

func (h *Handler) AdminRefundSupport(w http.ResponseWriter, r *http.Request) {
	if _, ok := tenantClaims(r.Context()); !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	out := map[string]bool{}
	for _, code := range payments.Supported() {
		out[code] = payments.RefundSupported(code)
	}
	writeJSON(w, http.StatusOK, map[string]any{"providers": out})
}

func isPaidStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "completed", "success", "succeeded", "paid":
		return true
	}
	return false
}

func reasonSuffix(reason string) string {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return ""
	}
	return ": " + reason
}

func formatMoney(v float64) string {
	return strconv.FormatFloat(v, 'f', 2, 64)
}

func (h *Handler) tenantProviderConfig(ctx context.Context, tenantID, provider string) (map[string]any, error) {
	var raw []byte
	err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT config FROM core.payment_providers
		WHERE tenant_id = $1 AND provider = $2 AND enabled = true
		LIMIT 1
	`, tenantID, provider).Scan(&raw)
	if err != nil {
		return nil, fmt.Errorf("шлюз %s не подключён у этого тенанта", provider)
	}
	plain, decErr := h.secrets.DecryptJSON(raw)
	if decErr != nil || len(plain) == 0 {
		plain = raw
	}
	cfg := map[string]any{}
	if err := json.Unmarshal(plain, &cfg); err != nil {
		return nil, fmt.Errorf("настройки шлюза %s нечитаемы", provider)
	}
	return cfg, nil
}

func (h *Handler) walletForUser(ctx context.Context, tenantID, userID, currency string) (string, float64, bool) {
	var walletID string
	var balance float64
	err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT id::text, balance FROM core.wallets
		WHERE tenant_id = $1 AND user_id = $2::uuid AND UPPER(currency) = UPPER($3)
		LIMIT 1
	`, tenantID, userID, currency).Scan(&walletID, &balance)
	if err != nil {
		return "", 0, false
	}
	return walletID, balance, true
}
