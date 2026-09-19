package handlers

import (
	"context"
	"crypto/rand"
	"errors"
	"log"
	"math"
	"math/big"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/vortanixapp/panel/pkg/i18n"
	"github.com/vortanixapp/panel/pkg/notify"
)

const (
	referralCookie   = "vtx_ref"
	referralAlphabet = "abcdefghjkmnpqrstuvwxyz23456789"
	referralCodeLen  = 8
	referralListMax  = 100
)

var referralCodePattern = regexp.MustCompile(`^[a-z0-9]{4,32}$`)

type referralConfig struct {
	Enabled    bool
	Percent    float64
	Months     int
	MinPayment float64
}

type referralReward struct {
	ReferrerID string
	Amount     float64
	Currency   string
	Percent    float64
}

func (h *Handler) referralSettings(ctx context.Context) referralConfig {
	cfg := referralConfig{Percent: 10, Months: 12}
	rows, err := h.dbOf(ctx).Query(ctx, `
		SELECT key, value FROM core.tenant_settings WHERE key = ANY($1)
	`, []string{"referral.enabled", "referral.percent", "referral.months", "referral.min_payment"})
	if err != nil {
		return cfg
	}
	defer rows.Close()
	for rows.Next() {
		var key string
		var raw []byte
		if rows.Scan(&key, &raw) != nil {
			continue
		}
		value := strings.TrimSpace(h.secrets.MustDecrypt(jsonValueToString(raw)))
		number, numErr := strconv.ParseFloat(strings.Replace(value, ",", ".", 1), 64)
		switch key {
		case "referral.enabled":
			cfg.Enabled = truthySetting(value)
		case "referral.percent":
			if numErr == nil {
				cfg.Percent = number
			}
		case "referral.months":
			if numErr == nil {
				cfg.Months = int(number)
			}
		case "referral.min_payment":
			if numErr == nil {
				cfg.MinPayment = number
			}
		}
	}
	cfg.Percent = math.Min(50, math.Max(1, cfg.Percent))
	cfg.Months = max(0, min(120, cfg.Months))
	cfg.MinPayment = math.Max(0, cfg.MinPayment)
	return cfg
}

func normalizeReferralCode(code string) string {
	code = strings.ToLower(strings.TrimSpace(code))
	if !referralCodePattern.MatchString(code) {
		return ""
	}
	return code
}

func referralCodeFrom(r *http.Request, explicit string) string {
	if code := normalizeReferralCode(explicit); code != "" {
		return code
	}
	if c, err := r.Cookie(referralCookie); err == nil {
		return normalizeReferralCode(c.Value)
	}
	return ""
}

func (h *Handler) attachReferrer(ctx context.Context, userID, code string) {
	if code == "" || userID == "" || !h.referralSettings(ctx).Enabled {
		return
	}
	var referrerID string
	err := h.dbOf(ctx).QueryRow(ctx, `
		UPDATE core.users u SET referrer_id = r.id, referred_at = now()
		FROM core.users r
		WHERE u.id = $1 AND u.referrer_id IS NULL
		  AND r.referral_code = $2 AND r.id <> u.id AND r.status = 'active'
		RETURNING r.id::text
	`, userID, code).Scan(&referrerID)
	if err != nil {
		return
	}
	audit(ctx, h.dbOf(ctx), userID, "referral.join", "user:"+userID, map[string]any{"referrer_id": referrerID})
	h.notifyUser(ctx, referrerID, notify.Event{
		Kind:      notify.KindReferralJoined,
		Title:     i18n.Key("notify.referral_joined.title"),
		Body:      i18n.Key("notify.referral_joined.body"),
		Action:    h.panelAction("notify.action.referrals", "/settings?tab=referrals"),
		DedupeKey: "referral.joined:" + userID,
	})
}

