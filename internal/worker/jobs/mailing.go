package jobs

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/vortanixapp/panel/pkg/mailer"
	"github.com/vortanixapp/panel/pkg/mailtpl"
)

const (
	mailingSendTimeout = 30 * time.Second
	mailLogErrorLimit  = 500
)

func (r *Runner) MailingLoop(ctx context.Context, wake <-chan struct{}) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-wake:
			r.heartbeat.Beat(LoopMailing)
			r.drainMailing(ctx)
		case <-ticker.C:
			r.heartbeat.Beat(LoopMailing)
			r.drainMailing(ctx)
		}
	}
}

func (r *Runner) drainMailing(ctx context.Context) {
	if r.panelFrozen(ctx) {
		return
	}
	for r.processMailingOne(ctx) {
	}
}

func (r *Runner) processMailingOne(ctx context.Context) bool {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return false
	}
	defer tx.Rollback(ctx)

	var jobID string
	var payload []byte
	err = tx.QueryRow(ctx, `
		SELECT id::text, payload
		FROM core.jobs
		WHERE type = 'send_mailing' AND status = 'pending' AND attempts < 5
		ORDER BY created_at ASC
		LIMIT 1
		FOR UPDATE SKIP LOCKED
	`).Scan(&jobID, &payload)
	if err != nil {
		return false
	}
	_, _ = tx.Exec(ctx, `UPDATE core.jobs SET status = 'running', attempts = attempts + 1 WHERE id = $1`, jobID)

	var pl struct {
		MailingID string `json:"mailing_id"`
	}
	_ = json.Unmarshal(payload, &pl)
	if pl.MailingID == "" {
		r.failMailingJob(ctx, tx, jobID, "missing mailing_id in payload")
		_ = tx.Commit(ctx)
		return true
	}

	var subject, body string
	var isHTML bool
	err = tx.QueryRow(ctx, `
		SELECT COALESCE(subject, ''), COALESCE(body, ''), is_html
		FROM core.mailings WHERE id = $1
	`, pl.MailingID).Scan(&subject, &body, &isHTML)
	if err != nil {
		msg := "mailing not found"
		if err != pgx.ErrNoRows {
			msg = err.Error()
		}
		r.failMailingJob(ctx, tx, jobID, msg)
		_ = tx.Commit(ctx)
		return true
	}

	_, _ = tx.Exec(ctx, `UPDATE core.mailings SET status = 'sending', started_at = now() WHERE id = $1`, pl.MailingID)
	_ = tx.Commit(ctx)

	cfg := r.mailConfig(ctx)
	if !cfg.Enabled() {
		reason := "SMTP не настроен"
		if cfg.Silent() {
			reason = "выбран режим " + cfg.MailerName() + ", письма не отправляются"
		}
		r.failJobDirect(ctx, jobID, reason)
		_, _ = r.db.Exec(ctx, `UPDATE core.mailings SET status = 'failed', finished_at = now() WHERE id = $1`, pl.MailingID)
		return true
	}

	rows, err := r.db.Query(ctx, `
		SELECT id::text, email FROM core.users
		WHERE status = 'active' AND email NOT LIKE '%@telegram.local'
		ORDER BY created_at
	`)
	if err != nil {
		r.failJobDirect(ctx, jobID, err.Error())
		_, _ = r.db.Exec(ctx, `UPDATE core.mailings SET status = 'failed', finished_at = now() WHERE id = $1`, pl.MailingID)
		return true
	}
	type recipient struct{ id, email string }
	var recipients []recipient
	for rows.Next() {
		var rec recipient
		if rows.Scan(&rec.id, &rec.email) == nil && mailer.ValidAddress(rec.email) {
			recipients = append(recipients, rec)
		}
	}
	rows.Close()

	msg := mailingMessage(r.mailBrand(ctx), subject, body, isHTML)
	sent := 0
	for _, rec := range recipients {
		letter := msg
		letter.To = rec.email
		sendCtx, cancel := context.WithTimeout(ctx, mailingSendTimeout)
		sendErr := cfg.Send(sendCtx, letter)
		cancel()
		status, failure := "sent", ""
		if sendErr == nil {
			sent++
			_, _ = r.db.Exec(ctx, `
				INSERT INTO core.mailing_deliveries (mailing_id, user_id, channel, address, status, sent_at)
				VALUES ($1, $2, 'email', $3, 'sent', now())
			`, pl.MailingID, rec.id, rec.email)
		} else {
			status, failure = "failed", sendErr.Error()
			log.Printf("рассылка %s: письмо на %s не отправлено: %v", pl.MailingID, rec.email, sendErr)
			_, _ = r.db.Exec(ctx, `
				INSERT INTO core.mailing_deliveries (mailing_id, user_id, channel, address, status, error)
				VALUES ($1, $2, 'email', $3, 'failed', $4)
			`, pl.MailingID, rec.id, rec.email, failure)
		}
		r.logMail(ctx, cfg, "mailing", rec.id, rec.email, subject, status, failure)
	}

	r.finishMailing(ctx, jobID, pl.MailingID, len(recipients), sent)
	return true
}

