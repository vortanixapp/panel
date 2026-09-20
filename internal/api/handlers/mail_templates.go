package handlers

import (
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"

	"github.com/vortanixapp/panel/internal/api/mail"
	"github.com/vortanixapp/panel/pkg/i18n"
	"github.com/vortanixapp/panel/pkg/mailer"
	"github.com/vortanixapp/panel/pkg/notify"
)

const (
	mailTemplateValueLimit = 8000
	mailTemplateGroupPanel = "panel"
)

type mailTemplateField struct {
	Name      string `json:"name"`
	Key       string `json:"key"`
	Value     string `json:"value"`
	Default   string `json:"default"`
	Multiline bool   `json:"multiline"`
	Optional  bool   `json:"optional"`
}

type mailTemplateDef struct {
	ID     string
	Group  string
	Prefix string
	Fields []string
}

var mailTemplateSampleRU = i18n.Params{
	"account":   "client@example.com",
	"action":    "перезапуск",
	"amount":    "450,00 ₽",
	"app":       "Vortanix",
	"balance":   "120,00 ₽",
	"code":      "PROMO-2026",
	"currency":  "RUB",
	"current":   "0.1.45",
	"date":      "20.09.2026",
	"days":      "3",
	"device":    "Chrome, Windows",
	"due":       "25.09.2026",
	"email":     "client@example.com",
	"free":      "4,2 ГБ",
	"hours":     "12",
	"ip":        "203.0.113.24",
	"left":      "5",
	"message":   "Сервер не отвечает дольше пяти минут",
	"name":      "Сервер Minecraft",
	"node":      "node-01",
	"number":    "1042",
	"percent":   "10",
	"period":    "30 дней",
	"prize":     "500 бонусов",
	"provider":  "Google",
	"reason":    "закончился баланс",
	"resource":  "диск",
	"subject":   "Не запускается сервер",
	"threshold": "100,00 ₽",
	"total":     "1 250,00 ₽",
	"until":     "25.09.2026",
	"value":     "8",
	"version":   "0.1.46",
	"who":       "Администратор",
}

var mailTemplateSampleEN = i18n.Params{
	"action":   "restart",
	"amount":   "$12.00",
	"balance":  "$4.00",
	"device":   "Chrome, Windows",
	"free":     "4.2 GB",
	"message":  "The server has not answered for five minutes",
	"name":     "Minecraft server",
	"period":   "30 days",
	"prize":    "500 bonuses",
	"reason":   "the balance ran out",
	"resource": "disk",
	"subject":  "The server does not start",
	"total":    "$42.00",
	"who":      "Administrator",
}

func mailTemplateSample(base string) i18n.Params {
	if base != "en" {
		return mailTemplateSampleRU
	}
	out := i18n.Params{}
	for key, value := range mailTemplateSampleRU {
		out[key] = value
	}
	for key, value := range mailTemplateSampleEN {
		out[key] = value
	}
	return out
}

var mailTemplateGroupByPrefix = []struct{ prefix, group string }{
	{"server_", "servers"},
	{"node_", "servers"},
	{"backup_", "servers"},
	{"abuse", "servers"},
	{"trial_", "servers"},
	{"hosting_", "servers"},
	{"payment_", "billing"},
	{"balance_", "billing"},
	{"bonus_", "billing"},
	{"referral", "billing"},
	{"renew", "billing"},
	{"refund", "billing"},
	{"service_", "billing"},
	{"support_", "support"},
	{"new_login", "security"},
	{"password_", "security"},
	{"twofactor_", "security"},
	{"social_", "security"},
	{"telegram_linked", "security"},
	{"email_", "security"},
	{"recovery_", "security"},
	{"api_token_", "security"},
	{"session", "security"},
	{"staff_", "staff"},
	{"audit", "staff"},
	{"disk_low", "staff"},
	{"panel_update", "staff"},
}