func randomReferralCode() (string, error) {
	limit := big.NewInt(int64(len(referralAlphabet)))
	out := make([]byte, referralCodeLen)
	for i := range out {
		n, err := rand.Int(rand.Reader, limit)
		if err != nil {
			return "", err
		}
		out[i] = referralAlphabet[n.Int64()]
	}
	return string(out), nil
}

func (h *Handler) ensureReferralCode(ctx context.Context, userID string) (string, error) {
	db := h.dbOf(ctx)
	var code string
	if err := db.QueryRow(ctx, `SELECT COALESCE(referral_code, '') FROM core.users WHERE id = $1`, userID).Scan(&code); err != nil {
		return "", err
	}
	if code != "" {
		return code, nil
	}
	for range 5 {
		candidate, err := randomReferralCode()
		if err != nil {
			return "", err
		}
		tag, err := db.Exec(ctx, `
			UPDATE core.users SET referral_code = $2 WHERE id = $1 AND referral_code IS NULL
		`, userID, candidate)
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				continue
			}
			return "", err
		}
		if tag.RowsAffected() == 0 {
			err = db.QueryRow(ctx, `SELECT COALESCE(referral_code, '') FROM core.users WHERE id = $1`, userID).Scan(&code)
			return code, err
		}
		return candidate, nil
	}
	return "", errors.New("referral code collision")
}

func roundMoney(v float64) float64 {
	return math.Round(v*100) / 100
}

func (h *Handler) creditReferralReward(ctx context.Context, tx pgx.Tx, paymentID, userID string, paid float64, currency string) *referralReward {
	cfg := h.referralSettings(ctx)
	if !cfg.Enabled || paid <= 0 || paid+0.009 < cfg.MinPayment {
		return nil
	}
	var referrerID string
	var referredAt time.Time
	err := tx.QueryRow(ctx, `
		SELECT r.id::text, u.referred_at
		FROM core.users u
		JOIN core.users r ON r.id = u.referrer_id AND r.status = 'active'
		WHERE u.id = $1::uuid AND u.referred_at IS NOT NULL
	`, userID).Scan(&referrerID, &referredAt)
	if err != nil {
		return nil
	}
	if cfg.Months > 0 && time.Now().After(referredAt.AddDate(0, cfg.Months, 0)) {
		return nil
	}
	amount := roundMoney(paid * cfg.Percent / 100)
	if amount < 0.01 {
		return nil
	}

	sp, err := tx.Begin(ctx)
	if err != nil {
		return nil
	}
	defer sp.Rollback(ctx)
	tag, err := sp.Exec(ctx, `
		INSERT INTO core.referral_rewards (referrer_id, referred_id, payment_id, amount, currency, percent)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, $6)
		ON CONFLICT (payment_id) DO NOTHING
	`, referrerID, userID, paymentID, amount, currency, cfg.Percent)
	if err != nil || tag.RowsAffected() == 0 {
		return nil
	}
	var walletID string
	if sp.QueryRow(ctx, `
		INSERT INTO core.wallets (user_id, currency, is_default)
		VALUES ($1::uuid, $2, NOT EXISTS(SELECT 1 FROM core.wallets WHERE user_id = $1::uuid AND is_default))
		ON CONFLICT (user_id, currency) DO UPDATE SET updated_at = now()
		RETURNING id::text
	`, referrerID, currency).Scan(&walletID) != nil {
		return nil
	}
	if _, err := sp.Exec(ctx, `
		UPDATE core.wallets SET balance = balance + $2, updated_at = now() WHERE id = $1::uuid
	`, walletID, amount); err != nil {
		return nil
	}
	if _, err := sp.Exec(ctx, `
		INSERT INTO core.transactions (wallet_id, type, amount, description, source_type, source_id)
		VALUES ($1::uuid, 'credit', $2, $3, 'referral', $4::uuid)
	`, walletID, amount, "Реферальное вознаграждение", paymentID); err != nil {
		return nil
	}
	if sp.Commit(ctx) != nil {
		return nil
	}
	return &referralReward{ReferrerID: referrerID, Amount: amount, Currency: currency, Percent: cfg.Percent}
}

