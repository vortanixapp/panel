package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/vortanixapp/panel/internal/api/payments"
)

const (
	identificationRequiredSetting  = "identification.required"
	identificationProvidersSetting = "identification.payment_providers"
	cookieBannerSetting            = "legal.cookie_banner"
)

var defaultIdentificationProviders = []string{"yookassa", "tkassa", "robokassa", "cloudpayments", "bank"}

type identificationSettings struct {
	Required  bool     `json:"required"`
	Providers []string `json:"providers"`
}

func identificationSettingsFrom(settings map[string]string) identificationSettings {
	s := identificationSettings{Required: truthySetting(settings[identificationRequiredSetting])}
	raw := strings.TrimSpace(settings[identificationProvidersSetting])
	if raw == "" || json.Unmarshal([]byte(raw), &s.Providers) != nil {
		s.Providers = append([]string{}, defaultIdentificationProviders...)
	}
	return s
}

func (h *Handler) identificationSettings(ctx context.Context) identificationSettings {
	return identificationSettingsFrom(h.loadTenantSettingStrings(ctx))
}

func (h *Handler) identifyUser(ctx context.Context, userID, method, note, paymentID, actorID string) (bool, error) {
	db := h.dbOf(ctx)
	tag, err := db.Exec(ctx, `
		UPDATE core.users
		SET identified_at = now(), identification_method = $2, identification_note = $3
		WHERE id = $1 AND identified_at IS NULL AND deleted_at IS NULL
	`, userID, method, note)
	if err != nil || tag.RowsAffected() == 0 {
		return false, err
	}
	_, err = db.Exec(ctx, `
		INSERT INTO core.identification_events (user_id, action, method, note, payment_id, actor_id)
		VALUES ($1, 'identified', $2, $3, NULLIF($4, '')::uuid, NULLIF($5, '')::uuid)
	`, userID, method, note, paymentID, actorID)
	return true, err
}

func (h *Handler) identifyByPayment(ctx context.Context, paymentID string) {
	var userID, provider string
	var invoice int64
	if h.dbOf(ctx).QueryRow(ctx, `
		SELECT user_id::text, provider, invoice_no FROM core.payments WHERE id = $1
	`, paymentID).Scan(&userID, &provider, &invoice) != nil {
		return
	}
	if !slices.Contains(h.identificationSettings(ctx).Providers, provider) {
		return
	}
	method := "payment"
	if provider == "bank" {
		method = "bank_transfer"
	}
	note := payments.Name(provider) + ", счёт № " + strconv.FormatInt(invoice, 10)
	if _, err := h.identifyUser(ctx, userID, method, note, paymentID, ""); err != nil {
		log.Printf("идентификация %s по платежу %s не записана: %v", userID, paymentID, err)
	}
}

func (h *Handler) refuseUnidentified(ctx context.Context, w http.ResponseWriter, userID, role string) bool {
	if isStaffRole(role) || !h.identificationSettings(ctx).Required {
		return false
	}
	var identified bool
	if h.dbOf(ctx).QueryRow(ctx, `
		SELECT identified_at IS NOT NULL FROM core.users WHERE id = $1
	`, userID).Scan(&identified) == nil && identified {
		return false
	}
	writeCodedError(w, http.StatusForbidden, "identification_required",
		"Перед заказом услуг нужно пройти идентификацию: пополните баланс картой российского банка, через СБП или банковским переводом либо обратитесь в поддержку")
	return true
}

func (h *Handler) AccountIdentification(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ctx := r.Context()
	db := h.dbOf(ctx)
	settings := h.identificationSettings(ctx)
	var identifiedAt *time.Time
	var method string
	_ = db.QueryRow(ctx, `
		SELECT identified_at, identification_method FROM core.users WHERE id = $1
	`, claims.UserID).Scan(&identifiedAt, &method)

	enabled := map[string]bool{}
	if rows, err := db.Query(ctx, `SELECT provider FROM core.payment_providers WHERE enabled`); err == nil {
		for rows.Next() {
			var code string
			if rows.Scan(&code) == nil {
				enabled[code] = true
			}
		}
		rows.Close()
	}
	methods := []string{}
	for _, code := range settings.Providers {
		if enabled[code] {
			methods = append(methods, payments.Name(code))
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"required":      settings.Required && !isStaffRole(claims.Role),
		"identified":    identifiedAt != nil,
		"identified_at": isoOrNil(identifiedAt),
		"method":        method,
		"methods":       methods,
	})
}

