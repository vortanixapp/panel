package yookassa

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const apiBase = "https://api.yookassa.ru/v3"

type Payment struct {
	ID       string
	Status   string
	Paid     bool
	Amount   float64
	Currency string
	Metadata map[string]string
}

type Client struct {
	ShopID  string
	Secret  string
	BaseURL string
	HTTP    *http.Client
}

func New(shopID, secret string) *Client {
	return &Client{
		ShopID:  shopID,
		Secret:  secret,
		BaseURL: apiBase,
		HTTP:    &http.Client{Timeout: 10 * time.Second},
	}
}

var ErrNotConfigured = fmt.Errorf("yookassa: не заданы shop_id/secret_key — проверить платёж нечем")

func (c *Client) GetPayment(ctx context.Context, paymentID string) (Payment, error) {
	var out Payment
	if c == nil || strings.TrimSpace(c.ShopID) == "" || strings.TrimSpace(c.Secret) == "" {
		return out, ErrNotConfigured
	}
	paymentID = strings.TrimSpace(paymentID)
	if paymentID == "" {
		return out, fmt.Errorf("yookassa: пустой идентификатор платежа")
	}

	base := c.BaseURL
	if base == "" {
		base = apiBase
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/payments/"+paymentID, nil)
	if err != nil {
		return out, err
	}
	req.SetBasicAuth(c.ShopID, c.Secret)

	client := c.HTTP
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return out, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 300 {
		return out, fmt.Errorf("yookassa api %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var wire struct {
		ID     string `json:"id"`
		Status string `json:"status"`
		Paid   bool   `json:"paid"`
		Amount struct {
			Value    string `json:"value"`
			Currency string `json:"currency"`
		} `json:"amount"`
		Metadata map[string]string `json:"metadata"`
	}
	if err := json.Unmarshal(body, &wire); err != nil {
		return out, err
	}
	amount, _ := strconv.ParseFloat(wire.Amount.Value, 64)
	return Payment{
		ID:       wire.ID,
		Status:   wire.Status,
		Paid:     wire.Paid,
		Amount:   amount,
		Currency: wire.Amount.Currency,
		Metadata: wire.Metadata,
	}, nil
}

func (p Payment) Succeeded() bool {
	return p.Status == "succeeded" && p.Paid
}

func (p Payment) Canceled() bool {
	return p.Status == "canceled"
}
