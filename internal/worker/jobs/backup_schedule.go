package jobs

import (
	"context"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/vortanix/vortanix/internal/worker/relay"
)

const backupScheduleInterval = 5 * time.Minute

func (r *Runner) BackupScheduleLoop(ctx context.Context) {
	ticker := time.NewTicker(backupScheduleInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.heartbeat.Beat(LoopBackupSchedule)
			r.runDueBackupSchedules(ctx)
		}
	}
}

type dueSchedule struct {
	serverID  string
	tenantID  string
	nodeID    string
	keepCount int
}

func (r *Runner) runDueBackupSchedules(ctx context.Context) {
	rows, err := r.db.Query(ctx, `
		UPDATE core.server_backup_schedules s
		SET last_run_at = now(), updated_at = now()
		WHERE s.server_id IN (
			SELECT sch.server_id
			FROM core.server_backup_schedules sch
			JOIN core.servers srv ON srv.id = sch.server_id
			WHERE sch.enabled = true
			  AND EXTRACT(HOUR FROM now() AT TIME ZONE 'UTC') = sch.hour_utc
			  AND (sch.frequency = 'daily'
			       OR EXTRACT(DOW FROM now() AT TIME ZONE 'UTC') = sch.day_of_week)
			  AND (sch.last_run_at IS NULL
			       OR (sch.frequency = 'daily'
			           AND sch.last_run_at < date_trunc('day', now()))
			       OR (sch.frequency = 'weekly'
			           AND sch.last_run_at < now() - interval '6 days'))
			  -- Просроченный или заблокированный сервер бэкапить незачем:
			  -- контейнер всё равно остановлен.
			  AND srv.suspended_at IS NULL
			  AND COALESCE(srv.is_blocked, false) = false
			ORDER BY sch.last_run_at NULLS FIRST
			LIMIT 50
			FOR UPDATE OF sch SKIP LOCKED
		)
		RETURNING s.server_id::text, s.tenant_id::text, s.keep_count
	`)
	if err != nil {
		log.Printf("backup schedule: query: %v", err)
		return
	}
	var due []dueSchedule
	for rows.Next() {
		var d dueSchedule
		if rows.Scan(&d.serverID, &d.tenantID, &d.keepCount) == nil {
			due = append(due, d)
		}
	}
	rows.Close()

	for _, d := range due {
		if err := r.db.QueryRow(ctx, `
			SELECT node_id::text FROM core.servers WHERE id = $1 AND tenant_id = $2
		`, d.serverID, d.tenantID).Scan(&d.nodeID); err != nil {
			continue
		}
		r.runScheduledBackup(ctx, d)
	}
}

func (r *Runner) runScheduledBackup(ctx context.Context, d dueSchedule) {
	name := "auto-" + time.Now().UTC().Format("2006-01-02-1504")
	resp, err := r.relay.CommandSync(ctx, d.nodeID, relay.CommandRequest{
		CommandID: uuid.NewString(),
		Action:    "backup_create",
		ServerID:  d.serverID,
		Payload:   map[string]any{"name": name},
	})
	if err != nil || resp == nil || !resp.OK {
		msg := "backup failed"
		if err != nil {
			msg = err.Error()
		} else if resp != nil && resp.Result != nil {
			if e, _ := resp.Result["error"].(string); e != "" {
				msg = e
			}
		}
		_, _ = r.db.Exec(ctx, `
			UPDATE core.server_backup_schedules SET last_error = $2, updated_at = now()
			WHERE server_id = $1
		`, d.serverID, msg)
		log.Printf("backup schedule: server %s: %s", d.serverID, msg)
		return
	}

	_, _ = r.db.Exec(ctx, `
		UPDATE core.server_backup_schedules SET last_error = NULL, updated_at = now()
		WHERE server_id = $1
	`, d.serverID)
	log.Printf("backup schedule: server %s: created %s", d.serverID, name)

	if r.remoteBackupsEnabled(ctx, d.tenantID) {
		var backupID string
		size := int64(0)
		if resp.Result != nil {
			if v, ok := resp.Result["size_bytes"].(float64); ok {
				size = int64(v)
			}
		}
		_ = r.db.QueryRow(ctx, `
			INSERT INTO core.server_backups (server_id, tenant_id, status, filename, size_bytes, completed_at, source)
			VALUES ($1, $2, 'completed', $3, $4, now(), 'schedule')
			RETURNING id::text
		`, d.serverID, d.tenantID, name, size).Scan(&backupID)
		r.EnqueueBackupUpload(ctx, d.tenantID, d.serverID, backupID, name, d.keepCount)
	}

	r.pruneAutoBackups(ctx, d)
}

// backupsDir — каталог копий в терминах файлового API агента: пути там
// отсчитываются от корня данных сервера, который сам и есть /data. С полным
// путём "/data/backups" перечисление уходило в /data/data/backups, не находило
// ничего, и старые автокопии не удалялись никогда — сколько бы ни стояло в
// «Хранить копий». Диск сервера при этом рос до упора в квоту тарифа.
const backupsDir = "/backups"

func (r *Runner) pruneAutoBackups(ctx context.Context, d dueSchedule) {
	resp, err := r.relay.CommandSync(ctx, d.nodeID, relay.CommandRequest{
		CommandID: uuid.NewString(),
		Action:    "files_list",
		ServerID:  d.serverID,
		Payload:   map[string]any{"path": backupsDir},
	})
	if err != nil || resp == nil || !resp.OK || resp.Result == nil {
		return
	}
	rawFiles, _ := resp.Result["files"].([]any)
	names := make([]string, 0, len(rawFiles))
	for _, item := range rawFiles {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		name, _ := entry["name"].(string)
		if strings.HasPrefix(name, "auto-") {
			names = append(names, name)
		}
	}
	if len(names) <= d.keepCount {
		return
	}
	sort.Strings(names)
	for _, name := range names[:len(names)-d.keepCount] {
		_, delErr := r.relay.CommandSync(ctx, d.nodeID, relay.CommandRequest{
			CommandID: uuid.NewString(),
			Action:    "files_delete",
			ServerID:  d.serverID,
			Payload:   map[string]any{"path": backupsDir + "/" + name},
		})
		if delErr != nil {
			log.Printf("backup schedule: server %s: prune %s: %v", d.serverID, name, delErr)
			return
		}
	}
}