func mailTemplateGroupOf(name string) string {
	for _, item := range mailTemplateGroupByPrefix {
		if strings.HasPrefix(name, item.prefix) {
			return item.group
		}
	}
	return string(notify.GroupSystem)
}

func mailTemplateDefs() []mailTemplateDef {
	defs := []mailTemplateDef{
		{Prefix: "mail.verify", Fields: []string{"subject", "title", "body", "action"}},
		{Prefix: "mail.reset", Fields: []string{"subject", "title", "body", "action"}},
		{Prefix: "mail.email_change", Fields: []string{"subject", "title", "body", "action"}},
		{Prefix: "mail.test", Fields: []string{"subject", "title", "body"}},
	}
	for i := range defs {
		defs[i].ID = defs[i].Prefix
		defs[i].Group = mailTemplateGroupPanel
	}

	catalog := i18n.Catalog()[i18n.DefaultLocale]
	extras := map[string][]string{}
	letters := []string{}
	for key := range catalog {
		name, field, ok := strings.Cut(strings.TrimPrefix(key, "notify."), ".")
		if !ok || !strings.HasPrefix(key, "notify.") || strings.Contains(field, ".") {
			continue
		}
		if catalog["notify."+name+".title"] == "" || catalog["notify."+name+".body"] == "" {
			continue
		}
		switch field {
		case "title":
			letters = append(letters, name)
		case "body":
		default:
			extras[name] = append(extras[name], field)
		}
	}
	sort.Strings(letters)
	for _, name := range letters {
		fields := append([]string{"title", "body"}, extras[name]...)
		sort.Strings(fields[2:])
		defs = append(defs, mailTemplateDef{
			ID:     "notify." + name,
			Group:  mailTemplateGroupOf(name),
			Prefix: "notify." + name,
			Fields: fields,
		})
	}
	return defs
}

func mailTemplateHasField(def mailTemplateDef, name string) bool {
	for _, field := range def.Fields {
		if field == name {
			return true
		}
	}
	return false
}

func mailTemplateByID(id string) (mailTemplateDef, bool) {
	for _, def := range mailTemplateDefs() {
		if def.ID == id {
			return def, true
		}
	}
	return mailTemplateDef{}, false
}

func (h *Handler) mailTemplateLocale(r *http.Request) (string, string) {
	requested := normalizeLanguageCode(r.URL.Query().Get("locale"))
	return h.resolveTemplateLocale(r, requested)
}

func (h *Handler) resolveTemplateLocale(r *http.Request, requested string) (code, base string) {
	ctx := r.Context()
	list := h.loadLanguages(ctx, false)
	fallback := defaultLanguage(list, h.tenantSettingString(ctx, defaultLocaleSetting))
	for _, l := range list {
		if l.Code == requested && l.Enabled {
			return l.Code, templateBase(l.Base)
		}
	}
	for _, l := range list {
		if l.Code == fallback {
			return l.Code, templateBase(l.Base)
		}
	}
	return i18n.DefaultLocale, i18n.DefaultLocale
}

func templateBase(base string) string {
	if _, ok := i18n.Catalog()[base]; ok {
		return base
	}
	return i18n.DefaultLocale
}