func (h *Handler) notifyReferralReward(ctx context.Context, paymentID string, reward *referralReward) {
	if reward == nil {
		return
	}
	h.notifyUser(ctx, reward.ReferrerID, notify.Event{
		Kind:  notify.KindReferralReward,
		Title: i18n.Key("notify.referral_reward.title"),
		Body: i18n.Key("notify.referral_reward.body", i18n.Params{
			"amount":   formatMoney(reward.Amount),
			"currency": reward.Currency,
			"percent":  strconv.FormatFloat(reward.Percent, 'f', -1, 64),
		}),
		Action:    h.panelAction("notify.action.referrals", "/settings?tab=referrals"),
		Meta:      map[string]any{"payment_id": paymentID, "amount": reward.Amount, "currency": reward.Currency},
		DedupeKey: "referral.reward:" + paymentID,
	})
}

func reverseReferralReward(ctx context.Context, tx pgx.Tx, paymentID string, refunded, paid float64) {
	if paid <= 0 || refunded <= 0 {
		return
	}
	sp, err := tx.Begin(ctx)
	if err != nil {
		return
	}
	defer sp.Rollback(ctx)
	if err := reverseReferralRewardIn(ctx, sp, paymentID, refunded, paid); err != nil {
		log.Printf("реферальное вознаграждение по платежу %s не списано: %v", paymentID, err)
		return
	}
	if err := sp.Commit(ctx); err != nil {
		log.Printf("реферальное вознаграждение по платежу %s не списано: %v", paymentID, err)
	}
}

func reverseReferralRewardIn(ctx context.Context, tx pgx.Tx, paymentID string, refunded, paid float64) error {
	var referrerID, currency string
	var amount, reversed float64
	err := tx.QueryRow(ctx, `
		SELECT referrer_id::text, currency, amount::float8, reversed::float8
		FROM core.referral_rewards WHERE payment_id = $1::uuid FOR UPDATE
	`, paymentID).Scan(&referrerID, &currency, &amount, &reversed)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	part := roundMoney(amount * math.Min(1, refunded/paid))
	if part > amount-reversed {
		part = roundMoney(amount - reversed)
	}
	if part < 0.01 {
		return nil
	}
	var walletID string
	err = tx.QueryRow(ctx, `
		SELECT id::text FROM core.wallets WHERE user_id = $1::uuid AND currency = $2
	`, referrerID, currency).Scan(&walletID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE core.wallets SET balance = balance - $2, updated_at = now() WHERE id = $1::uuid
	`, walletID, part); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO core.transactions (wallet_id, type, amount, description, source_type, source_id)
		VALUES ($1::uuid, 'debit', $2, $3, 'referral', $4::uuid)
	`, walletID, part, "Реферальное вознаграждение списано: приглашённому вернули платёж", paymentID); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		UPDATE core.referral_rewards SET reversed = reversed + $2 WHERE payment_id = $1::uuid
	`, paymentID, part)
	return err
}

func maskEmail(email string) string {
	email = strings.TrimSpace(email)
	if strings.HasSuffix(email, "@telegram.local") {
		return "Telegram"
	}
	at := strings.LastIndex(email, "@")
	if at <= 0 {
		return "***"
	}
	local := []rune(email[:at])
	keep := 2
	if len(local) <= 3 {
		keep = 1
	}
	return string(local[:keep]) + "***@" + email[at+1:]
}

type moneyAmount struct {
	Currency string  `json:"currency"`
	Amount   float64 `json:"amount"`
}

func (h *Handler) referralTotals(ctx context.Context, referrerID string) (map[string][]moneyAmount, []moneyAmount, int) {
	perUser := map[string][]moneyAmount{}
	total := map[string]float64{}
	var order []string
	payers := 0
	rows, err := h.dbOf(ctx).Query(ctx, `
		SELECT COALESCE(referred_id::text, ''), currency, SUM(amount - reversed)::float8
		FROM core.referral_rewards
		WHERE referrer_id = $1
		GROUP BY referred_id, currency
	`, referrerID)
	if err != nil {
		return perUser, nil, 0
	}
	defer rows.Close()
	paid := map[string]bool{}
	for rows.Next() {
		var referred, currency string
		var sum float64
		if rows.Scan(&referred, &currency, &sum) != nil {
			continue
		}
		sum = roundMoney(sum)
		if referred != "" {
			perUser[referred] = append(perUser[referred], moneyAmount{Currency: currency, Amount: sum})
			if sum > 0 && !paid[referred] {
				paid[referred] = true
				payers++
			}
		}
		if _, seen := total[currency]; !seen {
			order = append(order, currency)
		}
		total[currency] += sum
	}
	earned := make([]moneyAmount, 0, len(order))
	for _, currency := range order {
		earned = append(earned, moneyAmount{Currency: currency, Amount: roundMoney(total[currency])})
	}
	return perUser, earned, payers
}

func (h *Handler) referralsVisible(ctx context.Context, userID string) bool {
	if h.referralSettings(ctx).Enabled {
		return true
	}
	var exists bool
	_ = h.dbOf(ctx).QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM core.users WHERE referrer_id = $1)
		    OR EXISTS(SELECT 1 FROM core.referral_rewards WHERE referrer_id = $1)
	`, userID).Scan(&exists)
	return exists
}

