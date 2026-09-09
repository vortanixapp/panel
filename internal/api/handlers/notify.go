package handlers

import (
	"context"
	"log"

	"github.com/vortanixapp/panel/pkg/notify"
)

func (h *Handler) notifyConfig() notify.Config {
	return notify.Config{
		SMTPHost:         h.mail.Host,
		SMTPPort:         h.mail.Port,
		SMTPUser:         h.mail.User,
		SMTPPass:         h.mail.Pass,
		MailFrom:         h.mail.From,
		TelegramBotToken: h.telegramBotToken,
	}
}

func (h *Handler) notifyUser(ctx context.Context, tenantID, userID string, e notify.Event) {
	db := h.dbOf(ctx)
	r, err := notify.LoadRecipient(ctx, db, tenantID, userID)
	if err != nil {
		log.Printf("оповещение %s: не найден получатель %s: %v", e.Kind, userID, err)
		return
	}
	if _, err := notify.Dispatch(ctx, db, tenantID, r, e); err != nil {
		log.Printf("оповещение %s пользователю %s: %v", e.Kind, userID, err)
	}
}

func (h *Handler) notifyServerOwner(ctx context.Context, tenantID, serverID string, e notify.Event) {
	db := h.dbOf(ctx)
	r, err := notify.LoadServerOwner(ctx, db, tenantID, serverID)
	if err != nil {
		log.Printf("оповещение %s: не найден владелец сервера %s: %v", e.Kind, serverID, err)
		return
	}
	if _, err := notify.Dispatch(ctx, db, tenantID, r, e); err != nil {
		log.Printf("оповещение %s по серверу %s: %v", e.Kind, serverID, err)
	}
}

func (h *Handler) serverAction(label, serverID, suffix string) *notify.Action {
	base := h.frontendURL
	if base == "" {
		return nil
	}
	return &notify.Action{Label: label, Href: base + "/servers/" + serverID + suffix}
}

func (h *Handler) panelAction(label, path string) *notify.Action {
	if h.frontendURL == "" {
		return nil
	}
	return &notify.Action{Label: label, Href: h.frontendURL + path}
}
