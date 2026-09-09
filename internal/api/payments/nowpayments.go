package payments

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type NowPayments struct{}

func (NowPayments) Code() string { return "nowpayments" }

func nowPaymentsAPIURL(cfg map[string]any) string {
	base := strCfg(cfg, "api_url")
	if base == "" {
		base = "https://api.nowpayments.io"
	}
	return strings.TrimSuffix(base, "/")
}

func (NowPayments) CreateCheckout(ctx context.Context, cfg map[string]any, in CheckoutInput) (string, string, error) {
	apiKey := strCfg(cfg, "api_key")
	if apiKey == "" {
		return "", "", fmt.Errorf("nowpayments not configured")
	}
	priceCurrency := strCfg(cfg, "price_currency")
	if priceCurrency == "" {
		priceCurrency = "USD"
	}
	payCurrency := strCfg(cfg, "pay_currency")

	body := map[string]any{
		"price_amount":      in.Amount,
		"price_currency":    strings.ToLower(priceCurrency),
		"order_id":          in.PaymentID,
		"order_description": in.Description,
		"success_url":       in.ReturnURL,
		"cancel_url":        in.FailURL,
	}
	if payCurrency != "" {
		body["pay_currency"] = strings.ToLower(payCurrency)
	}
	raw, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, nowPaymentsAPIURL(cfg)+"/v1/invoice", bytes.NewReader(raw))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return "", "", fmt.Errorf("nowpayments api %d: %s", resp.StatusCode, string(respBody))
	}
	var parsed struct {
		ID         json.Number `json:"id"`
		InvoiceURL string      `json:"invoice_url"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", "", err
	}
	if parsed.InvoiceURL == "" {
		return "", "", fmt.Errorf("nowpayments: no invoice_url in response")
	}
	return parsed.InvoiceURL, parsed.ID.String(), nil
}

func VerifyNowPaymentsSign(cfg map[string]any, rawBody []byte, sigHeader string) bool {
	secret := strCfg(cfg, "ipn_secret")
	if secret == "" || sigHeader == "" {
		return false
	}
	dec := json.NewDecoder(bytes.NewReader(rawBody))
	dec.UseNumber()
	var data map[string]any
	if err := dec.Decode(&data); err != nil {
		return false
	}
	sorted, err := json.Marshal(data)
	if err != nil {
		return false
	}
	mac := hmac.New(sha512.New, []byte(secret))
	mac.Write(sorted)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(strings.TrimSpace(sigHeader)))
}
