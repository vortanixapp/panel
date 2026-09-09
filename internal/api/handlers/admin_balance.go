package handlers

import (
	"encoding/json"
	"math"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/vortanixapp/panel/pkg/notify"
)

func (h *Handler) AdminAdjustUserBalance(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	userID := chi.URLParam(r, "id")
	var body struct {
		Amount   float64 `json:"amount"`
		Currency string  `json:"currency"`
		Comment  string  `json:"comment"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	amount := math.Round(body.Amount*100) / 100
	if amount == 0 {
		writeError(w, http.StatusBadRequest, "amount must not be zero")
		return
	}
	currency := strings.ToUpper(strings.TrimSpace(body.Currency))
	if currency == "" {
		currency = "RUB"
	}
	comment := strings.TrimSpace(body.Comment)
	if comment == "" {
		writeError(w, http.StatusBadRequest, "укажите причину корректировки")
		return
	}

	tx, err := h.dbOf(r.Context()).Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer tx.Rollback(r.Context())

	var walletID string
	var balance float64
	err = tx.QueryRow(r.Context(), `
		INSERT INTO core.wallets ( user_id, currency)
		VALUES ( $1, $2)
		ON CONFLICT (user_id, currency) DO UPDATE SET updated_at = now()
		RETURNING id::text, balance::float8
	`, userID, currency).Scan(&walletID, &balance)
	if err != nil {
		writeError(w, http.StatusNotFound, "пользователь или кошелёк не найден")
		return
	}
	if balance+amount < 0 {
		writeError(w, http.StatusBadRequest, "списание больше текущего баланса")
		return
	}

	var newBalance float64
	if err := tx.QueryRow(r.Context(), `
		UPDATE core.wallets SET balance = balance + $2, updated_at = now()
		WHERE id = $1 RETURNING balance::float8
	`, walletID, amount).Scan(&newBalance); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}

	txType := "credit"
	if amount < 0 {
		txType = "debit"
	}
	meta, _ := json.Marshal(map[string]any{
		"manual":   true,
		"admin_id": claims.UserID,
		"comment":  comment,
	})
	if _, err := tx.Exec(r.Context(), `
		INSERT INTO core.transactions ( wallet_id, type, amount, description, meta, source_type)
		VALUES ( $1, $2, $3, $4, $5::jsonb, 'admin_adjustment')
	`, walletID, txType, math.Abs(amount), comment, meta); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}

	audit(r.Context(), h.dbOf(r.Context()), claims.UserID, "billing.adjust", "user:"+userID,
		map[string]any{"amount": amount, "currency": currency, "comment": comment})
	h.notifyUser(r.Context(), userID, notify.Event{
		Kind:   notify.KindPaymentReceived,
		Title:  "Баланс изменён администратором",
		Body:   comment,
		Action: h.panelAction("К биллингу", "/billing"),
		Meta:   map[string]any{"amount": amount, "currency": currency},
	})

	writeJSON(w, http.StatusOK, map[string]any{
		"wallet_id": walletID,
		"balance":   newBalance,
		"currency":  currency,
	})
}
