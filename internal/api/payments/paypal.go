package payments

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type PayPal struct{}

func (PayPal) Code() string { return "paypal" }

func paypalBaseURL(cfg map[string]any) string {
	if strCfg(cfg, "mode") == "live" {
		return "https://api-m.paypal.com"
	}
	return "https://api-m.sandbox.paypal.com"
}

func paypalAccessToken(ctx context.Context, base, clientID, clientSecret string) (string, error) {
	form := url.Values{"grant_type": {"client_credentials"}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/v1/oauth2/token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(clientID, clientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("paypal oauth %d: %s", resp.StatusCode, string(body))
	}
	var parsed struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", err
	}
	if parsed.AccessToken == "" {
		return "", fmt.Errorf("paypal: empty access token")
	}
	return parsed.AccessToken, nil
}

func (PayPal) CreateCheckout(ctx context.Context, cfg map[string]any, in CheckoutInput) (string, string, error) {
	clientID := strCfg(cfg, "client_id")
	clientSecret := strCfg(cfg, "client_secret")
	if clientID == "" || clientSecret == "" {
		return "", "", fmt.Errorf("paypal not configured")
	}
	currency := strCfg(cfg, "currency")
	if currency == "" {
		currency = strings.ToUpper(in.Currency)
	}
	if currency == "" {
		currency = "USD"
	}
	base := paypalBaseURL(cfg)
	token, err := paypalAccessToken(ctx, base, clientID, clientSecret)
	if err != nil {
		return "", "", err
	}

	body := map[string]any{
		"intent": "CAPTURE",
		"purchase_units": []map[string]any{
			{
				"reference_id": in.PaymentID,
				"custom_id":    in.PaymentID,
				"description":  in.Description,
				"amount": map[string]string{
					"currency_code": currency,
					"value":         fmt.Sprintf("%.2f", in.Amount),
				},
			},
		},
		"application_context": map[string]any{
			"return_url":  in.ReturnURL,
			"cancel_url":  in.FailURL,
			"user_action": "PAY_NOW",
		},
	}
	raw, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/v2/checkout/orders", bytes.NewReader(raw))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return "", "", fmt.Errorf("paypal orders api %d: %s", resp.StatusCode, string(respBody))
	}
	return parsePayPalOrderResponse(respBody)
}

func parsePayPalOrderResponse(respBody []byte) (approveURL, orderID string, err error) {
	var parsed struct {
		ID    string `json:"id"`
		Links []struct {
			Href string `json:"href"`
			Rel  string `json:"rel"`
		} `json:"links"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", "", err
	}
	for _, l := range parsed.Links {
		if l.Rel == "approve" {
			return l.Href, parsed.ID, nil
		}
	}
	return "", "", fmt.Errorf("paypal: no approve link in response")
}

func CapturePayPalOrder(ctx context.Context, cfg map[string]any, orderID string) error {
	clientID := strCfg(cfg, "client_id")
	clientSecret := strCfg(cfg, "client_secret")
	if clientID == "" || clientSecret == "" {
		return fmt.Errorf("paypal not configured")
	}
	base := paypalBaseURL(cfg)
	token, err := paypalAccessToken(ctx, base, clientID, clientSecret)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/v2/checkout/orders/"+url.PathEscape(orderID)+"/capture", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 && !strings.Contains(string(body), "ORDER_ALREADY_CAPTURED") {
		return fmt.Errorf("paypal capture %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

func VerifyPayPalWebhook(ctx context.Context, cfg map[string]any, headers http.Header, rawBody []byte) (bool, error) {
	clientID := strCfg(cfg, "client_id")
	clientSecret := strCfg(cfg, "client_secret")
	webhookID := strCfg(cfg, "webhook_id")
	if clientID == "" || clientSecret == "" || webhookID == "" {
		return false, fmt.Errorf("paypal webhook not configured")
	}
	base := paypalBaseURL(cfg)
	token, err := paypalAccessToken(ctx, base, clientID, clientSecret)
	if err != nil {
		return false, err
	}
	var event any
	if err := json.Unmarshal(rawBody, &event); err != nil {
		return false, err
	}
	verifyBody := map[string]any{
		"transmission_id":   headers.Get("Paypal-Transmission-Id"),
		"transmission_time": headers.Get("Paypal-Transmission-Time"),
		"cert_url":          headers.Get("Paypal-Cert-Url"),
		"auth_algo":         headers.Get("Paypal-Auth-Algo"),
		"transmission_sig":  headers.Get("Paypal-Transmission-Sig"),
		"webhook_id":        webhookID,
		"webhook_event":     event,
	}
	raw, _ := json.Marshal(verifyBody)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/v1/notifications/verify-webhook-signature", bytes.NewReader(raw))
	if err != nil {
		return false, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return false, fmt.Errorf("paypal verify %d: %s", resp.StatusCode, string(respBody))
	}
	var parsed struct {
		VerificationStatus string `json:"verification_status"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return false, err
	}
	return parsed.VerificationStatus == "SUCCESS", nil
}
