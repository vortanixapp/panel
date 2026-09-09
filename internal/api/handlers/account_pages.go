package handlers

import (
	"context"
	cryptorand "crypto/rand"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vortanixapp/panel/internal/api/payments"
	"github.com/vortanixapp/panel/pkg/notify"
)

var auditCategories = []struct {
	Key      string
	Label    string
	Prefixes []string
}{
	{"server", "Сервер", []string{"server.", "agent.", "node.", "location.", "image."}},
	{"billing", "Биллинг", []string{"billing.", "payment.", "wallet.", "invoice.", "promo.", "bonus."}},
	{"auth", "Доступ", []string{"auth.", "account.", "user.", "session.", "admin."}},
	{"support", "Поддержка", []string{"support.", "ticket."}},
}

func auditCategoryOf(action string) (key, label string) {
	for _, c := range auditCategories {
		for _, p := range c.Prefixes {
			if strings.HasPrefix(action, p) {
				return c.Key, c.Label
			}
		}
	}
	return "other", "Прочее"
}

func auditCategorySQL(key string, argIndex int) (string, []any) {
	for _, c := range auditCategories {
		if c.Key != key {
			continue
		}
		parts := make([]string, 0, len(c.Prefixes))
		args := make([]any, 0, len(c.Prefixes))
		for i, p := range c.Prefixes {
			parts = append(parts, fmt.Sprintf("a.action LIKE $%d", argIndex+i))
			args = append(args, p+"%")
		}
		return "(" + strings.Join(parts, " OR ") + ")", args
	}
	return "", nil
}

type activityRow struct {
	ID          int64          `json:"id"`
	Action      string         `json:"action"`
	Resource    string         `json:"resource"`
	Category    string         `json:"category"`
	CategoryKey string         `json:"category_key"`
	Description string         `json:"description"`
	UserEmail   string         `json:"user_email"`
	IP          string         `json:"ip"`
	Meta        map[string]any `json:"meta,omitempty"`
	CreatedAt   string         `json:"created_at"`
}

type activityFilters struct {
	Query    string
	Category string
	Days     int
	Limit    int
	Offset   int
}

func parseActivityFilters(r *http.Request) activityFilters {
	f := activityFilters{Days: 1, Limit: 100}

	switch r.URL.Query().Get("range") {
	case "7":
		f.Days = 7
	case "30":
		f.Days = 30
	}
	if c := r.URL.Query().Get("category"); c != "" && c != "all" {
		f.Category = c
	}
	f.Query = strings.TrimSpace(r.URL.Query().Get("q"))
	if n, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && n > 0 && n <= 500 {
		f.Limit = n
	}
	if n, err := strconv.Atoi(r.URL.Query().Get("offset")); err == nil && n > 0 {
		f.Offset = n
	}
	return f
}

