package payments

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/vortanixapp/panel/pkg/yookassa"
)

type YooKassa struct{}

func (YooKassa) Code() string { return "yookassa" }

func (YooKassa) CreateCheckout(ctx context.Context, cfg map[string]any, in CheckoutInput) (Checkout, error) {
	shopID, secret := strCfg(cfg, "shop_id"), strCfg(cfg, "secret_key")
	if shopID == "" || secret == "" {
		return Checkout{}, notConfigured("yookassa")
	}
	amount := map[string]string{"value": money(in.Amount), "currency": upper(in.Currency)}
	body := map[string]any{
		"amount":       amount,
		"confirmation": map[string]string{"type": "redirect", "return_url": in.ReturnURL},
		"capture":      true,
		"description":  truncate(in.Description, 128),
		"metadata":     map[string]string{"payment_id": in.PaymentID},
	}
	if boolCfg(cfg, "receipt") && in.Email != "" {
		body["receipt"] = yookassaReceipt(cfg, in.Email, in.Receipt, in.Description, amount)
	}
	var out struct {
		ID           string `json:"id"`
		Confirmation struct {
			URL string `json:"confirmation_url"`
		} `json:"confirmation"`
	}
	err := sendJSON(ctx, http.MethodPost, "https://api.yookassa.ru/v3/payments", body, &out,
		withBasicAuth(shopID, secret), withHeader("Idempotence-Key", in.PaymentID))
	if err != nil {
		return Checkout{}, fmt.Errorf("yookassa: %w", err)
	}
	if out.Confirmation.URL == "" {
		return Checkout{}, fmt.Errorf("yookassa: в ответе нет ссылки на оплату")
	}
	return Checkout{RedirectURL: out.Confirmation.URL, ProviderPaymentID: out.ID}, nil
}

func yookassaReceipt(cfg map[string]any, email string, rc *Receipt, description string, amount map[string]string) map[string]any {
	fallback, _ := strconv.Atoi(strCfg(cfg, "vat_code"))
	receipt := map[string]any{
		"customer": map[string]string{"email": email},
		"items": []map[string]any{{
			"description":     rc.item(description),
			"quantity":        "1.00",
			"amount":          amount,
			"vat_code":        rc.yookassaVAT(fallback),
			"payment_mode":    rc.mode(),
			"payment_subject": rc.subject(),
		}},
	}
	if code, ok := yookassaTaxSystems[rc.sno()]; ok {
		receipt["tax_system_code"] = code
	}
	return receipt
}

func (YooKassa) HandleNotification(ctx context.Context, cfg map[string]any, req *NotifyRequest) (Notification, error) {
	var ev struct {
		Event  string `json:"event"`
		Object struct {
			ID string `json:"id"`
		} `json:"object"`
	}
	if err := req.DecodeJSON(&ev); err != nil {
		return Notification{}, fmt.Errorf("yookassa: %w", err)
	}
	if ev.Object.ID == "" || !strings.HasPrefix(ev.Event, "payment.") {
		return Notification{}, nil
	}
	remote, err := yookassa.New(strCfg(cfg, "shop_id"), strCfg(cfg, "secret_key")).GetPayment(ctx, ev.Object.ID)
	if err != nil {
		return Notification{}, err
	}
	n := Notification{
		PaymentID:         firstNonEmpty(remote.Metadata["payment_id"], remote.Metadata["order_id"]),
		ProviderPaymentID: remote.ID,
		Amount:            remote.Amount,
		Currency:          upper(remote.Currency),
	}
	switch {
	case remote.Succeeded():
		n.Status = NotifyPaid
	case remote.Canceled():
		n.Status = NotifyFailed
	}
	return n, nil
}

type YooMoney struct{}

func (YooMoney) Code() string { return "yoomoney" }

func (YooMoney) CreateCheckout(_ context.Context, cfg map[string]any, in CheckoutInput) (Checkout, error) {
	wallet := strCfg(cfg, "wallet")
	if wallet == "" || strCfg(cfg, "notification_secret") == "" {
		return Checkout{}, notConfigured("yoomoney")
	}
	paymentType := strCfg(cfg, "payment_type")
	if paymentType != "PC" {
		paymentType = "AC"
	}
	form := formOf("https://yoomoney.ru/quickpay/confirm",
		"receiver", wallet,
		"quickpay-form", "button",
		"paymentType", paymentType,
		"sum", money(in.Amount),
		"label", in.PaymentID,
		"successURL", in.ReturnURL,
	)
	return Checkout{Form: form}, nil
}

func (YooMoney) HandleNotification(_ context.Context, cfg map[string]any, req *NotifyRequest) (Notification, error) {
	if req.Method != http.MethodPost || req.Value("notification_type") == "" {
		return Notification{}, nil
	}
	label := req.Value("label")
	operation := req.Value("operation_id")
	n := Notification{PaymentID: label, ProviderPaymentID: operation, Currency: "RUB"}
	if !yoomoneySigned(req, strCfg(cfg, "notification_secret")) {
		return n, ErrBadSignature
	}
	if req.Value("test_notification") == "true" {
		log.Printf("yoomoney: тестовое уведомление принято, подпись сходится")
		return Notification{}, nil
	}
	if label == "" {
		log.Printf("yoomoney: перевод %s пришёл без метки платежа и не зачислен", operation)
		return Notification{}, nil
	}
	if req.Value("codepro") == "true" || req.Value("unaccepted") == "true" {
		log.Printf("yoomoney: перевод %s по платежу %s заморожен в ЮMoney и не зачислен", operation, label)
		return n, nil
	}
	n.Amount = parseAmount(req.Value("withdraw_amount"))
	if n.Amount == 0 {
		n.Amount = parseAmount(req.Value("amount"))
	}
	n.Status = NotifyPaid
	return n, nil
}

