package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/vortanix/vortanix/internal/api/paneljwt"
)

func requireStaff(w http.ResponseWriter, claims *paneljwt.Claims) bool {
	if claims == nil || !isStaffRole(claims.Role) {
		writeError(w, http.StatusForbidden, "forbidden")
		return false
	}
	return true
}

func contactString(contacts map[string]any, key string) string {
	if contacts == nil {
		return ""
	}
	v, ok := contacts[key]
	if !ok || v == nil {
		return ""
	}
	s := strings.TrimSpace(fmt.Sprint(v))
	if s == "" || s == "<nil>" {
		return ""
	}
	return s
}

func parseContacts(raw []byte) map[string]any {
	out := map[string]any{}
	if len(raw) == 0 {
		return out
	}
	_ = json.Unmarshal(raw, &out)
	return out
}

func adminUserListItem(
	id, email, displayName, role, status string,
	emailVerified *string, balance float64, serversCount int, created string,
) map[string]any {
	isAdmin := role == "admin" || role == "owner"
	isBlocked := status == "disabled"
	return map[string]any{
		"id":                id,
		"email":             email,
		"name":              displayName,
		"role":              role,
		"status":            status,
		"is_admin":          isAdmin,
		"is_blocked":        isBlocked,
		"email_verified_at": emailVerified,
		"balance":           balance,
		"servers_count":     serversCount,
		"created_at":        created,
	}
}

func (h *Handler) adminUserCounts(ctx context.Context, tenantID string) map[string]int {
	var total, admins, support, users, active, unverified, blocked int
	_ = h.dbOf(ctx).QueryRow(ctx, `
		SELECT
			COUNT(*)::int,
			COUNT(*) FILTER (WHERE role IN ('admin', 'owner'))::int,
			COUNT(*) FILTER (WHERE role = 'support')::int,
			COUNT(*) FILTER (WHERE role = 'user')::int,
			COUNT(*) FILTER (WHERE status <> 'disabled' AND email_verified_at IS NOT NULL)::int,
			COUNT(*) FILTER (WHERE status <> 'disabled' AND email_verified_at IS NULL)::int,
			COUNT(*) FILTER (WHERE status = 'disabled')::int
		FROM core.users WHERE tenant_id = $1
	`, tenantID).Scan(&total, &admins, &support, &users, &active, &unverified, &blocked)
	return map[string]int{
		"total":      total,
		"admins":     admins,
		"support":    support,
		"users":      users,
		"active":     active,
		"unverified": unverified,
		"blocked":    blocked,
	}
}

