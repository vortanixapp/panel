package jobs

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/vortanixapp/panel/internal/worker/relay"

	"github.com/vortanixapp/panel/pkg/notify"
)

func (r *Runner) BackupLoop(ctx context.Context, wake <-chan struct{}) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-wake:
			r.heartbeat.Beat(LoopBackup)
			r.drainBackup(ctx)
		case <-ticker.C:
			r.heartbeat.Beat(LoopBackup)
			r.drainBackup(ctx)
		}
	}
}

func (r *Runner) drainBackup(ctx context.Context) {
	for r.processBackupOne(ctx) {
	}
}

func (r *Runner) processBackupOne(ctx context.Context) bool {
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
		WHERE type = 'backup_server' AND status = 'pending' AND attempts < 5
		ORDER BY created_at ASC
		LIMIT 1
		FOR UPDATE SKIP LOCKED
	`).Scan(&jobID, &tenantID, &payload)
	if err != nil {
		return false
	}

	_, _ = tx.Exec(ctx, `UPDATE core.jobs SET status = 'running', attempts = attempts + 1 WHERE id = $1`, jobID)

	var pl struct {
		ServerID string `json:"server_id"`
		Name     string `json:"name"`
	}
	_ = json.Unmarshal(payload, &pl)
	if pl.ServerID == "" {
		r.failBackupJob(ctx, tx, jobID, "missing server_id in payload")
		_ = tx.Commit(ctx)
		return true
	}

	var nodeID string
	err = tx.QueryRow(ctx, `
		SELECT node_id::text FROM core.servers WHERE id = $1 AND tenant_id = $2
	`, pl.ServerID, tenantID).Scan(&nodeID)
	if err != nil {
		msg := "server not found"
		if err != pgx.ErrNoRows {
			msg = err.Error()
		}
		r.failBackupJob(ctx, tx, jobID, msg)
		_ = tx.Commit(ctx)
		return true
	}

	resp, cmdErr := r.relay.CommandSync(ctx, nodeID, relay.CommandRequest{
		CommandID: uuid.NewString(),
		Action:    "backup_create",
		ServerID:  pl.ServerID,
		Payload:   map[string]any{"name": pl.Name},
	})
	if cmdErr != nil {
		log.Printf("backup job %s: relay error: %v", jobID, cmdErr)
		r.failBackupJob(ctx, tx, jobID, cmdErr.Error())
		_ = tx.Commit(ctx)
		return true
	}
	if resp == nil || !resp.OK {
		msg := "backup failed"
		if resp != nil && resp.Result != nil {
			if e, _ := resp.Result["error"].(string); e != "" {
				msg = e
			}
		}
		r.failBackupJob(ctx, tx, jobID, msg)
		_ = tx.Commit(ctx)
		// Оповещаем после фиксации: иначе запись об оповещении откатилась бы
		// вместе с транзакцией, а клиент бы о сбое не узнал.
		r.notifyBackup(ctx, tenantID, pl.ServerID, false, msg)
		return true
	}

	result, _ := json.Marshal(resp.Result)
	_, _ = tx.Exec(ctx, `
		UPDATE core.jobs SET status = 'completed', result = $2::jsonb WHERE id = $1
	`, jobID, result)
	_ = tx.Commit(ctx)
	r.notifyBackup(ctx, tenantID, pl.ServerID, true, "")
	return true
}

// notifyBackup сообщает владельцу об исходе резервного копирования.
//
// Раньше и успех, и сбой заканчивались одной записью статуса задачи: клиент,
// заказавший копию, не узнавал ни о готовности, ни о том, что копии не будет.
func (r *Runner) notifyBackup(ctx context.Context, tenantID, serverID string, ok bool, msg string) {
	if serverID == "" {
		return
	}
	var name string
	if r.db.QueryRow(ctx, `SELECT name FROM core.servers WHERE id = $1 AND tenant_id = $2`,
		serverID, tenantID).Scan(&name) != nil {
		return
	}
	rec, err := notify.LoadServerOwner(ctx, r.db, tenantID, serverID)
	if err != nil {
		return
	}
	e := notify.Event{
		Kind:   notify.KindBackupReady,
		Title:  "Резервная копия готова",
		Body:   "Копия сервера «" + name + "» создана.",
		Action: r.serverAction("К копиям", serverID, "/copies"),
		Meta:   map[string]any{"server_id": serverID},
	}
	if !ok {
		e.Kind = notify.KindBackupFailed
		e.Title = "Копию создать не удалось"
		e.Body = "Резервную копию сервера «" + name + "» создать не удалось. Причина: " + msg
	}
	if _, err := notify.Dispatch(ctx, r.db, tenantID, rec, e); err != nil {
		log.Printf("оповещение о копии сервера %s: %v", serverID, err)
	}
}

func (r *Runner) failBackupJob(ctx context.Context, tx pgx.Tx, jobID, msg string) {
	result, _ := json.Marshal(map[string]string{"error": msg})
	_, _ = tx.Exec(ctx, `UPDATE core.jobs SET status = 'failed', result = $2::jsonb WHERE id = $1`, jobID, result)
}
