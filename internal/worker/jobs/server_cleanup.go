package jobs

import (
	"context"
	"encoding/json"
	"log"

	"github.com/vortanixapp/panel/internal/worker/relay"
)

func (r *Runner) processServerDestroys(ctx context.Context) {
	if _, err := r.db.Exec(ctx, `
		UPDATE core.jobs j
		SET status = 'cancelled', result = '{"error":"Локация удалена"}'::jsonb
		WHERE j.type = 'server_destroy' AND j.status = 'pending'
		  AND NOT EXISTS (SELECT 1 FROM core.nodes n WHERE n.id::text = j.payload->>'node_id')
	`); err != nil {
		log.Printf("очистка удалённых серверов: %v", err)
		return
	}

	rows, err := r.db.Query(ctx, `
		UPDATE core.jobs SET status = 'running'
		WHERE id IN (
			SELECT id FROM core.jobs
			WHERE type = 'server_destroy' AND status = 'pending'
			  AND updated_at < now() - interval '1 minute'
			ORDER BY created_at
			LIMIT 20
			FOR UPDATE SKIP LOCKED
		)
		RETURNING id::text, COALESCE(payload->>'server_id', ''), COALESCE(payload->>'node_id', '')
	`)
	if err != nil {
		log.Printf("очистка удалённых серверов: %v", err)
		return
	}
	type task struct{ id, serverID, nodeID string }
	tasks := []task{}
	for rows.Next() {
		var t task
		if rows.Scan(&t.id, &t.serverID, &t.nodeID) == nil {
			tasks = append(tasks, t)
		}
	}
	rows.Close()

	for _, t := range tasks {
		err := r.relay.SendCommand(ctx, t.nodeID, relay.CommandRequest{
			Action:   "destroy",
			ServerID: t.serverID,
			Payload:  map[string]any{"wipe": true},
		})
		if err != nil {
			result, _ := json.Marshal(map[string]string{"error": "Нода недоступна, очистка повторится после подключения: " + err.Error()})
			_, _ = r.db.Exec(ctx, `UPDATE core.jobs SET status = 'pending', result = $2::jsonb WHERE id = $1`, t.id, result)
			continue
		}
		_, _ = r.db.Exec(ctx, `UPDATE core.jobs SET status = 'completed', result = '{"ok":true}'::jsonb WHERE id = $1`, t.id)
		log.Printf("очистка удалённого сервера %s отправлена на ноду %s", t.serverID, t.nodeID)
	}
}
