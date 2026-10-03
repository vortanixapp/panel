package handlers

import (
	"context"
	"fmt"
	"html"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/vortanixapp/panel/pkg/regions"
	"github.com/vortanixapp/panel/pkg/regions/tax"
)

var invoiceStrings = map[string]map[string]string{
	"en": {
		"title":          "Tax invoice",
		"number":         "Invoice No.",
		"date":           "Date",
		"seller":         "Seller",
		"buyer":          "Buyer",
		"description":    "Description",
		"qty":            "Qty",
		"net":            "Net amount",
		"rate":           "Tax rate",
		"tax":            "Tax",
		"gross":          "Total",
		"total_due":      "Total paid",
		"tax_id":         "Tax ID",
		"country":        "Country",
		"print":          "Print / save as PDF",
		"reverse_charge": "Reverse charge: VAT to be accounted for by the recipient of the service.",
		"exempt":         "Supply exempt from tax.",
		"none":           "No tax charged.",
		"outside_scope":  "Not subject to tax in the seller's registered jurisdictions.",
		"paid":           "Paid from the account balance.",
		"all_incl":       "Prices include tax.",
	},
	"ru": {
		"title":          "Инвойс",
		"number":         "Инвойс №",
		"date":           "Дата",
		"seller":         "Продавец",
		"buyer":          "Покупатель",
		"description":    "Наименование",
		"qty":            "Кол-во",
		"net":            "Сумма без налога",
		"rate":           "Ставка налога",
		"tax":            "Налог",
		"gross":          "Всего",
		"total_due":      "Оплачено",
		"tax_id":         "Налоговый номер",
		"country":        "Страна",
		"print":          "Печать / сохранить в PDF",
		"reverse_charge": "Обратное начисление (reverse charge): налог начисляет получатель услуги.",
		"exempt":         "Операция освобождена от налога.",
		"none":           "Налог не взимается.",
		"outside_scope":  "Не облагается налогом в юрисдикциях регистрации продавца.",
		"paid":           "Оплачено с баланса аккаунта.",
		"all_incl":       "Цены включают налог.",
	},
}

func invoiceLang(locale string) string {
	if strings.HasPrefix(strings.ToLower(locale), "ru") {
		return "ru"
	}
	return "en"
}

func invoiceMoney(v float64, currency string) string {
	v = math.Round(v*100) / 100
	intPart, frac, _ := strings.Cut(strconv.FormatFloat(math.Abs(v), 'f', 2, 64), ".")
	var b strings.Builder
	if v < 0 {
		b.WriteString("-")
	}
	for i, c := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			b.WriteRune(',')
		}
		b.WriteRune(c)
	}
	return b.String() + "." + frac + " " + currency
}

type invoiceData struct {
	Number      string
	Issued      time.Time
	Description string
	Currency    string
	Tax         tax.Result
	TaxID       string
}

func (h *Handler) invoiceBuyerLines(ctx context.Context, userID, lang string, payer billingPayer) []string {
	var line, city, region, postal, country string
	_ = h.dbOf(ctx).QueryRow(ctx, `
		SELECT COALESCE(address_line, ''), COALESCE(city, ''), COALESCE(region, ''),
		       COALESCE(postal_code, ''), COALESCE(country, '')
		FROM core.user_profiles WHERE user_id = $1
	`, userID).Scan(&line, &city, &region, &postal, &country)
	name := payer.title()
	lines := []string{html.EscapeString(name)}
	if payer.Address != "" {
		lines = append(lines, html.EscapeString(payer.Address))
	} else if addr := strings.Join(nonEmptyStrings(line, city, region, postal), ", "); addr != "" {
		lines = append(lines, html.EscapeString(addr))
	}
	if payer.Country != "" {
		country = payer.Country
	}
	if country != "" {
		lines = append(lines, html.EscapeString(invoiceStrings[lang]["country"]+": "+regions.NameEN(country)))
	}
	if payer.TaxID != "" {
		lines = append(lines, html.EscapeString(invoiceStrings[lang]["tax_id"]+": "+payer.TaxID))
	}
	return lines
}

