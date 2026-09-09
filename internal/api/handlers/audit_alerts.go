package handlers

import (
	"context"
	"fmt"
	"github.com/vortanixapp/panel/pkg/notify"
	"strings"
)

var auditAlertActions = map[string]string{
	"groups.update":                   "изменены права ролей",
	"user.impersonate":                "вход под другим пользователем",
	"api_key.create":                  "выпущен ключ API",
	"webhook.create":                  "добавлен исходящий вебхук",
	"payment.refund":                  "возврат платежа",
	"billing.adjust":                  "ручная корректировка баланса",
	"ip.block":                        "блокировка IP",
	"server.migrate":                  "перенос сервера между нодами",
	"server.delete":                   "удалён сервер",
	"node.bulk.stop":                  "массовая остановка серверов локации",
	"node.bulk.extend":                "массовое продление аренды",
	"location.delete":                 "удалена локация",
	"location.agent_token_regenerate": "перевыпущен токен агента ноды",
	"license.bind":                    "привязана лицензия",
	"settings.update":                 "изменены настройки панели",
}

func (h *Handler) auditAlert(ctx context.Context, tenantID, actorID, actorEmail, action, resource string) {
	label, watched := auditAlertActions[action]
	if !watched {
		return
	}

	who := strings.TrimSpace(actorEmail)
	if who == "" {
		who = "сотрудник"
	}
	title := "Действие в панели: " + label
	body := fmt.Sprintf("%s — %s (%s)", who, label, resource)

	rows, err := h.dbOf(ctx).Query(ctx, `
		SELECT id::text FROM core.users
		WHERE tenant_id = $1 AND role IN ('admin', 'owner') AND status = 'active'
		  AND ($2 = '' OR id::text <> $2)
	`, tenantID, actorID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var adminID string
			if rows.Scan(&adminID) == nil {
				h.notifyUser(ctx, tenantID, adminID, notify.Event{
					Kind:  notify.KindAnnounce,
					Title: title,
					Body:  body,
					Meta:  map[string]any{"action": action, "resource": resource},
				})
			}
		}
	}

	h.telegramAdminAlert(ctx, tenantID, "<b>"+title+"</b>\n"+body)
}

func (h *Handler) telegramAdminAlert(ctx context.Context, tenantID, message string) {
	if !truthySetting(h.tenantSettingString(ctx, tenantID, "telegram.notifications.enabled")) {
		return
	}
	token := strings.TrimSpace(h.tenantSettingString(ctx, tenantID, "telegram.notifications.bot_token"))
	chatID := strings.TrimSpace(h.tenantSettingString(ctx, tenantID, "telegram.notifications.admin_chat_id"))
	if token == "" || chatID == "" {
		return
	}
	go func() {
		_ = h.telegramSendMessage(token, chatID, message)
	}()
}
