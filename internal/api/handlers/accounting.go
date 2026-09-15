package handlers

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"html"
	"math"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata"
	"unicode/utf8"

	"github.com/vortanixapp/panel/internal/api/payments"
)

const (
	accountingDefaultTimezone    = "Europe/Moscow"
	accountingDefaultReceiptItem = "Оплата услуг хостинга игровых серверов"
	accountingDateLayout         = "2006-01-02"
	accountingMonthLayout        = "2006-01"
)

var (
	accountingLegalForms     = []string{"ip", "ooo", "npd", "other"}
	accountingTaxSystems     = []string{"osn", "usn_income", "usn_income_outcome", "patent", "npd"}
	accountingVATRates       = []string{"none", "0", "5", "7", "10", "20", "22"}
	accountingReceiptModes   = []string{"full_payment", "full_prepayment", "advance"}
	accountingServiceSources = []string{"server_rent", "server_renew", "server_tariff", "server_resources"}

	accountingServiceTitles = map[string]string{
		"server_rent":      "Аренда игрового сервера",
		"server_renew":     "Продление аренды игрового сервера",
		"server_tariff":    "Смена тарифа игрового сервера",
		"server_resources": "Изменение ресурсов игрового сервера",
	}

	kppPattern = regexp.MustCompile(`^\d{4}[\dA-Z]{2}\d{3}$`)

	monthsNominative = []string{"январь", "февраль", "март", "апрель", "май", "июнь", "июль", "август", "сентябрь", "октябрь", "ноябрь", "декабрь"}
	monthsGenitive   = []string{"января", "февраля", "марта", "апреля", "мая", "июня", "июля", "августа", "сентября", "октября", "ноября", "декабря"}
	quarterNumerals  = []string{"I", "II", "III", "IV"}
)

type accountingProfile struct {
	LegalForm       string         `json:"legal_form"`
	Name            string         `json:"name"`
	FullName        string         `json:"full_name"`
	INN             string         `json:"inn"`
	KPP             string         `json:"kpp"`
	OGRN            string         `json:"ogrn"`
	Address         string         `json:"address"`
	Email           string         `json:"email"`
	Phone           string         `json:"phone"`
	BankName        string         `json:"bank_name"`
	BankBIK         string         `json:"bank_bik"`
	BankAccount     string         `json:"bank_account"`
	BankCorrAccount string         `json:"bank_corr_account"`
	SignerName      string         `json:"signer_name"`
	SignerPosition  string         `json:"signer_position"`
	TaxSystem       string         `json:"tax_system"`
	VAT             string         `json:"vat"`
	ReceiptMode     string         `json:"receipt_mode"`
	ReceiptItem     string         `json:"receipt_item"`
	Timezone        string         `json:"timezone"`
	Location        *time.Location `json:"-"`
	AppName         string         `json:"-"`
	Site            string         `json:"-"`
}

type accountingField struct {
	setting string
	value   *string
}

func (p *accountingProfile) settingFields() []accountingField {
	return []accountingField{
		{"company.legal_form", &p.LegalForm},
		{"company.name", &p.Name},
		{"company.full_name", &p.FullName},
		{"company.inn", &p.INN},
		{"company.kpp", &p.KPP},
		{"company.ogrn", &p.OGRN},
		{"company.address", &p.Address},
		{"company.email", &p.Email},
		{"company.phone", &p.Phone},
		{"company.bank_name", &p.BankName},
		{"company.bank_bik", &p.BankBIK},
		{"company.bank_account", &p.BankAccount},
		{"company.bank_corr_account", &p.BankCorrAccount},
		{"company.signer_name", &p.SignerName},
		{"company.signer_position", &p.SignerPosition},
		{"tax.system", &p.TaxSystem},
		{"tax.vat", &p.VAT},
		{"tax.receipt_mode", &p.ReceiptMode},
		{"tax.receipt_item", &p.ReceiptItem},
		{"accounting.timezone", &p.Timezone},
	}
}

