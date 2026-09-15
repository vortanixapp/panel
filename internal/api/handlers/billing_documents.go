package handlers

import (
	"context"
	"encoding/json"
	"html"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"

	"github.com/vortanixapp/panel/internal/api/payments"
)

var billingPayerTypes = []string{"person", "ip", "company"}

type billingPayer struct {
	UserID     string `json:"user_id"`
	Email      string `json:"email"`
	PersonName string `json:"person_name"`
	PayerType  string `json:"payer_type"`
	LegalName  string `json:"legal_name"`
	INN        string `json:"inn"`
	KPP        string `json:"kpp"`
	OGRN       string `json:"ogrn"`
	Address    string `json:"address"`
}

func (p billingPayer) title() string {
	if p.PayerType != "person" && p.LegalName != "" {
		return p.LegalName
	}
	return firstNonEmpty(p.PersonName, p.Email)
}

func (p billingPayer) partyLine() string {
	parts := []string{p.title()}
	if p.INN != "" {
		parts = append(parts, "ИНН "+p.INN)
	}
	if p.KPP != "" {
		parts = append(parts, "КПП "+p.KPP)
	}
	if p.OGRN != "" {
		parts = append(parts, ogrnLabel(p.OGRN)+" "+p.OGRN)
	}
	if p.Address != "" {
		parts = append(parts, p.Address)
	}
	if p.Email != "" && p.title() != p.Email {
		parts = append(parts, p.Email)
	}
	return strings.Join(parts, ", ")
}

func (p *billingPayer) normalize() {
	p.PayerType = strings.TrimSpace(p.PayerType)
	p.LegalName = strings.TrimSpace(p.LegalName)
	p.INN = strings.TrimSpace(p.INN)
	p.KPP = strings.ToUpper(strings.TrimSpace(p.KPP))
	p.OGRN = strings.TrimSpace(p.OGRN)
	p.Address = strings.TrimSpace(p.Address)
	if p.PayerType == "person" {
		p.LegalName, p.KPP, p.OGRN = "", "", ""
	}
	if p.PayerType == "ip" {
		p.KPP = ""
	}
}

func (p billingPayer) validate() string {
	switch {
	case !slices.Contains(billingPayerTypes, p.PayerType):
		return "неизвестный тип плательщика"
	case p.PayerType != "person" && p.LegalName == "":
		return "укажите наименование организации или ИП"
	case p.PayerType != "person" && p.INN == "":
		return "укажите ИНН"
	case p.INN != "" && !validINN(p.INN):
		return "ИНН указан с ошибкой: проверьте цифры"
	case p.PayerType == "company" && len(p.INN) != 10:
		return "ИНН организации состоит из 10 цифр"
	case p.PayerType != "company" && p.INN != "" && len(p.INN) != 12:
		return "ИНН предпринимателя и физического лица состоит из 12 цифр"
	case p.KPP != "" && !kppPattern.MatchString(p.KPP):
		return "КПП состоит из 9 символов"
	case p.OGRN != "" && p.PayerType == "company" && !digitsOnly(p.OGRN, 13):
		return "ОГРН организации состоит из 13 цифр"
	case p.OGRN != "" && p.PayerType == "ip" && !digitsOnly(p.OGRN, 15):
		return "ОГРНИП состоит из 15 цифр"
	case utf8.RuneCountInString(p.LegalName) > 300:
		return "наименование — не длиннее 300 символов"
	case utf8.RuneCountInString(p.Address) > 500:
		return "адрес — не длиннее 500 символов"
	}
	return ""
}

