package payments

import "testing"

const samplePayPalOrderResponse = `{
  "id": "5O190127TN364715T",
  "status": "PAYER_ACTION_REQUIRED",
  "links": [
    {
      "href": "https://api-m.sandbox.paypal.com/v2/checkout/orders/5O190127TN364715T",
      "rel": "self",
      "method": "GET"
    },
    {
      "href": "https://www.sandbox.paypal.com/checkoutnow?token=5O190127TN364715T",
      "rel": "approve",
      "method": "GET"
    },
    {
      "href": "https://api-m.sandbox.paypal.com/v2/checkout/orders/5O190127TN364715T",
      "rel": "update",
      "method": "PATCH"
    },
    {
      "href": "https://api-m.sandbox.paypal.com/v2/checkout/orders/5O190127TN364715T/capture",
      "rel": "capture",
      "method": "POST"
    }
  ]
}`

func TestParsePayPalOrderResponse(t *testing.T) {
	approveURL, orderID, err := parsePayPalOrderResponse([]byte(samplePayPalOrderResponse))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if orderID != "5O190127TN364715T" {
		t.Fatalf("unexpected order id: %s", orderID)
	}
	if approveURL != "https://www.sandbox.paypal.com/checkoutnow?token=5O190127TN364715T" {
		t.Fatalf("unexpected approve url: %s", approveURL)
	}
}

func TestParsePayPalOrderResponse_NoApproveLink(t *testing.T) {
	_, _, err := parsePayPalOrderResponse([]byte(`{"id":"x","links":[{"href":"https://x","rel":"self"}]}`))
	if err == nil {
		t.Fatal("expected error when no approve link is present")
	}
}

func TestPayPalBaseURL(t *testing.T) {
	if got := paypalBaseURL(map[string]any{"mode": "live"}); got != "https://api-m.paypal.com" {
		t.Fatalf("unexpected live base url: %s", got)
	}
	if got := paypalBaseURL(map[string]any{"mode": "sandbox"}); got != "https://api-m.sandbox.paypal.com" {
		t.Fatalf("unexpected sandbox base url: %s", got)
	}
	if got := paypalBaseURL(map[string]any{}); got != "https://api-m.sandbox.paypal.com" {
		t.Fatalf("expected sandbox as the safe default, got: %s", got)
	}
}
