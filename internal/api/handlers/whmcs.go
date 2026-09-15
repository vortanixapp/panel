package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/vortanixapp/panel/internal/api/pricing"
	"github.com/vortanixapp/panel/pkg/buildinfo"
	"github.com/vortanixapp/panel/pkg/gamecatalog"
	"github.com/vortanixapp/panel/pkg/i18n"
	"github.com/vortanixapp/panel/pkg/notify"
)

const whmcsSSOTTL = 2 * time.Minute

var errWHMCSServiceNotFound = errors.New("услуга не найдена в панели")

type whmcsRefusal string

func (e whmcsRefusal) Error() string { return string(e) }

type whmcsClient struct {
	ID        int64  `json:"id"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Locale    string `json:"locale"`
}

type whmcsServiceRequest struct {
	Client       whmcsClient `json:"client"`
	Password     string      `json:"password"`
	TariffID     string      `json:"tariff_id"`
	Game         string      `json:"game"`
	Location     string      `json:"location"`
	Version      string      `json:"version"`
	Name         string      `json:"name"`
	Slots        int         `json:"slots"`
	CPUCores     int         `json:"cpu_cores"`
	RAMGb        int         `json:"ram_gb"`
	DiskGb       int         `json:"disk_gb"`
	Product      string      `json:"product"`
	BillingCycle string      `json:"billing_cycle"`
	NextDueDate  string      `json:"next_due_date"`
}

type whmcsService struct {
	ID             int64
	ClientID       int64
	UserID         string
	ServerID       string
	Status         string
	BlockedByWHMCS bool
}

type whmcsServerPlan struct {
	tariffID  string
	gameID    string
	nodeID    string
	versionID string
	name      string
	limits    map[string]any
}

type whmcsAccount struct {
	userID   string
	email    string
	password string
	created  bool
}

func whmcsIDParam(r *http.Request, name string) (int64, bool) {
	id, err := strconv.ParseInt(strings.TrimSpace(chi.URLParam(r, name)), 10, 64)
	return id, err == nil && id > 0
}

func whmcsDueDate(raw string) any {
	raw = strings.TrimSpace(raw)
	if len(raw) > 10 {
		raw = raw[:10]
	}
	t, err := time.Parse("2006-01-02", raw)
	if err != nil || t.Year() < 2000 {
		return nil
	}
	return t
}

func dateOrNil(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.Format("2006-01-02")
}

func whmcsRedirect(raw, serverID string) string {
	raw = strings.TrimSpace(raw)
	if raw != "" && len(raw) <= 512 && strings.HasPrefix(raw, "/") && !strings.HasPrefix(raw, "//") &&
		!strings.ContainsAny(raw, "\\\r\n") {
		if u, err := url.Parse(raw); err == nil && u.Scheme == "" && u.Host == "" {
			return raw
		}
	}
	if serverID != "" {
		return "/servers/" + serverID
	}
	return "/servers"
}

func requestForServer(r *http.Request, serverID string, body any) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", serverID)
	payload, _ := json.Marshal(body)
	req := r.Clone(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	req.Body = io.NopCloser(bytes.NewReader(payload))
	req.ContentLength = int64(len(payload))
	return req
}

func (h *Handler) loadWHMCSService(ctx context.Context, serviceID int64) (*whmcsService, error) {
	s := &whmcsService{ID: serviceID}
	err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT client_id, COALESCE(user_id::text, ''), COALESCE(server_id::text, ''), status, blocked_by_whmcs
		FROM core.whmcs_services WHERE service_id = $1
	`, serviceID).Scan(&s.ClientID, &s.UserID, &s.ServerID, &s.Status, &s.BlockedByWHMCS)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errWHMCSServiceNotFound
	}
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (h *Handler) whmcsServiceFromRequest(w http.ResponseWriter, r *http.Request) (*whmcsService, bool) {
	serviceID, ok := whmcsIDParam(r, "serviceId")
	if !ok {
		writeError(w, http.StatusBadRequest, "некорректный номер услуги WHMCS")
		return nil, false
	}
	s, err := h.loadWHMCSService(r.Context(), serviceID)
	if errors.Is(err, errWHMCSServiceNotFound) {
		writeError(w, http.StatusNotFound, err.Error())
		return nil, false
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return nil, false
	}
	return s, true
}

func requireWHMCSServer(w http.ResponseWriter, s *whmcsService) bool {
	if s.ServerID == "" {
		writeError(w, http.StatusConflict, "сервер этой услуги в панели не создан")
		return false
	}
	return true
}

