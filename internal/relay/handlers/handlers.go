package handlers

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/vortanixapp/panel/internal/relay/events"
	"github.com/vortanixapp/panel/internal/relay/hub"
	"github.com/vortanixapp/panel/internal/relay/metricsclient"
	"github.com/vortanixapp/panel/pkg/notify"
	"github.com/vortanixapp/panel/pkg/protocol"
	"github.com/vortanixapp/panel/pkg/secretbox"
)

type Handler struct {
	db      *pgxpool.Pool
	redis   *redis.Client
	hub     *hub.Hub
	waiter  *hub.CommandWaiter
	secret  string
	metrics *metricsclient.Client
	upgr    websocket.Upgrader

	panelURL string
}

func (h *Handler) WithPanelURL(u string) *Handler {
	h.panelURL = strings.TrimRight(u, "/")
	return h
}

func New(db *pgxpool.Pool, rdb *redis.Client, h *hub.Hub, secret, metricsURL string) *Handler {
	var mc *metricsclient.Client
	if metricsURL != "" {
		mc = metricsclient.New(metricsURL, secret)
	}
	return &Handler{
		db: db, redis: rdb, hub: h, waiter: hub.NewCommandWaiter(), secret: secret, metrics: mc,
		upgr: websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }},
	}
}

func (h *Handler) dbFor(context.Context) (*pgxpool.Pool, error) {
	return h.db, nil
}

func execLogged(ctx context.Context, db *pgxpool.Pool, sql string, args ...any) {
	if _, err := db.Exec(ctx, sql, args...); err != nil {
		q := strings.Join(strings.Fields(sql), " ")
		if len(q) > 70 {
			q = q[:70]
		}
		log.Printf("relay: запись не прошла (%s…): %v", q, err)
	}
}

func (h *Handler) lookupNode(ctx context.Context, token string) (nodeID string, err error) {
	err = h.db.QueryRow(ctx, `
		SELECT id::text FROM core.nodes
		WHERE agent_token_hash = $1
		   OR (agent_token_hash IS NULL AND agent_token = $2)
	`, secretbox.TokenHash(token), token).Scan(&nodeID)
	if err != nil {
		return "", err
	}
	return nodeID, nil
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/health", h.Health)
	r.Post("/internal/v1/nodes/{nodeID}/command", h.InternalCommand)
	r.Post("/internal/v1/nodes/{nodeID}/command/sync", h.InternalCommandSync)
	r.Post("/internal/v1/nodes/{nodeID}/command/sync-binary", h.InternalCommandSyncBinary)
	r.Post("/internal/v1/console/start", h.InternalConsoleStart)
	r.Post("/internal/v1/console/input", h.InternalConsoleInput)
	r.Post("/internal/v1/console/stop", h.InternalConsoleStop)
	return r
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if err := h.db.Ping(ctx); err != nil {
		writeError(w, http.StatusServiceUnavailable, "database unavailable")
		return
	}
	if err := h.redis.Ping(ctx).Err(); err != nil {
		writeError(w, http.StatusServiceUnavailable, "redis unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) AgentConnect(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if token == "" {
		writeError(w, http.StatusUnauthorized, "missing agent token")
		return
	}

	ctx := r.Context()
	nodeID, err := h.lookupNode(ctx, token)
	if err != nil {
		log.Printf("agent token lookup failed remote=%s: %v", r.RemoteAddr, err)
		writeError(w, http.StatusUnauthorized, "invalid agent token")
		return
	}

	db, err := h.dbFor(ctx)
	if err != nil {
		log.Printf("agent connect отклонён node=%s: %v", nodeID, err)
		writeError(w, http.StatusServiceUnavailable, "база арендатора недоступна")
		return
	}

	conn, err := h.upgr.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("agent upgrade failed: %v", err)
		return
	}

	log.Printf("agent connected node=%s remote=%s", nodeID, r.RemoteAddr)

	agent := &hub.AgentConn{
		NodeID: nodeID, DB: db, Conn: conn,
		Send: make(chan hub.OutboundMessage, 64),
	}
	h.hub.Register(agent)
	if _, err := db.Exec(ctx, `
		UPDATE core.nodes SET status = 'online', last_seen_at = now() WHERE id = $1
	`, nodeID); err != nil {
		log.Printf("relay: статус узла %s не записан: %v", nodeID, err)
	}
	if _, err := db.Exec(ctx, `
		INSERT INTO core.node_daemons ( node_id, status, last_seen_at)
		VALUES ( $1, 'online', now())
		ON CONFLICT (node_id) DO UPDATE SET status = 'online', last_seen_at = now(), updated_at = now()
	`, nodeID); err != nil {
		log.Printf("relay: демон узла %s не записан: %v", nodeID, err)
	}
	_ = h.redis.Del(ctx, "panel:nodes")
	events.PublishTenantEvent(ctx, h.redis, protocol.TenantEvent{
		Type: "node.status", NodeID: nodeID, Status: "online",
	})

	go agent.WritePump()
	h.readPump(agent)
}

