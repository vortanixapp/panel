package handlers

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	netmail "net/mail"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pquerna/otp/totp"
	"github.com/vortanixapp/panel/internal/api/mail"
	"github.com/vortanixapp/panel/internal/api/paneljwt"
	"github.com/vortanixapp/panel/pkg/i18n"
	"github.com/vortanixapp/panel/pkg/notify"
	"rsc.io/qr"
)

var (
	accountPhonePattern   = regexp.MustCompile(`^\+?[0-9][0-9 ()\-]{4,23}$`)
	accountPrefToken      = regexp.MustCompile(`^[a-z0-9-]{0,24}$`)
	accountStartPages     = map[string]bool{"": true, "/dashboard": true, "/servers": true, "/billing": true, "/notifications": true, "/support": true}
	accountThemes         = map[string]bool{"": true, "light": true, "dark": true, "system": true}
	accountContactColumns = map[string]string{"telegram": "telegram_id", "discord": "discord_id", "vk": "vk_id"}
)

const (
	accountNameMax    = 64
	accountContactMax = 64
	pendingEmailTTL   = 24 * time.Hour
)

type accountFieldError string

func (e accountFieldError) Error() string { return string(e) }

func (h *Handler) adminMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := tenantClaims(r.Context())
		if !ok || !isStaffRole(claims.Role) {
			writeError(w, http.StatusForbidden, "forbidden")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h *Handler) accountView(ctx context.Context, r *http.Request, claims *paneljwt.Claims) (map[string]any, error) {
	db := h.dbOf(ctx)
	var (
		email, role                      string
		twoFA, passwordSet               bool
		verifiedAt, lastLogin            *time.Time
		createdAt                        time.Time
		pendingEmail                     *string
		displayName, firstName, lastName *string
		phone, avatarURL                 *string
		locale, timezone                 string
		prefsRaw, contactsRaw            []byte
		avatarVersion, recoveryLeft      int
	)
	err := db.QueryRow(ctx, `
		SELECT u.email, u.role, u.two_factor_enabled, u.password_set, u.email_verified_at, u.created_at,
		       u.last_login_at, u.pending_email,
		       p.display_name, p.first_name, p.last_name, p.phone, p.avatar_url,
		       COALESCE(p.locale, ''), COALESCE(p.timezone, ''),
		       COALESCE(p.preferences, '{}'::jsonb), COALESCE(p.contacts, '{}'::jsonb),
		       COALESCE(p.avatar_version, 0),
		       COALESCE((SELECT jsonb_array_length(s.recovery_codes) FROM core.two_factor_secrets s
		                 WHERE s.user_id = u.id AND jsonb_typeof(s.recovery_codes) = 'array'), 0)
		FROM core.users u
		LEFT JOIN core.user_profiles p ON p.user_id = u.id
		WHERE u.id = $1
	`, claims.UserID).Scan(&email, &role, &twoFA, &passwordSet, &verifiedAt, &createdAt,
		&lastLogin, &pendingEmail,
		&displayName, &firstName, &lastName, &phone, &avatarURL,
		&locale, &timezone, &prefsRaw, &contactsRaw, &avatarVersion, &recoveryLeft)
	if err != nil {
		return nil, err
	}

	linked := []string{}
	rows, err := db.Query(ctx, `SELECT provider FROM core.user_social_accounts WHERE user_id = $1 ORDER BY provider`, claims.UserID)
	if err == nil {
		for rows.Next() {
			var p string
			if rows.Scan(&p) == nil && p != "" {
				linked = append(linked, publicProviderKey(p))
			}
		}
		rows.Close()
	}

	prefs := map[string]any{}
	_ = json.Unmarshal(prefsRaw, &prefs)
	contacts := parseContacts(contactsRaw)
	staff := isStaffRole(role)

	return map[string]any{
		"id":                  claims.UserID,
		"email":               email,
		"role":                role,
		"staff":               staff,
		"two_factor_enabled":  twoFA,
		"two_factor_required": staff && h.staffRequires2FA(ctx),
		"recovery_codes_left": recoveryLeft,
		"has_password":        passwordSet,
		"email_verified":      verifiedAt != nil,
		"email_verified_at":   verifiedAt,
		"pending_email":       pendingEmail,
		"created_at":          createdAt,
		"last_login_at":       lastLogin,
		"display_name":        displayName,
		"first_name":          firstName,
		"last_name":           lastName,
		"phone":               phone,
		"locale":              locale,
		"timezone":            timezone,
		"avatar_url":          versionedAvatarURL(h.avatarPublicURL(r, avatarURL), avatarVersion),
		"linked_providers":    linked,
		"contacts": map[string]string{
			"telegram": contactString(contacts, "telegram_id"),
			"discord":  contactString(contacts, "discord_id"),
			"vk":       contactString(contacts, "vk_id"),
		},
		"preferences": prefs,
	}, nil
}