func (h *Handler) ListActivity(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ctx := r.Context()
	f := parseActivityFilters(r)

	where, args := h.activityWhere(claims.UserID, claims.Role, f)

	var total int
	_ = h.readerOf(ctx).QueryRow(ctx, `
		SELECT COUNT(*) FROM core.audit_logs a
		LEFT JOIN core.users u ON u.id = a.user_id
		WHERE `+where, args...).Scan(&total)

	rows, err := h.readerOf(ctx).Query(ctx, `
		SELECT a.id, a.action, a.resource, COALESCE(u.email, ''), a.meta, a.created_at
		FROM core.audit_logs a
		LEFT JOIN core.users u ON u.id = a.user_id
		WHERE `+where+`
		ORDER BY a.created_at DESC
		LIMIT `+strconv.Itoa(f.Limit)+` OFFSET `+strconv.Itoa(f.Offset),
		args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()

	list := scanActivityRows(rows)

	writeJSON(w, http.StatusOK, map[string]any{
		"activity": list,
		"total":    total,
		"has_more": f.Offset+len(list) < total,
		"stats":    h.activityStats(ctx, claims.UserID, claims.Role),
		"range":    f.Days,
	})
}

func (h *Handler) activityWhere(
	userID, role string, f activityFilters,
) (string, []any) {
	args := []any{}
	where := "true"

	if !isStaffRole(role) {
		args = append(args, userID)
		where += fmt.Sprintf(" AND a.user_id = $%d", len(args))
	}

	args = append(args, strconv.Itoa(f.Days))
	where += fmt.Sprintf(" AND a.created_at >= now() - ($%d::text || ' days')::interval", len(args))

	if f.Category != "" {
		if clause, catArgs := auditCategorySQL(f.Category, len(args)+1); clause != "" {
			where += " AND " + clause
			args = append(args, catArgs...)
		}
	}

	if f.Query != "" {
		args = append(args, "%"+strings.ToLower(f.Query)+"%")
		i := len(args)
		where += fmt.Sprintf(
			" AND (lower(a.action) LIKE $%d OR lower(a.resource) LIKE $%d OR lower(COALESCE(u.email, '')) LIKE $%d)",
			i, i, i)
	}
	return where, args
}

func scanActivityRows(rows pgx.Rows) []activityRow {
	list := make([]activityRow, 0)
	for rows.Next() {
		var e activityRow
		var metaRaw []byte
		var created time.Time
		if rows.Scan(&e.ID, &e.Action, &e.Resource, &e.UserEmail, &metaRaw, &created) != nil {
			continue
		}
		e.CreatedAt = created.UTC().Format(time.RFC3339)
		e.CategoryKey, e.Category = auditCategoryOf(e.Action)

		if len(metaRaw) > 0 {
			meta := map[string]any{}
			if json.Unmarshal(metaRaw, &meta) == nil && len(meta) > 0 {
				e.Meta = meta
				e.IP = metaString(meta, "ip")
				e.Description = metaString(meta, "description")
			}
		}
		if e.Description == "" {
			e.Description = describeAuditAction(e.Action, e.Meta)
		}
		list = append(list, e)
	}
	return list
}

func describeAuditAction(action string, meta map[string]any) string {
	known := map[string]string{
		"server.power":           "Изменено состояние питания сервера",
		"server.reinstall":       "Запущена переустановка сервера",
		"server.renew":           "Продлена аренда сервера",
		"server.create":          "Создан сервер",
		"server.delete":          "Удалён сервер",
		"server.ftp_create":      "Создана FTP-учётная запись",
		"server.ftp_password":    "Изменён пароль FTP",
		"server.ftp_delete":      "Удалена FTP-учётная запись",
		"server.settings_update": "Обновлены игровые настройки",
		"billing.topup":          "Пополнение баланса",
		"billing.invoice_paid":   "Оплачен счёт",
		"bonus.spin":             "Получен ежедневный бонус",
		"auth.login":             "Вход в аккаунт",
		"auth.2fa":               "Двухфакторная аутентификация",
		"auth.2fa_enabled":       "Включена двухфакторная аутентификация",
		"support.ticket_create":  "Создан тикет поддержки",
		"agent.exec":             "Выполнена команда на ноде",
		"location.create":        "Создана локация",
		"location.delete":        "Удалена локация",
	}
	base, ok := known[action]
	if !ok {
		base = strings.ReplaceAll(strings.ReplaceAll(action, ".", " · "), "_", " ")
	}
	for _, key := range []string{"action", "state", "period", "cmd", "prize", "name"} {
		if v := metaString(meta, key); v != "" {
			return base + ": " + v
		}
	}
	return base
}

func (h *Handler) activityStats(ctx context.Context, userID, role string) map[string]any {
	scope := "true"
	args := []any{}
	if !isStaffRole(role) {
		args = append(args, userID)
		scope += " AND a.user_id = $2"
	}

	var events24h, serverActions, logins, errorsWeek int
	_ = h.readerOf(ctx).QueryRow(ctx, `
		SELECT
		    COUNT(*) FILTER (WHERE a.created_at >= now() - INTERVAL '24 hours'),
		    COUNT(*) FILTER (WHERE a.created_at >= now() - INTERVAL '24 hours' AND a.action LIKE 'server.%'),
		    COUNT(*) FILTER (WHERE a.created_at >= now() - INTERVAL '24 hours' AND a.action LIKE 'auth.login%'),
		    COUNT(*) FILTER (WHERE a.created_at >= now() - INTERVAL '7 days' AND (a.action LIKE '%.error' OR a.action LIKE '%.failed'))
		FROM core.audit_logs a
		WHERE `+scope, args...).Scan(&events24h, &serverActions, &logins, &errorsWeek)

	return map[string]any{
		"events_24h":     events24h,
		"server_actions": serverActions,
		"logins":         logins,
		"errors_7d":      errorsWeek,
	}
}

func (h *Handler) ExportActivityCSV(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	f := parseActivityFilters(r)
	f.Limit = 5000
	f.Offset = 0

	where, args := h.activityWhere(claims.UserID, claims.Role, f)
	rows, err := h.readerOf(r.Context()).Query(r.Context(), `
		SELECT a.id, a.action, a.resource, COALESCE(u.email, ''), a.meta, a.created_at
		FROM core.audit_logs a
		LEFT JOIN core.users u ON u.id = a.user_id
		WHERE `+where+`
		ORDER BY a.created_at DESC
		LIMIT 5000`, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()
	list := scanActivityRows(rows)

	filename := fmt.Sprintf("vortanix-activity-%s.csv", time.Now().UTC().Format("2006-01-02"))
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	_, _ = w.Write([]byte{0xEF, 0xBB, 0xBF})

	cw := csv.NewWriter(w)
	defer cw.Flush()
	_ = cw.Write([]string{"Дата", "Действие", "Категория", "Описание", "Ресурс", "Пользователь", "IP"})
	for _, e := range list {
		_ = cw.Write([]string{
			e.CreatedAt, e.Action, e.Category, e.Description, e.Resource, e.UserEmail, e.IP,
		})
	}
}

type bonusPrize struct {
	ID           string  `json:"id"`
	Label        string  `json:"label"`
	Type         string  `json:"type"`
	Value        float64 `json:"value"`
	Weight       int     `json:"weight"`
	Color        string  `json:"color"`
	Icon         string  `json:"icon"`
	Chance       float64 `json:"chance"`
	Duration     int     `json:"duration_hours"`
	DiscountType string  `json:"discount_type"`
}

func promoScopesForPrize(prizeType string) []string {
	switch prizeType {
	case "promo_rent":
		return []string{payments.ApplyRent}
	case "promo_renew":
		return []string{payments.ApplyRenew}
	case "promo_game":
		return []string{payments.ApplyRent, payments.ApplyRenew}
	case "promo_hosting":
		return []string{payments.ApplyHosting}
	default:
		return nil
	}
}

func bonusPromoCode() string {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	b := make([]byte, 8)
	_, _ = cryptorand.Read(b)
	out := make([]byte, len(b))
	for i, v := range b {
		out[i] = alphabet[int(v)%len(alphabet)]
	}
	return "BONUS-" + string(out)
}

const bonusCooldown = 24 * time.Hour

var bonusFallbackColors = []string{"#3a3b3d", "#555658", "#757678", "#9a9b9d", "#c9cacc", "#e8a03c"}

func (h *Handler) loadBonusPrizes(ctx context.Context) []bonusPrize {
	rows, err := h.readerOf(ctx).Query(ctx, `
		SELECT id::text, label, type, value::float8, weight,
		       COALESCE(color, ''), COALESCE(icon, ''), COALESCE(prize_duration_hours, 0),
		       COALESCE(discount_type, '')
		FROM core.daily_bonus_prizes
		WHERE active = true
		ORDER BY sort_order, created_at
	`)
	if err != nil {
		return []bonusPrize{}
	}
	defer rows.Close()

	out := []bonusPrize{}
	totalWeight := 0
	for rows.Next() {
		var p bonusPrize
		if rows.Scan(&p.ID, &p.Label, &p.Type, &p.Value, &p.Weight,
			&p.Color, &p.Icon, &p.Duration, &p.DiscountType) != nil {
			continue
		}
		if p.Weight < 0 {
			p.Weight = 0
		}
		totalWeight += p.Weight
		out = append(out, p)
	}
	for i := range out {
		if out[i].Color == "" {
			out[i].Color = bonusFallbackColors[i%len(bonusFallbackColors)]
		}
		if totalWeight > 0 {
			out[i].Chance = round2(float64(out[i].Weight) / float64(totalWeight) * 100)
		}
	}
	return out
}

func (h *Handler) lastSpinAt(ctx context.Context, userID string) *time.Time {
	var last *time.Time
	_ = h.readerOf(ctx).QueryRow(ctx, `
		SELECT created_at FROM core.daily_bonus_spins
		WHERE user_id = $1
		ORDER BY created_at DESC LIMIT 1
	`, userID).Scan(&last)
	return last
}

func (h *Handler) bonusStreak(ctx context.Context, userID string) (int, map[string]bool) {
	days := map[string]bool{}
	rows, err := h.readerOf(ctx).Query(ctx, `
		SELECT DISTINCT to_char(date_trunc('day', created_at), 'YYYY-MM-DD')
		FROM core.daily_bonus_spins
		WHERE user_id = $1
		  AND created_at >= now() - INTERVAL '60 days'
	`, userID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var d string
			if rows.Scan(&d) == nil {
				days[d] = true
			}
		}
	}

	today := time.Now().UTC().Truncate(24 * time.Hour)
	cursor := today
	if !days[cursor.Format("2006-01-02")] {
		cursor = cursor.AddDate(0, 0, -1)
	}
	streak := 0
	for days[cursor.Format("2006-01-02")] {
		streak++
		cursor = cursor.AddDate(0, 0, -1)
	}
	return streak, days
}

type bonusHistoryItem struct {
	Prize     string  `json:"prize"`
	Type      string  `json:"type"`
	Value     float64 `json:"value"`
	Code      string  `json:"code"`
	CreatedAt string  `json:"created_at"`
}

func (h *Handler) bonusHistory(ctx context.Context, userID string, limit int) []bonusHistoryItem {
	out := []bonusHistoryItem{}
	rows, err := h.readerOf(ctx).Query(ctx, `
		SELECT COALESCE(s.prize_snapshot->>'label', COALESCE(p.label, 'Приз')),
		       s.reward_type, s.reward_value::float8,
		       COALESCE(s.promotion_code, ''), s.created_at
		FROM core.daily_bonus_spins s
		LEFT JOIN core.daily_bonus_prizes p ON p.id = s.prize_id
		WHERE s.user_id = $1
		ORDER BY s.created_at DESC LIMIT $2
	`, userID, limit)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var it bonusHistoryItem
		var created time.Time
		if rows.Scan(&it.Prize, &it.Type, &it.Value, &it.Code, &created) != nil {
			continue
		}
		it.CreatedAt = created.UTC().Format(time.RFC3339)
		out = append(out, it)
	}
	return out
}