func (h *Handler) whmcsServiceInfo(ctx context.Context, serviceID int64) (map[string]any, error) {
	var clientID int64
	var userID, email, serverID, status, reason, product, cycle string
	var due *time.Time
	var createdAt time.Time
	err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT w.client_id, COALESCE(w.user_id::text, ''), COALESCE(u.email, ''), COALESCE(w.server_id::text, ''),
		       w.status, COALESCE(w.suspend_reason, ''), w.product, w.billing_cycle, w.next_due_date, w.created_at
		FROM core.whmcs_services w
		LEFT JOIN core.users u ON u.id = w.user_id
		WHERE w.service_id = $1
	`, serviceID).Scan(&clientID, &userID, &email, &serverID, &status, &reason, &product, &cycle, &due, &createdAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errWHMCSServiceNotFound
	}
	if err != nil {
		return nil, err
	}

	info := map[string]any{
		"service_id":     serviceID,
		"client_id":      clientID,
		"status":         status,
		"suspend_reason": nilIfEmpty(reason),
		"product":        product,
		"billing_cycle":  cycle,
		"next_due_date":  dateOrNil(due),
		"created_at":     createdAt.Format(time.RFC3339),
		"user":           nil,
		"server":         nil,
	}
	if userID != "" {
		info["user"] = map[string]any{"id": userID, "email": email}
	}
	if serverID == "" {
		return info, nil
	}
	item, _, ok := h.getEnrichedServer(ctx, "", rbacRoleAdmin, serverID)
	if !ok {
		return info, nil
	}

	var limitsRaw []byte
	_ = h.dbOf(ctx).QueryRow(ctx, `
		SELECT COALESCE(limits, '{}'::jsonb) FROM core.servers WHERE id = $1
	`, serverID).Scan(&limitsRaw)
	limits := map[string]any{}
	_ = json.Unmarshal(limitsRaw, &limits)
	item["limits"] = limits

	address, _ := item["ip_address"].(string)
	if port := intFromAny(item["port"]); port > 0 && address != "" {
		address = fmt.Sprintf("%s:%d", address, port)
	}
	item["address"] = address
	if used, _, _ := h.latestServerMetricExtras(ctx, serverID); used > 0 {
		item["disk_used_mb"] = used
	}
	if h.frontendURL != "" {
		item["panel_url"] = h.frontendURL + "/servers/" + serverID
		item["admin_url"] = h.frontendURL + "/admin/servers/" + serverID
	}
	info["server"] = item
	return info, nil
}

func (h *Handler) writeWHMCSService(w http.ResponseWriter, r *http.Request, serviceID int64, status int, extra map[string]any) {
	info, err := h.whmcsServiceInfo(r.Context(), serviceID)
	if errors.Is(err, errWHMCSServiceNotFound) {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	for k, v := range extra {
		info[k] = v
	}
	writeJSON(w, status, info)
}

func (h *Handler) WHMCSInfo(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	name := h.tenantSettingString(ctx, "panel.name")
	if name == "" {
		name = "Vortanix"
	}
	var services int
	_ = h.readerOf(ctx).QueryRow(ctx, `
		SELECT COUNT(*) FROM core.whmcs_services WHERE status <> 'terminated'
	`).Scan(&services)
	writeJSON(w, http.StatusOK, map[string]any{
		"panel":     name,
		"version":   buildinfo.Current(),
		"panel_url": h.frontendURL,
		"services":  services,
	})
}

func (h *Handler) WHMCSCatalog(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	rows, err := h.dbOf(ctx).Query(ctx, `
		SELECT id::text, slug, name FROM core.games WHERE active = true ORDER BY name
	`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	games := []map[string]any{}
	slugByID := map[string]string{}
	for rows.Next() {
		var id, slug, name string
		if rows.Scan(&id, &slug, &name) == nil {
			slugByID[id] = slug
			games = append(games, map[string]any{"slug": slug, "name": name})
		}
	}
	rows.Close()

	tariffs := []map[string]any{}
	for _, t := range h.loadRentTariffs(ctx, "", "") {
		gameName, locationName := "", ""
		if g, ok := t["game"].(map[string]string); ok {
			gameName = g["name"]
		}
		if l, ok := t["location"].(map[string]string); ok {
			locationName = l["name"]
		}
		gameID, _ := t["game_id"].(string)
		tariffs = append(tariffs, map[string]any{
			"id":            t["id"],
			"name":          t["name"],
			"game":          slugByID[gameID],
			"game_name":     gameName,
			"location_id":   t["location_id"],
			"location_name": locationName,
			"billing_type":  t["billing_type"],
			"price_monthly": t["price_monthly"],
			"currency":      t["currency"],
			"cpu_cores":     t["cpu_cores"],
			"ram_gb":        t["ram_gb"],
			"disk_gb":       t["disk_gb"],
			"min_slots":     t["min_slots"],
			"max_slots":     t["max_slots"],
		})
	}

	locations := []map[string]any{}
	for _, n := range h.queryRentNodes(ctx) {
		locations = append(locations, map[string]any{
			"id":          n["id"],
			"name":        n["name"],
			"country":     n["country"],
			"code":        n["code"],
			"online":      n["is_online"],
			"maintenance": n["maintenance_mode"],
		})
	}

	versions := []map[string]any{}
	for _, v := range h.queryGameVersions(ctx) {
		versions = append(versions, map[string]any{"id": v["id"], "game": v["game_slug"], "name": v["name"]})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"games":     games,
		"tariffs":   tariffs,
		"locations": locations,
		"versions":  versions,
	})
}

func (h *Handler) WHMCSServices(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	rows, err := h.readerOf(ctx).Query(ctx, `
		SELECT w.service_id, w.client_id, w.status, w.product, w.next_due_date, w.updated_at,
		       COALESCE(u.email, ''), COALESCE(w.server_id::text, ''), COALESCE(s.name, ''), COALESCE(s.status, '')
		FROM core.whmcs_services w
		LEFT JOIN core.users u ON u.id = w.user_id
		LEFT JOIN core.servers s ON s.id = w.server_id
		ORDER BY w.updated_at DESC
		LIMIT 500
	`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()

	list := []map[string]any{}
	for rows.Next() {
		var serviceID, clientID int64
		var status, product, email, serverID, serverName, serverStatus string
		var due *time.Time
		var updatedAt time.Time
		if rows.Scan(&serviceID, &clientID, &status, &product, &due, &updatedAt,
			&email, &serverID, &serverName, &serverStatus) != nil {
			continue
		}
		list = append(list, map[string]any{
			"service_id":    serviceID,
			"client_id":     clientID,
			"status":        status,
			"product":       product,
			"next_due_date": dateOrNil(due),
			"updated_at":    updatedAt.Format(time.RFC3339),
			"email":         email,
			"server_id":     nilIfEmpty(serverID),
			"server_name":   serverName,
			"server_status": serverStatus,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"services": list})
}

func (h *Handler) WHMCSUsage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	rows, err := h.readerOf(ctx).Query(ctx, `
		SELECT w.service_id, s.id::text, COALESCE(s.limits, '{}'::jsonb)
		FROM core.whmcs_services w
		JOIN core.servers s ON s.id = w.server_id
		WHERE w.status <> 'terminated'
	`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	type usageRow struct {
		serviceID int64
		serverID  string
		limits    []byte
	}
	items := []usageRow{}
	for rows.Next() {
		var it usageRow
		if rows.Scan(&it.serviceID, &it.serverID, &it.limits) == nil {
			items = append(items, it)
		}
	}
	rows.Close()

	list := make([]map[string]any, 0, len(items))
	for _, it := range items {
		limits := map[string]any{}
		_ = json.Unmarshal(it.limits, &limits)
		used, total, _ := h.latestServerMetricExtras(ctx, it.serverID)
		limit := intFromAny(limits["disk_mb"])
		if limit <= 0 {
			limit = total
		}
		list = append(list, map[string]any{
			"service_id":         it.serviceID,
			"server_id":          it.serverID,
			"disk_used_mb":       used,
			"disk_limit_mb":      limit,
			"bandwidth_used_mb":  0,
			"bandwidth_limit_mb": 0,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"services": list})
}

func (h *Handler) WHMCSServiceShow(w http.ResponseWriter, r *http.Request) {
	serviceID, ok := whmcsIDParam(r, "serviceId")
	if !ok {
		writeError(w, http.StatusBadRequest, "некорректный номер услуги WHMCS")
		return
	}
	h.writeWHMCSService(w, r, serviceID, http.StatusOK, nil)
}

func (h *Handler) WHMCSServiceCreate(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	serviceID, ok := whmcsIDParam(r, "serviceId")
	if !ok {
		writeError(w, http.StatusBadRequest, "некорректный номер услуги WHMCS")
		return
	}
	var body whmcsServiceRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	body.Client.Email = strings.ToLower(strings.TrimSpace(body.Client.Email))
	if body.Client.ID <= 0 || !strings.Contains(body.Client.Email, "@") {
		writeError(w, http.StatusBadRequest, "укажите номер и email клиента WHMCS")
		return
	}

	ctx := r.Context()
	if s, err := h.loadWHMCSService(ctx, serviceID); err == nil && s.ServerID != "" {
		h.writeWHMCSService(w, r, serviceID, http.StatusOK, nil)
		return
	}

	plan, err := h.planWHMCSServer(ctx, serviceID, body)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	tx, err := h.dbOf(ctx).Begin(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer tx.Rollback(ctx)

	var existingServer string
	if err := tx.QueryRow(ctx, `
		INSERT INTO core.whmcs_services (service_id, client_id)
		VALUES ($1, $2)
		ON CONFLICT (service_id) DO UPDATE SET updated_at = now()
		RETURNING COALESCE(server_id::text, '')
	`, serviceID, body.Client.ID).Scan(&existingServer); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	if existingServer != "" {
		_ = tx.Rollback(ctx)
		h.writeWHMCSService(w, r, serviceID, http.StatusOK, nil)
		return
	}

	account, err := h.resolveWHMCSAccount(ctx, tx, body.Client, body.Password)
	if err != nil {
		var refusal whmcsRefusal
		if errors.As(err, &refusal) {
			writeError(w, http.StatusConflict, refusal.Error())
			return
		}
		log.Printf("whmcs: учётная запись клиента %d не подготовлена: %v", body.Client.ID, err)
		writeError(w, http.StatusInternalServerError, "не удалось подготовить учётную запись клиента")
		return
	}

	limitsJSON, _ := json.Marshal(plan.limits)
	var serverID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO core.servers (node_id, game_id, game_version_id, name, limits, user_id, tariff_id,
		                          provisioning_status, billing_source, config)
		VALUES ($1::uuid, $2, NULLIF($3, '')::uuid, $4, $5::jsonb, $6::uuid, $7::uuid,
		        'provisioning', 'whmcs', jsonb_build_object('startup_params', $8::text))
		RETURNING id::text
	`, plan.nodeID, plan.gameID, plan.versionID, plan.name, limitsJSON, account.userID, plan.tariffID,
		h.defaultStartupParams(ctx, plan.gameID)).Scan(&serverID); err != nil {
		log.Printf("whmcs: сервер услуги %d не создан: %v", serviceID, err)
		writeError(w, http.StatusInternalServerError, "не удалось создать сервер")
		return
	}
	if _, err := tx.Exec(ctx, `
		UPDATE core.whmcs_services
		SET client_id = $2, user_id = $3::uuid, server_id = $4::uuid, status = 'active',
		    blocked_by_whmcs = false, suspend_reason = NULL,
		    product = $5, billing_cycle = $6, next_due_date = $7, updated_at = now()
		WHERE service_id = $1
	`, serviceID, body.Client.ID, account.userID, serverID, strings.TrimSpace(body.Product),
		strings.TrimSpace(body.BillingCycle), whmcsDueDate(body.NextDueDate)); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	if err := tx.Commit(ctx); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}

	h.ensureDefaultWallet(ctx, account.userID)
	if account.created {
		h.saveWHMCSProfile(ctx, account.userID, body.Client)
	}
	h.emitWebhook(ctx, "server.created", map[string]any{
		"server_id": serverID, "server_name": plan.name, "game_id": plan.gameID,
		"node_id": plan.nodeID, "user_id": account.userID, "whmcs_service_id": serviceID,
	})
	h.launchNewServer(ctx, serverID, plan.nodeID, plan.gameID, plan.name, limitsJSON)
	audit(ctx, h.dbOf(ctx), claims.UserID, "whmcs.service.create", "server:"+serverID,
		map[string]any{"service_id": serviceID, "client_id": body.Client.ID})

	extra := map[string]any{"username": account.email}
	if account.password != "" {
		extra["password"] = account.password
	}
	h.writeWHMCSService(w, r, serviceID, http.StatusCreated, extra)
}

