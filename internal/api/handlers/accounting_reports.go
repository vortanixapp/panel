package handlers

import (
	"context"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/vortanixapp/panel/internal/api/payments"
)

const (
	paymentPaidAmountSQL   = `COALESCE(NULLIF(p.meta->>'charge_amount', '')::numeric, p.amount)`
	paymentPaidCurrencySQL = `UPPER(COALESCE(NULLIF(p.meta->>'charge_currency', ''), p.currency))`
	payerNameSQL           = `COALESCE(NULLIF(bp.legal_name, ''), NULLIF(TRIM(COALESCE(up.last_name, '') || ' ' || COALESCE(up.first_name, '')), ''), NULLIF(up.display_name, ''), u.email)`
	payerJoinsSQL          = `
		LEFT JOIN core.user_profiles up ON up.user_id = u.id
		LEFT JOIN core.user_billing_profiles bp ON bp.user_id = u.id`
	signedAmountSQL = `CASE WHEN t.type = 'credit' THEN ABS(t.amount) ELSE -ABS(t.amount) END`
)

type accountingMoney struct {
	Count  int64   `json:"count"`
	Amount float64 `json:"amount"`
}

type accountingProviderRow struct {
	Provider string  `json:"provider"`
	Name     string  `json:"name"`
	Count    int64   `json:"count"`
	Amount   float64 `json:"amount"`
	Refunds  float64 `json:"refunds"`
}

type accountingMonthRow struct {
	Month    string  `json:"month"`
	Income   float64 `json:"income"`
	Refunds  float64 `json:"refunds"`
	Services float64 `json:"services"`
}

func (h *Handler) AdminAccountingSummary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	settings := h.loadTenantSettingStrings(ctx)
	profile := accountingProfileFrom(settings)
	now := time.Now().In(profile.Location)
	from, to, msg := accountingPeriod(r, profile.Location, monthStart(now), monthStart(now).AddDate(0, 1, -1))
	if msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	currency := accountingQueryCurrency(r, settings)
	toEx := to.AddDate(0, 0, 1)
	db := h.dbOf(ctx)
	otherSources := append(append([]string{}, accountingServiceSources...), "payment", "refund", "balance_refund")

	var err error
	scan := func(query string, args []any, dest ...any) {
		if err == nil {
			err = db.QueryRow(ctx, query, args...).Scan(dest...)
		}
	}
	period := []any{from, toEx, currency}

	var income, refunds, services accountingMoney
	var otherCredits, otherDebits, promo, opening, closing float64
	scan(`
		SELECT COUNT(*), COALESCE(SUM(`+paymentPaidAmountSQL+`), 0)::float8
		FROM core.payments p
		WHERE p.credited_at >= $1 AND p.credited_at < $2 AND p.status IN ('completed', 'refunded')
		  AND `+paymentPaidCurrencySQL+` = $3
	`, period, &income.Count, &income.Amount)
	scan(`
		SELECT COUNT(*), COALESCE(SUM(ABS(t.amount)), 0)::float8
		FROM core.transactions t
		JOIN core.wallets w ON w.id = t.wallet_id
		WHERE t.type = 'debit' AND t.source_type IN ('refund', 'balance_refund')
		  AND t.created_at >= $1 AND t.created_at < $2 AND UPPER(w.currency) = $3
	`, period, &refunds.Count, &refunds.Amount)
	scan(`
		SELECT COUNT(*), COALESCE(SUM(ABS(t.amount)), 0)::float8
		FROM core.transactions t
		JOIN core.wallets w ON w.id = t.wallet_id
		WHERE t.type = 'debit' AND t.source_type = ANY($4::text[]) AND t.amount <> 0
		  AND t.created_at >= $1 AND t.created_at < $2 AND UPPER(w.currency) = $3
	`, append(period, accountingServiceSources), &services.Count, &services.Amount)
	scan(`
		SELECT COALESCE(SUM(ABS(t.amount)) FILTER (WHERE t.type = 'credit'), 0)::float8,
		       COALESCE(SUM(ABS(t.amount)) FILTER (WHERE t.type = 'debit'), 0)::float8
		FROM core.transactions t
		JOIN core.wallets w ON w.id = t.wallet_id
		WHERE t.created_at >= $1 AND t.created_at < $2 AND UPPER(w.currency) = $3
		  AND NOT (COALESCE(t.source_type, '') = ANY($4::text[]))
	`, append(period, otherSources), &otherCredits, &otherDebits)
	scan(`
		SELECT COALESCE(SUM(GREATEST(COALESCE(NULLIF(p.meta->>'credited_amount', '')::numeric, p.amount) - p.amount, 0)), 0)::float8
		FROM core.payments p
		WHERE p.credited_at >= $1 AND p.credited_at < $2 AND p.status IN ('completed', 'refunded')
		  AND UPPER(p.currency) = $3
	`, period, &promo)
	scan(`
		SELECT COALESCE(SUM(`+signedAmountSQL+`) FILTER (WHERE t.created_at < $1), 0)::float8,
		       COALESCE(SUM(`+signedAmountSQL+`), 0)::float8
		FROM core.transactions t
		JOIN core.wallets w ON w.id = t.wallet_id
		WHERE t.created_at < $2 AND UPPER(w.currency) = $3
	`, period, &opening, &closing)

	var providers []accountingProviderRow
	var months []accountingMonthRow
	if err == nil {
		providers, err = h.accountingProviders(ctx, db, from, toEx, currency)
	}
	if err == nil {
		months, err = h.accountingMonths(ctx, db, from, toEx, currency, profile.Location.String())
	}
	if err != nil {
		log.Printf("бухгалтерская сводка не собрана: %v", err)
		writeError(w, http.StatusInternalServerError, "Не удалось собрать сводку")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"from":             from.Format(accountingDateLayout),
		"to":               to.Format(accountingDateLayout),
		"currency":         currency,
		"currencies":       h.accountingCurrencies(ctx, settings),
		"income":           income,
		"refunds":          refunds,
		"net_income":       income.Amount - refunds.Amount,
		"services":         services,
		"services_vat":     vatIncluded(profile.vatRate(), services.Amount),
		"bonuses":          otherCredits + promo,
		"other_debits":     otherDebits,
		"balance_open":     opening,
		"balance_close":    closing,
		"providers":        providers,
		"months":           months,
		"tax_system":       profile.TaxSystem,
		"vat":              profile.vatRate(),
		"timezone":         profile.Timezone,
		"requisites_ready": profile.ready(),
	})
}

