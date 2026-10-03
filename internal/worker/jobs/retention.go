package jobs

import (
	"context"
	"log"

	"github.com/vortanixapp/panel/pkg/settingsreg"
)

const historyBatch = 5000

type historyRule struct {
	label   string
	setting *settingsreg.Setting
	table   string
	column  string
	key     string
}

func historyRules() []historyRule {
	return []historyRule{
		{"попытки входа", settingsreg.RetentionLoginAttempts, "core.login_attempts", "created_at", "id"},
		{"журнал аудита", settingsreg.RetentionAuditLogs, "core.audit_logs", "created_at", "(id, created_at)"},
		{"сессии", settingsreg.RetentionSessions, "core.user_sessions", "last_active", "id"},
		{"ссылки сброса пароля", settingsreg.RetentionResetTokens, "core.password_reset_tokens", "expires_at", "id"},
		{"токены входа WHMCS", settingsreg.RetentionSSOTokens, "core.whmcs_sso_tokens", "expires_at", "token_hash"},
		{"доставки вебхуков", settingsreg.RetentionWebhookDeliveries, "core.webhook_deliveries", "created_at", "id"},
		{"почасовые списания", settingsreg.RetentionHourlyCharges, "core.server_hourly_charges", "created_at", "(server_id, period_start)"},
	}
}

func (r *Runner) cleanupHistory(ctx context.Context) {
	for _, rule := range historyRules() {
		days := int(rule.setting.Int())
		if days <= 0 {
			continue
		}
		var total int64
		for i := 0; i < 100; i++ {
			tag, err := r.db.Exec(ctx, `
				DELETE FROM `+rule.table+`
				WHERE `+rule.key+` IN (
					SELECT `+rule.key+` FROM `+rule.table+`
					WHERE `+rule.column+` < now() - make_interval(days => $1::int)
					LIMIT $2
				)
			`, days, historyBatch)
			if err != nil {
				log.Printf("очистка (%s): %v", rule.label, err)
				break
			}
			total += tag.RowsAffected()
			if tag.RowsAffected() < historyBatch {
				break
			}
		}
		if total > 0 {
			log.Printf("очистка (%s): удалено %d строк", rule.label, total)
		}
	}
}
