package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/vortanixapp/panel/internal/relay/events"
	"github.com/vortanixapp/panel/internal/relay/hub"
	"github.com/vortanixapp/panel/pkg/nodeevents"
	"github.com/vortanixapp/panel/pkg/protocol"
)

const agentEventInterval = 10 * time.Minute

var agentEventKinds = map[string]bool{
	string(nodeevents.FirewallFailed):    true,
	string(nodeevents.CronFailed):        true,
	string(nodeevents.DockerUnreachable): true,
	string(nodeevents.DockerRecovered):   true,
}

type eventLimiter struct {
	mu   sync.Mutex
	last map[string]time.Time
}

func (l *eventLimiter) allow(key string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.last == nil {
		l.last = map[string]time.Time{}
	}
	if at, ok := l.last[key]; ok && now.Sub(at) < agentEventInterval {
		return false
	}
	l.last[key] = now
	return true
}

func stringList(v any) []string {
	raw, _ := v.([]any)
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		if s, ok := item.(string); ok && s != "" {
			out = append(out, s)
		}
	}
	return out
}

func joinNonEmpty(sep string, parts ...string) string {
	var kept []string
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			kept = append(kept, p)
		}
	}
	return strings.Join(kept, sep)
}

func (h *Handler) storeHello(ctx context.Context, c *hub.AgentConn, env map[string]any, prevBoot string) {
	proto := intNum(env["proto"])
	caps := stringList(env["caps"])
	bootID, _ := env["boot_id"].(string)
	if proto < protocol.ProtoVersion {
		proto, caps, bootID = 1, nil, ""
	}
	c.SetProtocol(proto, caps, bootID)

	if proto < protocol.ProtoVersion {
		execLogged(ctx, c.DB, `
			UPDATE core.node_daemons
			SET proto = 1, caps = '{}', boot_id = NULL, started_at = NULL, updated_at = now()
			WHERE node_id = $1
		`, c.NodeID)
		return
	}

	var startedAt *time.Time
	if s, _ := env["started_at"].(string); s != "" {
		if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
			startedAt = &t
		}
	}
	hostInfo, _ := env["host_info"].(map[string]any)
	agentInfo, _ := env["agent_info"].(map[string]any)
	host := map[string]any{}
	for k, v := range hostInfo {
		host[k] = v
	}
	if len(agentInfo) > 0 {
		host["agent"] = agentInfo
	}
	hostRaw, _ := json.Marshal(host)
	osName, _ := hostInfo["os"].(string)
	kernel, _ := hostInfo["kernel"].(string)
	arch, _ := hostInfo["arch"].(string)
	platform := joinNonEmpty(" · ", osName, kernel, arch)

	execLogged(ctx, c.DB, `
		UPDATE core.node_daemons
		SET proto      = $2,
			caps       = $3::text[],
			boot_id    = NULLIF($4, ''),
			started_at = $5,
			host       = $6::jsonb,
			platform   = COALESCE(NULLIF($7, ''), platform),
			updated_at = now()
		WHERE node_id = $1
	`, c.NodeID, proto, caps, bootID, startedAt, string(hostRaw), platform)

	if bootID != "" && bootID != prevBoot {
		version, _ := env["version"].(string)
		_ = nodeevents.Record(ctx, c.DB, c.NodeID, nodeevents.AgentStarted, nodeevents.Info,
			map[string]any{"boot_id": bootID, "version": normalizeVersion(version)}, "")
		h.settleTasksOnBoot(ctx, c, bootID)
	}

	welcome, _ := json.Marshal(protocol.WelcomeMessage{
		Type:       protocol.MsgWelcome,
		Proto:      protocol.ProtoVersion,
		Caps:       protocol.RelayCaps,
		ServerTime: time.Now().UTC().Format(time.RFC3339Nano),
	})
	if err := h.hub.SendCommand(c.NodeID, welcome); err != nil {
		log.Printf("relay: приветствие узлу %s не отправлено: %v", c.NodeID, err)
	}
}

