package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/vortanixapp/panel/internal/api/payments"
	"github.com/vortanixapp/panel/pkg/yookassa"
)

func (h *Handler) YooKassaWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	var ev struct {
		Event  string `json:"event"`
		Object struct {
			ID       string            `json:"id"`
			Status   string            `json:"status"`
			Metadata map[string]string `json:"metadata"`
		} `json:"object"`
	}
	if err := json.Unmarshal(body, &ev); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if ev.Event != "payment.succeeded" && ev.Object.Status != "succeeded" {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ignored"})
		return
	}
	paymentID := paymentIDFromMeta(ev.Object.Metadata)
	if paymentID == "" {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ignored"})
		return
	}

	ctx, found := h.knownPayment(r.Context(), paymentID)
	if !found {
		writeError(w, http.StatusNotFound, "payment not found")
		return
	}
	r = r.WithContext(ctx)

	if !h.yookassaConfirms(r, paymentID, ev.Object.ID) {
		writeError(w, http.StatusUnauthorized, "касса не подтверждает платёж")
		return
	}

	if err := h.completeTopupPayment(ctx, paymentID, ev.Object.ID, "YooKassa"); err != nil {
		if strings.Contains(err.Error(), "no rows") {
			writeError(w, http.StatusNotFound, "payment not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to complete payment")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) FreekassaWebhook(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	q := r.URL.Query()
	merchantID := firstNonEmpty(r.FormValue("MERCHANT_ID"), r.FormValue("m"), q.Get("MERCHANT_ID"), q.Get("m"))
	amount := firstNonEmpty(r.FormValue("AMOUNT"), r.FormValue("oa"), q.Get("AMOUNT"), q.Get("oa"))
	orderID := firstNonEmpty(r.FormValue("MERCHANT_ORDER_ID"), r.FormValue("o"), q.Get("MERCHANT_ORDER_ID"), q.Get("o"))
	sign := firstNonEmpty(r.FormValue("SIGN"), r.FormValue("s"), q.Get("SIGN"), q.Get("s"))
	if orderID == "" {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ignored"})
		return
	}

	ctx, found := h.knownPayment(r.Context(), orderID)
	if !found {
		writeError(w, http.StatusNotFound, "payment not found")
		return
	}

	var cfgRaw []byte
	err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT p.config FROM core.payments pay
		JOIN core.payment_providers p ON p.tenant_id = pay.tenant_id AND p.provider = pay.provider
		WHERE pay.id = $1 AND pay.provider = 'freekassa'
	`, orderID).Scan(&cfgRaw)
	if err != nil {
		writeError(w, http.StatusNotFound, "payment not found")
		return
	}
	cfg := h.providerConfig(cfgRaw)
	if !payments.VerifyFreekassaSign(cfg, merchantID, amount, orderID, sign) {
		writeError(w, http.StatusUnauthorized, "invalid signature")
		return
	}
	if err := h.completeTopupPayment(ctx, orderID, merchantID+":"+amount, "Freekassa"); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to complete payment")
		return
	}
	w.Write([]byte("YES"))
}

func (h *Handler) RobokassaWebhook(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	q := r.URL.Query()
	outSum := firstNonEmpty(r.FormValue("OutSum"), q.Get("OutSum"))
	invID := firstNonEmpty(r.FormValue("InvId"), q.Get("InvId"))
	signature := firstNonEmpty(r.FormValue("SignatureValue"), q.Get("SignatureValue"))
	if invID == "" {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ignored"})
		return
	}

	ctx, found := h.knownPayment(r.Context(), invID)
	if !found {
		writeError(w, http.StatusNotFound, "payment not found")
		return
	}

	var cfgRaw []byte
	err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT p.config FROM core.payments pay
		JOIN core.payment_providers p ON p.tenant_id = pay.tenant_id AND p.provider = pay.provider
		WHERE pay.id = $1 AND pay.provider = 'robokassa'
	`, invID).Scan(&cfgRaw)
	if err != nil {
		writeError(w, http.StatusNotFound, "payment not found")
		return
	}
	cfg := h.providerConfig(cfgRaw)
	if !payments.VerifyRobokassaResult(cfg, outSum, invID, signature) {
		writeError(w, http.StatusUnauthorized, "invalid signature")
		return
	}
	if err := h.completeTopupPayment(ctx, invID, invID, "Robokassa"); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to complete payment")
		return
	}
	w.Write([]byte("OK" + invID))
}