func (h *Handler) billingPayer(ctx context.Context, userID string) (billingPayer, error) {
	p := billingPayer{UserID: userID}
	var first, last, display string
	err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT u.email, COALESCE(up.first_name, ''), COALESCE(up.last_name, ''), COALESCE(up.display_name, ''),
		       COALESCE(bp.payer_type, 'person'), COALESCE(bp.legal_name, ''), COALESCE(bp.inn, ''),
		       COALESCE(bp.kpp, ''), COALESCE(bp.ogrn, ''), COALESCE(bp.address, '')
		FROM core.users u
		LEFT JOIN core.user_profiles up ON up.user_id = u.id
		LEFT JOIN core.user_billing_profiles bp ON bp.user_id = u.id
		WHERE u.id = $1
	`, userID).Scan(&p.Email, &first, &last, &display, &p.PayerType, &p.LegalName, &p.INN, &p.KPP, &p.OGRN, &p.Address)
	p.PersonName = firstNonEmpty(strings.TrimSpace(strings.TrimSpace(last)+" "+strings.TrimSpace(first)), strings.TrimSpace(display))
	return p, err
}

func (h *Handler) BillingDocuments(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	h.writeBillingDocuments(w, r, claims.UserID)
}

func (h *Handler) BillingPayerUpdate(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var body billingPayer
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	body.normalize()
	if msg := body.validate(); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	ctx := r.Context()
	if _, err := h.dbOf(ctx).Exec(ctx, `
		INSERT INTO core.user_billing_profiles (user_id, payer_type, legal_name, inn, kpp, ogrn, address, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, now())
		ON CONFLICT (user_id) DO UPDATE SET
			payer_type = EXCLUDED.payer_type, legal_name = EXCLUDED.legal_name, inn = EXCLUDED.inn,
			kpp = EXCLUDED.kpp, ogrn = EXCLUDED.ogrn, address = EXCLUDED.address, updated_at = now()
	`, claims.UserID, body.PayerType, body.LegalName, body.INN, body.KPP, body.OGRN, body.Address); err != nil {
		writeError(w, http.StatusInternalServerError, "не удалось сохранить реквизиты")
		return
	}
	audit(ctx, h.dbOf(ctx), claims.UserID, "billing.payer.update", "user:"+claims.UserID, map[string]any{
		"payer_type": body.PayerType, "inn": body.INN,
	})
	h.writeBillingDocuments(w, r, claims.UserID)
}

func (h *Handler) BillingAct(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	h.writeBillingAct(w, r, claims.UserID)
}

func (h *Handler) BillingReconciliation(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	h.writeReconciliation(w, r, claims.UserID)
}

func (h *Handler) AdminAccountingClientDocuments(w http.ResponseWriter, r *http.Request) {
	h.writeBillingDocuments(w, r, chi.URLParam(r, "id"))
}

func (h *Handler) AdminAccountingClientAct(w http.ResponseWriter, r *http.Request) {
	h.writeBillingAct(w, r, chi.URLParam(r, "id"))
}

func (h *Handler) AdminAccountingClientReconciliation(w http.ResponseWriter, r *http.Request) {
	h.writeReconciliation(w, r, chi.URLParam(r, "id"))
}

type billingActMonth struct {
	Period string  `json:"period"`
	Count  int64   `json:"count"`
	Amount float64 `json:"amount"`
	Number *int64  `json:"number"`
	Closed bool    `json:"closed"`
}

func (h *Handler) writeBillingDocuments(w http.ResponseWriter, r *http.Request, userID string) {
	ctx := r.Context()
	settings := h.loadTenantSettingStrings(ctx)
	profile := accountingProfileFrom(settings)
	payer, err := h.billingPayer(ctx, userID)
	if err != nil {
		writeError(w, http.StatusNotFound, "пользователь не найден")
		return
	}
	currencies := h.billingCurrencies(ctx, userID)
	currency := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("currency")))
	if !slices.Contains(currencies, currency) {
		currency = defaultCurrencyFrom(settings)
		if len(currencies) > 0 && !slices.Contains(currencies, currency) {
			currency = currencies[0]
		}
	}
	months, err := h.billingActMonths(ctx, userID, currency, profile.Location)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "не удалось получить список документов")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"payer":      payer,
		"currency":   currency,
		"currencies": currencies,
		"months":     months,
		"company": map[string]any{
			"name":       profile.shortName(),
			"inn":        profile.INN,
			"tax_system": profile.TaxSystem,
			"ready":      profile.ready(),
		},
	})
}

func (h *Handler) billingCurrencies(ctx context.Context, userID string) []string {
	out := []string{}
	rows, err := h.dbOf(ctx).Query(ctx, `
		SELECT DISTINCT UPPER(currency) FROM core.wallets WHERE user_id = $1 ORDER BY 1
	`, userID)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var c string
		if rows.Scan(&c) == nil && c != "" {
			out = append(out, c)
		}
	}
	return out
}

func (h *Handler) billingActMonths(ctx context.Context, userID, currency string, loc *time.Location) ([]billingActMonth, error) {
	rows, err := h.dbOf(ctx).Query(ctx, `
		SELECT m.period, m.cnt, m.total, a.number
		FROM (
			SELECT to_char(date_trunc('month', t.created_at AT TIME ZONE $3), 'YYYY-MM') AS period,
			       COUNT(*) AS cnt, SUM(ABS(t.amount))::float8 AS total
			FROM core.transactions t
			JOIN core.wallets w ON w.id = t.wallet_id
			WHERE w.user_id = $1 AND UPPER(w.currency) = $2 AND t.type = 'debit'
			  AND t.source_type = ANY($4::text[]) AND t.amount <> 0
			GROUP BY 1
		) m
		LEFT JOIN core.billing_acts a
		  ON a.user_id = $1 AND a.currency = $2 AND to_char(a.period, 'YYYY-MM') = m.period
		ORDER BY m.period DESC
		LIMIT 60
	`, userID, currency, loc.String(), accountingServiceSources)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	current := time.Now().In(loc).Format(accountingMonthLayout)
	months := []billingActMonth{}
	for rows.Next() {
		var m billingActMonth
		if err := rows.Scan(&m.Period, &m.Count, &m.Amount, &m.Number); err != nil {
			return nil, err
		}
		m.Closed = m.Period < current
		months = append(months, m)
	}
	return months, rows.Err()
}

type billingServiceLine struct {
	At     time.Time
	Title  string
	Amount float64
}

func (h *Handler) billingServiceLines(ctx context.Context, userID, currency string, from, to time.Time) ([]billingServiceLine, error) {
	rows, err := h.dbOf(ctx).Query(ctx, `
		SELECT t.created_at, COALESCE(t.source_type, ''), ABS(t.amount)::float8,
		       COALESCE(s.name, ''), COALESCE(tr.name, '')
		FROM core.transactions t
		JOIN core.wallets w ON w.id = t.wallet_id
		LEFT JOIN core.servers s ON s.id = t.source_id
		LEFT JOIN core.tariffs tr ON tr.id = s.tariff_id
		WHERE w.user_id = $1 AND UPPER(w.currency) = $2 AND t.type = 'debit'
		  AND t.source_type = ANY($3::text[]) AND t.amount <> 0
		  AND t.created_at >= $4 AND t.created_at < $5
		ORDER BY t.created_at, t.id
	`, userID, currency, accountingServiceSources, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var lines []billingServiceLine
	for rows.Next() {
		var line billingServiceLine
		var source, server, tariff string
		if err := rows.Scan(&line.At, &source, &line.Amount, &server, &tariff); err != nil {
			return nil, err
		}
		line.Title = accountingServiceTitle(source, server, tariff)
		lines = append(lines, line)
	}
	return lines, rows.Err()
}

func (h *Handler) billingActNumber(ctx context.Context, userID string, period time.Time, currency string) (int64, error) {
	day := period.Format(accountingDateLayout)
	db := h.dbOf(ctx)
	if _, err := db.Exec(ctx, `
		INSERT INTO core.billing_acts (user_id, period, currency)
		SELECT $1::uuid, $2::date, $3::text
		WHERE NOT EXISTS (
			SELECT 1 FROM core.billing_acts
			WHERE user_id = $1::uuid AND period = $2::date AND currency = $3::text
		)
		ON CONFLICT DO NOTHING
	`, userID, day, currency); err != nil {
		return 0, err
	}
	var number int64
	err := db.QueryRow(ctx, `
		SELECT number FROM core.billing_acts WHERE user_id = $1 AND period = $2::date AND currency = $3
	`, userID, day, currency).Scan(&number)
	return number, err
}

func billingQueryCurrency(r *http.Request, settings map[string]string) string {
	return accountingQueryCurrency(r, settings)
}

func (h *Handler) writeBillingAct(w http.ResponseWriter, r *http.Request, userID string) {
	ctx := r.Context()
	settings := h.loadTenantSettingStrings(ctx)
	profile := accountingProfileFrom(settings)
	start, err := time.ParseInLocation(accountingMonthLayout, strings.TrimSpace(r.URL.Query().Get("period")), profile.Location)
	if err != nil {
		writeError(w, http.StatusBadRequest, "месяц акта указан неверно")
		return
	}
	end := start.AddDate(0, 1, 0)
	if end.After(time.Now()) {
		writeError(w, http.StatusConflict, "акт за месяц формируется после окончания месяца")
		return
	}
	currency := billingQueryCurrency(r, settings)
	payer, err := h.billingPayer(ctx, userID)
	if err != nil {
		writeError(w, http.StatusNotFound, "пользователь не найден")
		return
	}
	lines, err := h.billingServiceLines(ctx, userID, currency, start, end)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "не удалось собрать услуги за месяц")
		return
	}
	if len(lines) == 0 {
		writeError(w, http.StatusNotFound, "в этом месяце услуг не было")
		return
	}
	number, err := h.billingActNumber(ctx, userID, start, currency)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "не удалось присвоить номер акта")
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write([]byte(renderBillingAct(profile, payer, number, start, currency, lines)))
}

func renderBillingAct(profile accountingProfile, payer billingPayer, number int64, start time.Time, currency string, lines []billingServiceLine) string {
	esc := html.EscapeString
	var rows strings.Builder
	var total float64
	for i, line := range lines {
		total += line.Amount
		money := formatMoneyRu(line.Amount)
		rows.WriteString(`<tr><td class="c">` + strconv.Itoa(i+1) + `</td><td>` + esc(line.Title) +
			` (` + line.At.In(profile.Location).Format("02.01.2006") + `)</td><td class="c">1</td><td class="c">шт</td><td class="r">` +
			money + `</td><td class="r">` + money + `</td></tr>`)
	}
	vat := profile.vatRate()
	vatRow := `<tr><td colspan="5" class="r">Без налога (НДС)</td><td class="r">—</td></tr>`
	if vat != "none" {
		vatRow = `<tr><td colspan="5" class="r">В том числе ` + esc(vatLabel(vat)) + `</td><td class="r">` + formatMoneyRu(vatIncluded(vat, total)) + `</td></tr>`
	}
	words := ""
	if currency == "RUB" {
		words = `<p><b>` + esc(rublesInWords(total)) + `</b></p>`
	}
	basis := "Публичная оферта"
	if profile.Site != "" {
		basis += " на сайте " + profile.Site
	}
	title := "Акт № " + strconv.FormatInt(number, 10) + " от " + russianDate(start.AddDate(0, 1, -1))
	body := `<h1>` + esc(title) + `</h1>
  <div class="sub">об оказании услуг за ` + esc(russianMonth(start)) + `</div>
  <table class="parties">
    <tr><td>Исполнитель</td><td>` + esc(profile.partyLine()) + `</td></tr>
    <tr><td>Заказчик</td><td>` + esc(payer.partyLine()) + `</td></tr>
    <tr><td>Основание</td><td>` + esc(basis) + `</td></tr>
  </table>
  <table class="items">
    <thead><tr><th>№</th><th>Наименование услуги</th><th>Кол-во</th><th>Ед.</th><th>Цена, ` + esc(currency) + `</th><th>Сумма, ` + esc(currency) + `</th></tr></thead>
    <tbody>` + rows.String() + `</tbody>
    <tfoot>
      <tr class="strong"><td colspan="5" class="r">Итого</td><td class="r">` + formatMoneyRu(total) + `</td></tr>
      ` + vatRow + `
    </tfoot>
  </table>
  <p>Всего оказано услуг ` + strconv.Itoa(len(lines)) + `, на сумму ` + formatMoneyRu(total) + ` ` + esc(currency) + `.</p>
  ` + words + `
  <p>Вышеперечисленные услуги оказаны полностью и в срок. Заказчик претензий по объёму, качеству и срокам оказания услуг не имеет.</p>
  ` + documentSignatures(profile, payer)
	return documentHTML(title, body)
}

func documentSignatures(profile accountingProfile, payer billingPayer) string {
	esc := html.EscapeString
	signer := strings.TrimSpace(profile.SignerPosition + " " + profile.SignerName)
	return `<table class="signs"><tr>
    <td><div class="role">Исполнитель</div><div>` + esc(profile.shortName()) + `</div><div class="line">` + esc(firstNonEmpty(signer, "подпись")) + `</div></td>
    <td><div class="role">Заказчик</div><div>` + esc(payer.title()) + `</div><div class="line">подпись</div></td>
  </tr></table>`
}

type reconciliationEntry struct {
	At     time.Time
	Title  string
	Debit  float64
	Credit float64
}

func (h *Handler) writeReconciliation(w http.ResponseWriter, r *http.Request, userID string) {
	ctx := r.Context()
	settings := h.loadTenantSettingStrings(ctx)
	profile := accountingProfileFrom(settings)
	now := time.Now().In(profile.Location)
	from, to, msg := accountingPeriod(r, profile.Location,
		time.Date(now.Year(), 1, 1, 0, 0, 0, 0, profile.Location), dayStart(now))
	if msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	currency := billingQueryCurrency(r, settings)
	payer, err := h.billingPayer(ctx, userID)
	if err != nil {
		writeError(w, http.StatusNotFound, "пользователь не найден")
		return
	}
	opening, entries, err := h.reconciliationEntries(ctx, userID, currency, from, to.AddDate(0, 0, 1))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "не удалось собрать операции за период")
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write([]byte(renderReconciliation(profile, payer, currency, from, to, opening, entries)))
}

func (h *Handler) reconciliationEntries(ctx context.Context, userID, currency string, from, to time.Time) (float64, []reconciliationEntry, error) {
	db := h.dbOf(ctx)
	var opening float64
	if err := db.QueryRow(ctx, `
		SELECT COALESCE(SUM(CASE WHEN t.type = 'credit' THEN ABS(t.amount) ELSE -ABS(t.amount) END), 0)::float8
		FROM core.transactions t
		JOIN core.wallets w ON w.id = t.wallet_id
		WHERE w.user_id = $1 AND UPPER(w.currency) = $2 AND t.created_at < $3
	`, userID, currency, from).Scan(&opening); err != nil {
		return 0, nil, err
	}
	rows, err := db.Query(ctx, `
		SELECT t.created_at, t.type, COALESCE(t.source_type, ''), ABS(t.amount)::float8, COALESCE(t.description, ''),
		       COALESCE(p.invoice_no, 0), COALESCE(p.provider, ''), COALESCE(s.name, ''), COALESCE(tr.name, '')
		FROM core.transactions t
		JOIN core.wallets w ON w.id = t.wallet_id
		LEFT JOIN core.payments p ON p.id = t.source_id AND t.source_type IN ('payment', 'refund')
		LEFT JOIN core.servers s ON s.id = t.source_id AND t.source_type = ANY($5::text[])
		LEFT JOIN core.tariffs tr ON tr.id = s.tariff_id
		WHERE w.user_id = $1 AND UPPER(w.currency) = $2 AND t.created_at >= $3 AND t.created_at < $4 AND t.amount <> 0
		ORDER BY t.created_at, t.id
	`, userID, currency, from, to, accountingServiceSources)
	if err != nil {
		return 0, nil, err
	}
	defer rows.Close()
	var entries []reconciliationEntry
	for rows.Next() {
		var e reconciliationEntry
		var typ, source, desc, provider, server, tariff string
		var amount float64
		var invoice int64
		if err := rows.Scan(&e.At, &typ, &source, &amount, &desc, &invoice, &provider, &server, &tariff); err != nil {
			return 0, nil, err
		}
		e.Title = reconciliationTitle(source, desc, invoice, provider, server, tariff)
		if typ == "credit" {
			e.Credit = amount
		} else {
			e.Debit = amount
		}
		entries = append(entries, e)
	}
	return opening, entries, rows.Err()
}

func reconciliationTitle(source, desc string, invoice int64, provider, server, tariff string) string {
	switch {
	case source == "payment":
		title := "Оплата, счёт № " + strconv.FormatInt(invoice, 10)
		if provider != "" {
			title += " (" + payments.Name(provider) + ")"
		}
		return title
	case source == "refund":
		return "Возврат оплаты, счёт № " + strconv.FormatInt(invoice, 10)
	case slices.Contains(accountingServiceSources, source):
		return "Оказаны услуги: " + accountingServiceTitle(source, server, tariff)
	case source == "balance_refund":
		return firstNonEmpty(desc, "Возврат остатка аванса")
	case source == "daily_bonus":
		return "Бонусное зачисление"
	case source == "admin_adjustment" || source == "admin_user":
		return "Корректировка баланса"
	}
	return firstNonEmpty(desc, "Операция по балансу")
}

func renderReconciliation(profile accountingProfile, payer billingPayer, currency string, from, to time.Time, opening float64, entries []reconciliationEntry) string {
	esc := html.EscapeString
	saldo := func(label string, balance float64) string {
		debit, credit := "", ""
		switch {
		case balance > 0.004:
			credit = formatMoneyRu(balance)
		case balance < -0.004:
			debit = formatMoneyRu(-balance)
		}
		return `<tr class="strong"><td colspan="2">` + esc(label) + `</td><td class="r">` + debit + `</td><td class="r">` + credit + `</td></tr>`
	}
	var rows strings.Builder
	var debits, credits float64
	for _, e := range entries {
		debit, credit := "", ""
		if e.Debit > 0 {
			debit = formatMoneyRu(e.Debit)
			debits += e.Debit
		}
		if e.Credit > 0 {
			credit = formatMoneyRu(e.Credit)
			credits += e.Credit
		}
		rows.WriteString(`<tr><td class="c">` + e.At.In(profile.Location).Format("02.01.2006") + `</td><td>` + esc(e.Title) +
			`</td><td class="r">` + debit + `</td><td class="r">` + credit + `</td></tr>`)
	}
	closing := opening + credits - debits
	company, client := profile.shortName(), payer.title()
	amount := func(v float64) string {
		text := formatMoneyRu(v) + " " + currency
		if currency == "RUB" {
			text += " (" + rublesInWords(v) + ")"
		}
		return text
	}
	var result string
	switch {
	case closing > 0.004:
		result = "задолженность " + company + " перед " + client + " (полученный аванс) составляет " + amount(closing)
	case closing < -0.004:
		result = "задолженность " + client + " перед " + company + " составляет " + amount(-closing)
	default:
		result = "задолженность отсутствует"
	}
	period := from.Format("02.01.2006") + " — " + to.Format("02.01.2006")
	title := "Акт сверки взаимных расчётов за период " + period
	body := `<h1>Акт сверки взаимных расчётов</h1>
  <div class="sub">за период ` + esc(period) + `, валюта ` + esc(currency) + `</div>
  <table class="parties">
    <tr><td>Исполнитель</td><td>` + esc(profile.partyLine()) + `</td></tr>
    <tr><td>Заказчик</td><td>` + esc(payer.partyLine()) + `</td></tr>
  </table>
  <p>Мы, нижеподписавшиеся, составили настоящий акт сверки о том, что состояние взаимных расчётов по данным учёта ` + esc(company) + ` следующее:</p>
  <table class="items">
    <thead><tr><th>Дата</th><th>Документ</th><th>Дебет, ` + esc(currency) + `</th><th>Кредит, ` + esc(currency) + `</th></tr></thead>
    <tbody>
      ` + saldo("Сальдо на "+from.Format("02.01.2006"), opening) + `
      ` + rows.String() + `
      <tr class="strong"><td colspan="2">Обороты за период</td><td class="r">` + formatMoneyRu(debits) + `</td><td class="r">` + formatMoneyRu(credits) + `</td></tr>
      ` + saldo("Сальдо на "+to.Format("02.01.2006"), closing) + `
    </tbody>
  </table>
  <p>По данным ` + esc(company) + ` на ` + esc(russianDate(to)) + ` ` + esc(result) + `.</p>
  ` + documentSignatures(profile, payer)
	return documentHTML(title, body)
}
