package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/vortanixapp/panel/internal/api/relay"
	"github.com/vortanixapp/panel/pkg/gamecatalog"
	"github.com/vortanixapp/panel/pkg/portalloc"
)

type serverResponse struct {
	ID        string         `json:"id"`
	NodeID    string         `json:"node_id"`
	GameID    string         `json:"game_id"`
	Name      string         `json:"name"`
	Status    string         `json:"status"`
	Config    map[string]any `json:"config"`
	Limits    map[string]any `json:"limits"`
	CreatedAt time.Time      `json:"created_at"`
}

func (h *Handler) ListServers(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ctx := r.Context()
	cacheKey := "t:" + claims.TenantID + ":servers:" + claims.UserID
	var cached []serverResponse
	if hit, err := h.cache.GetJSON(ctx, cacheKey, &cached); err == nil && hit {
		writeJSON(w, http.StatusOK, map[string]any{"servers": cached})
		return
	}

	query := `
		SELECT id::text, node_id::text, game_id, name, status, config, limits, created_at
		FROM core.servers WHERE tenant_id = $1`
	// Это список кабинета, а не админки: сотрудник видит здесь свои серверы,
	// как любой клиент. Чужие серверы показывает раздел администратора, где
	// это и ожидается; раньше кабинет сотрудника показывал серверы всех.
	args := []any{claims.TenantID, claims.UserID}
	query += ` AND (user_id = $2 OR user_id IS NULL) ORDER BY created_at DESC`
	rows, err := h.readerOf(ctx).Query(ctx, query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()

	servers := make([]serverResponse, 0)
	for rows.Next() {
		var s serverResponse
		var cfg, lim []byte
		if err := rows.Scan(&s.ID, &s.NodeID, &s.GameID, &s.Name, &s.Status, &cfg, &lim, &s.CreatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "scan error")
			return
		}
		_ = json.Unmarshal(cfg, &s.Config)
		_ = json.Unmarshal(lim, &s.Limits)
		if s.Config == nil {
			s.Config = map[string]any{}
		}
		if s.Limits == nil {
			s.Limits = map[string]any{}
		}
		if live, ok := h.cache.GetServerStatus(ctx, s.ID); ok {
			s.Status = resolveEffectiveStatus(s.Status, "", &live)
		}
		servers = append(servers, s)
	}
	_ = h.cache.SetJSON(ctx, cacheKey, servers, 5*time.Second)
	writeJSON(w, http.StatusOK, map[string]any{"servers": servers})
}

type createServerRequest struct {
	NodeID string `json:"node_id"`
	Name   string `json:"name"`
	GameID string `json:"game_id"`
}

func (h *Handler) CreateServer(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req createServerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.NodeID == "" || req.Name == "" {
		writeError(w, http.StatusBadRequest, "node_id and name are required")
		return
	}
	if req.GameID == "" {
		req.GameID = "test"
	}

	ctx := r.Context()

	var nodeExists bool
	if err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM core.nodes WHERE id = $1 AND tenant_id = $2)
	`, req.NodeID, claims.TenantID).Scan(&nodeExists); err != nil || !nodeExists {
		writeError(w, http.StatusBadRequest, "node not found")
		return
	}
	if h.nodeInMaintenance(ctx, claims.TenantID, req.NodeID) {
		writeError(w, http.StatusConflict, "нода на техническом обслуживании")
		return
	}
	if reason := h.nodeCapacityReason(ctx, claims.TenantID, req.NodeID, req.GameID); reason != "" {
		writeError(w, http.StatusConflict, reason)
		return
	}

	limits := gamecatalog.DefaultLimits(req.GameID)
	limitsJSON, _ := json.Marshal(limits)
	var id string
	err := h.dbOf(ctx).QueryRow(ctx, `
		INSERT INTO core.servers (tenant_id, node_id, game_id, name, limits, user_id, config)
		VALUES ($1, $2, $3, $4, $5::jsonb, $6, jsonb_build_object('startup_params', $7::text))
		RETURNING id::text
	`, claims.TenantID, req.NodeID, req.GameID, req.Name, limitsJSON, claims.UserID,
		h.defaultStartupParams(ctx, claims.TenantID, req.GameID)).Scan(&id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create server")
		return
	}

	if req.GameID != "test" {
		if _, portErr := portalloc.Assign(ctx, h.dbOf(ctx), claims.TenantID, req.NodeID, id, req.GameID); portErr != nil {
			log.Printf("servers: не удалось выдать порт серверу %s (%s): %v", id, req.GameID, portErr)
		}
	}

	_ = h.cache.InvalidateTenantServers(ctx, claims.TenantID)
	audit(ctx, h.dbOf(ctx), claims.TenantID, claims.UserID, "server.create", "server:"+id, map[string]any{"name": req.Name})

	writeJSON(w, http.StatusCreated, serverResponse{
		ID: id, NodeID: req.NodeID, GameID: req.GameID, Name: req.Name,
		Status: "stopped", Limits: limits,
		CreatedAt: time.Now(),
	})
}

func (h *Handler) GetServer(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := chi.URLParam(r, "id")
	ctx := r.Context()
	var s serverResponse
	var cfg, lim []byte
	err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT id::text, node_id::text, game_id, name, status, config, limits, created_at
		FROM core.servers WHERE id = $1 AND tenant_id = $2
	`, id, claims.TenantID).Scan(&s.ID, &s.NodeID, &s.GameID, &s.Name, &s.Status, &cfg, &lim, &s.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "server not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	_ = json.Unmarshal(cfg, &s.Config)
	_ = json.Unmarshal(lim, &s.Limits)
	if live, ok := h.cache.GetServerStatus(ctx, s.ID); ok {
		s.Status = resolveEffectiveStatus(s.Status, "", &live)
	}
	writeJSON(w, http.StatusOK, s)
}

