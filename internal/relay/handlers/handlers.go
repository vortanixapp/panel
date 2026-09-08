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
	"github.com/vortanixapp/panel/pkg/tenantpools"
)

type Handler struct {
	db      *pgxpool.Pool
	pools   *tenantpools.Pools
	redis   *redis.Client
	hub     *hub.Hub
	waiter  *hub.CommandWaiter
	secret  string
	metrics *metricsclient.Client
	upgr    websocket.Upgrader

	// panelURL нужен кнопкам в оповещениях: то же уведомление уходит письмом и
	// в Telegram, где относительная ссылка никуда не ведёт.
	panelURL string
}

// WithPanelURL задаёт адрес панели для ссылок в оповещениях.
func (h *Handler) WithPanelURL(u string) *Handler {
	h.panelURL = strings.TrimRight(u, "/")
	return h
}

func New(db *pgxpool.Pool, pools *tenantpools.Pools, rdb *redis.Client, h *hub.Hub, secret, metricsURL string) *Handler {
	var mc *metricsclient.Client
	if metricsURL != "" {
		mc = metricsclient.New(metricsURL, secret)
	}
	return &Handler{
		db: db, pools: pools, redis: rdb, hub: h, waiter: hub.NewCommandWaiter(), secret: secret, metrics: mc,
		upgr: websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }},
	}
}

// dbFor отдаёт базу арендатора или ошибку. Центральной базы в ответе быть не
// может: там нет ни этой ноды, ни её серверов, и запись туда выглядела бы
// успешной, ничего не изменив.
func (h *Handler) dbFor(ctx context.Context, tenantID string) (*pgxpool.Pool, error) {
	if h.pools == nil {
		return h.db, nil
	}
	return h.pools.ForTenant(ctx, tenantID)
}

// execLogged выполняет запись и не даёт ошибке потеряться. Молчаливый сбой
// здесь неотличим снаружи от пропавшей связи с нодой — именно так heartbeat
// «не доходил», хотя приходил исправно.
func execLogged(ctx context.Context, db *pgxpool.Pool, sql string, args ...any) {
	if _, err := db.Exec(ctx, sql, args...); err != nil {
		q := strings.Join(strings.Fields(sql), " ")
		if len(q) > 70 {
			q = q[:70]
		}
		log.Printf("relay: запись не прошла (%s…): %v", q, err)
	}
}

// lookupNode ищет ноду по токену агента во всех базах: арендатор до этого
// момента неизвестен, агент присылает только токен.
func (h *Handler) lookupNode(ctx context.Context, token string) (nodeID, tenantID string, err error) {
	pools := []*pgxpool.Pool{h.db}
	if h.pools != nil {
		pools = h.pools.All(ctx)
	}
	for _, pool := range pools {
		err = pool.QueryRow(ctx, `
			SELECT id::text, tenant_id::text FROM core.nodes
			WHERE agent_token_hash = $1
			   OR (agent_token_hash IS NULL AND agent_token = $2)
		`, secretbox.TokenHash(token), token).Scan(&nodeID, &tenantID)
		if err == nil {
			return nodeID, tenantID, nil
		}
	}
	return "", "", err
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
	nodeID, tenantID, err := h.lookupNode(ctx, token)
	if err != nil {
		log.Printf("agent token lookup failed remote=%s: %v", r.RemoteAddr, err)
		writeError(w, http.StatusUnauthorized, "invalid agent token")
		return
	}

	// База разрешается до рукопожатия: без неё соединение бессмысленно —
	// статусы, метрики и ответы на команды писать было бы некуда. Отказ здесь
	// честнее молчаливой работы вхолостую.
	db, err := h.dbFor(ctx, tenantID)
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

	log.Printf("agent connected node=%s tenant=%s remote=%s", nodeID, tenantID, r.RemoteAddr)

	agent := &hub.AgentConn{
		NodeID: nodeID, TenantID: tenantID, DB: db, Conn: conn,
		Send: make(chan hub.OutboundMessage, 64),
	}
	h.hub.Register(agent)
	if _, err := db.Exec(ctx, `
		UPDATE core.nodes SET status = 'online', last_seen_at = now() WHERE id = $1
	`, nodeID); err != nil {
		log.Printf("relay: статус узла %s не записан: %v", nodeID, err)
	}
	if _, err := db.Exec(ctx, `
		INSERT INTO core.node_daemons (tenant_id, node_id, status, last_seen_at)
		VALUES ($1, $2, 'online', now())
		ON CONFLICT (node_id) DO UPDATE SET status = 'online', last_seen_at = now(), updated_at = now()
	`, tenantID, nodeID); err != nil {
		log.Printf("relay: демон узла %s не записан: %v", nodeID, err)
	}
	_ = h.redis.Del(ctx, "t:"+tenantID+":nodes")
	events.PublishTenantEvent(ctx, h.redis, tenantID, protocol.TenantEvent{
		Type: "node.status", NodeID: nodeID, Status: "online",
	})

	go agent.WritePump()
	h.readPump(agent)
}