func (h *Handler) readPump(c *hub.AgentConn) {
	defer func() {
		if !h.hub.Unregister(c.NodeID, c) {
			return
		}
		ctx := context.Background()
		var nodeName string
		if c.DB.QueryRow(ctx, `
			UPDATE core.nodes SET status = 'offline' WHERE id = $1 AND status <> 'offline'
			RETURNING COALESCE(fqdn, '')
		`, c.NodeID).Scan(&nodeName) == nil {
			h.notifyNodeOwners(ctx, c.DB, c.NodeID, nodeName)
			emitWebhook(ctx, c.DB, "node.offline", map[string]any{
				"node_id":   c.NodeID,
				"node_name": nodeName,
			})
		}
		execLogged(ctx, c.DB, `
			UPDATE core.node_daemons SET status = 'offline', updated_at = now() WHERE node_id = $1
		`, c.NodeID)
		_ = h.redis.Del(ctx, "panel:nodes")
		events.PublishTenantEvent(ctx, h.redis, protocol.TenantEvent{
			Type: "node.status", NodeID: c.NodeID, Status: "offline",
		})
	}()

	_ = c.Conn.SetReadDeadline(time.Now().Add(hub.PongWait))
	c.Conn.SetPongHandler(func(string) error {
		return c.Conn.SetReadDeadline(time.Now().Add(hub.PongWait))
	})

	for {
		_, data, err := c.Conn.ReadMessage()
		if err != nil {
			log.Printf("agent disconnected node=%s: %v", c.NodeID, err)
			return
		}
		_ = c.Conn.SetReadDeadline(time.Now().Add(hub.PongWait))
		h.handleAgentMessage(c, data)
	}
}

