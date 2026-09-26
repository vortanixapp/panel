package notify

import (
	"context"
	"strings"

	"github.com/vortanixapp/panel/pkg/i18n"
)

const (
	SettingTelegramEnabled   = "telegram.notifications.enabled"
	SettingTelegramAdminChat = "telegram.notifications.admin_chat_id"
)

func AdminChat(ctx context.Context, db DB) string {
	var enabled, chat string
	err := db.QueryRow(ctx, `
		SELECT COALESCE((SELECT value #>> '{}' FROM core.tenant_settings WHERE key = $1), ''),
		       COALESCE((SELECT value #>> '{}' FROM core.tenant_settings WHERE key = $2), '')
	`, SettingTelegramEnabled, SettingTelegramAdminChat).Scan(&enabled, &chat)
	if err != nil || !truthySetting(enabled) {
		return ""
	}
	return strings.TrimSpace(chat)
}

func truthySetting(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

func DispatchAdminChat(ctx context.Context, db DB, e Event) error {
	target := AdminChat(ctx, db)
	if target == "" {
		return nil
	}

	l := i18n.For(ctx, db, "")
	title := strings.TrimSpace(l.Text(e.Title))
	if title == "" {
		return nil
	}
	body := l.Paragraphs(append([]i18n.Msg{e.Body}, e.Extra...)...)
	label, href := "", ""
	if e.Action != nil {
		label = strings.TrimSpace(l.Text(e.Action.Label))
		href = strings.TrimSpace(e.Action.Href)
	}
	if label == "" || href == "" {
		label, href = "", ""
	}

	subject, text := render(title, body)
	_, err := db.Exec(ctx, `
		INSERT INTO core.notification_deliveries
			( kind, channel, target, subject, body, action_label, action_href)
		VALUES ( $1, $2, $3, $4, $5, $6, $7)
	`, string(e.Kind), string(ChannelTelegram), target, subject, text, label, href)
	return err
}
