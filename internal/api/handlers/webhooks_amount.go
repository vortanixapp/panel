package handlers

import (
	"context"
	"log"
	"strings"

	"github.com/google/uuid"
)

func (h *Handler) knownPayment(ctx context.Context, paymentID string) (context.Context, bool) {
	id := strings.TrimSpace(paymentID)
	if id == "" {
		return ctx, false
	}
	if _, err := uuid.Parse(id); err != nil {
		return ctx, false
	}
	var exists bool
	if err := h.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM core.payments WHERE id = $1::uuid)`, id,
	).Scan(&exists); err != nil {
		log.Printf("платёж %s не проверен: %v", id, err)
		return ctx, false
	}
	return ctx, exists
}

func (h *Handler) paymentCoversOrder(
	ctx context.Context, paymentID string, paidAmount float64, paidCurrency, provider string,
) bool {
	var orderAmount float64
	var orderCurrency string
	if err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT amount, COALESCE(currency, 'RUB') FROM core.payments WHERE id = $1::uuid
	`, paymentID).Scan(&orderAmount, &orderCurrency); err != nil {
		log.Printf("%s: сумма заказа %s не прочитана: %v", provider, paymentID, err)
		return false
	}
	if paidAmount <= 0 || paidCurrency == "" {
		return true
	}
	if !strings.EqualFold(strings.TrimSpace(paidCurrency), strings.TrimSpace(orderCurrency)) {
		log.Printf("%s: валюта оплаты %s не совпадает с валютой заказа %s по %s — сверка суммы пропущена",
			provider, paidCurrency, orderCurrency, paymentID)
		return true
	}
	if paidAmount+0.009 < orderAmount {
		log.Printf("%s: оплачено меньше заказа по %s: касса %.2f, заказ %.2f",
			provider, paymentID, paidAmount, orderAmount)
		return false
	}
	return true
}
