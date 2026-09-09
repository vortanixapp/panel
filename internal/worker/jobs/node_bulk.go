package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"

	"github.com/vortanixapp/panel/internal/worker/relay"

	"github.com/vortanixapp/panel/pkg/notify"
)

const (
	bulkCommandDelay = 2 * time.Second
	bulkMaxServers   = 1000
)

func (r *Runner) NodeBulkLoop(ctx context.Context, wake <-chan struct{}) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-wake:
			r.heartbeat.Beat(LoopNodeBulk)
			r.drainNodeBulk(ctx)
		case <-ticker.C:
			r.heartbeat.Beat(LoopNodeBulk)
			r.drainNodeBulk(ctx)
		}
	}
}

func (r *Runner) drainNodeBulk(ctx context.Context) {
	for r.processNodeBulkOne(ctx) {
	}
}

type nodeBulkPayload struct {
	NodeID      string `json:"node_id"`
	NodeName    string `json:"node_name"`
	Action      string `json:"action"`
	Message     string `json:"message"`
	Days        int    `json:"days"`
	OnlyRunning bool   `json:"only_running"`
}

func (r *Runner) processNodeBulkOne(ctx context.Context) bool {
	var jobID string
	var payload []byte
	err := r.db.QueryRow(ctx, `
		UPDATE core.jobs SET status = 'running', attempts = attempts + 1
		WHERE id = (
			SELECT id FROM core.jobs
			WHERE type = 'node_bulk' AND status = 'pending' AND attempts < 5
			ORDER BY created_at ASC
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		)
		RETURNING id::text, payload
	`).Scan(&jobID, &payload)
	if err != nil {
		return false
	}

	var pl nodeBulkPayload
	_ = json.Unmarshal(payload, &pl)
	if pl.NodeID == "" || pl.Action == "" {
		r.failJobDirect(ctx, jobID, "missing node_id or action in payload")
		return true
	}

	beatCtx, stopBeat := context.WithCancel(ctx)
	go func() {
		t := time.NewTicker(30 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-beatCtx.Done():
				return
			case <-t.C:
				r.heartbeat.Beat(LoopNodeBulk)
			}
		}
	}()
	defer stopBeat()

	done, failed, err := r.runNodeBulk(ctx, pl)
	if err != nil {
		r.failJobDirect(ctx, jobID, err.Error())
		return true
	}
	result, _ := json.Marshal(map[string]any{"ok": true, "done": done, "failed": failed})
	_, _ = r.db.Exec(ctx, `UPDATE core.jobs SET status = 'completed', result = $2::jsonb WHERE id = $1`,
		jobID, result)
	log.Printf("node bulk: нода %s, действие %s: успешно %d, с ошибкой %d", pl.NodeID, pl.Action, done, failed)
	return true
}

func (r *Runner) runNodeBulk(ctx context.Context, pl nodeBulkPayload) (int, int, error) {
	switch pl.Action {
	case "notify":
		return r.bulkNotify(ctx, pl)
	case "extend":
		return r.bulkExtend(ctx, pl)
	case "start", "stop", "restart":
		return r.bulkPower(ctx, pl)
	default:
		return 0, 0, fmt.Errorf("неизвестное действие %q", pl.Action)
	}
}

func (r *Runner) bulkPower(ctx context.Context, pl nodeBulkPayload) (int, int, error) {
	query := `
		SELECT id::text, name, game_id, limits, COALESCE(primary_port, 0),
		       COALESCE(config->>'startup_params', '')
		FROM core.servers
		WHERE node_id = $1`
	if pl.OnlyRunning {
		query += ` AND status = 'running'`
	}
	if pl.Action == "start" || pl.Action == "restart" {
		query += ` AND COALESCE(is_blocked, false) = false
		           AND (expires_at IS NULL OR expires_at > now())`
	}
	query += fmt.Sprintf(` ORDER BY created_at LIMIT %d`, bulkMaxServers)

	rows, err := r.db.Query(ctx, query, pl.NodeID)
	if err != nil {
		return 0, 0, err
	}
	type target struct {
		id, name, gameID string
		limits           []byte
		port             int
		startupParams    string
	}
	targets := []target{}
	for rows.Next() {
		var t target
		if rows.Scan(&t.id, &t.name, &t.gameID, &t.limits, &t.port, &t.startupParams) == nil {
			targets = append(targets, t)
		}
	}
	rows.Close()

	done, failed := 0, 0
	for _, t := range targets {
		cmdPayload := map[string]any{"power_action": pl.Action}
		if pl.Action == "start" || pl.Action == "restart" {
			var lim map[string]any
			_ = json.Unmarshal(t.limits, &lim)
			cmdPayload["name"] = t.name
			cmdPayload["game_id"] = t.gameID
			cmdPayload["limits"] = lim
			cmdPayload["startup_params"] = t.startupParams
			if t.port > 0 {
				cmdPayload["primary_port"] = t.port
			}
			if bindIP := r.serverBindIP(ctx, t.id); bindIP != "" {
				cmdPayload["bind_ip"] = bindIP
			}
			if img := r.resolveDockerImage(ctx, t.id); img != "" {
				cmdPayload["docker_image"] = img
			}
		}
		if err := r.relay.SendCommand(ctx, pl.NodeID, relay.CommandRequest{
			CommandID: uuid.NewString(),
			Action:    "power",
			ServerID:  t.id,
			Payload:   cmdPayload,
		}); err != nil {
			failed++
			log.Printf("node bulk: сервер %s: %v", t.id, err)
		} else {
			done++
		}
		select {
		case <-ctx.Done():
			return done, failed, ctx.Err()
		case <-time.After(bulkCommandDelay):
		}
	}
	return done, failed, nil
}

func (r *Runner) bulkExtend(ctx context.Context, pl nodeBulkPayload) (int, int, error) {
	tag, err := r.db.Exec(ctx, `
		UPDATE core.servers
		SET expires_at = expires_at + make_interval(days => $2::int)
		WHERE node_id = $1 AND expires_at IS NOT NULL
	`, pl.NodeID, pl.Days)
	if err != nil {
		return 0, 0, err
	}
	count := int(tag.RowsAffected())

	message := fmt.Sprintf(
		"Аренда ваших серверов на локации «%s» продлена на %d дн. — компенсация за технические работы.",
		pl.NodeName, pl.Days)
	_, _, _ = r.bulkNotify(ctx, nodeBulkPayload{
		NodeID: pl.NodeID, NodeName: pl.NodeName, Message: message,
	})
	return count, 0, nil
}

func (r *Runner) bulkNotify(ctx context.Context, pl nodeBulkPayload) (int, int, error) {
	rows, err := r.db.Query(ctx, `
		SELECT DISTINCT user_id::text FROM core.servers
		WHERE node_id = $1 AND user_id IS NOT NULL
	`, pl.NodeID)
	if err != nil {
		return 0, 0, err
	}
	owners := []string{}
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil && id != "" {
			owners = append(owners, id)
		}
	}
	rows.Close()

	title := "Локация " + pl.NodeName
	for _, userID := range owners {
		r.notifyUser(ctx, userID, notify.Event{
			Kind:  notify.KindNodeMaintenance,
			Title: title,
			Body:  pl.Message,
			Meta:  map[string]any{"node_id": pl.NodeID, "node_name": pl.NodeName},
		})
	}
	return len(owners), 0, nil
}
