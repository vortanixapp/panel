package payments

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"

	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/checkout/session"
	"github.com/stripe/stripe-go/v81/webhook"
)

type Stripe struct{}

func (Stripe) Code() string { return "stripe" }

func stripeSecret(cfg map[string]any) string {
	return firstNonEmpty(strCfg(cfg, "secret"), strCfg(cfg, "secret_key"))
}

func (Stripe) CreateCheckout(ctx context.Context, cfg map[string]any, in CheckoutInput) (Checkout, error) {
	secret := stripeSecret(cfg)
	if secret == "" {
		return Checkout{}, notConfigured("stripe")
	}
	params := &stripe.CheckoutSessionParams{
		Mode:              stripe.String(string(stripe.CheckoutSessionModePayment)),
		SuccessURL:        stripe.String(in.ReturnURL),
		CancelURL:         stripe.String(in.FailURL),
		ClientReferenceID: stripe.String(in.PaymentID),
		LineItems: []*stripe.CheckoutSessionLineItemParams{{
			Quantity: stripe.Int64(1),
			PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
				Currency: stripe.String(strings.ToLower(in.Currency)),
				ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
					Name: stripe.String(truncate(in.Description, 250)),
				},
				UnitAmount: stripe.Int64(minorUnits(in.Amount, in.Currency)),
			},
		}},
		PaymentIntentData: &stripe.CheckoutSessionPaymentIntentDataParams{
			Metadata: map[string]string{"payment_id": in.PaymentID},
		},
	}
	if in.Email != "" {
		params.CustomerEmail = stripe.String(in.Email)
	}
	params.Context = ctx
	params.AddMetadata("payment_id", in.PaymentID)
	sessions := session.Client{B: stripe.GetBackend(stripe.APIBackend), Key: secret}
	sess, err := sessions.New(params)
	if err != nil {
		return Checkout{}, fmt.Errorf("stripe: %w", err)
	}
	if sess.URL == "" {
		return Checkout{}, fmt.Errorf("stripe: в ответе нет ссылки на оплату")
	}
	return Checkout{RedirectURL: sess.URL, ProviderPaymentID: sess.ID}, nil
}

func (Stripe) HandleNotification(_ context.Context, cfg map[string]any, req *NotifyRequest) (Notification, error) {
	secret := firstNonEmpty(strCfg(cfg, "webhook_secret"), os.Getenv("STRIPE_WEBHOOK_SECRET"))
	if secret == "" {
		return Notification{}, fmt.Errorf("stripe: не задан Webhook Signing Secret")
	}
	event, err := webhook.ConstructEventWithOptions(req.Body, req.Header.Get("Stripe-Signature"), secret,
		webhook.ConstructEventOptions{IgnoreAPIVersionMismatch: true})
	if err != nil {
		return Notification{}, ErrBadSignature
	}
	var obj struct {
		ID                string            `json:"id"`
		PaymentStatus     string            `json:"payment_status"`
		AmountTotal       int64             `json:"amount_total"`
		AmountReceived    int64             `json:"amount_received"`
		Currency          string            `json:"currency"`
		ClientReferenceID string            `json:"client_reference_id"`
		Metadata          map[string]string `json:"metadata"`
	}
	if err := json.Unmarshal(event.Data.Raw, &obj); err != nil {
		return Notification{}, fmt.Errorf("stripe: %w", err)
	}
	n := Notification{
		PaymentID:         firstNonEmpty(obj.Metadata["payment_id"], obj.Metadata["vortanix_payment_id"], obj.ClientReferenceID),
		ProviderPaymentID: obj.ID,
		Currency:          upper(obj.Currency),
	}
	switch string(event.Type) {
	case "checkout.session.completed":
		n.Amount = fromMinorUnits(float64(obj.AmountTotal), obj.Currency)
		if obj.PaymentStatus == "paid" {
			n.Status = NotifyPaid
		}
	case "checkout.session.async_payment_succeeded":
		n.Amount = fromMinorUnits(float64(obj.AmountTotal), obj.Currency)
		n.Status = NotifyPaid
	case "payment_intent.succeeded":
		n.Amount = fromMinorUnits(float64(obj.AmountReceived), obj.Currency)
		n.Status = NotifyPaid
	case "checkout.session.async_payment_failed", "checkout.session.expired":
		n.Status = NotifyFailed
	}
	return n, nil
}

type PayPal struct{}

func (PayPal) Code() string { return "paypal" }

func paypalBaseURL(cfg map[string]any) string {
	if strCfg(cfg, "mode") == "live" {
		return "https://api-m.paypal.com"
	}
	return "https://api-m.sandbox.paypal.com"
}

func paypalAccessToken(ctx context.Context, cfg map[string]any) (string, error) {
	clientID, clientSecret := strCfg(cfg, "client_id"), strCfg(cfg, "client_secret")
	if clientID == "" || clientSecret == "" {
		return "", notConfigured("paypal")
	}
	var out struct {
		AccessToken string `json:"access_token"`
	}
	err := sendForm(ctx, paypalBaseURL(cfg)+"/v1/oauth2/token", url.Values{"grant_type": {"client_credentials"}}, &out,
		withBasicAuth(clientID, clientSecret))
	if err != nil {
		return "", fmt.Errorf("paypal oauth: %w", err)
	}
	if out.AccessToken == "" {
		return "", fmt.Errorf("paypal: пустой токен доступа")
	}
	return out.AccessToken, nil
}