func (h *Handler) planWHMCSServer(ctx context.Context, serviceID int64, body whmcsServiceRequest) (*whmcsServerPlan, error) {
	tariffID := strings.TrimSpace(body.TariffID)
	if tariffID == "" {
		return nil, errors.New("в настройках продукта WHMCS не выбран тариф панели")
	}
	tariff, err := h.loadTariffLegacyJSON(ctx, tariffID)
	if err != nil {
		return nil, errors.New("тариф панели не найден")
	}

	tariffGame := h.tariffGameSlug(ctx, tariffID)
	gameRef := strings.TrimSpace(body.Game)
	if gameRef == "" {
		gameRef = tariffGame
	}
	if gameRef == "" {
		return nil, errors.New("не указана игра: выберите её в продукте WHMCS или передайте настраиваемой опцией game")
	}
	gameID, ok := resolveGameSlug(ctx, h, gameRef)
	if !ok {
		return nil, fmt.Errorf("игра %q не найдена в каталоге панели", gameRef)
	}
	if tariffGame != "" && tariffGame != gameID {
		return nil, fmt.Errorf("тариф «%v» предназначен для другой игры", tariff["name"])
	}

	tariffNode, _ := tariff["location_id"].(string)
	nodeID, err := h.whmcsNode(ctx, body.Location, tariffNode, gameID)
	if err != nil {
		return nil, err
	}

	order := whmcsOrder(body.Slots, body.CPUCores, body.RAMGb, body.DiskGb, tariff)
	if msg := tariffMeetsGame(gameID, order); msg != "" {
		return nil, errors.New(msg)
	}
	limits := gamecatalog.DefaultLimits(gameID)
	applyOrderLimits(limits, order)

	versionID := ""
	if ref := strings.TrimSpace(body.Version); ref != "" {
		if versionID = h.resolveGameVersionRef(ctx, gameID, ref); versionID == "" {
			return nil, fmt.Errorf("версия %q не найдена у игры %s", ref, gameID)
		}
	}

	name := strings.TrimSpace(body.Name)
	if name == "" {
		name = fmt.Sprintf("%s #%d", h.gameDisplayName(ctx, gameID), serviceID)
	}
	if runes := []rune(name); len(runes) > 64 {
		name = string(runes[:64])
	}

	return &whmcsServerPlan{
		tariffID:  tariffID,
		gameID:    gameID,
		nodeID:    nodeID,
		versionID: versionID,
		name:      name,
		limits:    limits,
	}, nil
}

