package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/vortanixapp/panel/internal/api/payments"
)

const (
	receiptOffsetInterval    = time.Minute
	receiptOffsetMaxAttempts = 10
)

func (h *Handler) StartReceiptOffsetSweeper(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(receiptOffsetInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				h.allocateReceiptOffsets(ctx)
				h.sendReceiptOffsets(ctx)
			}
		}
	}()
}

type receiptCharge struct {
	id     string
	wallet string
	amount float64
	at     time.Time
}

type receiptSource struct {
	id        string
	provider  string
	available float64
}

func (h *Handler) allocateReceiptOffsets(ctx context.Context) {
	tx, err := h.dbOf(ctx).Begin(ctx)
	if err != nil {
		return
	}
	defer tx.Rollback(ctx)
	var locked bool
	if err := tx.QueryRow(ctx, `SELECT pg_try_advisory_xact_lock(hashtext('vortanix.receipt_offsets'))`).Scan(&locked); err != nil || !locked {
		return
	}
	rows, err := tx.Query(ctx, `
		SELECT t.id::text, t.wallet_id::text, ABS(t.amount)::float8, t.created_at
		FROM core.transactions t
		WHERE t.type = 'debit' AND t.source_type = ANY($1::text[]) AND t.amount <> 0
		  AND t.created_at > now() - interval '180 days'
		  AND NOT EXISTS (SELECT 1 FROM core.receipt_offsets o WHERE o.transaction_id = t.id)
		  AND EXISTS (
			SELECT 1 FROM core.payments p
			WHERE p.wallet_id = t.wallet_id AND p.credited_at <= t.created_at
			  AND p.meta->'receipt'->>'mode' IN ('advance', 'full_prepayment')
		  )
		ORDER BY t.created_at
		LIMIT 200
	`, accountingServiceSources)
	if err != nil {
		log.Printf("чеки зачёта: списания не выбраны: %v", err)
		return
	}
	var charges []receiptCharge
	for rows.Next() {
		var c receiptCharge
		if rows.Scan(&c.id, &c.wallet, &c.amount, &c.at) == nil {
			charges = append(charges, c)
		}
	}
	rows.Close()
	if len(charges) == 0 {
		return
	}
	for _, c := range charges {
		if err := allocateReceiptCharge(ctx, tx, c); err != nil {
			log.Printf("чеки зачёта: списание %s не распределено: %v", c.id, err)
			return
		}
	}
	if err := tx.Commit(ctx); err != nil {
		log.Printf("чеки зачёта: распределение не сохранено: %v", err)
	}
}