func (PayPal) CreateCheckout(ctx context.Context, cfg map[string]any, in CheckoutInput) (Checkout, error) {
	token, err := paypalAccessToken(ctx, cfg)
	if err != nil {
		return Checkout{}, err
	}
	body := map[string]any{
		"intent": "CAPTURE",
		"purchase_units": []map[string]any{{
			"reference_id": in.PaymentID,
			"custom_id":    in.PaymentID,
			"description":  truncate(in.Description, 127),
			"amount":       map[string]string{"currency_code": upper(in.Currency), "value": money(in.Amount)},
		}},
		"application_context": map[string]any{
			"return_url":          in.ReturnURL,
			"cancel_url":          in.FailURL,
			"user_action":         "PAY_NOW",
			"shipping_preference": "NO_SHIPPING",
		},
	}
	var out struct {
		ID    string `json:"id"`
		Links []struct {
			Href string `json:"href"`
			Rel  string `json:"rel"`
		} `json:"links"`
	}
	if err := sendJSON(ctx, http.MethodPost, paypalBaseURL(cfg)+"/v2/checkout/orders", body, &out, withBearer(token)); err != nil {
		return Checkout{}, fmt.Errorf("paypal: %w", err)
	}
	for _, l := range out.Links {
		if l.Rel == "approve" || l.Rel == "payer-action" {
			return Checkout{RedirectURL: l.Href, ProviderPaymentID: out.ID}, nil
		}
	}
	return Checkout{}, fmt.Errorf("paypal: в ответе нет ссылки на оплату")
}

func capturePayPalOrder(ctx context.Context, cfg map[string]any, orderID string) error {
	token, err := paypalAccessToken(ctx, cfg)
	if err != nil {
		return err
	}
	err = sendJSON(ctx, http.MethodPost, paypalBaseURL(cfg)+"/v2/checkout/orders/"+url.PathEscape(orderID)+"/capture",
		[]byte("{}"), nil, withBearer(token))
	if err != nil && !strings.Contains(err.Error(), "ORDER_ALREADY_CAPTURED") {
		return fmt.Errorf("paypal capture: %w", err)
	}
	return nil
}

func verifyPayPalWebhook(ctx context.Context, cfg map[string]any, headers http.Header, rawBody []byte) (bool, error) {
	webhookID := strCfg(cfg, "webhook_id")
	if webhookID == "" {
		return false, fmt.Errorf("paypal: не задан Webhook ID")
	}
	token, err := paypalAccessToken(ctx, cfg)
	if err != nil {
		return false, err
	}
	payload := struct {
		AuthAlgo         string          `json:"auth_algo"`
		CertURL          string          `json:"cert_url"`
		TransmissionID   string          `json:"transmission_id"`
		TransmissionSig  string          `json:"transmission_sig"`
		TransmissionTime string          `json:"transmission_time"`
		WebhookID        string          `json:"webhook_id"`
		WebhookEvent     json.RawMessage `json:"webhook_event"`
	}{
		AuthAlgo:         headers.Get("Paypal-Auth-Algo"),
		CertURL:          headers.Get("Paypal-Cert-Url"),
		TransmissionID:   headers.Get("Paypal-Transmission-Id"),
		TransmissionSig:  headers.Get("Paypal-Transmission-Sig"),
		TransmissionTime: headers.Get("Paypal-Transmission-Time"),
		WebhookID:        webhookID,
		WebhookEvent:     json.RawMessage(rawBody),
	}
	var out struct {
		VerificationStatus string `json:"verification_status"`
	}
	if err := sendJSON(ctx, http.MethodPost, paypalBaseURL(cfg)+"/v1/notifications/verify-webhook-signature", payload, &out, withBearer(token)); err != nil {
		return false, fmt.Errorf("paypal verify: %w", err)
	}
	return out.VerificationStatus == "SUCCESS", nil
}

func (PayPal) HandleNotification(ctx context.Context, cfg map[string]any, req *NotifyRequest) (Notification, error) {
	type paypalAmount struct {
		Value        string `json:"value"`
		CurrencyCode string `json:"currency_code"`
	}
	var ev struct {
		EventType string `json:"event_type"`
		Resource  struct {
			ID            string       `json:"id"`
			CustomID      string       `json:"custom_id"`
			Amount        paypalAmount `json:"amount"`
			PurchaseUnits []struct {
				CustomID string       `json:"custom_id"`
				Amount   paypalAmount `json:"amount"`
			} `json:"purchase_units"`
		} `json:"resource"`
	}
	if err := req.DecodeJSON(&ev); err != nil {
		return Notification{}, fmt.Errorf("paypal: %w", err)
	}
	ok, err := verifyPayPalWebhook(ctx, cfg, req.Header, req.Body)
	if err != nil {
		return Notification{}, err
	}
	if !ok {
		return Notification{}, ErrBadSignature
	}
	res := ev.Resource
	n := Notification{
		PaymentID:         res.CustomID,
		ProviderPaymentID: res.ID,
		Amount:            parseAmount(res.Amount.Value),
		Currency:          upper(res.Amount.CurrencyCode),
	}
	if len(res.PurchaseUnits) > 0 {
		unit := res.PurchaseUnits[0]
		n.PaymentID = firstNonEmpty(n.PaymentID, unit.CustomID)
		if n.Amount == 0 {
			n.Amount = parseAmount(unit.Amount.Value)
			n.Currency = upper(unit.Amount.CurrencyCode)
		}
	}
	switch ev.EventType {
	case "CHECKOUT.ORDER.APPROVED":
		if err := capturePayPalOrder(ctx, cfg, res.ID); err != nil {
			return n, err
		}
		n.Status = NotifyPaid
	case "PAYMENT.CAPTURE.COMPLETED":
		n.Status = NotifyPaid
	case "PAYMENT.CAPTURE.DENIED", "PAYMENT.CAPTURE.DECLINED":
		n.Status = NotifyFailed
	}
	return n, nil
}