func versionedAvatarURL(u *string, version int) *string {
	if u == nil || version <= 0 || strings.Contains(*u, "?") {
		return u
	}
	out := *u + "?v=" + strconv.Itoa(version)
	return &out
}

func (h *Handler) writeAccount(w http.ResponseWriter, r *http.Request, claims *paneljwt.Claims) {
	view, err := h.accountView(r.Context(), r, claims)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Не удалось загрузить аккаунт")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": view})
}

func (h *Handler) GetAccount(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	h.writeAccount(w, r, claims)
}

func accountText(raw json.RawMessage, max int, label string) (*string, error) {
	var v *string
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, accountFieldError(label + ": нужна строка")
	}
	if v == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*v)
	if trimmed == "" {
		return nil, nil
	}
	if utf8.RuneCountInString(trimmed) > max || strings.ContainsAny(trimmed, "\x00\r\n\t") {
		return nil, accountFieldError(label + ": не длиннее " + strconv.Itoa(max) + " символов, без переносов строк")
	}
	return &trimmed, nil
}

func normalizePreferences(raw json.RawMessage) (map[string]any, error) {
	var in map[string]any
	if err := json.Unmarshal(raw, &in); err != nil || in == nil {
		return nil, accountFieldError("Настройки внешнего вида переданы неверно")
	}
	out := map[string]any{}
	for key, value := range in {
		switch key {
		case "sidebar_collapsed":
			b, ok := value.(bool)
			if !ok {
				return nil, accountFieldError("Настройки внешнего вида переданы неверно")
			}
			out[key] = b
		case "theme", "font", "layout", "collapsible", "start_page":
			s, ok := value.(string)
			if !ok {
				return nil, accountFieldError("Настройки внешнего вида переданы неверно")
			}
			switch {
			case key == "theme" && !accountThemes[s],
				key == "start_page" && !accountStartPages[s],
				key != "theme" && key != "start_page" && !accountPrefToken.MatchString(s):
				return nil, accountFieldError("Недопустимое значение настройки " + key)
			}
			out[key] = s
		}
	}
	return out, nil
}