func whmcsOrder(slots, cpu, ram, disk int, tariff map[string]any) pricing.RentOrder {
	order := pricing.RentOrder{Slots: slots, CPUCores: cpu, RAMGb: ram, DiskGb: disk}
	if order.CPUCores <= 0 {
		order.CPUCores = intFromAny(tariff["cpu_cores"])
	}
	if order.RAMGb <= 0 {
		order.RAMGb = intFromAny(tariff["ram_gb"])
	}
	if order.DiskGb <= 0 {
		order.DiskGb = intFromAny(tariff["disk_gb"])
	}
	return order
}

func applyOrderLimits(limits map[string]any, order pricing.RentOrder) {
	if order.RAMGb > 0 {
		limits["memory_mb"] = order.RAMGb * 1024
		limits["ram_gb"] = order.RAMGb
	}
	if order.DiskGb > 0 {
		limits["disk_mb"] = order.DiskGb * 1024
		limits["disk_gb"] = order.DiskGb
	}
	if order.CPUCores > 0 {
		limits["cpu"] = order.CPUCores
		limits["cpu_cores"] = order.CPUCores
	}
	if order.Slots > 0 {
		limits["slots"] = order.Slots
	}
}

func (h *Handler) whmcsNode(ctx context.Context, ref, tariffNode, gameID string) (string, error) {
	nodeID := ""
	if ref = strings.TrimSpace(ref); ref != "" {
		if nodeID = h.resolveNodeRef(ctx, ref); nodeID == "" {
			return "", fmt.Errorf("локация %q не найдена", ref)
		}
	}
	if tariffNode = strings.TrimSpace(tariffNode); tariffNode != "" {
		if nodeID != "" && nodeID != tariffNode {
			return "", errors.New("тариф привязан к другой локации")
		}
		nodeID = tariffNode
	}
	if nodeID == "" {
		if nodeID = h.pickWHMCSNode(ctx, gameID); nodeID == "" {
			return "", errors.New("нет локации, на которой можно разместить сервер этой игры")
		}
		return nodeID, nil
	}
	if h.nodeInMaintenance(ctx, nodeID) {
		return "", errors.New("на выбранной локации идут технические работы")
	}
	if reason := h.nodeCapacityReason(ctx, nodeID, gameID); reason != "" {
		return "", errors.New(reason)
	}
	return nodeID, nil
}

