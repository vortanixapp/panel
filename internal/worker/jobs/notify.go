package jobs

import (
	"context"
	"log"
	"time"

	"github.com/vortanixapp/panel/pkg/notify"
)

func (r *Runner) notifyConfig() notify.Config {
	return notify.Config{
		SMTPHost:         r.mail.Host,
		SMTPPort:         r.mail.Port,
		SMTPUser:         r.mail.User,
		SMTPPass:         r.mail.Pass,
		MailFrom:         r.mail.From,
		TelegramBotToken: r.telegramBotToken,
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

func (r *Runner) serverAction(label, serverID, suffix string) *notify.Action {
	if r.panelURL == "" {
		return nil
	}
	return &notify.Action{Label: label, Href: r.panelURL + "/servers/" + serverID + suffix}
}

func (r *Runner) panelAction(label, path string) *notify.Action {
	if r.panelURL == "" {
		return nil
	}
	return &notify.Action{Label: label, Href: r.panelURL + path}
}

func (r *Runner) NotifyDeliveryLoop(ctx context.Context) {
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.heartbeat.Beat(LoopNotifyDelivery)
			sent, failed, err := notify.Drain(ctx, r.db, r.notifyConfig(), 50)
			if err != nil {
				log.Printf("доставка оповещений: %v", err)
			}
			if sent > 0 || failed > 0 {
				log.Printf("доставка оповещений: отправлено %d, не удалось %d", sent, failed)
			}
		}
	}
}