func allocateReceiptCharge(ctx context.Context, tx pgx.Tx, c receiptCharge) error {
	rows, err := tx.Query(ctx, `
		SELECT p.id::text, p.provider,
		       (p.amount - p.refunded_amount - COALESCE((
		           SELECT SUM(o.amount) FROM core.receipt_offsets o WHERE o.payment_id = p.id
		       ), 0))::float8
		FROM core.payments p
		WHERE p.wallet_id = $1 AND p.status IN ('completed', 'refunded') AND p.credited_at <= $2
		  AND p.meta->'receipt'->>'mode' IN ('advance', 'full_prepayment')
		  AND UPPER(COALESCE(NULLIF(p.meta->>'charge_currency', ''), p.currency)) = UPPER(p.currency)
		ORDER BY p.credited_at, p.id
	`, c.wallet, c.at)
	if err != nil {
		return err
	}
	var sources []receiptSource
	for rows.Next() {
		var s receiptSource
		if err := rows.Scan(&s.id, &s.provider, &s.available); err != nil {
			rows.Close()
			return err
		}
		sources = append(sources, s)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	remaining := c.amount
	for _, s := range sources {
		if remaining < 0.005 {
			break
		}
		if s.available < 0.005 {
			continue
		}
		part := math.Round(math.Min(s.available, remaining)*100) / 100
		status := "pending"
		if !payments.ClosingReceiptSupported(s.provider) {
			status = "manual"
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO core.receipt_offsets (transaction_id, payment_id, amount, provider, status)
			VALUES ($1, $2, $3, $4, $5)
		`, c.id, s.id, part, s.provider, status); err != nil {
			return err
		}
		remaining = math.Round((remaining-part)*100) / 100
	}
	if remaining >= 0.005 {
		if _, err := tx.Exec(ctx, `
			INSERT INTO core.receipt_offsets (transaction_id, amount, status) VALUES ($1, $2, 'skipped')
		`, c.id, remaining); err != nil {
			return err
		}
	}
	return nil
}

type receiptOffsetJob struct {
	id            string
	paymentID     string
	transactionID string
	provider      string
	amount        float64
	attempts      int
}

func (h *Handler) sendReceiptOffsets(ctx context.Context) {
	db := h.dbOf(ctx)
	rows, err := db.Query(ctx, `
		UPDATE core.receipt_offsets o
		SET locked_until = now() + interval '10 minutes', attempts = o.attempts + 1
		WHERE o.id IN (
			SELECT id FROM core.receipt_offsets
			WHERE status = 'pending' AND payment_id IS NOT NULL
			  AND (locked_until IS NULL OR locked_until < now())
			ORDER BY created_at
			LIMIT 20
			FOR UPDATE SKIP LOCKED
		)
		RETURNING o.id::text, o.payment_id::text, o.transaction_id::text, o.provider, o.amount::float8, o.attempts
	`)
	if err != nil {
		log.Printf("чеки зачёта: очередь не выбрана: %v", err)
		return
	}
	var jobs []receiptOffsetJob
	for rows.Next() {
		var j receiptOffsetJob
		if rows.Scan(&j.id, &j.paymentID, &j.transactionID, &j.provider, &j.amount, &j.attempts) == nil {
			jobs = append(jobs, j)
		}
	}
	rows.Close()
	if len(jobs) == 0 {
		return
	}
	profile := h.accountingProfile(ctx)
	for _, job := range jobs {
		reference, err := h.sendReceiptOffset(ctx, profile, job)
		if err == nil {
			_, _ = db.Exec(ctx, `
				UPDATE core.receipt_offsets
				SET status = 'sent', reference = $2, error = '', sent_at = now(), locked_until = NULL
				WHERE id = $1
			`, job.id, reference)
			continue
		}
		status := "pending"
		if job.attempts >= receiptOffsetMaxAttempts {
			status = "failed"
		}
		_, _ = db.Exec(ctx, `
			UPDATE core.receipt_offsets
			SET status = $2, error = $3, locked_until = now() + make_interval(mins => $4)
			WHERE id = $1
		`, job.id, status, clipText(err.Error(), 500), job.attempts*5)
	}
}

func (h *Handler) sendReceiptOffset(ctx context.Context, profile accountingProfile, job receiptOffsetJob) (string, error) {
	var providerPaymentID, userID, currency, source, server, tariff string
	var invoice int64
	var receiptRaw []byte
	if err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT COALESCE(p.provider_payment_id, ''), p.invoice_no, p.user_id::text,
		       UPPER(COALESCE(NULLIF(p.meta->>'charge_currency', ''), p.currency)), p.meta->'receipt',
		       COALESCE(t.source_type, ''), COALESCE(s.name, ''), COALESCE(tr.name, '')
		FROM core.payments p
		JOIN core.transactions t ON t.id = $2
		LEFT JOIN core.servers s ON s.id = t.source_id
		LEFT JOIN core.tariffs tr ON tr.id = s.tariff_id
		WHERE p.id = $1
	`, job.paymentID, job.transactionID).Scan(&providerPaymentID, &invoice, &userID, &currency, &receiptRaw,
		&source, &server, &tariff); err != nil {
		return "", err
	}
	var receipt payments.Receipt
	if len(receiptRaw) == 0 || json.Unmarshal(receiptRaw, &receipt) != nil {
		return "", fmt.Errorf("в платеже нет параметров кассового чека")
	}
	row, err := h.loadPaymentProvider(ctx, job.provider)
	if err != nil || !row.Exists || row.Unreadable {
		return "", fmt.Errorf("настройки кассы %s недоступны", job.provider)
	}
	sendCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	return payments.SendClosingReceipt(sendCtx, job.provider, row.Config, payments.ClosingReceiptInput{
		OffsetID:          job.id,
		PaymentID:         job.paymentID,
		ProviderPaymentID: providerPaymentID,
		InvoiceNo:         invoice,
		UserID:            userID,
		Amount:            job.amount,
		Currency:          currency,
		Item:              accountingServiceTitle(source, server, tariff),
		SellerINN:         profile.INN,
		Receipt:           receipt,
	})
}

func (h *Handler) AdminReceiptOffsets(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	db := h.dbOf(ctx)
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	rows, err := db.Query(ctx, `
		SELECT o.id::text, o.status, o.amount::float8, o.provider, o.attempts, o.error, o.reference,
		       o.created_at, o.sent_at, COALESCE(p.invoice_no, 0), COALESCE(u.email, ''),
		       COALESCE(t.source_type, ''), COALESCE(s.name, ''), COALESCE(tr.name, ''), t.created_at
		FROM core.receipt_offsets o
		LEFT JOIN core.payments p ON p.id = o.payment_id
		LEFT JOIN core.users u ON u.id = p.user_id
		LEFT JOIN core.transactions t ON t.id = o.transaction_id
		LEFT JOIN core.servers s ON s.id = t.source_id
		LEFT JOIN core.tariffs tr ON tr.id = s.tariff_id
		WHERE ($1 = '' AND o.status <> 'skipped') OR o.status = $1
		ORDER BY o.created_at DESC
		LIMIT 300
	`, status)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()
	list := []map[string]any{}
	for rows.Next() {
		var id, st, provider, errText, reference, email, source, server, tariff string
		var amount float64
		var attempts int
		var invoice int64
		var createdAt time.Time
		var sentAt, chargedAt *time.Time
		if rows.Scan(&id, &st, &amount, &provider, &attempts, &errText, &reference, &createdAt, &sentAt,
			&invoice, &email, &source, &server, &tariff, &chargedAt) != nil {
			continue
		}
		list = append(list, map[string]any{
			"id": id, "status": st, "amount": amount, "provider": provider, "provider_name": payments.Name(provider),
			"attempts": attempts, "error": errText, "reference": reference, "created_at": createdAt,
			"sent_at": isoOrNil(sentAt), "invoice_no": invoice, "user_email": email,
			"service": accountingServiceTitle(source, server, tariff), "charged_at": isoOrNil(chargedAt),
		})
	}
	counts := map[string]int{}
	if countRows, err := db.Query(ctx, `SELECT status, COUNT(*) FROM core.receipt_offsets GROUP BY status`); err == nil {
		for countRows.Next() {
			var st string
			var n int
			if countRows.Scan(&st, &n) == nil {
				counts[st] = n
			}
		}
		countRows.Close()
	}
	writeJSON(w, http.StatusOK, map[string]any{"offsets": list, "counts": counts})
}

func (h *Handler) AdminReceiptOffsetRetry(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ctx := r.Context()
	id := chi.URLParam(r, "id")
	var provider string
	if err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT provider FROM core.receipt_offsets WHERE id = $1 AND payment_id IS NOT NULL
	`, id).Scan(&provider); err != nil {
		writeError(w, http.StatusNotFound, "Чек не найден")
		return
	}
	if !payments.ClosingReceiptSupported(provider) {
		writeError(w, http.StatusConflict, "Касса "+payments.Name(provider)+" не принимает чеки зачёта по API — оформите чек в её личном кабинете")
		return
	}
	tag, err := h.dbOf(ctx).Exec(ctx, `
		UPDATE core.receipt_offsets SET status = 'pending', attempts = 0, error = '', locked_until = NULL
		WHERE id = $1 AND status IN ('failed', 'pending')
	`, id)
	if err != nil || tag.RowsAffected() == 0 {
		writeError(w, http.StatusConflict, "Повторить можно только неотправленный чек")
		return
	}
	audit(ctx, h.dbOf(ctx), claims.UserID, "receipt_offset.retry", "receipt_offset:"+id, nil)
	writeJSON(w, http.StatusOK, map[string]string{"status": "pending"})
}

func (h *Handler) AdminReceiptOffsetMarkSent(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var body struct {
		Reference string `json:"reference"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	reference := strings.TrimSpace(body.Reference)
	if reference == "" {
		writeError(w, http.StatusBadRequest, "Укажите номер чека или фискальный признак")
		return
	}
	ctx := r.Context()
	id := chi.URLParam(r, "id")
	tag, err := h.dbOf(ctx).Exec(ctx, `
		UPDATE core.receipt_offsets SET status = 'sent', reference = $2, error = '', sent_at = now(), locked_until = NULL
		WHERE id = $1 AND status IN ('manual', 'failed', 'pending')
	`, id, reference)
	if err != nil || tag.RowsAffected() == 0 {
		writeError(w, http.StatusConflict, "Чек уже отмечен отправленным")
		return
	}
	audit(ctx, h.dbOf(ctx), claims.UserID, "receipt_offset.mark_sent", "receipt_offset:"+id, map[string]any{"reference": reference})
	writeJSON(w, http.StatusOK, map[string]string{"status": "sent"})
}