func (h *Handler) settleTasksOnBoot(ctx context.Context, c *hub.AgentConn, bootID string) {
	rows, err := c.DB.Query(ctx, `
		UPDATE core.node_tasks
		SET status = 'done', result = jsonb_build_object('boot_id', $2::text),
			finished_at = now(), updated_at = now()
		WHERE node_id = $1 AND action = $3 AND status IN ('queued', 'sent', 'running')
		RETURNING id::text
	`, c.NodeID, bootID, protocol.ActionAgentRestart)
	if err == nil {
		var done []string
		for rows.Next() {
			var id string
			if rows.Scan(&id) == nil {
				done = append(done, id)
			}
		}
		rows.Close()
		for _, id := range done {
			h.publishTask(ctx, c.NodeID, id, "done")
		}
	}

	rows, err = c.DB.Query(ctx, `
		UPDATE core.node_tasks
		SET status = 'failed', error = 'агент перезапустился во время задачи', error_code = $3,
			finished_at = now(), updated_at = now()
		WHERE node_id = $1 AND action <> $4 AND status IN ('queued', 'sent', 'running')
		  AND boot_id IS NOT NULL AND boot_id <> $2
		RETURNING id::text, action
	`, c.NodeID, bootID, protocol.CodeAgentRestarted, protocol.ActionAgentRestart)
	if err != nil {
		return
	}
	type failedTask struct{ id, action string }
	var failed []failedTask
	for rows.Next() {
		var t failedTask
		if rows.Scan(&t.id, &t.action) == nil {
			failed = append(failed, t)
		}
	}
	rows.Close()
	for _, t := range failed {
		_ = nodeevents.Record(ctx, c.DB, c.NodeID, nodeevents.TaskFailed, nodeevents.Warn,
			map[string]any{"task_id": t.id, "action": t.action, "reason": protocol.CodeAgentRestarted}, "")
		h.publishTask(ctx, c.NodeID, t.id, "failed")
	}
}

func (h *Handler) publishTask(ctx context.Context, nodeID, taskID, status string) {
	events.PublishTenantEvent(ctx, h.redis, protocol.TenantEvent{
		Type: "node.task", NodeID: nodeID, Status: status, TaskID: taskID,
	})
}

func (h *Handler) gate(w http.ResponseWriter, nodeID, action string) bool {
	need := protocol.RequiredCap(action)
	if need == "" {
		return true
	}
	c := h.hub.Get(nodeID)
	if c == nil {
		writeCoded(w, http.StatusBadGateway, "node offline", protocol.CodeNodeOffline)
		return false
	}
	if !c.HasCap(need) {
		writeCoded(w, http.StatusConflict,
			"агент на ноде устарел и не умеет эту операцию — обновите его", protocol.CodeAgentTooOld)
		return false
	}
	return true
}

func writeCoded(w http.ResponseWriter, status int, msg, code string) {
	body := map[string]string{"error": msg}
	if code != "" {
		body["code"] = code
	}
	writeJSON(w, status, body)
}

func (h *Handler) completeTask(ctx context.Context, c *hub.AgentConn, ack protocol.AckMessage) {
	if _, err := uuid.Parse(ack.CommandID); err != nil {
		return
	}
	var action string
	if c.DB.QueryRow(ctx, `
		SELECT action FROM core.node_tasks
		WHERE id = $1::uuid AND node_id = $2 AND status IN ('queued', 'sent', 'running')
	`, ack.CommandID, c.NodeID).Scan(&action) != nil {
		return
	}
	status := "done"
	errMsg, code := ack.Error, ack.Code
	if !ack.OK {
		status = "failed"
	}
	if action == protocol.ActionAgentRestart && ack.OK {
		accepted, _ := ack.Result["accepted"].(bool)
		if accepted {
			status = "running"
		} else {
			status, code = "failed", protocol.CodeAgentBusy
			if errMsg == "" {
				errMsg = "на ноде идут операции, перезапуск отложен"
			}
		}
	}
	result, _ := json.Marshal(ack.Result)
	tag, err := c.DB.Exec(ctx, `
		UPDATE core.node_tasks
		SET status      = $3,
			result      = $4::jsonb,
			error       = NULLIF($5, ''),
			error_code  = NULLIF($6, ''),
			finished_at = CASE WHEN $3 IN ('done', 'failed') THEN now() ELSE finished_at END,
			updated_at  = now()
		WHERE id = $1::uuid AND node_id = $2 AND status IN ('queued', 'sent', 'running')
	`, ack.CommandID, c.NodeID, status, string(result), errMsg, code)
	if err != nil || tag.RowsAffected() == 0 {
		return
	}
	if status == "failed" {
		_ = nodeevents.Record(ctx, c.DB, c.NodeID, nodeevents.TaskFailed, nodeevents.Warn,
			map[string]any{"task_id": ack.CommandID, "action": action, "error": errMsg, "code": code}, "")
	}
	h.publishTask(ctx, c.NodeID, ack.CommandID, status)
}

