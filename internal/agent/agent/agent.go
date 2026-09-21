package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/subtle"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"

	"github.com/vortanixapp/panel/internal/agent/docker"
	"github.com/vortanixapp/panel/internal/agent/selfupdate"
	"github.com/vortanixapp/panel/pkg/buildinfo"
	"github.com/vortanixapp/panel/pkg/protocol"
	"github.com/vortanixapp/panel/pkg/relaytls"
)

type Agent struct {
	relayURL            string
	relayPin            string
	token               string
	nodeID              string
	version             string
	startedAt           time.Time
	conn                *websocket.Conn
	writeMu             sync.Mutex
	out                 outbox
	pendingBinaryWrites map[string]binaryWritePending
	updating            atomic.Bool
	reportOnce          sync.Once
	disp                *dispatcher
	ops                 opsRegistry
	metricsBusy         atomic.Bool
	serversRunning      atomic.Int64
	serversTotal        atomic.Int64
}

const (
	writeWait = 15 * time.Second

	pongWait = 90 * time.Second

	handshakeTimeout = 20 * time.Second

	reconnectMin = 2 * time.Second
	reconnectMax = 30 * time.Second

	binaryWriteTTL = 2 * time.Minute
)

var errNotConnected = errors.New("нет связи с relay")

func (a *Agent) send(payload []byte) error {
	a.writeMu.Lock()
	defer a.writeMu.Unlock()
	if a.conn == nil {
		return errNotConnected
	}
	return a.writeLocked(payload)
}

func (a *Agent) writeLocked(payload []byte) error {
	conn := a.conn
	_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
	err := conn.WriteMessage(websocket.TextMessage, payload)
	if err != nil {
		log.Printf("отправка не удалась, рвём связь: %v", err)
		_ = conn.Close()
		a.conn = nil
	}
	return err
}

func (a *Agent) sendReliable(key string, ack bool, live, queued []byte) {
	a.writeMu.Lock()
	defer a.writeMu.Unlock()
	if a.conn != nil && a.writeLocked(live) == nil {
		return
	}
	if queued == nil {
		queued = live
	}
	a.out.push(key, ack, queued)
}

type binaryWritePending struct {
	serverID string
	path     string
	at       time.Time
}

func New() *Agent {
	a := &Agent{
		relayURL:            os.Getenv("RELAY_URL"),
		relayPin:            strings.TrimSpace(os.Getenv("RELAY_PIN")),
		token:               os.Getenv("AGENT_TOKEN"),
		nodeID:              os.Getenv("NODE_ID"),
		version:             buildinfo.Current(),
		startedAt:           time.Now(),
		pendingBinaryWrites: map[string]binaryWritePending{},
	}
	a.disp = newDispatcher(func(j job, err error, code string) {
		a.sendAckCode(j.id, false, err, nil, code)
	})
	return a
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
		a.reportOnce.Do(func() { go a.reportUpgradeResult() })
		a.loop()
		time.Sleep(reconnectMin)
	}
}

func (a *Agent) connect() error {
	header := http.Header{}
	header.Set("Authorization", "Bearer "+a.token)
	dialer := &websocket.Dialer{HandshakeTimeout: handshakeTimeout}
	if a.relayPin != "" {
		dialer.TLSClientConfig = pinnedTLSConfig(a.relayPin)
	}
	conn, resp, err := dialer.Dial(a.relayURL, header)
	if err != nil {
		if resp != nil {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
			_ = resp.Body.Close()
			return fmt.Errorf("%w: relay ответил %s: %s", err, resp.Status,
				strings.TrimSpace(string(body)))
		}
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
	pending := a.out.drain()
	for i, it := range pending {
		_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
		if err := conn.WriteMessage(websocket.TextMessage, it.data); err != nil {
			a.out.restore(pending[i:])
			a.writeMu.Unlock()
			_ = conn.Close()
			return err
		}
	}
	a.conn = conn
	a.writeMu.Unlock()

	log.Printf("connected to relay as node=%s", a.nodeID)
	if len(pending) > 0 {
		log.Printf("доставлено %d сообщений, накопленных без связи", len(pending))
	}
	return nil
}

func pinnedTLSConfig(pin string) *tls.Config {
	want := strings.TrimSpace(pin)
	return &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: true,
		VerifyPeerCertificate: func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
			for _, raw := range rawCerts {
				cert, err := x509.ParseCertificate(raw)
				if err != nil {
					continue
				}
				got := relaytls.PinOf(cert)
				if subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1 {
					return nil
				}
			}
			return fmt.Errorf("сертификат relay не совпал с отпечатком из установки")
		},
	}
}

