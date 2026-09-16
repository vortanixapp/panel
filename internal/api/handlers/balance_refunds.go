package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/vortanixapp/panel/internal/api/payments"
	"github.com/vortanixapp/panel/pkg/i18n"
	"github.com/vortanixapp/panel/pkg/notify"
)

type balanceRefundRequest struct {
	ID                string     `json:"id"`
	Number            int64      `json:"number"`
	UserID            string     `json:"user_id"`
	UserEmail         string     `json:"user_email"`
	UserName          string     `json:"user_name"`
	Currency          string     `json:"currency"`
	Amount            float64    `json:"amount"`
	Refunded          float64    `json:"refunded"`
	Method            string     `json:"method"`
	Recipient         string     `json:"recipient"`
	BankAccount       string     `json:"bank_account"`
	BankBIK           string     `json:"bank_bik"`
	BankName          string     `json:"bank_name"`
	Reason            string     `json:"reason"`
	Status            string     `json:"status"`
	AdminNote         string     `json:"admin_note"`
	Reference         string     `json:"reference"`
	CreatedAt         time.Time  `json:"created_at"`
	ProcessedAt       *time.Time `json:"processed_at"`
	Balance           float64    `json:"balance"`
	GatewayRefundable float64    `json:"gateway_refundable"`
}

const balanceRefundSelect = `
	SELECT r.id::text, r.number, r.user_id::text, u.email, ` + payerNameSQL + `, r.currency, r.amount::float8,
	       r.refunded::float8, r.method, r.recipient, r.bank_account, r.bank_bik, r.bank_name, r.reason, r.status,
	       r.admin_note, r.reference, r.created_at, r.processed_at,
	       COALESCE((SELECT w.balance FROM core.wallets w
	                 WHERE w.user_id = r.user_id AND UPPER(w.currency) = r.currency LIMIT 1), 0)::float8
	FROM core.balance_refund_requests r
	JOIN core.users u ON u.id = r.user_id` + payerJoinsSQL

func scanBalanceRefund(row pgx.Row) (balanceRefundRequest, error) {
	var q balanceRefundRequest
	err := row.Scan(&q.ID, &q.Number, &q.UserID, &q.UserEmail, &q.UserName, &q.Currency, &q.Amount, &q.Refunded,
		&q.Method, &q.Recipient, &q.BankAccount, &q.BankBIK, &q.BankName, &q.Reason, &q.Status, &q.AdminNote,
		&q.Reference, &q.CreatedAt, &q.ProcessedAt, &q.Balance)
	return q, err
}

type refundablePayment struct {
	ID        string
	Provider  string
	Available float64
}

func (h *Handler) gatewayRefundable(ctx context.Context, userID, currency string) (float64, []refundablePayment) {
	rows, err := h.dbOf(ctx).Query(ctx, `
		SELECT p.id::text, p.provider, (p.amount - p.refunded_amount)::float8
		FROM core.payments p
		WHERE p.user_id = $1 AND UPPER(p.currency) = $2 AND p.status = 'completed'
		  AND p.amount > p.refunded_amount AND COALESCE(p.provider_payment_id, '') <> ''
		ORDER BY p.credited_at DESC NULLS LAST
	`, userID, currency)
	if err != nil {
		return 0, nil
	}
	defer rows.Close()
	var total float64
	var list []refundablePayment
	for rows.Next() {
		var p refundablePayment
		if rows.Scan(&p.ID, &p.Provider, &p.Available) == nil && payments.RefundSupported(p.Provider) {
			total += p.Available
			list = append(list, p)
		}
	}
	return total, list
}

func (h *Handler) BillingRefundRequests(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ctx := r.Context()
	rows, err := h.dbOf(ctx).Query(ctx, balanceRefundSelect+`
		WHERE r.user_id = $1
		ORDER BY r.created_at DESC
		LIMIT 50
	`, claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()
	requests := []balanceRefundRequest{}
	for rows.Next() {
		q, err := scanBalanceRefund(rows)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "database error")
			return
		}
		requests = append(requests, q)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"requests": requests,
		"wallets":  h.listUserWallets(r, claims.UserID),
	})
}

