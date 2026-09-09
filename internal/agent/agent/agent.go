package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/vortanixapp/panel/internal/agent/docker"
	"github.com/vortanixapp/panel/pkg/protocol"
)

type Agent struct {
	relayURL            string
	token               string
	nodeID              string
	version             string
	conn                *websocket.Conn
	writeMu             sync.Mutex
	pendingBinaryWrites map[string]binaryWritePending
}

const (
	writeWait = 15 * time.Second

	pongWait = 90 * time.Second

	handshakeTimeout = 20 * time.Second

	reconnectMin = 2 * time.Second
	reconnectMax = 30 * time.Second
)

func (a *Agent) send(payload []byte) error {
	a.writeMu.Lock()
	defer a.writeMu.Unlock()
	conn := a.conn
	if conn == nil {
		return nil
	}
	_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
	err := conn.WriteMessage(websocket.TextMessage, payload)
	if err != nil {
		log.Printf("отправка не удалась, рвём связь: %v", err)
		_ = conn.Close()
	}
	return err
}

type binaryWritePending struct {
	serverID string
	path     string
}

func New() *Agent {
	return &Agent{
		relayURL:            os.Getenv("RELAY_URL"),
		token:               os.Getenv("AGENT_TOKEN"),
		nodeID:              os.Getenv("NODE_ID"),
		version:             envOr("VORTANIX_VERSION", "dev"),
		pendingBinaryWrites: map[string]binaryWritePending{},
	}
}

func (a *Agent) Run() {
	if a.relayURL == "" || a.token == "" || a.nodeID == "" {
		log.Fatal("RELAY_URL, AGENT_TOKEN and NODE_ID are required")
	}
	backoff := reconnectMin
	for {
		if err := a.connect(); err != nil {
			log.Printf("не удалось подключиться: %v, повтор через %s", err, backoff)
			time.Sleep(backoff)
			if backoff *= 2; backoff > reconnectMax {
				backoff = reconnectMax
			}
			continue
		}
		backoff = reconnectMin
		a.loop()
		time.Sleep(reconnectMin)
	}
}

func (a *Agent) connect() error {
	header := http.Header{}
	header.Set("Authorization", "Bearer "+a.token)
	dialer := &websocket.Dialer{HandshakeTimeout: handshakeTimeout}
	conn, _, err := dialer.Dial(a.relayURL, header)
	if err != nil {
		return err
	}

	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPingHandler(func(appData string) error {
		_ = conn.SetReadDeadline(time.Now().Add(pongWait))
		err := conn.WriteControl(websocket.PongMessage, []byte(appData), time.Now().Add(writeWait))
		if err == websocket.ErrCloseSent {
			return nil
		}
		return err
	})

	hello, _ := json.Marshal(map[string]string{
		"type":    protocol.MsgHello,
		"node_id": a.nodeID,
		"version": a.version,
	})
	_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
	if err := conn.WriteMessage(websocket.TextMessage, hello); err != nil {
		_ = conn.Close()
		return err
	}

	a.writeMu.Lock()
	a.conn = conn
	a.writeMu.Unlock()

	log.Printf("connected to relay as node=%s", a.nodeID)
	return nil
}

func (a *Agent) loop() {
	defer a.conn.Close()

	done := make(chan struct{})
	defer close(done)
	go a.heartbeat(done)
	go a.metricsLoop(done)
	for {
		mt, data, err := a.conn.ReadMessage()
		if err != nil {
			log.Printf("связь потеряна: %v", err)
			return
		}
		_ = a.conn.SetReadDeadline(time.Now().Add(pongWait))
		if mt == websocket.BinaryMessage {
			a.handleBinaryMessage(data)
			continue
		}
		a.handleCommand(data)
	}
}

func (a *Agent) handleBinaryMessage(data []byte) {
	sep := -1
	for i, b := range data {
		if b == '\n' {
			sep = i
			break
		}
	}
	if sep <= 0 {
		return
	}
	cmdID := string(data[:sep])
	pending, ok := a.pendingBinaryWrites[cmdID]
	if !ok {
		return
	}
	delete(a.pendingBinaryWrites, cmdID)
	raw := data[sep+1:]
	execErr := docker.WriteFileBytes(context.Background(), pending.serverID, pending.path, raw)
	a.sendAck(cmdID, execErr == nil, execErr, map[string]any{
		"path":       pending.path,
		"size_bytes": len(raw),
	})
}

