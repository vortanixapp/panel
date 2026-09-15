package handlers

import (
	"context"
	"html"
	"strings"

	"github.com/vortanixapp/panel/pkg/i18n"
	"github.com/vortanixapp/panel/pkg/notify"
)

var auditAlertActions = map[string]bool{
	"groups.update":                   true,
	"groups.create":                   true,
	"groups.delete":                   true,
	"user.delete":                     true,
	"refund.request":                  true,
	"user.impersonate":                true,
	"api_key.create":                  true,
	"webhook.create":                  true,
	"payment.refund":                  true,
	"billing.adjust":                  true,
	"ip.block":                        true,
	"server.migrate":                  true,
	"server.delete":                   true,
	"node.bulk.stop":                  true,
	"node.bulk.extend":                true,
	"location.delete":                 true,
	"location.agent_token_regenerate": true,
	"license.bind":                    true,
	"settings.update":                 true,
}

func (h *Handler) auditAlert(ctx context.Context, actorID, actorEmail, action, resource string) {
	if !auditAlertActions[action] {
		return
	}

	who := i18n.Raw(strings.TrimSpace(actorEmail))
	if who.Raw == "" {
		who = i18n.Key("notify.audit.staff")
	}
	params := i18n.Params{
		"action":   i18n.Key("notify.audit.action." + action),
		"who":      who,
		"resource": resource,
	}
	title := i18n.Key("notify.audit.title", params)
	body := i18n.Key("notify.audit.body", params)

	rows, err := h.dbOf(ctx).Query(ctx, `
		SELECT id::text FROM core.users
		WHERE role IN ('admin', 'owner') AND status = 'active'
		  AND ($1 = '' OR id::text <> $1)
	`, actorID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var adminID string
			if rows.Scan(&adminID) == nil {
				h.notifyUser(ctx, adminID, notify.Event{
					Kind:  notify.KindAnnounce,
					Title: title,
					Body:  body,
					Meta:  map[string]any{"action": action, "resource": resource},
				})
			}
		}
	}

	l := i18n.For(ctx, h.dbOf(ctx), "")
	h.telegramAdminAlert(ctx, "<b>"+html.EscapeString(l.Text(title))+"</b>\n"+html.EscapeString(l.Text(body)))
}

func (h *Handler) telegramAdminAlert(ctx context.Context, message string) {
	if !truthySetting(h.tenantSettingString(ctx, "telegram.notifications.enabled")) {
		return
	}
	token := strings.TrimSpace(h.tenantSettingString(ctx, "telegram.notifications.bot_token"))
	chatID := strings.TrimSpace(h.tenantSettingString(ctx, "telegram.notifications.admin_chat_id"))
	if token == "" || chatID == "" {
		return
	}
	go func() {
		_ = h.telegramSendMessage(token, chatID, message)
	}()
}