type Mollie struct{}

func (Mollie) Code() string { return "mollie" }

func (Mollie) CreateCheckout(ctx context.Context, cfg map[string]any, in CheckoutInput) (Checkout, error) {
	key := strCfg(cfg, "api_key")
	if key == "" {
		return Checkout{}, notConfigured("mollie")
	}
	body := map[string]any{
		"amount":      map[string]string{"currency": upper(in.Currency), "value": money(in.Amount)},
		"description": truncate(in.Description, 255),
		"redirectUrl": in.ReturnURL,
		"cancelUrl":   in.FailURL,
		"webhookUrl":  in.NotifyURL,
		"metadata":    map[string]string{"payment_id": in.PaymentID},
	}
	var out struct {
		ID    string `json:"id"`
		Links struct {
			Checkout struct {
				Href string `json:"href"`
			} `json:"checkout"`
		} `json:"_links"`
	}
	if err := sendJSON(ctx, http.MethodPost, urlCfg(cfg, "api_url", "https://api.mollie.com")+"/v2/payments", body, &out, withBearer(key)); err != nil {
		return Checkout{}, fmt.Errorf("mollie: %w", err)
	}
	if out.Links.Checkout.Href == "" {
		return Checkout{}, fmt.Errorf("mollie: в ответе нет ссылки на оплату")
	}
	return Checkout{RedirectURL: out.Links.Checkout.Href, ProviderPaymentID: out.ID}, nil
}

func (Mollie) HandleNotification(ctx context.Context, cfg map[string]any, req *NotifyRequest) (Notification, error) {
	id := req.Value("id")
	if id == "" {
		return Notification{}, nil
	}
	var out struct {
		ID     string `json:"id"`
		Status string `json:"status"`
		Amount struct {
			Value    string `json:"value"`
			Currency string `json:"currency"`
		} `json:"amount"`
		Metadata map[string]any `json:"metadata"`
	}
	endpoint := urlCfg(cfg, "api_url", "https://api.mollie.com") + "/v2/payments/" + url.PathEscape(id)
	if err := sendJSON(ctx, http.MethodGet, endpoint, nil, &out, withBearer(strCfg(cfg, "api_key"))); err != nil {
		return Notification{}, fmt.Errorf("mollie: проверка платежа: %w", err)
	}
	n := Notification{
		PaymentID:         metaPaymentID(out.Metadata),
		ProviderPaymentID: out.ID,
		Amount:            parseAmount(out.Amount.Value),
		Currency:          upper(out.Amount.Currency),
	}
	switch out.Status {
	case "paid":
		n.Status = NotifyPaid
	case "failed", "canceled", "expired":
		n.Status = NotifyFailed
	}
	return n, nil
}

type Paystack struct{}

func (Paystack) Code() string { return "paystack" }

func (Paystack) CreateCheckout(ctx context.Context, cfg map[string]any, in CheckoutInput) (Checkout, error) {
	secret := strCfg(cfg, "secret_key")
	if secret == "" {
		return Checkout{}, notConfigured("paystack")
	}
	if in.Email == "" {
		return Checkout{}, fmt.Errorf("paystack: у клиента не указан email, без него Paystack не принимает оплату")
	}
	body := map[string]any{
		"email":        in.Email,
		"amount":       minorUnits(in.Amount, in.Currency),
		"currency":     upper(in.Currency),
		"reference":    in.PaymentID,
		"callback_url": in.ReturnURL,
		"metadata":     map[string]string{"payment_id": in.PaymentID, "cancel_action": in.FailURL},
	}
	var out struct {
		Status  bool   `json:"status"`
		Message string `json:"message"`
		Data    struct {
			URL       string `json:"authorization_url"`
			Reference string `json:"reference"`
		} `json:"data"`
	}
	if err := sendJSON(ctx, http.MethodPost, urlCfg(cfg, "api_url", "https://api.paystack.co")+"/transaction/initialize", body, &out, withBearer(secret)); err != nil {
		return Checkout{}, fmt.Errorf("paystack: %w", err)
	}
	if !out.Status || out.Data.URL == "" {
		return Checkout{}, fmt.Errorf("paystack: %s", firstNonEmpty(out.Message, "платёж не создан"))
	}
	return Checkout{RedirectURL: out.Data.URL, ProviderPaymentID: out.Data.Reference}, nil
}

