package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/vortanix/vortanix/internal/api/mail"
	"github.com/vortanix/vortanix/internal/api/payments"
	"github.com/vortanix/vortanix/pkg/secretbox"
	"golang.org/x/crypto/ssh"
)

const maskedFTPPassword = "__VTX_MASKED_FTP_PASSWORD__"

var paymentProviderFormKey = regexp.MustCompile(`^payment_providers\[([^\]]+)\]\[([^\]]+)\]$`)

func (h *Handler) GetAdminSettings(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	ctx := r.Context()
	values := h.loadTenantSettingStrings(ctx, claims.TenantID)

	if pwd := strings.TrimSpace(values["files.storage.ftp.password"]); pwd != "" {
		values["files.storage.ftp.password"] = maskedFTPPassword
	}

	providers := h.buildAdminPaymentProviders(ctx, claims.TenantID)
	writeJSON(w, http.StatusOK, map[string]any{
		"values":            values,
		"paymentProviders":  providers,
		"payment_providers": providers,
		"panelVersion":      h.currentPanelVersion(values),
		"panel_version":     h.currentPanelVersion(values),
	})
}

func (h *Handler) UpdateAdminSettings(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	ctx := r.Context()

	if r.Method == http.MethodPatch || strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid json")
			return
		}
		for k, v := range body {
			h.setTenantSettingAny(ctx, claims.TenantID, k, v)
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "message": "Настройки сохранены", "status": "updated"})
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "invalid multipart form")
		return
	}

	form := r.MultipartForm.Value
	for formKey, settingKey := range adminSettingsFormToDotKey() {
		if vals, ok := form[formKey]; ok && len(vals) > 0 {
			h.setTenantSettingString(ctx, claims.TenantID, settingKey, vals[0])
		}
	}

	h.applyBoolSetting(ctx, claims.TenantID, form, "require_verified_email", "auth.require_verified_email")
	h.applyBoolSetting(ctx, claims.TenantID, form, "server_status_notifications", "vtx_mail.server_status_notifications")
	h.applyBoolSetting(ctx, claims.TenantID, form, "telegram_notifications_enabled", "telegram.notifications.enabled")
	h.applyBoolSetting(ctx, claims.TenantID, form, "files_storage_ftp_passive", "files.storage.ftp.passive")
	h.applyBoolSetting(ctx, claims.TenantID, form, "files_storage_ftp_ssl", "files.storage.ftp.ssl")
	h.applyBoolSetting(ctx, claims.TenantID, form, "files_storage_s3_use_path_style_endpoint", "files.storage.s3.use_path_style_endpoint")

	if vals, ok := form["files_storage_ftp_password"]; ok {
		h.setTenantSettingString(ctx, claims.TenantID, "files.storage.ftp.password", h.resolveMaskedFTPPassword(ctx, claims.TenantID, vals[0]))
	}

	if vals, ok := form["mail_scheme"]; ok {
		scheme := strings.ToLower(strings.TrimSpace(vals[0]))
		switch scheme {
		case "ssl":
			scheme = "smtps"
		case "tls":
			scheme = "smtp"
		}
		h.setTenantSettingString(ctx, claims.TenantID, "mail.mailers.smtp.scheme", scheme)
	}

	if file, header, err := r.FormFile("logo"); err == nil {
		defer file.Close()
		if path, err := h.saveBrandingFile(file, header.Filename, "logo"); err == nil {
			h.setTenantSettingString(ctx, claims.TenantID, "app.branding.logo", path)
		}
	}
	if file, header, err := r.FormFile("icon"); err == nil {
		defer file.Close()
		if path, err := h.saveBrandingFile(file, header.Filename, "icon"); err == nil {
			h.setTenantSettingString(ctx, claims.TenantID, "app.branding.icon", path)
		}
	}

	h.savePaymentProvidersFromForm(ctx, claims.TenantID, form)

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "message": "Настройки сохранены"})
}