func (h *Handler) accountingProviders(ctx context.Context, db *pgxpool.Pool, from, to time.Time, currency string) ([]accountingProviderRow, error) {
	byCode := map[string]*accountingProviderRow{}
	rows, err := db.Query(ctx, `
		SELECT p.provider, COUNT(*), COALESCE(SUM(`+paymentPaidAmountSQL+`), 0)::float8
		FROM core.payments p
		WHERE p.credited_at >= $1 AND p.credited_at < $2 AND p.status IN ('completed', 'refunded')
		  AND `+paymentPaidCurrencySQL+` = $3
		GROUP BY p.provider
	`, from, to, currency)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		row := &accountingProviderRow{}
		if err := rows.Scan(&row.Provider, &row.Count, &row.Amount); err != nil {
			rows.Close()
			return nil, err
		}
		byCode[row.Provider] = row
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	rows, err = db.Query(ctx, `
		SELECT p.provider, COALESCE(SUM(ABS(t.amount)), 0)::float8
		FROM core.transactions t
		JOIN core.payments p ON p.id = t.source_id
		WHERE t.type = 'debit' AND t.source_type = 'refund'
		  AND t.created_at >= $1 AND t.created_at < $2 AND UPPER(p.currency) = $3
		GROUP BY p.provider
	`, from, to, currency)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var code string
		var amount float64
		if err := rows.Scan(&code, &amount); err != nil {
			rows.Close()
			return nil, err
		}
		row, ok := byCode[code]
		if !ok {
			row = &accountingProviderRow{Provider: code}
			byCode[code] = row
		}
		row.Refunds = amount
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := make([]accountingProviderRow, 0, len(byCode))
	for _, row := range byCode {
		row.Name = payments.Name(row.Provider)
		out = append(out, *row)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Amount > out[j].Amount })
	return out, nil
}

