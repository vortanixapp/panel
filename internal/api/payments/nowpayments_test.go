package payments

import (
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func computeNowPaymentsSig(t *testing.T, secret, sortedBody string) string {
	t.Helper()
	mac := hmac.New(sha512.New, []byte(secret))
	mac.Write([]byte(sortedBody))
	return hex.EncodeToString(mac.Sum(nil))
}

func TestVerifyNowPaymentsSign(t *testing.T) {
	cfg := map[string]any{"ipn_secret": "test-secret"}
	body := []byte(`{"payment_status":"finished","payment_id":123456,"order_id":"abc-1","price_amount":10.50}`)
	sig := computeNowPaymentsSig(t, "test-secret", `{"order_id":"abc-1","payment_id":123456,"payment_status":"finished","price_amount":10.50}`)

	if !VerifyNowPaymentsSign(cfg, body, sig) {
		t.Fatal("expected valid signature to verify")
	}
	if VerifyNowPaymentsSign(cfg, body, "deadbeef") {
		t.Fatal("expected garbage signature to be rejected")
	}
	wrongCfg := map[string]any{"ipn_secret": "other-secret"}
	if VerifyNowPaymentsSign(wrongCfg, body, sig) {
		t.Fatal("expected signature to fail verification under the wrong secret")
	}
}

func TestVerifyNowPaymentsSign_MissingSecretOrSig(t *testing.T) {
	body := []byte(`{"order_id":"abc-1"}`)
	if VerifyNowPaymentsSign(map[string]any{}, body, "sig") {
		t.Fatal("expected rejection when ipn_secret is not configured")
	}
	if VerifyNowPaymentsSign(map[string]any{"ipn_secret": "x"}, body, "") {
		t.Fatal("expected rejection when signature header is empty")
	}
}

func TestNowPaymentsCreateCheckout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/invoice" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("x-api-key") != "test-key" {
			t.Fatalf("missing/incorrect x-api-key header: %q", r.Header.Get("x-api-key"))
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["order_id"] != "pay_123" {
			t.Fatalf("expected order_id pay_123, got %v", body["order_id"])
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":          987654321,
			"invoice_url": "https://nowpayments.io/invoice/abc",
		})
	}))
	defer srv.Close()

	cfg := map[string]any{"api_key": "test-key", "api_url": srv.URL, "price_currency": "USD"}
	url, id, err := NowPayments{}.CreateCheckout(context.Background(), cfg, CheckoutInput{
		PaymentID:   "pay_123",
		Amount:      10.5,
		Currency:    "usd",
		Description: "top-up",
		ReturnURL:   "https://panel.example/return",
		FailURL:     "https://panel.example/fail",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url != "https://nowpayments.io/invoice/abc" {
		t.Fatalf("unexpected redirect url: %s", url)
	}
	if id != "987654321" {
		t.Fatalf("unexpected provider payment id: %s", id)
	}
}

func TestNowPaymentsCreateCheckout_NotConfigured(t *testing.T) {
	_, _, err := NowPayments{}.CreateCheckout(context.Background(), map[string]any{}, CheckoutInput{PaymentID: "x"})
	if err == nil {
		t.Fatal("expected error when api_key is missing")
	}
}
