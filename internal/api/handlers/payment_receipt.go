package handlers

import (
	"context"
	"fmt"
	"html"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/vortanixapp/panel/internal/api/payments"
)

type receiptData struct {
	Number     int64
	Invoice    int64
	IssuedAt   time.Time
	PaidAt     time.Time
	Amount     float64
	Refunded   float64
	Currency   string
	Provider   string
	ExternalID string
	Fiscal     bool
	Payer      billingPayer
	Company    accountingProfile
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
	var amount, refunded float64
	var createdAt time.Time
	var creditedAt *time.Time
	var number *int64
	var invoice int64
	var fiscal bool
	err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT p.user_id::text, p.provider, p.currency, p.status, COALESCE(u.email, ''),
		       p.provider_payment_id, p.amount::float8, p.created_at, p.credited_at, p.receipt_number,
		       p.invoice_no, p.refunded_amount::float8, p.meta ? 'receipt'
		FROM core.payments p
		LEFT JOIN core.users u ON u.id = p.user_id
		WHERE p.id = $1
	`, paymentID).Scan(&userID, &provider, &currency, &status, &email,
		&externalID, &amount, &createdAt, &creditedAt, &number, &invoice, &refunded, &fiscal)
	if err != nil {
		writeError(w, http.StatusNotFound, "payment not found")
		return
	}
	if userID != claims.UserID && !isStaffRole(claims.Role) {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	if !isPaidStatus(status) && status != "refunded" {
		writeError(w, http.StatusConflict, "квитанция выдаётся только по оплаченному платежу")
		return
	}

	if number == nil {
		assigned, err := h.assignReceiptNumber(ctx, paymentID)
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
	payer, err := h.billingPayer(ctx, userID)
	if err != nil {
		payer = billingPayer{UserID: userID, Email: email, PayerType: "person"}
	}
	data := receiptData{
		Number:     *number,
		Invoice:    invoice,
		IssuedAt:   time.Now(),
		PaidAt:     paidAt,
		Amount:     amount,
		Refunded:   refunded,
		Currency:   strings.ToUpper(currency),
		Provider:   provider,
		ExternalID: strPtr(externalID),
		Fiscal:     fiscal,
		Payer:      payer,
		Company:    h.accountingProfile(ctx),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(renderReceipt(data)))
}

func (h *Handler) assignReceiptNumber(ctx context.Context, paymentID string) (int64, error) {
	tx, err := h.dbOf(ctx).Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	var number int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO core.document_counters (kind, value)
		VALUES ('receipt', 1)
		ON CONFLICT (kind) DO UPDATE SET value = core.document_counters.value + 1
		RETURNING value
	`).Scan(&number); err != nil {
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

func renderReceipt(d receiptData) string {
	esc := html.EscapeString
	loc := d.Company.Location
	method := payments.Name(d.Provider)
	rows := []string{
		row("Номер", fmt.Sprintf("№ %d", d.Number)),
		row("Счёт", fmt.Sprintf("№ %d", d.Invoice)),
		row("Дата платежа", d.PaidAt.In(loc).Format("02.01.2006 15:04")),
		row("Сумма", esc(formatMoneyRu(d.Amount)+" "+d.Currency)),
		row("Способ оплаты", esc(method)),
		row("Плательщик", esc(d.Payer.title())),
	}
	if d.Payer.INN != "" {
		rows = append(rows, row("ИНН плательщика", esc(d.Payer.INN)))
	}
	if d.Payer.Email != "" && d.Payer.Email != d.Payer.title() {
		rows = append(rows, row("Email", esc(d.Payer.Email)))
	}
	if d.ExternalID != "" {
		rows = append(rows, row("Идентификатор в платёжной системе", esc(d.ExternalID)))
	}
	if d.Refunded > 0 {
		rows = append(rows, row("Возвращено", esc(formatMoneyRu(d.Refunded)+" "+d.Currency)))
	}
	if d.Fiscal {
		rows = append(rows, row("Кассовый чек", esc("передан в "+method)))
	}

	c := d.Company
	company := []string{}
	for _, item := range []struct{ label, value string }{
		{"Получатель", firstNonEmpty(c.FullName, c.Name, c.AppName)},
		{"ИНН", c.INN},
		{"КПП", c.KPP},
		{ogrnLabel(c.OGRN), c.OGRN},
		{"Адрес", c.Address},
		{"Расчётный счёт", c.BankAccount},
		{"Банк", c.BankName},
		{"БИК", c.BankBIK},
		{"Корреспондентский счёт", c.BankCorrAccount},
		{"Email", c.Email},
		{"Телефон", c.Phone},
		{"Сайт", c.Site},
	} {
		if item.value != "" {
			company = append(company, row(item.label, esc(item.value)))
		}
	}
	companyBlock := ""
	if len(company) > 0 {
		companyBlock = `<h2>Получатель платежа</h2><table>` + strings.Join(company, "") + `</table>`
	}

	note := "Документ подтверждает зачисление средств на баланс в панели управления. Это не кассовый чек: если продавец работает по 54-ФЗ, чек приходит от оператора фискальных данных отдельно."
	switch {
	case d.Fiscal:
		note = "Документ подтверждает зачисление средств на баланс в панели управления. Кассовый чек по 54-ФЗ формирует онлайн-касса платёжной системы и отправляет на email плательщика."
	case c.TaxSystem == "npd":
		note = "Документ подтверждает зачисление средств на баланс в панели управления. Продавец применяет налог на профессиональный доход: чек формируется в приложении «Мой налог»."
	}

	return `<!doctype html>
<html lang="ru"><head><meta charset="utf-8">
<title>Квитанция № ` + fmt.Sprint(d.Number) + `</title>
<style>
  body { font-family: system-ui, -apple-system, "Segoe UI", sans-serif; color: #111; margin: 40px auto; max-width: 640px; padding: 0 16px; }
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
  <div style="color:#555;font-size:13px">Сформирована ` + d.IssuedAt.In(loc).Format("02.01.2006 15:04") + `</div>
  <h2>Платёж</h2>
  <table>` + strings.Join(rows, "") + `</table>
  ` + companyBlock + `
  <p class="note">` + esc(note) + `</p>
</body></html>`
}

func row(label, value string) string {
	return "<tr><td>" + html.EscapeString(label) + "</td><td>" + value + "</td></tr>"
}
