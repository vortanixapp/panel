package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"

	"github.com/vortanixapp/panel/internal/console/relayclient"
	"github.com/vortanixapp/panel/pkg/paneljwt"
	"github.com/vortanixapp/panel/pkg/protocol"
)

const (
	consolePongWait   = 70 * time.Second
	consolePingPeriod = 25 * time.Second
	consoleWriteWait  = 10 * time.Second
	consoleReadLimit  = 64 << 10

	consoleLeasePeriod = 60 * time.Second
	consoleRelayWait   = 10 * time.Second
)

type Handler struct {
	redis *redis.Client
	relay *relayclient.Client
	jwt   *paneljwt.Verifier
	upgr  websocket.Upgrader
}

func New(rdb *redis.Client, relay *relayclient.Client, jwt *paneljwt.Verifier) *Handler {
	return &Handler{
		redis: rdb, relay: relay, jwt: jwt,
		upgr: websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }},
	}
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/health", h.Health)
	r.Get("/v1/console", h.ConsoleWS)
	r.Get("/v1/dashboard/stream", h.DashboardWS)
	return r
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) ConsoleWS(w http.ResponseWriter, r *http.Request) {
	ticket := r.URL.Query().Get("ticket")
	if ticket == "" {
		writeError(w, http.StatusBadRequest, "ticket required")
		return
	}
	ctx := r.Context()
	var data map[string]any
	ok, err := h.getJSON(ctx, "console:ticket:"+ticket, &data)
	if err != nil || !ok {
		writeError(w, http.StatusUnauthorized, "invalid or expired ticket")
		return
	}
	session := relayclient.ConsoleSessionRequest{}
	session.SessionID, _ = data["session_id"].(string)
	session.ServerID, _ = data["server_id"].(string)
	session.NodeID, _ = data["node_id"].(string)
	gameID, _ := data["game_id"].(string)
	canCommand, _ := data["can_command"].(bool)
	_ = h.redis.Del(ctx, "console:ticket:"+ticket)

	conn, err := h.upgr.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	conn.SetReadLimit(consoleReadLimit)
	_ = conn.SetReadDeadline(time.Now().Add(consolePongWait))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(consolePongWait))
	})

	var writeMu sync.Mutex
	write := func(kind int, payload []byte) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		_ = conn.SetWriteDeadline(time.Now().Add(consoleWriteWait))
		return conn.WriteMessage(kind, payload)
	}
	notice := func(code string) {
		_ = write(websocket.TextMessage, protocol.EncodeConsoleFrame(protocol.ConsoleFrameNotice, code, ""))
	}

	pubsub := h.redis.Subscribe(ctx, protocol.ConsoleChannel(session.SessionID))
	defer pubsub.Close()
	if _, err := pubsub.Receive(ctx); err != nil {
		notice("stream_unavailable")
		return
	}

	defer h.stopConsole(ctx, session)
	if err := h.startConsole(ctx, session); err != nil {
		notice("node_offline")
		return
	}

	streamCtx, stopStream := context.WithCancel(ctx)
	defer stopStream()
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer conn.Close()
		ping := time.NewTicker(consolePingPeriod)
		defer ping.Stop()
		lease := time.NewTicker(consoleLeasePeriod)
		defer lease.Stop()
		messages := pubsub.Channel()
		for {
			select {
			case msg, ok := <-messages:
				if !ok {
					return
				}
				if write(websocket.TextMessage, []byte(msg.Payload)) != nil {
					return
				}
			case <-ping.C:
				if write(websocket.PingMessage, nil) != nil {
					return
				}
			case <-lease.C:
				_ = h.startConsole(streamCtx, session)
			case <-streamCtx.Done():
				return
			}
		}
	}()

	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			break
		}
		var input struct {
			Type string `json:"type"`
			Data string `json:"data"`
		}
		if json.Unmarshal(raw, &input) != nil || input.Type != "input" {
			continue
		}
		if !canCommand {
			notice("no_permission")
			continue
		}
		if strings.TrimSpace(input.Data) == "" {
			continue
		}
		sendCtx, cancel := context.WithTimeout(ctx, consoleRelayWait)
		err = h.relay.ConsoleInput(sendCtx, relayclient.ConsoleInputRequest{
			SessionID: session.SessionID, ServerID: session.ServerID, NodeID: session.NodeID,
			GameID: gameID, Data: input.Data,
		})
		cancel()
		if err != nil {
			notice("node_offline")
		}
	}
	stopStream()
	<-done
}

func (h *Handler) startConsole(ctx context.Context, session relayclient.ConsoleSessionRequest) error {
	ctx, cancel := context.WithTimeout(ctx, consoleRelayWait)
	defer cancel()
	return h.relay.StartConsole(ctx, session)
}

func (h *Handler) stopConsole(ctx context.Context, session relayclient.ConsoleSessionRequest) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), consoleRelayWait)
	defer cancel()
	_ = h.relay.StopConsole(ctx, session)
}

func (h *Handler) DashboardWS(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if token == "" {
		token = r.URL.Query().Get("token")
	}
	if token == "" {
		if c, err := r.Cookie("vtx_access"); err == nil && c != nil {
			token = strings.TrimSpace(c.Value)
		}
	}
	if token == "" {
		writeError(w, http.StatusUnauthorized, "missing token")
		return
	}
	claims, err := h.jwt.ParseAccess(token)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid token")
		return
	}
	switch claims.Role {
	case "owner", "admin", "support":
	default:
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	conn, err := h.upgr.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	ctx := r.Context()
	pubsub := h.redis.Subscribe(ctx, protocol.TenantEventsChannel())
	defer pubsub.Close()

	conn.SetReadLimit(consoleReadLimit)
	_ = conn.SetReadDeadline(time.Now().Add(consolePongWait))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(consolePongWait))
	})
	go func() {
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				_ = conn.Close()
				return
			}
		}
	}()

	ticker := time.NewTicker(consolePingPeriod)
	defer ticker.Stop()
	messages := pubsub.Channel()
	for {
		select {
		case msg, ok := <-messages:
			if !ok {
				return
			}
			_ = conn.SetWriteDeadline(time.Now().Add(consoleWriteWait))
			if conn.WriteMessage(websocket.TextMessage, []byte(msg.Payload)) != nil {
				return
			}
		case <-ticker.C:
			_ = conn.SetWriteDeadline(time.Now().Add(consoleWriteWait))
			if conn.WriteMessage(websocket.PingMessage, nil) != nil {
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

func (h *Handler) getJSON(ctx context.Context, key string, dest any) (bool, error) {
	val, err := h.redis.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, json.Unmarshal([]byte(val), dest)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
