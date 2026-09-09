package handlers

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/vortanixapp/panel/internal/api/mail"
	"github.com/vortanixapp/panel/pkg/oauth"
	"golang.org/x/crypto/bcrypt"
)

const (
	socialIntentLogin = "login"
	socialIntentLink  = "link"
)

type oauthState struct {
	Intent     string `json:"intent"`
	TenantSlug string `json:"tenant_slug"`
	ReturnPath string `json:"return_path"`
	UserID     string `json:"user_id"`
}

func (h *Handler) encodeOAuthState(st oauthState) string {
	raw, _ := json.Marshal(st)
	payload := base64.RawURLEncoding.EncodeToString(raw)
	mac := hmac.New(sha256.New, []byte(h.jwtSecret))
	mac.Write([]byte(payload))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return payload + "." + sig
}

func (h *Handler) decodeOAuthState(state string) (oauthState, error) {
	parts := strings.Split(state, ".")
	if len(parts) != 2 {
		return oauthState{}, fmt.Errorf("invalid state")
	}
	mac := hmac.New(sha256.New, []byte(h.jwtSecret))
	mac.Write([]byte(parts[0]))
	expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(parts[1])) {
		return oauthState{}, fmt.Errorf("invalid state signature")
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return oauthState{}, err
	}
	var st oauthState
	if err := json.Unmarshal(raw, &st); err != nil {
		return oauthState{}, err
	}
	return st, nil
}

func (h *Handler) SocialRedirect(w http.ResponseWriter, r *http.Request) {
	h.startSocialOAuth(w, r, socialIntentLogin)
}

func (h *Handler) SocialLinkRedirect(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	h.startSocialOAuth(w, r, socialIntentLink, claims.UserID)
}

func (h *Handler) startSocialOAuth(w http.ResponseWriter, r *http.Request, intent string, linkUserID ...string) {
	provider := chi.URLParam(r, "provider")
	p, ok := h.oauth.Get(provider)
	if !ok || !p.Configured {
		writeError(w, http.StatusNotFound, "provider not configured")
		return
	}
	tenantSlug := singleTenantSlug
	returnPath := "/login"
	if intent == socialIntentLink {
		returnPath = "/settings/account"
	}
	st := oauthState{
		Intent:     intent,
		TenantSlug: tenantSlug,
		ReturnPath: returnPath,
	}
	if len(linkUserID) > 0 {
		st.UserID = linkUserID[0]
	}
	state := h.encodeOAuthState(st)
	authURL, err := p.AuthCodeURL(state)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to build oauth url")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "url": authURL})
}

type socialExchangeRequest struct {
	Code  string `json:"code"`
	State string `json:"state"`
}

func (h *Handler) SocialExchange(w http.ResponseWriter, r *http.Request) {
	h.socialExchange(w, r, socialIntentLogin)
}

func (h *Handler) SocialLinkExchange(w http.ResponseWriter, r *http.Request) {
	h.socialExchange(w, r, socialIntentLink)
}

func (h *Handler) socialExchange(w http.ResponseWriter, r *http.Request, forcedIntent string) {
	provider := chi.URLParam(r, "provider")
	var req socialExchangeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Code == "" {
		writeError(w, http.StatusBadRequest, "code is required")
		return
	}
	st, err := h.decodeOAuthState(req.State)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid state")
		return
	}
	if forcedIntent != "" {
		st.Intent = forcedIntent
	}
	profile, err := h.oauth.Exchange(r.Context(), provider, req.Code)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "oauth exchange failed")
		return
	}
	if profile.ProviderUserID == "" {
		writeError(w, http.StatusUnprocessableEntity, "provider profile missing id")
		return
	}
	providerKey := normalizeProviderKey(provider)
	if st.Intent == socialIntentLink {
		h.linkSocialAccount(w, r, st, providerKey, profile)
		return
	}
	if profile.Email == "" && providerKey != "telegram" {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"ok":      false,
			"message": "Провайдер не передал email. Для входа через соцсети email обязателен.",
		})
		return
	}
	h.loginOrRegisterSocial(w, r, st.TenantSlug, providerKey, profile)
}

