package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/stripe/stripe-go/v81/webhook"
)

type stripeWebhookEvent struct {
	Type string `json:"type"`
	Data struct {
		Object struct {
			ID            string            `json:"id"`
			PaymentStatus string            `json:"payment_status"`
			Status        string            `json:"status"`
			Metadata      map[string]string `json:"metadata"`
		} `json:"object"`
	} `json:"data"`
}

func (h *Handler) StripeWebhook(w http.ResponseWriter, r *http.Request) {
	secret := strings.TrimSpace(os.Getenv("STRIPE_WEBHOOK_SECRET"))
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}

	if secret == "" {
		log.Printf("stripe webhook: STRIPE_WEBHOOK_SECRET не задан — уведомление отклонено")
		writeError(w, http.StatusServiceUnavailable,
			"приём Stripe не настроен: не задан секрет подписи вебхука")
		return
	}

	var ev stripeWebhookEvent
	sig := r.Header.Get("Stripe-Signature")
	if sig == "" {
		writeError(w, http.StatusUnauthorized, "missing stripe signature")
		return
	}
	stripeEvent, err := webhook.ConstructEvent(body, sig, secret)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid signature")
		return
	}
	ev.Type = string(stripeEvent.Type)
	if err := json.Unmarshal(stripeEvent.Data.Raw, &ev.Data.Object); err != nil {
		writeError(w, http.StatusBadRequest, "invalid event data")
		return
	}

	paymentID := ev.Data.Object.Metadata["payment_id"]
	if paymentID == "" {
		paymentID = ev.Data.Object.Metadata["vortanix_payment_id"]
	}
	if paymentID == "" {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ignored"})
		return
	}

	ctx, found := h.withPaymentTenant(r.Context(), paymentID)
	if !found {
		writeError(w, http.StatusNotFound, "payment not found")
		return
	}

	switch ev.Type {
	case "checkout.session.completed", "payment_intent.succeeded":
		if err := h.completeTopupPayment(ctx, paymentID, ev.Data.Object.ID, "Stripe"); err != nil {
			if strings.Contains(err.Error(), "not found") {
				writeError(w, http.StatusNotFound, "payment not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "failed to complete payment")
			return
		}
	default:
		writeJSON(w, http.StatusOK, map[string]string{"status": "ignored"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