func nonEmptyStrings(items ...string) []string {
	out := []string{}
	for _, item := range items {
		if strings.TrimSpace(item) != "" {
			out = append(out, strings.TrimSpace(item))
		}
	}
	return out
}

func renderInvoice(lang string, profile accountingProfile, buyerLines []string, d invoiceData) string {
	s := invoiceStrings[lang]
	if s == nil {
		s = invoiceStrings["en"]
	}
	esc := html.EscapeString
	var b strings.Builder
	b.WriteString(`<h1>` + esc(s["title"]) + ` ` + esc(d.Number) + `</h1>`)
	b.WriteString(`<div class="sub">` + esc(s["date"]) + `: ` + esc(d.Issued.In(profile.Location).Format("2006-01-02")) + `</div>`)
	b.WriteString(`<table class="parties">`)
	b.WriteString(`<tr><td>` + esc(s["seller"]) + `</td><td>` + esc(profile.partyLine()) + `</td></tr>`)
	b.WriteString(`<tr><td>` + esc(s["buyer"]) + `</td><td>` + strings.Join(buyerLines, "<br>") + `</td></tr>`)
	b.WriteString(`</table>`)

	rate := "—"
	if d.Tax.Rate > 0 {
		rate = strconv.FormatFloat(d.Tax.Rate, 'f', -1, 64) + "%"
	} else if d.Tax.Regime == tax.RegimeReverseCharge {
		rate = "0% RC"
	} else {
		rate = "0%"
	}
	b.WriteString(`<table class="items"><tr><th>` + esc(s["description"]) + `</th><th class="c">` + esc(s["qty"]) +
		`</th><th class="r">` + esc(s["net"]) + `</th><th class="c">` + esc(s["rate"]) +
		`</th><th class="r">` + esc(s["tax"]) + `</th><th class="r">` + esc(s["gross"]) + `</th></tr>`)
	b.WriteString(`<tr><td>` + esc(d.Description) + `</td><td class="c">1</td><td class="r">` +
		esc(invoiceMoney(d.Tax.Net, d.Currency)) + `</td><td class="c">` + esc(rate) + `</td><td class="r">` +
		esc(invoiceMoney(d.Tax.Tax, d.Currency)) + `</td><td class="r">` + esc(invoiceMoney(d.Tax.Gross, d.Currency)) + `</td></tr>`)
	b.WriteString(`<tr class="strong"><td colspan="5" class="r">` + esc(s["total_due"]) + `</td><td class="r">` +
		esc(invoiceMoney(d.Tax.Gross, d.Currency)) + `</td></tr></table>`)

	notes := []string{s["paid"], s["all_incl"]}
	switch d.Tax.Regime {
	case tax.RegimeReverseCharge:
		notes = append(notes, s["reverse_charge"])
	case tax.RegimeExempt:
		notes = append(notes, s["exempt"])
	case tax.RegimeOutsideScope:
		notes = append(notes, s["outside_scope"])
	case tax.RegimeNone:
		notes = append(notes, s["none"])
	}
	for _, note := range notes {
		b.WriteString(`<p>` + esc(note) + `</p>`)
	}
	return documentHTMLLang(lang, s["title"]+" "+d.Number, s["print"], b.String())
}

