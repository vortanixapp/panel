package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/vortanixapp/panel/internal/api/payments"
	"github.com/vortanixapp/panel/pkg/i18n"
	"github.com/vortanixapp/panel/pkg/notify"
)

type refundBody struct {
	Amount float64 `json:"amount"`
	Reason string  `json:"reason"`
}

type refundOutcome struct {
	Status    string  `json:"status"`
	Refunded  float64 `json:"refunded_amount"`
	Reference string  `json:"refund_reference"`
}

type refundFailure struct {
	code    int
	message string
}

func (e *refundFailure) Error() string {
	return e.message
}

func refundFail(code int, message string) error {
	return &refundFailure{code: code, message: message}
}

func (h *Handler) AdminRefundPayment(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var body refundBody
	_ = json.NewDecoder(r.Body).Decode(&body)
	outcome, err := h.refundPayment(r.Context(), claims.UserID, claims.Email, chi.URLParam(r, "id"), body.Amount, body.Reason)
	if err != nil {
		var failure *refundFailure
		if errors.As(err, &failure) {
			writeError(w, failure.code, failure.message)
			return
		}
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	writeJSON(w, http.StatusOK, outcome)
}

func (h *Handler) refundPayment(ctx context.Context, actorID, actorEmail, paymentID string, requested float64, reason string) (refundOutcome, error) {
	reason = strings.TrimSpace(reason)
	var provider, currency, status, userID string
	var providerPaymentID *string
	var amount, refunded float64
	var meta []byte
	err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT provider, currency, status, user_id::text, provider_payment_id,
		       amount, COALESCE(refunded_amount, 0), meta
		FROM core.payments WHERE id = $1
	`, paymentID).Scan(
		&provider, &currency, &status, &userID, &providerPaymentID, &amount, &refunded, &meta)
	if err != nil {
		return refundOutcome{}, refundFail(http.StatusNotFound, "payment not found")
	}
	if !isPaidStatus(status) {
		return refundOutcome{}, refundFail(http.StatusConflict, "возврат возможен только по оплаченному платежу (статус "+status+")")
	}
	if !payments.RefundSupported(provider) {
		return refundOutcome{}, refundFail(http.StatusConflict,
			"у шлюза "+provider+" нет API возврата — верните платёж в его личном кабинете")
	}

	available := amount - refunded
	if available <= 0 {
		return refundOutcome{}, refundFail(http.StatusConflict, "платёж уже возвращён полностью")
	}
	refundAmount := requested
	if refundAmount <= 0 {
		refundAmount = available
	}
	if refundAmount > available+0.009 {
		return refundOutcome{}, refundFail(http.StatusBadRequest, "сумма больше остатка по платежу")
	}

	walletID, balance, ok := h.walletForUser(ctx, userID, currency)
	if !ok {
		return refundOutcome{}, refundFail(http.StatusConflict, "у клиента нет кошелька в валюте платежа")
	}
	if balance+0.009 < refundAmount {
		return refundOutcome{}, refundFail(http.StatusConflict,
			"на балансе клиента меньше суммы возврата — он уже потратил эти деньги; вернуть можно "+formatMoney(balance)+" "+currency)
	}

	providerRow, rowErr := h.loadPaymentProvider(ctx, provider)
	if rowErr != nil || !providerRow.Exists || providerRow.Unreadable {
		return refundOutcome{}, refundFail(http.StatusConflict, "настройки шлюза "+provider+" недоступны")
	}
	var stored struct {
		Receipt *payments.Receipt `json:"receipt"`
	}
	_ = json.Unmarshal(meta, &stored)
	reference, refErr := payments.Refund(ctx, provider, providerRow.Config, strPtr(providerPaymentID),
		refundAmount, currency, reason, stored.Receipt)
	if refErr != nil {
		return refundOutcome{}, refundFail(http.StatusBadGateway, refErr.Error())
	}

	tx, err := h.dbOf(ctx).Begin(ctx)
	if err != nil {
		return refundOutcome{}, refundFail(http.StatusInternalServerError, "database error")
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		UPDATE core.wallets SET balance = balance - $2, updated_at = now() WHERE id = $1
	`, walletID, refundAmount); err != nil {
		return refundOutcome{}, refundFail(http.StatusInternalServerError, "database error")
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO core.transactions ( wallet_id, type, amount, description, source_type, source_id)
		VALUES ( $1, 'debit', $2, $3, 'refund', $4::uuid)
	`, walletID, refundAmount, "Возврат платежа"+reasonSuffix(reason), paymentID); err != nil {
		return refundOutcome{}, refundFail(http.StatusInternalServerError, "database error")
	}
	reverseReferralReward(ctx, tx, paymentID, refundAmount, amount)
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
	`, paymentID, newRefunded, reference, reason, newStatus); err != nil {
		return refundOutcome{}, refundFail(http.StatusInternalServerError, "database error")
	}
	if err := tx.Commit(ctx); err != nil {
		return refundOutcome{}, refundFail(http.StatusInternalServerError, "database error")
	}

	refundKey := "notify.payment_refunded.body"
	if reason != "" {
		refundKey = "notify.payment_refunded.body_reason"
	}
	h.notifyUser(ctx, userID, notify.Event{
		Kind:  notify.KindPaymentRefunded,
		Title: i18n.Key("notify.payment_refunded.title"),
		Body: i18n.Key(refundKey, i18n.Params{
			"amount": formatMoney(refundAmount), "currency": currency, "reason": reason,
		}),
		Action: h.panelAction("notify.action.payments", "/billing"),
		Meta:   map[string]any{"payment_id": paymentID, "amount": refundAmount},
	})
	audit(ctx, h.dbOf(ctx), actorID, "payment.refund", "payment:"+paymentID,
		map[string]any{"amount": refundAmount, "provider": provider, "reference": reference})
	h.auditAlert(ctx, actorID, actorEmail, "payment.refund", "платёж "+paymentID)
	h.emitWebhook(ctx, "payment.refunded", map[string]any{
		"payment_id": paymentID, "amount": refundAmount,
		"currency": currency, "provider": provider, "user_id": userID,
	})

	return refundOutcome{Status: newStatus, Refunded: newRefunded, Reference: reference}, nil
}

func (h *Handler) AdminRefundSupport(w http.ResponseWriter, r *http.Request) {
	if _, ok := tenantClaims(r.Context()); !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	out := map[string]bool{}
	for _, code := range payments.Codes() {
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

func (h *Handler) walletForUser(ctx context.Context, userID, currency string) (string, float64, bool) {
	var walletID string
	var balance float64
	err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT id::text, balance FROM core.wallets
		WHERE user_id = $1::uuid AND UPPER(currency) = UPPER($2)
		LIMIT 1
	`, userID, currency).Scan(&walletID, &balance)
	if err != nil {
		return "", 0, false
	}
	return walletID, balance, true
}