func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}

	q := strings.TrimSpace(r.URL.Query().Get("q"))
	roleFilter := strings.TrimSpace(r.URL.Query().Get("role"))
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	perPage := 15
	if page < 1 {
		page = 1
	}
	usePagination := r.URL.Query().Has("page") || q != "" || roleFilter != ""

	ctx := r.Context()
	tenantID := claims.TenantID

	where := []string{"u.tenant_id = $1"}
	args := []any{tenantID}
	argN := 2

	if q != "" {
		pattern := "%" + q + "%"
		where = append(where, fmt.Sprintf(`(
			u.email ILIKE $%d OR COALESCE(p.display_name, '') ILIKE $%d OR
			COALESCE(p.first_name, '') ILIKE $%d OR COALESCE(p.last_name, '') ILIKE $%d OR
			COALESCE(p.contacts->>'public_id', '') ILIKE $%d
		)`, argN, argN, argN, argN, argN))
		args = append(args, pattern)
		argN++
	}
	if roleFilter != "" && roleFilter != "all" {
		where = append(where, fmt.Sprintf("u.role = $%d", argN))
		args = append(args, roleFilter)
		argN++
	}
	whereSQL := strings.Join(where, " AND ")

	var total int
	countQ := `SELECT COUNT(*)::int FROM core.users u
		LEFT JOIN core.user_profiles p ON p.user_id = u.id WHERE ` + whereSQL
	if err := h.dbOf(ctx).QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}

	listQ := `
		SELECT
			u.id::text,
			u.email,
			COALESCE(NULLIF(p.display_name, ''), NULLIF(p.first_name, ''), ''),
			u.role,
			u.status,
			u.email_verified_at::text,
			COALESCE((
				SELECT w.balance::float8
				FROM core.wallets w
				WHERE w.user_id = u.id AND w.is_default = true
				LIMIT 1
			), 0),
			COALESCE((
				SELECT COUNT(*)::int
				FROM core.servers s
				WHERE s.user_id = u.id AND s.tenant_id = u.tenant_id
			), 0),
			u.created_at::text
		FROM core.users u
		LEFT JOIN core.user_profiles p ON p.user_id = u.id
		WHERE ` + whereSQL + `
		ORDER BY u.created_at DESC`

	if usePagination {
		offset := (page - 1) * perPage
		listQ += fmt.Sprintf(" LIMIT %d OFFSET %d", perPage, offset)
	}

	rows, err := h.dbOf(ctx).Query(ctx, listQ, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()

	items := []map[string]any{}
	for rows.Next() {
		var id, email, name, role, status, created string
		var emailVerified *string
		var balance float64
		var serversCount int
		if err := rows.Scan(&id, &email, &name, &role, &status, &emailVerified, &balance, &serversCount, &created); err != nil {
			writeError(w, http.StatusInternalServerError, "database error")
			return
		}
		items = append(items, adminUserListItem(id, email, name, role, status, emailVerified, balance, serversCount, created))
	}

	if !usePagination {
		writeJSON(w, http.StatusOK, map[string]any{"users": items})
		return
	}

	lastPage := int(math.Ceil(float64(total) / float64(perPage)))
	if lastPage < 1 {
		lastPage = 1
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"users": map[string]any{
			"data":         items,
			"current_page": page,
			"last_page":    lastPage,
			"per_page":     perPage,
			"total":        total,
		},
		"q":      q,
		"role":   roleFilter,
		"counts": h.adminUserCounts(ctx, tenantID),
	})
}