func (h *Handler) resolveNodeRef(ctx context.Context, ref string) string {
	var id string
	_ = h.dbOf(ctx).QueryRow(ctx, `
		SELECT id::text FROM core.nodes
		WHERE id::text = $1 OR lower(name) = lower($1) OR lower(COALESCE(meta->>'code', '')) = lower($1)
		ORDER BY (id::text = $1) DESC
		LIMIT 1
	`, ref).Scan(&id)
	return id
}

func (h *Handler) pickWHMCSNode(ctx context.Context, gameID string) string {
	rows, err := h.dbOf(ctx).Query(ctx, `
		SELECT n.id::text
		FROM core.nodes n
		WHERE COALESCE(n.is_active, n.active, true) = true
		  AND COALESCE(n.maintenance_mode, false) = false
		ORDER BY (n.status = 'online') DESC,
		         (SELECT COUNT(*) FROM core.servers s WHERE s.node_id = n.id) ASC
		LIMIT 20
	`)
	if err != nil {
		return ""
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}
	rows.Close()
	for _, id := range ids {
		if h.nodeCapacityReason(ctx, id, gameID) == "" {
			return id
		}
	}
	return ""
}

func (h *Handler) tariffGameSlug(ctx context.Context, tariffID string) string {
	var slug string
	_ = h.dbOf(ctx).QueryRow(ctx, `
		SELECT COALESCE(g.slug, '')
		FROM core.tariffs t
		LEFT JOIN core.games g ON g.id = t.game_id
		WHERE t.id::text = $1
	`, tariffID).Scan(&slug)
	return slug
}

func (h *Handler) resolveGameVersionRef(ctx context.Context, gameSlug, ref string) string {
	var id string
	_ = h.dbOf(ctx).QueryRow(ctx, `
		SELECT gv.id::text
		FROM core.game_versions gv
		JOIN core.games g ON g.id = gv.game_id
		WHERE g.slug = $1 AND gv.active = true
		  AND (gv.id::text = $2 OR lower(COALESCE(gv.version, '')) = lower($2))
		ORDER BY (gv.id::text = $2) DESC
		LIMIT 1
	`, gameSlug, ref).Scan(&id)
	return id
}

func (h *Handler) gameDisplayName(ctx context.Context, slug string) string {
	var name string
	if h.dbOf(ctx).QueryRow(ctx, `SELECT name FROM core.games WHERE slug = $1`, slug).Scan(&name) == nil && name != "" {
		return name
	}
	if g, ok := gamecatalog.Resolve(slug); ok && g.Name != "" {
		return g.Name
	}
	return slug
}