func (Paystack) HandleNotification(ctx context.Context, cfg map[string]any, req *NotifyRequest) (Notification, error) {
	secret := strCfg(cfg, "secret_key")
	if !secureEqualFold(hmacSHA512Hex([]byte(secret), req.Body), req.Header.Get("x-paystack-signature")) {
		return Notification{}, ErrBadSignature
	}
	var ev struct {
		Event string `json:"event"`
		Data  struct {
			Reference string `json:"reference"`
		} `json:"data"`
	}
	if err := req.DecodeJSON(&ev); err != nil {
		return Notification{}, fmt.Errorf("paystack: %w", err)
	}
	if ev.Event != "charge.success" || ev.Data.Reference == "" {
		return Notification{}, nil
	}
	var out struct {
		Data struct {
			Status    string  `json:"status"`
			Reference string  `json:"reference"`
			Amount    float64 `json:"amount"`
			Currency  string  `json:"currency"`
		} `json:"data"`
	}
	endpoint := urlCfg(cfg, "api_url", "https://api.paystack.co") + "/transaction/verify/" + url.PathEscape(ev.Data.Reference)
	if err := sendJSON(ctx, http.MethodGet, endpoint, nil, &out, withBearer(secret)); err != nil {
		return Notification{}, fmt.Errorf("paystack: проверка платежа: %w", err)
	}
	n := Notification{
		ProviderPaymentID: out.Data.Reference,
		Amount:            fromMinorUnits(out.Data.Amount, out.Data.Currency),
		Currency:          upper(out.Data.Currency),
	}
	orderReference(&n, out.Data.Reference)
	switch out.Data.Status {
	case "success":
		n.Status = NotifyPaid
	case "failed", "abandoned", "reversed":
		n.Status = NotifyFailed
	}
	return n, nil
}

type Flutterwave struct{}

func (Flutterwave) Code() string { return "flutterwave" }

func (Flutterwave) CreateCheckout(ctx context.Context, cfg map[string]any, in CheckoutInput) (Checkout, error) {
	secret := strCfg(cfg, "secret_key")
	if secret == "" {
		return Checkout{}, notConfigured("flutterwave")
	}
	if in.Email == "" {
		return Checkout{}, fmt.Errorf("flutterwave: у клиента не указан email, без него Flutterwave не принимает оплату")
	}
	body := map[string]any{
		"tx_ref":         in.PaymentID,
		"amount":         roundCents(in.Amount),
		"currency":       upper(in.Currency),
		"redirect_url":   in.ReturnURL,
		"customer":       map[string]string{"email": in.Email},
		"customizations": map[string]string{"title": truncate(in.Description, 100)},
		"meta":           map[string]string{"payment_id": in.PaymentID},
	}
	var out struct {
		Status  string `json:"status"`
		Message string `json:"message"`
		Data    struct {
			Link string `json:"link"`
		} `json:"data"`
	}
	if err := sendJSON(ctx, http.MethodPost, urlCfg(cfg, "api_url", "https://api.flutterwave.com")+"/v3/payments", body, &out, withBearer(secret)); err != nil {
		return Checkout{}, fmt.Errorf("flutterwave: %w", err)
	}
	if out.Data.Link == "" {
		return Checkout{}, fmt.Errorf("flutterwave: %s", firstNonEmpty(out.Message, "платёж не создан"))
	}
	return Checkout{RedirectURL: out.Data.Link, ProviderPaymentID: in.PaymentID}, nil
}

func (Flutterwave) HandleNotification(ctx context.Context, cfg map[string]any, req *NotifyRequest) (Notification, error) {
	if !secureEqual(strCfg(cfg, "webhook_secret_hash"), req.Header.Get("verif-hash")) {
		return Notification{}, ErrBadSignature
	}
	var ev struct {
		Data struct {
			ID    any    `json:"id"`
			TxRef string `json:"tx_ref"`
		} `json:"data"`
	}
	if err := req.DecodeJSON(&ev); err != nil {
		return Notification{}, fmt.Errorf("flutterwave: %w", err)
	}
	id := firstNonEmpty(fmt.Sprint(ev.Data.ID))
	if id == "" {
		return Notification{}, nil
	}
	var out struct {
		Data struct {
			Status   string  `json:"status"`
			TxRef    string  `json:"tx_ref"`
			Amount   float64 `json:"amount"`
			Currency string  `json:"currency"`
		} `json:"data"`
	}
	endpoint := urlCfg(cfg, "api_url", "https://api.flutterwave.com") + "/v3/transactions/" + url.PathEscape(id) + "/verify"
	if err := sendJSON(ctx, http.MethodGet, endpoint, nil, &out, withBearer(strCfg(cfg, "secret_key"))); err != nil {
		return Notification{}, fmt.Errorf("flutterwave: проверка платежа: %w", err)
	}
	n := Notification{ProviderPaymentID: id, Amount: out.Data.Amount, Currency: upper(out.Data.Currency)}
	orderReference(&n, out.Data.TxRef)
	switch out.Data.Status {
	case "successful":
		n.Status = NotifyPaid
	case "failed":
		n.Status = NotifyFailed
	}
	return n, nil
}

type Razorpay struct{}

func (Razorpay) Code() string { return "razorpay" }