func (h *Handler) taskProgress(ctx context.Context, c *hub.AgentConn, data []byte) {
	var msg protocol.TaskProgressMessage
	if json.Unmarshal(data, &msg) != nil {
		return
	}
	if _, err := uuid.Parse(msg.CommandID); err != nil {
		return
	}
	raw, _ := json.Marshal(msg.Progress)
	tag, err := c.DB.Exec(ctx, `
		UPDATE core.node_tasks
		SET progress = $3::jsonb, status = 'running', updated_at = now()
		WHERE id = $1::uuid AND node_id = $2 AND status IN ('queued', 'sent', 'running')
	`, msg.CommandID, c.NodeID, string(raw))
	if err == nil && tag.RowsAffected() > 0 {
		h.publishTask(ctx, c.NodeID, msg.CommandID, "running")
	}
}

func (h *Handler) agentEvent(ctx context.Context, c *hub.AgentConn, data []byte) {
	var msg protocol.NodeEventMessage
	if json.Unmarshal(data, &msg) != nil || !agentEventKinds[msg.Kind] {
		return
	}
	if !h.events.allow(c.NodeID+"|"+msg.Kind, time.Now()) {
		return
	}
	level := nodeevents.Level(msg.Level)
	switch level {
	case nodeevents.Info, nodeevents.Success, nodeevents.Warn, nodeevents.Error:
	default:
		level = nodeevents.Warn
	}
	if err := nodeevents.Record(ctx, c.DB, c.NodeID, nodeevents.Kind(msg.Kind), level, msg.Data, ""); err != nil {
		log.Printf("relay: событие узла %s не записано: %v", c.NodeID, err)
	}
}

func (h *Handler) reconcileSnapshot(ctx context.Context, c *hub.AgentConn, data []byte) {
	var snap protocol.StateSnapshotMessage
	if json.Unmarshal(data, &snap) != nil {
		return
	}
	fixed := 0
	for _, s := range snap.Servers {
		if s.ServerID == "" || !h.nodeOwnsServer(ctx, c, s.ServerID) || pendingServerOperation(ctx, h.redis, s.ServerID) {
			continue
		}
		var dbStatus string
		if c.DB.QueryRow(ctx, `SELECT COALESCE(status, '') FROM core.servers WHERE id = $1`, s.ServerID).Scan(&dbStatus) != nil {
			continue
		}
		if !snapshotDiffers(dbStatus, s.Status) {
			continue
		}
		h.applyServerStatus(ctx, c, s.ServerID, s.Status, s.Error)
		fixed++
	}
	if fixed > 0 {
		log.Printf("relay: узел %s сообщил состояние, поправлено серверов: %d", c.NodeID, fixed)
	}
}

func snapshotDiffers(dbStatus, agentStatus string) bool {
	switch agentStatus {
	case "running":
		switch dbStatus {
		case "stopped", "offline", "error", "starting", "stopping":
			return true
		}
	case "stopped":
		switch dbStatus {
		case "running", "starting", "stopping":
			return true
		}
	case "error":
		switch dbStatus {
		case "running", "starting":
			return true
		}
	}
	return false
}