func (h *Handler) PayPalWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	var ev struct {
		EventType string `json:"event_type"`
		Resource  struct {
			ID       string `json:"id"`
			CustomID string `json:"custom_id"`
			Amount   struct {
				Value        string `json:"value"`
				CurrencyCode string `json:"currency_code"`
			} `json:"amount"`
			PurchaseUnits []struct {
				CustomID string `json:"custom_id"`
				Amount   struct {
					Value        string `json:"value"`
					CurrencyCode string `json:"currency_code"`
				} `json:"amount"`
			} `json:"purchase_units"`
		} `json:"resource"`
	}
	if err := json.Unmarshal(body, &ev); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	paymentID := ev.Resource.CustomID
	if paymentID == "" && len(ev.Resource.PurchaseUnits) > 0 {
		paymentID = ev.Resource.PurchaseUnits[0].CustomID
	}
	if paymentID == "" {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ignored"})
		return
	}

	ctx, found := h.knownPayment(r.Context(), paymentID)
	if !found {
		writeError(w, http.StatusNotFound, "payment not found")
		return
	}

	var cfgRaw []byte
	err = h.dbOf(ctx).QueryRow(ctx, `
		SELECT p.config FROM core.payments pay
		JOIN core.payment_providers p ON p.tenant_id = pay.tenant_id AND p.provider = pay.provider
		WHERE pay.id = $1 AND pay.provider = 'paypal'
	`, paymentID).Scan(&cfgRaw)
	if err != nil {
		writeError(w, http.StatusNotFound, "payment not found")
		return
	}
	cfg := h.providerConfig(cfgRaw)

	ok, verr := payments.VerifyPayPalWebhook(ctx, cfg, r.Header, body)
	if verr != nil || !ok {
		writeError(w, http.StatusUnauthorized, "invalid signature")
		return
	}

	paidValue, paidCurrency := ev.Resource.Amount.Value, ev.Resource.Amount.CurrencyCode
	if paidValue == "" && len(ev.Resource.PurchaseUnits) > 0 {
		paidValue = ev.Resource.PurchaseUnits[0].Amount.Value
		paidCurrency = ev.Resource.PurchaseUnits[0].Amount.CurrencyCode
	}
	paid, _ := strconv.ParseFloat(strings.TrimSpace(paidValue), 64)
	if !h.paymentCoversOrder(ctx, paymentID, paid, paidCurrency, "PayPal") {
		writeError(w, http.StatusBadRequest, "оплаченная сумма меньше заказа")
		return
	}

	switch ev.EventType {
	case "CHECKOUT.ORDER.APPROVED":
		if err := payments.CapturePayPalOrder(ctx, cfg, ev.Resource.ID); err != nil {
			writeError(w, http.StatusInternalServerError, "capture failed")
			return
		}
		if err := h.completeTopupPayment(ctx, paymentID, ev.Resource.ID, "PayPal"); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to complete payment")
			return
		}
	case "PAYMENT.CAPTURE.COMPLETED":
		if err := h.completeTopupPayment(ctx, paymentID, ev.Resource.ID, "PayPal"); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to complete payment")
			return
		}
	default:
		writeJSON(w, http.StatusOK, map[string]string{"status": "ignored"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) NowPaymentsWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	var ev struct {
		PaymentStatus string      `json:"payment_status"`
		OrderID       string      `json:"order_id"`
		PaymentID     json.Number `json:"payment_id"`
		PriceAmount   float64     `json:"price_amount"`
		PriceCurrency string      `json:"price_currency"`
	}
	if err := json.Unmarshal(body, &ev); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if ev.OrderID == "" {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ignored"})
		return
	}

	ctx, found := h.knownPayment(r.Context(), ev.OrderID)
	if !found {
		writeError(w, http.StatusNotFound, "payment not found")
		return
	}

	var cfgRaw []byte
	err = h.dbOf(ctx).QueryRow(ctx, `
		SELECT p.config FROM core.payments pay
		JOIN core.payment_providers p ON p.tenant_id = pay.tenant_id AND p.provider = pay.provider
		WHERE pay.id = $1 AND pay.provider = 'nowpayments'
	`, ev.OrderID).Scan(&cfgRaw)
	if err != nil {
		writeError(w, http.StatusNotFound, "payment not found")
		return
	}
	cfg := h.providerConfig(cfgRaw)

	sig := r.Header.Get("x-nowpayments-sig")
	if !payments.VerifyNowPaymentsSign(cfg, body, sig) {
		writeError(w, http.StatusUnauthorized, "invalid signature")
		return
	}

	if !h.paymentCoversOrder(ctx, ev.OrderID, ev.PriceAmount, ev.PriceCurrency, "NowPayments") {
		writeError(w, http.StatusBadRequest, "оплаченная сумма меньше заказа")
		return
	}

	switch ev.PaymentStatus {
	case "finished", "confirmed":
		if err := h.completeTopupPayment(ctx, ev.OrderID, ev.PaymentID.String(), "NowPayments"); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to complete payment")
			return
		}
	default:
		writeJSON(w, http.StatusOK, map[string]string{"status": "ignored"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func (h *Handler) providerConfig(raw []byte) map[string]any {
	cfg := map[string]any{}
	plain, err := h.secrets.DecryptJSON(raw)
	if err != nil {
		return cfg
	}
	_ = json.Unmarshal(plain, &cfg)
	return cfg
}

func (h *Handler) yookassaConfirms(r *http.Request, localPaymentID, providerPaymentID string) bool {
	if providerPaymentID == "" {
		return false
	}
	var cfgRaw []byte
	var amount float64
	var currency string
	err := h.dbOf(r.Context()).QueryRow(r.Context(), `
		SELECT p.config, pay.amount, COALESCE(pay.currency, 'RUB')
		FROM core.payments pay
		JOIN core.payment_providers p ON p.tenant_id = pay.tenant_id AND p.provider = pay.provider
		WHERE pay.id = $1 AND pay.provider = 'yookassa'
	`, localPaymentID).Scan(&cfgRaw, &amount, &currency)
	if err != nil {
		log.Printf("yookassa webhook: платёж %s не найден: %v", localPaymentID, err)
		return false
	}
	cfg := h.providerConfig(cfgRaw)

	client := yookassa.New(strCfgValue(cfg, "shop_id"), strCfgValue(cfg, "secret_key"))
	remote, err := client.GetPayment(r.Context(), providerPaymentID)
	if err != nil {
		log.Printf("yookassa webhook: проверка платежа %s не удалась: %v", localPaymentID, err)
		return false
	}
	if !remote.Succeeded() {
		log.Printf("yookassa webhook: касса не подтверждает %s (status=%s paid=%v)",
			localPaymentID, remote.Status, remote.Paid)
		return false
	}
	if remote.Amount+0.009 < amount {
		log.Printf("yookassa webhook: сумма не сходится по %s: касса %.2f, заказ %.2f",
			localPaymentID, remote.Amount, amount)
		return false
	}
	return true
}

func strCfgValue(cfg map[string]any, key string) string {
	if v, ok := cfg[key].(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}
