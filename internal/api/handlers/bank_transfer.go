package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"golang.org/x/text/encoding/charmap"

	"github.com/vortanixapp/panel/internal/api/payments"
)

const bankStatementLimit = 5 << 20

var invoiceNumberPattern = regexp.MustCompile(`\d{6,12}`)

func (h *Handler) PaymentInvoice(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ctx := r.Context()
	var userID, provider, status, currency string
	var amount float64
	var invoice int64
	var createdAt time.Time
	err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT p.user_id::text, p.provider, p.status, `+paymentPaidCurrencySQL+`, `+paymentPaidAmountSQL+`::float8,
		       p.invoice_no, p.created_at
		FROM core.payments p WHERE p.id = $1
	`, chi.URLParam(r, "id")).Scan(&userID, &provider, &status, &currency, &amount, &invoice, &createdAt)
	if err != nil {
		writeError(w, http.StatusNotFound, "payment not found")
		return
	}
	if userID != claims.UserID && !isStaffRole(claims.Role) {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	if provider != "bank" {
		writeError(w, http.StatusConflict, "счёт на оплату выдаётся только для оплаты банковским переводом")
		return
	}
	if status == "failed" || status == "cancelled" {
		writeError(w, http.StatusConflict, "платёж отменён — создайте новое пополнение")
		return
	}
	profile := h.accountingProfile(ctx)
	if profile.BankAccount == "" {
		if row, err := h.loadPaymentProvider(ctx, "bank"); err == nil && row.Exists {
			info := payments.BankInstructions(row.Config)
			profile.BankName = firstNonEmpty(profile.BankName, info["bank_name"])
			profile.BankAccount = info["account_number"]
			profile.BankBIK = firstNonEmpty(profile.BankBIK, info["swift_bic"])
			if profile.FullName == "" && profile.Name == "" {
				profile.Name = info["account_holder"]
			}
		}
	}
	payer, err := h.billingPayer(ctx, userID)
	if err != nil {
		payer = billingPayer{UserID: userID, PayerType: "person"}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write([]byte(renderPaymentInvoice(profile, payer, invoice, createdAt, amount, currency)))
}

func renderPaymentInvoice(p accountingProfile, payer billingPayer, invoice int64, issued time.Time, amount float64, currency string) string {
	esc := html.EscapeString
	local := issued.In(p.Location)
	number := strconv.FormatInt(invoice, 10)
	total := formatMoneyRu(amount)
	vat := p.vatRate()
	vatRow := `<tr><td colspan="5" class="r">Без налога (НДС)</td><td class="r">—</td></tr>`
	vatText := "НДС не облагается"
	if vat != "none" {
		tax := vatIncluded(vat, amount)
		vatRow = `<tr><td colspan="5" class="r">В том числе ` + esc(vatLabel(vat)) + `</td><td class="r">` + formatMoneyRu(tax) + `</td></tr>`
		vatText = "В т. ч. " + vatLabel(vat) + " — " + formatMoneyRu(tax) + " " + currency
	}
	words := ""
	if currency == "RUB" {
		words = `<p><b>` + esc(rublesInWords(amount)) + `</b></p>`
	}
	seller := firstNonEmpty(p.FullName, p.Name, p.AppName)
	item := firstNonEmpty(p.ReceiptItem, accountingDefaultReceiptItem) + " (пополнение лицевого счёта)"
	purpose := "Оплата по счёту № " + number + " от " + local.Format("02.01.2006") + ". " + vatText
	signer := firstNonEmpty(strings.TrimSpace(p.SignerPosition+" "+p.SignerName), "подпись")
	title := "Счёт на оплату № " + number + " от " + russianDate(local)
	body := `<table class="items">
    <tr><td colspan="2" rowspan="2">` + esc(p.BankName) + `<br><small>Банк получателя</small></td><td>БИК</td><td>` + esc(p.BankBIK) + `</td></tr>
    <tr><td>Сч. №</td><td>` + esc(p.BankCorrAccount) + `</td></tr>
    <tr><td>ИНН ` + esc(p.INN) + `</td><td>КПП ` + esc(p.KPP) + `</td><td rowspan="2">Сч. №</td><td rowspan="2">` + esc(p.BankAccount) + `</td></tr>
    <tr><td colspan="2">` + esc(seller) + `<br><small>Получатель</small></td></tr>
  </table>
  <h1>` + esc(title) + `</h1>
  <table class="parties">
    <tr><td>Поставщик</td><td>` + esc(p.partyLine()) + `</td></tr>
    <tr><td>Покупатель</td><td>` + esc(payer.partyLine()) + `</td></tr>
  </table>
  <table class="items">
    <thead><tr><th>№</th><th>Наименование</th><th>Кол-во</th><th>Ед.</th><th>Цена, ` + esc(currency) + `</th><th>Сумма, ` + esc(currency) + `</th></tr></thead>
    <tbody><tr><td class="c">1</td><td>` + esc(item) + `</td><td class="c">1</td><td class="c">шт</td><td class="r">` + total + `</td><td class="r">` + total + `</td></tr></tbody>
    <tfoot>
      <tr class="strong"><td colspan="5" class="r">Итого</td><td class="r">` + total + `</td></tr>
      ` + vatRow + `
      <tr class="strong"><td colspan="5" class="r">Всего к оплате</td><td class="r">` + total + `</td></tr>
    </tfoot>
  </table>
  <p>Всего наименований 1, на сумму ` + total + ` ` + esc(currency) + `.</p>
  ` + words + `
  <p><b>Назначение платежа:</b> ` + esc(purpose) + `</p>
  <p>Баланс пополняется после поступления денег на расчётный счёт. Укажите номер счёта в назначении платежа — по нему поступление зачисляется на ваш баланс.</p>
  <table class="signs"><tr><td><div class="role">Руководитель</div><div class="line">` + esc(signer) + `</div></td><td></td></tr></table>`
	return documentHTML(title, body)
}

type bankStatementLine struct {
	Fingerprint  string  `json:"fingerprint"`
	Number       string  `json:"number"`
	Date         string  `json:"date"`
	Amount       float64 `json:"amount"`
	PayerName    string  `json:"payer_name"`
	PayerINN     string  `json:"payer_inn"`
	PayerAccount string  `json:"payer_account"`
	Purpose      string  `json:"purpose"`
	Status       string  `json:"status"`
	PaymentID    string  `json:"payment_id,omitempty"`
	InvoiceNo    int64   `json:"invoice_no,omitempty"`
	UserEmail    string  `json:"user_email,omitempty"`
	Note         string  `json:"note,omitempty"`
}

func parseClientBankExchange(data []byte) ([]map[string]string, error) {
	if !utf8.Valid(data) {
		decoded, err := charmap.Windows1251.NewDecoder().Bytes(data)
		if err != nil {
			return nil, err
		}
		data = decoded
	}
	text := strings.ReplaceAll(string(bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})), string([]byte{13, 10}), string([]byte{10}))
	if !strings.HasPrefix(strings.TrimSpace(text), "1CClientBankExchange") {
		return nil, fmt.Errorf("файл не похож на выписку в формате 1CClientBankExchange")
	}
	var docs []map[string]string
	var current map[string]string
	for _, raw := range strings.Split(text, "\n") {
		line := strings.TrimSpace(raw)
		switch {
		case strings.HasPrefix(line, "СекцияДокумент"):
			current = map[string]string{}
		case line == "КонецДокумента":
			if current != nil {
				docs = append(docs, current)
			}
			current = nil
		case current != nil:
			if key, value, ok := strings.Cut(line, "="); ok {
				current[key] = strings.TrimSpace(value)
			}
		}
	}
	return docs, nil
}

func (h *Handler) AdminBankStatementPreview(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, bankStatementLimit+(1<<20))
	if err := r.ParseMultipartForm(bankStatementLimit + (1 << 20)); err != nil {
		writeError(w, http.StatusBadRequest, "загрузите файл выписки размером до 5 МБ")
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "файл выписки не получен")
		return
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, bankStatementLimit+1))
	if err != nil || len(data) > bankStatementLimit {
		writeError(w, http.StatusBadRequest, "загрузите файл выписки размером до 5 МБ")
		return
	}
	docs, err := parseClientBankExchange(data)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	ctx := r.Context()
	profile := h.accountingProfile(ctx)
	if profile.INN == "" && profile.BankAccount == "" {
		writeError(w, http.StatusConflict, "заполните ИНН и расчётный счёт в реквизитах — по ним находятся поступления")
		return
	}
	lines := []bankStatementLine{}
	outgoing := 0
	for _, doc := range docs {
		recipientAccount := firstNonEmpty(doc["ПолучательСчет"], doc["ПолучательРасчСчет"])
		incoming := (profile.INN != "" && doc["ПолучательИНН"] == profile.INN) ||
			(profile.BankAccount != "" && recipientAccount == profile.BankAccount)
		if !incoming {
			outgoing++
			continue
		}
		amount, _ := strconv.ParseFloat(strings.ReplaceAll(strings.ReplaceAll(doc["Сумма"], " ", ""), ",", "."), 64)
		date := ""
		if t, err := time.Parse("02.01.2006", firstNonEmpty(doc["ДатаПоступило"], doc["Дата"])); err == nil {
			date = t.Format(accountingDateLayout)
		}
		line := bankStatementLine{
			Number:       doc["Номер"],
			Date:         date,
			Amount:       math.Round(amount*100) / 100,
			PayerName:    firstNonEmpty(doc["Плательщик1"], doc["Плательщик"]),
			PayerINN:     doc["ПлательщикИНН"],
			PayerAccount: firstNonEmpty(doc["ПлательщикСчет"], doc["ПлательщикРасчСчет"]),
			Purpose:      doc["НазначениеПлатежа"],
		}
		line.Fingerprint = sha256Hex(strings.Join([]string{
			line.Date, line.Number, formatMoney(line.Amount), line.PayerINN, line.PayerAccount, line.Purpose,
		}, "|"))
		lines = append(lines, line)
	}
	h.matchBankStatementLines(r, lines)
	writeJSON(w, http.StatusOK, map[string]any{"lines": lines, "outgoing": outgoing, "documents": len(docs)})
}

func (h *Handler) matchBankStatementLines(r *http.Request, lines []bankStatementLine) {
	if len(lines) == 0 {
		return
	}
	ctx := r.Context()
	db := h.dbOf(ctx)
	fingerprints := make([]string, 0, len(lines))
	numbers := []int64{}
	for _, l := range lines {
		fingerprints = append(fingerprints, l.Fingerprint)
		for _, m := range invoiceNumberPattern.FindAllString(l.Purpose, -1) {
			if n, err := strconv.ParseInt(m, 10, 64); err == nil {
				numbers = append(numbers, n)
			}
		}
	}
	imported := map[string]bool{}
	if rows, err := db.Query(ctx, `
		SELECT fingerprint FROM core.bank_statement_lines WHERE fingerprint = ANY($1::text[])
	`, fingerprints); err == nil {
		for rows.Next() {
			var f string
			if rows.Scan(&f) == nil {
				imported[f] = true
			}
		}
		rows.Close()
	}
	type candidate struct {
		id     string
		status string
		amount float64
		email  string
	}
	byInvoice := map[int64]candidate{}
	if len(numbers) > 0 {
		if rows, err := db.Query(ctx, `
			SELECT p.invoice_no, p.id::text, p.status, `+paymentPaidAmountSQL+`::float8, u.email
			FROM core.payments p
			JOIN core.users u ON u.id = p.user_id
			WHERE p.provider = 'bank' AND p.invoice_no = ANY($1::bigint[])
		`, numbers); err == nil {
			for rows.Next() {
				var n int64
				var c candidate
				if rows.Scan(&n, &c.id, &c.status, &c.amount, &c.email) == nil {
					byInvoice[n] = c
				}
			}
			rows.Close()
		}
	}
	for i := range lines {
		l := &lines[i]
		if imported[l.Fingerprint] {
			l.Status, l.Note = "imported", "уже загружено"
			continue
		}
		l.Status, l.Note = "unmatched", "в назначении платежа нет номера выставленного счёта"
		for _, m := range invoiceNumberPattern.FindAllString(l.Purpose, -1) {
			n, _ := strconv.ParseInt(m, 10, 64)
			c, ok := byInvoice[n]
			if !ok {
				continue
			}
			l.InvoiceNo, l.PaymentID, l.UserEmail = n, c.id, c.email
			switch {
			case c.status == "completed" || c.status == "refunded":
				l.Status, l.Note = "paid", "счёт уже оплачен"
			case math.Abs(c.amount-l.Amount) > 0.009:
				l.Status, l.Note = "unmatched", "сумма не совпадает со счётом: "+formatMoney(c.amount)
			case c.status == "pending" || c.status == "processing":
				l.Status, l.Note = "match", ""
			default:
				l.Status, l.Note = "unmatched", "счёт в статусе "+c.status
			}
			break
		}
	}
}

func (h *Handler) AdminBankStatementApply(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var body struct {
		Lines []bankStatementLine `json:"lines"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	ctx := r.Context()
	db := h.dbOf(ctx)
	applied, skipped := 0, 0
	failures := []string{}
	for _, line := range body.Lines {
		if line.PaymentID == "" || line.Fingerprint == "" {
			skipped++
			continue
		}
		var provider, status string
		var amount float64
		var invoice int64
		err := db.QueryRow(ctx, `
			SELECT p.provider, p.status, `+paymentPaidAmountSQL+`::float8, p.invoice_no FROM core.payments p WHERE p.id = $1
		`, line.PaymentID).Scan(&provider, &status, &amount, &invoice)
		if err != nil || provider != "bank" || (status != "pending" && status != "processing") {
			failures = append(failures, fmt.Sprintf("п/п № %s: счёт уже обработан или не найден", line.Number))
			continue
		}
		if math.Abs(amount-line.Amount) > 0.009 {
			failures = append(failures, fmt.Sprintf("п/п № %s: сумма не совпадает со счётом № %d", line.Number, invoice))
			continue
		}
		var docDate *time.Time
		reference := "п/п № " + line.Number
		if t, err := time.Parse(accountingDateLayout, line.Date); err == nil {
			docDate = &t
			reference += " от " + t.Format("02.01.2006")
		}
		tag, err := db.Exec(ctx, `
			INSERT INTO core.bank_statement_lines (fingerprint, doc_number, doc_date, amount, payer_name, payer_inn, purpose, payment_id, imported_by)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			ON CONFLICT (fingerprint) DO NOTHING
		`, line.Fingerprint, line.Number, docDate, line.Amount, line.PayerName, line.PayerINN, line.Purpose, line.PaymentID, claims.UserID)
		if err != nil {
			failures = append(failures, fmt.Sprintf("п/п № %s: не удалось сохранить строку выписки", line.Number))
			continue
		}
		if tag.RowsAffected() == 0 {
			skipped++
			continue
		}
		if err := h.completeTopupPayment(ctx, line.PaymentID, reference, payments.Name("bank")); err != nil {
			_, _ = db.Exec(ctx, `DELETE FROM core.bank_statement_lines WHERE fingerprint = $1`, line.Fingerprint)
			failures = append(failures, fmt.Sprintf("п/п № %s: платёж не зачислен", line.Number))
			continue
		}
		applied++
		audit(ctx, db, claims.UserID, "payment.confirm", "payment:"+line.PaymentID, map[string]any{
			"source": "bank_statement", "document": reference, "payer_inn": line.PayerINN,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"applied": applied, "skipped": skipped, "failures": failures})
}
