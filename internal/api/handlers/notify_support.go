package handlers

import (
	"context"
	"strings"

	"github.com/vortanix/vortanix/pkg/notify"
)

// notifySupportReply сообщает клиенту, что оператор ответил.
//
// Раньше об ответе не сообщали ничем: клиент узнавал о нём, только открыв
// обращение сам. При этом флажок «уведомления по обращению» в интерфейсе был,
// колонка notify в базе была, и не читал её никто — выключать было нечего,
// потому что по обращениям вообще ничего не слали.
func (h *Handler) notifySupportReply(ctx context.Context, tenantID, ticketID, message string) {
	var userID, subject string
	var wants bool
	err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT COALESCE(user_id::text, ''), COALESCE(subject, ''), COALESCE(notify, true)
		FROM core.support_tickets
		WHERE id = $1::uuid AND tenant_id = $2
	`, ticketID, tenantID).Scan(&userID, &subject, &wants)
	if err != nil || userID == "" || !wants {
		return
	}
	if subject == "" {
		subject = "обращение"
	}

	h.notifyUser(ctx, tenantID, userID, notify.Event{
		Kind:   notify.KindSupportReply,
		Title:  "Ответ по обращению",
		Body:   "По обращению «" + subject + "» пришёл ответ.\n\n" + excerpt(message, 300),
		Action: h.panelAction("Открыть обращение", "/support/"+ticketID),
		Meta:   map[string]any{"ticket_id": ticketID},
	})
}

// notifySupportStatus сообщает автору, что обращение сменило состояние.
//
// Событие support.status было объявлено в справочнике, но не отправлялось
// ниоткуда: тикет закрывали, а клиент об этом не узнавал.
func (h *Handler) notifySupportStatus(ctx context.Context, tenantID, ticketID, status string) {
	var userID, subject string
	var wants bool
	err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT COALESCE(user_id::text, ''), COALESCE(subject, ''), COALESCE(notify, true)
		FROM core.support_tickets
		WHERE id = $1::uuid AND tenant_id = $2
	`, ticketID, tenantID).Scan(&userID, &subject, &wants)
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

	h.notifyUser(ctx, tenantID, userID, notify.Event{
		Kind:   notify.KindSupportStatus,
		Title:  title,
		Body:   body,
		Action: h.panelAction("Открыть обращение", "/support/"+ticketID),
		Meta:   map[string]any{"ticket_id": ticketID, "status": status},
	})
}

// excerpt обрезает текст по границе слова.
//
// Ответ оператора уходит в письмо и в Telegram целиком не нужен: там важно
// понять, что ответ есть, а читать переписку удобнее в панели.
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
