package yookassa

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetPaymentParsesResponse(t *testing.T) {
	var gotAuth, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"2d1f0e1e-000f-5000-9000-1f0e1e000001",
			"status":"succeeded",
			"paid":true,
			"amount":{"value":"1490.00","currency":"RUB"},
			"metadata":{"payment_id":"local-42"}
		}`))
	}))
	defer srv.Close()

	c := New("shop-1", "secret-1")
	c.BaseURL = srv.URL
	p, err := c.GetPayment(context.Background(), "2d1f0e1e-000f-5000-9000-1f0e1e000001")
	if err != nil {
		t.Fatalf("GetPayment: %v", err)
	}
	if !p.Succeeded() {
		t.Errorf("платёж должен считаться оплаченным: %+v", p)
	}
	if p.Amount != 1490.00 {
		t.Errorf("сумма %v, ожидалось 1490", p.Amount)
	}
	if p.Metadata["payment_id"] != "local-42" {
		t.Errorf("metadata не разобрана: %+v", p.Metadata)
	}
	if !strings.HasPrefix(gotAuth, "Basic ") {
		t.Errorf("запрос без Basic-авторизации: %q", gotAuth)
	}
	if !strings.HasSuffix(gotPath, "/payments/2d1f0e1e-000f-5000-9000-1f0e1e000001") {
		t.Errorf("неверный путь: %q", gotPath)
	}
}

func TestNotYetPaidIsNotSucceeded(t *testing.T) {
	cases := []Payment{
		{Status: "waiting_for_capture", Paid: true},
		{Status: "succeeded", Paid: false},
		{Status: "pending", Paid: false},
	}
	for _, p := range cases {
		if p.Succeeded() {
			t.Errorf("%+v не должен считаться оплаченным", p)
		}
	}
}

func TestMissingCredentialsIsAnError(t *testing.T) {
	for _, c := range []*Client{nil, New("", ""), New("shop", "")} {
		if _, err := c.GetPayment(context.Background(), "abc"); err == nil {
			t.Error("без ключей должна возвращаться ошибка, а не успех")
		}
	}
}

func TestApiErrorIsPropagated(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"type":"error","code":"not_found"}`))
	}))
	defer srv.Close()
	c := New("shop", "secret")
	c.BaseURL = srv.URL
	if _, err := c.GetPayment(context.Background(), "missing"); err == nil {
		t.Fatal("404 от кассы должен быть ошибкой")
	}
}
