package notify

import (
	"context"

	"github.com/vortanixapp/panel/pkg/settingsreg"
)

const cleanupBatch = 5000

func Cleanup(ctx context.Context, db DB) (int64, error) {
	var removed int64
	for {
		tag, err := db.Exec(ctx, `
			DELETE FROM core.notifications
			WHERE id IN (
				SELECT id FROM core.notifications
				WHERE created_at < now() - make_interval(days => $2::int)
				   OR (read_at IS NOT NULL AND created_at < now() - make_interval(days => $3::int))
				LIMIT $1
			)
		`, cleanupBatch, int(settingsreg.RetentionNotifications.Int()), int(settingsreg.RetentionNotificationsRead.Int()))
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
				WHERE status <> 'queued' AND created_at < now() - make_interval(days => $2::int)
				LIMIT $1
			)
		`, cleanupBatch, int(settingsreg.RetentionDeliveries.Int()))
		if err != nil {
			return removed, err
		}
		if tag.RowsAffected() < cleanupBatch {
			break
		}
	}
	for {
		tag, err := db.Exec(ctx, `
			DELETE FROM core.mail_log
			WHERE id IN (
				SELECT id FROM core.mail_log
				WHERE created_at < now() - make_interval(days => $2::int)
				LIMIT $1
			)
		`, cleanupBatch, int(settingsreg.RetentionMailLog.Int()))
		if err != nil {
			return removed, err
		}
		if tag.RowsAffected() < cleanupBatch {
			break
		}
	}
	return removed, nil
}