func (h *Handler) AdminUserIdentification(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	userID := chi.URLParam(r, "id")
	var body struct {
		Identified bool   `json:"identified"`
		Note       string `json:"note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	note := strings.TrimSpace(body.Note)
	if note == "" {
		writeError(w, http.StatusBadRequest, "Укажите основание: каким документом или способом проверена личность клиента")
		return
	}
	ctx := r.Context()
	db := h.dbOf(ctx)
	if body.Identified {
		done, err := h.identifyUser(ctx, userID, "manual", note, "", claims.UserID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "database error")
			return
		}
		if !done {
			writeError(w, http.StatusConflict, "Клиент уже идентифицирован или удалён")
			return
		}
	} else {
		tag, err := db.Exec(ctx, `
			UPDATE core.users SET identified_at = NULL, identification_method = '', identification_note = ''
			WHERE id = $1 AND identified_at IS NOT NULL
		`, userID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "database error")
			return
		}
		if tag.RowsAffected() == 0 {
			writeError(w, http.StatusConflict, "Клиент не идентифицирован")
			return
		}
		_, _ = db.Exec(ctx, `
			INSERT INTO core.identification_events (user_id, action, note, actor_id)
			VALUES ($1, 'revoked', $2, $3)
		`, userID, note, claims.UserID)
	}
	audit(ctx, db, claims.UserID, "user.identification", "user:"+userID, map[string]any{
		"identified": body.Identified, "note": note,
	})
	writeJSON(w, http.StatusOK, map[string]any{"identified": body.Identified})
}

func (h *Handler) userDeletionBlocker(ctx context.Context, userID string, ignoreBalance bool) string {
	db := h.dbOf(ctx)
	var servers, hosting, pendingRefunds int
	var balance float64
	_ = db.QueryRow(ctx, `SELECT COUNT(*) FROM core.servers WHERE user_id = $1`, userID).Scan(&servers)
	_ = db.QueryRow(ctx, `
		SELECT COUNT(*) FROM core.hosting_accounts WHERE user_id = $1 AND status <> 'terminated'
	`, userID).Scan(&hosting)
	_ = db.QueryRow(ctx, `
		SELECT COUNT(*) FROM core.balance_refund_requests WHERE user_id = $1 AND status = 'pending'
	`, userID).Scan(&pendingRefunds)
	_ = db.QueryRow(ctx, `
		SELECT COALESCE(SUM(balance), 0)::float8 FROM core.wallets WHERE user_id = $1
	`, userID).Scan(&balance)
	switch {
	case servers > 0:
		return "У пользователя есть игровые серверы — сначала удалите их"
	case hosting > 0:
		return "У пользователя есть аккаунты веб-хостинга — сначала закройте их"
	case pendingRefunds > 0:
		return "Есть необработанная заявка на возврат остатка баланса"
	case !ignoreBalance && balance > 0.004:
		return "На балансе остались деньги — сначала верните остаток через заявку на возврат"
	}
	return ""
}

func (h *Handler) anonymizeUser(ctx context.Context, userID, actorID, source string) error {
	db := h.dbOf(ctx)
	tag, err := db.Exec(ctx, `
		UPDATE core.users
		SET email = 'deleted-' || replace(id::text, '-', '') || '@deleted.invalid',
		    password_hash = '!', status = 'disabled', two_factor_enabled = false,
		    deleted_at = now(), retain_until = (now() + interval '5 years')::date
		WHERE id = $1 AND role <> 'owner' AND deleted_at IS NULL
	`, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	for _, stmt := range []string{
		`UPDATE core.user_profiles SET display_name = NULL, first_name = NULL, last_name = NULL, phone = NULL,
		        avatar_url = NULL, contacts = '{}'::jsonb, updated_at = now() WHERE user_id = $1`,
		`UPDATE core.user_billing_profiles SET inn = '', address = '', updated_at = now()
		 WHERE user_id = $1 AND payer_type = 'person'`,
		`DELETE FROM core.user_sessions WHERE user_id = $1`,
		`DELETE FROM core.two_factor_secrets WHERE user_id = $1`,
		`DELETE FROM core.password_reset_tokens WHERE user_id = $1`,
		`DELETE FROM core.user_social_accounts WHERE user_id = $1`,
		`DELETE FROM core.user_notification_channels WHERE user_id = $1`,
		`DELETE FROM core.notification_deliveries WHERE user_id = $1`,
		`DELETE FROM core.notifications WHERE user_id = $1`,
		`DELETE FROM core.server_friends WHERE user_id = $1`,
		`DELETE FROM core.whmcs_sso_tokens WHERE user_id = $1`,
		`DELETE FROM core.support_tickets WHERE user_id = $1`,
		`UPDATE core.login_attempts SET email = '', user_agent = '' WHERE user_id = $1`,
		`UPDATE core.api_keys SET revoked_at = now() WHERE user_id = $1 AND revoked_at IS NULL`,
	} {
		if _, err := db.Exec(ctx, stmt, userID); err != nil {
			log.Printf("обезличивание пользователя %s: %v", userID, err)
		}
	}
	if _, err := db.Exec(ctx, `
		INSERT INTO core.legal_consents (user_id, kind, version, action, source)
		SELECT $1::uuid, d.kind, MAX(d.version), 'withdrawn', $2::text
		FROM core.legal_documents d
		WHERE d.kind = 'consent'
		GROUP BY d.kind
	`, userID, "account_deletion:"+source); err != nil {
		log.Printf("отзыв согласия пользователя %s не записан: %v", userID, err)
	}
	audit(ctx, db, actorID, "user.delete", "user:"+userID, map[string]any{"source": source, "anonymized": true})
	return nil
}

func (h *Handler) AccountDelete(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if isStaffRole(claims.Role) {
		writeError(w, http.StatusForbidden, "Учётную запись сотрудника удаляет владелец панели")
		return
	}
	var body struct {
		ConfirmEmail string `json:"confirm_email"`
		Password     string `json:"password"`
		Code         string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	ctx := r.Context()
	var email, hash string
	var passwordSet, twoFA bool
	if err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT email, password_hash, password_set, two_factor_enabled FROM core.users WHERE id = $1 AND deleted_at IS NULL
	`, claims.UserID).Scan(&email, &hash, &passwordSet, &twoFA); err != nil {
		writeError(w, http.StatusNotFound, "Учётная запись не найдена")
		return
	}
	if !strings.EqualFold(strings.TrimSpace(body.ConfirmEmail), email) {
		writeError(w, http.StatusBadRequest, "Введите email учётной записи, чтобы подтвердить удаление")
		return
	}
	if !h.checkAccountPassword(w, r, claims.UserID, passwordSet, hash, body.Password) {
		return
	}
	if twoFA && !h.checkSecondFactor(ctx, r, claims.UserID, body.Code) {
		writeError(w, http.StatusUnauthorized, "Введите верный код 2FA или резервный код")
		return
	}
	if msg := h.userDeletionBlocker(ctx, claims.UserID, false); msg != "" {
		writeError(w, http.StatusConflict, strings.Replace(msg, "У пользователя есть", "У вас есть", 1))
		return
	}
	if err := h.anonymizeUser(ctx, claims.UserID, claims.UserID, "self"); err != nil {
		writeError(w, http.StatusInternalServerError, "Не удалось удалить учётную запись")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

var userExportSections = []struct {
	key   string
	query string
}{
	{"account", `SELECT to_jsonb(u) - 'password_hash' FROM core.users u WHERE u.id = $1`},
	{"profile", `SELECT to_jsonb(p) FROM core.user_profiles p WHERE p.user_id = $1`},
	{"billing_profile", `SELECT to_jsonb(b) FROM core.user_billing_profiles b WHERE b.user_id = $1`},
	{"identification", `SELECT COALESCE(jsonb_agg(to_jsonb(e) ORDER BY e.created_at), '[]'::jsonb)
		FROM core.identification_events e WHERE e.user_id = $1`},
	{"consents", `SELECT COALESCE(jsonb_agg(to_jsonb(c) ORDER BY c.created_at), '[]'::jsonb)
		FROM core.legal_consents c WHERE c.user_id = $1`},
	{"sessions", `SELECT COALESCE(jsonb_agg(to_jsonb(s) ORDER BY s.created_at), '[]'::jsonb)
		FROM core.user_sessions s WHERE s.user_id = $1`},
	{"login_attempts", `SELECT COALESCE(jsonb_agg(to_jsonb(a) ORDER BY a.created_at DESC), '[]'::jsonb)
		FROM (SELECT * FROM core.login_attempts WHERE user_id = $1 ORDER BY created_at DESC LIMIT 500) a`},
	{"social_accounts", `SELECT COALESCE(jsonb_agg(to_jsonb(s) - 'access_token' - 'refresh_token' - 'token'), '[]'::jsonb)
		FROM core.user_social_accounts s WHERE s.user_id = $1`},
	{"notification_channels", `SELECT to_jsonb(c) FROM core.user_notification_channels c WHERE c.user_id = $1`},
	{"wallets", `SELECT COALESCE(jsonb_agg(to_jsonb(w)), '[]'::jsonb) FROM core.wallets w WHERE w.user_id = $1`},
	{"payments", `SELECT COALESCE(jsonb_agg(to_jsonb(p) ORDER BY p.created_at), '[]'::jsonb)
		FROM core.payments p WHERE p.user_id = $1`},
	{"transactions", `SELECT COALESCE(jsonb_agg(to_jsonb(t) ORDER BY t.created_at), '[]'::jsonb)
		FROM core.transactions t JOIN core.wallets w ON w.id = t.wallet_id WHERE w.user_id = $1`},
	{"acts", `SELECT COALESCE(jsonb_agg(to_jsonb(a) ORDER BY a.period), '[]'::jsonb)
		FROM core.billing_acts a WHERE a.user_id = $1`},
	{"refund_requests", `SELECT COALESCE(jsonb_agg(to_jsonb(q) ORDER BY q.created_at), '[]'::jsonb)
		FROM core.balance_refund_requests q WHERE q.user_id = $1`},
	{"servers", `SELECT COALESCE(jsonb_agg(jsonb_build_object('id', s.id, 'name', s.name, 'status', s.status,
		'created_at', s.created_at) ORDER BY s.created_at), '[]'::jsonb) FROM core.servers s WHERE s.user_id = $1`},
	{"hosting_accounts", `SELECT COALESCE(jsonb_agg(jsonb_build_object('id', a.id, 'username', a.username,
		'primary_domain', a.primary_domain, 'status', a.status, 'expires_at', a.expires_at)), '[]'::jsonb)
		FROM core.hosting_accounts a WHERE a.user_id = $1`},
	{"support_tickets", `SELECT COALESCE(jsonb_agg(to_jsonb(t) ORDER BY t.created_at), '[]'::jsonb)
		FROM core.support_tickets t WHERE t.user_id = $1`},
	{"support_messages", `SELECT COALESCE(jsonb_agg(to_jsonb(m) ORDER BY m.created_at), '[]'::jsonb)
		FROM core.support_messages m JOIN core.support_tickets t ON t.id = m.ticket_id WHERE t.user_id = $1`},
	{"notifications", `SELECT COALESCE(jsonb_agg(to_jsonb(n) ORDER BY n.created_at DESC), '[]'::jsonb)
		FROM (SELECT * FROM core.notifications WHERE user_id = $1 ORDER BY created_at DESC LIMIT 500) n`},
	{"api_keys", `SELECT COALESCE(jsonb_agg(to_jsonb(k) - 'key_hash' - 'hash' - 'secret'), '[]'::jsonb)
		FROM core.api_keys k WHERE k.user_id = $1`},
}

func (h *Handler) AccountExport(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	h.writeUserExport(w, r, claims.UserID, claims.UserID)
}

func (h *Handler) AdminUserExport(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	h.writeUserExport(w, r, chi.URLParam(r, "id"), claims.UserID)
}

func (h *Handler) writeUserExport(w http.ResponseWriter, r *http.Request, userID, actorID string) {
	ctx := r.Context()
	db := h.dbOf(ctx)
	data := map[string]json.RawMessage{}
	for _, section := range userExportSections {
		var raw []byte
		err := db.QueryRow(ctx, section.query, userID).Scan(&raw)
		switch {
		case err == nil && raw != nil:
			data[section.key] = raw
		case err != nil && !errors.Is(err, pgx.ErrNoRows):
			log.Printf("выгрузка данных %s, раздел %s: %v", userID, section.key, err)
			data[section.key] = json.RawMessage("null")
		default:
			data[section.key] = json.RawMessage("null")
		}
	}
	if string(data["account"]) == "null" {
		writeError(w, http.StatusNotFound, "Пользователь не найден")
		return
	}
	profile := h.accountingProfile(ctx)
	payload := map[string]any{
		"generated_at": time.Now().UTC().Format(time.RFC3339),
		"operator": map[string]string{
			"name":  firstNonEmpty(profile.FullName, profile.Name, profile.AppName),
			"inn":   profile.INN,
			"email": profile.Email,
		},
		"subject_id": userID,
		"data":       data,
	}
	audit(ctx, db, actorID, "user.export", "user:"+userID, nil)
	short := userID
	if len(short) > 8 {
		short = short[:8]
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="personal-data-`+short+`.json"`)
	w.Header().Set("Cache-Control", "no-store")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(payload)
}