func (h *Handler) validLocale(ctx context.Context, code string) bool {
	if code == "ru" || code == "en" {
		return true
	}
	var ok bool
	_ = h.dbOf(ctx).QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM core.languages WHERE code = $1 AND enabled)`, code).Scan(&ok)
	return ok
}

func (h *Handler) UpdateAccount(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var body map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	ctx := r.Context()
	sets := []string{}
	args := []any{claims.UserID}
	add := func(expr string, value any) {
		args = append(args, value)
		sets = append(sets, strings.ReplaceAll(expr, "?", "$"+strconv.Itoa(len(args))))
	}
	fail := func(err error) {
		var fe accountFieldError
		if errors.As(err, &fe) {
			writeError(w, http.StatusUnprocessableEntity, fe.Error())
			return
		}
		writeError(w, http.StatusBadRequest, "invalid request")
	}

	for key, label := range map[string]string{"display_name": "Отображаемое имя", "first_name": "Имя", "last_name": "Фамилия"} {
		raw, present := body[key]
		if !present {
			continue
		}
		v, err := accountText(raw, accountNameMax, label)
		if err != nil {
			fail(err)
			return
		}
		add(key+" = ?", v)
	}
	if raw, present := body["phone"]; present {
		v, err := accountText(raw, 24, "Телефон")
		if err != nil {
			fail(err)
			return
		}
		if v != nil && !accountPhonePattern.MatchString(*v) {
			fail(accountFieldError("Телефон: цифры, пробелы, скобки и дефисы, в начале можно +"))
			return
		}
		add("phone = ?", v)
	}
	if raw, present := body["locale"]; present {
		v, err := accountText(raw, 16, "Язык")
		if err != nil || v == nil || !h.validLocale(ctx, *v) {
			fail(accountFieldError("Такого языка нет в панели"))
			return
		}
		add("locale = ?", *v)
	}
	if raw, present := body["timezone"]; present {
		v, err := accountText(raw, 64, "Часовой пояс")
		if err != nil {
			fail(err)
			return
		}
		zone := ""
		if v != nil {
			if _, err := time.LoadLocation(*v); err != nil || *v == "Local" {
				fail(accountFieldError("Неизвестный часовой пояс"))
				return
			}
			zone = *v
		}
		add("timezone = ?", zone)
	}
	if raw, present := body["contacts"]; present {
		var in map[string]json.RawMessage
		if err := json.Unmarshal(raw, &in); err != nil || in == nil {
			fail(accountFieldError("Контакты переданы неверно"))
			return
		}
		patch := map[string]string{}
		for key, column := range accountContactColumns {
			value, present := in[key]
			if !present {
				continue
			}
			v, err := accountText(value, accountContactMax, "Контакт")
			if err != nil {
				fail(err)
				return
			}
			patch[column] = ""
			if v != nil {
				patch[column] = *v
			}
		}
		patchJSON, _ := json.Marshal(patch)
		add("contacts = COALESCE(contacts, '{}'::jsonb) || ?::jsonb", string(patchJSON))
	}
	if raw, present := body["preferences"]; present {
		prefs, err := normalizePreferences(raw)
		if err != nil {
			fail(err)
			return
		}
		prefsJSON, _ := json.Marshal(prefs)
		add("preferences = COALESCE(preferences, '{}'::jsonb) || ?::jsonb", string(prefsJSON))
	}

	if len(sets) > 0 {
		db := h.dbOf(ctx)
		if _, err := db.Exec(ctx, `INSERT INTO core.user_profiles (user_id) VALUES ($1) ON CONFLICT (user_id) DO NOTHING`, claims.UserID); err != nil {
			writeError(w, http.StatusInternalServerError, "Не удалось сохранить профиль")
			return
		}
		query := "UPDATE core.user_profiles SET " + strings.Join(sets, ", ") + ", updated_at = now() WHERE user_id = $1"
		if _, err := db.Exec(ctx, query, args...); err != nil {
			writeError(w, http.StatusInternalServerError, "Не удалось сохранить профиль")
			return
		}
	}
	h.writeAccount(w, r, claims)
}

type destroySessionRequest struct {
	SessionID string `json:"session_id"`
	AllOthers bool   `json:"all_others"`
	All       bool   `json:"all"`
}

func (h *Handler) ListAccountSessions(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	rows, err := h.dbOf(r.Context()).Query(r.Context(), `
		SELECT id::text, COALESCE(ip_address, ''), COALESCE(user_agent, ''), last_active, created_at
		FROM core.user_sessions
		WHERE user_id = $1
		ORDER BY (id::text = $2) DESC, last_active DESC
		LIMIT 50
	`, claims.UserID, claims.SessionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Не удалось загрузить сессии")
		return
	}
	defer rows.Close()
	sessions := []map[string]any{}
	for rows.Next() {
		var id, ip, ua string
		var lastActive, created time.Time
		if rows.Scan(&id, &ip, &ua, &lastActive, &created) != nil {
			continue
		}
		sessions = append(sessions, map[string]any{
			"id":            id,
			"ip_address":    ip,
			"user_agent":    ua,
			"device":        deviceFromUserAgent(ua),
			"last_activity": lastActive,
			"created_at":    created,
			"is_current":    id == claims.SessionID,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"sessions":           sessions,
		"current_session_id": claims.SessionID,
	})
}

func (h *Handler) DestroySession(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req destroySessionRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	ctx := r.Context()
	db := h.dbOf(ctx)
	var tag pgconn.CommandTag
	var err error
	switch {
	case req.All:
		tag, err = db.Exec(ctx, `DELETE FROM core.user_sessions WHERE user_id = $1`, claims.UserID)
	case req.AllOthers:
		tag, err = db.Exec(ctx, `DELETE FROM core.user_sessions WHERE user_id = $1 AND id::text <> $2`, claims.UserID, claims.SessionID)
	case req.SessionID != "" && req.SessionID != claims.SessionID:
		tag, err = db.Exec(ctx, `DELETE FROM core.user_sessions WHERE id::text = $1 AND user_id = $2`, req.SessionID, claims.UserID)
	default:
		writeError(w, http.StatusBadRequest, "Укажите сессию")
		return
	}
	closed := tag.RowsAffected()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Не удалось завершить сессии")
		return
	}
	if closed > 0 {
		audit(ctx, db, claims.UserID, "user.sessions_close", "user:"+claims.UserID, map[string]any{"closed": closed, "all": req.All})
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "closed": closed})
}

type changeEmailRequest struct {
	Email           string `json:"email"`
	CurrentPassword string `json:"current_password"`
}

func normalizeEmailInput(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	addr, err := netmail.ParseAddress(raw)
	if err != nil || addr.Name != "" || !strings.Contains(addr.Address, ".") || len(addr.Address) > 254 {
		return "", false
	}
	return strings.ToLower(addr.Address), true
}

func (h *Handler) ChangeEmail(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req changeEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	newEmail, valid := normalizeEmailInput(req.Email)
	if !valid {
		writeError(w, http.StatusUnprocessableEntity, "Введите корректный email")
		return
	}
	ctx := r.Context()
	db := h.dbOf(ctx)
	var current, hash string
	var passwordSet bool
	if err := db.QueryRow(ctx, `
		SELECT email, password_hash, password_set FROM core.users WHERE id = $1 AND status = 'active'
	`, claims.UserID).Scan(&current, &hash, &passwordSet); err != nil {
		writeError(w, http.StatusNotFound, "Учётная запись не найдена")
		return
	}
	if strings.EqualFold(current, newEmail) {
		writeError(w, http.StatusConflict, "Это ваш текущий адрес")
		return
	}
	if !h.checkAccountPassword(w, r, claims.UserID, passwordSet, hash, req.CurrentPassword) {
		return
	}
	var taken bool
	if err := db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM core.users WHERE lower(email) = $1 AND id <> $2)`, newEmail, claims.UserID).Scan(&taken); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	if taken {
		writeError(w, http.StatusConflict, "Этот email уже занят")
		return
	}

	token := randomToken(32)
	sum := sha256.Sum256([]byte(token))
	if _, err := db.Exec(ctx, `
		UPDATE core.users
		SET pending_email = $2, pending_email_hash = $3, pending_email_expires = now() + $4::interval
		WHERE id = $1
	`, claims.UserID, newEmail, hex.EncodeToString(sum[:]), pendingEmailTTL.String()); err != nil {
		writeError(w, http.StatusInternalServerError, "Не удалось сохранить новый адрес")
		return
	}

	confirmURL := strings.TrimRight(h.frontendURL, "/") + "/verify-email?change=" + token
	resp := map[string]any{"status": "pending", "pending_email": newEmail}
	if h.mail.Enabled() {
		subject, text := mail.EmailChangeEmail(i18n.ForUser(ctx, db, claims.UserID), h.mailBrand(ctx, r), newEmail, confirmURL)
		if err := h.mail.Send(newEmail, subject, text); err != nil {
			writeError(w, http.StatusBadGateway, "Не удалось отправить письмо на новый адрес")
			return
		}
	} else if h.mail.DevExpose {
		resp["confirm_url"] = confirmURL
	}
	audit(ctx, db, claims.UserID, "user.email_change_request", "user:"+claims.UserID, map[string]any{"email": newEmail})
	h.notifyUser(ctx, claims.UserID, notify.Event{
		Kind:   notify.KindEmailChange,
		Title:  i18n.Key("notify.email_change_requested.title"),
		Body:   i18n.Key("notify.email_change_requested.body", i18n.Params{"email": newEmail}),
		Action: h.panelAction("notify.action.security", "/settings?tab=contacts"),
	})
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) CancelEmailChange(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ctx := r.Context()
	_, _ = h.dbOf(ctx).Exec(ctx, `
		UPDATE core.users SET pending_email = NULL, pending_email_hash = NULL, pending_email_expires = NULL WHERE id = $1
	`, claims.UserID)
	h.writeAccount(w, r, claims)
}