func (h *Handler) DailyBonusIndex(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ctx := r.Context()

	prizes := h.loadBonusPrizes(ctx)
	last := h.lastSpinAt(ctx, claims.UserID)
	streak, days := h.bonusStreak(ctx, claims.UserID)

	canSpin := len(prizes) > 0 && (last == nil || time.Since(*last) >= bonusCooldown)
	var nextAt any
	if last != nil && time.Since(*last) < bonusCooldown {
		nextAt = last.Add(bonusCooldown).UTC().Format(time.RFC3339)
	}

	week := make([]map[string]any, 0, 7)
	today := time.Now().UTC().Truncate(24 * time.Hour)
	for i := 6; i >= 0; i-- {
		d := today.AddDate(0, 0, -i)
		key := d.Format("2006-01-02")
		week = append(week, map[string]any{
			"day":     key,
			"weekday": int(d.Weekday()),
			"claimed": days[key],
			"today":   i == 0,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"can_spin":        canSpin,
		"next_spin_at":    nextAt,
		"prizes":          prizes,
		"streak":          streak,
		"streak_days":     week,
		"history":         h.bonusHistory(ctx, claims.UserID, 10),
		"prizes_disabled": len(prizes) == 0,
	})
}

func (h *Handler) DailyBonusSpin(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ctx := r.Context()

	if last := h.lastSpinAt(ctx, claims.UserID); last != nil {
		if wait := bonusCooldown - time.Since(*last); wait > 0 {
			writeJSON(w, http.StatusTooManyRequests, map[string]any{
				"error":        "bonus already claimed",
				"next_spin_at": last.Add(bonusCooldown).UTC().Format(time.RFC3339),
			})
			return
		}
	}

	prizes := h.loadBonusPrizes(ctx)
	prize, ok := pickWeightedPrize(prizes)
	if !ok {
		writeError(w, http.StatusServiceUnavailable, "no active prizes configured")
		return
	}

	credited, promoCode, err := h.grantBonusPrize(ctx, claims.UserID, prize)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to grant prize")
		return
	}

	auditWithIP(ctx, r, h.dbOf(ctx), claims.UserID, "bonus.spin", "daily_bonus",
		map[string]any{"prize": prize.Label, "type": prize.Type, "value": prize.Value,
			"promo_code": promoCode})

	body := fmt.Sprintf("Ваш приз: %s.", prize.Label)
	if promoCode != "" {
		body += fmt.Sprintf("\n\nПромокод: %s", promoCode)
	}
	h.notifyUser(ctx, claims.UserID, notify.Event{
		Kind:   notify.KindBonusGranted,
		Title:  "Ежедневный бонус получен",
		Body:   body,
		Action: h.panelAction("К биллингу", "/billing"),
	})

	streak, _ := h.bonusStreak(ctx, claims.UserID)
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":           true,
		"prize_id":     prize.ID,
		"prize":        prize,
		"credited":     credited,
		"promo_code":   promoCode,
		"streak":       streak,
		"next_spin_at": time.Now().Add(bonusCooldown).UTC().Format(time.RFC3339),
	})
}