func (h *Handler) AdminSettingsTestMail(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	var body struct {
		TestMailTo string `json:"test_mail_to"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.TestMailTo) == "" {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"ok": false, "message": "Укажите email для тестового письма"})
		return
	}
	to := strings.TrimSpace(body.TestMailTo)
	cfg := h.mailConfigFromTenant(r.Context(), claims.TenantID)
	mailer := h.tenantSettingString(r.Context(), claims.TenantID, "mail.default")
	if mailer == "" {
		mailer = "smtp"
	}
	if mailer == "log" || mailer == "array" {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"ok": false, "message": fmt.Sprintf(`Сейчас выбран mailer "%s": письмо не будет отправлено на email. Выберите mailer "smtp" в настройках и повторите.`, mailer),
		})
		return
	}
	if !cfg.Enabled() {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"ok": false, "message": "SMTP не настроен"})
		return
	}
	subject := "Vortanix test mail"
	bodyHTML := fmt.Sprintf("<p>Тестовое письмо отправлено на %s</p>", to)
	if err := cfg.Send(to, subject, bodyHTML); err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"ok": false, "message": fmt.Sprintf("Не удалось отправить тестовое письмо (mailer: %s): %s", mailer, err.Error()),
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok": true, "message": fmt.Sprintf("Тестовое письмо отправлено на %s (mailer: %s)", to, mailer),
	})
}

func (h *Handler) AdminSettingsTestTelegram(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	ctx := r.Context()
	token := h.tenantSettingString(ctx, claims.TenantID, "telegram.notifications.bot_token")
	if token == "" {
		token = h.telegramBotToken
	}
	if token == "" {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"ok": false, "message": "Bot Token не указан"})
		return
	}
	botUsername, err := h.telegramGetMe(token)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"ok": false, "message": "Telegram Bot: " + err.Error()})
		return
	}
	chatID := h.tenantSettingString(ctx, claims.TenantID, "telegram.notifications.admin_chat_id")
	if chatID == "" {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"ok": false, "message": "Admin Chat ID не указан."})
		return
	}
	appName := h.tenantSettingString(ctx, claims.TenantID, "app.name")
	if appName == "" {
		appName = "Vortanix"
	}
	msg := fmt.Sprintf("✅ <b>Тестовое сообщение</b>\nУведомления %s работают!", appName)
	if err := h.telegramSendMessage(token, chatID, msg); err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"ok": false, "message": "Не удалось отправить сообщение. Проверьте Chat ID и убедитесь что бот добавлен в чат.",
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok": true, "message": fmt.Sprintf("Тестовое сообщение отправлено в Telegram (bot: @%s)", botUsername),
	})
}

func (h *Handler) AdminSettingsTestFilesStorage(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	var in map[string]any
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	driver := strings.ToLower(strings.TrimSpace(fmt.Sprint(in["files_storage_driver"])))
	switch driver {
	case "sftp":
		if err := testSFTPConnection(in, h.resolveMaskedFTPPassword(r.Context(), claims.TenantID, fmt.Sprint(in["files_storage_sftp_password"]))); err != nil {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"ok": false, "message": "Ошибка подключения: " + err.Error()})
			return
		}
	case "s3":
		if err := testS3Storage(in); err != nil {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"ok": false, "message": "Ошибка подключения: " + err.Error()})
			return
		}
	default:
		pwd := fmt.Sprint(in["files_storage_ftp_password"])
		if pwd == maskedFTPPassword {
			pwd = h.tenantSettingString(r.Context(), claims.TenantID, "files.storage.ftp.password")
		}
		if err := testFTPStorage(in, pwd); err != nil {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"ok": false, "message": "Ошибка подключения: " + err.Error()})
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "message": "Подключение к файловому хранилищу успешно."})
}

func (h *Handler) AdminCutoverReadiness(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	ctx := r.Context()
	values := h.loadTenantSettingStrings(ctx, claims.TenantID)
	mailer := strings.TrimSpace(values["mail.default"])
	if mailer == "" {
		mailer = "smtp"
	}
	smtpConfigured := h.mailConfigFromTenant(ctx, claims.TenantID).Enabled()
	providers := h.buildAdminPaymentProviders(ctx, claims.TenantID)
	enabledProviders := 0
	providersWithWebhook := 0
	for _, raw := range providers {
		p, ok := raw.(map[string]any)
		if !ok || !boolFromAny(p["enabled"]) {
			continue
		}
		enabledProviders++
		code := strings.TrimSpace(fmt.Sprint(p["code"]))
		switch code {
		case "stripe":
			cfg, _ := p["config"].(map[string]any)
			if strings.TrimSpace(fmt.Sprint(cfg["webhook_secret"])) != "" {
				providersWithWebhook++
			}
		case "freekassa", "robokassa", "yookassa":
			providersWithWebhook++
		}
	}
	oauthProviders := map[string]bool{
		"google":  strings.TrimSpace(envOr("SOCIAL_GOOGLE_CLIENT_ID", "")) != "",
		"discord": strings.TrimSpace(envOr("SOCIAL_DISCORD_CLIENT_ID", "")) != "",
		"vk":      strings.TrimSpace(envOr("SOCIAL_VK_CLIENT_ID", "")) != "",
	}
	oauthReady := false
	for _, ready := range oauthProviders {
		if ready {
			oauthReady = true
			break
		}
	}
	blockers := []map[string]any{}
	if !smtpConfigured || mailer == "log" || mailer == "array" {
		blockers = append(blockers, map[string]any{
			"key":      "mail",
			"message":  "Почта не настроена — письма для восстановления пароля и подтверждения адреса не отправляются.",
			"fallback": "Пока настраиваете SMTP, подтверждайте адреса вручную в разделе «Пользователи».",
		})
	}
	if !oauthReady {
		blockers = append(blockers, map[string]any{
			"key":      "oauth",
			"message":  "Вход через Google, Discord и VK не настроен — не заданы ключи приложений.",
			"fallback": "Вход по логину с паролем и двухфакторной проверкой работает как обычно.",
		})
	}
	if enabledProviders == 0 {
		blockers = append(blockers, map[string]any{
			"key":      "payments",
			"message":  "Не включён ни один способ оплаты — клиенты не смогут пополнить баланс сами.",
			"fallback": "Баланс можно пополнить вручную из карточки пользователя.",
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok": len(blockers) == 0,
		"checks": map[string]any{
			"mail": map[string]any{
				"mailer":          mailer,
				"smtp_configured": smtpConfigured,
			},
			"oauth": map[string]any{
				"configured": oauthProviders,
				"any_ready":  oauthReady,
			},
			"payments": map[string]any{
				"enabled_providers":      enabledProviders,
				"providers_with_webhook": providersWithWebhook,
			},
		},
		"blockers": blockers,
	})
}

func (h *Handler) loadTenantSettingStrings(ctx context.Context, tenantID string) map[string]string {
	out := map[string]string{}
	rows, err := h.dbOf(ctx).Query(ctx, `SELECT key, value FROM core.tenant_settings WHERE tenant_id = $1`, tenantID)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var k string
		var v []byte
		if rows.Scan(&k, &v) == nil {
			out[k] = h.secrets.MustDecrypt(jsonValueToString(v))
		}
	}
	return out
}

var secretSettingKeys = map[string]bool{
	"services.recaptcha.secret_key":    true,
	"services.google.client_secret":    true,
	"services.discord.client_secret":   true,
	"services.vkontakte.client_secret": true,
	"dockerhub.token":                  true,
	"mail.mailers.smtp.password":       true,
	"telegram.notifications.bot_token": true,
	"files.storage.ftp.password":       true,
	"files.storage.s3.secret":          true,
	"files.storage.sftp.password":      true,
}

func (h *Handler) tenantSettingString(ctx context.Context, tenantID, key string) string {
	var raw []byte
	if h.dbOf(ctx).QueryRow(ctx, `SELECT value FROM core.tenant_settings WHERE tenant_id = $1 AND key = $2`, tenantID, key).Scan(&raw) != nil {
		return ""
	}
	return h.secrets.MustDecrypt(jsonValueToString(raw))
}

func (h *Handler) setTenantSettingString(ctx context.Context, tenantID, key, value string) {
	if secretSettingKeys[key] {
		if value == "" && h.storedSecretUnreadable(ctx, tenantID, key) {
			return
		}
		sealed, err := h.secrets.Encrypt(value)
		if err != nil {
			log.Printf("настройка %s не зашифрована: %v", key, err)
			return
		}
		value = sealed
	}
	b, _ := json.Marshal(value)
	// Ошибку не глушим. Именно из-за неё настройки не сохранялись молча:
	// форма отвечала «сохранено», а запись падала — например, когда
	// арендатора нет в той базе, куда её адресовали.
	if _, err := h.dbOf(ctx).Exec(ctx, `
		INSERT INTO core.tenant_settings (tenant_id, key, value) VALUES ($1, $2, $3::jsonb)
		ON CONFLICT (tenant_id, key) DO UPDATE SET value = EXCLUDED.value, updated_at = now()
	`, tenantID, key, b); err != nil {
		log.Printf("настройка %s арендатора %s не сохранена: %v", key, tenantID, err)
	}
}

func (h *Handler) storedSecretUnreadable(ctx context.Context, tenantID, key string) bool {
	var raw []byte
	if h.dbOf(ctx).QueryRow(ctx, `SELECT value FROM core.tenant_settings WHERE tenant_id = $1 AND key = $2`, tenantID, key).Scan(&raw) != nil {
		return false
	}
	stored := jsonValueToString(raw)
	if !secretbox.IsEncrypted(stored) {
		return false
	}
	_, err := h.secrets.Decrypt(stored)
	return err != nil
}

func (h *Handler) setTenantSettingAny(ctx context.Context, tenantID, key string, value any) {
	if secretSettingKeys[key] {
		if str, ok := value.(string); ok {
			h.setTenantSettingString(ctx, tenantID, key, str)
			return
		}
	}
	b, _ := json.Marshal(value)
	_, _ = h.dbOf(ctx).Exec(ctx, `
		INSERT INTO core.tenant_settings (tenant_id, key, value) VALUES ($1, $2, $3::jsonb)
		ON CONFLICT (tenant_id, key) DO UPDATE SET value = EXCLUDED.value, updated_at = now()
	`, tenantID, key, b)
}

func jsonValueToString(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var n json.Number
	if json.Unmarshal(raw, &n) == nil {
		return n.String()
	}
	var b bool
	if json.Unmarshal(raw, &b) == nil {
		if b {
			return "1"
		}
		return "0"
	}
	return strings.Trim(string(raw), `"`)
}

func (h *Handler) applyBoolSetting(ctx context.Context, tenantID string, form map[string][]string, formKey, settingKey string) {
	val := "0"
	if vals, ok := form[formKey]; ok && formTruthy(vals[0]) {
		val = "1"
	}
	h.setTenantSettingString(ctx, tenantID, settingKey, val)
}

func formTruthy(v string) bool {
	v = strings.TrimSpace(strings.ToLower(v))
	return v == "1" || v == "true" || v == "on" || v == "yes"
}

func (h *Handler) resolveMaskedFTPPassword(ctx context.Context, tenantID, input string) string {
	if input != maskedFTPPassword {
		return input
	}
	return h.tenantSettingString(ctx, tenantID, "files.storage.ftp.password")
}

func (h *Handler) currentPanelVersion(values map[string]string) string {
	for _, k := range []string{"panel.version", "app.version"} {
		if v := strings.TrimSpace(values[k]); v != "" {
			return v
		}
	}
	if v := strings.TrimSpace(envOr("APP_VERSION", "0.0.0")); v != "" {
		return v
	}
	return "0.0.0"
}

func (h *Handler) buildAdminPaymentProviders(ctx context.Context, tenantID string) map[string]any {
	stored := map[string]struct {
		enabled bool
		config  map[string]any
	}{}
	rows, _ := h.dbOf(ctx).Query(ctx, `SELECT provider, enabled, config FROM core.payment_providers WHERE tenant_id = $1`, tenantID)
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var provider string
			var enabled bool
			var cfg []byte
			if rows.Scan(&provider, &enabled, &cfg) == nil {
				m := map[string]any{}
				if plain, err := h.secrets.DecryptJSON(cfg); err == nil {
					_ = json.Unmarshal(plain, &m)
				}
				stored[provider] = struct {
					enabled bool
					config  map[string]any
				}{enabled: enabled, config: m}
			}
		}
	}

	out := map[string]any{}
	for _, def := range payments.AdminCatalog() {
		row := stored[def.Key]
		cfg := map[string]string{}
		for fk, field := range def.Fields {
			if v, ok := row.config[fk]; ok {
				cfg[fk] = fmt.Sprint(v)
			} else if field.Default != "" {
				cfg[fk] = field.Default
			} else {
				cfg[fk] = ""
			}
		}
		fields := map[string]any{}
		for fk, field := range def.Fields {
			entry := map[string]any{"label": field.Label}
			if field.Type != "" {
				entry["type"] = field.Type
			}
			if field.Default != "" {
				entry["default"] = field.Default
			}
			if len(field.Options) > 0 {
				entry["options"] = field.Options
			}
			fields[fk] = entry
		}
		out[def.Key] = map[string]any{
			"key":       def.Key,
			"name":      def.Name,
			"fields":    fields,
			"enabled":   row.enabled,
			"config":    cfg,
			"supported": payments.IsSupported(def.Key),
		}
	}
	return out
}

