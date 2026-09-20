package handlers

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/vortanixapp/panel/pkg/i18n"
	"github.com/vortanixapp/panel/pkg/notify"
)

func (h *Handler) notifyConfig(ctx context.Context, r *http.Request) notify.Config {
	return notify.Config{
		Mail:             h.mailerConfig(ctx),
		Brand:            h.mailBrand(ctx, r),
		TelegramBotToken: h.notifyTelegramToken(ctx),
	}
}

func (h *Handler) notifyTelegramToken(ctx context.Context) string {
	if token := strings.TrimSpace(h.tenantSettingString(ctx, "telegram.notifications.bot_token")); token != "" {
		return token
	}
	return strings.TrimSpace(h.telegramBotToken)
}

func (h *Handler) notifyBotUsername(ctx context.Context, token string) string {
	if token == "" {
		return ""
	}
	h.notifyBotMu.Lock()
	defer h.notifyBotMu.Unlock()
	if h.notifyBotToken == token && time.Now().Before(h.notifyBotUntil) {
		return h.notifyBotName
	}
	lookup, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	name, err := notify.TelegramBotUsername(lookup, &http.Client{Timeout: 10 * time.Second}, token)
	if err != nil || name == "" {
		name = strings.TrimPrefix(strings.TrimSpace(h.tenantSettingString(ctx, "telegram.notifications.bot_username")), "@")
		h.notifyBotToken, h.notifyBotName, h.notifyBotUntil = token, name, time.Now().Add(5*time.Minute)
		return name
	}
	h.notifyBotToken, h.notifyBotName, h.notifyBotUntil = token, name, time.Now().Add(time.Hour)
	return name
}

func (h *Handler) notifyUser(ctx context.Context, userID string, e notify.Event) {
	db := h.dbOf(ctx)
	r, err := notify.LoadRecipient(ctx, db, userID)
	if err != nil {
		log.Printf("оповещение %s: не найден получатель %s: %v", e.Kind, userID, err)
		return
	}
	if _, err := notify.Dispatch(ctx, db, r, e); err != nil {
		log.Printf("оповещение %s пользователю %s: %v", e.Kind, userID, err)
	}
}

func (h *Handler) notifyServerOwner(ctx context.Context, serverID string, e notify.Event) {
	db := h.dbOf(ctx)
	r, err := notify.LoadServerOwner(ctx, db, serverID)
	if err != nil {
		log.Printf("оповещение %s: не найден владелец сервера %s: %v", e.Kind, serverID, err)
		return
	}
	if _, err := notify.Dispatch(ctx, db, r, e); err != nil {
		log.Printf("оповещение %s по серверу %s: %v", e.Kind, serverID, err)
	}
}

func (h *Handler) notifyStaff(ctx context.Context, roles []string, exclude string, e notify.Event) {
	if _, err := notify.DispatchStaff(ctx, h.dbOf(ctx), roles, exclude, e); err != nil {
		log.Printf("оповещение %s персоналу: %v", e.Kind, err)
	}
}

func (h *Handler) serverAction(labelKey, serverID, suffix string) *notify.Action {
	base := h.frontendURL
	if base == "" {
		return nil
	}
	return &notify.Action{Label: i18n.Key(labelKey), Href: base + "/servers/" + serverID + suffix}
}

func (h *Handler) panelAction(labelKey, path string) *notify.Action {
	if h.frontendURL == "" {
		return nil
	}
	return &notify.Action{Label: i18n.Key(labelKey), Href: h.frontendURL + path}
}