func (a *Agent) loop() {
	a.writeMu.Lock()
	conn := a.conn
	a.writeMu.Unlock()
	if conn == nil {
		return
	}
	defer func() {
		a.writeMu.Lock()
		if a.conn == conn {
			a.conn = nil
		}
		a.writeMu.Unlock()
		_ = conn.Close()
	}()

	done := make(chan struct{})
	defer close(done)
	go a.heartbeat(done)
	go a.metricsLoop(done)
	for {
		mt, data, err := conn.ReadMessage()
		if err != nil {
			log.Printf("связь потеряна: %v", err)
			return
		}
		_ = conn.SetReadDeadline(time.Now().Add(pongWait))
		a.expireBinaryWrites()
		if mt == websocket.BinaryMessage {
			a.handleBinaryMessage(data)
			continue
		}
		a.handleCommand(data)
	}
}

func (a *Agent) expireBinaryWrites() {
	for id, p := range a.pendingBinaryWrites {
		if time.Since(p.at) > binaryWriteTTL {
			delete(a.pendingBinaryWrites, id)
			a.sendAck(id, false, fmt.Errorf("не дождались данных файла %s", p.path), nil)
		}
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
	a.disp.serial("srv:"+pending.serverID, job{
		id: cmdID, action: protocol.ActionFilesWriteBinary,
		timeout: actionTimeout(protocol.ActionFilesWriteBinary),
		run: func(ctx context.Context) {
			execErr := docker.WriteFileBytes(ctx, pending.serverID, pending.path, raw)
			a.sendAck(cmdID, execErr == nil, execErr, map[string]any{
				"path":       pending.path,
				"size_bytes": len(raw),
			})
		},
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
		payload := map[string]any{
			"type":    protocol.MsgHeartbeat,
			"node_id": a.nodeID,
			"version": a.version,
			"host":    docker.CollectHostStats(),
		}
		if self, ok := docker.CollectAgentStats(); ok {
			self.UptimeSec = int64(time.Since(a.startedAt).Seconds())
			payload["agent"] = self
		}
		running, queued := a.disp.stats()
		payload["queue"] = map[string]any{"running": running, "queued": queued}
		payload["servers"] = map[string]any{
			"total":   a.serversTotal.Load(),
			"running": a.serversRunning.Load(),
		}
		msg, _ := json.Marshal(payload)
		if err := a.send(msg); err != nil {
			return
		}
	}
}

func (a *Agent) metricsLoop(done <-chan struct{}) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	tick := 0
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
		}
		if !a.metricsBusy.CompareAndSwap(false, true) {
			continue
		}
		refreshTotal := tick%4 == 0
		tick++
		go func() {
			defer a.metricsBusy.Store(false)
			ctx := context.Background()
			if refreshTotal {
				listCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
				if ids, err := docker.ListManagedServerIDs(listCtx); err == nil {
					a.serversTotal.Store(int64(len(ids)))
				}
				cancel()
			}
			stats, err := docker.CollectRunningStats(ctx)
			if err != nil {
				return
			}
			a.serversRunning.Store(int64(len(stats)))
			for id, st := range stats {
				msg, _ := json.Marshal(protocol.MetricsMessage{
					Type: protocol.MsgMetrics, ServerID: id,
					CPUPct: st.CPUPct, MemUsedMB: st.MemUsedMB, MemLimitMB: st.MemLimitMB,
				})
				if a.send(msg) != nil {
					return
				}
			}
		}()
	}
}