func (h *Handler) ConfirmEmailChange(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Token) == "" {
		writeError(w, http.StatusBadRequest, "Ссылка подтверждения неполная")
		return
	}
	if h.tooManyAttempts(w, r, "email-change", 20, 15*time.Minute) {
		return
	}
	ctx := r.Context()
	db := h.dbOf(ctx)
	sum := sha256.Sum256([]byte(strings.TrimSpace(body.Token)))
	var userID, email string
	err := db.QueryRow(ctx, `
		UPDATE core.users u
		SET email = u.pending_email, email_verified_at = now(),
		    pending_email = NULL, pending_email_hash = NULL, pending_email_expires = NULL
		WHERE u.pending_email_hash = $1 AND u.pending_email_expires > now() AND u.status = 'active'
		  AND NOT EXISTS (SELECT 1 FROM core.users o WHERE lower(o.email) = lower(u.pending_email) AND o.id <> u.id)
		RETURNING u.id::text, u.email
	`, hex.EncodeToString(sum[:])).Scan(&userID, &email)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusGone, "Ссылка устарела, уже использована или адрес занят. Запросите смену почты заново")
		return
	}
	if err != nil {
		writeError(w, http.StatusConflict, "Не удалось подтвердить адрес")
		return
	}
	audit(ctx, db, userID, "user.email_change", "user:"+userID, map[string]any{"email": email})
	h.notifyUser(ctx, userID, notify.Event{
		Kind:  notify.KindEmailChange,
		Title: i18n.Key("notify.email_changed.title"),
		Body:  i18n.Key("notify.email_changed.body", i18n.Params{"email": email}),
	})
	writeJSON(w, http.StatusOK, map[string]any{"status": "changed", "email": email})
}

var allowedAvatarTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

type avatarURLRequest struct {
	AvatarURL string `json:"avatar_url"`
}

func (h *Handler) removeAvatarFiles(userID, keep string) {
	for _, ext := range []string{".jpg", ".png", ".webp"} {
		if ext == keep {
			continue
		}
		_ = os.Remove(filepath.Join(h.uploadDir, "avatars", userID+ext))
	}
}

func (h *Handler) saveAvatarURL(ctx context.Context, userID string, avatarURL *string) {
	_, _ = h.dbOf(ctx).Exec(ctx, `
		INSERT INTO core.user_profiles (user_id, avatar_url, avatar_version, updated_at)
		VALUES ($1, $2, 1, now())
		ON CONFLICT (user_id) DO UPDATE SET avatar_url = EXCLUDED.avatar_url,
			avatar_version = core.user_profiles.avatar_version + 1, updated_at = now()
	`, userID, avatarURL)
}

func (h *Handler) UploadAvatar(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ctx := r.Context()

	if strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		var req avatarURLRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid json")
			return
		}
		avatarURL := strings.TrimSpace(req.AvatarURL)
		if !strings.HasPrefix(avatarURL, "https://") && !strings.HasPrefix(avatarURL, "http://") {
			writeError(w, http.StatusUnprocessableEntity, "Укажите ссылку на изображение")
			return
		}
		h.removeAvatarFiles(claims.UserID, "")
		h.saveAvatarURL(ctx, claims.UserID, &avatarURL)
		h.writeAccount(w, r, claims)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, (4<<20)+(1<<20))
	if err := r.ParseMultipartForm(4 << 20); err != nil {
		writeError(w, http.StatusRequestEntityTooLarge, "Файл больше 4 МБ или повреждён")
		return
	}
	file, header, err := r.FormFile("avatar")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Выберите изображение")
		return
	}
	defer file.Close()

	ext, allowed := allowedAvatarTypes[header.Header.Get("Content-Type")]
	if !allowed {
		lower := strings.ToLower(header.Filename)
		switch {
		case strings.HasSuffix(lower, ".jpg"), strings.HasSuffix(lower, ".jpeg"):
			ext = ".jpg"
		case strings.HasSuffix(lower, ".png"):
			ext = ".png"
		case strings.HasSuffix(lower, ".webp"):
			ext = ".webp"
		default:
			writeError(w, http.StatusUnprocessableEntity, "Подходят только JPG, PNG и WebP")
			return
		}
	}
	head := make([]byte, 512)
	n, _ := io.ReadFull(file, head)
	switch http.DetectContentType(head[:n]) {
	case "image/jpeg", "image/png", "image/webp":
	default:
		writeError(w, http.StatusUnprocessableEntity, "Файл не похож на изображение JPG, PNG или WebP")
		return
	}

	avatarDir := filepath.Join(h.uploadDir, "avatars")
	if err := os.MkdirAll(avatarDir, 0o755); err != nil {
		writeError(w, http.StatusInternalServerError, "Хранилище недоступно")
		return
	}
	filename := claims.UserID + ext
	out, err := os.Create(filepath.Join(avatarDir, filename))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Не удалось сохранить аватар")
		return
	}
	_, err = io.Copy(out, io.MultiReader(bytes.NewReader(head[:n]), file))
	out.Close()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Не удалось сохранить аватар")
		return
	}
	h.removeAvatarFiles(claims.UserID, ext)
	avatarURL := h.publicBaseURL(r) + "/v1/uploads/avatars/" + filename
	h.saveAvatarURL(ctx, claims.UserID, &avatarURL)
	h.writeAccount(w, r, claims)
}