func (r *Runner) finishMailing(ctx context.Context, jobID, mailingID string, total, sent int) {
	status := "completed"
	if total > 0 && sent == 0 {
		status = "failed"
	}
	_, _ = r.db.Exec(ctx, `
		UPDATE core.mailings SET status = $2, total_recipients = $3, sent_count = $4, finished_at = now() WHERE id = $1
	`, mailingID, status, total, sent)
	result, _ := json.Marshal(map[string]int{"total": total, "sent": sent})
	_, _ = r.db.Exec(ctx, `UPDATE core.jobs SET status = 'completed', result = $2::jsonb WHERE id = $1`, jobID, result)
}

func (r *Runner) failMailingJob(ctx context.Context, tx pgx.Tx, jobID, msg string) {
	result, _ := json.Marshal(map[string]string{"error": msg})
	_, _ = tx.Exec(ctx, `UPDATE core.jobs SET status = 'failed', result = $2::jsonb WHERE id = $1`, jobID, result)
}

func (r *Runner) failJobDirect(ctx context.Context, jobID, msg string) {
	result, _ := json.Marshal(map[string]string{"error": msg})
	_, _ = r.db.Exec(ctx, `UPDATE core.jobs SET status = 'failed', result = $2::jsonb WHERE id = $1`, jobID, result)
}

func (r *Runner) mailConfig(ctx context.Context) mailer.Config {
	return mailer.FromSettings(r.mail, r.tenantSettings(ctx))
}

func (r *Runner) tenantSettings(ctx context.Context) map[string]string {
	out := map[string]string{}
	rows, err := r.db.Query(ctx, `SELECT key, value FROM core.tenant_settings`)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var key string
		var raw []byte
		if rows.Scan(&key, &raw) != nil {
			continue
		}
		var value string
		if json.Unmarshal(raw, &value) == nil {
			out[key] = r.secrets.MustDecrypt(value)
		}
	}
	return out
}

func (r *Runner) tenantSettingString(ctx context.Context, key string) string {
	var raw []byte
	if r.db.QueryRow(ctx, `SELECT value FROM core.tenant_settings WHERE key = $1`, key).Scan(&raw) != nil {
		return ""
	}
	if len(raw) == 0 {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return r.secrets.MustDecrypt(s)
	}
	return ""
}

func mailingMessage(brand mailtpl.Brand, subject, body string, isHTML bool) mailer.Message {
	content := mailtpl.Message{Body: mailtpl.Paragraphs(body)}
	if isHTML {
		content.Body = body
	}
	out := mailer.Message{
		Subject: subject,
		HTML:    mailtpl.Render(brand, content),
		Text:    mailtpl.PlainText(brand, content),
	}
	if isHTML && strings.Contains(strings.ToLower(body), "<html") {
		out.HTML = body
	}
	return out
}

func (r *Runner) logMail(ctx context.Context, cfg mailer.Config, template, userID, to, subject, status, failure string) {
	if len([]rune(failure)) > mailLogErrorLimit {
		failure = string([]rune(failure)[:mailLogErrorLimit])
	}
	_, _ = r.db.Exec(ctx, `
		INSERT INTO core.mail_log (template, to_address, subject, status, error, mailer, user_id)
		VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, '')::uuid)
	`, template, to, subject, status, failure, cfg.MailerName(), userID)
}