func (Razorpay) CreateCheckout(ctx context.Context, cfg map[string]any, in CheckoutInput) (Checkout, error) {
	keyID, keySecret := strCfg(cfg, "key_id"), strCfg(cfg, "key_secret")
	if keyID == "" || keySecret == "" {
		return Checkout{}, notConfigured("razorpay")
	}
	body := map[string]any{
		"amount":          minorUnits(in.Amount, in.Currency),
		"currency":        upper(in.Currency),
		"description":     truncate(in.Description, 2048),
		"reference_id":    in.PaymentID,
		"callback_url":    in.ReturnURL,
		"callback_method": "get",
		"notes":           map[string]string{"payment_id": in.PaymentID},
		"notify":          map[string]bool{"sms": false, "email": false},
	}
	if in.Email != "" {
		body["customer"] = map[string]string{"email": in.Email}
	}
	var out struct {
		ID       string `json:"id"`
		ShortURL string `json:"short_url"`
	}
	if err := sendJSON(ctx, http.MethodPost, urlCfg(cfg, "api_url", "https://api.razorpay.com")+"/v1/payment_links", body, &out, withBasicAuth(keyID, keySecret)); err != nil {
		return Checkout{}, fmt.Errorf("razorpay: %w", err)
	}
	if out.ShortURL == "" {
		return Checkout{}, fmt.Errorf("razorpay: в ответе нет ссылки на оплату")
	}
	return Checkout{RedirectURL: out.ShortURL, ProviderPaymentID: out.ID}, nil
}

func (Razorpay) HandleNotification(_ context.Context, cfg map[string]any, req *NotifyRequest) (Notification, error) {
	if !secureEqualFold(hmacSHA256Hex(strCfg(cfg, "webhook_secret"), req.Body), req.Header.Get("X-Razorpay-Signature")) {
		return Notification{}, ErrBadSignature
	}
	var ev struct {
		Event   string `json:"event"`
		Payload struct {
			PaymentLink struct {
				Entity struct {
					ID          string `json:"id"`
					ReferenceID string `json:"reference_id"`
					AmountPaid  any    `json:"amount_paid"`
					Currency    string `json:"currency"`
				} `json:"entity"`
			} `json:"payment_link"`
		} `json:"payload"`
	}
	if err := req.DecodeJSON(&ev); err != nil {
		return Notification{}, fmt.Errorf("razorpay: %w", err)
	}
	e := ev.Payload.PaymentLink.Entity
	n := Notification{
		ProviderPaymentID: e.ID,
		Amount:            fromMinorUnits(numberValue(e.AmountPaid), e.Currency),
		Currency:          upper(e.Currency),
	}
	orderReference(&n, e.ReferenceID)
	switch ev.Event {
	case "payment_link.paid":
		n.Status = NotifyPaid
	case "payment_link.cancelled", "payment_link.expired":
		n.Status = NotifyFailed
	}
	return n, nil
}

type PayFast struct{}

func (PayFast) Code() string { return "payfast" }

func payfastHost(cfg map[string]any) string {
	if boolCfg(cfg, "test") {
		return "https://sandbox.payfast.co.za"
	}
	return "https://www.payfast.co.za"
}

func payfastParamString(fields []FormField, skipEmpty bool) string {
	parts := make([]string, 0, len(fields))
	for _, f := range fields {
		if skipEmpty && f.Value == "" {
			continue
		}
		parts = append(parts, f.Name+"="+phpURLEncode(f.Value))
	}
	return strings.Join(parts, "&")
}

func payfastWithPassphrase(params, passphrase string) string {
	if passphrase == "" {
		return params
	}
	return params + "&passphrase=" + phpURLEncode(passphrase)
}

func (PayFast) CreateCheckout(_ context.Context, cfg map[string]any, in CheckoutInput) (Checkout, error) {
	merchantID, merchantKey := strCfg(cfg, "merchant_id"), strCfg(cfg, "merchant_key")
	if merchantID == "" || merchantKey == "" {
		return Checkout{}, notConfigured("payfast")
	}
	fields := []FormField{
		{Name: "merchant_id", Value: merchantID},
		{Name: "merchant_key", Value: merchantKey},
		{Name: "return_url", Value: in.ReturnURL},
		{Name: "cancel_url", Value: in.FailURL},
		{Name: "notify_url", Value: in.NotifyURL},
		{Name: "email_address", Value: in.Email},
		{Name: "m_payment_id", Value: in.PaymentID},
		{Name: "amount", Value: money(in.Amount)},
		{Name: "item_name", Value: truncate(in.Description, 100)},
	}
	signature := md5Hex(payfastWithPassphrase(payfastParamString(fields, true), strCfg(cfg, "passphrase")))
	form := &Form{Action: payfastHost(cfg) + "/eng/process", Method: http.MethodPost}
	for _, f := range fields {
		if f.Value != "" {
			form.Fields = append(form.Fields, f)
		}
	}
	form.add("signature", signature)
	return Checkout{Form: form}, nil
}

