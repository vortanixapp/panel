package handlers

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/pquerna/otp/totp"
	"github.com/vortanixapp/panel/pkg/i18n"
	"github.com/vortanixapp/panel/pkg/notify"
	"golang.org/x/crypto/bcrypt"
)

const (
	recoveryCodeCount    = 10
	recoveryCodeAlphabet = "abcdefghjkmnpqrstuvwxyz23456789"
	loginHistoryPageMax  = 50
)

func normalizeRecoveryCode(code string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(code) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func hashRecoveryCode(code string) string {
	sum := sha256.Sum256([]byte(normalizeRecoveryCode(code)))
	return hex.EncodeToString(sum[:])
}

func newRecoveryCodes(n int) (codes, hashes []string) {
	max := big.NewInt(int64(len(recoveryCodeAlphabet)))
	for len(codes) < n {
		raw := make([]byte, 0, 8)
		for len(raw) < 8 {
			v, err := rand.Int(rand.Reader, max)
			if err != nil {
				continue
			}
			raw = append(raw, recoveryCodeAlphabet[v.Int64()])
		}
		code := string(raw[:4]) + "-" + string(raw[4:])
		codes = append(codes, code)
		hashes = append(hashes, hashRecoveryCode(code))
	}
	return codes, hashes
}

func (h *Handler) consumeRecoveryCode(ctx context.Context, userID, code string) (int, bool) {
	if len(normalizeRecoveryCode(code)) != 8 {
		return 0, false
	}
	var left int
	err := h.dbOf(ctx).QueryRow(ctx, `
		UPDATE core.two_factor_secrets
		SET recovery_codes = recovery_codes - $2::text
		WHERE user_id = $1 AND recovery_codes ? $2::text
		RETURNING jsonb_array_length(recovery_codes)
	`, userID, hashRecoveryCode(code)).Scan(&left)
	return left, err == nil
}

func (h *Handler) checkAccountPassword(w http.ResponseWriter, r *http.Request, userID string, passwordSet bool, hash, password string) bool {
	if !passwordSet {
		return true
	}
	if strings.TrimSpace(password) == "" {
		writeError(w, http.StatusBadRequest, "Введите текущий пароль")
		return false
	}
	if h.tooManyAttempts(w, r, "account-password", 10, 15*time.Minute, userID) {
		return false
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		writeError(w, http.StatusUnauthorized, "Неверный текущий пароль")
		return false
	}
	return true
}

func (h *Handler) checkSecondFactor(ctx context.Context, r *http.Request, userID, code string) bool {
	code = strings.TrimSpace(code)
	var sealed string
	if err := h.dbOf(ctx).QueryRow(ctx, `SELECT secret FROM core.two_factor_secrets WHERE user_id = $1`, userID).Scan(&sealed); err != nil || sealed == "" {
		return false
	}
	digits := strings.ReplaceAll(code, " ", "")
	if len(digits) == 6 && strings.Trim(digits, "0123456789") == "" && totp.Validate(digits, h.secrets.MustDecrypt(sealed)) {
		return true
	}
	left, ok := h.consumeRecoveryCode(ctx, userID, code)
	if !ok {
		return false
	}
	audit(ctx, h.dbOf(ctx), userID, "user.2fa_recovery_code", "user:"+userID, map[string]any{"left": left, "ip": clientIP(r)})
	h.notifyUser(ctx, userID, notify.Event{
		Kind:   notify.KindRecoveryCode,
		Title:  i18n.Key("notify.recovery_code_used.title"),
		Body:   i18n.Key("notify.recovery_code_used.body", i18n.Params{"left": strconv.Itoa(left), "ip": clientIP(r)}),
		Action: h.panelAction("notify.action.security", "/settings?tab=security"),
	})
	return true
}

func (h *Handler) RegenerateRecoveryCodes(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var body struct {
		Code string `json:"code"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	ctx := r.Context()
	db := h.dbOf(ctx)
	var enabled bool
	_ = db.QueryRow(ctx, `SELECT two_factor_enabled FROM core.users WHERE id = $1`, claims.UserID).Scan(&enabled)
	if !enabled {
		writeError(w, http.StatusConflict, "Сначала включите двухфакторную защиту")
		return
	}
	if strings.TrimSpace(body.Code) == "" {
		writeError(w, http.StatusBadRequest, "Введите код из приложения")
		return
	}
	if h.tooManyAttempts(w, r, "2fa-codes", 10, 15*time.Minute, claims.UserID) {
		return
	}
	if !h.checkSecondFactor(ctx, r, claims.UserID, body.Code) {
		writeError(w, http.StatusUnauthorized, "Код не подошёл")
		return
	}
	codes, hashes := newRecoveryCodes(recoveryCodeCount)
	hashesJSON, _ := json.Marshal(hashes)
	if _, err := db.Exec(ctx, `UPDATE core.two_factor_secrets SET recovery_codes = $2::jsonb WHERE user_id = $1`, claims.UserID, string(hashesJSON)); err != nil {
		writeError(w, http.StatusInternalServerError, "Не удалось выпустить новые коды")
		return
	}
	audit(ctx, db, claims.UserID, "user.2fa_recovery_regenerate", "user:"+claims.UserID, nil)
	writeJSON(w, http.StatusOK, map[string]any{"recovery_codes": codes})
}

func (h *Handler) ListAccountLogins(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > loginHistoryPageMax {
		limit = 20
	}
	before, _ := strconv.ParseInt(r.URL.Query().Get("before"), 10, 64)
	ctx := r.Context()
	rows, err := h.dbOf(ctx).Query(ctx, `
		SELECT id, success, reason, ip, user_agent, created_at
		FROM core.login_attempts
		WHERE user_id = $1 AND ($2::bigint = 0 OR id < $2::bigint)
		ORDER BY id DESC
		LIMIT $3
	`, claims.UserID, before, limit+1)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Не удалось загрузить историю входов")
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	var last int64
	hasMore := false
	for rows.Next() {
		var id int64
		var success bool
		var reason, ip, agent string
		var created time.Time
		if rows.Scan(&id, &success, &reason, &ip, &agent, &created) != nil {
			continue
		}
		if len(items) == limit {
			hasMore = true
			break
		}
		last = id
		items = append(items, map[string]any{
			"id":         id,
			"success":    success,
			"reason":     reason,
			"ip":         ip,
			"device":     deviceFromUserAgent(agent),
			"created_at": created,
		})
	}
	resp := map[string]any{"logins": items, "has_more": hasMore}
	if hasMore {
		resp["next_before"] = last
	}
	writeJSON(w, http.StatusOK, resp)
}

type deletionBlocker struct {
	Kind    string           `json:"kind"`
	Count   int              `json:"count,omitempty"`
	Amounts []deletionAmount `json:"amounts,omitempty"`
}

type deletionAmount struct {
	Currency string  `json:"currency"`
	Amount   float64 `json:"amount"`
}

func (h *Handler) deletionBlockers(ctx context.Context, userID string) []deletionBlocker {
	db := h.dbOf(ctx)
	out := []deletionBlocker{}
	var servers, hosting, refunds int
	_ = db.QueryRow(ctx, `SELECT COUNT(*) FROM core.servers WHERE user_id = $1`, userID).Scan(&servers)
	_ = db.QueryRow(ctx, `SELECT COUNT(*) FROM core.hosting_accounts WHERE user_id = $1 AND status <> 'terminated'`, userID).Scan(&hosting)
	_ = db.QueryRow(ctx, `SELECT COUNT(*) FROM core.balance_refund_requests WHERE user_id = $1 AND status = 'pending'`, userID).Scan(&refunds)
	if servers > 0 {
		out = append(out, deletionBlocker{Kind: "servers", Count: servers})
	}
	if hosting > 0 {
		out = append(out, deletionBlocker{Kind: "hosting", Count: hosting})
	}
	if refunds > 0 {
		out = append(out, deletionBlocker{Kind: "refund", Count: refunds})
	}
	amounts := []deletionAmount{}
	rows, err := db.Query(ctx, `SELECT currency, balance::float8 FROM core.wallets WHERE user_id = $1 AND balance > 0.004 ORDER BY currency`, userID)
	if err == nil {
		for rows.Next() {
			var a deletionAmount
			if rows.Scan(&a.Currency, &a.Amount) == nil {
				amounts = append(amounts, a)
			}
		}
		rows.Close()
	}
	if len(amounts) > 0 {
		out = append(out, deletionBlocker{Kind: "balance", Amounts: amounts})
	}
	return out
}

func (h *Handler) AccountDeleteCheck(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ctx := r.Context()
	var passwordSet, twoFA bool
	_ = h.dbOf(ctx).QueryRow(ctx, `SELECT password_set, two_factor_enabled FROM core.users WHERE id = $1`, claims.UserID).Scan(&passwordSet, &twoFA)
	staff := isStaffRole(claims.Role)
	blockers := []deletionBlocker{}
	if !staff {
		blockers = h.deletionBlockers(ctx, claims.UserID)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"allowed":  !staff && len(blockers) == 0,
		"staff":    staff,
		"blockers": blockers,
		"requires": map[string]bool{"password": passwordSet, "two_factor": twoFA},
	})
}