func yoomoneySigned(req *NotifyRequest, secret string) bool {
	if secret == "" {
		return false
	}
	if sign := req.Form.Get("sign"); strings.TrimSpace(sign) != "" {
		for _, params := range yoomoneyParamSets(req) {
			if secureEqualFold(hmacSHA256Hex(secret, []byte(yoomoneySignString(params))), sign) {
				return true
			}
		}
	}
	if hash := req.Value("sha1_hash"); hash != "" {
		parts := []string{
			req.Value("notification_type"), req.Value("operation_id"), req.Value("amount"),
			req.Value("currency"), req.Value("datetime"), req.Value("sender"), req.Value("codepro"),
			secret, req.Value("label"),
		}
		return secureEqualFold(sha1Hex(strings.Join(parts, "&")), hash)
	}
	return false
}

func yoomoneyParamSets(req *NotifyRequest) []url.Values {
	sets := []url.Values{req.Form}
	if !bytes.Contains(req.Body, []byte("+")) {
		return sets
	}
	literal := url.Values{}
	for _, pair := range strings.Split(string(req.Body), "&") {
		if pair == "" {
			continue
		}
		key, value, _ := strings.Cut(pair, "=")
		k, errKey := url.PathUnescape(key)
		v, errValue := url.PathUnescape(value)
		if errKey != nil || errValue != nil {
			return sets
		}
		literal.Add(k, v)
	}
	return append(sets, literal)
}

func yoomoneySignString(params url.Values) string {
	keys := make([]string, 0, len(params))
	for key := range params {
		if key != "sign" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+strings.ReplaceAll(url.QueryEscape(params.Get(key)), "+", "%20"))
	}
	return strings.Join(parts, "&")
}

type TKassa struct{}

func (TKassa) Code() string { return "tkassa" }

func tkassaToken(fields map[string]string, password string) string {
	values := make(map[string]string, len(fields)+1)
	for k, v := range fields {
		values[k] = v
	}
	values["Password"] = password
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		b.WriteString(values[k])
	}
	return sha256Hex(b.String())
}

func (TKassa) CreateCheckout(ctx context.Context, cfg map[string]any, in CheckoutInput) (Checkout, error) {
	terminal, password := strCfg(cfg, "terminal_key"), strCfg(cfg, "password")
	if terminal == "" || password == "" {
		return Checkout{}, notConfigured("tkassa")
	}
	amount := minorUnits(in.Amount, "RUB")
	desc := truncate(in.Description, 140)
	scalars := map[string]string{
		"TerminalKey":     terminal,
		"Amount":          strconv.FormatInt(amount, 10),
		"OrderId":         in.PaymentID,
		"Description":     desc,
		"SuccessURL":      in.ReturnURL,
		"FailURL":         in.FailURL,
		"NotificationURL": in.NotifyURL,
	}
	body := map[string]any{
		"TerminalKey":     terminal,
		"Amount":          amount,
		"OrderId":         in.PaymentID,
		"Description":     desc,
		"SuccessURL":      in.ReturnURL,
		"FailURL":         in.FailURL,
		"NotificationURL": in.NotifyURL,
		"Token":           tkassaToken(scalars, password),
	}
	if in.Email != "" {
		body["DATA"] = map[string]string{"Email": in.Email}
		if boolCfg(cfg, "receipt") {
			receipt := map[string]any{
				"Email": in.Email,
				"Items": []map[string]any{{
					"Name":          in.Receipt.item(desc),
					"Price":         amount,
					"Quantity":      1,
					"Amount":        amount,
					"Tax":           in.Receipt.atolTax(),
					"PaymentMethod": in.Receipt.mode(),
					"PaymentObject": in.Receipt.subject(),
				}},
			}
			if sno := in.Receipt.sno(); sno != "" {
				receipt["Taxation"] = sno
			}
			body["Receipt"] = receipt
		}
	}
	var out struct {
		Success    bool            `json:"Success"`
		ErrorCode  string          `json:"ErrorCode"`
		Message    string          `json:"Message"`
		Details    string          `json:"Details"`
		PaymentURL string          `json:"PaymentURL"`
		PaymentID  json.RawMessage `json:"PaymentId"`
	}
	if err := sendJSON(ctx, http.MethodPost, urlCfg(cfg, "api_url", "https://securepay.tinkoff.ru/v2")+"/Init", body, &out); err != nil {
		return Checkout{}, fmt.Errorf("t-kassa: %w", err)
	}
	if !out.Success || out.PaymentURL == "" {
		return Checkout{}, fmt.Errorf("t-kassa: %s %s (код %s)", out.Message, out.Details, out.ErrorCode)
	}
	return Checkout{RedirectURL: out.PaymentURL, ProviderPaymentID: strings.Trim(string(out.PaymentID), `"`)}, nil
}

func (TKassa) HandleNotification(_ context.Context, cfg map[string]any, req *NotifyRequest) (Notification, error) {
	var raw map[string]any
	if err := req.DecodeJSON(&raw); err != nil {
		return Notification{}, fmt.Errorf("t-kassa: %w", err)
	}
	fields := map[string]string{}
	for k, v := range raw {
		if k == "Token" {
			continue
		}
		switch t := v.(type) {
		case string:
			fields[k] = t
		case bool:
			fields[k] = strconv.FormatBool(t)
		case json.Number:
			fields[k] = t.String()
		}
	}
	n := Notification{
		ProviderPaymentID: fields["PaymentId"],
		Amount:            fromMinorUnits(parseAmount(fields["Amount"]), "RUB"),
		Currency:          "RUB",
	}
	orderReference(&n, fields["OrderId"])
	token, _ := raw["Token"].(string)
	if !secureEqualFold(tkassaToken(fields, strCfg(cfg, "password")), token) {
		return n, ErrBadSignature
	}
	switch fields["Status"] {
	case "CONFIRMED":
		n.Status = NotifyPaid
	case "REJECTED", "CANCELED", "DEADLINE_EXPIRED", "AUTH_FAIL":
		n.Status = NotifyFailed
	}
	return n, nil
}