func accountingProfileFrom(settings map[string]string) accountingProfile {
	var p accountingProfile
	for _, f := range p.settingFields() {
		*f.value = strings.TrimSpace(settings[f.setting])
	}
	p.AppName = firstNonEmpty(strings.TrimSpace(settings["app.name"]), strings.TrimSpace(settings["panel.name"]), "Vortanix")
	p.Site = strings.TrimSpace(settings["app.url"])
	p.normalize()
	return p
}

func (h *Handler) accountingProfile(ctx context.Context) accountingProfile {
	return accountingProfileFrom(h.loadTenantSettingStrings(ctx))
}

func (p *accountingProfile) normalize() {
	if !slices.Contains(accountingLegalForms, p.LegalForm) {
		p.LegalForm = ""
	}
	if !slices.Contains(accountingTaxSystems, p.TaxSystem) {
		p.TaxSystem = ""
	}
	if !slices.Contains(accountingVATRates, p.VAT) {
		p.VAT = ""
	}
	if !slices.Contains(accountingReceiptModes, p.ReceiptMode) {
		p.ReceiptMode = "full_payment"
	}
	if p.ReceiptItem == "" {
		p.ReceiptItem = accountingDefaultReceiptItem
	}
	p.Location = accountingLocation(p.Timezone)
	p.Timezone = p.Location.String()
}

func (p accountingProfile) vatRate() string {
	if p.VAT == "" {
		return "none"
	}
	return p.VAT
}

func (p accountingProfile) shortName() string {
	return firstNonEmpty(p.Name, p.FullName, p.AppName)
}

func (p accountingProfile) ready() bool {
	return p.INN != "" && firstNonEmpty(p.Name, p.FullName) != "" && p.TaxSystem != ""
}

func (p accountingProfile) partyLine() string {
	parts := []string{firstNonEmpty(p.FullName, p.Name, p.AppName)}
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
	if p.BankAccount != "" {
		bank := "р/с " + p.BankAccount
		if p.BankName != "" {
			bank += " в " + p.BankName
		}
		if p.BankBIK != "" {
			bank += ", БИК " + p.BankBIK
		}
		if p.BankCorrAccount != "" {
			bank += ", к/с " + p.BankCorrAccount
		}
		parts = append(parts, bank)
	}
	return strings.Join(parts, ", ")
}

func (p accountingProfile) receipt(email string) *payments.Receipt {
	return &payments.Receipt{Email: email, Item: p.ReceiptItem, Taxation: p.TaxSystem, VAT: p.VAT, Mode: p.ReceiptMode}
}

func (p *accountingProfile) validate() string {
	switch {
	case p.LegalForm != "" && !slices.Contains(accountingLegalForms, p.LegalForm):
		return "неизвестная организационно-правовая форма"
	case p.TaxSystem != "" && !slices.Contains(accountingTaxSystems, p.TaxSystem):
		return "неизвестная система налогообложения"
	case p.VAT != "" && !slices.Contains(accountingVATRates, p.VAT):
		return "неизвестная ставка НДС"
	case p.ReceiptMode != "" && !slices.Contains(accountingReceiptModes, p.ReceiptMode):
		return "неизвестный способ расчёта в чеке"
	case p.TaxSystem == "npd" && p.VAT != "" && p.VAT != "none":
		return "плательщики налога на профессиональный доход работают без НДС"
	case p.INN != "" && !validINN(p.INN):
		return "ИНН указан с ошибкой: проверьте цифры"
	case p.LegalForm == "ooo" && p.INN != "" && len(p.INN) != 10:
		return "ИНН организации состоит из 10 цифр"
	case (p.LegalForm == "ip" || p.LegalForm == "npd") && p.INN != "" && len(p.INN) != 12:
		return "ИНН предпринимателя и самозанятого состоит из 12 цифр"
	case p.KPP != "" && !kppPattern.MatchString(p.KPP):
		return "КПП состоит из 9 символов: четыре цифры, две цифры или заглавные латинские буквы, три цифры"
	case p.OGRN != "" && !digitsOnly(p.OGRN, 13) && !digitsOnly(p.OGRN, 15):
		return "ОГРН состоит из 13 цифр, ОГРНИП — из 15"
	case p.BankBIK != "" && !digitsOnly(p.BankBIK, 9):
		return "БИК состоит из 9 цифр"
	case p.BankAccount != "" && !digitsOnly(p.BankAccount, 20):
		return "расчётный счёт состоит из 20 цифр"
	case p.BankCorrAccount != "" && !digitsOnly(p.BankCorrAccount, 20):
		return "корреспондентский счёт состоит из 20 цифр"
	case p.Timezone != "" && !validTimezone(p.Timezone):
		return "неизвестный часовой пояс"
	case utf8.RuneCountInString(p.ReceiptItem) > 128:
		return "название позиции в чеке — не длиннее 128 символов"
	}
	return ""
}