func (h *Handler) unreadableProviderConfigs(ctx context.Context, tenantID string) map[string]bool {
	out := map[string]bool{}
	rows, err := h.dbOf(ctx).Query(ctx, `SELECT provider, config FROM core.payment_providers WHERE tenant_id = $1`, tenantID)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var provider string
		var cfg []byte
		if rows.Scan(&provider, &cfg) != nil {
			continue
		}
		if _, err := h.secrets.DecryptJSON(cfg); err != nil {
			out[provider] = true
		}
	}
	return out
}

func (h *Handler) savePaymentProvidersFromForm(ctx context.Context, tenantID string, form map[string][]string) {
	input := map[string]map[string]string{}
	enabled := map[string]bool{}
	for key, vals := range form {
		if m := paymentProviderFormKey.FindStringSubmatch(key); len(m) == 3 {
			prov, field := m[1], m[2]
			if input[prov] == nil {
				input[prov] = map[string]string{}
			}
			if len(vals) > 0 {
				if field == "enabled" {
					enabled[prov] = formTruthy(vals[0])
				} else {
					input[prov][field] = vals[0]
				}
			}
		}
	}
	unreadable := h.unreadableProviderConfigs(ctx, tenantID)

	catalog := payments.AdminCatalogMap()
	for provKey, provDef := range catalog {
		if unreadable[provKey] {
			continue
		}
		provInput := input[provKey]
		cfg := map[string]any{}
		for fieldKey, fieldDef := range provDef.Fields {
			val := strings.TrimSpace(provInput[fieldKey])
			if fieldDef.Type == "checkbox" {
				if formTruthy(val) {
					cfg[fieldKey] = "1"
				} else {
					cfg[fieldKey] = "0"
				}
				continue
			}
			if val == "" {
				if fieldDef.Default != "" {
					cfg[fieldKey] = fieldDef.Default
				}
				continue
			}
			cfg[fieldKey] = val
		}
		en := enabled[provKey]
		raw, _ := json.Marshal(cfg)
		sealed, err := h.secrets.EncryptJSON(raw)
		if err != nil {
			continue
		}
		raw = sealed
		_, _ = h.dbOf(ctx).Exec(ctx, `
			INSERT INTO core.payment_providers (tenant_id, provider, enabled, config)
			VALUES ($1, $2, $3, $4::jsonb)
			ON CONFLICT (tenant_id, provider) DO UPDATE SET enabled = EXCLUDED.enabled, config = EXCLUDED.config
		`, tenantID, provKey, en, raw)
	}
}

