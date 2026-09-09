package payments

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	shopID := strCfg(cfg, "shop_id")
	secret := strCfg(cfg, "secret_key")
	if shopID == "" || secret == "" {
		return "", fmt.Errorf("yookassa not configured")
	}
	body := map[string]any{
		"payment_id": providerPaymentID,
		"amount": map[string]string{
			"value":    fmt.Sprintf("%.2f", amount),
			"currency": strings.ToUpper(currency),
		},
	}
	if strings.TrimSpace(reason) != "" {
		body["description"] = reason
	}
	raw, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.yookassa.ru/v3/refunds", bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(shopID, secret)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotence-Key", fmt.Sprintf("refund-%s-%.2f", providerPaymentID, amount))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("yookassa refund %d: %s", resp.StatusCode, string(respBody))
	}
	var parsed struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", err
	}
	if parsed.Status == "canceled" {
		return parsed.ID, fmt.Errorf("yookassa отменила возврат")
	}
	return parsed.ID, nil
}

func (Stripe) Refund(ctx context.Context, cfg map[string]any, providerPaymentID string, amount float64, currency, reason string) (string, error) {
	secret := strCfg(cfg, "secret_key")
	if secret == "" {
		return "", fmt.Errorf("stripe not configured")
	}
	stripe.Key = secret

	intentID := providerPaymentID
	if strings.HasPrefix(providerPaymentID, "cs_") {
		sess, err := session.Get(providerPaymentID, nil)
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
		Amount:        stripe.Int64(int64(amount*100 + 0.5)),
	}
	params.Context = ctx
	if strings.TrimSpace(reason) != "" {
		params.AddMetadata("reason", reason)
	}
	created, err := refund.New(params)
	if err != nil {
		return "", err
	}
	return created.ID, nil
}