func (a *Agent) heartbeat(done <-chan struct{}) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
		}
		stats := docker.CollectHostStats()
		msg, _ := json.Marshal(map[string]any{
			"type":    protocol.MsgHeartbeat,
			"node_id": a.nodeID,
			"version": a.version,
			"host":    stats,
		})
		if err := a.send(msg); err != nil {
			return
		}
	}
}

func (a *Agent) metricsLoop(done <-chan struct{}) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
		}
		ctx := context.Background()
		ids, err := docker.ListManagedServerIDs(ctx)
		if err != nil {
			continue
		}
		for _, id := range ids {
			if docker.Status(ctx, id) != "running" {
				continue
			}
			st, err := docker.CollectStats(ctx, id)
			if err != nil {
				continue
			}
			msg, _ := json.Marshal(protocol.MetricsMessage{
				Type: protocol.MsgMetrics, ServerID: id,
				CPUPct: st.CPUPct, MemUsedMB: st.MemUsedMB, MemLimitMB: st.MemLimitMB,
			})
			_ = a.send(msg)
		}
	}
}

func (a *Agent) handleCommand(data []byte) {
	var cmd protocol.CommandMessage
	if err := json.Unmarshal(data, &cmd); err != nil || cmd.Type != protocol.MsgCommand {
		return
	}
	ctx := context.Background()
	var execErr error
	status := "stopped"

	switch cmd.Action {
	case "power":
		action, _ := cmd.Payload["power_action"].(string)
		name, _ := cmd.Payload["name"].(string)
		gameID, _ := cmd.Payload["game_id"].(string)
		dockerImage, _ := cmd.Payload["docker_image"].(string)
		limits, _ := cmd.Payload["limits"].(map[string]any)
		primaryPort := docker.IntFromPayload(cmd.Payload["primary_port"])
		bindIP, _ := cmd.Payload["bind_ip"].(string)
		startupParams, _ := cmd.Payload["startup_params"].(string)
		install := docker.InstallSpecFromPayload(cmd.Payload["install"])
		switch action {
		case "start", "restart":
			if execErr = docker.WriteStartupParams(cmd.ServerID, startupParams); execErr != nil {
				a.sendStatus(cmd.ServerID, "error", execErr.Error())
				a.sendAck(cmd.ID, false, execErr, nil)
				return
			}
			if install.HasSource() && docker.NeedsInstall(cmd.ServerID) {
				a.sendStatus(cmd.ServerID, "installing", "")
				go a.installAndStart(cmd.ServerID, name, gameID, limits, dockerImage, primaryPort, bindIP, install)
				a.sendAck(cmd.ID, true, nil, map[string]any{"status": "installing"})
				return
			}
			if action == "restart" {
				execErr = docker.Restart(ctx, cmd.ServerID, name, gameID, limits, dockerImage, primaryPort, bindIP)
			} else {
				execErr = docker.Start(ctx, cmd.ServerID, name, gameID, limits, dockerImage, primaryPort, bindIP)
			}
			status = "running"
		case "stop":
			execErr = docker.Stop(ctx, cmd.ServerID)
		case "kill":
			execErr = docker.Kill(ctx, cmd.ServerID)
		}
		if execErr != nil {
			status = "error"
		} else if action == "stop" || action == "kill" {
			status = "stopped"
		} else {
			status = docker.DetailedStatus(ctx, cmd.ServerID)
		}
		errMsg := ""
		if execErr != nil {
			errMsg = execErr.Error()
		} else if status == "error" {
			errMsg = docker.ContainerError(ctx, cmd.ServerID)
			if errMsg == "" {
				errMsg = "container failed to start"
			}
		}
		a.sendStatus(cmd.ServerID, status, errMsg)
		a.sendAck(cmd.ID, execErr == nil, execErr, nil)
		return
	case "update":
		spec := docker.InstallSpecFromPayload(cmd.Payload["install"])
		if !spec.HasSource() {
			execErr = fmt.Errorf("для этой версии игры не задан источник файлов")
			a.sendStatus(cmd.ServerID, docker.DetailedStatus(ctx, cmd.ServerID), execErr.Error())
			a.sendAck(cmd.ID, false, execErr, nil)
			return
		}
		name, _ := cmd.Payload["name"].(string)
		gameID, _ := cmd.Payload["game_id"].(string)
		dockerImage, _ := cmd.Payload["docker_image"].(string)
		limits, _ := cmd.Payload["limits"].(map[string]any)
		primaryPort := docker.IntFromPayload(cmd.Payload["primary_port"])
		bindIP, _ := cmd.Payload["bind_ip"].(string)
		a.sendStatus(cmd.ServerID, "updating", "")
		go a.updateAndStart(cmd.ServerID, name, gameID, limits, dockerImage, primaryPort, bindIP, spec)
		a.sendAck(cmd.ID, true, nil, map[string]any{"status": "updating"})
		return
	case "destroy":
		execErr = docker.Destroy(ctx, cmd.ServerID)
		if wipe, _ := cmd.Payload["wipe"].(bool); wipe && execErr == nil {
			execErr = docker.WipeData(cmd.ServerID)
		}
		for _, err := range docker.CleanupServerTraces(ctx, cmd.ServerID) {
			log.Printf("очистка следов сервера %s: %v", cmd.ServerID, err)
		}
		status = "stopped"
	case "logs":
		tail := 100
		if v, ok := cmd.Payload["tail"].(float64); ok && v > 0 {
			tail = int(v)
		}
		lines, logErr := docker.TailLogs(ctx, cmd.ServerID, tail)
		execErr = logErr
		a.sendAck(cmd.ID, execErr == nil, execErr, map[string]any{"lines": lines})
		return
	case "files_list":
		path, _ := cmd.Payload["path"].(string)
		files, listErr := docker.ListFiles(ctx, cmd.ServerID, path)
		execErr = listErr
		a.sendAck(cmd.ID, execErr == nil, execErr, map[string]any{"files": files})
		return
	case "files_read":
		path, _ := cmd.Payload["path"].(string)
		encoding, _ := cmd.Payload["encoding"].(string)
		if encoding == "base64" {
			compressPayload := false
			if v, ok := cmd.Payload["compress"]; ok {
				switch b := v.(type) {
				case bool:
					compressPayload = b
				case string:
					compressPayload = b == "1" || b == "true"
				}
			}
			raw, readErr := docker.ReadFileBytes(ctx, cmd.ServerID, path)
			execErr = readErr
			out := raw
			compression := ""
			if execErr == nil && compressPayload && len(raw) > 1024 {
				var buf bytes.Buffer
				gz := gzip.NewWriter(&buf)
				if _, err := gz.Write(raw); err == nil && gz.Close() == nil && buf.Len() < len(raw) {
					out = buf.Bytes()
					compression = "gzip"
				}
			}
			a.sendAck(cmd.ID, execErr == nil, execErr, map[string]any{
				"content_base64":  base64.StdEncoding.EncodeToString(out),
				"path":            path,
				"size_bytes":      len(raw),
				"wire_size_bytes": len(out),
				"compression":     compression,
			})
			return
		}
		content, readErr := docker.ReadFile(ctx, cmd.ServerID, path)
		execErr = readErr
		a.sendAck(cmd.ID, execErr == nil, execErr, map[string]any{"content": content, "path": path})
		return
	case "files_write":
		path, _ := cmd.Payload["path"].(string)
		encoding, _ := cmd.Payload["encoding"].(string)
		if encoding == "base64" {
			contentBase64, _ := cmd.Payload["content_base64"].(string)
			raw, decodeErr := base64.StdEncoding.DecodeString(contentBase64)
			if decodeErr != nil {
				a.sendAck(cmd.ID, false, decodeErr, nil)
				return
			}
			execErr = docker.WriteFileBytes(ctx, cmd.ServerID, path, raw)
			a.sendAck(cmd.ID, execErr == nil, execErr, map[string]any{
				"path":       path,
				"size_bytes": len(raw),
			})
			return
		}
		content, _ := cmd.Payload["content"].(string)
		execErr = docker.WriteFile(ctx, cmd.ServerID, path, content)
		a.sendAck(cmd.ID, execErr == nil, execErr, map[string]any{"path": path})
		return
	case protocol.ActionFilesWriteBinary:
		path, _ := cmd.Payload["path"].(string)
		if path == "" {
			a.sendAck(cmd.ID, false, os.ErrInvalid, nil)
			return
		}
		a.pendingBinaryWrites[cmd.ID] = binaryWritePending{
			serverID: cmd.ServerID,
			path:     path,
		}
		return
	case "files_mkdir":
		path, _ := cmd.Payload["path"].(string)
		execErr = docker.Mkdir(ctx, cmd.ServerID, path)
		a.sendAck(cmd.ID, execErr == nil, execErr, nil)
		return
	case "files_delete":
		path, _ := cmd.Payload["path"].(string)
		execErr = docker.DeletePath(ctx, cmd.ServerID, path)
		a.sendAck(cmd.ID, execErr == nil, execErr, nil)
		return
	case "backup_create":
		name, _ := cmd.Payload["name"].(string)
		filename, size, backupErr := docker.CreateBackup(ctx, cmd.ServerID, name)
		execErr = backupErr
		a.sendAck(cmd.ID, execErr == nil, execErr, map[string]any{
			"filename": filename, "size_bytes": size,
		})
		return
	case "backup_restore":
		name, _ := cmd.Payload["name"].(string)
		execErr = docker.RestoreBackup(ctx, cmd.ServerID, name)
		a.sendAck(cmd.ID, execErr == nil, execErr, nil)
		return
	case "game_query":
		gameID, _ := cmd.Payload["game_id"].(string)
		limits, _ := cmd.Payload["limits"].(map[string]any)
		port := docker.IntFromPayload(cmd.Payload["port"])
		query := docker.QueryGameServer(ctx, cmd.ServerID, gameID, limits, port)
		a.sendAck(cmd.ID, true, nil, map[string]any{
			"online": query.Online, "max_players": query.MaxPlayers,
			"online_players": query.OnlinePlayers, "players_online": query.PlayersOnline,
			"current_map": query.CurrentMap, "runtime_status": query.RuntimeStatus,
			"ping_ms": query.PingMS, "tps": query.TPS,
		})
		return
	case "stats":
		limits, _ := cmd.Payload["limits"].(map[string]any)
		diskLimit := docker.IntFromPayload(limits["disk_mb"])
		st, statErr := docker.CollectStatsExtended(ctx, cmd.ServerID, diskLimit)
		if statErr != nil {
			a.sendAck(cmd.ID, false, statErr, nil)
			return
		}
		memPct := 0.0
		if st.MemLimitMB > 0 {
			memPct = float64(st.MemUsedMB) / float64(st.MemLimitMB) * 100
		}
		diskPct := 0.0
		if st.DiskTotalMB > 0 && st.DiskUsedMB > 0 {
			diskPct = float64(st.DiskUsedMB) / float64(st.DiskTotalMB) * 100
		}
		a.sendAck(cmd.ID, true, nil, map[string]any{
			"ok": true, "cpu_percent": st.CPUPct, "mem_percent": memPct,
			"mem_used_mb": st.MemUsedMB, "mem_limit_mb": st.MemLimitMB,
			"disk_used_mb": st.DiskUsedMB, "disk_total_mb": st.DiskTotalMB, "disk_percent": diskPct,
			"started_at": st.StartedAt, "uptime": st.Uptime,
		})
		return
	case "console_command":
		gameID, _ := cmd.Payload["game_id"].(string)
		command, _ := cmd.Payload["command"].(string)
		out, cmdErr := docker.ExecConsoleCommand(ctx, cmd.ServerID, gameID, command)
		a.sendAck(cmd.ID, cmdErr == nil, cmdErr, map[string]any{"output": out})
		return
	case "archive_cache_fetch":
		res, err := docker.FetchArchiveToCache(ctx, cmd.Payload)
		a.sendAck(cmd.ID, err == nil, err, res)
		return
	case "plugins_apply":
		execErr = docker.ApplyPlugin(ctx, cmd.ServerID, cmd.Payload)
		a.sendAck(cmd.ID, execErr == nil, execErr, map[string]any{"ok": execErr == nil})
		return
	case "maps_apply":
		gameID, _ := cmd.Payload["game_id"].(string)
		execErr = docker.ApplyMap(ctx, cmd.ServerID, gameID, cmd.Payload)
		a.sendAck(cmd.ID, execErr == nil, execErr, map[string]any{"ok": execErr == nil})
		return
	case "mysql_create_db":
		execErr = docker.MySQLCreateDB(ctx, cmd.Payload)
		a.sendAck(cmd.ID, execErr == nil, execErr, map[string]any{"ok": execErr == nil})
		return
	case "mysql_delete_db":
		execErr = docker.MySQLDeleteDB(ctx, cmd.Payload)
		a.sendAck(cmd.ID, execErr == nil, execErr, map[string]any{"ok": execErr == nil})
		return
	case "mysql_migrate_db":
		execErr = docker.MySQLMigrateDB(ctx, cmd.ServerID, cmd.Payload)
		a.sendAck(cmd.ID, execErr == nil, execErr, map[string]any{"ok": execErr == nil})
		return
	case "mysql_list_catalog":
		result, listErr := docker.MySQLListCatalog(ctx, cmd.Payload)
		a.sendAck(cmd.ID, listErr == nil, listErr, result)
		return
	case "mysql_create_database":
		execErr = docker.MySQLCreateDatabase(ctx, cmd.Payload)
		a.sendAck(cmd.ID, execErr == nil, execErr, map[string]any{"ok": execErr == nil})
		return
	case "mysql_delete_database":
		execErr = docker.MySQLDeleteDatabase(ctx, cmd.Payload)
		a.sendAck(cmd.ID, execErr == nil, execErr, map[string]any{"ok": execErr == nil})
		return
	case "mysql_create_user":
		execErr = docker.MySQLCreateUser(ctx, cmd.Payload)
		a.sendAck(cmd.ID, execErr == nil, execErr, map[string]any{"ok": execErr == nil})
		return
	case "mysql_delete_user":
		execErr = docker.MySQLDeleteUser(ctx, cmd.Payload)
		a.sendAck(cmd.ID, execErr == nil, execErr, map[string]any{"ok": execErr == nil})
		return
	case "mysql_reset_user_password":
		execErr = docker.MySQLResetUserPassword(ctx, cmd.Payload)
		a.sendAck(cmd.ID, execErr == nil, execErr, map[string]any{"ok": execErr == nil})
		return
	case protocol.ActionCronSync:
		jobs, parseErr := docker.DecodeCronJobs(cmd.Payload["jobs"])
		if parseErr != nil {
			a.sendAck(cmd.ID, false, parseErr, nil)
			return
		}
		execErr = docker.SyncCron(ctx, cmd.ServerID, jobs)
		a.sendAck(cmd.ID, execErr == nil, execErr, map[string]any{"jobs": len(jobs)})
		return
	case protocol.ActionFirewallSync:
		rules, parseErr := docker.DecodeFirewallRules(cmd.Payload["rules"])
		if parseErr != nil {
			a.sendAck(cmd.ID, false, parseErr, nil)
			return
		}
		execErr = docker.SyncFirewall(ctx, cmd.ServerID, rules)
		a.sendAck(cmd.ID, execErr == nil, execErr, map[string]any{"rules": len(rules)})
		return
	case protocol.ActionPortsSync:
		ports := docker.DecodeExtraPorts(cmd.Payload["ports"])
		changed, syncErr := docker.SyncExtraPorts(cmd.ServerID, ports)
		if syncErr != nil {
			a.sendAck(cmd.ID, false, syncErr, nil)
			return
		}
		restarted := false
		if changed && docker.Status(ctx, cmd.ServerID) == "running" {
			name, _ := cmd.Payload["name"].(string)
			gameID, _ := cmd.Payload["game_id"].(string)
			dockerImage, _ := cmd.Payload["docker_image"].(string)
			limits, _ := cmd.Payload["limits"].(map[string]any)
			primaryPort := docker.IntFromPayload(cmd.Payload["primary_port"])
			bindIP, _ := cmd.Payload["bind_ip"].(string)
			execErr = docker.Restart(ctx, cmd.ServerID, name, gameID, limits, dockerImage, primaryPort, bindIP)
			restarted = execErr == nil
			status := docker.DetailedStatus(ctx, cmd.ServerID)
			errMsg := ""
			if execErr != nil {
				errMsg = execErr.Error()
			}
			a.sendStatus(cmd.ServerID, status, errMsg)
		}
		a.sendAck(cmd.ID, execErr == nil, execErr, map[string]any{
			"ports": len(ports), "changed": changed, "restarted": restarted,
		})
		return
	case protocol.ActionConsole:
		sessionID, _ := cmd.Payload["session_id"].(string)
		execErr = docker.StartConsoleStream(ctx, cmd.ServerID, sessionID, func(line string) {
			out, _ := json.Marshal(protocol.ConsoleOutputMessage{
				Type: protocol.MsgConsoleOutput, SessionID: sessionID,
				ServerID: cmd.ServerID, Data: line,
			})
			_ = a.send(out)
		})
	case protocol.ActionConsoleIn:
		sessionID, _ := cmd.Payload["session_id"].(string)
		input, _ := cmd.Payload["data"].(string)
		execErr = docker.ConsoleInput(ctx, cmd.ServerID, input)
		_ = sessionID
	}

	if cmd.Action == protocol.ActionConsole || cmd.Action == protocol.ActionConsoleIn {
		a.sendAck(cmd.ID, execErr == nil, execErr, nil)
		return
	}

	a.sendStatus(cmd.ServerID, status, "")
	a.sendAck(cmd.ID, execErr == nil, execErr, nil)
}