func (h *Handler) invoiceNumber(ctx context.Context, txID string, at time.Time) (string, error) {
	var series string
	var number int64
	err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT series, number FROM core.invoices WHERE transaction_id = $1::uuid
	`, txID).Scan(&series, &number)
	if err == nil {
		return fmt.Sprintf("%s-%06d", series, number), nil
	}
	tx, err := h.dbOf(ctx).Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)
	series = fmt.Sprintf("INV-%d", at.Year())
	if err := tx.QueryRow(ctx, `
		INSERT INTO core.document_counters (kind, value) VALUES ($1, 1)
		ON CONFLICT (kind) DO UPDATE SET value = core.document_counters.value + 1
		RETURNING value
	`, "invoice-"+strconv.Itoa(at.Year())).Scan(&number); err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO core.invoices (transaction_id, series, number, issued_at) VALUES ($1::uuid, $2, $3, now())
		ON CONFLICT (transaction_id) DO NOTHING
	`, txID, series, number); err != nil {
		return "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	if err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT series, number FROM core.invoices WHERE transaction_id = $1::uuid
	`, txID).Scan(&series, &number); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s-%06d", series, number), nil
}

func (h *Handler) BillingInvoices(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ctx := r.Context()
	limit := jobQueryInt(r.URL.Query().Get("limit"), 50, 1, 200)
	offset := jobQueryInt(r.URL.Query().Get("offset"), 0, 0, 100000)
	onlyForeign := h.sellerCountry(h.loadTenantSettingStrings(ctx)) == "RU"

	rows, err := h.dbOf(ctx).Query(ctx, `
		SELECT t.id::text, t.created_at, COALESCE(t.description, ''), UPPER(w.currency),
		       tt.gross::float8, tt.net::float8, tt.tax::float8, tt.rate::float8, tt.regime, tt.country,
		       COALESCE(i.series || '-' || lpad(i.number::text, 6, '0'), '')
		FROM core.transaction_taxes tt
		JOIN core.transactions t ON t.id = tt.transaction_id
		JOIN core.wallets w ON w.id = t.wallet_id
		LEFT JOIN core.invoices i ON i.transaction_id = t.id
		WHERE w.user_id = $1 AND ($2::boolean = false OR tt.country <> 'RU')
		ORDER BY t.created_at DESC
		LIMIT $3 OFFSET $4
	`, claims.UserID, onlyForeign, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()

	items := []map[string]any{}
	for rows.Next() {
		var id, desc, currency, regime, country, number string
		var created time.Time
		var gross, net, taxAmount, rate float64
		if rows.Scan(&id, &created, &desc, &currency, &gross, &net, &taxAmount, &rate, &regime, &country, &number) != nil {
			continue
		}
		items = append(items, map[string]any{
			"transaction_id": id, "date": created.UTC().Format(time.RFC3339), "description": desc,
			"currency": currency, "gross": gross, "net": net, "tax": taxAmount, "rate": rate,
			"regime": regime, "country": country, "number": number,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"invoices": items})
}

func (h *Handler) BillingInvoice(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ctx := r.Context()
	txID := chi.URLParam(r, "id")

	var userID, desc, currency, regime, country, subdivision, buyerTaxID, locale string
	var created time.Time
	var gross, net, taxAmount, rate float64
	var reverse bool
	err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT w.user_id::text, COALESCE(t.description, ''), UPPER(w.currency), t.created_at,
		       tt.regime, tt.country, tt.subdivision, tt.gross::float8, tt.net::float8, tt.tax::float8,
		       tt.rate::float8, tt.reverse_charge, tt.buyer_tax_id, COALESCE(up.locale, '')
		FROM core.transaction_taxes tt
		JOIN core.transactions t ON t.id = tt.transaction_id
		JOIN core.wallets w ON w.id = t.wallet_id
		LEFT JOIN core.user_profiles up ON up.user_id = w.user_id
		WHERE t.id = $1::uuid
	`, txID).Scan(&userID, &desc, &currency, &created, &regime, &country, &subdivision, &gross, &net,
		&taxAmount, &rate, &reverse, &buyerTaxID, &locale)
	if err != nil || (userID != claims.UserID && !isStaffRole(claims.Role)) {
		writeError(w, http.StatusNotFound, "invoice not found")
		return
	}

	profile := h.accountingProfile(ctx)
	payer, err := h.billingPayer(ctx, userID)
	if err != nil {
		writeError(w, http.StatusNotFound, "invoice not found")
		return
	}
	lang := invoiceLang(locale)
	number, err := h.invoiceNumber(ctx, txID, created.In(profile.Location))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Не удалось присвоить номер инвойса")
		return
	}
	d := invoiceData{
		Number: number, Issued: created, Description: desc, Currency: currency,
		Tax: tax.Result{
			Regime: regime, Country: country, Subdivision: subdivision, Rate: rate,
			Net: net, Tax: taxAmount, Gross: gross, ReverseCharge: reverse,
		},
		TaxID: buyerTaxID,
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(renderInvoice(lang, profile, h.invoiceBuyerLines(ctx, userID, lang, payer), d)))
}