func (h *Handler) DeleteAvatar(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	h.removeAvatarFiles(claims.UserID, "")
	h.saveAvatarURL(r.Context(), claims.UserID, nil)
	h.writeAccount(w, r, claims)
}

func (h *Handler) ServeAvatar(w http.ResponseWriter, r *http.Request) {
	filename := chi.URLParam(r, "filename")
	if filename == "" || strings.Contains(filename, "..") || strings.Contains(filename, "/") {
		writeError(w, http.StatusBadRequest, "invalid filename")
		return
	}
	path := filepath.Join(h.uploadDir, "avatars", filename)
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	contentType := "application/octet-stream"
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".jpg", ".jpeg":
		contentType = "image/jpeg"
	case ".png":
		contentType = "image/png"
	case ".webp":
		contentType = "image/webp"
	}
	setUploadHeaders(w, contentType)
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	if r.URL.Query().Get("v") == "" {
		w.Header().Set("Cache-Control", "public, max-age=60")
	}
	http.ServeFile(w, r, path)
}

func (h *Handler) panelName(ctx context.Context) string {
	return firstNonEmpty(h.tenantSettingString(ctx, "app.name"), h.tenantSettingString(ctx, "panel.name"), "Vortanix")
}

func qrDataURL(text string) string {
	code, err := qr.Encode(text, qr.M)
	if err != nil {
		return ""
	}
	code.Scale = 6
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(code.PNG())
}

func (h *Handler) Generate2FA(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ctx := r.Context()
	var enabled bool
	var email string
	if err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT COALESCE(two_factor_enabled, false), email FROM core.users WHERE id = $1
	`, claims.UserID).Scan(&enabled, &email); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	if enabled {
		writeError(w, http.StatusConflict, "Двухфакторная защита уже включена")
		return
	}
	key, err := totp.Generate(totp.GenerateOpts{Issuer: h.panelName(ctx), AccountName: email})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Не удалось создать ключ 2FA")
		return
	}
	sealed, err := h.secrets.Encrypt(key.Secret())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Не удалось сохранить ключ 2FA")
		return
	}
	_, _ = h.dbOf(ctx).Exec(ctx, `
		INSERT INTO core.two_factor_secrets (user_id, secret, recovery_codes, enabled_at)
		VALUES ($1, $2, '[]'::jsonb, NULL)
		ON CONFLICT (user_id) DO UPDATE SET secret = EXCLUDED.secret, recovery_codes = '[]'::jsonb, enabled_at = NULL
	`, claims.UserID, sealed)
	writeJSON(w, http.StatusOK, map[string]string{
		"secret": key.Secret(),
		"uri":    key.URL(),
		"qr":     qrDataURL(key.URL()),
	})
}

func (h *Handler) Enable2FA(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ctx := r.Context()
	var body struct {
		Code string `json:"code"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	code := strings.TrimSpace(body.Code)
	if code == "" {
		writeError(w, http.StatusBadRequest, "Введите код из приложения-аутентификатора")
		return
	}
	if h.tooManyAttempts(w, r, "2fa-enable", 10, 15*time.Minute, claims.UserID) {
		return
	}
	db := h.dbOf(ctx)
	var sealed string
	if err := db.QueryRow(ctx, `SELECT secret FROM core.two_factor_secrets WHERE user_id = $1`, claims.UserID).Scan(&sealed); err != nil || sealed == "" {
		writeError(w, http.StatusConflict, "Сначала создайте ключ второго фактора")
		return
	}
	if !totp.Validate(code, h.secrets.MustDecrypt(sealed)) {
		writeError(w, http.StatusUnauthorized, "Код не подошёл — проверьте время на устройстве и повторите")
		return
	}

	codes, hashes := newRecoveryCodes(recoveryCodeCount)
	hashesJSON, _ := json.Marshal(hashes)
	_, _ = db.Exec(ctx, `UPDATE core.two_factor_secrets SET recovery_codes = $2::jsonb, enabled_at = now() WHERE user_id = $1`, claims.UserID, string(hashesJSON))
	_, _ = db.Exec(ctx, `UPDATE core.users SET two_factor_enabled = true WHERE id = $1`, claims.UserID)
	audit(ctx, db, claims.UserID, "user.2fa_enable", "user:"+claims.UserID, nil)
	h.notifyUser(ctx, claims.UserID, notify.Event{
		Kind:   notify.KindTwoFactor,
		Title:  i18n.Key("notify.twofactor_enabled.title"),
		Body:   i18n.Key("notify.twofactor_enabled.body"),
		Action: h.panelAction("notify.action.security", "/settings?tab=security"),
	})
	writeJSON(w, http.StatusOK, map[string]any{"status": "enabled", "recovery_codes": codes})
}

