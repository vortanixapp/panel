package handlers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/vortanixapp/panel/pkg/i18n"
	"github.com/vortanixapp/panel/pkg/notify"
)

const (
	telegramLinkTTL     = 10 * time.Minute
	notificationTestGap = 30 * time.Second
)

var telegramChatIDPattern = regexp.MustCompile(`^(-?\d{1,20}|@[A-Za-z][A-Za-z0-9_]{4,31})$`)

type notificationChannels struct {
	Email          bool   `json:"email"`
	Telegram       bool   `json:"telegram"`
	Discord        bool   `json:"discord"`
	TelegramChatID string `json:"telegram_chat_id"`
	DiscordWebhook string `json:"discord_webhook"`
}

type notificationGroupPrefs struct {
	ID       string          `json:"id"`
	Label    string          `json:"label"`
	Channels []string        `json:"channels"`
	Locked   []string        `json:"locked"`
	Routes   map[string]bool `json:"routes"`
}

type telegramLink struct {
	UserID string `json:"user_id"`
}

func telegramLinkKey(code string) string {
	return "notify:tg-link:" + code
}

func (h *Handler) loadNotificationPrefs(ctx context.Context, db notify.DB, userID string) (notificationChannels, notify.Routes) {
	c := notificationChannels{Email: true}
	var raw []byte
	_ = db.QueryRow(ctx, `
		SELECT email_enabled, telegram_enabled, discord_enabled,
		       telegram_chat_id, discord_webhook, routes
		FROM core.user_notification_channels
		WHERE user_id = $1
	`, userID).Scan(&c.Email, &c.Telegram, &c.Discord, &c.TelegramChatID, &c.DiscordWebhook, &raw)
	return c, notify.ParseRoutes(raw)
}

func (h *Handler) notificationPrefsPayload(ctx context.Context, role string, l i18n.Localizer, c notificationChannels, routes notify.Routes) map[string]any {
	groups := []notificationGroupPrefs{}
	for _, g := range notify.GroupsFor(isStaffRole(role)) {
		channels := notify.GroupChannels(g)
		if len(channels) == 0 {
			continue
		}
		item := notificationGroupPrefs{
			ID: string(g), Label: l.Text(notify.GroupLabel(g)),
			Channels: []string{}, Locked: []string{}, Routes: map[string]bool{},
		}
		for _, ch := range channels {
			item.Channels = append(item.Channels, string(ch))
			item.Routes[string(ch)] = routes.Allows(g, ch)
			if notify.GroupLocked(g, ch) {
				item.Locked = append(item.Locked, string(ch))
				item.Routes[string(ch)] = true
			}
		}
		groups = append(groups, item)
	}

	token := h.notifyTelegramToken(ctx)
	return map[string]any{
		"channels": c,
		"groups":   groups,
		"telegram": map[string]any{
			"available":    token != "",
			"bot_username": h.notifyBotUsername(ctx, token),
		},
	}
}

func (h *Handler) NotificationChannelsShow(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ctx := r.Context()
	c, routes := h.loadNotificationPrefs(ctx, h.dbOf(ctx), claims.UserID)
	l := i18n.ForUser(ctx, h.dbOf(ctx), claims.UserID)
	writeJSON(w, http.StatusOK, h.notificationPrefsPayload(ctx, claims.Role, l, c, routes))
}