func (h *Handler) adminUserDetail(ctx context.Context, tenantID, userID string) (map[string]any, error) {
	var email, role, status, created string
	var emailVerified *string
	var twoFA bool
	err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT email, role, status, email_verified_at::text, two_factor_enabled, created_at::text
		FROM core.users WHERE id = $1 AND tenant_id = $2
	`, userID, tenantID).Scan(&email, &role, &status, &emailVerified, &twoFA, &created)
	if err != nil {
		return nil, err
	}

	var displayName, firstName, lastName, phone *string
	var contactsRaw []byte
	_ = h.dbOf(ctx).QueryRow(ctx, `
		SELECT display_name, first_name, last_name, phone, contacts
		FROM core.user_profiles WHERE user_id = $1
	`, userID).Scan(&displayName, &firstName, &lastName, &phone, &contactsRaw)

	contacts := parseContacts(contactsRaw)
	name := ""
	if displayName != nil && strings.TrimSpace(*displayName) != "" {
		name = strings.TrimSpace(*displayName)
	} else if firstName != nil {
		name = strings.TrimSpace(*firstName)
	}

	lastNameStr := ""
	if lastName != nil {
		lastNameStr = strings.TrimSpace(*lastName)
	}
	phoneStr := ""
	if phone != nil {
		phoneStr = strings.TrimSpace(*phone)
	}

	user := map[string]any{
		"id":                 userID,
		"name":               name,
		"last_name":          lastNameStr,
		"public_id":          contactString(contacts, "public_id"),
		"email":              email,
		"phone":              phoneStr,
		"telegram_id":        contactString(contacts, "telegram_id"),
		"discord_id":         contactString(contacts, "discord_id"),
		"vk_id":              contactString(contacts, "vk_id"),
		"role":               role,
		"status":             status,
		"is_blocked":         status == "disabled",
		"two_factor_enabled": twoFA,
		"email_verified_at":  emailVerified,
		"created_at":         created,
	}

	wallets := []map[string]any{}
	var defaultWallet map[string]any
	wrows, err := h.dbOf(ctx).Query(ctx, `
		SELECT id::text, currency, balance::float8, is_default
		FROM core.wallets WHERE user_id = $1 AND tenant_id = $2
		ORDER BY is_default DESC, currency
	`, userID, tenantID)
	if err == nil {
		defer wrows.Close()
		for wrows.Next() {
			var wid, cur string
			var bal float64
			var isDefault bool
			if wrows.Scan(&wid, &cur, &bal, &isDefault) != nil {
				continue
			}
			w := map[string]any{
				"id":         wid,
				"currency":   cur,
				"balance":    bal,
				"is_default": isDefault,
			}
			wallets = append(wallets, w)
			if isDefault || defaultWallet == nil {
				defaultWallet = w
			}
		}
	}

	servers := []map[string]any{}
	srows, err := h.dbOf(ctx).Query(ctx, `
		SELECT s.id::text, s.name, COALESCE(s.ip_address, ''), COALESCE(s.primary_port, 0),
		       COALESCE(g.name, s.game_id, '')
		FROM core.servers s
		LEFT JOIN core.games g ON g.tenant_id = s.tenant_id AND g.slug = s.game_id
		WHERE s.user_id = $1 AND s.tenant_id = $2
		ORDER BY s.created_at DESC
		LIMIT 20
	`, userID, tenantID)
	if err == nil {
		defer srows.Close()
		for srows.Next() {
			var sid, sname, ip, gameName string
			var port int
			if srows.Scan(&sid, &sname, &ip, &port, &gameName) != nil {
				continue
			}
			game := map[string]any{"name": gameName}
			if gameName == "" {
				game = nil
			}
			servers = append(servers, map[string]any{
				"id":         sid,
				"name":       sname,
				"ip_address": ip,
				"port":       port,
				"game":       game,
			})
		}
	}

	transactions := []map[string]any{}
	trows, err := h.dbOf(ctx).Query(ctx, `
		SELECT t.id::text, t.type, t.amount::float8, COALESCE(t.description, ''), t.created_at::text, w.currency
		FROM core.transactions t
		JOIN core.wallets w ON w.id = t.wallet_id
		WHERE w.user_id = $1 AND w.tenant_id = $2
		ORDER BY t.created_at DESC
		LIMIT 20
	`, userID, tenantID)
	if err == nil {
		defer trows.Close()
		for trows.Next() {
			var tid, typ, desc, createdAt, currency string
			var amount float64
			if trows.Scan(&tid, &typ, &amount, &desc, &createdAt, &currency) != nil {
				continue
			}
			transactions = append(transactions, map[string]any{
				"id":          tid,
				"type":        typ,
				"amount":      amount,
				"description": desc,
				"created_at":  createdAt,
				"wallet":      map[string]any{"currency": currency},
			})
		}
	}

	sessions := []map[string]any{}
	sessRows, err := h.dbOf(ctx).Query(ctx, `
		SELECT id::text, COALESCE(ip_address, ''), COALESCE(user_agent, ''), last_active::text
		FROM core.user_sessions
		WHERE user_id = $1 AND tenant_id = $2
		ORDER BY last_active DESC
		LIMIT 20
	`, userID, tenantID)
	if err == nil {
		defer sessRows.Close()
		for sessRows.Next() {
			var sid, ip, ua, lastActive string
			if sessRows.Scan(&sid, &ip, &ua, &lastActive) != nil {
				continue
			}
			sessions = append(sessions, map[string]any{
				"id":            sid,
				"ip_address":    ip,
				"user_agent":    ua,
				"last_activity": lastActive,
			})
		}
	}

	return map[string]any{
		"user":          user,
		"servers":       servers,
		"wallets":       wallets,
		"defaultWallet": defaultWallet,
		"transactions":  transactions,
		"sessions":      sessions,
	}, nil
}

func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	detail, err := h.adminUserDetail(r.Context(), claims.TenantID, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (h *Handler) AdminUserEditForm(w http.ResponseWriter, r *http.Request) {
	h.GetUser(w, r)
}

func (h *Handler) ToggleUserBlock(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	if id == claims.UserID {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"ok":      false,
			"message": "Нельзя заблокировать собственный аккаунт.",
		})
		return
	}

	var role, status string
	err := h.dbOf(r.Context()).QueryRow(r.Context(), `
		SELECT role, status FROM core.users WHERE id = $1 AND tenant_id = $2
	`, id, claims.TenantID).Scan(&role, &status)
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if isStaffRole(role) {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"ok":      false,
			"message": "Нельзя блокировать администраторов.",
		})
		return
	}

	newStatus := "disabled"
	msg := "Пользователь заблокирован."
	if status == "disabled" {
		newStatus = "active"
		msg = "Пользователь разблокирован."
	}

	_, err = h.dbOf(r.Context()).Exec(r.Context(), `
		UPDATE core.users SET status = $3 WHERE id = $1 AND tenant_id = $2
	`, id, claims.TenantID, newStatus)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "message": msg, "status": newStatus})
}

func (h *Handler) adjustWalletBalance(
	ctx context.Context, tenantID, walletID, adminID, userID, currency string, newBalance float64,
) error {
	var current float64
	if err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT balance::float8 FROM core.wallets WHERE id = $1 AND user_id = $2 AND tenant_id = $3
	`, walletID, userID, tenantID).Scan(&current); err != nil {
		return err
	}
	diff := newBalance - current
	if math.Abs(diff) < 0.005 {
		return nil
	}

	tx, err := h.dbOf(ctx).Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if diff > 0 {
		if _, err := tx.Exec(ctx, `
			UPDATE core.wallets SET balance = balance + $2, updated_at = now() WHERE id = $1
		`, walletID, diff); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO core.transactions (tenant_id, wallet_id, type, amount, description, meta, source_type, source_id)
			VALUES ($1, $2, 'credit', $3, $4, $5::jsonb, 'admin_user', $6::uuid)
		`, tenantID, walletID, diff, "Корректировка баланса админом", fmt.Sprintf(
			`{"action":"admin_wallet_balance_set","admin_id":"%s","user_id":"%s","currency":"%s","old_balance":%f,"new_balance":%f}`,
			adminID, userID, currency, current, newBalance,
		), userID); err != nil {
			return err
		}
	} else {
		amount := math.Abs(diff)
		if _, err := tx.Exec(ctx, `
			UPDATE core.wallets SET balance = balance - $2, updated_at = now() WHERE id = $1
		`, walletID, amount); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO core.transactions (tenant_id, wallet_id, type, amount, description, meta, source_type, source_id)
			VALUES ($1, $2, 'debit', $3, $4, $5::jsonb, 'admin_user', $6::uuid)
		`, tenantID, walletID, amount, "Корректировка баланса админом", fmt.Sprintf(
			`{"action":"admin_wallet_balance_set","admin_id":"%s","user_id":"%s","currency":"%s","old_balance":%f,"new_balance":%f}`,
			adminID, userID, currency, current, newBalance,
		), userID); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	ctx := r.Context()

	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	var existingRole string
	if err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT role FROM core.users WHERE id = $1 AND tenant_id = $2
	`, id, claims.TenantID).Scan(&existingRole); err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if existingRole == "owner" && claims.Role != "owner" {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	if role, ok := body["role"].(string); ok && role != "" {
		if role != "user" && role != "admin" && role != "support" {
			writeError(w, http.StatusBadRequest, "invalid role")
			return
		}
		if claims.Role != "owner" && (role == "admin" || existingRole == "admin") {
			writeError(w, http.StatusForbidden, "only owner can change admin roles")
			return
		}
		if role == "admin" && existingRole != "admin" && !h.checkAdminQuota(ctx, claims.TenantID, w) {
			return
		}
		_, _ = h.dbOf(ctx).Exec(ctx, `UPDATE core.users SET role = $3 WHERE id = $1 AND tenant_id = $2`, id, claims.TenantID, role)
	}

	if status, ok := body["status"].(string); ok && status != "" {
		if status == "active" || status == "disabled" {
			_, _ = h.dbOf(ctx).Exec(ctx, `UPDATE core.users SET status = $3 WHERE id = $1 AND tenant_id = $2 AND role != 'owner'`, id, claims.TenantID, status)
		}
	}

	email, _ := body["email"].(string)
	if strings.TrimSpace(email) != "" {
		_, err := h.dbOf(ctx).Exec(ctx, `
			UPDATE core.users SET email = $3 WHERE id = $1 AND tenant_id = $2
		`, id, claims.TenantID, strings.ToLower(strings.TrimSpace(email)))
		if err != nil {
			if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
				writeError(w, http.StatusConflict, "email already registered")
				return
			}
		}
	}

	if pw, ok := body["password"].(string); ok && strings.TrimSpace(pw) != "" {
		confirm, _ := body["password_confirmation"].(string)
		if confirm != "" && confirm != pw {
			writeError(w, http.StatusBadRequest, "password confirmation mismatch")
			return
		}
		if len(pw) < 8 {
			writeError(w, http.StatusBadRequest, "password must be at least 8 characters")
			return
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
		if err == nil {
			_, _ = h.dbOf(ctx).Exec(ctx, `UPDATE core.users SET password_hash = $3 WHERE id = $1 AND tenant_id = $2`, id, claims.TenantID, string(hash))
		}
	}

	name, _ := body["name"].(string)
	lastName, _ := body["last_name"].(string)
	phone, _ := body["phone"].(string)
	publicID, _ := body["public_id"].(string)
	telegramID, _ := body["telegram_id"].(string)
	discordID, _ := body["discord_id"].(string)
	vkID, _ := body["vk_id"].(string)

	contacts := map[string]any{}
	var contactsRaw []byte
	_ = h.dbOf(ctx).QueryRow(ctx, `SELECT contacts FROM core.user_profiles WHERE user_id = $1`, id).Scan(&contactsRaw)
	contacts = parseContacts(contactsRaw)
	if publicID != "" || body["public_id"] != nil {
		contacts["public_id"] = strings.TrimSpace(publicID)
	}
	if telegramID != "" || body["telegram_id"] != nil {
		contacts["telegram_id"] = strings.TrimSpace(telegramID)
	}
	if discordID != "" || body["discord_id"] != nil {
		contacts["discord_id"] = strings.TrimSpace(discordID)
	}
	if vkID != "" || body["vk_id"] != nil {
		contacts["vk_id"] = strings.TrimSpace(vkID)
	}
	contactsJSON, _ := json.Marshal(contacts)

	_, _ = h.dbOf(ctx).Exec(ctx, `
		INSERT INTO core.user_profiles (user_id, display_name, first_name, last_name, phone, contacts, updated_at)
		VALUES ($1, NULLIF($2, ''), NULLIF($2, ''), NULLIF($3, ''), NULLIF($4, ''), $5::jsonb, now())
		ON CONFLICT (user_id) DO UPDATE SET
			display_name = COALESCE(NULLIF(EXCLUDED.display_name, ''), core.user_profiles.display_name),
			first_name = COALESCE(NULLIF(EXCLUDED.first_name, ''), core.user_profiles.first_name),
			last_name = COALESCE(NULLIF(EXCLUDED.last_name, ''), core.user_profiles.last_name),
			phone = COALESCE(NULLIF(EXCLUDED.phone, ''), core.user_profiles.phone),
			contacts = EXCLUDED.contacts,
			updated_at = now()
	`, id, strings.TrimSpace(name), strings.TrimSpace(lastName), strings.TrimSpace(phone), contactsJSON)

	if wallets, ok := body["wallets"].(map[string]any); ok {
		for _, currency := range []string{"RUB", "USD", "EUR"} {
			raw, exists := wallets[currency]
			if !exists {
				continue
			}
			newBal := floatFromAny(raw)
			var walletID string
			err := h.dbOf(ctx).QueryRow(ctx, `
				SELECT id::text FROM core.wallets WHERE user_id = $1 AND tenant_id = $2 AND UPPER(currency) = $3
			`, id, claims.TenantID, currency).Scan(&walletID)
			if err != nil {
				continue
			}
			_ = h.adjustWalletBalance(ctx, claims.TenantID, walletID, claims.UserID, id, currency, newBal)
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "status": "updated"})
}