func (PayFast) HandleNotification(ctx context.Context, cfg map[string]any, req *NotifyRequest) (Notification, error) {
	fields := []FormField{}
	signature := ""
	for _, part := range strings.Split(string(req.Body), "&") {
		if part == "" {
			continue
		}
		k, v, _ := strings.Cut(part, "=")
		key, _ := url.QueryUnescape(k)
		val, _ := url.QueryUnescape(v)
		if key == "signature" {
			signature = val
			break
		}
		fields = append(fields, FormField{Name: key, Value: val})
	}
	values := url.Values{}
	for _, f := range fields {
		values.Set(f.Name, f.Value)
	}
	n := Notification{
		PaymentID:         values.Get("m_payment_id"),
		ProviderPaymentID: values.Get("pf_payment_id"),
		Amount:            parseAmount(values.Get("amount_gross")),
		Currency:          "ZAR",
	}
	if n.PaymentID == "" {
		return n, nil
	}
	params := payfastParamString(fields, false)
	if !secureEqualFold(md5Hex(payfastWithPassphrase(params, strCfg(cfg, "passphrase"))), signature) {
		return n, ErrBadSignature
	}
	if values.Get("merchant_id") != strCfg(cfg, "merchant_id") {
		return n, fmt.Errorf("payfast: уведомление для другого магазина")
	}
	if err := payfastValidate(ctx, payfastHost(cfg), params); err != nil {
		return n, err
	}
	switch values.Get("payment_status") {
	case "COMPLETE":
		n.Status = NotifyPaid
	case "CANCELLED":
		n.Status = NotifyFailed
	}
	return n, nil
}

func payfastValidate(ctx context.Context, host, params string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, host+"/eng/query/validate", strings.NewReader(params))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("payfast: проверка уведомления: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	if strings.TrimSpace(string(raw)) != "VALID" {
		return fmt.Errorf("payfast: сервер не подтвердил уведомление (%s)", truncate(strings.TrimSpace(string(raw)), 80))
	}
	return nil
}

type Square struct{}

func (Square) Code() string { return "square" }

const squareAPIVersion = "2024-07-17"

func (Square) CreateCheckout(ctx context.Context, cfg map[string]any, in CheckoutInput) (Checkout, error) {
	token, location := strCfg(cfg, "access_token"), strCfg(cfg, "location_id")
	if token == "" || location == "" {
		return Checkout{}, notConfigured("square")
	}
	body := map[string]any{
		"idempotency_key": in.PaymentID,
		"quick_pay": map[string]any{
			"name":        truncate(in.Description, 255),
			"price_money": map[string]any{"amount": minorUnits(in.Amount, in.Currency), "currency": upper(in.Currency)},
			"location_id": location,
		},
		"checkout_options": map[string]string{"redirect_url": in.ReturnURL},
		"payment_note":     "vortanix:" + in.PaymentID,
	}
	var out struct {
		PaymentLink struct {
			URL     string `json:"url"`
			OrderID string `json:"order_id"`
		} `json:"payment_link"`
	}
	endpoint := urlCfg(cfg, "api_url", "https://connect.squareup.com") + "/v2/online-checkout/payment-links"
	if err := sendJSON(ctx, http.MethodPost, endpoint, body, &out, withBearer(token), withHeader("Square-Version", squareAPIVersion)); err != nil {
		return Checkout{}, fmt.Errorf("square: %w", err)
	}
	if out.PaymentLink.URL == "" {
		return Checkout{}, fmt.Errorf("square: в ответе нет ссылки на оплату")
	}
	return Checkout{RedirectURL: out.PaymentLink.URL, ProviderPaymentID: out.PaymentLink.OrderID}, nil
}

func (Square) HandleNotification(_ context.Context, cfg map[string]any, req *NotifyRequest) (Notification, error) {
	notificationURL := firstNonEmpty(strCfg(cfg, "webhook_notification_url"), req.NotifyURL)
	signed := append([]byte(notificationURL), req.Body...)
	if !secureEqual(hmacSHA256Base64(strCfg(cfg, "webhook_signature_key"), signed), req.Header.Get("x-square-hmacsha256-signature")) {
		return Notification{}, ErrBadSignature
	}
	var ev struct {
		Data struct {
			Object struct {
				Payment struct {
					ID          string `json:"id"`
					OrderID     string `json:"order_id"`
					Status      string `json:"status"`
					Note        string `json:"note"`
					AmountMoney struct {
						Amount   any    `json:"amount"`
						Currency string `json:"currency"`
					} `json:"amount_money"`
				} `json:"payment"`
			} `json:"object"`
		} `json:"data"`
	}
	if err := req.DecodeJSON(&ev); err != nil {
		return Notification{}, fmt.Errorf("square: %w", err)
	}
	p := ev.Data.Object.Payment
	if p.ID == "" {
		return Notification{}, nil
	}
	n := Notification{
		ProviderPaymentID: p.OrderID,
		Amount:            fromMinorUnits(numberValue(p.AmountMoney.Amount), p.AmountMoney.Currency),
		Currency:          upper(p.AmountMoney.Currency),
	}
	if id := strings.TrimPrefix(strings.TrimSpace(p.Note), "vortanix:"); isUUID(id) {
		n.PaymentID = id
	}
	switch p.Status {
	case "COMPLETED":
		n.Status = NotifyPaid
	case "FAILED", "CANCELED":
		n.Status = NotifyFailed
	}
	return n, nil
}

type AuthorizeNet struct{}

func (AuthorizeNet) Code() string { return "authorizenet" }

type anetAuth struct {
	Name           string `json:"name"`
	TransactionKey string `json:"transactionKey"`
}

type anetMessages struct {
	ResultCode string `json:"resultCode"`
	Message    []struct {
		Code string `json:"code"`
		Text string `json:"text"`
	} `json:"message"`
}

func (m anetMessages) err() error {
	if m.ResultCode == "Ok" {
		return nil
	}
	texts := []string{}
	for _, msg := range m.Message {
		texts = append(texts, msg.Code+" "+msg.Text)
	}
	return fmt.Errorf("authorize.net: %s", firstNonEmpty(strings.Join(texts, "; "), "запрос отклонён"))
}