func (h *Handler) loginOrRegisterSocial(w http.ResponseWriter, r *http.Request, tenantSlug, providerKey string, profile oauth.Profile) {
	ctx := r.Context()
	var tenantID string
	if err := h.dbOf(ctx).QueryRow(ctx, `SELECT id::text FROM core.tenants WHERE slug = $1 AND status = 'active'`, tenantSlug).Scan(&tenantID); err != nil {
		writeError(w, http.StatusNotFound, "tenant not found")
		return
	}

	var userID, email, role string
	err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT u.id::text, u.email, u.role
		FROM core.user_social_accounts s
		JOIN core.users u ON u.id = s.user_id
		WHERE s.provider = $1 AND s.provider_user_id = $2 AND u.tenant_id = $3 AND u.status = 'active'
	`, providerKey, profile.ProviderUserID, tenantID).Scan(&userID, &email, &role)
	if err != nil {
		email = strings.ToLower(strings.TrimSpace(profile.Email))
		if email != "" {
			_ = h.dbOf(ctx).QueryRow(ctx, `
				SELECT id::text, email, role FROM core.users
				WHERE tenant_id = $1 AND email = $2 AND status = 'active'
			`, tenantID, email).Scan(&userID, &email, &role)
		}
		if userID == "" {
			userID, email, role, err = h.createSocialUser(ctx, tenantID, email, profile)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "failed to create user")
				return
			}
		}
		_, _ = h.dbOf(ctx).Exec(ctx, `
			INSERT INTO core.user_social_accounts (tenant_id, user_id, provider, provider_user_id, email, name, avatar_url)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (provider, provider_user_id) DO UPDATE SET
				user_id = EXCLUDED.user_id, email = EXCLUDED.email, name = EXCLUDED.name, avatar_url = EXCLUDED.avatar_url
		`, tenantID, userID, providerKey, profile.ProviderUserID, nullString(profile.Email), nullString(profile.Name), nullString(profile.AvatarURL))
		if email != "" {
			_, _ = h.dbOf(ctx).Exec(ctx, `UPDATE core.users SET email_verified_at = COALESCE(email_verified_at, now()) WHERE id = $1`, userID)
		}
	}

	var twoFAEnabled bool
	_ = h.dbOf(ctx).QueryRow(ctx, `
		SELECT COALESCE(two_factor_enabled, false) FROM core.users WHERE id = $1::uuid
	`, userID).Scan(&twoFAEnabled)
	if twoFAEnabled {
		challenge := randomToken(16)
		_ = h.cache.SetJSON(ctx, "2fa:"+challenge, map[string]string{
			"tenant_id": tenantID, "user_id": userID, "email": email,
			"role": role, "tenant_slug": tenantSlug,
		}, 5*time.Minute)
		h.recordLoginAttempt(ctx, r, tenantID, userID, email, "соцвход: требуется 2FA", false)
		writeJSON(w, http.StatusOK, map[string]any{
			"requires_2fa":     true,
			"two_factor_token": challenge,
		})
		return
	}

	access, refresh, err := h.issueAuthTokens(r, tenantID, tenantSlug, userID, email, role, "", rememberRefreshTTL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to issue tokens")
		return
	}
	h.recordLoginAttempt(ctx, r, tenantID, userID, email, "вход через "+providerKey, true)
	redirect := "/dashboard"
	if isStaffRole(role) {
		redirect = "/admin/dashboard"
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":            true,
		"access_token":  access,
		"refresh_token": refresh,
		"redirect":      redirect,
		"user":          map[string]string{"id": userID, "email": email, "role": role},
	})
}

func (h *Handler) createSocialUser(ctx context.Context, tenantID, email string, profile oauth.Profile) (userID, outEmail, role string, err error) {
	if email == "" {
		email = fmt.Sprintf("tg_%s@telegram.local", profile.ProviderUserID)
	}
	hashBytes, _ := bcrypt.GenerateFromPassword([]byte(randomToken(32)), bcrypt.DefaultCost)
	role = "user"
	err = h.dbOf(ctx).QueryRow(ctx, `
		INSERT INTO core.users (tenant_id, email, password_hash, role, email_verified_at)
		VALUES ($1, $2, $3, $4, CASE WHEN $5 = '' THEN NULL ELSE now() END)
		RETURNING id::text
	`, tenantID, strings.ToLower(email), string(hashBytes), role, profile.Email).Scan(&userID)
	outEmail = email
	return
}

func (h *Handler) linkSocialAccount(w http.ResponseWriter, r *http.Request, st oauthState, providerKey string, profile oauth.Profile) {
	claims, ok := tenantClaims(r.Context())
	userID := st.UserID
	tenantID := ""
	if ok {
		userID = claims.UserID
		tenantID = claims.TenantID
	}
	if userID == "" || tenantID == "" {
		writeError(w, http.StatusUnauthorized, "link session expired")
		return
	}
	ctx := r.Context()
	var ownerID string
	if err := h.dbOf(ctx).QueryRow(ctx, `SELECT user_id::text FROM core.user_social_accounts WHERE provider = $1 AND provider_user_id = $2`, providerKey, profile.ProviderUserID).Scan(&ownerID); err == nil && ownerID != userID {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"ok": false, "message": "Этот аккаунт уже привязан к другому пользователю."})
		return
	}
	_, err := h.dbOf(ctx).Exec(ctx, `
		INSERT INTO core.user_social_accounts (tenant_id, user_id, provider, provider_user_id, email, name, avatar_url)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (tenant_id, user_id, provider) DO UPDATE SET
			provider_user_id = EXCLUDED.provider_user_id,
			email = EXCLUDED.email, name = EXCLUDED.name, avatar_url = EXCLUDED.avatar_url
	`, tenantID, userID, providerKey, profile.ProviderUserID, nullString(profile.Email), nullString(profile.Name), nullString(profile.AvatarURL))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "link failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":       true,
		"status":   "social-linked",
		"provider": providerKey,
		"redirect": st.ReturnPath,
	})
}

func (h *Handler) SocialUnlink(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	providerKey := normalizeProviderKey(chi.URLParam(r, "provider"))
	if providerKey == "" {
		writeError(w, http.StatusNotFound, "unknown provider")
		return
	}
	_, _ = h.dbOf(r.Context()).Exec(r.Context(), `
		DELETE FROM core.user_social_accounts WHERE tenant_id = $1 AND user_id = $2 AND provider = $3
	`, claims.TenantID, claims.UserID, providerKey)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "message": "Соцсеть отвязана."})
}

func (h *Handler) TelegramCallback(w http.ResponseWriter, r *http.Request) {
	h.handleTelegram(w, r, socialIntentLogin)
}

func (h *Handler) TelegramLink(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	st := oauthState{Intent: socialIntentLink, UserID: claims.UserID, TenantSlug: claims.TenantSlug, ReturnPath: "/settings/account"}
	h.handleTelegramWithState(w, r, st)
}

func (h *Handler) handleTelegram(w http.ResponseWriter, r *http.Request, intent string) {
	tenantSlug := singleTenantSlug
	st := oauthState{Intent: intent, TenantSlug: tenantSlug, ReturnPath: "/login"}
	h.handleTelegramWithState(w, r, st)
}

func (h *Handler) handleTelegramWithState(w http.ResponseWriter, r *http.Request, st oauthState) {
	data := extractTelegramData(r)
	if !verifyTelegramHash(data, h.telegramBotToken) {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"ok": false, "error": "Не удалось подтвердить данные от Telegram."})
		return
	}
	telegramID := strings.TrimSpace(data["id"])
	if telegramID == "" {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"ok": false, "error": "Не удалось получить Telegram ID."})
		return
	}
	name := strings.TrimSpace(data["first_name"] + " " + data["last_name"])
	username := strings.TrimSpace(data["username"])
	if name == "" && username != "" {
		name = username
	}
	profile := oauth.Profile{
		ProviderUserID: telegramID,
		Name:           name,
		AvatarURL:      strings.TrimSpace(data["photo_url"]),
	}
	if st.Intent == socialIntentLink {
		h.linkSocialAccount(w, r, st, "telegram", profile)
		return
	}
	h.loginOrRegisterSocial(w, r, st.TenantSlug, "telegram", profile)
}

func extractTelegramData(r *http.Request) map[string]string {
	keys := []string{"id", "first_name", "last_name", "username", "photo_url", "auth_date", "hash"}
	out := make(map[string]string, len(keys))
	for _, k := range keys {
		out[k] = r.URL.Query().Get(k)
	}
	return out
}

func verifyTelegramHash(data map[string]string, botToken string) bool {
	hash := data["hash"]
	if hash == "" || botToken == "" {
		return false
	}
	authDate, _ := strconv.ParseInt(data["auth_date"], 10, 64)
	if authDate == 0 || time.Now().Unix()-authDate > 86400 {
		return false
	}
	var pairs []string
	for k, v := range data {
		if k == "hash" || v == "" {
			continue
		}
		pairs = append(pairs, k+"="+v)
	}
	sort.Strings(pairs)
	checkString := strings.Join(pairs, "\n")
	secretKey := sha256.Sum256([]byte(botToken))
	mac := hmac.New(sha256.New, secretKey[:])
	mac.Write([]byte(checkString))
	computed := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(computed), []byte(hash))
}

func normalizeProviderKey(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "google":
		return "google"
	case "discord":
		return "discord"
	case "vk", "vkontakte":
		return "vkontakte"
	case "telegram":
		return "telegram"
	default:
		return ""
	}
}

func nullString(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}

func emailVerificationHash(email string) string {
	sum := sha1.Sum([]byte(strings.ToLower(strings.TrimSpace(email))))
	return hex.EncodeToString(sum[:])
}

func (h *Handler) buildSignedVerifyURL(userID, email string) string {
	hash := emailVerificationHash(email)
	expires := time.Now().Add(60 * time.Minute).Unix()
	payload := fmt.Sprintf("%s:%s:%d", userID, hash, expires)
	mac := hmac.New(sha256.New, []byte(h.jwtSecret))
	mac.Write([]byte("verify:" + payload))
	sig := hex.EncodeToString(mac.Sum(nil))
	apiPath := fmt.Sprintf("/v1/email/verify/%s/%s?expires=%d&signature=%s", userID, hash, expires, sig)
	apiURL := strings.TrimRight(h.apiPublicURL, "/") + apiPath
	frontend := strings.TrimRight(h.frontendURL, "/")
	return frontend + "/verify-email?verify_url=" + url.QueryEscape(apiURL)
}

func (h *Handler) SendEmailVerification(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ctx := r.Context()
	var verified *time.Time
	if err := h.dbOf(ctx).QueryRow(ctx, `SELECT email_verified_at FROM core.users WHERE id = $1`, claims.UserID).Scan(&verified); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	if verified != nil {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "message": "Email уже подтвержден."})
		return
	}
	verifyURL := h.buildSignedVerifyURL(claims.UserID, claims.Email)
	resp := map[string]any{"ok": true, "status": "verification-link-sent", "message": "Письмо для подтверждения отправлено."}
	if h.mail.Enabled() {
		_ = h.mail.Send(claims.Email, "Подтверждение email", mail.VerificationBody(verifyURL))
	} else if h.mail.DevExpose {
		resp["verification_url"] = verifyURL
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	userID := chi.URLParam(r, "id")
	hash := chi.URLParam(r, "hash")
	if userID != claims.UserID {
		writeError(w, http.StatusForbidden, "invalid verification user")
		return
	}
	if hash != emailVerificationHash(claims.Email) {
		writeError(w, http.StatusForbidden, "invalid verification hash")
		return
	}
	expiresStr := r.URL.Query().Get("expires")
	sig := r.URL.Query().Get("signature")
	expires, _ := strconv.ParseInt(expiresStr, 10, 64)
	if expires == 0 || time.Now().Unix() > expires {
		writeError(w, http.StatusBadRequest, "verification link expired")
		return
	}
	payload := fmt.Sprintf("%s:%s:%d", userID, hash, expires)
	mac := hmac.New(sha256.New, []byte(h.jwtSecret))
	mac.Write([]byte("verify:" + payload))
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(sig)) {
		writeError(w, http.StatusForbidden, "invalid signature")
		return
	}
	_, _ = h.dbOf(r.Context()).Exec(r.Context(), `UPDATE core.users SET email_verified_at = now() WHERE id = $1`, userID)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "message": "Email подтвержден"})
}

func (h *Handler) maybeSendVerificationEmail(userID, email string) {
	if !h.mail.Enabled() && !h.mail.DevExpose {
		return
	}
	verifyURL := h.buildSignedVerifyURL(userID, email)
	if h.mail.Enabled() {
		_ = h.mail.Send(email, "Подтверждение email", mail.VerificationBody(verifyURL))
	}
}

func publicProviderKey(stored string) string {
	if strings.EqualFold(strings.TrimSpace(stored), "vkontakte") {
		return "vk"
	}
	return strings.ToLower(strings.TrimSpace(stored))
}

func (h *Handler) telegramLoginUsername() string {
	if h.telegramBotToken == "" {
		return ""
	}
	h.tgBotMu.Lock()
	defer h.tgBotMu.Unlock()
	if time.Now().Before(h.tgBotUntil) {
		return h.tgBotUsername
	}
	username, err := h.telegramGetMe(h.telegramBotToken)
	if err != nil || username == "" {
		h.tgBotUsername = ""
		h.tgBotUntil = time.Now().Add(5 * time.Minute)
		return ""
	}
	h.tgBotUsername = username
	h.tgBotUntil = time.Now().Add(time.Hour)
	return username
}

func (h *Handler) SocialProviders(w http.ResponseWriter, r *http.Request) {
	providers := []string{}
	if h.oauth != nil {
		providers = h.oauth.ConfiguredKeys()
	}
	telegramUser := h.telegramLoginUsername()
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":        true,
		"providers": providers,
		"telegram": map[string]any{
			"enabled":      telegramUser != "",
			"bot_username": telegramUser,
		},
	})
}
