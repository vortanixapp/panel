package notify

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// MaxAttempts — сколько раз пробовать доставить, прежде чем сдаться.
const MaxAttempts = 5

// backoff — пауза перед следующей попыткой.
//
// Растёт вдвое: 1, 2, 4, 8 минут. Прежняя очередь писем биллинга паузы не имела
// вовсе — упавшая строка оставалась в очереди и её брали в ближайшем же проходе,
// так что все пять попыток сгорали за минуты и упирались в тот же самый сбой.
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

// Drain отправляет готовые записи очереди. Возвращает число отправленных и
// число окончательно провалившихся.
//
// Забирает строки через FOR UPDATE SKIP LOCKED. Это единственный способ
// безопасно запускать разбор из нескольких мест сразу: очередь писем биллинга
// выбирает строки без блокировки, а претендентов на неё пятеро — четыре
// обработчика задач и ежеминутный планировщик, — из-за чего одно письмо может
// уйти адресату дважды.
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
			break // очередь пуста
		}
		sent++
	}
	return sent, failed, nil
}

// drainOne обрабатывает одну запись в собственной транзакции.
//
// Транзакция на запись, а не на всю пачку: длинная транзакция держала бы
// блокировки на время сетевых обращений к Telegram и Discord, а они бывают
// медленными.
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
		// Повторять нечего: пока владелец не настроит канал, следующая попытка
		// провалится так же. Помечаем сразу и сохраняем причину — чтобы в
		// админке было видно, что доставки нет и почему, а не тишина.
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

// Retry возвращает провалившуюся доставку в очередь.
//
// Нужна, потому что после исчерпания попыток строка становится недостижимой:
// выборка берёт только queued. В очереди писем биллинга ровно это и происходит,
// и вернуть письмо в работу нельзя ни из интерфейса, ни запросом.
func Retry(ctx context.Context, db DB, tenantID, deliveryID string) error {
	_, err := db.Exec(ctx, `
		UPDATE core.notification_deliveries
		SET status = 'queued', attempts = 0, last_error = '', next_attempt_at = now()
		WHERE id = $1::uuid AND tenant_id = $2 AND status = 'failed'
	`, deliveryID, tenantID)
	return err
}
