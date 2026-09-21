package agent

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"
	"time"

	"github.com/gorilla/websocket"

	"github.com/vortanixapp/panel/internal/agent/docker"
	"github.com/vortanixapp/panel/pkg/dockerapi"
	"github.com/vortanixapp/panel/pkg/protocol"
)

const (
	agentLogsMaxLines = 1000
	agentLogsMaxBytes = 512 << 10
	agentRestartExit  = 75
)

var errOutsideDocker = errors.New("агент запущен вне контейнера Docker")

func (a *Agent) nodeInfo(ctx context.Context, cmd protocol.CommandMessage) {
	host, self := a.collectFacts(ctx)
	running, queued := a.disp.stats()
	a.sendAck(cmd.ID, true, nil, map[string]any{
		protocol.ResultVersionKey: protocol.ResultVersionCurrent,
		"host":                    host,
		"agent":                   self,
		"stats":                   docker.CollectHostStats(),
		"queue":                   map[string]any{"running": running, "queued": queued},
		"ops":                     a.ops.list(),
	})
}

func payloadTime(v any) time.Time {
	s, _ := v.(string)
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

func (a *Agent) agentLogs(ctx context.Context, cmd protocol.CommandMessage) {
	id := dockerapi.SelfContainerID()
	if id == "" {
		a.sendAck(cmd.ID, false, errOutsideDocker, nil)
		return
	}
	since := payloadTime(cmd.Payload["since"])
	until := payloadTime(cmd.Payload["until"])
	if !until.IsZero() {
		until = until.Add(-time.Nanosecond)
	}
	tail := docker.IntFromPayload(cmd.Payload["tail"])
	if tail <= 0 || tail > agentLogsMaxLines {
		tail = agentLogsMaxLines
	}
	lines, truncated, err := dockerapi.New().LogsRange(ctx, id, since, until, tail, agentLogsMaxBytes)
	if err != nil {
		a.sendAck(cmd.ID, false, err, nil)
		return
	}
	out := make([]map[string]any, 0, len(lines))
	var first, last time.Time
	for _, l := range lines {
		ts := ""
		if !l.TS.IsZero() {
			ts = l.TS.UTC().Format(time.RFC3339Nano)
			if first.IsZero() {
				first = l.TS
			}
			last = l.TS
		}
		out = append(out, map[string]any{"ts": ts, "text": l.Text})
	}
	result := map[string]any{
		protocol.ResultVersionKey: protocol.ResultVersionCurrent,
		"lines":                   out,
		"truncated":               truncated,
	}
	switch {
	case !last.IsZero():
		result["cursor"] = last.Add(time.Nanosecond).UTC().Format(time.RFC3339Nano)
	case !since.IsZero():
		result["cursor"] = since.UTC().Format(time.RFC3339Nano)
	}
	if !first.IsZero() {
		result["first"] = first.UTC().Format(time.RFC3339Nano)
	}
	a.sendAck(cmd.ID, true, nil, result)
}

func (a *Agent) restartSelf(cmd protocol.CommandMessage) {
	force, _ := cmd.Payload["force"].(bool)
	if ops := a.ops.list(); len(ops) > 0 && !force {
		a.sendAck(cmd.ID, true, nil, map[string]any{
			protocol.ResultVersionKey: protocol.ResultVersionCurrent,
			"accepted":                false,
			"busy":                    ops,
		})
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	id := dockerapi.SelfContainerID()
	if id == "" {
		a.sendAckCode(cmd.ID, false, errOutsideDocker, nil, protocol.CodeRestartPolicy)
		return
	}
	c, err := dockerapi.New().Inspect(ctx, id)
	if err != nil {
		a.sendAck(cmd.ID, false, err, nil)
		return
	}
	policy := ""
	if rp, ok := c.HostConfig["RestartPolicy"].(map[string]any); ok {
		policy, _ = rp["Name"].(string)
	}
	if policy == "" || policy == "no" {
		a.sendAckCode(cmd.ID, false,
			errors.New("у контейнера агента нет политики перезапуска, Docker не поднимет его обратно"),
			nil, protocol.CodeRestartPolicy)
		return
	}
	a.sendAck(cmd.ID, true, nil, map[string]any{
		protocol.ResultVersionKey: protocol.ResultVersionCurrent,
		"accepted":                true,
		"boot_id":                 a.bootID,
	})
	log.Printf("перезапуск агента по команде панели")
	time.Sleep(500 * time.Millisecond)
	a.closeConnection()
	os.Exit(agentRestartExit)
}

func (a *Agent) closeConnection() {
	a.writeMu.Lock()
	defer a.writeMu.Unlock()
	if a.conn == nil {
		return
	}
	_ = a.conn.WriteControl(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, "restart"),
		time.Now().Add(time.Second))
	_ = a.conn.Close()
	a.conn = nil
}

func (a *Agent) onWelcome(data []byte) {
	var w protocol.WelcomeMessage
	if json.Unmarshal(data, &w) != nil {
		return
	}
	caps := make(map[string]bool, len(w.Caps))
	for _, c := range w.Caps {
		caps[c] = true
	}
	a.relayCaps.Store(caps)
	if t, err := time.Parse(time.RFC3339Nano, w.ServerTime); err == nil {
		a.clockSkewMs.Store(time.Since(t).Milliseconds())
	}
	if caps[protocol.CapStateSnapshot] {
		go a.sendStateSnapshot()
	}
}

func (a *Agent) relayHas(capability string) bool {
	caps, _ := a.relayCaps.Load().(map[string]bool)
	return caps[capability]
}

func (a *Agent) sendStateSnapshot() {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	servers, err := docker.SnapshotServers(ctx)
	if err != nil {
		log.Printf("снимок состояния серверов не собран: %v", err)
		return
	}
	for i := range servers {
		if op, busy := a.ops.serverOp(servers[i].ServerID); busy {
			switch op {
			case "install":
				servers[i].Status, servers[i].Error = "installing", ""
			case "update":
				servers[i].Status, servers[i].Error = "updating", ""
			}
		}
	}
	msg, _ := json.Marshal(protocol.StateSnapshotMessage{Type: protocol.MsgStateSnapshot, Servers: servers})
	_ = a.send(msg)
}

func (a *Agent) nodeEvent(kind, level string, data map[string]any) {
	if !a.relayHas(protocol.CapNodeEvents) {
		return
	}
	msg, _ := json.Marshal(protocol.NodeEventMessage{Type: protocol.MsgNodeEvent, Kind: kind, Level: level, Data: data})
	_ = a.send(msg)
}