func (h *Handler) handleAgentMessage(c *hub.AgentConn, data []byte) {
	var env map[string]any
	if err := json.Unmarshal(data, &env); err != nil {
		if len(data) > 200 {
			data = data[:200]
		}
		log.Printf("relay: неразобранное сообщение от узла %s: %v (%q)", c.NodeID, err, data)
		return
	}
	typ, _ := env["type"].(string)
	ctx := context.Background()

	switch typ {
	case protocol.MsgHello:
		helloNode, _ := env["node_id"].(string)
		if helloNode != "" && helloNode != c.NodeID {
			log.Printf("agent hello node_id mismatch: connection=%s hello=%s", c.NodeID, helloNode)
		}
		h.saveDaemonState(ctx, c, env)
	case protocol.MsgHeartbeat:
		tag, err := c.DB.Exec(ctx,
			`UPDATE core.nodes SET last_seen_at = now(), status = 'online' WHERE id = $1`, c.NodeID)
		if err != nil {
			log.Printf("relay: heartbeat узла %s не записан: %v", c.NodeID, err)
		} else if tag.RowsAffected() == 0 {
			log.Printf("relay: heartbeat узла %s не нашёл строку в базе", c.NodeID)
		}
		h.saveDaemonState(ctx, c, env)
	case protocol.MsgServerStatus:
		serverID, _ := env["server_id"].(string)
		status, _ := env["status"].(string)
		errMsg, _ := env["error"].(string)
		if serverID == "" || status == "" {
			return
		}
		var prevStatus string
		_ = c.DB.QueryRow(ctx, `
			SELECT COALESCE(status, '') FROM core.servers WHERE id = $1
		`, serverID).Scan(&prevStatus)
		runtimeStatus := status
		provStatus := ""
		switch status {
		case "running":
			runtimeStatus = "running"
			provStatus = "ready"
		case "error":
			runtimeStatus = "offline"
			provStatus = "failed"
		case "stopped":
			runtimeStatus = "stopped"
		case "installing":
			runtimeStatus = "offline"
			provStatus = "provisioning"
		}
		if provStatus == "ready" {
			var prevProv, name string
			err := c.DB.QueryRow(ctx, `
				WITH prev AS (
					SELECT provisioning_status FROM core.servers
					WHERE id = $4
				)
				UPDATE core.servers s
				SET status = $1, runtime_status = $2, provisioning_status = $3, provisioning_error = NULL
				FROM prev
				WHERE s.id = $4
				RETURNING prev.provisioning_status, s.name
			`, status, runtimeStatus, provStatus, serverID).Scan(&prevProv, &name)
			if err == nil && prevProv != "ready" {
				h.notifyServerOwner(ctx, c.DB, serverID, notify.Event{
					Kind:   notify.KindServerReady,
					Title:  "Сервер готов",
					Body:   "Сервер «" + name + "» установлен и запущен — можно подключаться.",
					Action: h.serverAction("Открыть сервер", serverID, ""),
					Meta:   map[string]any{"server_id": serverID},
				})
			}
		} else if provStatus == "failed" {
			if errMsg == "" {
				errMsg = "container failed to start"
			}
			var prevProv, name string
			err := c.DB.QueryRow(ctx, `
				WITH prev AS (
					SELECT provisioning_status FROM core.servers
					WHERE id = $5
				)
				UPDATE core.servers s
				SET status = $1, runtime_status = $2, provisioning_status = $3, provisioning_error = $4
				FROM prev
				WHERE s.id = $5
				RETURNING prev.provisioning_status, s.name
			`, status, runtimeStatus, provStatus, errMsg, serverID).Scan(&prevProv, &name)
			if err == nil && prevProv != "failed" {
				e := notify.Event{
					Kind:   notify.KindServerDown,
					Title:  "Сервер остановился с ошибкой",
					Body:   "Сервер «" + name + "» перестал работать. Причина: " + errMsg,
					Action: h.serverAction("Открыть сервер", serverID, ""),
					Meta:   map[string]any{"server_id": serverID, "error": errMsg},
				}
				if prevProv == "provisioning" || prevProv == "pending" {
					e.Kind = notify.KindServerFailed
					e.Title = "Установка не удалась"
					e.Body = "Сервер «" + name + "» не удалось установить. Причина: " + errMsg +
						". Попробуйте переустановить его или напишите в поддержку."
				}
				h.notifyServerOwner(ctx, c.DB, serverID, e)
			}
		} else if provStatus != "" {
			execLogged(ctx, c.DB, `
				UPDATE core.servers
				SET status = $1, runtime_status = $2, provisioning_status = $3
				WHERE id = $4
			`, status, runtimeStatus, provStatus, serverID)
		} else {
			execLogged(ctx, c.DB, `
				UPDATE core.servers SET status = $1, runtime_status = $2 WHERE id = $3
			`, status, runtimeStatus, serverID)
		}
		_ = h.redis.Set(ctx, "srv:"+serverID+":status", status, 30*time.Second)
		_ = h.redis.Del(ctx, "panel:servers")
		events.PublishTenantEvent(ctx, h.redis, protocol.TenantEvent{
			Type: "server.status", ServerID: serverID, Status: status,
		})
		if prevStatus != status {
			emitWebhook(ctx, c.DB, "server.status", map[string]any{
				"server_id":       serverID,
				"node_id":         c.NodeID,
				"status":          status,
				"runtime_status":  runtimeStatus,
				"previous_status": prevStatus,
				"error":           errMsg,
			})
		}
	case protocol.MsgInstallProgress:
		serverID, _ := env["server_id"].(string)
		if serverID == "" {
			return
		}
		stage, _ := env["stage"].(string)
		message, _ := env["message"].(string)
		line, _ := env["line"].(string)
		percent := intNum(env["percent"])

		if percent >= 0 {
			progress := map[string]any{"percent": percent, "stage": stage}
			if message != "" {
				progress["message"] = message
			}
			_ = h.setJSON(ctx, "srv:"+serverID+":provisioning_progress", progress, 30*time.Minute)
		}

		if line != "" {
			key := "install:log:" + serverID
			_ = h.redis.RPush(ctx, key, line).Err()
			_ = h.redis.LTrim(ctx, key, -500, -1).Err()
			_ = h.redis.Expire(ctx, key, 24*time.Hour).Err()
		}

		ev := protocol.TenantEvent{
			Type: "server.install_progress", ServerID: serverID, Status: stage,
			Message: message,
		}
		if percent >= 0 {
			p := percent
			ev.Percent = &p
		}
		events.PublishTenantEvent(ctx, h.redis, ev)
	case protocol.MsgMetrics:
		serverID, _ := env["server_id"].(string)
		cpu, _ := env["cpu_pct"].(float64)
		memUsed := intNum(env["mem_used_mb"])
		memLimit := intNum(env["mem_limit_mb"])
		if serverID == "" {
			return
		}
		if pendingServerOperation(ctx, h.redis, serverID) {
			if h.metrics != nil {
				h.metrics.IngestAsync(metricsclient.IngestRequest{
					ServerID: serverID,
					CPUPct:   cpu, MemUsedMB: memUsed, MemLimitMB: memLimit,
				})
			}
			return
		}

		tag, _ := c.DB.Exec(ctx, `
			UPDATE core.servers
			SET runtime_status = 'running',
				status = CASE
					WHEN status IN ('stopped', 'offline', 'starting', 'stopping', 'error') THEN 'running'
					ELSE status
				END,
				provisioning_status = CASE
					WHEN provisioning_status IN ('pending', 'provisioning', 'failed') THEN 'ready'
					ELSE provisioning_status
				END,
				provisioning_error = CASE
					WHEN provisioning_status IN ('pending', 'provisioning', 'failed') THEN NULL
					ELSE provisioning_error
				END
			WHERE id = $1
			  AND (
				runtime_status IS DISTINCT FROM 'running'
				OR status IN ('stopped', 'offline', 'starting', 'stopping', 'error')
				OR provisioning_status IN ('pending', 'provisioning', 'failed')
			  )
		`, serverID)
		if tag.RowsAffected() > 0 {
			log.Printf("metrics reconcile runtime running server=%s", serverID)
		}
		_ = h.redis.Set(ctx, "srv:"+serverID+":status", "running", 45*time.Second)
		_ = h.redis.Del(ctx, "panel:servers")
		if h.metrics != nil {
			h.metrics.IngestAsync(metricsclient.IngestRequest{
				ServerID: serverID,
				CPUPct:   cpu, MemUsedMB: memUsed, MemLimitMB: memLimit,
			})
		} else {
			_ = events.StoreMetricPoint(ctx, h.redis, serverID, cpu, memUsed, memLimit)
		}
		events.PublishTenantEvent(ctx, h.redis, protocol.TenantEvent{
			Type: "server.metrics", ServerID: serverID,
			Metrics: map[string]any{"cpu_pct": cpu, "mem_used_mb": memUsed, "mem_limit_mb": memLimit},
		})
	case protocol.MsgConsoleOutput:
		sessionID, _ := env["session_id"].(string)
		dataStr, _ := env["data"].(string)
		if sessionID != "" {
			events.PublishConsoleOutput(ctx, h.redis, sessionID, dataStr)
		}
	case protocol.MsgAck:
		var ack protocol.AckMessage
		if json.Unmarshal(data, &ack) == nil && ack.CommandID != "" {
			h.waiter.Complete(ack)
		}
	default:
		log.Printf("relay: неизвестный тип сообщения от узла %s: %q", c.NodeID, typ)
	}
}

