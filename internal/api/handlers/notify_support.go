package handlers

import (
	"context"
	"strconv"
	"strings"
	"time"

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

func (h *Handler) notifyStaffTicketNew(ctx context.Context, ticketID, authorID, authorEmail, subject, message string) {
	h.notifyStaff(ctx, notify.StaffSupport, authorID, notify.Event{
		Kind:      notify.KindStaffTicketNew,
		Title:     i18n.Key("notify.staff_ticket_new.title", i18n.Params{"subject": supportSubject(subject)}),
		Body:      i18n.Key("notify.staff_ticket_new.body", i18n.Params{"who": authorEmail}),
		Extra:     []i18n.Msg{i18n.Raw(excerpt(message, 300))},
		Action:    h.panelAction("notify.action.ticket", "/admin/support/"+ticketID),
		Meta:      map[string]any{"ticket_id": ticketID},
		DedupeKey: "staff.ticket_new:" + ticketID,
	})
}

func (h *Handler) notifyStaffTicketReply(ctx context.Context, ticketID, authorID, message string) {
	db := h.dbOf(ctx)
	var subject, email string
	if err := db.QueryRow(ctx, `
		SELECT COALESCE(t.subject, ''), COALESCE(u.email, '')
		FROM core.support_tickets t
		LEFT JOIN core.users u ON u.id = t.user_id
		WHERE t.id = $1::uuid
	`, ticketID).Scan(&subject, &email); err != nil {
		return
	}

	rows, err := db.Query(ctx, `
		SELECT DISTINCT u.id::text
		FROM core.support_messages m
		JOIN core.users u ON u.id = m.user_id
		WHERE m.ticket_id = $1::uuid AND m.is_staff
		  AND u.status = 'active' AND u.role = ANY($2::text[])
	`, ticketID, notify.StaffSupport)
	if err != nil {
		return
	}
	var participants []string
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil && id != authorID {
			participants = append(participants, id)
		}
	}
	rows.Close()

	event := notify.Event{
		Kind:      notify.KindStaffTicketReply,
		Title:     i18n.Key("notify.staff_ticket_reply.title", i18n.Params{"subject": supportSubject(subject)}),
		Body:      i18n.Key("notify.staff_ticket_reply.body", i18n.Params{"who": email}),
		Extra:     []i18n.Msg{i18n.Raw(excerpt(message, 300))},
		Action:    h.panelAction("notify.action.ticket", "/admin/support/"+ticketID),
		Meta:      map[string]any{"ticket_id": ticketID},
		DedupeKey: "staff.ticket_reply:" + ticketID + ":" + strconv.FormatInt(time.Now().Unix()/600, 10),
	}
	if len(participants) == 0 {
		h.notifyStaff(ctx, notify.StaffSupport, authorID, event)
		return
	}
	for _, id := range participants {
		h.notifyUser(ctx, id, event)
	}
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