func (h *Handler) NotificationChannelsUpdate(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ctx := r.Context()
	db := h.dbOf(ctx)
	current, routes := h.loadNotificationPrefs(ctx, db, claims.UserID)

	var body struct {
		Email          *bool                      `json:"email"`
		Telegram       *bool                      `json:"telegram"`
		Discord        *bool                      `json:"discord"`
		TelegramChatID *string                    `json:"telegram_chat_id"`
		DiscordWebhook *string                    `json:"discord_webhook"`
		Routes         map[string]map[string]bool `json:"routes"`
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
		v := strings.TrimSpace(*body.TelegramChatID)
		if v != "" && !telegramChatIDPattern.MatchString(v) {
			writeError(w, http.StatusBadRequest, "invalid_chat_id")
			return
		}
		current.TelegramChatID = v
	}
	if body.DiscordWebhook != nil {
		v := strings.TrimSpace(*body.DiscordWebhook)
		if v != "" && (len(v) > 300 || !notify.IsDiscordWebhook(v)) {
			writeError(w, http.StatusBadRequest, "invalid_webhook")
			return
		}
		current.DiscordWebhook = v
	}
	if (body.Telegram != nil || body.TelegramChatID != nil) && current.Telegram && current.TelegramChatID == "" {
		writeError(w, http.StatusBadRequest, "chat_id_required")
		return
	}
	if (body.Discord != nil || body.DiscordWebhook != nil) && current.Discord && current.DiscordWebhook == "" {
		writeError(w, http.StatusBadRequest, "webhook_required")
		return
	}

	staff := isStaffRole(claims.Role)
	for g, cells := range body.Routes {
		group := notify.Group(g)
		if !notify.KnownGroup(group) || (group == notify.GroupStaff && !staff) {
			continue
		}
		for ch, on := range cells {
			routes = routes.Set(group, notify.Channel(ch), on)
		}
	}

	if _, err := db.Exec(ctx, `
		INSERT INTO core.user_notification_channels
		    (user_id, email_enabled, telegram_enabled, discord_enabled,
		     telegram_chat_id, discord_webhook, routes, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, now())
		ON CONFLICT (user_id) DO UPDATE SET
		    email_enabled    = EXCLUDED.email_enabled,
		    telegram_enabled = EXCLUDED.telegram_enabled,
		    discord_enabled  = EXCLUDED.discord_enabled,
		    telegram_chat_id = EXCLUDED.telegram_chat_id,
		    discord_webhook  = EXCLUDED.discord_webhook,
		    routes           = EXCLUDED.routes,
		    updated_at       = now()
	`, claims.UserID, current.Email, current.Telegram, current.Discord,
		current.TelegramChatID, current.DiscordWebhook, routes.JSON()); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}

	l := i18n.ForUser(ctx, db, claims.UserID)
	writeJSON(w, http.StatusOK, h.notificationPrefsPayload(ctx, claims.Role, l, current, routes))
}

func (h *Handler) panelDisplayName(ctx context.Context, r *http.Request) string {
	if name := strings.TrimSpace(h.mailBrand(ctx, r).Name); name != "" {
		return name
	}
	return "Vortanix"
}