func (h *Handler) saveBrandingFile(file io.Reader, filename, base string) (string, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".png", ".jpg", ".jpeg", ".webp", ".svg", ".ico":
	default:
		return "", fmt.Errorf("unsupported file type")
	}
	dir := filepath.Join(h.uploadDir, "branding")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	dest := filepath.Join(dir, base+ext)
	out, err := os.Create(dest)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(out, file); err != nil {
		out.Close()
		return "", err
	}
	out.Close()
	return "branding/" + base + ext, nil
}

func (h *Handler) mailConfigFromTenant(ctx context.Context, tenantID string) mail.Config {
	host := h.tenantSettingString(ctx, tenantID, "mail.mailers.smtp.host")
	port := h.tenantSettingString(ctx, tenantID, "mail.mailers.smtp.port")
	user := h.tenantSettingString(ctx, tenantID, "mail.mailers.smtp.username")
	pass := h.tenantSettingString(ctx, tenantID, "mail.mailers.smtp.password")
	from := h.tenantSettingString(ctx, tenantID, "mail.from.address")
	if host == "" {
		return h.mail
	}
	if port == "" {
		port = "587"
	}
	if from == "" {
		from = h.mail.From
	}
	return mail.Config{Host: host, Port: port, User: user, Pass: pass, From: from}
}

