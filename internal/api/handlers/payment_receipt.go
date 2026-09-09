package handlers

import (
	"context"
	"fmt"
	"html"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

type receiptData struct {
	Number     int64
	IssuedAt   time.Time
	PaidAt     time.Time
	Amount     float64
	Currency   string
	Provider   string
	ExternalID string
	Email      string
	Company    map[string]string
}

func (h *Handler) PaymentReceipt(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	paymentID := chi.URLParam(r, "id")
	ctx := r.Context()

	var userID, provider, currency, status, email string
	var externalID *string
	var amount float64
	var createdAt time.Time
	var creditedAt *time.Time
	var number *int64
	err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT p.user_id::text, p.provider, p.currency, p.status, COALESCE(u.email, ''),
		       p.provider_payment_id, p.amount::float8, p.created_at, p.credited_at, p.receipt_number
		FROM core.payments p
		LEFT JOIN core.users u ON u.id = p.user_id
		WHERE p.id = $1 AND p.tenant_id = $2
	`, paymentID, claims.TenantID).Scan(&userID, &provider, &currency, &status, &email,
		&externalID, &amount, &createdAt, &creditedAt, &number)
	if err != nil {
		writeError(w, http.StatusNotFound, "payment not found")
		return
	}
	if userID != claims.UserID && !isStaffRole(claims.Role) {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	if !isPaidStatus(status) {
		writeError(w, http.StatusConflict, "квитанция выдаётся только по оплаченному платежу")
		return
	}

	if number == nil {
		assigned, err := h.assignReceiptNumber(ctx, claims.TenantID, paymentID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "не удалось выдать номер квитанции")
			return
		}
		number = &assigned
	}

	paidAt := createdAt
	if creditedAt != nil {
		paidAt = *creditedAt
	}
	data := receiptData{
		Number:     *number,
		IssuedAt:   time.Now(),
		PaidAt:     paidAt,
		Amount:     amount,
		Currency:   strings.ToUpper(currency),
		Provider:   provider,
		ExternalID: strPtr(externalID),
		Email:      email,
		Company:    h.receiptCompany(ctx, claims.TenantID),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(renderReceipt(data)))
}

func (h *Handler) assignReceiptNumber(ctx context.Context, tenantID, paymentID string) (int64, error) {
	tx, err := h.dbOf(ctx).Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	var number int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO core.document_counters (tenant_id, kind, value)
		VALUES ($1, 'receipt', 1)
		ON CONFLICT (tenant_id, kind) DO UPDATE SET value = core.document_counters.value + 1
		RETURNING value
	`, tenantID).Scan(&number); err != nil {
		return 0, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE core.payments SET receipt_number = $2, receipt_issued_at = now()
		WHERE id = $1 AND receipt_number IS NULL
	`, paymentID, number); err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return number, nil
}

func (h *Handler) receiptCompany(ctx context.Context, tenantID string) map[string]string {
	out := map[string]string{}
	keys := map[string]string{
		"name":    "company.name",
		"inn":     "company.inn",
		"address": "company.address",
		"email":   "company.email",
		"phone":   "company.phone",
		"site":    "app.url",
	}
	for field, key := range keys {
		if v := strings.TrimSpace(h.tenantSettingString(ctx, tenantID, key)); v != "" {
			out[field] = v
		}
	}
	if out["name"] == "" {
		if v := strings.TrimSpace(h.tenantSettingString(ctx, tenantID, "app.name")); v != "" {
			out["name"] = v
		}
	}
	return out
}

func renderReceipt(d receiptData) string {
	esc := html.EscapeString
	rows := []string{
		row("Номер", fmt.Sprintf("№ %d", d.Number)),
		row("Дата платежа", d.PaidAt.Format("02.01.2006 15:04")),
		row("Сумма", fmt.Sprintf("%.2f %s", d.Amount, esc(d.Currency))),
		row("Способ оплаты", esc(d.Provider)),
		row("Плательщик", esc(d.Email)),
	}
	if d.ExternalID != "" {
		rows = append(rows, row("Идентификатор в платёжной системе", esc(d.ExternalID)))
	}

	company := []string{}
	for _, item := range []struct{ label, key string }{
		{"Получатель", "name"},
		{"ИНН", "inn"},
		{"Адрес", "address"},
		{"Email", "email"},
		{"Телефон", "phone"},
		{"Сайт", "site"},
	} {
		if v := d.Company[item.key]; v != "" {
			company = append(company, row(item.label, esc(v)))
		}
	}
	companyBlock := ""
	if len(company) > 0 {
		companyBlock = `<h2>Получатель платежа</h2><table>` + strings.Join(company, "") + `</table>`
	}

	return `<!doctype html>
<html lang="ru"><head><meta charset="utf-8">
<title>Квитанция № ` + fmt.Sprint(d.Number) + `</title>
<style>
  body { font-family: system-ui, -apple-system, "Segoe UI", sans-serif; color: #111; margin: 40px auto; max-width: 640px; }
  h1 { font-size: 20px; margin-bottom: 4px; }
  h2 { font-size: 14px; margin: 24px 0 8px; text-transform: uppercase; letter-spacing: .06em; color: #555; }
  table { width: 100%; border-collapse: collapse; }
  td { padding: 8px 0; border-bottom: 1px solid #eee; vertical-align: top; }
  td:first-child { color: #555; width: 45%; }
  td:last-child { text-align: right; font-variant-numeric: tabular-nums; }
  .note { margin-top: 24px; font-size: 12px; color: #777; line-height: 1.5; }
  @media print { body { margin: 0; } .noprint { display: none; } }
  .noprint button { padding: 8px 14px; font-size: 13px; cursor: pointer; }
</style></head>
<body>
  <div class="noprint" style="margin-bottom:16px"><button onclick="window.print()">Печать / сохранить в PDF</button></div>
  <h1>Квитанция об оплате</h1>
  <div style="color:#555;font-size:13px">Сформирована ` + d.IssuedAt.Format("02.01.2006 15:04") + `</div>
  <h2>Платёж</h2>
  <table>` + strings.Join(rows, "") + `</table>
  ` + companyBlock + `
  <p class="note">
    Документ подтверждает зачисление средств на баланс в панели управления.
    Это не фискальный чек: если хостер работает по 54-ФЗ, чек приходит от
    оператора фискальных данных отдельно.
  </p>
</body></html>`
}

func row(label, value string) string {
	return "<tr><td>" + html.EscapeString(label) + "</td><td>" + value + "</td></tr>"
}