func (h *Handler) Disable2FA(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ctx := r.Context()
	var body struct {
		Password string `json:"password"`
		Code     string `json:"code"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	db := h.dbOf(ctx)
	var hash, role string
	var passwordSet bool
	if err := db.QueryRow(ctx, `SELECT password_hash, password_set, role FROM core.users WHERE id = $1`, claims.UserID).Scan(&hash, &passwordSet, &role); err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if isStaffRole(role) && h.staffRequires2FA(ctx) {
		writeError(w, http.StatusForbidden, "Сотрудникам панели двухфакторная защита обязательна")
		return
	}
	if !h.checkAccountPassword(w, r, claims.UserID, passwordSet, hash, body.Password) {
		return
	}
	if strings.TrimSpace(body.Code) == "" {
		writeError(w, http.StatusBadRequest, "Введите код из приложения или резервный код")
		return
	}
	if !h.checkSecondFactor(ctx, r, claims.UserID, body.Code) {
		h.recordLoginAttempt(ctx, r, claims.UserID, claims.Email, "отключение 2FA: неверный код", false)
		writeError(w, http.StatusUnauthorized, "Код не подошёл")
		return
	}

	_, _ = db.Exec(ctx, `UPDATE core.users SET two_factor_enabled = false WHERE id = $1`, claims.UserID)
	_, _ = db.Exec(ctx, `DELETE FROM core.two_factor_secrets WHERE user_id = $1`, claims.UserID)
	audit(ctx, db, claims.UserID, "user.2fa_disable", "user:"+claims.UserID, nil)
	h.notifyUser(ctx, claims.UserID, notify.Event{
		Kind:   notify.KindTwoFactor,
		Title:  i18n.Key("notify.twofactor_disabled.title"),
		Body:   i18n.Key("notify.twofactor_disabled.body"),
		Action: h.panelAction("notify.action.security", "/settings?tab=security"),
	})
	writeJSON(w, http.StatusOK, map[string]string{"status": "disabled"})
}

func (h *Handler) SetLocale(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	locale := chi.URLParam(r, "locale")
	if !h.validLocale(r.Context(), locale) {
		writeError(w, http.StatusUnprocessableEntity, "Такого языка нет в панели")
		return
	}
	_, _ = h.dbOf(r.Context()).Exec(r.Context(), `
		INSERT INTO core.user_profiles (user_id, locale) VALUES ($1, $2)
		ON CONFLICT (user_id) DO UPDATE SET locale = EXCLUDED.locale, updated_at = now()
	`, claims.UserID, locale)
	writeJSON(w, http.StatusOK, map[string]string{"locale": locale})
}
