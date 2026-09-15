package payments

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

type ClosingReceiptInput struct {
	OffsetID          string
	PaymentID         string
	ProviderPaymentID string
	InvoiceNo         int64
	UserID            string
	Amount            float64
	Currency          string
	Item              string
	SellerINN         string
	Receipt           Receipt
}

type ClosingReceipter interface {
	SendClosingReceipt(ctx context.Context, cfg map[string]any, in ClosingReceiptInput) (string, error)
}

func ClosingReceiptSupported(code string) bool {
	p, err := Get(code)
	if err != nil {
		return false
	}
	_, ok := p.(ClosingReceipter)
	return ok
}

func SendClosingReceipt(ctx context.Context, code string, cfg map[string]any, in ClosingReceiptInput) (string, error) {
	p, err := Get(code)
	if err != nil {
		return "", fmt.Errorf("платёжный провайдер %s не подключён", code)
	}
	sender, ok := p.(ClosingReceipter)
	if !ok {
		return "", fmt.Errorf("у провайдера %s нет API чека зачёта предоплаты", code)
	}
	if strings.TrimSpace(in.Receipt.Email) == "" {
		return "", fmt.Errorf("у платежа нет email покупателя для чека")
	}
	return sender.SendClosingReceipt(ctx, cfg, in)
}

func (in ClosingReceiptInput) full() *Receipt {
	r := in.Receipt
	r.Mode = "full_payment"
	r.Item = ""
	return &r
}

func (in ClosingReceiptInput) invoiceRef() string {
	if in.InvoiceNo > 0 {
		return strconv.FormatInt(in.InvoiceNo, 10)
	}
	return in.PaymentID
}

func (YooKassa) SendClosingReceipt(ctx context.Context, cfg map[string]any, in ClosingReceiptInput) (string, error) {
	shopID, secret := strCfg(cfg, "shop_id"), strCfg(cfg, "secret_key")
	if shopID == "" || secret == "" {
		return "", notConfigured("yookassa")
	}
	if strings.TrimSpace(in.ProviderPaymentID) == "" {
		return "", fmt.Errorf("yookassa: у платежа нет идентификатора в кассе")
	}
	full := in.full()
	fallback, _ := strconv.Atoi(strCfg(cfg, "vat_code"))
	amount := map[string]string{"value": money(in.Amount), "currency": upper(in.Currency)}
	body := map[string]any{
		"customer":   map[string]string{"email": in.Receipt.Email},
		"payment_id": in.ProviderPaymentID,
		"type":       "payment",
		"send":       true,
		"items": []map[string]any{{
			"description":     full.item(in.Item),
			"quantity":        "1.00",
			"amount":          amount,
			"vat_code":        full.yookassaVAT(fallback),
			"payment_mode":    "full_payment",
			"payment_subject": "service",
		}},
		"settlements": []map[string]any{{"type": "prepayment", "amount": amount}},
	}
	if code, ok := yookassaTaxSystems[full.sno()]; ok {
		body["tax_system_code"] = code
	}
	var out struct {
		ID string `json:"id"`
	}
	if err := sendJSON(ctx, http.MethodPost, "https://api.yookassa.ru/v3/receipts", body, &out,
		withBasicAuth(shopID, secret), withHeader("Idempotence-Key", "offset-"+in.OffsetID)); err != nil {
		return "", fmt.Errorf("yookassa: %w", err)
	}
	return out.ID, nil
}

func (TKassa) SendClosingReceipt(ctx context.Context, cfg map[string]any, in ClosingReceiptInput) (string, error) {
	terminal, password := strCfg(cfg, "terminal_key"), strCfg(cfg, "password")
	if terminal == "" || password == "" {
		return "", notConfigured("tkassa")
	}
	if strings.TrimSpace(in.ProviderPaymentID) == "" {
		return "", fmt.Errorf("t-kassa: у платежа нет идентификатора в кассе")
	}
	full := in.full()
	amount := minorUnits(in.Amount, "RUB")
	receipt := map[string]any{
		"Email": in.Receipt.Email,
		"Items": []map[string]any{{
			"Name":          full.item(in.Item),
			"Price":         amount,
			"Quantity":      1,
			"Amount":        amount,
			"Tax":           full.atolTax(),
			"PaymentMethod": "full_payment",
			"PaymentObject": "service",
		}},
		"Payments": map[string]any{"AdvancePayment": amount},
	}
	if sno := full.sno(); sno != "" {
		receipt["Taxation"] = sno
	}
	body := map[string]any{
		"TerminalKey": terminal,
		"PaymentId":   in.ProviderPaymentID,
		"Receipt":     receipt,
		"Token":       tkassaToken(map[string]string{"TerminalKey": terminal, "PaymentId": in.ProviderPaymentID}, password),
	}
	var out struct {
		Success   bool   `json:"Success"`
		ErrorCode string `json:"ErrorCode"`
		Message   string `json:"Message"`
		Details   string `json:"Details"`
	}
	if err := sendJSON(ctx, http.MethodPost, urlCfg(cfg, "api_url", "https://securepay.tinkoff.ru/v2")+"/SendClosingReceipt", body, &out); err != nil {
		return "", fmt.Errorf("t-kassa: %w", err)
	}
	if !out.Success {
		return "", fmt.Errorf("t-kassa: %s %s (код %s)", out.Message, out.Details, out.ErrorCode)
	}
	return in.ProviderPaymentID, nil
}

func (CloudPayments) SendClosingReceipt(ctx context.Context, cfg map[string]any, in ClosingReceiptInput) (string, error) {
	publicID, secret := strCfg(cfg, "public_id"), strCfg(cfg, "api_secret")
	if publicID == "" || secret == "" {
		return "", notConfigured("cloudpayments")
	}
	if strings.TrimSpace(in.SellerINN) == "" {
		return "", fmt.Errorf("cloudpayments: заполните ИНН продавца в разделе «Бухгалтерия»")
	}
	full := in.full()
	sum := roundCents(in.Amount)
	receipt := map[string]any{
		"Items": []map[string]any{{
			"label":    full.item(in.Item),
			"price":    sum,
			"quantity": 1,
			"amount":   sum,
			"vat":      full.cloudPaymentsVAT(),
			"method":   4,
			"object":   4,
		}},
		"email":   in.Receipt.Email,
		"amounts": map[string]any{"advancePayment": sum},
	}
	if code, ok := cloudPaymentsTaxations[full.sno()]; ok {
		receipt["taxationSystem"] = code
	}
	body := map[string]any{
		"Inn":             in.SellerINN,
		"Type":            "Income",
		"CustomerReceipt": receipt,
		"InvoiceId":       in.invoiceRef(),
		"AccountId":       in.UserID,
	}
	var out struct {
		Success bool   `json:"Success"`
		Message string `json:"Message"`
		Model   struct {
			ID string `json:"Id"`
		} `json:"Model"`
	}
	if err := sendJSON(ctx, http.MethodPost, "https://api.cloudpayments.ru/kkt/receipt", body, &out, withBasicAuth(publicID, secret)); err != nil {
		return "", fmt.Errorf("cloudpayments: %w", err)
	}
	if !out.Success {
		return "", fmt.Errorf("cloudpayments: %s", firstNonEmpty(out.Message, "чек не принят"))
	}
	return out.Model.ID, nil
}