func (h *Handler) AdminMailTemplates(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	ctx := r.Context()
	code, base := h.mailTemplateLocale(r)
	defaults := i18n.Catalog()[base]
	overrides := h.loadTranslations(ctx, code, true)
	l := i18n.ForUser(ctx, h.dbOf(ctx), claims.UserID)

	languages := []map[string]any{}
	for _, item := range h.loadLanguages(ctx, false) {
		if item.Enabled {
			languages = append(languages, map[string]any{"code": item.Code, "name": item.Name})
		}
	}

	byGroup := map[string][]map[string]any{}
	for _, def := range mailTemplateDefs() {
		fields := make([]mailTemplateField, 0, len(def.Fields))
		params := map[string]bool{}
		for _, name := range def.Fields {
			key := def.Prefix + "." + name
			field := mailTemplateField{
				Name:      name,
				Key:       key,
				Value:     overrides[key],
				Default:   defaults[key],
				Multiline: longMailField(name, defaults[key]),
				Optional:  name != "title" && name != "body" && name != "subject",
			}
			for _, placeholder := range i18n.Placeholders(field.Default) {
				params[placeholder] = true
			}
			fields = append(fields, field)
		}
		byGroup[def.Group] = append(byGroup[def.Group], map[string]any{
			"id":         def.ID,
			"name":       templateName(fields),
			"fields":     fields,
			"params":     sortedKeys(params),
			"customized": templateCustomized(fields),
		})
	}

	groups := []map[string]any{}
	order := append([]string{mailTemplateGroupPanel}, groupIDs()...)
	seen := map[string]bool{}
	for _, id := range order {
		items := byGroup[id]
		if len(items) == 0 || seen[id] {
			continue
		}
		seen[id] = true
		groups = append(groups, map[string]any{
			"id":        id,
			"label":     mailGroupLabel(l, id),
			"templates": items,
		})
	}

	cfg := h.mailerConfig(ctx)
	writeJSON(w, http.StatusOK, map[string]any{
		"locale":    code,
		"languages": languages,
		"groups":    groups,
		"mail": map[string]any{
			"ready":  cfg.Enabled(),
			"mailer": cfg.MailerName(),
			"from":   cfg.From(),
		},
	})
}

func groupIDs() []string {
	out := make([]string, 0, len(notify.Groups()))
	for _, group := range notify.Groups() {
		out = append(out, string(group))
	}
	return out
}

func mailGroupLabel(l i18n.Localizer, id string) string {
	if id == mailTemplateGroupPanel {
		return l.T("mail.group.system")
	}
	return l.Text(notify.GroupLabel(notify.Group(id)))
}

func longMailField(name, text string) bool {
	if name == "body" {
		return true
	}
	return strings.Contains(text, "\n") || utf8.RuneCountInString(text) > 90
}

func templateName(fields []mailTemplateField) string {
	for _, want := range []string{"subject", "title"} {
		for _, field := range fields {
			if field.Name != want {
				continue
			}
			if value := strings.TrimSpace(field.Value); value != "" {
				return value
			}
			if value := strings.TrimSpace(field.Default); value != "" {
				return value
			}
		}
	}
	return ""
}

func templateCustomized(fields []mailTemplateField) bool {
	for _, field := range fields {
		if strings.TrimSpace(field.Value) != "" {
			return true
		}
	}
	return false
}