func (h *Handler) AdminAccountingRequisites(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.accountingRequisitesPayload(r.Context()))
}

func (h *Handler) AdminAccountingRequisitesUpdate(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var body accountingProfile
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	for _, f := range body.settingFields() {
		*f.value = strings.TrimSpace(*f.value)
	}
	body.KPP = strings.ToUpper(body.KPP)
	if msg := body.validate(); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	ctx := r.Context()
	for _, f := range body.settingFields() {
		h.setTenantSettingString(ctx, f.setting, *f.value)
	}
	audit(ctx, h.dbOf(ctx), claims.UserID, "settings.update", "accounting", map[string]any{
		"inn": body.INN, "tax_system": body.TaxSystem, "vat": body.VAT, "receipt_mode": body.ReceiptMode,
	})
	writeJSON(w, http.StatusOK, h.accountingRequisitesPayload(ctx))
}

func (h *Handler) accountingRequisitesPayload(ctx context.Context) map[string]any {
	p := h.accountingProfile(ctx)
	return map[string]any{
		"requisites":    p,
		"legal_forms":   accountingLegalForms,
		"tax_systems":   accountingTaxSystems,
		"vat_rates":     accountingVATRates,
		"receipt_modes": accountingReceiptModes,
		"ready":         p.ready(),
	}
}

func accountingLocation(name string) *time.Location {
	if name != "" {
		if loc, err := time.LoadLocation(name); err == nil {
			return loc
		}
	}
	if loc, err := time.LoadLocation(accountingDefaultTimezone); err == nil {
		return loc
	}
	return time.UTC
}

func validTimezone(name string) bool {
	if name == "" || name == "Local" {
		return false
	}
	_, err := time.LoadLocation(name)
	return err == nil
}