func pickWeightedPrize(prizes []bonusPrize) (bonusPrize, bool) {
	total := 0
	for _, p := range prizes {
		total += p.Weight
	}
	if len(prizes) == 0 {
		return bonusPrize{}, false
	}
	if total <= 0 {
		return prizes[rand.Intn(len(prizes))], true
	}
	roll := rand.Intn(total)
	for _, p := range prizes {
		roll -= p.Weight
		if roll < 0 {
			return p, true
		}
	}
	return prizes[len(prizes)-1], true
}

func (h *Handler) grantBonusPrize(
	ctx context.Context, userID string, prize bonusPrize,
) (float64, string, error) {
	tx, err := h.dbOf(ctx).Begin(ctx)
	if err != nil {
		return 0, "", err
	}
	defer tx.Rollback(ctx)

	credited := 0.0
	if prize.Type == "balance" && prize.Value > 0 {
		var walletID string
		err := tx.QueryRow(ctx, `
			SELECT id::text FROM core.wallets
			WHERE user_id = $1
			ORDER BY is_default DESC, created_at LIMIT 1
		`, userID).Scan(&walletID)

		if err != nil {
			if err := tx.QueryRow(ctx, `
				INSERT INTO core.wallets ( user_id, currency, balance, is_default)
				VALUES ( $1, 'RUB', 0, true)
				RETURNING id::text
			`, userID).Scan(&walletID); err != nil {
				return 0, "", err
			}
		}

		if _, err := tx.Exec(ctx, `
			UPDATE core.wallets SET balance = balance + $2, updated_at = now() WHERE id = $1
		`, walletID, prize.Value); err != nil {
			return 0, "", err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO core.transactions
			    ( wallet_id, type, amount, description, source_type)
			VALUES ( $1, 'credit', $2, $3, 'daily_bonus')
		`, walletID, prize.Value, "Ежедневный бонус: "+prize.Label); err != nil {
			return 0, "", err
		}
		credited = prize.Value
	}

	promoID, promoCode := "", ""
	if scopes := promoScopesForPrize(prize.Type); len(scopes) > 0 && prize.Value > 0 {
		discountType := prize.DiscountType
		if discountType != "percent" && discountType != "fixed" {
			discountType = "percent"
		}
		scopesJSON, _ := json.Marshal(scopes)
		filtersJSON, _ := json.Marshal(map[string]any{"user_ids": []string{userID}})
		var endsAt *time.Time
		if prize.Duration > 0 {
			t := time.Now().Add(time.Duration(prize.Duration) * time.Hour)
			endsAt = &t
		}
		for attempt := 0; attempt < 5 && promoID == ""; attempt++ {
			code := bonusPromoCode()
			err := tx.QueryRow(ctx, `
				INSERT INTO core.promotions (
					title, code, active, starts_at, ends_at, applies_to,
					discount_type, discount_value, max_uses, filters, description
				) VALUES ($1, $2, true, now(), $3, $4::jsonb, $5, $6, 1, $7::jsonb, $8)
				ON CONFLICT (code) DO NOTHING
				RETURNING id::text
			`, "Ежедневный бонус: "+prize.Label, code, endsAt, scopesJSON,
				discountType, prize.Value, filtersJSON,
				"Выдан колесом ежедневного бонуса").Scan(&promoID)
			if err == nil {
				promoCode = code
				break
			}
			if !errors.Is(err, pgx.ErrNoRows) {
				return 0, "", err
			}
		}
		if promoID == "" {
			return 0, "", fmt.Errorf("не удалось выдать промокод за приз %s", prize.Label)
		}
	}

	snapshot, _ := json.Marshal(map[string]any{
		"label": prize.Label, "type": prize.Type,
		"value": prize.Value, "duration_hours": prize.Duration,
	})
	if _, err := tx.Exec(ctx, `
		INSERT INTO core.daily_bonus_spins
		    ( user_id, prize_id, prize_snapshot, reward_type, reward_value,
		     promotion_id, promotion_code)
		VALUES ( $1, $2::uuid, $3::jsonb, $4, $5, NULLIF($6, '')::uuid, NULLIF($7, ''))
	`, userID, prize.ID, snapshot, prize.Type, prize.Value,
		promoID, promoCode); err != nil {
		return 0, "", err
	}

	return credited, promoCode, tx.Commit(ctx)
}

type notificationChannels struct {
	Email          bool   `json:"email"`
	Telegram       bool   `json:"telegram"`
	Discord        bool   `json:"discord"`
	TelegramChatID string `json:"telegram_chat_id"`
	DiscordWebhook string `json:"discord_webhook"`
}

func (h *Handler) loadNotificationChannels(ctx context.Context, userID string) notificationChannels {
	c := notificationChannels{Email: true}
	_ = h.readerOf(ctx).QueryRow(ctx, `
		SELECT email_enabled, telegram_enabled, discord_enabled,
		       telegram_chat_id, discord_webhook
		FROM core.user_notification_channels
		WHERE user_id = $1
	`, userID).Scan(&c.Email, &c.Telegram, &c.Discord, &c.TelegramChatID, &c.DiscordWebhook)
	return c
}

func (h *Handler) NotificationChannelsShow(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"channels": h.loadNotificationChannels(r.Context(), claims.UserID),
	})
}

func (h *Handler) NotificationChannelsUpdate(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ctx := r.Context()
	current := h.loadNotificationChannels(ctx, claims.UserID)

	var body struct {
		Email          *bool   `json:"email"`
		Telegram       *bool   `json:"telegram"`
		Discord        *bool   `json:"discord"`
		TelegramChatID *string `json:"telegram_chat_id"`
		DiscordWebhook *string `json:"discord_webhook"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if body.Email != nil {
		current.Email = *body.Email
	}
	if body.Telegram != nil {
		current.Telegram = *body.Telegram
	}
	if body.Discord != nil {
		current.Discord = *body.Discord
	}
	if body.TelegramChatID != nil {
		current.TelegramChatID = clampText(*body.TelegramChatID, 64)
	}
	if body.DiscordWebhook != nil {
		current.DiscordWebhook = clampText(*body.DiscordWebhook, 300)
	}

	if _, err := h.dbOf(ctx).Exec(ctx, `
		INSERT INTO core.user_notification_channels
		    (user_id, email_enabled, telegram_enabled, discord_enabled,
		     telegram_chat_id, discord_webhook, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, now())
		ON CONFLICT (user_id) DO UPDATE SET
		    email_enabled    = EXCLUDED.email_enabled,
		    telegram_enabled = EXCLUDED.telegram_enabled,
		    discord_enabled  = EXCLUDED.discord_enabled,
		    telegram_chat_id = EXCLUDED.telegram_chat_id,
		    discord_webhook  = EXCLUDED.discord_webhook,
		    updated_at       = now()
	`, claims.UserID, current.Email, current.Telegram, current.Discord,
		current.TelegramChatID, current.DiscordWebhook); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "channels": current})
}

func auditWithIP(
	ctx context.Context, r *http.Request, db *pgxpool.Pool,
	userID, action, resource string, meta map[string]any,
) {
	if meta == nil {
		meta = map[string]any{}
	}
	if ip := clientIP(r); ip != "" {
		meta["ip"] = ip
	}
	metaJSON, _ := json.Marshal(meta)
	_, _ = db.Exec(ctx, `
		INSERT INTO core.audit_logs ( user_id, action, resource, meta)
		VALUES ( $1, $2, $3, $4::jsonb)
	`, nullableUUID(userID), action, resource, metaJSON)
}
