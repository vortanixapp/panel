package payments

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/checkout/session"
	"github.com/stripe/stripe-go/v81/refund"
)

type Refunder interface {
	Provider
	Refund(ctx context.Context, cfg map[string]any, providerPaymentID string, amount float64, currency, reason string) (string, error)
}

func RefundSupported(code string) bool {
	p, err := Get(code)
	if err != nil {
		return false
	}
	_, refundable := p.(Refunder)
	return refundable
}

func Refund(ctx context.Context, code string, cfg map[string]any, providerPaymentID string, amount float64, currency, reason string) (string, error) {
	p, err := Get(code)
	if err != nil {
		return "", fmt.Errorf("платёжный провайдер %s не подключён", code)
	}
	r, refundable := p.(Refunder)
	if !refundable {
		return "", fmt.Errorf("у провайдера %s нет API возврата — верните платёж в его личном кабинете", code)
	}
	if strings.TrimSpace(providerPaymentID) == "" {
		return "", fmt.Errorf("у платежа нет идентификатора в шлюзе — возврат по API невозможен")
	}
	return r.Refund(ctx, cfg, providerPaymentID, amount, currency, reason)
}

func (YooKassa) Refund(ctx context.Context, cfg map[string]any, providerPaymentID string, amount float64, currency, reason string) (string, error) {
	shopID, secret := strCfg(cfg, "shop_id"), strCfg(cfg, "secret_key")
	if shopID == "" || secret == "" {
		return "", notConfigured("yookassa")
	}
	body := map[string]any{
		"payment_id": providerPaymentID,
		"amount": map[string]string{
			"value":    money(amount),
			"currency": upper(currency),
		},
	}
	if strings.TrimSpace(reason) != "" {
		body["description"] = reason
	}
	var out struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	err := sendJSON(ctx, http.MethodPost, "https://api.yookassa.ru/v3/refunds", body, &out,
		withBasicAuth(shopID, secret),
		withHeader("Idempotence-Key", fmt.Sprintf("refund-%s-%.2f", providerPaymentID, amount)))
	if err != nil {
		return "", fmt.Errorf("yookassa refund: %w", err)
	}
	if out.Status == "canceled" {
		return out.ID, fmt.Errorf("yookassa отменила возврат")
	}
	return out.ID, nil
}

func (Stripe) Refund(ctx context.Context, cfg map[string]any, providerPaymentID string, amount float64, currency, reason string) (string, error) {
	secret := stripeSecret(cfg)
	if secret == "" {
		return "", notConfigured("stripe")
	}
	backend := stripe.GetBackend(stripe.APIBackend)

	intentID := providerPaymentID
	if strings.HasPrefix(providerPaymentID, "cs_") {
		params := &stripe.CheckoutSessionParams{}
		params.Context = ctx
		sess, err := session.Client{B: backend, Key: secret}.Get(providerPaymentID, params)
		if err != nil {
			return "", err
		}
		if sess.PaymentIntent == nil || sess.PaymentIntent.ID == "" {
			return "", fmt.Errorf("stripe: у сессии %s нет платежа", providerPaymentID)
		}
		intentID = sess.PaymentIntent.ID
	}

	params := &stripe.RefundParams{
		PaymentIntent: stripe.String(intentID),
		Amount:        stripe.Int64(minorUnits(amount, currency)),
	}
	params.Context = ctx
	if strings.TrimSpace(reason) != "" {
		params.AddMetadata("reason", reason)
	}
	created, err := refund.Client{B: backend, Key: secret}.New(params)
	if err != nil {
		return "", err
	}
	return created.ID, nil
}