func digitsOnly(s string, n int) bool {
	if len(s) != n {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func validINN(inn string) bool {
	if !digitsOnly(inn, 10) && !digitsOnly(inn, 12) {
		return false
	}
	d := make([]int, len(inn))
	for i, c := range inn {
		d[i] = int(c - '0')
	}
	check := func(coef []int) int {
		sum := 0
		for i, k := range coef {
			sum += k * d[i]
		}
		return sum % 11 % 10
	}
	if len(d) == 10 {
		return check([]int{2, 4, 10, 3, 5, 9, 4, 6, 8}) == d[9]
	}
	return check([]int{7, 2, 4, 10, 3, 5, 9, 4, 6, 8}) == d[10] &&
		check([]int{3, 7, 2, 4, 10, 3, 5, 9, 4, 6, 8}) == d[11]
}

func ogrnLabel(ogrn string) string {
	if len(ogrn) == 15 {
		return "ОГРНИП"
	}
	return "ОГРН"
}

func accountingServiceTitle(source, server, tariff string) string {
	title := accountingServiceTitles[source]
	if title == "" {
		title = "Услуги хостинга"
	}
	if tariff != "" {
		title += " по тарифу «" + tariff + "»"
	}
	if server != "" {
		title += ", сервер «" + server + "»"
	}
	return title
}

func vatIncluded(vat string, sum float64) float64 {
	rate, err := strconv.Atoi(vat)
	if err != nil || rate <= 0 {
		return 0
	}
	return math.Round(sum*float64(rate)/float64(100+rate)*100) / 100
}

func vatLabel(vat string) string {
	if _, err := strconv.Atoi(vat); err != nil {
		return "Без НДС"
	}
	return "НДС " + vat + "%"
}

func accountingPeriod(r *http.Request, loc *time.Location, from, to time.Time) (time.Time, time.Time, string) {
	q := r.URL.Query()
	if raw := strings.TrimSpace(q.Get("from")); raw != "" {
		t, err := time.ParseInLocation(accountingDateLayout, raw, loc)
		if err != nil {
			return from, to, "дата начала периода указана неверно"
		}
		from = t
	}
	if raw := strings.TrimSpace(q.Get("to")); raw != "" {
		t, err := time.ParseInLocation(accountingDateLayout, raw, loc)
		if err != nil {
			return from, to, "дата окончания периода указана неверно"
		}
		to = t
	}
	if to.Before(from) {
		return from, to, "дата окончания периода раньше даты начала"
	}
	if to.Sub(from) > 3*366*24*time.Hour {
		return from, to, "период — не больше трёх лет"
	}
	return from, to, ""
}

func monthStart(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}

func dayStart(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func accountingQueryCurrency(r *http.Request, settings map[string]string) string {
	currency := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("currency")))
	if slices.Contains(walletCurrencies, currency) {
		return currency
	}
	return defaultCurrencyFrom(settings)
}

func (h *Handler) accountingCurrencies(ctx context.Context, settings map[string]string) []string {
	out := []string{}
	rows, err := h.dbOf(ctx).Query(ctx, `SELECT DISTINCT UPPER(currency) FROM core.wallets ORDER BY 1`)
	if err == nil {
		for rows.Next() {
			var c string
			if rows.Scan(&c) == nil && c != "" {
				out = append(out, c)
			}
		}
		rows.Close()
	}
	if def := defaultCurrencyFrom(settings); !slices.Contains(out, def) {
		out = append([]string{def}, out...)
	}
	return out
}

func writeAccountingCSV(w http.ResponseWriter, filename string, rows [][]string) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write([]byte("\xEF\xBB\xBF"))
	cw := csv.NewWriter(w)
	cw.Comma = ';'
	cw.UseCRLF = true
	_ = cw.WriteAll(rows)
}

func csvMoney(v float64) string {
	return strings.Replace(formatMoney(math.Round(v*100)/100), ".", ",", 1)
}

func formatMoneyRu(v float64) string {
	v = math.Round(v*100) / 100
	intPart, frac, _ := strings.Cut(formatMoney(math.Abs(v)), ".")
	var b strings.Builder
	if v < 0 {
		b.WriteString("−")
	}
	for i, c := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			b.WriteRune(' ')
		}
		b.WriteRune(c)
	}
	return b.String() + "," + frac
}

func russianDate(t time.Time) string {
	return strconv.Itoa(t.Day()) + " " + monthsGenitive[t.Month()-1] + " " + strconv.Itoa(t.Year()) + " г."
}

func russianMonth(t time.Time) string {
	return monthsNominative[t.Month()-1] + " " + strconv.Itoa(t.Year()) + " г."
}

func quarterLabel(t time.Time) string {
	return quarterNumerals[(int(t.Month())-1)/3] + " квартал " + strconv.Itoa(t.Year()) + " г."
}

var (
	wordsUnitsMale   = []string{"", "один", "два", "три", "четыре", "пять", "шесть", "семь", "восемь", "девять"}
	wordsUnitsFemale = []string{"", "одна", "две", "три", "четыре", "пять", "шесть", "семь", "восемь", "девять"}
	wordsTeens       = []string{"десять", "одиннадцать", "двенадцать", "тринадцать", "четырнадцать", "пятнадцать", "шестнадцать", "семнадцать", "восемнадцать", "девятнадцать"}
	wordsTens        = []string{"", "", "двадцать", "тридцать", "сорок", "пятьдесят", "шестьдесят", "семьдесят", "восемьдесят", "девяносто"}
	wordsHundreds    = []string{"", "сто", "двести", "триста", "четыреста", "пятьсот", "шестьсот", "семьсот", "восемьсот", "девятьсот"}
	wordsScales      = []struct {
		female         bool
		one, few, many string
	}{
		{false, "", "", ""},
		{true, "тысяча", "тысячи", "тысяч"},
		{false, "миллион", "миллиона", "миллионов"},
		{false, "миллиард", "миллиарда", "миллиардов"},
	}
)