func intNum(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	default:
		return 0
	}
}

func secretEqual(got, want string) bool {
	return subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1
}

func (h *Handler) setJSON(ctx context.Context, key string, v any, ttl time.Duration) error {
	raw, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return h.redis.Set(ctx, key, raw, ttl).Err()
}

func (h *Handler) InternalCommand(w http.ResponseWriter, r *http.Request) {
	if !secretEqual(r.Header.Get("X-Internal-Secret"), h.secret) {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	nodeID := chi.URLParam(r, "nodeID")
	var req protocol.InternalCommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.CommandID == "" {
		req.CommandID = uuid.NewString()
	}
	if err := h.sendToNode(nodeID, req.CommandID, req.Action, req.ServerID, req.Payload); err != nil {
		writeError(w, http.StatusBadGateway, "node offline")
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"command_id": req.CommandID, "status": "sent"})
}

func (h *Handler) InternalCommandSync(w http.ResponseWriter, r *http.Request) {
	if !secretEqual(r.Header.Get("X-Internal-Secret"), h.secret) {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	nodeID := chi.URLParam(r, "nodeID")
	var req protocol.InternalCommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.CommandID == "" {
		req.CommandID = uuid.NewString()
	}
	waitCh := h.waiter.Register(req.CommandID)
	defer h.waiter.Cancel(req.CommandID)
	if err := h.sendToNode(nodeID, req.CommandID, req.Action, req.ServerID, req.Payload); err != nil {
		writeError(w, http.StatusBadGateway, "node offline")
		return
	}
	ack, ok := h.waiter.Wait(req.CommandID, waitCh, 12*time.Second)
	if !ok {
		writeError(w, http.StatusGatewayTimeout, "agent command timeout")
		return
	}
	if !ack.OK {
		msg := ack.Error
		if msg == "" {
			msg = "agent command failed"
		}
		writeError(w, http.StatusBadGateway, msg)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"command_id": req.CommandID,
		"ok":         true,
		"result":     ack.Result,
	})
}

