package payments

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type YooKassa struct{}

func (YooKassa) Code() string { return "yookassa" }

func (YooKassa) CreateCheckout(ctx context.Context, cfg map[string]any, in CheckoutInput) (string, string, error) {
	shopID := strCfg(cfg, "shop_id")
	secret := strCfg(cfg, "secret_key")
	if shopID == "" || secret == "" {
		return "", "", fmt.Errorf("yookassa not configured")
	}
	amountStr := fmt.Sprintf("%.2f", in.Amount)
	body := map[string]any{
		"amount":       map[string]string{"value": amountStr, "currency": strings.ToUpper(in.Currency)},
		"confirmation": map[string]string{"type": "redirect", "return_url": in.ReturnURL},
		"capture":      true,
		"description":  in.Description,
		"metadata":     map[string]string{"payment_id": in.PaymentID, "order_id": in.PaymentID},
	}
	raw, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.yookassa.ru/v3/payments", bytes.NewReader(raw))
	if err != nil {
		return "", "", err
	}
	req.SetBasicAuth(shopID, secret)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotence-Key", in.PaymentID)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return "", "", fmt.Errorf("yookassa api %d: %s", resp.StatusCode, string(respBody))
	}
	var parsed struct {
		ID           string `json:"id"`
		Confirmation struct {
			ConfirmationURL string `json:"confirmation_url"`
		} `json:"confirmation"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", "", err
	}
	if parsed.Confirmation.ConfirmationURL == "" {
		return "", "", fmt.Errorf("yookassa: no confirmation_url")
	}
	return parsed.Confirmation.ConfirmationURL, parsed.ID, nil
}

type Freekassa struct{}

func (Freekassa) Code() string { return "freekassa" }

func (Freekassa) CreateCheckout(_ context.Context, cfg map[string]any, in CheckoutInput) (string, string, error) {
	merchantID := strCfg(cfg, "merchant_id")
	secret1 := strCfg(cfg, "secret1")
	if merchantID == "" || secret1 == "" {
		return "", "", fmt.Errorf("freekassa not configured")
	}
	amount := fmt.Sprintf("%.2f", in.Amount)
	sign := md5Hex(merchantID + ":" + amount + ":" + secret1 + ":" + in.PaymentID)
	q := url.Values{}
	q.Set("m", merchantID)
	q.Set("oa", amount)
	q.Set("o", in.PaymentID)
	q.Set("s", sign)
	q.Set("currency", strings.ToUpper(in.Currency))
	if in.MethodID != "" {
		q.Set("i", in.MethodID)
	}
	if in.Email != "" {
		q.Set("em", in.Email)
	}
	return "https://pay.fk.money/?" + q.Encode(), in.PaymentID, nil
}

func VerifyFreekassaSign(cfg map[string]any, merchantID, amount, orderID, sign string) bool {
	secret2 := strCfg(cfg, "secret2")
	expected := md5Hex(merchantID + ":" + amount + ":" + secret2 + ":" + orderID)
	return strings.EqualFold(expected, sign)
}

type Robokassa struct{}

func (Robokassa) Code() string { return "robokassa" }

func (Robokassa) CreateCheckout(_ context.Context, cfg map[string]any, in CheckoutInput) (string, string, error) {
	login := strCfg(cfg, "merchant_login")
	pass1 := strCfg(cfg, "password1")
	if login == "" || pass1 == "" {
		return "", "", fmt.Errorf("robokassa not configured")
	}
	outSum := fmt.Sprintf("%.2f", in.Amount)
	sign := md5Hex(login + ":" + outSum + ":" + in.PaymentID + ":" + pass1)
	q := url.Values{}
	q.Set("MerchantLogin", login)
	q.Set("OutSum", outSum)
	q.Set("InvId", in.PaymentID)
	q.Set("SignatureValue", strings.ToUpper(sign))
	q.Set("Description", in.Description)
	if in.ReturnURL != "" {
		q.Set("SuccessURL", in.ReturnURL)
	}
	if in.FailURL != "" {
		q.Set("FailURL", in.FailURL)
	}
	return "https://auth.robokassa.ru/Merchant/Index.aspx?" + q.Encode(), in.PaymentID, nil
}

func VerifyRobokassaResult(cfg map[string]any, outSum, invID, signature string) bool {
	pass2 := strCfg(cfg, "password2")
	expected := strings.ToUpper(md5Hex(outSum + ":" + invID + ":" + pass2))
	return strings.EqualFold(expected, strings.TrimSpace(signature))
}

type Stripe struct{}

func (Stripe) Code() string { return "stripe" }

func (Stripe) CreateCheckout(_ context.Context, cfg map[string]any, in CheckoutInput) (string, string, error) {
	secret := strCfg(cfg, "secret_key")
	if secret == "" {
		panel := strings.TrimSuffix(in.ReturnURL, "/billing/topup/payment/"+in.PaymentID)
		return panel + "/billing/topup/payment/" + in.PaymentID, in.PaymentID, nil
	}
	return createStripeCheckout(secret, in)
}

func strCfg(cfg map[string]any, key string) string {
	if cfg == nil {
		return ""
	}
	v, ok := cfg[key]
	if !ok {
		return ""
	}
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case float64:
		return strconv.FormatInt(int64(t), 10)
	default:
		return fmt.Sprint(v)
	}
}

func md5Hex(s string) string {
	h := md5.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}

func panelPaymentURL(base, paymentID string) string {
	base = strings.TrimSuffix(base, "/")
	if base == "" {
		return "/billing/topup/payment/" + paymentID
	}
	return base + "/billing/topup/payment/" + paymentID
}

func defaultReturnURLs(panelBase, paymentID string) (success, fail string) {
	success = panelPaymentURL(panelBase, paymentID)
	fail = success
	return
}
