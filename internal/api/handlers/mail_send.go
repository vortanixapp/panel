package handlers

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/vortanixapp/panel/pkg/mailer"
)

const (
	mailSendTimeout = 25 * time.Second
	mailErrorLimit  = 500
)

func (h *Handler) mailerConfig(ctx context.Context) mailer.Config {
	return mailer.FromSettings(h.mail, h.loadTenantSettingStrings(ctx))
}

func (h *Handler) mailConfigured(ctx context.Context) bool {
	return h.mailerConfig(ctx).Configured()
}

func (h *Handler) sendMail(ctx context.Context, template, userID, to string, msg mailer.Message) error {
	cfg := h.mailerConfig(ctx)
	if !cfg.Configured() {
		h.logMail(ctx, cfg, template, userID, to, msg.Subject, "skipped", mailer.ErrNotConfigured.Error())
		return mailer.ErrNotConfigured
	}
	if cfg.Silent() {
		note := "выбран режим " + cfg.MailerName() + ", письмо не отправлялось"
		log.Printf("почта (%s): письмо «%s» для %s не отправлено: %s", template, msg.Subject, to, note)
		h.logMail(ctx, cfg, template, userID, to, msg.Subject, "skipped", note)
		return nil
	}

	sendCtx, cancel := context.WithTimeout(ctx, mailSendTimeout)
	defer cancel()
	err := cfg.Send(sendCtx, msg)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			err = errors.New("почтовый сервер не ответил вовремя")
		}
		log.Printf("почта (%s): письмо на %s не отправлено: %v", template, to, err)
		h.logMail(ctx, cfg, template, userID, to, msg.Subject, "failed", err.Error())
		return err
	}
	h.logMail(ctx, cfg, template, userID, to, msg.Subject, "sent", "")
	return nil
}

func (h *Handler) sendMailAsync(template, userID, to string, msg mailer.Message) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), mailSendTimeout+5*time.Second)
		defer cancel()
		_ = h.sendMail(ctx, template, userID, to, msg)
	}()
}

func (h *Handler) logMail(ctx context.Context, cfg mailer.Config, template, userID, to, subject, status, failure string) {
	if ctx.Err() != nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
	}
	var user any
	if strings.TrimSpace(userID) != "" {
		user = userID
	}
	if len([]rune(failure)) > mailErrorLimit {
		failure = string([]rune(failure)[:mailErrorLimit])
	}
	if _, err := h.dbOf(ctx).Exec(ctx, `
		INSERT INTO core.mail_log (template, to_address, subject, status, error, mailer, user_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, template, to, subject, status, failure, cfg.MailerName(), user); err != nil {
		log.Printf("журнал почты: запись не сохранена: %v", err)
	}
}