func (a *Agent) handleCommand(data []byte) {
	var cmd protocol.CommandMessage
	if err := json.Unmarshal(data, &cmd); err != nil || cmd.Type != protocol.MsgCommand {
		return
	}
	switch cmd.Action {
	case protocol.ActionFilesWriteBinary:
		path, _ := cmd.Payload["path"].(string)
		if path == "" {
			a.sendAck(cmd.ID, false, os.ErrInvalid, nil)
			return
		}
		a.pendingBinaryWrites[cmd.ID] = binaryWritePending{serverID: cmd.ServerID, path: path, at: time.Now()}
		return
	case protocol.ActionConsole:
		sessionID, _ := cmd.Payload["session_id"].(string)
		if detach, _ := cmd.Payload["detach"].(bool); detach {
			docker.StopConsole(sessionID)
		} else {
			leased, _ := cmd.Payload["lease"].(bool)
			docker.AttachConsole(cmd.ServerID, sessionID, leased, a.consoleEmitter(cmd.ServerID, sessionID))
		}
		a.sendAck(cmd.ID, true, nil, nil)
		return
	case protocol.ActionConsoleIn:
		sessionID, _ := cmd.Payload["session_id"].(string)
		input, _ := cmd.Payload["data"].(string)
		gameID, _ := cmd.Payload["game_id"].(string)
		go a.consoleInput(cmd.ID, cmd.ServerID, sessionID, gameID, input)
		return
	case protocol.ActionAgentUpdate:
		a.startSelfUpdate(cmd)
		return
	}

	j := job{
		id: cmd.ID, action: cmd.Action, timeout: actionTimeout(cmd.Action),
		run: func(ctx context.Context) { a.run(ctx, cmd) },
	}
	switch {
	case cmd.Action == "logs" || cmd.Action == "stats" || cmd.Action == "game_query":
		if cmd.ServerID == "" {
			a.disp.nodeReadJob(j)
			return
		}
		a.disp.serverRead(cmd.ServerID, j)
	case cmd.Action == "mysql_migrate_db" && cmd.ServerID != "":
		a.disp.serial("srv:"+cmd.ServerID, j)
	case strings.HasPrefix(cmd.Action, "mysql_"), cmd.Action == "archive_cache_fetch", cmd.ServerID == "":
		a.disp.serial("node:ctl", j)
	default:
		a.disp.serial("srv:"+cmd.ServerID, j)
	}
}

func (a *Agent) startSelfUpdate(cmd protocol.CommandMessage) {
	image, _ := cmd.Payload["image"].(string)
	target, _ := cmd.Payload["version"].(string)
	if strings.TrimSpace(image) == "" {
		a.sendAck(cmd.ID, false, fmt.Errorf("не указан образ агента"), nil)
		return
	}
	if !a.updating.CompareAndSwap(false, true) {
		a.sendAck(cmd.ID, false, fmt.Errorf("обновление агента уже идёт"), nil)
		return
	}
	a.sendAck(cmd.ID, true, nil, map[string]any{"accepted": true, "version": a.version})
	go func() {
		defer a.updating.Store(false)
		selfupdate.Start(context.Background(), strings.TrimSpace(image), target, a.sendUpdateStatus)
	}()
}

func (a *Agent) run(ctx context.Context, cmd protocol.CommandMessage) {
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
		defer a.ops.begin(cmd.ServerID, "destroy")()
		execErr = docker.Destroy(ctx, cmd.ServerID)
		if wipe, _ := cmd.Payload["wipe"].(bool); wipe && execErr == nil {
			execErr = docker.WipeData(cmd.ServerID)
		}
		for _, err := range docker.CleanupServerTraces(ctx, cmd.ServerID) {
			log.Printf("очистка следов сервера %s: %v", cmd.ServerID, err)
		}
		a.sendStatus(cmd.ServerID, status, "")
		a.sendAck(cmd.ID, execErr == nil, execErr, nil)
		return
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
		defer a.ops.begin(cmd.ServerID, "backup")()
		name, _ := cmd.Payload["name"].(string)
		filename, size, backupErr := docker.CreateBackup(ctx, cmd.ServerID, name)
		execErr = backupErr
		a.sendAck(cmd.ID, execErr == nil, execErr, map[string]any{
			"filename": filename, "size_bytes": size,
		})
		return
	case "backup_restore":
		defer a.ops.begin(cmd.ServerID, "restore")()
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
		defer a.ops.begin(cmd.ServerID, "plugins")()
		execErr = docker.ApplyPlugin(ctx, cmd.ServerID, cmd.Payload)
		a.sendAck(cmd.ID, execErr == nil, execErr, map[string]any{"ok": execErr == nil})
		return
	case "maps_apply":
		defer a.ops.begin(cmd.ServerID, "maps")()
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
		defer a.ops.begin(cmd.ServerID, "mysql_migrate")()
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
	default:
		a.sendAckCode(cmd.ID, false, fmt.Errorf("агент не знает команду %q", cmd.Action), nil, codeUnsupportedAction)
	}
}

