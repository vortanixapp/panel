package jobs

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/vortanix/vortanix/internal/worker/mail"
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
	for r.processMailingOne(ctx) {
	}
}

func (r *Runner) processMailingOne(ctx context.Context) bool {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return false
	}
	defer tx.Rollback(ctx)

	var jobID, tenantID string
	var payload []byte
	err = tx.QueryRow(ctx, `
		SELECT id::text, tenant_id::text, payload
		FROM core.jobs
		WHERE type = 'send_mailing' AND status = 'pending' AND attempts < 5
		ORDER BY created_at ASC
		LIMIT 1
		FOR UPDATE SKIP LOCKED
	`).Scan(&jobID, &tenantID, &payload)
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
		FROM core.mailings WHERE id = $1 AND tenant_id = $2
	`, pl.MailingID, tenantID).Scan(&subject, &body, &isHTML)
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

	cfg := r.mailConfigForTenant(ctx, tenantID)
	if !cfg.Enabled() {
		r.finishMailing(ctx, jobID, pl.MailingID, tenantID, 0, 0)
		r.failJobDirect(ctx, jobID, "smtp not configured for tenant")
		_, _ = r.db.Exec(ctx, `UPDATE core.mailings SET status = 'failed' WHERE id = $1`, pl.MailingID)
		return true
	}

	rows, err := r.db.Query(ctx, `
		SELECT id::text, email FROM core.users WHERE tenant_id = $1 AND status = 'active'
	`, tenantID)
	if err != nil {
		r.failJobDirect(ctx, jobID, err.Error())
		_, _ = r.db.Exec(ctx, `UPDATE core.mailings SET status = 'failed' WHERE id = $1`, pl.MailingID)
		return true
	}
	type recipient struct{ id, email string }
	var recipients []recipient
	for rows.Next() {
		var rec recipient
		if rows.Scan(&rec.id, &rec.email) == nil {
			recipients = append(recipients, rec)
		}
	}
	rows.Close()

	sent := 0
	for _, rec := range recipients {
		sendErr := cfg.Send(rec.email, subject, mailingBody(body, isHTML))
		if sendErr == nil {
			sent++
			_, _ = r.db.Exec(ctx, `
				INSERT INTO core.mailing_deliveries (tenant_id, mailing_id, user_id, channel, address, status, sent_at)
				VALUES ($1, $2, $3, 'email', $4, 'sent', now())
			`, tenantID, pl.MailingID, rec.id, rec.email)
		} else {
			log.Printf("mailing %s: send to %s failed: %v", pl.MailingID, rec.email, sendErr)
			_, _ = r.db.Exec(ctx, `
				INSERT INTO core.mailing_deliveries (tenant_id, mailing_id, user_id, channel, address, status, error)
				VALUES ($1, $2, $3, 'email', $4, 'failed', $5)
			`, tenantID, pl.MailingID, rec.id, rec.email, sendErr.Error())
		}
	}

	r.finishMailing(ctx, jobID, pl.MailingID, tenantID, len(recipients), sent)
	return true
}

func (r *Runner) finishMailing(ctx context.Context, jobID, mailingID, tenantID string, total, sent int) {
	status := "completed"
	if sent < total {
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

func (r *Runner) mailConfigForTenant(ctx context.Context, tenantID string) mail.Config {
	host := r.tenantSettingString(ctx, tenantID, "mail.mailers.smtp.host")
	port := r.tenantSettingString(ctx, tenantID, "mail.mailers.smtp.port")
	user := r.tenantSettingString(ctx, tenantID, "mail.mailers.smtp.username")
	pass := r.tenantSettingString(ctx, tenantID, "mail.mailers.smtp.password")
	from := r.tenantSettingString(ctx, tenantID, "mail.from.address")
	if host == "" {
		return r.mail
	}
	if port == "" {
		port = "587"
	}
	if from == "" {
		from = r.mail.From
	}
	return mail.Config{Host: host, Port: port, User: user, Pass: pass, From: from}
}

func (r *Runner) tenantSettingString(ctx context.Context, tenantID, key string) string {
	var raw []byte
	if r.db.QueryRow(ctx, `SELECT value FROM core.tenant_settings WHERE tenant_id = $1 AND key = $2`, tenantID, key).Scan(&raw) != nil {
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

func mailingBody(body string, isHTML bool) string {
	if isHTML {
		return body
	}
	return "<pre style=\"font-family:inherit;white-space:pre-wrap\">" + body + "</pre>"
}