func (h *Handler) InternalCommandSyncBinary(w http.ResponseWriter, r *http.Request) {
	if !secretEqual(r.Header.Get("X-Internal-Secret"), h.secret) {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	nodeID := chi.URLParam(r, "nodeID")
	if err := r.ParseMultipartForm(256 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "invalid multipart")
		return
	}
	reqJSON := r.FormValue("request")
	if strings.TrimSpace(reqJSON) == "" {
		writeError(w, http.StatusBadRequest, "request required")
		return
	}
	var req protocol.InternalCommandRequest
	if err := json.Unmarshal([]byte(reqJSON), &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if req.CommandID == "" {
		req.CommandID = uuid.NewString()
	}
	file, _, err := r.FormFile("binary")
	if err != nil {
		writeError(w, http.StatusBadRequest, "binary file required")
		return
	}
	defer file.Close()
	raw, err := io.ReadAll(file)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed reading binary")
		return
	}
	waitCh := h.waiter.Register(req.CommandID)
	defer h.waiter.Cancel(req.CommandID)
	if err := h.sendToNode(nodeID, req.CommandID, req.Action, req.ServerID, req.Payload); err != nil {
		writeError(w, http.StatusBadGateway, "node offline")
		return
	}
	packet := append([]byte(req.CommandID+"\n"), raw...)
	if err := h.hub.SendBinary(nodeID, packet); err != nil {
		writeError(w, http.StatusBadGateway, "node offline")
		return
	}
	ack, ok := h.waiter.Wait(req.CommandID, waitCh, 20*time.Second)
	if !ok {
		writeError(w, http.StatusGatewayTimeout, "agent command timeout")
		return
	}
	if !ack.OK {
		msg := ack.Error
		if msg == "" {
			msg = "agent command failed"
		}
		writeError(w, http.StatusBadGateway, msg)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"command_id": req.CommandID,
		"ok":         true,
		"result":     ack.Result,
	})
}