const codeUnsupportedAction = "unsupported_action"

func (a *Agent) consoleEmitter(serverID, sessionID string) docker.ConsoleEmit {
	return func(kind, code, data string) {
		if kind == protocol.ConsoleFrameOutput {
			kind = ""
		}
		out, _ := json.Marshal(protocol.ConsoleOutputMessage{
			Type: protocol.MsgConsoleOutput, SessionID: sessionID, ServerID: serverID,
			Data: data, Kind: kind, Code: code,
		})
		_ = a.send(out)
	}
}

func (a *Agent) consoleInput(cmdID, serverID, sessionID, gameID, input string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	emit := a.consoleEmitter(serverID, sessionID)
	reply, err := docker.SendConsoleCommand(ctx, serverID, gameID, input)
	switch {
	case errors.Is(err, docker.ErrConsoleStopped):
		emit(protocol.ConsoleFrameNotice, "input_stopped", "")
	case errors.Is(err, docker.ErrConsoleNoStdin):
		emit(protocol.ConsoleFrameNotice, "input_no_stdin", "")
	case err != nil:
		emit(protocol.ConsoleFrameNotice, "input_failed", err.Error())
	case reply.Output != "":
		emit(protocol.ConsoleFrameReply, reply.Via, reply.Output)
	}
	a.sendAck(cmdID, err == nil, err, map[string]any{"via": reply.Via})
}

func (a *Agent) sendInstallProgress(serverID, stage string, percent int, message, line string) {
	progress := protocol.InstallProgressMessage{
		Type: protocol.MsgInstallProgress, ServerID: serverID,
		Stage: stage, Percent: percent, Message: message, Line: line,
	}
	live, _ := json.Marshal(progress)
	progress.Line = ""
	queued, _ := json.Marshal(progress)
	a.sendReliable("progress:"+serverID, false, live, queued)
}

func (a *Agent) sendStatus(serverID, status, errMsg string) {
	msg, _ := json.Marshal(protocol.ServerStatusMessage{
		Type: protocol.MsgServerStatus, ServerID: serverID, Status: status, Error: errMsg,
	})
	a.sendReliable("status:"+serverID, false, msg, nil)
}

func (a *Agent) sendAck(cmdID string, ok bool, err error, result map[string]any) {
	a.sendAckCode(cmdID, ok, err, result, "")
}

func (a *Agent) sendAckCode(cmdID string, ok bool, err error, result map[string]any, code string) {
	ack := protocol.AckMessage{Type: protocol.MsgAck, CommandID: cmdID, OK: ok, Result: result, Code: code}
	if err != nil {
		ack.Error = err.Error()
	}
	msg, _ := json.Marshal(ack)
	a.sendReliable("", true, msg, nil)
}

func (a *Agent) sendUpdateStatus(stage, target, errMsg string) {
	if errMsg != "" {
		log.Printf("обновление агента: %s", errMsg)
	}
	msg, _ := json.Marshal(protocol.AgentUpdateMessage{
		Type: protocol.MsgAgentUpdate, Stage: stage, Target: target, Error: errMsg,
	})
	a.sendReliable("update", false, msg, nil)
}

func (a *Agent) reportUpgradeResult() {
	res, ok := selfupdate.CollectResult(context.Background())
	if !ok {
		return
	}
	a.sendUpdateStatus(res.State, res.Target, res.Error)
}

func (a *Agent) installAndStart(serverID, name, gameID string, limits map[string]any, dockerImage string, primaryPort int, bindIP string, spec docker.InstallSpec) {
	defer a.ops.begin(serverID, "install")()
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
	defer a.ops.begin(serverID, "update")()
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