func sortedKeys(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for key := range set {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

type mailTemplateRequest struct {
	Locale string            `json:"locale"`
	Values map[string]string `json:"values"`
	To     string            `json:"to"`
}

func (h *Handler) mailTemplateInput(w http.ResponseWriter, r *http.Request) (mailTemplateDef, mailTemplateRequest, string, string, bool) {
	def, ok := mailTemplateByID(chi.URLParam(r, "id"))
	if !ok {
		writeError(w, http.StatusNotFound, "Такого письма нет")
		return def, mailTemplateRequest{}, "", "", false
	}
	var body mailTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return def, body, "", "", false
	}
	allowed := map[string]bool{}
	for _, name := range def.Fields {
		allowed[name] = true
	}
	for name, value := range body.Values {
		if !allowed[name] {
			writeError(w, http.StatusUnprocessableEntity, "Неизвестное поле письма: "+name)
			return def, body, "", "", false
		}
		if strings.ContainsRune(value, 0) {
			writeError(w, http.StatusUnprocessableEntity, "Поле "+name+" содержит недопустимые символы")
			return def, body, "", "", false
		}
		if utf8.RuneCountInString(value) > mailTemplateValueLimit {
			writeError(w, http.StatusUnprocessableEntity, "Поле "+name+" длиннее 8000 символов")
			return def, body, "", "", false
		}
	}
	code, base := h.resolveTemplateLocale(r, normalizeLanguageCode(body.Locale))
	return def, body, code, base, true
}

func (h *Handler) AdminMailTemplateSave(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	def, body, code, _, ok := h.mailTemplateInput(w, r)
	if !ok {
		return
	}
	ctx := r.Context()

	keys := []string{}
	values := []string{}
	cleared := []string{}
	for _, name := range def.Fields {
		key := def.Prefix + "." + name
		value, present := body.Values[name]
		if !present {
			continue
		}
		if strings.TrimSpace(value) == "" {
			cleared = append(cleared, key)
			continue
		}
		keys = append(keys, key)
		values = append(values, value)
	}

	tx, err := h.dbOf(ctx).Begin(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Не удалось сохранить письмо")
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if len(cleared) > 0 {
		_, err = tx.Exec(ctx, `DELETE FROM core.translation_keys WHERE locale = $1 AND key = ANY($2::text[])`, code, cleared)
	}
	if err == nil && len(keys) > 0 {
		_, err = tx.Exec(ctx, `
			INSERT INTO core.translation_keys (locale, key, value)
			SELECT $1, m.key, m.value FROM unnest($2::text[], $3::text[]) AS m(key, value)
			ON CONFLICT (locale, key) DO UPDATE SET value = EXCLUDED.value, updated_at = now()
		`, code, keys, values)
	}
	if err == nil {
		err = tx.Commit(ctx)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Не удалось сохранить письмо")
		return
	}

	i18n.Invalidate()
	audit(ctx, h.dbOf(ctx), claims.UserID, "mail.template.save", "mail_template:"+def.ID, map[string]any{
		"locale":  code,
		"saved":   len(keys),
		"cleared": len(cleared),
	})
	writeJSON(w, http.StatusOK, map[string]any{"status": "updated", "locale": code})
}

func (h *Handler) AdminMailTemplateReset(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	def, ok := mailTemplateByID(chi.URLParam(r, "id"))
	if !ok {
		writeError(w, http.StatusNotFound, "Такого письма нет")
		return
	}
	ctx := r.Context()
	code, _ := h.mailTemplateLocale(r)
	keys := make([]string, 0, len(def.Fields))
	for _, name := range def.Fields {
		keys = append(keys, def.Prefix+"."+name)
	}
	if _, err := h.dbOf(ctx).Exec(ctx,
		`DELETE FROM core.translation_keys WHERE locale = $1 AND key = ANY($2::text[])`, code, keys,
	); err != nil {
		writeError(w, http.StatusInternalServerError, "Не удалось вернуть письмо к исходному виду")
		return
	}
	i18n.Invalidate()
	audit(ctx, h.dbOf(ctx), claims.UserID, "mail.template.reset", "mail_template:"+def.ID, map[string]any{"locale": code})
	writeJSON(w, http.StatusOK, map[string]any{"status": "reset", "locale": code})
}

func (h *Handler) mailTemplateLetter(def mailTemplateDef, base string, values map[string]string, r *http.Request) mail.Letter {
	defaults := i18n.Catalog()[base]
	text := func(name string) string {
		key := def.Prefix + "." + name
		value := strings.TrimSpace(values[name])
		if value == "" {
			value = defaults[key]
		}
		return i18n.Format(value, mailTemplateSample(base))
	}
	letter := mail.Letter{
		Subject: text("subject"),
		Title:   text("title"),
		Body:    text("body"),
	}
	if letter.Subject == "" {
		letter.Subject = letter.Title
	}
	if mailTemplateHasField(def, "action") {
		letter.ActionLabel = text("action")
		letter.ActionURL = strings.TrimRight(h.frontendURL, "/") + "/settings"
	}
	return letter
}

func (h *Handler) AdminMailTemplatePreview(w http.ResponseWriter, r *http.Request) {
	if _, ok := tenantClaims(r.Context()); !ok {
		return
	}
	def, body, _, base, ok := h.mailTemplateInput(w, r)
	if !ok {
		return
	}
	brand := h.mailBrand(r.Context(), r)
	msg := h.mailTemplateLetter(def, base, body.Values, r).Message(brand)
	writeJSON(w, http.StatusOK, map[string]any{
		"subject": msg.Subject,
		"html":    msg.HTML,
		"text":    msg.Text,
	})
}

func (h *Handler) AdminMailTemplateTest(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	def, body, _, base, ok := h.mailTemplateInput(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	to := strings.TrimSpace(body.To)
	if to == "" {
		if err := h.dbOf(ctx).QueryRow(ctx, `SELECT email FROM core.users WHERE id = $1`, claims.UserID).Scan(&to); err != nil {
			writeError(w, http.StatusUnprocessableEntity, "Укажите адрес для проверочного письма")
			return
		}
	}
	if !mailer.ValidAddress(to) {
		writeError(w, http.StatusUnprocessableEntity, "Адрес получателя выглядит неверно")
		return
	}
	cfg := h.mailerConfig(ctx)
	if cfg.Silent() {
		writeError(w, http.StatusUnprocessableEntity, `Сейчас выбран режим "`+cfg.MailerName()+`": письмо не уйдёт на почту`)
		return
	}
	if !cfg.Configured() {
		writeError(w, http.StatusUnprocessableEntity, "SMTP не настроен: укажите адрес сервера в настройках")
		return
	}
	msg := h.mailTemplateLetter(def, base, body.Values, r).Message(h.mailBrand(ctx, r))
	msg.To = to
	if err := h.sendMail(ctx, def.ID, claims.UserID, to, msg); err != nil {
		writeError(w, http.StatusBadGateway, "Не удалось отправить письмо: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "sent", "to": to})
}

func (h *Handler) AdminMailLog(w http.ResponseWriter, r *http.Request) {
	if _, ok := tenantClaims(r.Context()); !ok {
		return
	}
	ctx := r.Context()
	limit := 50
	if n, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && n > 0 && n <= 200 {
		limit = n
	}
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	if status != "sent" && status != "failed" && status != "skipped" {
		status = ""
	}

	rows, err := h.readerOf(ctx).Query(ctx, `
		SELECT l.id::text, l.template, l.to_address, l.subject, l.status, l.error,
		       l.mailer, l.created_at, COALESCE(u.email, '')
		FROM core.mail_log l
		LEFT JOIN core.users u ON u.id = l.user_id
		WHERE ($1 = '' OR l.status = $1)
		ORDER BY l.created_at DESC
		LIMIT $2
	`, status, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Не удалось загрузить журнал писем")
		return
	}
	defer rows.Close()

	items := []map[string]any{}
	for rows.Next() {
		var id, template, to, subject, state, failure, mailerName, user string
		var created time.Time
		if rows.Scan(&id, &template, &to, &subject, &state, &failure, &mailerName, &created, &user) != nil {
			continue
		}
		items = append(items, map[string]any{
			"id":         id,
			"template":   template,
			"to":         to,
			"subject":    subject,
			"status":     state,
			"error":      failure,
			"mailer":     mailerName,
			"user_email": user,
			"created_at": created,
		})
	}

	var sent, failed, skipped int
	_ = h.readerOf(ctx).QueryRow(ctx, `
		SELECT count(*) FILTER (WHERE status = 'sent'),
		       count(*) FILTER (WHERE status = 'failed'),
		       count(*) FILTER (WHERE status = 'skipped')
		FROM core.mail_log
		WHERE created_at > now() - interval '7 days'
	`).Scan(&sent, &failed, &skipped)

	writeJSON(w, http.StatusOK, map[string]any{
		"items": items,
		"week":  map[string]int{"sent": sent, "failed": failed, "skipped": skipped},
	})
}