type consoleStartRequest struct {
	SessionID string `json:"session_id"`
	ServerID  string `json:"server_id"`
	NodeID    string `json:"node_id"`
}

func (h *Handler) InternalConsoleStart(w http.ResponseWriter, r *http.Request) {
	if !secretEqual(r.Header.Get("X-Internal-Secret"), h.secret) {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	var req consoleStartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	cmdID := uuid.NewString()
	err := h.sendToNode(req.NodeID, cmdID, protocol.ActionConsole, req.ServerID, map[string]any{
		"session_id": req.SessionID,
	})
	if err != nil {
		writeError(w, http.StatusBadGateway, "node offline")
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "started"})
}

type consoleInputRequest struct {
	SessionID string `json:"session_id"`
	ServerID  string `json:"server_id"`
	NodeID    string `json:"node_id"`
	Data      string `json:"data"`
}

func (h *Handler) InternalConsoleInput(w http.ResponseWriter, r *http.Request) {
	if !secretEqual(r.Header.Get("X-Internal-Secret"), h.secret) {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	var req consoleInputRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	cmdID := uuid.NewString()
	if err := h.sendToNode(req.NodeID, cmdID, protocol.ActionConsoleIn, req.ServerID, map[string]any{
		"session_id": req.SessionID, "data": req.Data,
	}); err != nil {
		writeError(w, http.StatusBadGateway, "node offline")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) InternalConsoleStop(w http.ResponseWriter, r *http.Request) {
	if !secretEqual(r.Header.Get("X-Internal-Secret"), h.secret) {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	var req consoleStartRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	writeJSON(w, http.StatusOK, map[string]string{"status": "stopped"})
}

func (h *Handler) sendToNode(nodeID, cmdID, action, serverID string, payload map[string]any) error {
	cmd := protocol.CommandMessage{
		Type: protocol.MsgCommand, ID: cmdID,
		Action: action, ServerID: serverID, Payload: payload,
	}
	data, _ := json.Marshal(cmd)
	return h.hub.SendCommand(nodeID, data)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func (h *Handler) saveDaemonState(ctx context.Context, c *hub.AgentConn, env map[string]any) {
	version, _ := env["version"].(string)
	execLogged(ctx, c.DB, `
		INSERT INTO core.node_daemons (node_id, status, version, last_seen_at)
		VALUES ($1, 'online', NULLIF($2, ''), now())
		ON CONFLICT (node_id) DO UPDATE SET
			status       = 'online',
			version      = COALESCE(NULLIF($2, ''), core.node_daemons.version),
			last_seen_at = now(),
			updated_at   = now()
	`, c.NodeID, version)

	h.saveHostStats(ctx, c, env)
}

func (h *Handler) saveHostStats(ctx context.Context, c *hub.AgentConn, env map[string]any) {
	raw, ok := env["host"].(map[string]any)
	if !ok || len(raw) == 0 {
		return
	}
	metrics := map[string]string{
		"cpu_percent":   "cpu_usage",
		"ram_percent":   "ram_usage",
		"disk_percent":  "disk_usage",
		"ram_total_mb":  "ram_total_mb",
		"ram_used_mb":   "ram_used_mb",
		"disk_total_mb": "disk_total_mb",
		"disk_used_mb":  "disk_used_mb",
	}
	if q, ok := raw["disk_quota"].(bool); ok {
		v := 0.0
		if q {
			v = 1
		}
		execLogged(ctx, c.DB, `
			INSERT INTO core.node_metrics (node_id, metric_type, value, measured_at)
			VALUES ($1, 'disk_quota', $2, now())
		`, c.NodeID, v)
	}

	for field, metricType := range metrics {
		v, ok := raw[field].(float64)
		if !ok {
			continue
		}
		execLogged(ctx, c.DB, `
			INSERT INTO core.node_metrics (node_id, metric_type, value, measured_at)
			VALUES ($1, $2, $3, now())
		`, c.NodeID, metricType, v)
	}
}