type Robokassa struct{}

func (Robokassa) Code() string { return "robokassa" }

func (Robokassa) CreateCheckout(_ context.Context, cfg map[string]any, in CheckoutInput) (Checkout, error) {
	login, pass1 := strCfg(cfg, "merchant_login"), strCfg(cfg, "password1")
	if login == "" || pass1 == "" || strCfg(cfg, "password2") == "" {
		return Checkout{}, notConfigured("robokassa")
	}
	outSum := money(in.Amount)
	invID := invoiceRef(in)
	parts := []string{login, outSum, invID}
	outSumCurrency := ""
	if cur := upper(in.Currency); cur != "" && cur != "RUB" && cur != "RUR" {
		outSumCurrency = cur
		parts = append(parts, cur)
	}
	receipt := ""
	if boolCfg(cfg, "receipt") {
		item := map[string]any{
			"name":           in.Receipt.item(in.Description),
			"quantity":       1,
			"sum":            roundCents(in.Amount),
			"payment_method": in.Receipt.mode(),
			"payment_object": in.Receipt.subject(),
			"tax":            in.Receipt.atolTax(),
		}
		data := map[string]any{"items": []map[string]any{item}}
		if sno := in.Receipt.sno(); sno != "" {
			data["sno"] = sno
		}
		raw, err := json.Marshal(data)
		if err != nil {
			return Checkout{}, fmt.Errorf("robokassa: %w", err)
		}
		receipt = url.QueryEscape(string(raw))
		parts = append(parts, receipt)
	}
	parts = append(parts, pass1, "Shp_payment="+in.PaymentID)

	q := url.Values{}
	q.Set("MerchantLogin", login)
	q.Set("OutSum", outSum)
	q.Set("InvId", invID)
	q.Set("Description", truncate(in.Description, 100))
	q.Set("SignatureValue", md5Hex(strings.Join(parts, ":")))
	q.Set("Shp_payment", in.PaymentID)
	q.Set("Culture", "ru")
	if outSumCurrency != "" {
		q.Set("OutSumCurrency", outSumCurrency)
	}
	if receipt != "" {
		q.Set("Receipt", receipt)
	}
	if in.Email != "" {
		q.Set("Email", in.Email)
	}
	if boolCfg(cfg, "test") {
		q.Set("IsTest", "1")
	}
	if in.ReturnURL != "" {
		q.Set("SuccessUrl2", in.ReturnURL)
		q.Set("SuccessUrl2Method", "GET")
	}
	if in.FailURL != "" {
		q.Set("FailUrl2", in.FailURL)
		q.Set("FailUrl2Method", "GET")
	}
	return Checkout{RedirectURL: "https://auth.robokassa.ru/Merchant/Index.aspx?" + q.Encode(), ProviderPaymentID: invID}, nil
}

func (Robokassa) HandleNotification(_ context.Context, cfg map[string]any, req *NotifyRequest) (Notification, error) {
	outSum, invID, shp := req.Value("OutSum"), req.Value("InvId"), req.Value("Shp_payment")
	n := Notification{InvoiceNo: parseInvoiceNo(invID), ProviderPaymentID: invID}
	if isUUID(shp) {
		n.PaymentID = shp
	}
	if invID == "" {
		return n, nil
	}
	base := outSum + ":" + invID + ":" + strCfg(cfg, "password2")
	if shp != "" {
		base += ":Shp_payment=" + shp
	}
	if !secureEqualFold(md5Hex(base), req.Value("SignatureValue")) {
		return n, ErrBadSignature
	}
	n.Status = NotifyPaid
	n.Amount = parseAmount(outSum)
	if cur := upper(cfgOr(cfg, "currency", "RUB")); cur == "RUB" || cur == "RUR" {
		n.Currency = "RUB"
	}
	return n, nil
}

func (Robokassa) Accept(n Notification) Response {
	return textResponse(http.StatusOK, "OK"+strconv.FormatInt(n.InvoiceNo, 10))
}

func (Robokassa) Reject(_ Notification, err error) Response {
	return textResponse(http.StatusBadRequest, errText(err))
}

type Freekassa struct{}

func (Freekassa) Code() string { return "freekassa" }

func (Freekassa) CreateCheckout(_ context.Context, cfg map[string]any, in CheckoutInput) (Checkout, error) {
	merchantID, secret1 := strCfg(cfg, "merchant_id"), strCfg(cfg, "secret1")
	if merchantID == "" || secret1 == "" || strCfg(cfg, "secret2") == "" {
		return Checkout{}, notConfigured("freekassa")
	}
	amount := money(in.Amount)
	currency := upper(in.Currency)
	order := invoiceRef(in)
	q := url.Values{}
	q.Set("m", merchantID)
	q.Set("oa", amount)
	q.Set("currency", currency)
	q.Set("o", order)
	q.Set("s", md5Hex(strings.Join([]string{merchantID, amount, secret1, currency, order}, ":")))
	q.Set("lang", "ru")
	q.Set("us_payment", in.PaymentID)
	if in.MethodID != "" {
		q.Set("i", in.MethodID)
	}
	if in.Email != "" {
		q.Set("em", in.Email)
	}
	return Checkout{RedirectURL: withQuery(cfgOr(cfg, "pay_url", "https://pay.fk.money/"), q), ProviderPaymentID: order}, nil
}

