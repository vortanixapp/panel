package handlers

import (
	"context"
	"strings"

	"github.com/vortanixapp/panel/pkg/notify"
)

func (h *Handler) notifySupportReply(ctx context.Context, ticketID, message string) {
	var userID, subject string
	var wants bool
	err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT COALESCE(user_id::text, ''), COALESCE(subject, ''), COALESCE(notify, true)
		FROM core.support_tickets
		WHERE id = $1::uuid
	`, ticketID).Scan(&userID, &subject, &wants)
	if err != nil || userID == "" || !wants {
		return
	}
	if subject == "" {
		subject = "обращение"
	}

	h.notifyUser(ctx, userID, notify.Event{
		Kind:   notify.KindSupportReply,
		Title:  "Ответ по обращению",
		Body:   "По обращению «" + subject + "» пришёл ответ.\n\n" + excerpt(message, 300),
		Action: h.panelAction("Открыть обращение", "/support/"+ticketID),
		Meta:   map[string]any{"ticket_id": ticketID},
	})
}

func (h *Handler) notifySupportStatus(ctx context.Context, ticketID, status string) {
	var userID, subject string
	var wants bool
	err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT COALESCE(user_id::text, ''), COALESCE(subject, ''), COALESCE(notify, true)
		FROM core.support_tickets
		WHERE id = $1::uuid
	`, ticketID).Scan(&userID, &subject, &wants)
	if err != nil || userID == "" || !wants {
		return
	}
	if subject == "" {
		subject = "обращение"
	}
	title := "Обращение закрыто"
	body := "Обращение «" + subject + "» закрыто."
	switch status {
	case "open":
		title = "Обращение открыто заново"
		body = "Обращение «" + subject + "» снова открыто."
	case "pending", "in_progress":
		title = "Обращение в работе"
		body = "Обращение «" + subject + "» взято в работу."
	}

	h.notifyUser(ctx, userID, notify.Event{
		Kind:   notify.KindSupportStatus,
		Title:  title,
		Body:   body,
		Action: h.panelAction("Открыть обращение", "/support/"+ticketID),
		Meta:   map[string]any{"ticket_id": ticketID, "status": status},
	})
}

func excerpt(s string, limit int) string {
	s = strings.TrimSpace(s)
	r := []rune(s)
	if len(r) <= limit {
		return s
	}
	cut := string(r[:limit])
	if i := strings.LastIndexAny(cut, " \n"); i > limit/2 {
		cut = cut[:i]
	}
	return strings.TrimSpace(cut) + "…"
}