func (h *Handler) NotificationChannelTest(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ctx := r.Context()
	var body struct {
		Channel string `json:"channel"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	channel := notify.Channel(strings.TrimSpace(body.Channel))
	if !notify.KnownChannel(channel) {
		writeError(w, http.StatusBadRequest, "invalid_channel")
		return
	}

	db := h.dbOf(ctx)
	c, _ := h.loadNotificationPrefs(ctx, db, claims.UserID)
	target := ""
	switch channel {
	case notify.ChannelEmail:
		target = strings.TrimSpace(claims.Email)
	case notify.ChannelTelegram:
		target = c.TelegramChatID
	case notify.ChannelDiscord:
		target = c.DiscordWebhook
	}
	if target == "" {
		writeError(w, http.StatusUnprocessableEntity, "channel_not_configured")
		return
	}

	if h.cache != nil {
		key := "notify:test:" + claims.UserID + ":" + string(channel)
		if claimed, err := h.cache.Claim(ctx, key, notificationTestGap); err == nil && !claimed {
			writeError(w, http.StatusTooManyRequests, "too_many_requests")
			return
		}
	}

	l := i18n.ForUser(ctx, db, claims.UserID)
	d := notify.Delivery{
		Kind:    notify.KindAnnounce,
		Channel: channel,
		Target:  target,
		Subject: l.T("notify.channel_test.title"),
		Body:    l.T("notify.channel_test.body", i18n.Params{"app": h.panelDisplayName(ctx, r)}),
	}
	if action := h.panelAction("notify.action.notifications", "/notifications"); action != nil {
		d.ActionLabel = l.Text(action.Label)
		d.ActionHref = action.Href
	}

	sendCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	if err := notify.Send(sendCtx, h.notifyConfig(ctx, r), d); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) NotificationTelegramLink(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ctx := r.Context()
	token := h.notifyTelegramToken(ctx)
	username := h.notifyBotUsername(ctx, token)
	if token == "" || username == "" || h.cache == nil {
		writeError(w, http.StatusUnprocessableEntity, "telegram_unavailable")
		return
	}

	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		writeError(w, http.StatusInternalServerError, "random failure")
		return
	}
	code := base64.RawURLEncoding.EncodeToString(buf)
	if err := h.cache.SetJSON(ctx, telegramLinkKey(code), telegramLink{UserID: claims.UserID}, telegramLinkTTL); err != nil {
		writeError(w, http.StatusInternalServerError, "cache error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"code":       code,
		"url":        "https://t.me/" + username + "?start=" + code,
		"expires_at": time.Now().Add(telegramLinkTTL).UTC().Format(time.RFC3339),
	})
}

func (h *Handler) NotificationTelegramLinkCheck(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ctx := r.Context()
	var body struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	code := strings.TrimSpace(body.Code)
	if code == "" || len(code) > 64 || h.cache == nil {
		writeJSON(w, http.StatusOK, map[string]string{"status": "expired"})
		return
	}

	var link telegramLink
	found, err := h.cache.GetJSON(ctx, telegramLinkKey(code), &link)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "cache error")
		return
	}
	if !found || link.UserID != claims.UserID {
		writeJSON(w, http.StatusOK, map[string]string{"status": "expired"})
		return
	}

	token := h.notifyTelegramToken(ctx)
	if token == "" {
		writeJSON(w, http.StatusOK, map[string]string{"status": "unavailable"})
		return
	}

	lookup, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	chat, matched, err := notify.TelegramFindStart(lookup, &http.Client{Timeout: 15 * time.Second}, token, code, time.Now())
	if errors.Is(err, notify.ErrTelegramWebhook) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "unavailable", "reason": "webhook"})
		return
	}
	if err != nil {
		log.Printf("telegram: проверка подключения оповещений: %v", err)
		writeJSON(w, http.StatusOK, map[string]string{"status": "pending"})
		return
	}
	if !matched {
		writeJSON(w, http.StatusOK, map[string]string{"status": "pending"})
		return
	}

	db := h.dbOf(ctx)
	if _, err := db.Exec(ctx, `
		INSERT INTO core.user_notification_channels (user_id, telegram_enabled, telegram_chat_id, updated_at)
		VALUES ($1, true, $2, now())
		ON CONFLICT (user_id) DO UPDATE SET
		    telegram_enabled = true,
		    telegram_chat_id = EXCLUDED.telegram_chat_id,
		    updated_at       = now()
	`, claims.UserID, chat.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	_ = h.cache.Delete(ctx, telegramLinkKey(code))

	l := i18n.ForUser(ctx, db, claims.UserID)
	if err := notify.Send(ctx, h.notifyConfig(ctx, r), notify.Delivery{
		Kind:    notify.KindAnnounce,
		Channel: notify.ChannelTelegram,
		Target:  chat.ID,
		Subject: l.T("notify.telegram_linked.title"),
		Body:    l.T("notify.telegram_linked.body", i18n.Params{"app": h.panelDisplayName(ctx, r)}),
	}); err != nil {
		log.Printf("telegram: приветствие после подключения не отправлено: %v", err)
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "linked", "chat": chat.Title})
}