func (a *Agent) sendInstallProgress(serverID, stage string, percent int, message, line string) {
	msg, _ := json.Marshal(protocol.InstallProgressMessage{
		Type: protocol.MsgInstallProgress, ServerID: serverID,
		Stage: stage, Percent: percent, Message: message, Line: line,
	})
	_ = a.send(msg)
}

func (a *Agent) sendStatus(serverID, status, errMsg string) {
	msg, _ := json.Marshal(protocol.ServerStatusMessage{
		Type: protocol.MsgServerStatus, ServerID: serverID, Status: status, Error: errMsg,
	})
	_ = a.send(msg)
}

func (a *Agent) sendAck(cmdID string, ok bool, err error, result map[string]any) {
	ack := protocol.AckMessage{Type: protocol.MsgAck, CommandID: cmdID, OK: ok, Result: result}
	if err != nil {
		ack.Error = err.Error()
	}
	msg, _ := json.Marshal(ack)
	_ = a.send(msg)
}

func (a *Agent) installAndStart(serverID, name, gameID string, limits map[string]any, dockerImage string, primaryPort int, bindIP string, spec docker.InstallSpec) {
	ctx := context.Background()
	report := func(stage string, percent int, message, line string) {
		a.sendInstallProgress(serverID, stage, percent, message, line)
	}
	if err := docker.Install(ctx, serverID, spec, report); err != nil {
		log.Printf("install failed for %s: %v", serverID, err)
		a.sendStatus(serverID, "error", "установка файлов сервера: "+err.Error())
		return
	}
	docker.ReapplyOwnership(ctx, serverID)
	if err := docker.Start(ctx, serverID, name, gameID, limits, dockerImage, primaryPort, bindIP); err != nil {
		log.Printf("start after install failed for %s: %v", serverID, err)
		a.sendStatus(serverID, "error", err.Error())
		return
	}
	status := docker.DetailedStatus(ctx, serverID)
	errMsg := ""
	if status == "error" {
		errMsg = docker.ContainerError(ctx, serverID)
	}
	a.sendStatus(serverID, status, errMsg)
}