func (h *Handler) accountingMonths(ctx context.Context, db *pgxpool.Pool, from, to time.Time, currency, tz string) ([]accountingMonthRow, error) {
	byMonth := map[string]*accountingMonthRow{}
	month := func(key string) *accountingMonthRow {
		row, ok := byMonth[key]
		if !ok {
			row = &accountingMonthRow{Month: key}
			byMonth[key] = row
		}
		return row
	}
	rows, err := db.Query(ctx, `
		SELECT to_char(p.credited_at AT TIME ZONE $4, 'YYYY-MM'), COALESCE(SUM(`+paymentPaidAmountSQL+`), 0)::float8
		FROM core.payments p
		WHERE p.credited_at >= $1 AND p.credited_at < $2 AND p.status IN ('completed', 'refunded')
		  AND `+paymentPaidCurrencySQL+` = $3
		GROUP BY 1
	`, from, to, currency, tz)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var key string
		var amount float64
		if err := rows.Scan(&key, &amount); err != nil {
			rows.Close()
			return nil, err
		}
		month(key).Income = amount
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	rows, err = db.Query(ctx, `
		SELECT to_char(t.created_at AT TIME ZONE $4, 'YYYY-MM'),
		       COALESCE(SUM(ABS(t.amount)) FILTER (WHERE t.source_type IN ('refund', 'balance_refund')), 0)::float8,
		       COALESCE(SUM(ABS(t.amount)) FILTER (WHERE t.source_type = ANY($5::text[])), 0)::float8
		FROM core.transactions t
		JOIN core.wallets w ON w.id = t.wallet_id
		WHERE t.type = 'debit' AND t.created_at >= $1 AND t.created_at < $2 AND UPPER(w.currency) = $3
		GROUP BY 1
	`, from, to, currency, tz, accountingServiceSources)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var key string
		var refunds, services float64
		if err := rows.Scan(&key, &refunds, &services); err != nil {
			rows.Close()
			return nil, err
		}
		if refunds == 0 && services == 0 {
			continue
		}
		row := month(key)
		row.Refunds, row.Services = refunds, services
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := make([]accountingMonthRow, 0, len(byMonth))
	for _, row := range byMonth {
		out = append(out, *row)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Month < out[j].Month })
	return out, nil
}

type accountingReport struct {
	ctx      context.Context
	db       *pgxpool.Pool
	profile  accountingProfile
	from     time.Time
	to       time.Time
	currency string
}

func (h *Handler) AdminAccountingReport(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	settings := h.loadTenantSettingStrings(ctx)
	profile := accountingProfileFrom(settings)
	now := time.Now().In(profile.Location)
	from, to, msg := accountingPeriod(r, profile.Location, monthStart(now), monthStart(now).AddDate(0, 1, -1))
	if msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	rep := accountingReport{
		ctx: ctx, db: h.dbOf(ctx), profile: profile,
		from: from, to: to.AddDate(0, 0, 1), currency: accountingQueryCurrency(r, settings),
	}
	kind := strings.TrimSuffix(chi.URLParam(r, "kind"), ".csv")
	var rows [][]string
	var err error
	switch kind {
	case "kudir":
		rows, err = rep.kudir()
	case "payments":
		rows, err = rep.payments()
	case "refunds":
		rows, err = rep.refunds()
	case "services":
		rows, err = rep.services()
	case "balances":
		rows, err = rep.balances()
	case "acts":
		rows, err = rep.acts()
	case "offsets":
		rows, err = rep.offsets()
	default:
		writeError(w, http.StatusNotFound, "Неизвестный отчёт")
		return
	}
	if err != nil {
		log.Printf("отчёт %s не сформирован: %v", kind, err)
		writeError(w, http.StatusInternalServerError, "Не удалось сформировать отчёт")
		return
	}
	name := "vortanix-" + kind + "-" + from.Format(accountingDateLayout) + "-" + to.Format(accountingDateLayout) + "-" + strings.ToLower(rep.currency) + ".csv"
	writeAccountingCSV(w, name, rows)
}