func pluralRu(n int64, one, few, many string) string {
	n %= 100
	if n >= 11 && n <= 14 {
		return many
	}
	switch n % 10 {
	case 1:
		return one
	case 2, 3, 4:
		return few
	}
	return many
}

func tripletWords(n int64, female bool) []string {
	var out []string
	if n/100 > 0 {
		out = append(out, wordsHundreds[n/100])
	}
	rest := n % 100
	if rest >= 10 && rest < 20 {
		return append(out, wordsTeens[rest-10])
	}
	if rest/10 > 0 {
		out = append(out, wordsTens[rest/10])
	}
	if rest%10 > 0 {
		if female {
			out = append(out, wordsUnitsFemale[rest%10])
		} else {
			out = append(out, wordsUnitsMale[rest%10])
		}
	}
	return out
}

func rublesInWords(v float64) string {
	kopecks := int64(math.Round(math.Abs(v) * 100))
	rubles := kopecks / 100
	var parts []string
	if rubles == 0 {
		parts = append(parts, "ноль")
	}
	for i := len(wordsScales) - 1; i >= 0; i-- {
		div := int64(1)
		for j := 0; j < i; j++ {
			div *= 1000
		}
		chunk := (rubles / div) % 1000
		if chunk == 0 {
			continue
		}
		parts = append(parts, tripletWords(chunk, wordsScales[i].female)...)
		if i > 0 {
			parts = append(parts, pluralRu(chunk, wordsScales[i].one, wordsScales[i].few, wordsScales[i].many))
		}
	}
	cents := kopecks % 100
	text := strings.Join(parts, " ") + " " + pluralRu(rubles, "рубль", "рубля", "рублей") +
		" " + strconv.FormatInt(cents/10, 10) + strconv.FormatInt(cents%10, 10) + " " + pluralRu(cents, "копейка", "копейки", "копеек")
	first, size := utf8.DecodeRuneInString(text)
	return strings.ToUpper(string(first)) + text[size:]
}

func documentHTML(title, body string) string {
	return `<!doctype html>
<html lang="ru"><head><meta charset="utf-8">
<title>` + html.EscapeString(title) + `</title>
<style>
  body { font-family: "Times New Roman", Georgia, serif; color: #111; margin: 32px auto; max-width: 860px; padding: 0 16px; font-size: 14px; line-height: 1.45; }
  h1 { font-size: 20px; margin: 0 0 4px; text-align: center; }
  .sub { text-align: center; color: #333; margin-bottom: 20px; }
  table { width: 100%; border-collapse: collapse; margin: 12px 0; }
  table.parties td { padding: 4px 8px 4px 0; vertical-align: top; }
  table.parties td:first-child { width: 20%; color: #444; }
  table.items th, table.items td { border: 1px solid #333; padding: 5px 6px; vertical-align: top; }
  table.items th { background: #f2f2f2; font-weight: 600; }
  table.items tr.strong td { font-weight: 600; background: #fafafa; }
  .r { text-align: right; white-space: nowrap; }
  .c { text-align: center; white-space: nowrap; }
  table.signs td { width: 50%; padding: 28px 16px 0 0; vertical-align: top; }
  .role { font-weight: 600; margin-bottom: 4px; }
  .line { margin-top: 32px; border-top: 1px solid #333; padding-top: 4px; font-size: 12px; color: #444; }
  .noprint { margin-bottom: 16px; font-family: system-ui, -apple-system, "Segoe UI", sans-serif; }
  .noprint button { padding: 8px 14px; font-size: 13px; cursor: pointer; }
  @media print { body { margin: 0; max-width: none; } .noprint { display: none; } }
</style></head>
<body>
  <div class="noprint"><button onclick="window.print()">Печать / сохранить в PDF</button></div>
` + body + `
</body></html>`
}