func (h *Handler) DeleteServer(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := chi.URLParam(r, "id")
	ctx := r.Context()

	// Проверялась только принадлежность сервера арендатору, то есть «сервер
	// того же хостера», а не «сервер этого клиента»: любой зарегистрированный
	// клиент панели удалял сервер соседа, зная его id. Восстановить нечем —
	// мир, конфиги и локальные копии уничтожаются вместе с контейнером.
	if !h.authorizeServerAction(w, r, claims, id, "delete") {
		return
	}

	var nodeID string
	err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT node_id::text FROM core.servers WHERE id = $1 AND tenant_id = $2
	`, id, claims.TenantID).Scan(&nodeID)
	if err != nil {
		writeError(w, http.StatusNotFound, "server not found")
		return
	}

	// Без wipe каталог сервера оставался на ноде навсегда: данные удалённого
	// клиента продолжали лежать на диске, а место не возвращалось — ноды
	// забивались до отказа новых установок. Так же удаляет просроченные
	// серверы сборщик в воркере.
	agentErr := h.relay.SendCommand(ctx, nodeID, relay.CommandRequest{
		CommandID: uuid.NewString(),
		Action:    "destroy",
		ServerID:  id,
		Payload:   map[string]any{"wipe": true},
	})
	force := r.URL.Query().Get("force") == "1"
	if agentErr != nil && !force {
		// Упавшая нода запирала удаление навсегда: запись оставалась, а вместе
		// с ней квота, порт, адрес и напоминания о продлении. Даём выход, но
		// не молча: контейнер на ноде переживёт такое удаление.
		writeError(w, http.StatusConflict,
			"нода недоступна: "+agentErr.Error()+
				". Повторите позже или удалите принудительно (?force=1) — тогда контейнер на ноде придётся снять вручную")
		return
	}

	h.enqueueFtpCleanup(r, claims.TenantID, id, nodeID)

	// Выделенный адрес возвращаем в пул: внешний ключ обнулял ссылку на сервер,
	// но статус оставался «занят», и адрес пропадал из обращения навсегда.
	if _, err := h.dbOf(ctx).Exec(ctx, `
		UPDATE core.ip_pools SET status = 'free', server_id = NULL
		WHERE server_id = $1::uuid AND tenant_id = $2
	`, id, claims.TenantID); err != nil {
		log.Printf("удаление сервера %s: адрес не возвращён в пул: %v", id, err)
	}

	tag, err := h.dbOf(ctx).Exec(ctx, `DELETE FROM core.servers WHERE id = $1 AND tenant_id = $2`, id, claims.TenantID)
	if err != nil || tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "server not found")
		return
	}
	_ = h.cache.InvalidateTenantServers(ctx, claims.TenantID)
	audit(ctx, h.dbOf(ctx), claims.TenantID, claims.UserID, "server.delete", "server:"+id, map[string]any{
		"node_cleaned": agentErr == nil,
		"forced":       force,
	})
	res := map[string]string{"status": "deleted"}
	if agentErr != nil {
		res["warning"] = "контейнер на ноде не снят: нода была недоступна"
	}
	writeJSON(w, http.StatusOK, res)
}

type powerRequest struct {
	Action string `json:"action"`
}

func (h *Handler) PowerServer(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := chi.URLParam(r, "id")
	var req powerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	action := req.Action
	switch action {
	case "start", "stop", "restart", "kill":
	default:
		writeError(w, http.StatusBadRequest, "action must be start, stop, restart or kill")
		return
	}

	// Проверка доступа. Её здесь не было вовсе: сервер искался по паре
	// «идентификатор и арендатор», а владелец не проверялся ни разу — то есть
	// любой вошедший в панель мог выключить чужой сервер. Заодно это приводит в
	// действие права друзей can_start / can_stop / can_restart, которые
	// объявлены и показаны в интерфейсе, но не проверялись нигде, и блокировку
	// администратором.
	if !h.authorizeServerAction(w, r, claims, id, "power_"+action) {
		return
	}

	ctx := r.Context()
	var nodeID, name, gameID, status string
	var limits []byte
	var primaryPort int
	var expired bool
	var startupParams string
	err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT node_id::text, name, game_id, status, limits, COALESCE(primary_port, 0),
		       (expires_at IS NOT NULL AND expires_at < now()) AS expired,
		       COALESCE(config->>'startup_params', '')
		FROM core.servers WHERE id = $1 AND tenant_id = $2
	`, id, claims.TenantID).Scan(&nodeID, &name, &gameID, &status, &limits, &primaryPort, &expired, &startupParams)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "server not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}

	if expired && (action == "start" || action == "restart") {
		writeError(w, http.StatusPaymentRequired, "оплаченный период закончился — продлите аренду, чтобы запустить сервер")
		return
	}

	transient := map[string]string{"start": "starting", "stop": "stopping", "restart": "starting", "kill": "stopping"}[action]
	_, _ = h.dbOf(ctx).Exec(ctx, `UPDATE core.servers SET status = $1 WHERE id = $2`, transient, id)
	_ = h.cache.SetServerStatus(ctx, id, transient)
	_ = h.cache.InvalidateTenantServers(ctx, claims.TenantID)

	var lim map[string]any
	_ = json.Unmarshal(limits, &lim)
	dockerImage := h.resolveDockerImage(ctx, claims.TenantID, id)
	payload := map[string]any{
		"power_action": action,
		"name":         name,
		"game_id":      gameID,
		"limits":       lim,
	}
	if primaryPort > 0 && (action == "start" || action == "restart") {
		payload["primary_port"] = primaryPort
	}
	// Параметры запуска агент кладёт в /data/.vtx/startup_params до старта — файл
	// читают все образы игр. Передаём и при пустом значении: агент тогда удаляет
	// файл, иначе очистка поля в панели не имела бы эффекта.
	if action == "start" || action == "restart" {
		payload["startup_params"] = startupParams
	}
	if bindIP := h.serverBindIP(ctx, claims.TenantID, id); bindIP != "" {
		payload["bind_ip"] = bindIP
	}
	if dockerImage != "" {
		payload["docker_image"] = dockerImage
	}
	if action == "start" || action == "restart" {
		if spec := h.resolveInstallSpec(ctx, claims.TenantID, id); spec != nil {
			payload["install"] = spec
		}
	}
	cmdID := uuid.NewString()
	err = h.relay.SendCommand(ctx, nodeID, relay.CommandRequest{
		CommandID: cmdID,
		Action:    "power",
		ServerID:  id,
		Payload:   payload,
	})
	if err != nil {
		writeError(w, http.StatusBadGateway, "agent unreachable: "+err.Error())
		return
	}

	audit(ctx, h.dbOf(ctx), claims.TenantID, claims.UserID, "server.power", "server:"+id, map[string]any{"action": action})
	writeJSON(w, http.StatusAccepted, map[string]any{
		"server_id":  id,
		"action":     action,
		"status":     transient,
		"command_id": cmdID,
	})
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