func (a *Agent) updateAndStart(serverID, name, gameID string, limits map[string]any, dockerImage string, primaryPort int, bindIP string, spec docker.InstallSpec) {
	ctx := context.Background()
	wasRunning := docker.Status(ctx, serverID) == "running"
	if wasRunning {
		if err := docker.Stop(ctx, serverID); err != nil {
			log.Printf("update: остановка сервера %s: %v", serverID, err)
			a.sendStatus(serverID, "error", "остановка перед обновлением: "+err.Error())
			return
		}
	}

	report := func(stage string, percent int, message, line string) {
		a.sendInstallProgress(serverID, stage, percent, message, line)
	}
	if err := docker.Update(ctx, serverID, spec, report); err != nil {
		log.Printf("update failed for %s: %v", serverID, err)
		a.sendStatus(serverID, "error", "обновление файлов сервера: "+err.Error())
		return
	}
	docker.ReapplyOwnership(ctx, serverID)

	if !wasRunning {
		a.sendStatus(serverID, "stopped", "")
		return
	}
	if err := docker.Start(ctx, serverID, name, gameID, limits, dockerImage, primaryPort, bindIP); err != nil {
		log.Printf("update: запуск после обновления %s: %v", serverID, err)
		a.sendStatus(serverID, "error", err.Error())
		return
	}
	status := docker.DetailedStatus(ctx, serverID)
	errMsg := ""
	if status == "error" {
		errMsg = docker.ContainerError(ctx, serverID)
	}
	a.sendStatus(serverID, status, errMsg)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
