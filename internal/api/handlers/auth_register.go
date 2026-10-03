package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/vortanixapp/panel/internal/api/mail"
	"github.com/vortanixapp/panel/pkg/i18n"
	"github.com/vortanixapp/panel/pkg/settingsreg"
)

const (
	uniformResponseFloor = 300 * time.Millisecond
	localeCookie         = "vortanix-locale"
)

type pendingRegistration struct {
	Email        string
	PasswordHash string
	FirstName    string
	LastName     string
	ReferralCode string
	Consents     []string
}

func (h *Handler) registrationConfirmRequired(ctx context.Context) bool {
	if !settingsreg.AuthRegisterConfirmEmail.Bool() {
		return false
	}
	cfg := h.mailerConfig(ctx)
	return cfg.Configured() && !cfg.Silent()
}

func holdUntil(started time.Time, floor time.Duration) {
	if wait := floor - time.Since(started); wait > 0 {
		time.Sleep(wait)
	}
}

func (h *Handler) frontendLink(r *http.Request, path string) string {
	base := strings.TrimRight(strings.TrimSpace(h.frontendURL), "/")
	if base == "" {
		base = strings.TrimRight(h.publicBaseURL(r), "/")
	}
	return base + path
}

func (h *Handler) registerWithConfirmation(w http.ResponseWriter, r *http.Request, started time.Time, p pendingRegistration) {
	ctx := r.Context()
	if !h.allowAttempt(ctx, "register:mail:"+sha256Hex(p.Email)[:16], intOf(settingsreg.AuthRegisterMailPerHour), time.Hour) {
		writeError(w, http.StatusTooManyRequests, "Слишком много попыток, попробуйте позже")
		return
	}

	var exists bool
	if err := h.dbOf(ctx).QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM core.users WHERE email = $1)`, p.Email).Scan(&exists); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}

	loc := i18n.For(ctx, h.dbOf(ctx), cookieValue(r, localeCookie))
	brand := h.mailBrand(ctx, r)
	if exists {
		if claimed, err := h.cache.Claim(ctx, "register:exists:"+sha256Hex(p.Email)[:16], settingsreg.AuthExistsMailInterval.Duration()); err != nil || claimed {
			msg := mail.RegistrationExistsEmail(loc, brand, h.frontendLink(r, "/login"))
			msg.To = p.Email
			h.sendMailAsync("mail.register_exists", "", p.Email, msg)
		}
	} else {
		token := randomToken(32)
		if err := h.savePendingRegistration(ctx, r, p, sha256Hex(token)); err != nil {
			writeError(w, http.StatusInternalServerError, "database error")
			return
		}
		msg := mail.RegistrationConfirmEmail(loc, brand, h.frontendLink(r, "/verify-email?register="+url.QueryEscape(token)))
		msg.To = p.Email
		h.sendMailAsync("mail.register_confirm", "", p.Email, msg)
	}

	holdUntil(started, uniformResponseFloor)
	writeJSON(w, http.StatusAccepted, map[string]any{"status": "confirmation_sent", "email": p.Email})
}

func (h *Handler) savePendingRegistration(ctx context.Context, r *http.Request, p pendingRegistration, tokenHash string) error {
	db := h.dbOf(ctx)
	_, _ = db.Exec(ctx, `DELETE FROM core.pending_registrations WHERE expires_at < now()`)
	agent := r.Header.Get("User-Agent")
	if len(agent) > 300 {
		agent = agent[:300]
	}
	consents := p.Consents
	if consents == nil {
		consents = []string{}
	}
	_, err := db.Exec(ctx, `
		INSERT INTO core.pending_registrations
			(email, password_hash, first_name, last_name, referral_code, consents, ip, user_agent, token_hash, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, now() + $10::interval)
		ON CONFLICT (email) DO UPDATE SET
			password_hash = EXCLUDED.password_hash,
			first_name    = EXCLUDED.first_name,
			last_name     = EXCLUDED.last_name,
			referral_code = EXCLUDED.referral_code,
			consents      = EXCLUDED.consents,
			ip            = EXCLUDED.ip,
			user_agent    = EXCLUDED.user_agent,
			token_hash    = EXCLUDED.token_hash,
			expires_at    = EXCLUDED.expires_at,
			created_at    = now()
	`, p.Email, p.PasswordHash, p.FirstName, p.LastName, p.ReferralCode, consents,
		clientIP(r), agent, tokenHash, settingsreg.AuthRegisterConfirmTTL.Duration().String())
	return err
}

func (h *Handler) ConfirmRegistration(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Token) == "" {
		writeError(w, http.StatusBadRequest, "Ссылка подтверждения неполная")
		return
	}
	if h.tooManyAttempts(w, r, "register-confirm", intOf(settingsreg.AuthConfirmAttempts), attemptsWindow()) {
		return
	}
	ctx := r.Context()
	tx, err := h.dbOf(ctx).Begin(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer tx.Rollback(ctx)

	var p pendingRegistration
	var ip, agent string
	err = tx.QueryRow(ctx, `
		DELETE FROM core.pending_registrations
		WHERE token_hash = $1 AND expires_at > now()
		RETURNING email, password_hash, first_name, last_name, referral_code, consents, ip, user_agent
	`, sha256Hex(strings.TrimSpace(body.Token))).Scan(
		&p.Email, &p.PasswordHash, &p.FirstName, &p.LastName, &p.ReferralCode, &p.Consents, &ip, &agent)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusGone, "Ссылка устарела или уже использована. Зарегистрируйтесь заново")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}

	var userID string
	err = tx.QueryRow(ctx, `
		INSERT INTO core.users (email, password_hash, role, email_verified_at)
		VALUES ($1, $2, 'user', now())
		RETURNING id::text
	`, p.Email, p.PasswordHash).Scan(&userID)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		writeError(w, http.StatusConflict, "Эта почта уже зарегистрирована: войдите или восстановите пароль")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create user")
		return
	}
	if displayName := strings.TrimSpace(p.FirstName + " " + p.LastName); displayName != "" {
		if _, err := tx.Exec(ctx, `
			INSERT INTO core.user_profiles (user_id, display_name, first_name, last_name, updated_at)
			VALUES ($1, $2, NULLIF($3, ''), NULLIF($4, ''), now())
		`, userID, displayName, p.FirstName, p.LastName); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to create profile")
			return
		}
	}
	if err := tx.Commit(ctx); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}

	h.ensureDefaultWallet(ctx, userID)
	if len(p.Consents) > 0 {
		if err := h.recordConsentsAt(ctx, userID, p.Consents, "register", ip, agent); err != nil && !errors.Is(err, pgx.ErrNoRows) {
			log.Printf("согласия при регистрации %s не записаны: %v", userID, err)
		}
	}
	h.attachReferrer(ctx, userID, p.ReferralCode)

	access, refresh, err := h.issueAuthTokens(r, userID, p.Email, "user", 0)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to issue tokens")
		return
	}
	h.startSession(w, r, userID, p.Email, "user", access, refresh, 0)
	writeJSON(w, http.StatusCreated, map[string]any{
		"access_token":  access,
		"refresh_token": refresh,
		"user": map[string]string{
			"id":    userID,
			"email": p.Email,
			"role":  "user",
		},
	})
}