func (AuthorizeNet) CreateCheckout(ctx context.Context, cfg map[string]any, in CheckoutInput) (Checkout, error) {
	login, key := strCfg(cfg, "login_id"), strCfg(cfg, "transaction_key")
	if login == "" || key == "" {
		return Checkout{}, notConfigured("authorizenet")
	}
	returnOptions, _ := json.Marshal(map[string]any{
		"showReceipt":   false,
		"url":           in.ReturnURL,
		"urlText":       "Continue",
		"cancelUrl":     in.FailURL,
		"cancelUrlText": "Cancel",
	})
	type setting struct {
		SettingName  string `json:"settingName"`
		SettingValue string `json:"settingValue"`
	}
	type order struct {
		InvoiceNumber string `json:"invoiceNumber"`
		Description   string `json:"description"`
	}
	type customer struct {
		Email string `json:"email"`
	}
	type transactionRequest struct {
		TransactionType string    `json:"transactionType"`
		Amount          string    `json:"amount"`
		Order           order     `json:"order"`
		Customer        *customer `json:"customer,omitempty"`
	}
	type hostedSettings struct {
		Setting []setting `json:"setting"`
	}
	type hostedPageRequest struct {
		MerchantAuthentication anetAuth           `json:"merchantAuthentication"`
		RefID                  string             `json:"refId"`
		TransactionRequest     transactionRequest `json:"transactionRequest"`
		HostedPaymentSettings  hostedSettings     `json:"hostedPaymentSettings"`
	}
	request := hostedPageRequest{
		MerchantAuthentication: anetAuth{Name: login, TransactionKey: key},
		RefID:                  invoiceRef(in),
		TransactionRequest: transactionRequest{
			TransactionType: "authCaptureTransaction",
			Amount:          money(in.Amount),
			Order:           order{InvoiceNumber: invoiceRef(in), Description: truncate(in.Description, 255)},
		},
		HostedPaymentSettings: hostedSettings{Setting: []setting{
			{SettingName: "hostedPaymentReturnOptions", SettingValue: string(returnOptions)},
			{SettingName: "hostedPaymentButtonOptions", SettingValue: `{"text":"Pay"}`},
		}},
	}
	if in.Email != "" {
		request.TransactionRequest.Customer = &customer{Email: in.Email}
	}
	var out struct {
		Token    string       `json:"token"`
		Messages anetMessages `json:"messages"`
	}
	endpoint := cfgOr(cfg, "api_url", "https://api2.authorize.net/xml/v1/request.api")
	if err := sendJSON(ctx, http.MethodPost, endpoint, map[string]any{"getHostedPaymentPageRequest": request}, &out); err != nil {
		return Checkout{}, fmt.Errorf("authorize.net: %w", err)
	}
	if err := out.Messages.err(); err != nil {
		return Checkout{}, err
	}
	form := formOf(cfgOr(cfg, "hosted_url", "https://accept.authorize.net/payment/payment"), "token", out.Token)
	return Checkout{Form: form, ProviderPaymentID: invoiceRef(in)}, nil
}

func (AuthorizeNet) HandleNotification(ctx context.Context, cfg map[string]any, req *NotifyRequest) (Notification, error) {
	given := strings.TrimPrefix(strings.TrimSpace(req.Header.Get("X-ANET-Signature")), "sha512=")
	key := strCfg(cfg, "signature_key")
	valid := secureEqualFold(hmacSHA512Hex([]byte(key), req.Body), given)
	if !valid {
		if raw, err := hex.DecodeString(key); err == nil {
			valid = secureEqualFold(hmacSHA512Hex(raw, req.Body), given)
		}
	}
	if !valid {
		return Notification{}, ErrBadSignature
	}
	var ev struct {
		EventType string `json:"eventType"`
		Payload   struct {
			ID         string `json:"id"`
			EntityName string `json:"entityName"`
		} `json:"payload"`
	}
	if err := req.DecodeJSON(&ev); err != nil {
		return Notification{}, fmt.Errorf("authorize.net: %w", err)
	}
	if ev.Payload.ID == "" || ev.Payload.EntityName != "transaction" {
		return Notification{}, nil
	}
	type detailsRequest struct {
		MerchantAuthentication anetAuth `json:"merchantAuthentication"`
		TransID                string   `json:"transId"`
	}
	var out struct {
		Transaction struct {
			TransID      string  `json:"transId"`
			ResponseCode any     `json:"responseCode"`
			AuthAmount   float64 `json:"authAmount"`
			Order        struct {
				InvoiceNumber string `json:"invoiceNumber"`
			} `json:"order"`
		} `json:"transaction"`
		Messages anetMessages `json:"messages"`
	}
	payload := map[string]any{"getTransactionDetailsRequest": detailsRequest{
		MerchantAuthentication: anetAuth{Name: strCfg(cfg, "login_id"), TransactionKey: strCfg(cfg, "transaction_key")},
		TransID:                ev.Payload.ID,
	}}
	endpoint := cfgOr(cfg, "api_url", "https://api2.authorize.net/xml/v1/request.api")
	if err := sendJSON(ctx, http.MethodPost, endpoint, payload, &out); err != nil {
		return Notification{}, fmt.Errorf("authorize.net: проверка платежа: %w", err)
	}
	if err := out.Messages.err(); err != nil {
		return Notification{}, err
	}
	tx := out.Transaction
	n := Notification{
		InvoiceNo:         parseInvoiceNo(tx.Order.InvoiceNumber),
		ProviderPaymentID: tx.TransID,
		Amount:            tx.AuthAmount,
		Currency:          upper(cfgOr(cfg, "currency", "USD")),
	}
	switch numberValue(tx.ResponseCode) {
	case 1:
		if strings.Contains(ev.EventType, "capture") {
			n.Status = NotifyPaid
		}
	case 2, 3:
		n.Status = NotifyFailed
	}
	return n, nil
}

