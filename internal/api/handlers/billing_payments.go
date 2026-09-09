package handlers

import (
	"context"
	"encoding/json"
	"strings"

	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/vortanixapp/panel/pkg/notify"
)

func (h *Handler) completeTopupPayment(ctx context.Context, paymentID, providerPaymentID, providerLabel string) error {
	tx, err := h.dbOf(ctx).Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var tenantID, userID, status, currency string
	var amount float64
	var meta []byte
	err = tx.QueryRow(ctx, `
		SELECT tenant_id::text, user_id::text, status, amount, currency, meta
		FROM core.payments WHERE id = $1 FOR UPDATE
	`, paymentID).Scan(&tenantID, &userID, &status, &amount, &currency, &meta)
	if err != nil {
		return err
	}
	if status == "completed" {
		return tx.Commit(ctx)
	}
	if currency == "" {
		currency = "RUB"
	}

	creditAmount := amount
	var promoID *string
	if len(meta) > 0 {
		var m map[string]any
		if json.Unmarshal(meta, &m) == nil {
			if v, ok := m["credited_amount"]; ok {
				creditAmount = floatFromAny(v)
			}
			if pid, ok := m["promo_id"].(string); ok && pid != "" {
				promoID = &pid
			}
		}
	}
	if creditAmount <= 0 {
		creditAmount = amount
	}

	var walletID string
	err = tx.QueryRow(ctx, `
		INSERT INTO core.wallets (tenant_id, user_id, currency, is_default)
		VALUES ($1::uuid, $2::uuid, $3, true)
		ON CONFLICT (user_id, currency) DO UPDATE SET updated_at = now()
		RETURNING id::text
	`, tenantID, userID, currency).Scan(&walletID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		UPDATE core.wallets SET balance = balance + $2, updated_at = now() WHERE id = $1::uuid
	`, walletID, creditAmount)
	if err != nil {
		return err
	}

	desc := "Top-up"
	if providerLabel != "" {
		desc = "Top-up via " + providerLabel
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO core.transactions (tenant_id, wallet_id, type, amount, description, source_type, source_id)
		VALUES ($1::uuid, $2::uuid, 'credit', $3, $4, 'payment', $5::uuid)
	`, tenantID, walletID, creditAmount, desc, paymentID)
	if err != nil {
		return err
	}

	if promoID != nil {
		_, _ = tx.Exec(ctx, `
			UPDATE core.promotions
			SET used_count = used_count + 1
			WHERE id = $1::uuid AND (max_uses IS NULL OR used_count < max_uses)
		`, *promoID)
	}

	tag, err := tx.Exec(ctx, `
		UPDATE core.payments
		SET status = 'completed', wallet_id = $2::uuid, provider_payment_id = $3,
		    promotion_id = COALESCE($4::uuid, promotion_id),
		    credited_at = now(), updated_at = now()
		WHERE id = $1::uuid AND status <> 'completed'
	`, paymentID, walletID, providerPaymentID, promoID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	h.emitWebhook(ctx, tenantID, "payment.completed", map[string]any{
		"payment_id": paymentID, "user_id": userID,
		"amount": creditAmount, "currency": currency, "provider": providerLabel,
	})
	h.notifyUser(ctx, tenantID, userID, notify.Event{
		Kind:      notify.KindPaymentReceived,
		Title:     "Платёж зачислен",
		Body:      fmt.Sprintf("Баланс пополнен на %.2f %s (%s).", creditAmount, currency, providerLabel),
		Action:    h.panelAction("К балансу", "/billing"),
		Meta:      map[string]any{"payment_id": paymentID, "amount": creditAmount, "currency": currency},
		DedupeKey: "payment.received:" + paymentID,
	})
	return nil
}

func paymentIDFromMeta(meta map[string]string) string {
	if meta == nil {
		return ""
	}
	for _, k := range []string{"payment_id", "vortanix_payment_id", "order_id"} {
		if v := strings.TrimSpace(meta[k]); v != "" {
			return v
		}
	}
	return ""
}