func (h *Handler) BillingRefundRequestCreate(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var body struct {
		Currency    string  `json:"currency"`
		Amount      float64 `json:"amount"`
		Method      string  `json:"method"`
		Recipient   string  `json:"recipient"`
		BankAccount string  `json:"bank_account"`
		BankBIK     string  `json:"bank_bik"`
		BankName    string  `json:"bank_name"`
		Reason      string  `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	ctx := r.Context()
	currency := strings.ToUpper(strings.TrimSpace(body.Currency))
	amount := math.Round(body.Amount*100) / 100
	recipient := strings.TrimSpace(body.Recipient)
	account := strings.TrimSpace(body.BankAccount)
	bik := strings.TrimSpace(body.BankBIK)
	_, balance, found := h.walletForUser(ctx, claims.UserID, currency)
	switch {
	case !found:
		writeError(w, http.StatusBadRequest, "кошелёк в этой валюте не найден")
		return
	case amount <= 0:
		writeError(w, http.StatusBadRequest, "сумма возврата должна быть больше нуля")
		return
	case amount > balance+0.009:
		writeError(w, http.StatusBadRequest, "на балансе меньше: доступно "+formatMoney(balance)+" "+currency)
		return
	case !slices.Contains([]string{"original", "bank"}, body.Method):
		writeError(w, http.StatusBadRequest, "выберите способ возврата")
		return
	case body.Method == "bank" && recipient == "":
		writeError(w, http.StatusBadRequest, "укажите получателя: ФИО или наименование организации")
		return
	case body.Method == "bank" && !digitsOnly(account, 20):
		writeError(w, http.StatusBadRequest, "номер счёта получателя состоит из 20 цифр")
		return
	case body.Method == "bank" && !digitsOnly(bik, 9):
		writeError(w, http.StatusBadRequest, "БИК банка состоит из 9 цифр")
		return
	case utf8.RuneCountInString(body.Reason) > 1000 || utf8.RuneCountInString(recipient) > 300:
		writeError(w, http.StatusBadRequest, "слишком длинный текст заявки")
		return
	}
	if body.Method != "bank" {
		recipient, account, bik, body.BankName = "", "", "", ""
	}
	var id string
	var number int64
	err := h.dbOf(ctx).QueryRow(ctx, `
		INSERT INTO core.balance_refund_requests (user_id, currency, amount, method, recipient, bank_account, bank_bik, bank_name, reason)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id::text, number
	`, claims.UserID, currency, amount, body.Method, recipient, account, bik,
		strings.TrimSpace(body.BankName), strings.TrimSpace(body.Reason)).Scan(&id, &number)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			writeError(w, http.StatusConflict, "заявка на возврат в этой валюте уже ждёт обработки")
			return
		}
		writeError(w, http.StatusInternalServerError, "не удалось создать заявку")
		return
	}
	audit(ctx, h.dbOf(ctx), claims.UserID, "refund.request", "refund_request:"+id, map[string]any{
		"amount": amount, "currency": currency, "method": body.Method,
	})
	h.auditAlert(ctx, claims.UserID, claims.Email, "refund.request",
		"заявка № "+strconv.FormatInt(number, 10)+", "+formatMoney(amount)+" "+currency)
	h.BillingRefundRequests(w, r)
}

func (h *Handler) BillingRefundRequestCancel(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ctx := r.Context()
	tag, err := h.dbOf(ctx).Exec(ctx, `
		UPDATE core.balance_refund_requests SET status = 'cancelled', processed_at = now()
		WHERE id = $1 AND user_id = $2 AND status = 'pending' AND refunded = 0
	`, chi.URLParam(r, "id"), claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusConflict, "заявку уже нельзя отменить")
		return
	}
	h.BillingRefundRequests(w, r)
}

func (h *Handler) AdminRefundRequests(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	if status == "" {
		status = "pending"
	}
	rows, err := h.dbOf(ctx).Query(ctx, balanceRefundSelect+`
		WHERE ($1 = 'all' OR r.status = $1)
		ORDER BY (r.status <> 'pending'), r.created_at DESC
		LIMIT 300
	`, status)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	requests := []balanceRefundRequest{}
	for rows.Next() {
		q, err := scanBalanceRefund(rows)
		if err != nil {
			rows.Close()
			writeError(w, http.StatusInternalServerError, "database error")
			return
		}
		requests = append(requests, q)
	}
	rows.Close()
	for i := range requests {
		if requests[i].Status == "pending" {
			requests[i].GatewayRefundable, _ = h.gatewayRefundable(ctx, requests[i].UserID, requests[i].Currency)
		}
	}
	var pending int
	_ = h.dbOf(ctx).QueryRow(ctx, `SELECT COUNT(*) FROM core.balance_refund_requests WHERE status = 'pending'`).Scan(&pending)
	writeJSON(w, http.StatusOK, map[string]any{"requests": requests, "pending": pending})
}

func (h *Handler) AdminRefundRequestComplete(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var body struct {
		Mode      string `json:"mode"`
		Reference string `json:"reference"`
		Note      string `json:"note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	ctx := r.Context()
	db := h.dbOf(ctx)
	id := chi.URLParam(r, "id")
	q, err := scanBalanceRefund(db.QueryRow(ctx, balanceRefundSelect+` WHERE r.id = $1`, id))
	if err != nil {
		writeError(w, http.StatusNotFound, "заявка не найдена")
		return
	}
	if q.Status != "pending" {
		writeError(w, http.StatusConflict, "заявка уже обработана")
		return
	}
	remaining := math.Round((q.Amount-q.Refunded)*100) / 100
	note := strings.TrimSpace(body.Note)
	reference := strings.TrimSpace(body.Reference)
	numberText := strconv.FormatInt(q.Number, 10)

	switch body.Mode {
	case "gateway":
		_, list := h.gatewayRefundable(ctx, q.UserID, q.Currency)
		done := 0.0
		failure := ""
		for _, p := range list {
			left := math.Round((remaining-done)*100) / 100
			if left < 0.005 {
				break
			}
			part := math.Round(math.Min(p.Available, left)*100) / 100
			if _, err := h.refundPayment(ctx, claims.UserID, claims.Email, p.ID, part, "возврат остатка баланса по заявке № "+numberText); err != nil {
				failure = err.Error()
				break
			}
			done += part
		}
		if done < 0.005 {
			msg := "через платёжные системы вернуть нечего — верните остаток переводом на счёт клиента"
			if failure != "" {
				msg = failure
			}
			writeError(w, http.StatusConflict, msg)
			return
		}
		newRefunded := math.Round((q.Refunded+done)*100) / 100
		status := "pending"
		if newRefunded+0.009 >= q.Amount {
			status = "completed"
		}
		if _, err := db.Exec(ctx, `
			UPDATE core.balance_refund_requests
			SET refunded = $2, status = $3, admin_note = $4,
			    reference = CASE WHEN $5 <> '' THEN $5 ELSE reference END,
			    processed_by = $6, processed_at = CASE WHEN $3 = 'completed' THEN now() ELSE processed_at END
			WHERE id = $1
		`, q.ID, newRefunded, status, note, reference, claims.UserID); err != nil {
			writeError(w, http.StatusInternalServerError, "database error")
			return
		}
		audit(ctx, db, claims.UserID, "refund.request.gateway", "refund_request:"+q.ID, map[string]any{"refunded": done})
		if status == "completed" {
			h.notifyRefundDone(ctx, q, newRefunded)
			writeJSON(w, http.StatusOK, map[string]any{"status": status, "refunded": newRefunded})
			return
		}
		message := "возвращено " + formatMoney(newRefunded) + " из " + formatMoney(q.Amount) + " " + q.Currency + ": остаток верните переводом"
		if failure != "" {
			message += " (" + failure + ")"
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": status, "refunded": newRefunded, "message": message})
	case "manual":
		if reference == "" {
			writeError(w, http.StatusBadRequest, "укажите номер и дату платёжного поручения")
			return
		}
		tx, err := db.Begin(ctx)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "database error")
			return
		}
		defer tx.Rollback(ctx)
		var walletID string
		var balance float64
		if err := tx.QueryRow(ctx, `
			SELECT id::text, balance::float8 FROM core.wallets
			WHERE user_id = $1 AND UPPER(currency) = $2
			LIMIT 1 FOR UPDATE
		`, q.UserID, q.Currency).Scan(&walletID, &balance); err != nil {
			writeError(w, http.StatusConflict, "кошелёк клиента не найден")
			return
		}
		if balance+0.009 < remaining {
			writeError(w, http.StatusConflict, "на балансе клиента меньше суммы заявки: "+formatMoney(balance)+" "+q.Currency)
			return
		}
		if _, err := tx.Exec(ctx, `
			UPDATE core.wallets SET balance = balance - $2, updated_at = now() WHERE id = $1
		`, walletID, remaining); err != nil {
			writeError(w, http.StatusInternalServerError, "database error")
			return
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO core.transactions (wallet_id, type, amount, description, source_type, source_id)
			VALUES ($1, 'debit', $2, $3, 'balance_refund', $4::uuid)
		`, walletID, remaining, "Возврат остатка баланса по заявке № "+numberText+", "+reference, q.ID); err != nil {
			writeError(w, http.StatusInternalServerError, "database error")
			return
		}
		tag, err := tx.Exec(ctx, `
			UPDATE core.balance_refund_requests
			SET refunded = amount, status = 'completed', reference = $2, admin_note = $3,
			    processed_by = $4, processed_at = now()
			WHERE id = $1 AND status = 'pending'
		`, q.ID, reference, note, claims.UserID)
		if err != nil || tag.RowsAffected() == 0 {
			writeError(w, http.StatusConflict, "заявка уже обработана")
			return
		}
		if err := tx.Commit(ctx); err != nil {
			writeError(w, http.StatusInternalServerError, "database error")
			return
		}
		audit(ctx, db, claims.UserID, "refund.request.manual", "refund_request:"+q.ID, map[string]any{
			"amount": remaining, "reference": reference,
		})
		h.notifyRefundDone(ctx, q, q.Amount)
		writeJSON(w, http.StatusOK, map[string]any{"status": "completed", "refunded": q.Amount})
	default:
		writeError(w, http.StatusBadRequest, "выберите способ возврата")
	}
}

func (h *Handler) notifyRefundDone(ctx context.Context, q balanceRefundRequest, refunded float64) {
	h.notifyUser(ctx, q.UserID, notify.Event{
		Kind:  notify.KindPaymentRefunded,
		Title: i18n.Key("notify.refund_request_done.title"),
		Body: i18n.Key("notify.refund_request_done.body", i18n.Params{
			"number": strconv.FormatInt(q.Number, 10), "amount": formatMoney(refunded), "currency": q.Currency,
		}),
		Action:    h.panelAction("notify.action.balance", "/billing"),
		Meta:      map[string]any{"refund_request_id": q.ID},
		DedupeKey: "refund_request.done:" + q.ID,
	})
}

func (h *Handler) AdminRefundRequestReject(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var body struct {
		Note string `json:"note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	note := strings.TrimSpace(body.Note)
	if note == "" {
		writeError(w, http.StatusBadRequest, "укажите причину отказа — клиент увидит её в уведомлении")
		return
	}
	ctx := r.Context()
	db := h.dbOf(ctx)
	q, err := scanBalanceRefund(db.QueryRow(ctx, balanceRefundSelect+` WHERE r.id = $1`, chi.URLParam(r, "id")))
	if err != nil {
		writeError(w, http.StatusNotFound, "заявка не найдена")
		return
	}
	tag, err := db.Exec(ctx, `
		UPDATE core.balance_refund_requests
		SET status = 'rejected', admin_note = $2, processed_by = $3, processed_at = now()
		WHERE id = $1 AND status = 'pending'
	`, q.ID, note, claims.UserID)
	if err != nil || tag.RowsAffected() == 0 {
		writeError(w, http.StatusConflict, "заявка уже обработана")
		return
	}
	audit(ctx, db, claims.UserID, "refund.request.reject", "refund_request:"+q.ID, map[string]any{"note": note})
	h.notifyUser(ctx, q.UserID, notify.Event{
		Kind:  notify.KindRefundRejected,
		Title: i18n.Key("notify.refund_request_rejected.title"),
		Body: i18n.Key("notify.refund_request_rejected.body", i18n.Params{
			"number": strconv.FormatInt(q.Number, 10), "reason": note,
		}),
		Action:    h.panelAction("notify.action.balance", "/billing"),
		Meta:      map[string]any{"refund_request_id": q.ID},
		DedupeKey: "refund_request.rejected:" + q.ID,
	})
	writeJSON(w, http.StatusOK, map[string]string{"status": "rejected"})
}