func (h *Handler) telegramGetMe(token string) (string, error) {
	resp, err := http.Get("https://api.telegram.org/bot" + token + "/getMe")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var parsed struct {
		OK     bool `json:"ok"`
		Result struct {
			Username string `json:"username"`
		} `json:"result"`
		Description string `json:"description"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&parsed)
	if !parsed.OK {
		if parsed.Description != "" {
			return "", fmt.Errorf("%s", parsed.Description)
		}
		return "", fmt.Errorf("getMe failed")
	}
	return parsed.Result.Username, nil
}

func (h *Handler) telegramSendMessage(token, chatID, text string) error {
	form := url.Values{}
	form.Set("chat_id", chatID)
	form.Set("text", text)
	form.Set("parse_mode", "HTML")
	resp, err := http.PostForm("https://api.telegram.org/bot"+token+"/sendMessage", form)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var parsed struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&parsed)
	if !parsed.OK {
		if parsed.Description != "" {
			return fmt.Errorf("%s", parsed.Description)
		}
		return fmt.Errorf("send failed")
	}
	return nil
}

func testSFTPConnection(in map[string]any, password string) error {
	host := fmt.Sprint(in["files_storage_sftp_host"])
	port := fmt.Sprint(in["files_storage_sftp_port"])
	if port == "" {
		port = "22"
	}
	user := fmt.Sprint(in["files_storage_sftp_username"])
	timeout := 30 * time.Second
	if t := fmt.Sprint(in["files_storage_sftp_timeout"]); t != "" {
		if sec, err := strconv.Atoi(t); err == nil && sec > 0 {
			timeout = time.Duration(sec) * time.Second
		}
	}
	cfg := &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{ssh.Password(password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         timeout,
	}
	client, err := ssh.Dial("tcp", net.JoinHostPort(host, port), cfg)
	if err != nil {
		return err
	}
	defer client.Close()
	sess, err := client.NewSession()
	if err != nil {
		return err
	}
	defer sess.Close()
	return sess.Run("echo vtx-test")
}

func adminSettingsFormToDotKey() map[string]string {
	return map[string]string{
		"app_name": "app.name", "site_description": "app.site.description", "site_domain": "app.site.domain",
		"site_ip": "app.site.ip", "site_subnet": "app.site.subnet", "default_template": "app.site.default_template",
		"recaptcha_site_key": "services.recaptcha.site_key", "recaptcha_secret_key": "services.recaptcha.secret_key",
		"google_client_id": "services.google.client_id", "google_client_secret": "services.google.client_secret",
		"google_redirect_uri": "services.google.redirect", "discord_client_id": "services.discord.client_id",
		"discord_client_secret": "services.discord.client_secret", "discord_redirect_uri": "services.discord.redirect",
		"vk_client_id": "services.vkontakte.client_id", "vk_client_secret": "services.vkontakte.client_secret",
		"vk_redirect_uri": "services.vkontakte.redirect", "telegram_url": "app.links.telegram",
		"discord_url": "app.links.discord", "support_url": "app.links.support",
		"mail_mailer": "mail.default", "mail_host": "mail.mailers.smtp.host", "mail_port": "mail.mailers.smtp.port",
		"mail_username": "mail.mailers.smtp.username", "mail_password": "mail.mailers.smtp.password",
		"mail_from_address": "mail.from.address", "mail_from_name": "mail.from.name",
		"payment_fee_freekassa": "payments.providers.freekassa.fee_percent", "payment_fee_nowpayments": "payments.providers.nowpayments.fee_percent",
		"payment_fee_stripe": "payments.providers.stripe.fee_percent", "payment_fee_paypal": "payments.providers.paypal.fee_percent",
		"payment_fee_yookassa": "payments.providers.yookassa.fee_percent", "payment_fee_yoomoney": "payments.providers.yoomoney.fee_percent",
		"payment_fee_cloudpayments": "payments.providers.cloudpayments.fee_percent", "payment_fee_unitpay": "payments.providers.unitpay.fee_percent",
		"payment_fee_robokassa": "payments.providers.robokassa.fee_percent", "payment_fee_cryptocloud": "payments.providers.cryptocloud.fee_percent",
		"payment_fee_coinbase": "payments.providers.coinbase.fee_percent", "fx_fee_percent": "payments.fx.fee_percent",
		"dockerhub_username": "dockerhub.username", "dockerhub_token": "dockerhub.token",
		"telegram_bot_token": "telegram.notifications.bot_token", "telegram_bot_username": "telegram.notifications.bot_username",
		"telegram_admin_chat_id": "telegram.notifications.admin_chat_id",
		"files_storage_driver":   "files.storage.driver", "files_storage_url": "files.storage.url",
		"files_storage_ftp_host": "files.storage.ftp.host", "files_storage_ftp_port": "files.storage.ftp.port",
		"files_storage_ftp_username": "files.storage.ftp.username", "files_storage_ftp_root": "files.storage.ftp.root",
		"files_storage_ftp_timeout": "files.storage.ftp.timeout",
		"files_storage_s3_key":      "files.storage.s3.key", "files_storage_s3_secret": "files.storage.s3.secret",
		"files_storage_s3_region": "files.storage.s3.region", "files_storage_s3_bucket": "files.storage.s3.bucket",
		"files_storage_s3_endpoint": "files.storage.s3.endpoint",
		"files_storage_sftp_host":   "files.storage.sftp.host", "files_storage_sftp_port": "files.storage.sftp.port",
		"files_storage_sftp_username": "files.storage.sftp.username", "files_storage_sftp_password": "files.storage.sftp.password",
		"files_storage_sftp_root": "files.storage.sftp.root", "files_storage_sftp_timeout": "files.storage.sftp.timeout",
		"files_storage_path_avatars": "files.storage.path.avatars", "files_storage_path_plugins": "files.storage.path.plugins",
		"files_storage_path_maps": "files.storage.path.maps", "files_storage_path_support": "files.storage.path.support",
	}
}