func (h *Handler) AccountReferrals(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ctx := r.Context()
	cfg := h.referralSettings(ctx)
	db := h.dbOf(ctx)

	code := ""
	if cfg.Enabled {
		generated, err := h.ensureReferralCode(ctx, claims.UserID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Не удалось выдать реферальную ссылку")
			return
		}
		code = generated
	} else {
		_ = db.QueryRow(ctx, `SELECT COALESCE(referral_code, '') FROM core.users WHERE id = $1`, claims.UserID).Scan(&code)
	}

	perUser, earned, payers := h.referralTotals(ctx, claims.UserID)

	var invited int
	_ = db.QueryRow(ctx, `SELECT count(*) FROM core.users WHERE referrer_id = $1`, claims.UserID).Scan(&invited)

	rows, err := db.Query(ctx, `
		SELECT id::text, email, COALESCE(referred_at, created_at)
		FROM core.users
		WHERE referrer_id = $1
		ORDER BY COALESCE(referred_at, created_at) DESC
		LIMIT $2
	`, claims.UserID, referralListMax)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()
	list := []map[string]any{}
	for rows.Next() {
		var id, email string
		var joined time.Time
		if rows.Scan(&id, &email, &joined) != nil {
			continue
		}
		var until any
		active := true
		if cfg.Months > 0 {
			end := joined.AddDate(0, cfg.Months, 0)
			until = end.Format(time.RFC3339)
			active = time.Now().Before(end)
		}
		rewards := perUser[id]
		if rewards == nil {
			rewards = []moneyAmount{}
		}
		deleted := strings.HasSuffix(email, "@deleted.invalid")
		masked := maskEmail(email)
		if deleted {
			masked = ""
		}
		list = append(list, map[string]any{
			"email":        masked,
			"deleted":      deleted,
			"joined_at":    joined.Format(time.RFC3339),
			"earned":       rewards,
			"active":       active,
			"active_until": until,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"enabled":     cfg.Enabled,
		"code":        code,
		"percent":     cfg.Percent,
		"months":      cfg.Months,
		"min_payment": cfg.MinPayment,
		"currency":    h.defaultCurrency(ctx),
		"stats": map[string]any{
			"invited": invited,
			"paid":    payers,
			"earned":  earned,
		},
		"referrals": list,
	})
}
