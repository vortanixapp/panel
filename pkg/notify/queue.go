package notify

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const MaxAttempts = 5

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
		ok, deliveredErr := drainOne(ctx, pool, cfg)
		if deliveredErr != nil {
			return sent, failed, deliveredErr
		}
		if !ok {
			break
		}
		sent++
	}
	return sent, failed, nil
}

func drainOne(ctx context.Context, pool *pgxpool.Pool, cfg Config) (bool, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var d Delivery
	var attempts int
	err = tx.QueryRow(ctx, `
		SELECT id::text, kind, channel, target, subject, body, action_label, action_href, attempts
		FROM core.notification_deliveries
		WHERE status = 'queued' AND next_attempt_at <= now()
		ORDER BY next_attempt_at, id
		FOR UPDATE SKIP LOCKED
		LIMIT 1
	`).Scan(&d.ID, &d.Kind, &d.Channel, &d.Target, &d.Subject, &d.Body,
		&d.ActionLabel, &d.ActionHref, &attempts)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	sendErr := Send(ctx, cfg, d)
	attempts++

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
		_, err = tx.Exec(ctx, `
			UPDATE core.notification_deliveries
			SET attempts = $2, last_error = $3, next_attempt_at = now() + $4::interval
			WHERE id = $1::uuid
		`, d.ID, attempts, sendErr.Error(), backoff(attempts).String())
	}
	if err != nil {
		return false, err
	}
	return true, tx.Commit(ctx)
}

func Retry(ctx context.Context, db DB, deliveryID string) error {
	_, err := db.Exec(ctx, `
		UPDATE core.notification_deliveries
		SET status = 'queued', attempts = 0, last_error = '', next_attempt_at = now()
		WHERE id = $1::uuid AND status = 'failed'
	`, deliveryID)
	return err
}