type Paddle struct{}

func (Paddle) Code() string { return "paddle" }

func paddleVendorsHost(cfg map[string]any) string {
	if boolCfg(cfg, "sandbox") {
		return "https://sandbox-vendors.paddle.com"
	}
	return "https://vendors.paddle.com"
}

func (Paddle) CreateCheckout(ctx context.Context, cfg map[string]any, in CheckoutInput) (Checkout, error) {
	vendorID, auth := strCfg(cfg, "vendor_id"), strCfg(cfg, "vendor_auth_code")
	if vendorID == "" || auth == "" {
		return Checkout{}, notConfigured("paddle")
	}
	passthrough, _ := json.Marshal(map[string]string{"payment_id": in.PaymentID})
	form := url.Values{}
	form.Set("vendor_id", vendorID)
	form.Set("vendor_auth_code", auth)
	form.Set("title", truncate(in.Description, 200))
	form.Set("webhook_url", in.NotifyURL)
	form.Set("prices[0]", upper(in.Currency)+":"+money(in.Amount))
	form.Set("quantity_variable", "0")
	form.Set("passthrough", string(passthrough))
	form.Set("return_url", in.ReturnURL)
	if in.Email != "" {
		form.Set("customer_email", in.Email)
	}
	var out struct {
		Success  bool `json:"success"`
		Response struct {
			URL string `json:"url"`
		} `json:"response"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := sendForm(ctx, paddleVendorsHost(cfg)+"/api/2.0/product/generate_pay_link", form, &out); err != nil {
		return Checkout{}, fmt.Errorf("paddle: %w", err)
	}
	if !out.Success || out.Response.URL == "" {
		return Checkout{}, fmt.Errorf("paddle: %s", firstNonEmpty(out.Error.Message, "ссылка на оплату не создана"))
	}
	return Checkout{RedirectURL: out.Response.URL}, nil
}

func phpSerializeStrings(values map[string]string) string {
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	fmt.Fprintf(&b, "a:%d:{", len(keys))
	for _, k := range keys {
		v := values[k]
		fmt.Fprintf(&b, "s:%d:\"%s\";s:%d:\"%s\";", len(k), k, len(v), v)
	}
	b.WriteString("}")
	return b.String()
}

func paddlePublicKey(raw string) (*rsa.PublicKey, error) {
	body := strings.ReplaceAll(raw, `\n`, "\n")
	body = strings.ReplaceAll(body, "-----BEGIN PUBLIC KEY-----", "")
	body = strings.ReplaceAll(body, "-----END PUBLIC KEY-----", "")
	der, err := base64.StdEncoding.DecodeString(strings.Join(strings.Fields(body), ""))
	if err != nil {
		return nil, fmt.Errorf("paddle: публичный ключ не читается: %w", err)
	}
	parsed, err := x509.ParsePKIXPublicKey(der)
	if err != nil {
		return nil, fmt.Errorf("paddle: публичный ключ не читается: %w", err)
	}
	key, ok := parsed.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("paddle: ключ не RSA")
	}
	return key, nil
}

func (Paddle) HandleNotification(_ context.Context, cfg map[string]any, req *NotifyRequest) (Notification, error) {
	signature, err := base64.StdEncoding.DecodeString(req.Form.Get("p_signature"))
	if err != nil || len(signature) == 0 {
		return Notification{}, ErrBadSignature
	}
	values := map[string]string{}
	for k := range req.Form {
		if k != "p_signature" {
			values[k] = req.Form.Get(k)
		}
	}
	key, err := paddlePublicKey(strCfg(cfg, "public_key"))
	if err != nil {
		return Notification{}, err
	}
	digest := sha1.Sum([]byte(phpSerializeStrings(values)))
	if rsa.VerifyPKCS1v15(key, crypto.SHA1, digest[:], signature) != nil {
		return Notification{}, ErrBadSignature
	}
	var passthrough map[string]any
	_ = json.Unmarshal([]byte(values["passthrough"]), &passthrough)
	n := Notification{PaymentID: metaPaymentID(passthrough)}
	switch values["alert_name"] {
	case "":
		n.ProviderPaymentID = values["p_order_id"]
		n.Amount = parseAmount(values["p_sale_gross"])
		n.Currency = upper(values["p_currency"])
		if n.ProviderPaymentID != "" {
			n.Status = NotifyPaid
		}
	case "payment_succeeded":
		n.ProviderPaymentID = values["order_id"]
		n.Amount = parseAmount(values["sale_gross"])
		n.Currency = upper(values["currency"])
		n.Status = NotifyPaid
	}
	return n, nil
}
