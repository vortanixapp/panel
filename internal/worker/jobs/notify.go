package jobs

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/vortanixapp/panel/pkg/i18n"
	"github.com/vortanixapp/panel/pkg/mailer"
	"github.com/vortanixapp/panel/pkg/notify"
)

const notifyCleanupInterval = 6 * time.Hour

func (r *Runner) notifyConfig(ctx context.Context) notify.Config {
	values := r.tenantSettings(ctx)
	token := strings.TrimSpace(values["telegram.notifications.bot_token"])
	if token == "" {
		token = strings.TrimSpace(r.telegramBotToken)
	}
	return notify.Config{
		Mail:             mailer.FromSettings(r.mail, values),
		Brand:            r.mailBrand(ctx),
		TelegramBotToken: token,
	}
}

func (r *Runner) notifyUser(ctx context.Context, userID string, e notify.Event) {
	rec, err := notify.LoadRecipient(ctx, r.db, userID)
	if err != nil {
		log.Printf("оповещение %s: не найден получатель %s: %v", e.Kind, userID, err)
		return
	}
	if _, err := notify.Dispatch(ctx, r.db, rec, e); err != nil {
		log.Printf("оповещение %s пользователю %s: %v", e.Kind, userID, err)
	}
}

func (r *Runner) notifyStaff(ctx context.Context, e notify.Event) {
	if _, err := notify.DispatchStaff(ctx, r.db, notify.StaffAdmins, "", e); err != nil {
		log.Printf("оповещение %s персоналу: %v", e.Kind, err)
	}
}

func (r *Runner) serverAction(labelKey, serverID, suffix string) *notify.Action {
	if r.panelURL == "" {
		return nil
	}
	return &notify.Action{Label: i18n.Key(labelKey), Href: r.panelURL + "/servers/" + serverID + suffix}
}

func (r *Runner) panelAction(labelKey, path string) *notify.Action {
	if r.panelURL == "" {
		return nil
	}
	return &notify.Action{Label: i18n.Key(labelKey), Href: r.panelURL + path}
}

func (r *Runner) NotifyDeliveryLoop(ctx context.Context) {
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()
	var cleaned time.Time
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.heartbeat.Beat(LoopNotifyDelivery)
			if r.deliveriesDue(ctx) {
				sent, failed, err := notify.Drain(ctx, r.db, r.notifyConfig(ctx), 50)
				if err != nil {
					log.Printf("доставка оповещений: %v", err)
				}
				if sent > 0 || failed > 0 {
					log.Printf("доставка оповещений: отправлено %d, не удалось %d", sent, failed)
				}
			}
			if time.Since(cleaned) >= notifyCleanupInterval {
				cleaned = time.Now()
				r.cleanupNotifications(ctx)
			}
		}
	}
}

func (r *Runner) deliveriesDue(ctx context.Context) bool {
	var due bool
	if err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM core.notification_deliveries
			WHERE status = 'queued' AND next_attempt_at <= now()
		)
	`).Scan(&due); err != nil {
		return true
	}
	return due
}

func (r *Runner) cleanupNotifications(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	removed, err := notify.Cleanup(ctx, r.db)
	if err != nil {
		log.Printf("очистка оповещений: %v", err)
		return
	}
	if removed > 0 {
		log.Printf("очистка оповещений: удалено старых %d", removed)
	}
}
