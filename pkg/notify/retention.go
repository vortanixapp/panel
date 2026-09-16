package notify

import "context"

const cleanupBatch = 5000

func Cleanup(ctx context.Context, db DB) (int64, error) {
	var removed int64
	for {
		tag, err := db.Exec(ctx, `
			DELETE FROM core.notifications
			WHERE id IN (
				SELECT id FROM core.notifications
				WHERE created_at < now() - interval '180 days'
				   OR (read_at IS NOT NULL AND created_at < now() - interval '90 days')
				LIMIT $1
			)
		`, cleanupBatch)
		if err != nil {
			return removed, err
		}
		removed += tag.RowsAffected()
		if tag.RowsAffected() < cleanupBatch {
			break
		}
	}
	for {
		tag, err := db.Exec(ctx, `
			DELETE FROM core.notification_deliveries
			WHERE id IN (
				SELECT id FROM core.notification_deliveries
				WHERE status <> 'queued' AND created_at < now() - interval '30 days'
				LIMIT $1
			)
		`, cleanupBatch)
		if err != nil {
			return removed, err
		}
		if tag.RowsAffected() < cleanupBatch {
			break
		}
	}
	return removed, nil
}
