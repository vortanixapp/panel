package payments

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type NowPayments struct{}

func (NowPayments) Code() string { return "nowpayments" }

func (NowPayments) CreateCheckout(ctx context.Context, cfg map[string]any, in CheckoutInput) (Checkout, error) {
	apiKey := strCfg(cfg, "api_key")
	if apiKey == "" || strCfg(cfg, "ipn_secret") == "" {
		return Checkout{}, notConfigured("nowpayments")
	}
	body := map[string]any{
		"price_amount":      roundCents(in.Amount),
		"price_currency":    strings.ToLower(in.Currency),
		"order_id":          in.PaymentID,
		"order_description": truncate(in.Description, 255),
		"ipn_callback_url":  in.NotifyURL,
		"success_url":       in.ReturnURL,
		"cancel_url":        in.FailURL,
	}
	if pay := strCfg(cfg, "pay_currency"); pay != "" {
		body["pay_currency"] = strings.ToLower(pay)
	}
	var out struct {
		ID         json.RawMessage `json:"id"`
		InvoiceURL string          `json:"invoice_url"`
	}
	endpoint := urlCfg(cfg, "api_url", "https://api.nowpayments.io") + "/v1/invoice"
	if err := sendJSON(ctx, http.MethodPost, endpoint, body, &out, withHeader("x-api-key", apiKey)); err != nil {
		return Checkout{}, fmt.Errorf("nowpayments: %w", err)
	}
	if out.InvoiceURL == "" {
		return Checkout{}, fmt.Errorf("nowpayments: в ответе нет ссылки на оплату")
	}
	return Checkout{RedirectURL: out.InvoiceURL, ProviderPaymentID: strings.Trim(strings.TrimSpace(string(out.ID)), `"`)}, nil
}

