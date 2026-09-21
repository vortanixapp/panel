package handlers

import (
	"context"
	"log"
	"time"

	"github.com/vortanixapp/panel/internal/relay/events"
	"github.com/vortanixapp/panel/pkg/nodeevents"
	"github.com/vortanixapp/panel/pkg/protocol"
)

const (
	sweepFirst    = 90 * time.Second
	sweepInterval = 30 * time.Second
)

func (h *Handler) RunSweeper(ctx context.Context) {
	timer := time.NewTimer(sweepFirst)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}
		h.sweepStaleNodes(ctx)
		h.sweepOfflineNotices(ctx)
		h.sweepTasks(ctx)
		timer.Reset(sweepInterval)
	}
}

func (h *Handler) sweepStaleNodes(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	rows, err := h.db.Query(ctx, `
		SELECT id::text FROM core.nodes
		WHERE status = 'online'
		  AND (last_seen_at IS NULL OR last_seen_at < now() - interval '90 seconds')
		  AND NOT (id::text = ANY($1::text[]))
	`, h.hub.Connected())
	if err != nil {
		log.Printf("relay: поиск зависших узлов: %v", err)
		return
	}
	var stale []string
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			stale = append(stale, id)
		}
	}
	rows.Close()
	for _, id := range stale {
		log.Printf("relay: узел %s давно не выходил на связь, помечаю недоступным", id)
		h.markNodeOffline(ctx, h.db, id, "stale")
	}
}

func (h *Handler) sweepOfflineNotices(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	rows, err := h.db.Query(ctx, `
		UPDATE core.node_daemons d
		SET offline_notified_at = now()
		FROM core.nodes n
		WHERE d.node_id = n.id
		  AND n.status = 'offline'
		  AND d.offline_notified_at IS NULL
		  AND n.last_seen_at IS NOT NULL
		  AND n.last_seen_at < now() - interval '2 minutes'
		  AND n.last_seen_at > now() - interval '1 day'
		  AND NOT COALESCE(n.maintenance_mode, false)
		RETURNING n.id::text, COALESCE(n.fqdn, ''), n.last_seen_at
	`)
	if err != nil {
		log.Printf("relay: поиск узлов для оповещения о недоступности: %v", err)
		return
	}
	type offlineNode struct {
		id, name string
		lastSeen *time.Time
	}
	var nodes []offlineNode
	for rows.Next() {
		var n offlineNode
		if rows.Scan(&n.id, &n.name, &n.lastSeen) == nil {
			nodes = append(nodes, n)
		}
	}
	rows.Close()
	for _, n := range nodes {
		h.notifyNodeOffline(ctx, h.db, n.id, n.name, n.lastSeen)
	}
}

func (h *Handler) sweepTasks(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	rows, err := h.db.Query(ctx, `
		UPDATE core.node_tasks
		SET status      = 'expired',
			error       = COALESCE(error, 'агент не ответил вовремя'),
			error_code  = COALESCE(error_code, 'timeout'),
			finished_at = now(),
			updated_at  = now()
		WHERE status IN ('queued', 'sent', 'running') AND deadline_at < now()
		RETURNING id::text, node_id::text, action
	`)
	if err != nil {
		log.Printf("relay: закрытие просроченных задач: %v", err)
		return
	}
	type expiredTask struct{ id, nodeID, action string }
	var tasks []expiredTask
	for rows.Next() {
		var t expiredTask
		if rows.Scan(&t.id, &t.nodeID, &t.action) == nil {
			tasks = append(tasks, t)
		}
	}
	rows.Close()
	for _, t := range tasks {
		_ = nodeevents.Record(ctx, h.db, t.nodeID, nodeevents.TaskFailed, nodeevents.Warn,
			map[string]any{"task_id": t.id, "action": t.action, "reason": "expired"}, "")
		events.PublishTenantEvent(ctx, h.redis, protocol.TenantEvent{
			Type: "node.task", NodeID: t.nodeID, Status: "expired", TaskID: t.id,
		})
	}
}