func (Freekassa) HandleNotification(_ context.Context, cfg map[string]any, req *NotifyRequest) (Notification, error) {
	merchantID, amount, order := req.Value("MERCHANT_ID"), req.Value("AMOUNT"), req.Value("MERCHANT_ORDER_ID")
	n := Notification{ProviderPaymentID: req.Value("intid")}
	if order == "" {
		return n, nil
	}
	orderReference(&n, order)
	if merchantID != strCfg(cfg, "merchant_id") {
		return n, fmt.Errorf("freekassa: уведомление для другого магазина (%s)", merchantID)
	}
	expected := md5Hex(strings.Join([]string{merchantID, amount, strCfg(cfg, "secret2"), order}, ":"))
	if !secureEqualFold(expected, req.Value("SIGN")) {
		return n, ErrBadSignature
	}
	n.Status = NotifyPaid
	n.Amount = parseAmount(amount)
	n.Currency = upper(cfgOr(cfg, "currency", "RUB"))
	return n, nil
}

func (Freekassa) Accept(Notification) Response {
	return textResponse(http.StatusOK, "YES")
}

func (Freekassa) Reject(_ Notification, err error) Response {
	return textResponse(http.StatusBadRequest, errText(err))
}

type CloudPayments struct{}

func (CloudPayments) Code() string { return "cloudpayments" }

func (CloudPayments) CreateCheckout(ctx context.Context, cfg map[string]any, in CheckoutInput) (Checkout, error) {
	publicID, secret := strCfg(cfg, "public_id"), strCfg(cfg, "api_secret")
	if publicID == "" || secret == "" {
		return Checkout{}, notConfigured("cloudpayments")
	}
	body := map[string]any{
		"Amount":              roundCents(in.Amount),
		"Currency":            upper(in.Currency),
		"Description":         truncate(in.Description, 200),
		"InvoiceId":           invoiceRef(in),
		"AccountId":           in.UserID,
		"RequireConfirmation": false,
		"SendEmail":           false,
		"SuccessRedirectUrl":  in.ReturnURL,
		"FailRedirectUrl":     in.FailURL,
	}
	if in.Email != "" {
		body["Email"] = in.Email
		if boolCfg(cfg, "receipt") {
			sum := roundCents(in.Amount)
			receipt := map[string]any{
				"Items": []map[string]any{{
					"label":    in.Receipt.item(in.Description),
					"price":    sum,
					"quantity": 1,
					"amount":   sum,
					"vat":      in.Receipt.cloudPaymentsVAT(),
					"method":   in.Receipt.cloudPaymentsMethod(),
					"object":   in.Receipt.cloudPaymentsObject(),
				}},
				"email":   in.Email,
				"amounts": map[string]any{"electronic": sum},
			}
			if code, ok := cloudPaymentsTaxations[in.Receipt.sno()]; ok {
				receipt["taxationSystem"] = code
			}
			body["JsonData"] = map[string]any{"CloudPayments": map[string]any{"CustomerReceipt": receipt}}
		}
	}
	var out struct {
		Success bool   `json:"Success"`
		Message string `json:"Message"`
		Model   struct {
			ID  string `json:"Id"`
			URL string `json:"Url"`
		} `json:"Model"`
	}
	if err := sendJSON(ctx, http.MethodPost, "https://api.cloudpayments.ru/orders/create", body, &out, withBasicAuth(publicID, secret)); err != nil {
		return Checkout{}, fmt.Errorf("cloudpayments: %w", err)
	}
	if !out.Success || out.Model.URL == "" {
		return Checkout{}, fmt.Errorf("cloudpayments: %s", firstNonEmpty(out.Message, "счёт не создан"))
	}
	return Checkout{RedirectURL: out.Model.URL, ProviderPaymentID: out.Model.ID}, nil
}

func (CloudPayments) HandleNotification(_ context.Context, cfg map[string]any, req *NotifyRequest) (Notification, error) {
	secret := strCfg(cfg, "api_secret")
	signed := false
	if h := req.Header.Get("Content-HMAC"); h != "" && secureEqual(hmacSHA256Base64(secret, req.Body), h) {
		signed = true
	}
	if !signed {
		if h := req.Header.Get("X-Content-HMAC"); h != "" {
			if decoded, err := url.QueryUnescape(string(req.Body)); err == nil && secureEqual(hmacSHA256Base64(secret, []byte(decoded)), h) {
				signed = true
			}
		}
	}
	if !signed {
		return Notification{}, ErrBadSignature
	}
	values := map[string]string{}
	if req.IsJSON() {
		var raw map[string]any
		if err := req.DecodeJSON(&raw); err != nil {
			return Notification{}, fmt.Errorf("cloudpayments: %w", err)
		}
		for k, v := range raw {
			if v != nil {
				values[k] = strings.TrimSpace(fmt.Sprint(v))
			}
		}
	} else {
		for k := range req.Form {
			values[k] = req.Form.Get(k)
		}
	}
	n := Notification{
		ProviderPaymentID: values["TransactionId"],
		Amount:            parseAmount(values["Amount"]),
		Currency:          upper(values["Currency"]),
	}
	orderReference(&n, values["InvoiceId"])
	switch strings.ToLower(req.Query.Get("type")) {
	case "check":
		n.Status = NotifyCheck
	case "fail":
		n.Status = NotifyFailed
	default:
		if values["Status"] == "Completed" || values["Status"] == "Authorized" {
			n.Status = NotifyPaid
		}
	}
	return n, nil
}

func (CloudPayments) Accept(Notification) Response {
	return jsonResponse(http.StatusOK, map[string]int{"code": 0})
}

func (CloudPayments) Reject(Notification, error) Response {
	return jsonResponse(http.StatusOK, map[string]int{"code": 13})
}

type Unitpay struct{}

func (Unitpay) Code() string { return "unitpay" }

func unitpayBase(cfg map[string]any) string {
	u, err := url.Parse(urlCfg(cfg, "api_url", "https://unitpay.money/api"))
	if err != nil || u.Host == "" {
		return "https://unitpay.money"
	}
	return u.Scheme + "://" + u.Host
}