func verifyNowPaymentsSign(cfg map[string]any, rawBody []byte, sig string) bool {
	secret := strCfg(cfg, "ipn_secret")
	if secret == "" || strings.TrimSpace(sig) == "" {
		return false
	}
	dec := json.NewDecoder(bytes.NewReader(rawBody))
	dec.UseNumber()
	var data map[string]any
	if err := dec.Decode(&data); err != nil {
		return false
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(data); err != nil {
		return false
	}
	return secureEqualFold(hmacSHA512Hex([]byte(secret), bytes.TrimRight(buf.Bytes(), "\n")), sig)
}

func (NowPayments) HandleNotification(_ context.Context, cfg map[string]any, req *NotifyRequest) (Notification, error) {
	if !verifyNowPaymentsSign(cfg, req.Body, req.Header.Get("x-nowpayments-sig")) {
		return Notification{}, ErrBadSignature
	}
	var ev struct {
		PaymentStatus string `json:"payment_status"`
		OrderID       string `json:"order_id"`
		PaymentID     any    `json:"payment_id"`
		PriceAmount   any    `json:"price_amount"`
		PriceCurrency string `json:"price_currency"`
	}
	if err := req.DecodeJSON(&ev); err != nil {
		return Notification{}, fmt.Errorf("nowpayments: %w", err)
	}
	n := Notification{
		ProviderPaymentID: firstNonEmpty(fmt.Sprint(ev.PaymentID)),
		Amount:            numberValue(ev.PriceAmount),
		Currency:          upper(ev.PriceCurrency),
	}
	orderReference(&n, ev.OrderID)
	switch ev.PaymentStatus {
	case "finished", "confirmed":
		n.Status = NotifyPaid
	case "failed", "expired", "refunded":
		n.Status = NotifyFailed
	}
	return n, nil
}

type CryptoCloud struct{}

func (CryptoCloud) Code() string { return "cryptocloud" }

func cryptoCloudBase(cfg map[string]any) string {
	base := urlCfg(cfg, "api_url", "https://api.trybit.com")
	if strings.Contains(base, "cryptocloud.plus") {
		return "https://api.trybit.com"
	}
	return base
}

func (CryptoCloud) CreateCheckout(ctx context.Context, cfg map[string]any, in CheckoutInput) (Checkout, error) {
	shopID, key := strCfg(cfg, "shop_id"), strCfg(cfg, "api_key")
	if shopID == "" || key == "" {
		return Checkout{}, notConfigured("cryptocloud")
	}
	body := map[string]any{
		"shop_id":  shopID,
		"amount":   roundCents(in.Amount),
		"currency": upper(in.Currency),
		"order_id": in.PaymentID,
	}
	if in.Email != "" {
		body["email"] = in.Email
	}
	var out struct {
		Status string `json:"status"`
		Result struct {
			UUID string `json:"uuid"`
			Link string `json:"link"`
		} `json:"result"`
	}
	if err := sendJSON(ctx, http.MethodPost, cryptoCloudBase(cfg)+"/v2/invoice/create", body, &out, withHeader("Authorization", "Token "+key)); err != nil {
		return Checkout{}, fmt.Errorf("cryptocloud: %w", err)
	}
	if out.Result.Link == "" {
		return Checkout{}, fmt.Errorf("cryptocloud: в ответе нет ссылки на оплату")
	}
	return Checkout{RedirectURL: out.Result.Link, ProviderPaymentID: out.Result.UUID}, nil
}

func (CryptoCloud) HandleNotification(ctx context.Context, cfg map[string]any, req *NotifyRequest) (Notification, error) {
	invoiceID := req.Value("invoice_id")
	if req.IsJSON() {
		var ev struct {
			InvoiceID string `json:"invoice_id"`
		}
		if err := req.DecodeJSON(&ev); err != nil {
			return Notification{}, fmt.Errorf("cryptocloud: %w", err)
		}
		invoiceID = ev.InvoiceID
	}
	if invoiceID == "" {
		return Notification{}, nil
	}
	uuid := invoiceID
	if !strings.HasPrefix(upper(uuid), "INV-") {
		uuid = "INV-" + uuid
	}
	var out struct {
		Result []struct {
			UUID    string `json:"uuid"`
			Status  string `json:"status"`
			OrderID any    `json:"order_id"`
		} `json:"result"`
	}
	err := sendJSON(ctx, http.MethodPost, cryptoCloudBase(cfg)+"/v2/invoice/merchant/info", map[string]any{"uuids": []string{uuid}}, &out,
		withHeader("Authorization", "Token "+strCfg(cfg, "api_key")))
	if err != nil {
		return Notification{}, fmt.Errorf("cryptocloud: проверка счёта: %w", err)
	}
	if len(out.Result) == 0 {
		return Notification{}, fmt.Errorf("cryptocloud: счёт %s не найден", uuid)
	}
	invoice := out.Result[0]
	n := Notification{ProviderPaymentID: invoice.UUID}
	orderReference(&n, fmt.Sprint(invoice.OrderID))
	if n.PaymentID == "" && n.InvoiceNo == 0 {
		return n, fmt.Errorf("cryptocloud: у счёта %s нет номера заказа", uuid)
	}
	switch invoice.Status {
	case "paid", "overpaid":
		n.Status = NotifyPaid
	case "canceled":
		n.Status = NotifyFailed
	}
	return n, nil
}

type Coinbase struct{}

func (Coinbase) Code() string { return "coinbase" }

func (Coinbase) CreateCheckout(ctx context.Context, cfg map[string]any, in CheckoutInput) (Checkout, error) {
	key := strCfg(cfg, "api_key")
	if key == "" || strCfg(cfg, "webhook_secret") == "" {
		return Checkout{}, notConfigured("coinbase")
	}
	body := map[string]any{
		"name":         truncate(in.Description, 100),
		"description":  truncate(in.Description, 200),
		"pricing_type": "fixed_price",
		"local_price":  map[string]string{"amount": money(in.Amount), "currency": upper(in.Currency)},
		"metadata":     map[string]string{"payment_id": in.PaymentID},
		"redirect_url": in.ReturnURL,
		"cancel_url":   in.FailURL,
	}
	var out struct {
		Data struct {
			Code      string `json:"code"`
			HostedURL string `json:"hosted_url"`
		} `json:"data"`
	}
	err := sendJSON(ctx, http.MethodPost, urlCfg(cfg, "api_url", "https://api.commerce.coinbase.com")+"/charges", body, &out,
		withHeader("X-CC-Api-Key", key), withHeader("X-CC-Version", cfgOr(cfg, "api_version", "2018-03-22")))
	if err != nil {
		return Checkout{}, fmt.Errorf("coinbase: %w", err)
	}
	if out.Data.HostedURL == "" {
		return Checkout{}, fmt.Errorf("coinbase: в ответе нет ссылки на оплату")
	}
	return Checkout{RedirectURL: out.Data.HostedURL, ProviderPaymentID: out.Data.Code}, nil
}

func (Coinbase) HandleNotification(_ context.Context, cfg map[string]any, req *NotifyRequest) (Notification, error) {
	if !secureEqualFold(hmacSHA256Hex(strCfg(cfg, "webhook_secret"), req.Body), req.Header.Get("X-CC-Webhook-Signature")) {
		return Notification{}, ErrBadSignature
	}
	var ev struct {
		Event struct {
			Type string `json:"type"`
			Data struct {
				Code     string         `json:"code"`
				Metadata map[string]any `json:"metadata"`
				Pricing  struct {
					Local struct {
						Amount   string `json:"amount"`
						Currency string `json:"currency"`
					} `json:"local"`
				} `json:"pricing"`
			} `json:"data"`
		} `json:"event"`
	}
	if err := req.DecodeJSON(&ev); err != nil {
		return Notification{}, fmt.Errorf("coinbase: %w", err)
	}
	d := ev.Event.Data
	n := Notification{
		PaymentID:         metaPaymentID(d.Metadata),
		ProviderPaymentID: d.Code,
		Amount:            parseAmount(d.Pricing.Local.Amount),
		Currency:          upper(d.Pricing.Local.Currency),
	}
	switch ev.Event.Type {
	case "charge:confirmed", "charge:resolved":
		n.Status = NotifyPaid
	case "charge:failed":
		n.Status = NotifyFailed
	}
	return n, nil
}

type CryptoCom struct{}

func (CryptoCom) Code() string { return "cryptocom" }

func (CryptoCom) CreateCheckout(ctx context.Context, cfg map[string]any, in CheckoutInput) (Checkout, error) {
	key := strCfg(cfg, "api_key")
	if key == "" || strCfg(cfg, "secret_key") == "" {
		return Checkout{}, notConfigured("cryptocom")
	}
	body := map[string]any{
		"amount":      minorUnits(in.Amount, in.Currency),
		"currency":    upper(in.Currency),
		"description": truncate(in.Description, 255),
		"order_id":    in.PaymentID,
		"metadata":    map[string]string{"payment_id": in.PaymentID},
		"return_url":  in.ReturnURL,
		"cancel_url":  in.FailURL,
	}
	var out struct {
		ID         string `json:"id"`
		PaymentURL string `json:"payment_url"`
	}
	if err := sendJSON(ctx, http.MethodPost, urlCfg(cfg, "api_url", "https://pay.crypto.com")+"/api/payments", body, &out, withBasicAuth(key, "")); err != nil {
		return Checkout{}, fmt.Errorf("crypto.com: %w", err)
	}
	if out.PaymentURL == "" {
		return Checkout{}, fmt.Errorf("crypto.com: в ответе нет ссылки на оплату")
	}
	return Checkout{RedirectURL: out.PaymentURL, ProviderPaymentID: out.ID}, nil
}

func (CryptoCom) HandleNotification(_ context.Context, cfg map[string]any, req *NotifyRequest) (Notification, error) {
	timestamp := ""
	signatures := []string{}
	for _, part := range strings.Split(req.Header.Get("Pay-Signature"), ",") {
		k, v, _ := strings.Cut(strings.TrimSpace(part), "=")
		switch k {
		case "t":
			timestamp = v
		case "v1":
			signatures = append(signatures, v)
		}
	}
	expected := hmacSHA256Hex(strCfg(cfg, "secret_key"), []byte(timestamp+"."+string(req.Body)))
	valid := false
	for _, s := range signatures {
		if timestamp != "" && secureEqualFold(expected, s) {
			valid = true
		}
	}
	if !valid {
		return Notification{}, ErrBadSignature
	}
	var ev struct {
		Type string `json:"type"`
		Data struct {
			Object struct {
				ID       string         `json:"id"`
				Amount   any            `json:"amount"`
				Currency string         `json:"currency"`
				Status   string         `json:"status"`
				OrderID  string         `json:"order_id"`
				Metadata map[string]any `json:"metadata"`
			} `json:"object"`
		} `json:"data"`
	}
	if err := req.DecodeJSON(&ev); err != nil {
		return Notification{}, fmt.Errorf("crypto.com: %w", err)
	}
	o := ev.Data.Object
	n := Notification{
		ProviderPaymentID: o.ID,
		Amount:            fromMinorUnits(numberValue(o.Amount), o.Currency),
		Currency:          upper(o.Currency),
	}
	orderReference(&n, firstNonEmpty(metaPaymentID(o.Metadata), o.OrderID))
	switch {
	case ev.Type == "payment.captured" || o.Status == "succeeded":
		n.Status = NotifyPaid
	case o.Status == "failed" || o.Status == "cancelled":
		n.Status = NotifyFailed
	}
	return n, nil
}

type PerfectMoney struct{}

func (PerfectMoney) Code() string { return "perfectmoney" }

func perfectMoneyAltHash(cfg map[string]any) string {
	v := strCfg(cfg, "alt_passphrase_hash")
	if len(v) == 32 && isHexString(v) {
		return upper(v)
	}
	return upper(md5Hex(v))
}

func (PerfectMoney) CreateCheckout(_ context.Context, cfg map[string]any, in CheckoutInput) (Checkout, error) {
	account := upper(strCfg(cfg, "payee_account"))
	if account == "" || strCfg(cfg, "alt_passphrase_hash") == "" {
		return Checkout{}, notConfigured("perfectmoney")
	}
	form := formOf(cfgOr(cfg, "process_url", "https://perfectmoney.com/api/step1.asp"),
		"PAYEE_ACCOUNT", account,
		"PAYEE_NAME", cfgOr(cfg, "payee_name", "Top-up"),
		"PAYMENT_ID", invoiceRef(in),
		"PAYMENT_AMOUNT", money(in.Amount),
		"PAYMENT_UNITS", upper(in.Currency),
		"STATUS_URL", in.NotifyURL,
		"PAYMENT_URL", in.ReturnURL,
		"PAYMENT_URL_METHOD", "GET",
		"NOPAYMENT_URL", in.FailURL,
		"NOPAYMENT_URL_METHOD", "GET",
		"SUGGESTED_MEMO", truncate(in.Description, 100),
		"BAGGAGE_FIELDS", "",
	)
	return Checkout{Form: form, ProviderPaymentID: invoiceRef(in)}, nil
}

func (PerfectMoney) HandleNotification(_ context.Context, cfg map[string]any, req *NotifyRequest) (Notification, error) {
	get := req.Value
	n := Notification{
		InvoiceNo:         parseInvoiceNo(get("PAYMENT_ID")),
		ProviderPaymentID: get("PAYMENT_BATCH_NUM"),
		Amount:            parseAmount(get("PAYMENT_AMOUNT")),
		Currency:          upper(get("PAYMENT_UNITS")),
	}
	if n.InvoiceNo == 0 {
		return n, nil
	}
	if !strings.EqualFold(get("PAYEE_ACCOUNT"), strCfg(cfg, "payee_account")) {
		return n, fmt.Errorf("perfectmoney: оплата на другой счёт")
	}
	data := strings.Join([]string{
		get("PAYMENT_ID"), get("PAYEE_ACCOUNT"), get("PAYMENT_AMOUNT"), get("PAYMENT_UNITS"),
		get("PAYMENT_BATCH_NUM"), get("PAYER_ACCOUNT"), perfectMoneyAltHash(cfg), get("TIMESTAMPGMT"),
	}, ":")
	if !secureEqualFold(md5Hex(data), get("V2_HASH")) {
		return n, ErrBadSignature
	}
	n.Status = NotifyPaid
	return n, nil
}

type AdvCash struct{}

func (AdvCash) Code() string { return "advcash" }

func (AdvCash) CreateCheckout(_ context.Context, cfg map[string]any, in CheckoutInput) (Checkout, error) {
	email, sci, secret := strCfg(cfg, "account_email"), strCfg(cfg, "sci_name"), strCfg(cfg, "secret_key")
	if email == "" || sci == "" || secret == "" {
		return Checkout{}, notConfigured("advcash")
	}
	amount := money(in.Amount)
	currency := upper(in.Currency)
	order := invoiceRef(in)
	form := formOf(cfgOr(cfg, "process_url", "https://wallet.advcash.com/sci/"),
		"ac_account_email", email,
		"ac_sci_name", sci,
		"ac_amount", amount,
		"ac_currency", currency,
		"ac_order_id", order,
		"ac_sign", sha256Hex(strings.Join([]string{email, sci, amount, currency, secret, order}, ":")),
		"ac_comments", truncate(in.Description, 200),
		"ac_success_url", in.ReturnURL,
		"ac_success_url_method", "GET",
		"ac_fail_url", in.FailURL,
		"ac_fail_url_method", "GET",
		"ac_status_url", in.NotifyURL,
		"ac_status_url_method", "POST",
	)
	return Checkout{Form: form, ProviderPaymentID: order}, nil
}

func (AdvCash) HandleNotification(_ context.Context, cfg map[string]any, req *NotifyRequest) (Notification, error) {
	get := req.Value
	n := Notification{
		InvoiceNo:         parseInvoiceNo(get("ac_order_id")),
		ProviderPaymentID: get("ac_transfer"),
		Amount:            parseAmount(get("ac_amount")),
		Currency:          upper(get("ac_merchant_currency")),
	}
	if get("ac_transfer") == "" {
		return n, nil
	}
	data := strings.Join([]string{
		get("ac_transfer"), get("ac_start_date"), get("ac_sci_name"), get("ac_src_wallet"),
		get("ac_dest_wallet"), get("ac_order_id"), get("ac_amount"), get("ac_merchant_currency"),
		strCfg(cfg, "secret_key"),
	}, ":")
	if !secureEqualFold(sha256Hex(data), get("ac_hash")) {
		return n, ErrBadSignature
	}
	if !strings.EqualFold(get("ac_sci_name"), strCfg(cfg, "sci_name")) {
		return n, fmt.Errorf("advcash: уведомление для другого SCI")
	}
	switch upper(get("ac_transaction_status")) {
	case "COMPLETED":
		n.Status = NotifyPaid
	case "CANCELED", "CANCELLED":
		n.Status = NotifyFailed
	}
	return n, nil
}