func (rep accountingReport) local(t time.Time) time.Time {
	return t.In(rep.profile.Location)
}

func (rep accountingReport) kudir() ([][]string, error) {
	rows, err := rep.db.Query(rep.ctx, `
		SELECT 'payment', p.credited_at, p.invoice_no, `+paymentPaidAmountSQL+`::float8, p.provider, `+payerNameSQL+`
		FROM core.payments p
		JOIN core.users u ON u.id = p.user_id`+payerJoinsSQL+`
		WHERE p.credited_at >= $1 AND p.credited_at < $2 AND p.status IN ('completed', 'refunded')
		  AND `+paymentPaidCurrencySQL+` = $3
		UNION ALL
		SELECT 'refund', t.created_at, p.invoice_no, ABS(t.amount)::float8, p.provider, `+payerNameSQL+`
		FROM core.transactions t
		JOIN core.payments p ON p.id = t.source_id
		JOIN core.users u ON u.id = p.user_id`+payerJoinsSQL+`
		WHERE t.type = 'debit' AND t.source_type = 'refund'
		  AND t.created_at >= $1 AND t.created_at < $2 AND UPPER(p.currency) = $3
		UNION ALL
		SELECT 'balance_refund', t.created_at, rr.number, ABS(t.amount)::float8, '', `+payerNameSQL+`
		FROM core.transactions t
		JOIN core.balance_refund_requests rr ON rr.id = t.source_id
		JOIN core.users u ON u.id = rr.user_id`+payerJoinsSQL+`
		WHERE t.type = 'debit' AND t.source_type = 'balance_refund'
		  AND t.created_at >= $1 AND t.created_at < $2 AND UPPER(rr.currency) = $3
		ORDER BY 2, 3
	`, rep.from, rep.to, rep.currency)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := [][]string{{
		"№ п/п", "Дата и номер первичного документа", "Содержание операции",
		"Доходы, учитываемые при исчислении налоговой базы", "Расходы, учитываемые при исчислении налоговой базы",
	}}
	var n int
	var quarterSum, total float64
	quarter := ""
	flush := func() {
		if quarter != "" {
			out = append(out, []string{"", "", "Итого за " + quarter, csvMoney(quarterSum), ""})
		}
		quarterSum = 0
	}
	for rows.Next() {
		var kind, provider, payer string
		var at time.Time
		var invoice int64
		var amount float64
		if err := rows.Scan(&kind, &at, &invoice, &amount, &provider, &payer); err != nil {
			return nil, err
		}
		local := rep.local(at)
		if q := quarterLabel(local); q != quarter {
			flush()
			quarter = q
		}
		n++
		date := local.Format("02.01.2006")
		doc := date + ", счёт № " + strconv.FormatInt(invoice, 10)
		content := "Оплата услуг от " + payer + " через " + payments.Name(provider)
		switch kind {
		case "refund":
			amount = -amount
			doc = date + ", возврат по счёту № " + strconv.FormatInt(invoice, 10)
			content = "Возврат оплаты покупателю " + payer + " через " + payments.Name(provider)
		case "balance_refund":
			amount = -amount
			doc = date + ", заявка на возврат № " + strconv.FormatInt(invoice, 10)
			content = "Возврат остатка аванса покупателю " + payer + " на банковский счёт"
		}
		quarterSum += amount
		total += amount
		out = append(out, []string{strconv.Itoa(n), doc, content, csvMoney(amount), ""})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	flush()
	out = append(out, []string{"", "", "Итого за период", csvMoney(total), ""})
	return out, nil
}

func (rep accountingReport) payments() ([][]string, error) {
	rows, err := rep.db.Query(rep.ctx, `
		SELECT p.credited_at, p.invoice_no, `+payerNameSQL+`, u.email, COALESCE(bp.inn, ''),
		       p.provider, COALESCE(p.provider_payment_id, ''),
		       `+paymentPaidAmountSQL+`::float8, `+paymentPaidCurrencySQL+`,
		       COALESCE(NULLIF(p.meta->>'credited_amount', '')::numeric, p.amount)::float8, UPPER(p.currency),
		       p.refunded_amount::float8, p.status, p.meta ? 'receipt'
		FROM core.payments p
		JOIN core.users u ON u.id = p.user_id`+payerJoinsSQL+`
		WHERE p.credited_at >= $1 AND p.credited_at < $2 AND p.status IN ('completed', 'refunded')
		  AND (UPPER(p.currency) = $3 OR `+paymentPaidCurrencySQL+` = $3)
		ORDER BY p.credited_at, p.invoice_no
	`, rep.from, rep.to, rep.currency)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := [][]string{{
		"Дата оплаты", "Время", "№ счёта", "Плательщик", "ИНН", "Email", "Способ оплаты",
		"ID платежа в платёжной системе", "Оплачено", "Валюта оплаты", "Зачислено на баланс", "Валюта баланса",
		"Возвращено", "Статус", "Кассовый чек",
	}}
	var paid, credited, refunded float64
	for rows.Next() {
		var at time.Time
		var invoice int64
		var payer, email, inn, provider, external, paidCurrency, walletCurrency, status string
		var paidAmount, creditedAmount, refundedAmount float64
		var fiscal bool
		if err := rows.Scan(&at, &invoice, &payer, &email, &inn, &provider, &external,
			&paidAmount, &paidCurrency, &creditedAmount, &walletCurrency, &refundedAmount, &status, &fiscal); err != nil {
			return nil, err
		}
		local := rep.local(at)
		statusLabel := "оплачен"
		switch {
		case status == "refunded":
			statusLabel = "возвращён"
		case refundedAmount > 0:
			statusLabel = "частично возвращён"
		}
		receipt := ""
		if fiscal {
			receipt = "передан в " + payments.Name(provider)
		}
		paid += paidAmount
		credited += creditedAmount
		refunded += refundedAmount
		out = append(out, []string{
			local.Format("02.01.2006"), local.Format("15:04"), strconv.FormatInt(invoice, 10), payer, inn, email,
			payments.Name(provider), external, csvMoney(paidAmount), paidCurrency, csvMoney(creditedAmount), walletCurrency,
			csvMoney(refundedAmount), statusLabel, receipt,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out = append(out, []string{"Итого", "", "", "", "", "", "", "", csvMoney(paid), "", csvMoney(credited), "", csvMoney(refunded), "", ""})
	return out, nil
}

func (rep accountingReport) refunds() ([][]string, error) {
	rows, err := rep.db.Query(rep.ctx, `
		SELECT t.created_at, 'счёт № ' || p.invoice_no, `+payerNameSQL+`, u.email, COALESCE(bp.inn, ''), p.provider,
		       ABS(t.amount)::float8, UPPER(p.currency), COALESCE(t.description, ''), COALESCE(p.refund_reference, '')
		FROM core.transactions t
		JOIN core.payments p ON p.id = t.source_id
		JOIN core.users u ON u.id = p.user_id`+payerJoinsSQL+`
		WHERE t.type = 'debit' AND t.source_type = 'refund'
		  AND t.created_at >= $1 AND t.created_at < $2 AND UPPER(p.currency) = $3
		UNION ALL
		SELECT t.created_at, 'заявка № ' || rr.number, `+payerNameSQL+`, u.email, COALESCE(bp.inn, ''), '',
		       ABS(t.amount)::float8, rr.currency, COALESCE(t.description, ''), rr.reference
		FROM core.transactions t
		JOIN core.balance_refund_requests rr ON rr.id = t.source_id
		JOIN core.users u ON u.id = rr.user_id`+payerJoinsSQL+`
		WHERE t.type = 'debit' AND t.source_type = 'balance_refund'
		  AND t.created_at >= $1 AND t.created_at < $2 AND UPPER(rr.currency) = $3
		ORDER BY 1
	`, rep.from, rep.to, rep.currency)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := [][]string{{
		"Дата возврата", "Документ", "Плательщик", "ИНН", "Email", "Способ возврата",
		"Сумма возврата", "Валюта", "Основание", "Номер возврата в платёжной системе",
	}}
	var total float64
	for rows.Next() {
		var at time.Time
		var document, payer, email, inn, provider, currency, desc, reference string
		var amount float64
		if err := rows.Scan(&at, &document, &payer, &email, &inn, &provider, &amount, &currency, &desc, &reference); err != nil {
			return nil, err
		}
		method := "Перевод на банковский счёт"
		if provider != "" {
			method = payments.Name(provider)
		}
		total += amount
		out = append(out, []string{
			rep.local(at).Format("02.01.2006"), document, payer, inn, email,
			method, csvMoney(amount), currency, desc, reference,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out = append(out, []string{"Итого", "", "", "", "", "", csvMoney(total), "", "", ""})
	return out, nil
}

func (rep accountingReport) services() ([][]string, error) {
	rows, err := rep.db.Query(rep.ctx, `
		SELECT t.created_at, `+payerNameSQL+`, u.email, COALESCE(bp.inn, ''), COALESCE(t.source_type, ''),
		       COALESCE(s.name, ''), COALESCE(tr.name, ''), ABS(t.amount)::float8
		FROM core.transactions t
		JOIN core.wallets w ON w.id = t.wallet_id
		JOIN core.users u ON u.id = w.user_id`+payerJoinsSQL+`
		LEFT JOIN core.servers s ON s.id = t.source_id
		LEFT JOIN core.tariffs tr ON tr.id = s.tariff_id
		WHERE t.type = 'debit' AND t.source_type = ANY($4::text[]) AND t.amount <> 0
		  AND t.created_at >= $1 AND t.created_at < $2 AND UPPER(w.currency) = $3
		ORDER BY t.created_at
	`, rep.from, rep.to, rep.currency, accountingServiceSources)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	vat := rep.profile.vatRate()
	out := [][]string{{"Дата", "Покупатель", "ИНН", "Email", "Услуга", "Сумма", "НДС", "В т. ч. НДС", "Валюта"}}
	var total, totalVAT float64
	for rows.Next() {
		var at time.Time
		var payer, email, inn, source, server, tariff string
		var amount float64
		if err := rows.Scan(&at, &payer, &email, &inn, &source, &server, &tariff, &amount); err != nil {
			return nil, err
		}
		tax := vatIncluded(vat, amount)
		total += amount
		totalVAT += tax
		out = append(out, []string{
			rep.local(at).Format("02.01.2006"), payer, inn, email, accountingServiceTitle(source, server, tariff),
			csvMoney(amount), vatLabel(vat), csvMoney(tax), rep.currency,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out = append(out, []string{"Итого", "", "", "", "", csvMoney(total), "", csvMoney(totalVAT), ""})
	return out, nil
}

func (rep accountingReport) balances() ([][]string, error) {
	rows, err := rep.db.Query(rep.ctx, `
		SELECT `+payerNameSQL+`, u.email, COALESCE(bp.inn, ''),
		       COALESCE(SUM(`+signedAmountSQL+`) FILTER (WHERE t.created_at < $1), 0)::float8,
		       COALESCE(SUM(ABS(t.amount)) FILTER (WHERE t.created_at >= $1 AND t.type = 'credit' AND t.source_type = 'payment'), 0)::float8,
		       COALESCE(SUM(ABS(t.amount)) FILTER (WHERE t.created_at >= $1 AND t.type = 'credit' AND COALESCE(t.source_type, '') <> 'payment'), 0)::float8,
		       COALESCE(SUM(ABS(t.amount)) FILTER (WHERE t.created_at >= $1 AND t.type = 'debit' AND COALESCE(t.source_type, '') = ANY($4::text[])), 0)::float8,
		       COALESCE(SUM(ABS(t.amount)) FILTER (WHERE t.created_at >= $1 AND t.type = 'debit' AND NOT (COALESCE(t.source_type, '') = ANY($4::text[]))), 0)::float8,
		       COALESCE(SUM(`+signedAmountSQL+`), 0)::float8
		FROM core.transactions t
		JOIN core.wallets w ON w.id = t.wallet_id
		JOIN core.users u ON u.id = w.user_id`+payerJoinsSQL+`
		WHERE t.created_at < $2 AND UPPER(w.currency) = $3
		GROUP BY u.id, u.email, bp.legal_name, bp.inn, up.last_name, up.first_name, up.display_name
		HAVING COUNT(*) FILTER (WHERE t.created_at >= $1) > 0
		    OR ABS(COALESCE(SUM(`+signedAmountSQL+`), 0)) > 0.004
		ORDER BY 1
	`, rep.from, rep.to, rep.currency, accountingServiceSources)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := [][]string{{
		"Покупатель", "ИНН", "Email", "Сальдо на начало", "Оплачено", "Бонусы и прочие зачисления",
		"Оказано услуг", "Возвраты и прочие списания", "Сальдо на конец", "Валюта",
	}}
	var sums [6]float64
	for rows.Next() {
		var payer, email, inn string
		var values [6]float64
		if err := rows.Scan(&payer, &email, &inn, &values[0], &values[1], &values[2], &values[3], &values[4], &values[5]); err != nil {
			return nil, err
		}
		row := []string{payer, inn, email}
		for i, v := range values {
			sums[i] += v
			row = append(row, csvMoney(v))
		}
		out = append(out, append(row, rep.currency))
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	total := []string{"Итого", "", ""}
	for _, v := range sums {
		total = append(total, csvMoney(v))
	}
	out = append(out, append(total, ""))
	return out, nil
}

func (rep accountingReport) acts() ([][]string, error) {
	out := [][]string{{
		"№ акта", "Дата акта", "Период", "Покупатель", "ИНН", "КПП", "Email", "Услуг", "Сумма", "В т. ч. НДС", "Валюта",
	}}
	loc := rep.profile.Location
	start := monthStart(rep.from.In(loc))
	end := monthStart(rep.to.Add(-time.Nanosecond).In(loc)).AddDate(0, 1, 0)
	if current := monthStart(time.Now().In(loc)); end.After(current) {
		end = current
	}
	if !start.Before(end) {
		return out, nil
	}

	const monthsSQL = `
		WITH m AS (
			SELECT w.user_id, date_trunc('month', t.created_at AT TIME ZONE $4)::date AS period,
			       COUNT(*) AS cnt, SUM(ABS(t.amount))::float8 AS total
			FROM core.transactions t
			JOIN core.wallets w ON w.id = t.wallet_id
			WHERE t.type = 'debit' AND t.source_type = ANY($5::text[]) AND t.amount <> 0
			  AND t.created_at >= $1 AND t.created_at < $2 AND UPPER(w.currency) = $3
			GROUP BY 1, 2
		)`
	args := []any{start, end, rep.currency, loc.String(), accountingServiceSources}
	if _, err := rep.db.Exec(rep.ctx, monthsSQL+`
		INSERT INTO core.billing_acts (user_id, period, currency)
		SELECT m.user_id, m.period, $3 FROM m
		WHERE NOT EXISTS (
			SELECT 1 FROM core.billing_acts a
			WHERE a.user_id = m.user_id AND a.period = m.period AND a.currency = $3
		)
		ORDER BY m.period, m.user_id
		ON CONFLICT DO NOTHING
	`, args...); err != nil {
		return nil, err
	}
	rows, err := rep.db.Query(rep.ctx, monthsSQL+`
		SELECT a.number, m.period, m.cnt, m.total, `+payerNameSQL+`, u.email, COALESCE(bp.inn, ''), COALESCE(bp.kpp, '')
		FROM m
		JOIN core.billing_acts a ON a.user_id = m.user_id AND a.period = m.period AND a.currency = $3
		JOIN core.users u ON u.id = m.user_id`+payerJoinsSQL+`
		ORDER BY m.period, a.number
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	vat := rep.profile.vatRate()
	var total, totalVAT float64
	for rows.Next() {
		var number, count int64
		var period time.Time
		var amount float64
		var payer, email, inn, kpp string
		if err := rows.Scan(&number, &period, &count, &amount, &payer, &email, &inn, &kpp); err != nil {
			return nil, err
		}
		month := time.Date(period.Year(), period.Month(), 1, 0, 0, 0, 0, loc)
		tax := vatIncluded(vat, amount)
		total += amount
		totalVAT += tax
		out = append(out, []string{
			strconv.FormatInt(number, 10), month.AddDate(0, 1, -1).Format("02.01.2006"), month.Format("01.2006"),
			payer, inn, kpp, email, strconv.FormatInt(count, 10), csvMoney(amount), csvMoney(tax), rep.currency,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out = append(out, []string{"Итого", "", "", "", "", "", "", "", csvMoney(total), csvMoney(totalVAT), ""})
	return out, nil
}

var receiptOffsetStatusLabels = map[string]string{
	"pending": "в очереди",
	"sent":    "отправлен",
	"failed":  "ошибка",
	"manual":  "оформить вручную",
}

func (rep accountingReport) offsets() ([][]string, error) {
	rows, err := rep.db.Query(rep.ctx, `
		SELECT t.created_at, p.invoice_no, `+payerNameSQL+`, u.email, o.provider, o.amount::float8, o.status,
		       o.reference, o.error, COALESCE(t.source_type, ''), COALESCE(s.name, ''), COALESCE(tr.name, '')
		FROM core.receipt_offsets o
		JOIN core.payments p ON p.id = o.payment_id
		JOIN core.transactions t ON t.id = o.transaction_id
		JOIN core.users u ON u.id = p.user_id`+payerJoinsSQL+`
		LEFT JOIN core.servers s ON s.id = t.source_id
		LEFT JOIN core.tariffs tr ON tr.id = s.tariff_id
		WHERE t.created_at >= $1 AND t.created_at < $2 AND UPPER(p.currency) = $3
		ORDER BY t.created_at
	`, rep.from, rep.to, rep.currency)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := [][]string{{
		"Дата услуги", "Счёт предоплаты", "Покупатель", "Email", "Касса", "Услуга",
		"Сумма зачёта", "Статус", "Номер чека", "Ошибка",
	}}
	var total float64
	for rows.Next() {
		var at time.Time
		var invoice int64
		var payer, email, provider, status, reference, errText, source, server, tariff string
		var amount float64
		if err := rows.Scan(&at, &invoice, &payer, &email, &provider, &amount, &status, &reference, &errText,
			&source, &server, &tariff); err != nil {
			return nil, err
		}
		total += amount
		out = append(out, []string{
			rep.local(at).Format("02.01.2006"), strconv.FormatInt(invoice, 10), payer, email, payments.Name(provider),
			accountingServiceTitle(source, server, tariff), csvMoney(amount),
			firstNonEmpty(receiptOffsetStatusLabels[status], status), reference, errText,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out = append(out, []string{"Итого", "", "", "", "", "", csvMoney(total), "", "", ""})
	return out, nil
}

func (h *Handler) AdminAccountingClients(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	rows, err := h.dbOf(ctx).Query(ctx, `
		SELECT u.id::text, u.email, `+payerNameSQL+`, COALESCE(bp.inn, ''), COALESCE(bp.payer_type, 'person')
		FROM core.users u`+payerJoinsSQL+`
		WHERE $1 = ''
		   OR u.email ILIKE '%' || $1 || '%'
		   OR COALESCE(bp.inn, '') LIKE $1 || '%'
		   OR COALESCE(bp.legal_name, '') ILIKE '%' || $1 || '%'
		   OR (COALESCE(up.last_name, '') || ' ' || COALESCE(up.first_name, '')) ILIKE '%' || $1 || '%'
		ORDER BY (bp.user_id IS NULL), u.created_at DESC
		LIMIT 20
	`, q)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()
	clients := []map[string]string{}
	for rows.Next() {
		var id, email, name, inn, payerType string
		if rows.Scan(&id, &email, &name, &inn, &payerType) == nil {
			clients = append(clients, map[string]string{
				"id": id, "email": email, "name": name, "inn": inn, "payer_type": payerType,
			})
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"clients": clients})
}
