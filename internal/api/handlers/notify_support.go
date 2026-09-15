package handlers

import (
	"context"
	"strings"

	"github.com/vortanixapp/panel/pkg/i18n"
	"github.com/vortanixapp/panel/pkg/notify"
)

func supportSubject(subject string) i18n.Msg {
	if strings.TrimSpace(subject) == "" {
		return i18n.Key("notify.support.untitled")
	}
	return i18n.Raw(subject)
}

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

	h.notifyUser(ctx, userID, notify.Event{
		Kind:   notify.KindSupportReply,
		Title:  i18n.Key("notify.support_reply.title"),
		Body:   i18n.Key("notify.support_reply.body", i18n.Params{"subject": supportSubject(subject)}),
		Extra:  []i18n.Msg{i18n.Raw(excerpt(message, 300))},
		Action: h.panelAction("notify.action.ticket", "/support/"+ticketID),
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
	key := "notify.support_closed"
	switch status {
	case "open":
		key = "notify.support_reopened"
	case "pending", "in_progress":
		key = "notify.support_in_progress"
	}

	h.notifyUser(ctx, userID, notify.Event{
		Kind:   notify.KindSupportStatus,
		Title:  i18n.Key(key + ".title"),
		Body:   i18n.Key(key+".body", i18n.Params{"subject": supportSubject(subject)}),
		Action: h.panelAction("notify.action.ticket", "/support/"+ticketID),
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
