package agent

import (
	"context"
	"encoding/json"
	"time"

	"github.com/vortanixapp/panel/internal/agent/docker"
	"github.com/vortanixapp/panel/pkg/protocol"
)

func payloadStrings(v any) []string {
	raw, _ := v.([]any)
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		if s, ok := item.(string); ok && s != "" {
			out = append(out, s)
		}
	}
	return out
}

func payloadSet(payload map[string]any, key string) map[string]bool {
	if _, ok := payload[key]; !ok {
		return nil
	}
	set := map[string]bool{}
	for _, s := range payloadStrings(payload[key]) {
		set[s] = true
	}
	return set
}

func v2(result map[string]any) map[string]any {
	result[protocol.ResultVersionKey] = protocol.ResultVersionCurrent
	return result
}

func (a *Agent) taskProgress(cmdID string, progress map[string]any) {
	if !a.relayHas(protocol.CapTasks) {
		return
	}
	msg, _ := json.Marshal(protocol.TaskProgressMessage{Type: protocol.MsgTaskProgress, CommandID: cmdID, Progress: progress})
	_ = a.send(msg)
}

func (a *Agent) routeNodeOp(cmd protocol.CommandMessage) bool {
	run := func(ctx context.Context) { a.nodeOp(ctx, cmd) }
	switch cmd.Action {
	case protocol.ActionContainersList:
		a.disp.nodeReadJob(job{id: cmd.ID, action: cmd.Action, timeout: time.Minute, run: run})
	case protocol.ActionDiskUsage:
		a.disp.serial("node:disk", job{id: cmd.ID, action: cmd.Action, timeout: 30 * time.Minute, run: run})
	case protocol.ActionCleanupPreview:
		a.disp.serial("node:ctl", job{id: cmd.ID, action: cmd.Action, timeout: 5 * time.Minute, run: run})
	case protocol.ActionCleanupApply:
		a.disp.serial("node:ctl", job{id: cmd.ID, action: cmd.Action, timeout: 30 * time.Minute, run: run})
	case protocol.ActionDiagnostics:
		a.disp.serial("node:diag", job{id: cmd.ID, action: cmd.Action, timeout: 5 * time.Minute, run: run})
	default:
		return false
	}
	return true
}

func (a *Agent) nodeOp(ctx context.Context, cmd protocol.CommandMessage) {
	switch cmd.Action {
	case protocol.ActionContainersList:
		list, err := docker.ListNodeContainers(ctx)
		if err != nil {
			a.sendAck(cmd.ID, false, err, nil)
			return
		}
		a.sendAck(cmd.ID, true, nil, v2(map[string]any{"containers": list}))

	case protocol.ActionDiskUsage:
		var last time.Time
		rep := docker.NodeDiskUsage(ctx, payloadSet(cmd.Payload, "known_servers"), func(stage string, done, total int) {
			if time.Since(last) < time.Second && done != total {
				return
			}
			last = time.Now()
			a.taskProgress(cmd.ID, map[string]any{"stage": stage, "done": done, "total": total})
		})
		a.sendAck(cmd.ID, true, nil, v2(map[string]any{"report": rep}))

	case protocol.ActionCleanupPreview:
		items, err := docker.CleanupPreview(ctx, cleanupScope(cmd.Payload))
		if err != nil {
			a.sendAck(cmd.ID, false, err, nil)
			return
		}
		a.sendAck(cmd.ID, true, nil, v2(map[string]any{"items": items}))

	case protocol.ActionCleanupApply:
		a.taskProgress(cmd.ID, map[string]any{"stage": "cleanup"})
		res, err := docker.CleanupApply(ctx, cleanupScope(cmd.Payload), payloadStrings(cmd.Payload["items"]))
		if err != nil {
			a.sendAck(cmd.ID, false, err, nil)
			return
		}
		a.sendAck(cmd.ID, true, nil, v2(map[string]any{"result": res}))

	case protocol.ActionDiagnostics:
		_, self := a.collectFacts(ctx)
		target, _ := cmd.Payload["target_image"].(string)
		if target == "" {
			target = self.Image
		}
		in := docker.DiagnosticsInput{
			ClockSkewMs:   a.clockSkewMs.Load(),
			TargetImage:   target,
			Reconnects:    int(a.reconnects.Load()),
			UptimeSec:     int64(time.Since(a.startedAt).Seconds()),
			RestartPolicy: self.RestartPolicy,
		}
		checks := docker.RunDiagnostics(ctx, in, func(id string) {
			a.taskProgress(cmd.ID, map[string]any{"stage": id})
		})
		a.sendAck(cmd.ID, true, nil, v2(map[string]any{"checks": checks}))
	}
}

func cleanupScope(payload map[string]any) docker.CleanupScope {
	return docker.CleanupScope{
		KnownServers: payloadSet(payload, "known_servers"),
		GameRepos:    payloadSet(payload, "game_repos"),
	}
}
