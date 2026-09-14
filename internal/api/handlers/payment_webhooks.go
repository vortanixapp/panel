package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/vortanixapp/panel/internal/api/payments"
)

type notifiedPayment struct {
	ID        string
	Provider  string
	Status    string
	Currency  string
	Amount    float64
	InvoiceNo int64
	Meta      map[string]any
}

func writePaymentResponse(w http.ResponseWriter, resp payments.Response) {
	if resp.ContentType != "" {
		w.Header().Set("Content-Type", resp.ContentType)
	}
	status := resp.Status
	if status == 0 {
		status = http.StatusOK
	}
	w.WriteHeader(status)
	_, _ = io.WriteString(w, resp.Body)
}

func (h *Handler) PaymentWebhook(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "provider")
	provider, err := payments.Get(code)
	if err != nil {
		writeError(w, http.StatusNotFound, "unknown payment provider")
		return
	}
	notifier, ok := provider.(payments.Notifier)
	if !ok {
		writeError(w, http.StatusNotFound, "provider does not accept notifications")
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	ctx := r.Context()

	row, err := h.loadPaymentProvider(ctx, code)
	if err != nil || !row.Exists || row.Unreadable {
		log.Printf("%s: уведомление пришло, но настройки провайдера недоступны", code)
		writePaymentResponse(w, payments.RejectResponse(provider, payments.Notification{}, errors.New("provider is not configured")))
		return
	}

	req := payments.NewNotifyRequest(r, body, h.paymentNotifyURL(code))
	n, err := notifier.HandleNotification(ctx, row.Config, req)
	if err != nil {
		log.Printf("%s: уведомление отклонено: %v", code, err)
		writePaymentResponse(w, payments.RejectResponse(provider, n, err))
		return
	}
	if n.Status == payments.NotifyIgnore {
		writePaymentResponse(w, payments.AcceptResponse(provider, n))
		return
	}

	payment, err := h.findNotifiedPayment(ctx, code, n)
	if err != nil {
		log.Printf("%s: %v", code, err)
		writePaymentResponse(w, payments.RejectResponse(provider, n, err))
		return
	}

	switch n.Status {
	case payments.NotifyCheck:
		if payment.Status != "pending" && payment.Status != "processing" {
			writePaymentResponse(w, payments.RejectResponse(provider, n, fmt.Errorf("платёж уже обработан")))
			return
		}
		if !paymentCovers(code, payment, n) {
			writePaymentResponse(w, payments.RejectResponse(provider, n, fmt.Errorf("сумма не совпадает с заказом")))
			return
		}
	case payments.NotifyFailed:
		if payment.Status == "pending" || payment.Status == "processing" {
			_, _ = h.dbOf(ctx).Exec(ctx, `
				UPDATE core.payments SET status = 'failed', updated_at = now()
				WHERE id = $1 AND status IN ('pending', 'processing')
			`, payment.ID)
		}
	case payments.NotifyPaid:
		if payment.Status == "refunded" {
			log.Printf("%s: повторное уведомление об оплате возвращённого платежа %s пропущено", code, payment.ID)
			break
		}
		if !paymentCovers(code, payment, n) {
			writePaymentResponse(w, payments.RejectResponse(provider, n, fmt.Errorf("оплаченная сумма меньше заказа")))
			return
		}
		if err := h.completeTopupPayment(ctx, payment.ID, n.ProviderPaymentID, payments.Name(code)); err != nil {
			log.Printf("%s: платёж %s не зачислен: %v", code, payment.ID, err)
			writePaymentResponse(w, payments.Response{Status: http.StatusInternalServerError, ContentType: "text/plain; charset=utf-8", Body: "failed"})
			return
		}
	}
	writePaymentResponse(w, payments.AcceptResponse(provider, n))
}

func (h *Handler) findNotifiedPayment(ctx context.Context, code string, n payments.Notification) (notifiedPayment, error) {
	const columns = `id::text, provider, status, currency, amount::float8, invoice_no, meta`
	var row pgx.Row
	switch {
	case n.PaymentID != "":
		if _, err := uuid.Parse(n.PaymentID); err != nil {
			return notifiedPayment{}, fmt.Errorf("в уведомлении неверный номер платежа %q", n.PaymentID)
		}
		row = h.dbOf(ctx).QueryRow(ctx, `SELECT `+columns+` FROM core.payments WHERE id = $1`, n.PaymentID)
	case n.InvoiceNo > 0:
		row = h.dbOf(ctx).QueryRow(ctx, `SELECT `+columns+` FROM core.payments WHERE invoice_no = $1`, n.InvoiceNo)
	case n.ProviderPaymentID != "":
		row = h.dbOf(ctx).QueryRow(ctx, `
			SELECT `+columns+` FROM core.payments
			WHERE provider = $1 AND provider_payment_id = $2
			ORDER BY created_at DESC LIMIT 1
		`, code, n.ProviderPaymentID)
	default:
		return notifiedPayment{}, fmt.Errorf("в уведомлении нет номера платежа")
	}

	var p notifiedPayment
	var meta []byte
	if err := row.Scan(&p.ID, &p.Provider, &p.Status, &p.Currency, &p.Amount, &p.InvoiceNo, &meta); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return p, fmt.Errorf("платёж из уведомления не найден")
		}
		return p, err
	}
	if p.Provider != code {
		return p, fmt.Errorf("платёж %s создан через %s, а уведомление пришло от %s", p.ID, p.Provider, code)
	}
	if n.PaymentID != "" && n.InvoiceNo > 0 && n.InvoiceNo != p.InvoiceNo {
		return p, fmt.Errorf("номер счёта %d не совпадает с платежом %s", n.InvoiceNo, p.ID)
	}
	p.Meta = map[string]any{}
	_ = json.Unmarshal(meta, &p.Meta)
	return p, nil
}

func paymentCovers(code string, p notifiedPayment, n payments.Notification) bool {
	expected, currency := p.Amount, p.Currency
	if v, ok := p.Meta["charge_amount"].(float64); ok && v > 0 {
		expected = v
		if c, ok := p.Meta["charge_currency"].(string); ok && c != "" {
			currency = c
		}
	}
	if n.Amount <= 0 || n.Currency == "" {
		return true
	}
	if !strings.EqualFold(n.Currency, currency) {
		log.Printf("%s: валюта оплаты %s не совпадает с валютой счёта %s по %s — сверка суммы пропущена", code, n.Currency, currency, p.ID)
		return true
	}
	if n.Amount+0.009 < expected {
		log.Printf("%s: оплачено меньше счёта по %s: касса %.2f, счёт %.2f", code, p.ID, n.Amount, expected)
		return false
	}
	return true
}