func (h *Handler) resolveWHMCSAccount(ctx context.Context, tx pgx.Tx, c whmcsClient, password string) (*whmcsAccount, error) {
	acc := &whmcsAccount{}
	var role string
	err := tx.QueryRow(ctx, `
		SELECT u.id::text, u.email, u.role
		FROM core.whmcs_clients c
		JOIN core.users u ON u.id = c.user_id
		WHERE c.client_id = $1
	`, c.ID).Scan(&acc.userID, &acc.email, &role)
	if err == nil {
		if isStaffRole(role) {
			return nil, whmcsRefusal("клиент WHMCS связан с сотрудником панели")
		}
		return acc, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	err = tx.QueryRow(ctx, `
		SELECT id::text, email, role FROM core.users WHERE email = $1
	`, c.Email).Scan(&acc.userID, &acc.email, &role)
	switch {
	case err == nil:
		if isStaffRole(role) {
			return nil, whmcsRefusal("email " + c.Email + " принадлежит сотруднику панели — у клиента WHMCS должен быть другой адрес")
		}
	case errors.Is(err, pgx.ErrNoRows):
		acc.password = strings.TrimSpace(password)
		if len(acc.password) < 8 {
			acc.password = randomToken(9)
		}
		hash, hashErr := bcrypt.GenerateFromPassword([]byte(acc.password), bcrypt.DefaultCost)
		if hashErr != nil {
			return nil, hashErr
		}
		if err := tx.QueryRow(ctx, `
			INSERT INTO core.users (email, password_hash, role, email_verified_at)
			VALUES ($1, $2, 'user', now())
			RETURNING id::text
		`, c.Email, string(hash)).Scan(&acc.userID); err != nil {
			return nil, err
		}
		acc.email = c.Email
		acc.created = true
	default:
		return nil, err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO core.whmcs_clients (client_id, user_id) VALUES ($1, $2::uuid)
		ON CONFLICT DO NOTHING
	`, c.ID, acc.userID); err != nil {
		return nil, err
	}
	return acc, nil
}

func (h *Handler) saveWHMCSProfile(ctx context.Context, userID string, c whmcsClient) {
	first := strings.TrimSpace(c.FirstName)
	last := strings.TrimSpace(c.LastName)
	display := strings.TrimSpace(first + " " + last)
	if _, err := h.dbOf(ctx).Exec(ctx, `
		INSERT INTO core.user_profiles (user_id, display_name, first_name, last_name, locale, updated_at)
		VALUES ($1::uuid, NULLIF($2, ''), NULLIF($3, ''), NULLIF($4, ''),
		        COALESCE(
		            (SELECT code FROM core.languages WHERE code = $5 AND enabled),
		            NULLIF((SELECT value #>> '{}' FROM core.tenant_settings WHERE key = 'i18n.default_locale'), ''),
		            'ru'),
		        now())
		ON CONFLICT (user_id) DO NOTHING
	`, userID, display, first, last, strings.ToLower(strings.TrimSpace(c.Locale))); err != nil {
		log.Printf("whmcs: профиль пользователя %s не сохранён: %v", userID, err)
	}
}

func (h *Handler) WHMCSServiceTerminate(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	serviceID, ok := whmcsIDParam(r, "serviceId")
	if !ok {
		writeError(w, http.StatusBadRequest, "некорректный номер услуги WHMCS")
		return
	}
	ctx := r.Context()
	s, err := h.loadWHMCSService(ctx, serviceID)
	if errors.Is(err, errWHMCSServiceNotFound) {
		writeJSON(w, http.StatusOK, map[string]any{"service_id": serviceID, "status": "terminated"})
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}

	deferred := false
	if s.ServerID != "" {
		var nodeID string
		if scanErr := h.dbOf(ctx).QueryRow(ctx, `
			SELECT node_id::text FROM core.servers WHERE id = $1
		`, s.ServerID).Scan(&nodeID); scanErr == nil {
			var destroyErr error
			deferred, destroyErr = h.destroyServer(r, s.ServerID, nodeID)
			if destroyErr != nil && !errors.Is(destroyErr, pgx.ErrNoRows) {
				writeError(w, http.StatusInternalServerError, destroyErr.Error())
				return
			}
		}
	}

	if _, err := h.dbOf(ctx).Exec(ctx, `
		UPDATE core.whmcs_services
		SET status = 'terminated', server_id = NULL, blocked_by_whmcs = false, suspend_reason = NULL, updated_at = now()
		WHERE service_id = $1
	`, serviceID); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	audit(ctx, h.dbOf(ctx), claims.UserID, "whmcs.service.terminate", fmt.Sprintf("whmcs:%d", serviceID),
		map[string]any{"server_id": s.ServerID, "cleanup_deferred": deferred})
	writeJSON(w, http.StatusOK, map[string]any{
		"service_id":       serviceID,
		"status":           "terminated",
		"cleanup_deferred": deferred,
	})
}

func (h *Handler) WHMCSServiceSuspend(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	s, ok := h.whmcsServiceFromRequest(w, r)
	if !ok {
		return
	}
	var body struct {
		Reason string `json:"reason"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	reason := strings.TrimSpace(body.Reason)
	ctx := r.Context()

	blockedNow := false
	if s.ServerID != "" {
		var blocked bool
		_ = h.dbOf(ctx).QueryRow(ctx, `
			SELECT COALESCE(is_blocked, false) FROM core.servers WHERE id = $1
		`, s.ServerID).Scan(&blocked)
		switch {
		case !blocked:
			ownerID, name, err := h.applyServerBlock(ctx, s.ServerID, true, reason)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "не удалось приостановить сервер")
				return
			}
			blockedNow = true
			h.notifyWHMCSSuspension(ctx, ownerID, s, name, true)
		case s.BlockedByWHMCS:
			_, _ = h.dbOf(ctx).Exec(ctx, `
				UPDATE core.servers SET blocked_reason = NULLIF($2, '') WHERE id = $1
			`, s.ServerID, reason)
		}
	}

	if _, err := h.dbOf(ctx).Exec(ctx, `
		UPDATE core.whmcs_services
		SET status = 'suspended', suspend_reason = NULLIF($2, ''),
		    blocked_by_whmcs = blocked_by_whmcs OR $3, updated_at = now()
		WHERE service_id = $1
	`, s.ID, reason, blockedNow); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	audit(ctx, h.dbOf(ctx), claims.UserID, "whmcs.service.suspend", fmt.Sprintf("whmcs:%d", s.ID),
		map[string]any{"server_id": s.ServerID, "reason": reason})
	h.writeWHMCSService(w, r, s.ID, http.StatusOK, nil)
}

func (h *Handler) WHMCSServiceUnsuspend(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	s, ok := h.whmcsServiceFromRequest(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	if s.ServerID != "" && s.BlockedByWHMCS {
		ownerID, name, err := h.applyServerBlock(ctx, s.ServerID, false, "")
		if err != nil {
			writeError(w, http.StatusInternalServerError, "не удалось возобновить сервер")
			return
		}
		h.notifyWHMCSSuspension(ctx, ownerID, s, name, false)
	}
	if _, err := h.dbOf(ctx).Exec(ctx, `
		UPDATE core.whmcs_services
		SET status = CASE WHEN status = 'terminated' THEN status ELSE 'active' END,
		    suspend_reason = NULL, blocked_by_whmcs = false, updated_at = now()
		WHERE service_id = $1
	`, s.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	audit(ctx, h.dbOf(ctx), claims.UserID, "whmcs.service.unsuspend", fmt.Sprintf("whmcs:%d", s.ID),
		map[string]any{"server_id": s.ServerID})
	h.writeWHMCSService(w, r, s.ID, http.StatusOK, nil)
}

func (h *Handler) notifyWHMCSSuspension(ctx context.Context, ownerID string, s *whmcsService, name string, suspended bool) {
	if ownerID == "" {
		return
	}
	event := notify.Event{
		Kind:   notify.KindServerUnblocked,
		Title:  i18n.Key("notify.service_unsuspended.title"),
		Body:   i18n.Key("notify.service_unsuspended.body", i18n.Params{"name": name}),
		Action: h.serverAction("notify.action.open_server", s.ServerID, ""),
		Meta:   map[string]any{"server_id": s.ServerID, "whmcs_service_id": s.ID},
	}
	if suspended {
		event.Kind = notify.KindServerBlocked
		event.Title = i18n.Key("notify.service_suspended.title")
		event.Body = i18n.Key("notify.service_suspended.body", i18n.Params{"name": name})
		event.Action = nil
		if link := h.whmcsServiceURL(ctx, s.ID); link != "" {
			event.Action = &notify.Action{Label: i18n.Key("notify.action.billing"), Href: link}
		}
	}
	h.notifyUser(ctx, ownerID, event)
}

func (h *Handler) WHMCSServicePackage(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	s, ok := h.whmcsServiceFromRequest(w, r)
	if !ok || !requireWHMCSServer(w, s) {
		return
	}
	var body whmcsServiceRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	ctx := r.Context()
	row, err := h.loadServerTariffRow(ctx, s.ServerID)
	if err != nil {
		writeError(w, http.StatusNotFound, "server not found")
		return
	}

	tariffID := strings.TrimSpace(body.TariffID)
	if tariffID == "" {
		tariffID = row.tariffID
	}
	if tariffID == "" {
		writeError(w, http.StatusUnprocessableEntity, "в настройках продукта WHMCS не выбран тариф панели")
		return
	}
	tariff, err := h.loadTariffLegacyJSON(ctx, tariffID)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "тариф панели не найден")
		return
	}
	if game := h.tariffGameSlug(ctx, tariffID); game != "" && game != row.gameID {
		writeError(w, http.StatusUnprocessableEntity,
			"новый тариф предназначен для другой игры — смена игры выполняется новой услугой")
		return
	}
	if node, _ := tariff["location_id"].(string); node != "" && node != row.nodeID {
		writeError(w, http.StatusUnprocessableEntity, "новый тариф привязан к другой локации")
		return
	}

	order := whmcsOrder(body.Slots, body.CPUCores, body.RAMGb, body.DiskGb, tariff)
	if msg := tariffMeetsGame(row.gameID, order); msg != "" {
		writeError(w, http.StatusUnprocessableEntity, msg)
		return
	}
	applyOrderLimits(row.limits, order)
	limitsJSON, _ := json.Marshal(row.limits)

	if _, err := h.dbOf(ctx).Exec(ctx, `
		UPDATE core.servers SET tariff_id = $2::uuid, limits = $3::jsonb WHERE id = $1
	`, s.ServerID, tariffID, limitsJSON); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	_, _ = h.dbOf(ctx).Exec(ctx, `
		UPDATE core.whmcs_services
		SET product = COALESCE(NULLIF($2, ''), product),
		    billing_cycle = COALESCE(NULLIF($3, ''), billing_cycle),
		    next_due_date = COALESCE($4::date, next_due_date),
		    updated_at = now()
		WHERE service_id = $1
	`, s.ID, strings.TrimSpace(body.Product), strings.TrimSpace(body.BillingCycle), whmcsDueDate(body.NextDueDate))
	_ = h.cache.InvalidateTenantServers(ctx)
	audit(ctx, h.dbOf(ctx), claims.UserID, "whmcs.service.package", "server:"+s.ServerID,
		map[string]any{"service_id": s.ID, "tariff_id": tariffID, "limits": row.limits})
	h.writeWHMCSService(w, r, s.ID, http.StatusOK, map[string]any{"applies_on_restart": true})
}

func (h *Handler) WHMCSServiceRenew(w http.ResponseWriter, r *http.Request) {
	s, ok := h.whmcsServiceFromRequest(w, r)
	if !ok {
		return
	}
	var body struct {
		NextDueDate string `json:"next_due_date"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	ctx := r.Context()
	if _, err := h.dbOf(ctx).Exec(ctx, `
		UPDATE core.whmcs_services
		SET next_due_date = COALESCE($2::date, next_due_date), updated_at = now()
		WHERE service_id = $1
	`, s.ID, whmcsDueDate(body.NextDueDate)); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	h.writeWHMCSService(w, r, s.ID, http.StatusOK, nil)
}

func (h *Handler) WHMCSServicePassword(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	s, ok := h.whmcsServiceFromRequest(w, r)
	if !ok {
		return
	}
	var body struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	password := strings.TrimSpace(body.Password)
	if len(password) < 8 {
		writeError(w, http.StatusUnprocessableEntity, "пароль должен быть не короче 8 символов")
		return
	}
	if s.UserID == "" {
		writeError(w, http.StatusConflict, "у услуги нет учётной записи в панели")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}
	ctx := r.Context()
	tag, err := h.dbOf(ctx).Exec(ctx, `
		UPDATE core.users SET password_hash = $2 WHERE id = $1 AND role = 'user'
	`, s.UserID, string(hash))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusConflict, "пароль этой учётной записи из WHMCS не меняется")
		return
	}
	audit(ctx, h.dbOf(ctx), claims.UserID, "whmcs.service.password", "user:"+s.UserID,
		map[string]any{"service_id": s.ID})
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (h *Handler) whmcsServerUsable(w http.ResponseWriter, r *http.Request) (*whmcsService, bool) {
	s, ok := h.whmcsServiceFromRequest(w, r)
	if !ok || !requireWHMCSServer(w, s) {
		return nil, false
	}
	if s.Status == "suspended" {
		writeError(w, http.StatusForbidden, "услуга приостановлена в WHMCS")
		return nil, false
	}
	var blocked bool
	var reason string
	_ = h.dbOf(r.Context()).QueryRow(r.Context(), `
		SELECT COALESCE(is_blocked, false), COALESCE(blocked_reason, '') FROM core.servers WHERE id = $1
	`, s.ServerID).Scan(&blocked, &reason)
	if blocked {
		if reason == "" {
			reason = "обратитесь в поддержку"
		}
		writeError(w, http.StatusForbidden, "сервер заблокирован: "+reason)
		return nil, false
	}
	return s, true
}

func (h *Handler) WHMCSServicePower(w http.ResponseWriter, r *http.Request) {
	s, ok := h.whmcsServerUsable(w, r)
	if !ok {
		return
	}
	var body struct {
		Action string `json:"action"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	h.PowerServer(w, requestForServer(r, s.ServerID, map[string]string{"action": body.Action}))
}

func (h *Handler) WHMCSServiceReinstall(w http.ResponseWriter, r *http.Request) {
	s, ok := h.whmcsServerUsable(w, r)
	if !ok {
		return
	}
	h.ReinstallServer(w, requestForServer(r, s.ServerID, map[string]string{}))
}

func (h *Handler) WHMCSServiceSSO(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	s, ok := h.whmcsServiceFromRequest(w, r)
	if !ok {
		return
	}
	if s.UserID == "" {
		writeError(w, http.StatusConflict, "у услуги нет учётной записи в панели")
		return
	}
	if s.Status == "terminated" {
		writeError(w, http.StatusConflict, "услуга удалена")
		return
	}
	var body struct {
		Redirect string `json:"redirect"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	ctx := r.Context()
	var role, status string
	if err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT role, status FROM core.users WHERE id = $1
	`, s.UserID).Scan(&role, &status); err != nil {
		writeError(w, http.StatusNotFound, "учётная запись не найдена")
		return
	}
	if isStaffRole(role) {
		writeError(w, http.StatusForbidden, "вход под сотрудником панели из WHMCS запрещён")
		return
	}
	if status != "active" {
		writeError(w, http.StatusForbidden, "учётная запись в панели отключена")
		return
	}
	if h.frontendURL == "" {
		writeError(w, http.StatusInternalServerError, "не задан публичный адрес панели (FRONTEND_URL)")
		return
	}

	token := randomToken(32)
	_, _ = h.dbOf(ctx).Exec(ctx, `
		DELETE FROM core.whmcs_sso_tokens WHERE expires_at < now() - interval '1 day'
	`)
	if _, err := h.dbOf(ctx).Exec(ctx, `
		INSERT INTO core.whmcs_sso_tokens (token_hash, user_id, redirect, expires_at)
		VALUES ($1, $2::uuid, $3, $4)
	`, sha256Hex(token), s.UserID, whmcsRedirect(body.Redirect, s.ServerID), time.Now().Add(whmcsSSOTTL)); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	audit(ctx, h.dbOf(ctx), claims.UserID, "whmcs.sso", "user:"+s.UserID, map[string]any{"service_id": s.ID})
	writeJSON(w, http.StatusOK, map[string]any{
		"url":        h.frontendURL + "/auth/whmcs?token=" + token,
		"expires_in": int(whmcsSSOTTL.Seconds()),
	})
}

func (h *Handler) WHMCSClientUpdate(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	clientID, ok := whmcsIDParam(r, "clientId")
	if !ok {
		writeError(w, http.StatusBadRequest, "некорректный номер клиента WHMCS")
		return
	}
	var body whmcsClient
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	ctx := r.Context()
	var userID, role string
	err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT u.id::text, u.role
		FROM core.whmcs_clients c
		JOIN core.users u ON u.id = c.user_id
		WHERE c.client_id = $1
	`, clientID).Scan(&userID, &role)
	if errors.Is(err, pgx.ErrNoRows) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "unlinked"})
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	if isStaffRole(role) {
		writeError(w, http.StatusConflict, "клиент WHMCS связан с сотрудником панели")
		return
	}

	if email := strings.ToLower(strings.TrimSpace(body.Email)); strings.Contains(email, "@") {
		var taken bool
		_ = h.dbOf(ctx).QueryRow(ctx, `
			SELECT EXISTS(SELECT 1 FROM core.users WHERE email = $1 AND id <> $2::uuid)
		`, email, userID).Scan(&taken)
		if taken {
			writeError(w, http.StatusConflict, "email "+email+" уже занят другой учётной записью панели")
			return
		}
		if _, err := h.dbOf(ctx).Exec(ctx, `
			UPDATE core.users SET email = $1 WHERE id = $2::uuid
		`, email, userID); err != nil {
			writeError(w, http.StatusInternalServerError, "database error")
			return
		}
	}

	first := strings.TrimSpace(body.FirstName)
	last := strings.TrimSpace(body.LastName)
	if first != "" || last != "" {
		display := strings.TrimSpace(first + " " + last)
		_, _ = h.dbOf(ctx).Exec(ctx, `
			INSERT INTO core.user_profiles (user_id, display_name, first_name, last_name, updated_at)
			VALUES ($1::uuid, $2, NULLIF($3, ''), NULLIF($4, ''), now())
			ON CONFLICT (user_id) DO UPDATE SET display_name = EXCLUDED.display_name,
			    first_name = EXCLUDED.first_name, last_name = EXCLUDED.last_name, updated_at = now()
		`, userID, display, first, last)
	}
	_, _ = h.dbOf(ctx).Exec(ctx, `
		UPDATE core.whmcs_clients SET updated_at = now() WHERE client_id = $1
	`, clientID)
	audit(ctx, h.dbOf(ctx), claims.UserID, "whmcs.client.update", "user:"+userID,
		map[string]any{"client_id": clientID})
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated", "user_id": userID})
}