func (h *Handler) readPump(c *hub.AgentConn) {
	defer func() {
		// Если сеанс уже заменён новым, статус не трогаем: агент на связи, и
		// «offline» от закрывшегося соединения означал бы обратное.
		if !h.hub.Unregister(c.NodeID, c) {
			return
		}
		ctx := context.Background()
		var nodeName string
		if c.DB.QueryRow(ctx, `
			UPDATE core.nodes SET status = 'offline' WHERE id = $1 AND status <> 'offline'
			RETURNING COALESCE(fqdn, '')
		`, c.NodeID).Scan(&nodeName) == nil {
			// Сообщаем только о переходе в офлайн: условие в UPDATE отсекает
			// повторные срабатывания, иначе владельцы получали бы уведомление на
			// каждое переподключение агента.
			h.notifyNodeOwners(ctx, c.DB, c.TenantID, c.NodeID, nodeName)
			emitWebhook(ctx, c.DB, c.TenantID, "node.offline", map[string]any{
				"node_id":   c.NodeID,
				"node_name": nodeName,
			})
		}
		execLogged(ctx, c.DB, `
			UPDATE core.node_daemons SET status = 'offline', updated_at = now() WHERE node_id = $1
		`, c.NodeID)
		_ = h.redis.Del(ctx, "t:"+c.TenantID+":nodes")
		events.PublishTenantEvent(ctx, h.redis, c.TenantID, protocol.TenantEvent{
			Type: "node.status", NodeID: c.NodeID, Status: "offline",
		})
	}()

	// Дедлайн продлевается каждым pong'ом и каждым сообщением агента. Без него
	// оборванное соединение (пропало питание, отключили сеть) висело бы вечно,
	// и нода в панели оставалась бы «в сети».
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
		// Молча отбрасывать нельзя: неразобранное сообщение выглядит как
		// полное отсутствие связи, а искать причину не по чему.
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
		// Ошибку записи не глушим: снаружи неудачный UPDATE неотличим от
		// неприходящего heartbeat, и нода «уходит в офлайн» без объяснений.
		tag, err := c.DB.Exec(ctx,
			`UPDATE core.nodes SET last_seen_at = now(), status = 'online' WHERE id = $1`, c.NodeID)
		if err != nil {
			log.Printf("relay: heartbeat узла %s не записан: %v", c.NodeID, err)
		} else if tag.RowsAffected() == 0 {
			log.Printf("relay: heartbeat узла %s не нашёл строку в базе арендатора %s", c.NodeID, c.TenantID)
		}
		h.saveDaemonState(ctx, c, env)
	case protocol.MsgServerStatus:
		serverID, _ := env["server_id"].(string)
		status, _ := env["status"].(string)
		errMsg, _ := env["error"].(string)
		if serverID == "" || status == "" {
			return
		}
		// Прежний статус нужен только для вебхука: агент повторяет отчёт на
		// каждое действие питания, и без сравнения подписчик получал бы
		// «сервер сменил статус» на неизменившийся статус.
		var prevStatus string
		_ = c.DB.QueryRow(ctx, `
			SELECT COALESCE(status, '') FROM core.servers WHERE id = $1 AND tenant_id = $2
		`, serverID, c.TenantID).Scan(&prevStatus)
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
			// Прежнее состояние нужно, чтобы отличить настоящее событие от
			// повторного отчёта: агент присылает статус на каждое действие
			// питания, и без этой проверки «сервер готов» приходило бы клиенту
			// после каждого перезапуска.
			var prevProv, name string
			err := c.DB.QueryRow(ctx, `
				WITH prev AS (
					SELECT provisioning_status FROM core.servers
					WHERE id = $4 AND tenant_id = $5
				)
				UPDATE core.servers s
				SET status = $1, runtime_status = $2, provisioning_status = $3, provisioning_error = NULL
				FROM prev
				WHERE s.id = $4 AND s.tenant_id = $5
				RETURNING prev.provisioning_status, s.name
			`, status, runtimeStatus, provStatus, serverID, c.TenantID).Scan(&prevProv, &name)
			if err == nil && prevProv != "ready" {
				h.notifyServerOwner(ctx, c.DB, c.TenantID, serverID, notify.Event{
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
					WHERE id = $5 AND tenant_id = $6
				)
				UPDATE core.servers s
				SET status = $1, runtime_status = $2, provisioning_status = $3, provisioning_error = $4
				FROM prev
				WHERE s.id = $5 AND s.tenant_id = $6
				RETURNING prev.provisioning_status, s.name
			`, status, runtimeStatus, provStatus, errMsg, serverID, c.TenantID).Scan(&prevProv, &name)
			if err == nil && prevProv != "failed" {
				// Установка не удалась и падение уже работавшего сервера — разные
				// события: в первом случае клиент ждёт готовности, во втором у
				// него только что перестало работать то, что работало.
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
				h.notifyServerOwner(ctx, c.DB, c.TenantID, serverID, e)
			}
		} else if provStatus != "" {
			execLogged(ctx, c.DB, `
				UPDATE core.servers
				SET status = $1, runtime_status = $2, provisioning_status = $3
				WHERE id = $4 AND tenant_id = $5
			`, status, runtimeStatus, provStatus, serverID, c.TenantID)
		} else {
			execLogged(ctx, c.DB, `
				UPDATE core.servers SET status = $1, runtime_status = $2 WHERE id = $3 AND tenant_id = $4
			`, status, runtimeStatus, serverID, c.TenantID)
		}
		_ = h.redis.Set(ctx, "srv:"+serverID+":status", status, 30*time.Second)
		_ = h.redis.Del(ctx, "t:"+c.TenantID+":servers")
		events.PublishTenantEvent(ctx, h.redis, c.TenantID, protocol.TenantEvent{
			Type: "server.status", ServerID: serverID, Status: status,
		})
		if prevStatus != status {
			emitWebhook(ctx, c.DB, c.TenantID, "server.status", map[string]any{
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
		// Отрицательный процент у агента означает «это просто строка лога,
		// шкалу не трогать» — так его и передаём, не подменяя нулём.
		if percent >= 0 {
			p := percent
			ev.Percent = &p
		}
		events.PublishTenantEvent(ctx, h.redis, c.TenantID, ev)
	case protocol.MsgMetrics:
		serverID, _ := env["server_id"].(string)
		cpu, _ := env["cpu_pct"].(float64)
		memUsed := intNum(env["mem_used_mb"])
		memLimit := intNum(env["mem_limit_mb"])
		if serverID == "" {
			return
		}
		// Метрики не должны воскрешать сервер, который прямо сейчас
		// останавливают или переустанавливают.
		//
		// Агент собирает метрики раз в 15 секунд по всем ещё живым контейнерам,
		// и тик, попавший между командой и её завершением, откатывал состояние
		// обратно: клиент видел «Останавливается» → «Работает» → «Выключен».
		// Признак идущей операции уже есть — переходный статус в кэше, который
		// кладёт обработчик питания; его короткий срок жизни и служит окном
		// ожидания, после которого сверка снова вступает в силу и чинит
		// зависшее состояние.
		if pendingServerOperation(ctx, h.redis, serverID) {
			if h.metrics != nil {
				h.metrics.IngestAsync(metricsclient.IngestRequest{
					TenantID: c.TenantID, ServerID: serverID,
					CPUPct: cpu, MemUsedMB: memUsed, MemLimitMB: memLimit,
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
			WHERE id = $1 AND tenant_id = $2
			  AND (
				runtime_status IS DISTINCT FROM 'running'
				OR status IN ('stopped', 'offline', 'starting', 'stopping', 'error')
				OR provisioning_status IN ('pending', 'provisioning', 'failed')
			  )
		`, serverID, c.TenantID)
		if tag.RowsAffected() > 0 {
			log.Printf("metrics reconcile runtime running server=%s tenant=%s", serverID, c.TenantID)
		}
		_ = h.redis.Set(ctx, "srv:"+serverID+":status", "running", 45*time.Second)
		_ = h.redis.Del(ctx, "t:"+c.TenantID+":servers")
		if h.metrics != nil {
			h.metrics.IngestAsync(metricsclient.IngestRequest{
				TenantID: c.TenantID, ServerID: serverID,
				CPUPct: cpu, MemUsedMB: memUsed, MemLimitMB: memLimit,
			})
		} else {
			_ = events.StoreMetricPoint(ctx, h.redis, serverID, cpu, memUsed, memLimit)
		}
		events.PublishTenantEvent(ctx, h.redis, c.TenantID, protocol.TenantEvent{
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
		INSERT INTO core.node_daemons (tenant_id, node_id, status, version, last_seen_at)
		VALUES ($1, $2, 'online', NULLIF($3, ''), now())
		ON CONFLICT (node_id) DO UPDATE SET
			status       = 'online',
			version      = COALESCE(NULLIF($3, ''), core.node_daemons.version),
			last_seen_at = now(),
			updated_at   = now()
	`, c.TenantID, c.NodeID, version)

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
	// Признак квот приходит булевым, а метрики хранятся числом: 1 и 0 читаются
	// в графиках так же, как остальные, и не требуют отдельной таблицы.
	if q, ok := raw["disk_quota"].(bool); ok {
		v := 0.0
		if q {
			v = 1
		}
		execLogged(ctx, c.DB, `
			INSERT INTO core.node_metrics (tenant_id, node_id, metric_type, value, measured_at)
			VALUES ($1, $2, 'disk_quota', $3, now())
		`, c.TenantID, c.NodeID, v)
	}

	for field, metricType := range metrics {
		v, ok := raw[field].(float64)
		if !ok {
			continue
		}
		execLogged(ctx, c.DB, `
			INSERT INTO core.node_metrics (tenant_id, node_id, metric_type, value, measured_at)
			VALUES ($1, $2, $3, $4, now())
		`, c.TenantID, c.NodeID, metricType, v)
	}
}
