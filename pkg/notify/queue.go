package notify

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	MaxAttempts       = 5
	mailLogErrorLimit = 500
)

func backoff(attempt int) time.Duration {
	const max = 30 * time.Minute
	d := time.Minute
	for i := 1; i < attempt; i++ {
		d *= 2
		if d >= max {
			return max
		}
	}
	return d
}

func Drain(ctx context.Context, pool *pgxpool.Pool, cfg Config, limit int) (sent, failed int, err error) {
	if limit <= 0 {
		limit = 50
	}
	for i := 0; i < limit; i++ {
		ok, delivered, deliveredErr := drainOne(ctx, pool, cfg)
		if deliveredErr != nil {
			return sent, failed, deliveredErr
		}
		if !ok {
			break
		}
		if delivered {
			sent++
		} else {
			failed++
		}
	}
	return sent, failed, nil
}

func drainOne(ctx context.Context, pool *pgxpool.Pool, cfg Config) (found, delivered bool, err error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return false, false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var d Delivery
	var attempts int
	var userID string
	err = tx.QueryRow(ctx, `
		SELECT id::text, kind, channel, target, subject, body, action_label, action_href, attempts,
			COALESCE(user_id::text, '')
		FROM core.notification_deliveries
		WHERE status = 'queued' AND next_attempt_at <= now()
		ORDER BY next_attempt_at, id
		FOR UPDATE SKIP LOCKED
		LIMIT 1
	`).Scan(&d.ID, &d.Kind, &d.Channel, &d.Target, &d.Subject, &d.Body,
		&d.ActionLabel, &d.ActionHref, &attempts, &userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, false, nil
	}
	if err != nil {
		return false, false, err
	}

	sendErr := Send(ctx, cfg, d)
	attempts++
	terminal := true

	switch {
	case sendErr == nil:
		_, err = tx.Exec(ctx, `
			UPDATE core.notification_deliveries
			SET status = 'sent', attempts = $2, sent_at = now(), last_error = ''
			WHERE id = $1::uuid
		`, d.ID, attempts)

	case errors.Is(sendErr, ErrChannelUnavailable):
		_, err = tx.Exec(ctx, `
			UPDATE core.notification_deliveries
			SET status = 'failed', attempts = $2, last_error = $3
			WHERE id = $1::uuid
		`, d.ID, attempts, sendErr.Error())

	case attempts >= MaxAttempts:
		_, err = tx.Exec(ctx, `
			UPDATE core.notification_deliveries
			SET status = 'failed', attempts = $2, last_error = $3
			WHERE id = $1::uuid
		`, d.ID, attempts, sendErr.Error())

	default:
		terminal = false
		_, err = tx.Exec(ctx, `
			UPDATE core.notification_deliveries
			SET attempts = $2, last_error = $3, next_attempt_at = now() + $4::interval
			WHERE id = $1::uuid
		`, d.ID, attempts, sendErr.Error(), backoff(attempts).String())
	}
	if err != nil {
		return false, false, err
	}
	if d.Channel == ChannelEmail && terminal {
		logMail(ctx, tx, cfg, userID, d, sendErr)
	}
	return true, sendErr == nil, tx.Commit(ctx)
}

func logMail(ctx context.Context, db DB, cfg Config, userID string, d Delivery, sendErr error) {
	status, failure := "sent", ""
	if sendErr != nil {
		status, failure = "failed", sendErr.Error()
		if len([]rune(failure)) > mailLogErrorLimit {
			failure = string([]rune(failure)[:mailLogErrorLimit])
		}
	}
	_, _ = db.Exec(ctx, `
		INSERT INTO core.mail_log (template, to_address, subject, status, error, mailer, user_id)
		VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, '')::uuid)
	`, "notify:"+string(d.Kind), d.Target, d.Subject, status, failure, cfg.Mail.MailerName(), userID)
}

func Retry(ctx context.Context, db DB, deliveryID string) error {
	_, err := db.Exec(ctx, `
		UPDATE core.notification_deliveries
		SET status = 'queued', attempts = 0, last_error = '', next_attempt_at = now()
		WHERE id = $1::uuid AND status = 'failed'
	`, deliveryID)
	return err
}