func (Unitpay) CreateCheckout(_ context.Context, cfg map[string]any, in CheckoutInput) (Checkout, error) {
	publicKey, secret := strCfg(cfg, "public_key"), strCfg(cfg, "secret_key")
	if publicKey == "" || secret == "" {
		return Checkout{}, notConfigured("unitpay")
	}
	account := invoiceRef(in)
	currency := upper(in.Currency)
	desc := truncate(in.Description, 200)
	sum := money(in.Amount)
	q := url.Values{}
	q.Set("sum", sum)
	q.Set("account", account)
	q.Set("desc", desc)
	q.Set("currency", currency)
	q.Set("signature", sha256Hex(strings.Join([]string{account, currency, desc, sum, secret}, "{up}")))
	q.Set("locale", "ru")
	if in.ReturnURL != "" {
		q.Set("resultUrl", in.ReturnURL)
	}
	if in.FailURL != "" {
		q.Set("backUrl", in.FailURL)
	}
	if in.Email != "" {
		q.Set("customerEmail", in.Email)
	}
	return Checkout{RedirectURL: unitpayBase(cfg) + "/pay/" + url.PathEscape(publicKey) + "?" + q.Encode(), ProviderPaymentID: account}, nil
}

func (Unitpay) HandleNotification(_ context.Context, cfg map[string]any, req *NotifyRequest) (Notification, error) {
	method := req.Value("method")
	params := map[string]string{}
	for _, source := range []url.Values{req.Query, req.Form} {
		for k, vals := range source {
			if strings.HasPrefix(k, "params[") && strings.HasSuffix(k, "]") && len(vals) > 0 {
				params[k[len("params["):len(k)-1]] = vals[0]
			}
		}
	}
	n := Notification{
		ProviderPaymentID: params["unitpayId"],
		Amount:            parseAmount(params["orderSum"]),
		Currency:          upper(params["orderCurrency"]),
	}
	orderReference(&n, params["account"])
	keys := make([]string, 0, len(params))
	for k := range params {
		if k != "signature" && k != "sign" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	parts := []string{method}
	for _, k := range keys {
		parts = append(parts, params[k])
	}
	parts = append(parts, strCfg(cfg, "secret_key"))
	if !secureEqualFold(sha256Hex(strings.Join(parts, "{up}")), params["signature"]) {
		return n, ErrBadSignature
	}
	switch method {
	case "check":
		n.Status = NotifyCheck
	case "pay":
		n.Status = NotifyPaid
	case "error":
		n.Status = NotifyFailed
	}
	return n, nil
}

func (Unitpay) Accept(Notification) Response {
	return jsonResponse(http.StatusOK, map[string]any{"result": map[string]string{"message": "OK"}})
}

func (Unitpay) Reject(_ Notification, err error) Response {
	return jsonResponse(http.StatusOK, map[string]any{"error": map[string]string{"message": errText(err)}})
}

type Lava struct{}

func (Lava) Code() string { return "lava" }

func (Lava) CreateCheckout(ctx context.Context, cfg map[string]any, in CheckoutInput) (Checkout, error) {
	shopID, secret := strCfg(cfg, "shop_id"), strCfg(cfg, "secret_key")
	if shopID == "" || secret == "" {
		return Checkout{}, notConfigured("lava")
	}
	payload, _ := json.Marshal(map[string]any{
		"sum":        roundCents(in.Amount),
		"orderId":    in.PaymentID,
		"shopId":     shopID,
		"hookUrl":    in.NotifyURL,
		"successUrl": in.ReturnURL,
		"failUrl":    in.FailURL,
		"expire":     300,
		"comment":    truncate(in.Description, 255),
	})
	var out struct {
		Data struct {
			ID  string `json:"id"`
			URL string `json:"url"`
		} `json:"data"`
	}
	err := sendJSON(ctx, http.MethodPost, urlCfg(cfg, "api_url", "https://api.lava.ru")+"/business/invoice/create", payload, &out,
		withHeader("Signature", hmacSHA256Hex(secret, payload)))
	if err != nil {
		return Checkout{}, fmt.Errorf("lava: %w", err)
	}
	if out.Data.URL == "" {
		return Checkout{}, fmt.Errorf("lava: в ответе нет ссылки на оплату")
	}
	return Checkout{RedirectURL: out.Data.URL, ProviderPaymentID: out.Data.ID}, nil
}

func (Lava) HandleNotification(ctx context.Context, cfg map[string]any, req *NotifyRequest) (Notification, error) {
	var hook struct {
		InvoiceID string `json:"invoice_id"`
		OrderID   string `json:"order_id"`
	}
	if err := req.DecodeJSON(&hook); err != nil {
		return Notification{}, fmt.Errorf("lava: %w", err)
	}
	if hook.InvoiceID == "" && hook.OrderID == "" {
		return Notification{}, nil
	}
	secret := strCfg(cfg, "secret_key")
	payload, _ := json.Marshal(map[string]any{
		"shopId":    strCfg(cfg, "shop_id"),
		"invoiceId": hook.InvoiceID,
		"orderId":   hook.OrderID,
	})
	var out struct {
		Data struct {
			ID      string `json:"id"`
			OrderID string `json:"order_id"`
			Status  string `json:"status"`
			Amount  any    `json:"amount"`
		} `json:"data"`
	}
	err := sendJSON(ctx, http.MethodPost, urlCfg(cfg, "api_url", "https://api.lava.ru")+"/business/invoice/status", payload, &out,
		withHeader("Signature", hmacSHA256Hex(secret, payload)))
	if err != nil {
		return Notification{}, fmt.Errorf("lava: проверка счёта: %w", err)
	}
	n := Notification{
		ProviderPaymentID: firstNonEmpty(out.Data.ID, hook.InvoiceID),
		Amount:            numberValue(out.Data.Amount),
		Currency:          "RUB",
	}
	orderReference(&n, firstNonEmpty(out.Data.OrderID, hook.OrderID))
	switch strings.ToLower(out.Data.Status) {
	case "success":
		n.Status = NotifyPaid
	case "expired", "cancel", "canceled", "error":
		n.Status = NotifyFailed
	}
	return n, nil
}

type Enot struct{}

func (Enot) Code() string { return "enot" }

func enotBase(cfg map[string]any) string {
	base := urlCfg(cfg, "api_url", "https://api.enot.io")
	if u, err := url.Parse(base); err == nil && strings.EqualFold(u.Host, "enot.io") {
		return "https://api.enot.io"
	}
	return base
}

func (Enot) CreateCheckout(ctx context.Context, cfg map[string]any, in CheckoutInput) (Checkout, error) {
	shopID, secret := strCfg(cfg, "shop_id"), strCfg(cfg, "secret_key")
	if shopID == "" || secret == "" {
		return Checkout{}, notConfigured("enot")
	}
	body := map[string]any{
		"amount":      roundCents(in.Amount),
		"order_id":    in.PaymentID,
		"currency":    upper(in.Currency),
		"shop_id":     shopID,
		"hook_url":    in.NotifyURL,
		"success_url": in.ReturnURL,
		"fail_url":    in.FailURL,
		"comment":     truncate(in.Description, 255),
		"expire":      300,
	}
	if in.Email != "" {
		body["email"] = in.Email
	}
	var out struct {
		Data struct {
			ID  string `json:"id"`
			URL string `json:"url"`
		} `json:"data"`
	}
	if err := sendJSON(ctx, http.MethodPost, enotBase(cfg)+"/invoice/create", body, &out, withHeader("x-api-key", secret)); err != nil {
		return Checkout{}, fmt.Errorf("enot: %w", err)
	}
	if out.Data.URL == "" {
		return Checkout{}, fmt.Errorf("enot: в ответе нет ссылки на оплату")
	}
	return Checkout{RedirectURL: out.Data.URL, ProviderPaymentID: out.Data.ID}, nil
}

func (Enot) HandleNotification(ctx context.Context, cfg map[string]any, req *NotifyRequest) (Notification, error) {
	var hook struct {
		InvoiceID string `json:"invoice_id"`
		OrderID   string `json:"order_id"`
	}
	if err := req.DecodeJSON(&hook); err != nil {
		return Notification{}, fmt.Errorf("enot: %w", err)
	}
	if hook.InvoiceID == "" && hook.OrderID == "" {
		return Notification{}, nil
	}
	q := url.Values{}
	q.Set("shop_id", strCfg(cfg, "shop_id"))
	if hook.InvoiceID != "" {
		q.Set("invoice_id", hook.InvoiceID)
	} else {
		q.Set("order_id", hook.OrderID)
	}
	var out struct {
		Data struct {
			InvoiceID     string `json:"invoice_id"`
			OrderID       string `json:"order_id"`
			Status        string `json:"status"`
			InvoiceAmount any    `json:"invoice_amount"`
			Amount        any    `json:"amount"`
			Currency      string `json:"currency"`
		} `json:"data"`
	}
	if err := sendJSON(ctx, http.MethodGet, enotBase(cfg)+"/invoice/info?"+q.Encode(), nil, &out, withHeader("x-api-key", strCfg(cfg, "secret_key"))); err != nil {
		return Notification{}, fmt.Errorf("enot: проверка счёта: %w", err)
	}
	amount := numberValue(out.Data.InvoiceAmount)
	if amount == 0 {
		amount = numberValue(out.Data.Amount)
	}
	n := Notification{
		ProviderPaymentID: firstNonEmpty(out.Data.InvoiceID, hook.InvoiceID),
		Amount:            amount,
		Currency:          upper(out.Data.Currency),
	}
	orderReference(&n, firstNonEmpty(out.Data.OrderID, hook.OrderID))
	switch strings.ToLower(out.Data.Status) {
	case "success":
		n.Status = NotifyPaid
	case "fail", "expired", "refund":
		n.Status = NotifyFailed
	}
	return n, nil
}

type PayMaster struct{}

func (PayMaster) Code() string { return "paymaster" }

func (PayMaster) CreateCheckout(_ context.Context, cfg map[string]any, in CheckoutInput) (Checkout, error) {
	merchantID := strCfg(cfg, "merchant_id")
	if merchantID == "" || strCfg(cfg, "secret_key") == "" {
		return Checkout{}, notConfigured("paymaster")
	}
	form := formOf(cfgOr(cfg, "process_url", "https://paymaster.ru/payment/init"),
		"LMI_MERCHANT_ID", merchantID,
		"LMI_PAYMENT_AMOUNT", money(in.Amount),
		"LMI_CURRENCY", upper(in.Currency),
		"LMI_PAYMENT_NO", invoiceRef(in),
		"LMI_PAYMENT_DESC_BASE64", base64Text(truncate(in.Description, 255)),
		"LMI_PAYMENT_NOTIFICATION_URL", in.NotifyURL,
		"LMI_SUCCESS_URL", in.ReturnURL,
		"LMI_FAILURE_URL", in.FailURL,
		"payment_id", in.PaymentID,
	)
	if in.Email != "" {
		form.add("LMI_PAYER_EMAIL", in.Email)
	}
	return Checkout{Form: form, ProviderPaymentID: invoiceRef(in)}, nil
}

func paymasterDigest(algo string, data string) []byte {
	var h hash.Hash
	switch strings.ToLower(algo) {
	case "sha1":
		h = sha1.New()
	case "sha256":
		h = sha256.New()
	default:
		h = md5.New()
	}
	h.Write([]byte(data))
	return h.Sum(nil)
}

func (PayMaster) HandleNotification(_ context.Context, cfg map[string]any, req *NotifyRequest) (Notification, error) {
	get := req.Value
	n := Notification{
		InvoiceNo:         parseInvoiceNo(get("LMI_PAYMENT_NO")),
		ProviderPaymentID: get("LMI_SYS_PAYMENT_ID"),
		Amount:            parseAmount(get("LMI_PAYMENT_AMOUNT")),
		Currency:          upper(firstNonEmpty(get("LMI_CURRENCY"), cfgOr(cfg, "currency", "RUB"))),
	}
	if isUUID(get("payment_id")) {
		n.PaymentID = get("payment_id")
	}
	if n.InvoiceNo == 0 && n.PaymentID == "" {
		return n, nil
	}
	if get("LMI_MERCHANT_ID") != strCfg(cfg, "merchant_id") {
		return n, fmt.Errorf("paymaster: уведомление для другого магазина")
	}
	if get("LMI_PREREQUEST") == "1" {
		n.Status = NotifyCheck
		return n, nil
	}
	secret := strCfg(cfg, "secret_key")
	candidates := []string{
		strings.Join([]string{
			get("LMI_MERCHANT_ID"), get("LMI_PAYMENT_NO"), get("LMI_SYS_PAYMENT_ID"), get("LMI_SYS_PAYMENT_DATE"),
			get("LMI_PAYMENT_AMOUNT"), get("LMI_CURRENCY"), get("LMI_PAID_AMOUNT"), get("LMI_PAID_CURRENCY"),
			get("LMI_PAYMENT_SYSTEM"), get("LMI_SIM_MODE"), secret,
		}, ";"),
		get("LMI_MERCHANT_ID") + get("LMI_PAYMENT_NO") + get("LMI_SYS_PAYMENT_ID") + get("LMI_SYS_PAYMENT_DATE") +
			get("LMI_PAYMENT_AMOUNT") + get("LMI_PAID_AMOUNT") + get("LMI_PAYMENT_SYSTEM") + get("LMI_MODE") + secret,
	}
	algos := []string{strCfg(cfg, "hash_algo"), "md5", "sha1", "sha256"}
	given := get("LMI_HASH")
	valid := false
	for _, data := range candidates {
		for _, algo := range algos {
			sum := paymasterDigest(algo, data)
			if secureEqual(base64.StdEncoding.EncodeToString(sum), given) || secureEqualFold(hex.EncodeToString(sum), given) {
				valid = true
			}
		}
	}
	if !valid {
		return n, ErrBadSignature
	}
	if mode := get("LMI_SIM_MODE"); mode != "" && mode != "0" {
		return n, nil
	}
	n.Status = NotifyPaid
	return n, nil
}

func (PayMaster) Accept(n Notification) Response {
	if n.Status == NotifyCheck {
		return textResponse(http.StatusOK, "YES")
	}
	return textResponse(http.StatusOK, "OK")
}

func (PayMaster) Reject(_ Notification, err error) Response {
	return textResponse(http.StatusBadRequest, errText(err))
}

type Payeer struct{}

func (Payeer) Code() string { return "payeer" }

func (Payeer) CreateCheckout(_ context.Context, cfg map[string]any, in CheckoutInput) (Checkout, error) {
	shop, key := strCfg(cfg, "shop_id"), strCfg(cfg, "secret_key")
	if shop == "" || key == "" {
		return Checkout{}, notConfigured("payeer")
	}
	orderID := invoiceRef(in)
	amount := money(in.Amount)
	currency := upper(in.Currency)
	desc := base64Text(truncate(in.Description, 200))
	sign := upper(sha256Hex(strings.Join([]string{shop, orderID, amount, currency, desc, key}, ":")))
	q := url.Values{}
	q.Set("m_shop", shop)
	q.Set("m_orderid", orderID)
	q.Set("m_amount", amount)
	q.Set("m_curr", currency)
	q.Set("m_desc", desc)
	q.Set("m_sign", sign)
	return Checkout{RedirectURL: withQuery(cfgOr(cfg, "process_url", "https://payeer.com/merchant/"), q), ProviderPaymentID: orderID}, nil
}

func (Payeer) HandleNotification(_ context.Context, cfg map[string]any, req *NotifyRequest) (Notification, error) {
	get := req.Value
	n := Notification{
		InvoiceNo:         parseInvoiceNo(get("m_orderid")),
		ProviderPaymentID: get("m_operation_id"),
		Amount:            parseAmount(get("m_amount")),
		Currency:          upper(get("m_curr")),
	}
	if get("m_operation_id") == "" {
		return n, nil
	}
	parts := []string{}
	for _, f := range []string{"m_operation_id", "m_operation_ps", "m_operation_date", "m_operation_pay_date", "m_shop", "m_orderid", "m_amount", "m_curr", "m_desc", "m_status"} {
		parts = append(parts, get(f))
	}
	if p := get("m_params"); p != "" {
		parts = append(parts, p)
	}
	parts = append(parts, strCfg(cfg, "secret_key"))
	if !secureEqualFold(sha256Hex(strings.Join(parts, ":")), get("m_sign")) {
		return n, ErrBadSignature
	}
	if get("m_status") == "success" {
		n.Status = NotifyPaid
	} else {
		n.Status = NotifyFailed
	}
	return n, nil
}

func (Payeer) Accept(n Notification) Response {
	return textResponse(http.StatusOK, strconv.FormatInt(n.InvoiceNo, 10)+"|success")
}

func (Payeer) Reject(n Notification, _ error) Response {
	return textResponse(http.StatusOK, strconv.FormatInt(n.InvoiceNo, 10)+"|error")
}

type WebMoney struct{}

func (WebMoney) Code() string { return "webmoney" }

func (WebMoney) CreateCheckout(_ context.Context, cfg map[string]any, in CheckoutInput) (Checkout, error) {
	purse := strCfg(cfg, "purse")
	if purse == "" || strCfg(cfg, "secret_key") == "" {
		return Checkout{}, notConfigured("webmoney")
	}
	form := formOf(cfgOr(cfg, "process_url", "https://merchant.webmoney.ru/lmi/payment_utf.asp"),
		"LMI_PAYEE_PURSE", purse,
		"LMI_PAYMENT_AMOUNT", money(in.Amount),
		"LMI_PAYMENT_NO", invoiceRef(in),
		"LMI_PAYMENT_DESC_BASE64", base64Text(truncate(in.Description, 255)),
		"LMI_RESULT_URL", in.NotifyURL,
		"LMI_SUCCESS_URL", in.ReturnURL,
		"LMI_SUCCESS_METHOD", "0",
		"LMI_FAIL_URL", in.FailURL,
		"LMI_FAIL_METHOD", "0",
		"payment_id", in.PaymentID,
	)
	return Checkout{Form: form, ProviderPaymentID: invoiceRef(in)}, nil
}

func (WebMoney) HandleNotification(_ context.Context, cfg map[string]any, req *NotifyRequest) (Notification, error) {
	get := req.Value
	n := Notification{
		InvoiceNo:         parseInvoiceNo(get("LMI_PAYMENT_NO")),
		ProviderPaymentID: get("LMI_SYS_TRANS_NO"),
		Amount:            parseAmount(get("LMI_PAYMENT_AMOUNT")),
		Currency:          webmoneyCurrency(cfg),
	}
	if isUUID(get("payment_id")) {
		n.PaymentID = get("payment_id")
	}
	if n.InvoiceNo == 0 {
		return n, nil
	}
	if !strings.EqualFold(get("LMI_PAYEE_PURSE"), strCfg(cfg, "purse")) {
		return n, fmt.Errorf("webmoney: оплата на другой кошелёк")
	}
	if get("LMI_PREREQUEST") == "1" {
		n.Status = NotifyCheck
		return n, nil
	}
	data := get("LMI_PAYEE_PURSE") + get("LMI_PAYMENT_AMOUNT") + get("LMI_PAYMENT_NO") + get("LMI_MODE") +
		get("LMI_SYS_INVS_NO") + get("LMI_SYS_TRANS_NO") + get("LMI_SYS_TRANS_DATE") + strCfg(cfg, "secret_key") +
		get("LMI_PAYER_PURSE") + get("LMI_PAYER_WM")
	given := get("LMI_HASH")
	if !secureEqualFold(sha256Hex(data), given) && !secureEqualFold(md5Hex(data), given) {
		return n, ErrBadSignature
	}
	if get("LMI_MODE") == "1" {
		return n, nil
	}
	n.Status = NotifyPaid
	return n, nil
}

func (WebMoney) Accept(n Notification) Response {
	if n.Status == NotifyCheck {
		return textResponse(http.StatusOK, "YES")
	}
	return textResponse(http.StatusOK, "OK")
}

func (WebMoney) Reject(_ Notification, err error) Response {
	return textResponse(http.StatusOK, errText(err))
}

type WebPay struct{}

func (WebPay) Code() string { return "webpay" }

func (WebPay) CreateCheckout(_ context.Context, cfg map[string]any, in CheckoutInput) (Checkout, error) {
	storeID, secret := strCfg(cfg, "store_id"), strCfg(cfg, "secret_key")
	if storeID == "" || secret == "" {
		return Checkout{}, notConfigured("webpay")
	}
	test := "0"
	if boolCfg(cfg, "test") {
		test = "1"
	}
	action := cfgOr(cfg, "process_url", "https://payment.webpay.by")
	if test == "1" && strings.Contains(action, "payment.webpay.by") {
		action = "https://securesandbox.webpay.by"
	}
	seed := strconv.FormatInt(time.Now().Unix(), 10)
	order := invoiceRef(in)
	total := money(in.Amount)
	currency := upper(in.Currency)
	form := formOf(rootPath(action),
		"*scart", "",
		"wsb_version", "2",
		"wsb_storeid", storeID,
		"wsb_order_num", order,
		"wsb_currency_id", currency,
		"wsb_seed", seed,
		"wsb_test", test,
		"wsb_invoice_item_name[0]", truncate(in.Description, 100),
		"wsb_invoice_item_quantity[0]", "1",
		"wsb_invoice_item_price[0]", total,
		"wsb_total", total,
		"wsb_signature", sha1Hex(seed+storeID+order+test+currency+total+secret),
		"wsb_return_url", in.ReturnURL,
		"wsb_cancel_return_url", in.FailURL,
		"wsb_notify_url", in.NotifyURL,
	)
	if in.Email != "" {
		form.add("wsb_email", in.Email)
	}
	return Checkout{Form: form, ProviderPaymentID: order}, nil
}

func (WebPay) HandleNotification(_ context.Context, cfg map[string]any, req *NotifyRequest) (Notification, error) {
	get := req.Value
	n := Notification{
		InvoiceNo:         parseInvoiceNo(get("site_order_id")),
		ProviderPaymentID: get("transaction_id"),
		Amount:            parseAmount(get("amount")),
		Currency:          upper(get("currency_id")),
	}
	if n.InvoiceNo == 0 {
		return n, nil
	}
	data := get("batch_timestamp") + get("currency_id") + get("amount") + get("payment_method") + get("order_id") +
		get("site_order_id") + get("transaction_id") + get("payment_type") + get("rrn") + strCfg(cfg, "secret_key")
	if !secureEqualFold(md5Hex(data), get("wsb_signature")) {
		return n, ErrBadSignature
	}
	switch get("payment_type") {
	case "1", "4":
		n.Status = NotifyPaid
	case "2":
		n.Status = NotifyFailed
	}
	return n, nil
}

func errText(err error) string {
	if err == nil {
		return "error"
	}
	return err.Error()
}
