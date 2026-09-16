package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/vortanixapp/panel/internal/api/pricing"
)

func (h *Handler) ServerTariffList(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	serverID := chi.URLParam(r, "id")
	access, err := h.resolveServerAccess(r.Context(), claims.UserID, claims.Role, serverID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "server not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	if !access.IsOwner && !access.IsStaff {
		writeError(w, http.StatusForbidden, "only server owner can view tariffs")
		return
	}

	var nodeID, gameID, tariffID string
	err = h.dbOf(r.Context()).QueryRow(r.Context(), `
		SELECT COALESCE(node_id::text, ''), COALESCE(game_id, ''), COALESCE(tariff_id::text, '')
		FROM core.servers WHERE id = $1
	`, serverID).Scan(&nodeID, &gameID, &tariffID)
	if err != nil {
		writeError(w, http.StatusNotFound, "server not found")
		return
	}

	q := tariffSelectSQL + `
		WHERE t.active = true
		  AND (t.node_id IS NULL OR t.node_id::text = $1)
		  AND (t.game_id IS NULL OR t.game_id::text IN (
		    SELECT g.id::text FROM core.games g WHERE g.slug = $2
		  ))
		ORDER BY t.position ASC, t.name ASC`
	rows, err := h.dbOf(r.Context()).Query(r.Context(), q, nodeID, gameID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load tariffs")
		return
	}
	defer rows.Close()

	list := []map[string]any{}
	for rows.Next() {
		row, scanErr := scanTariffRow(rows)
		if scanErr != nil {
			continue
		}
		list = append(list, tariffToLegacyJSON(row))
	}
	writeJSON(w, http.StatusOK, map[string]any{"tariffs": list})
}

func (h *Handler) ServerTariffChange(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	serverID := chi.URLParam(r, "id")
	access, err := h.resolveServerAccess(r.Context(), claims.UserID, claims.Role, serverID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "server not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	if !access.IsOwner {
		writeError(w, http.StatusForbidden, "only server owner can change tariff")
		return
	}
	if h.refuseWHMCSBilled(r.Context(), w, serverID) {
		return
	}

	var body struct {
		TariffID any    `json:"tariff_id"`
		WalletID string `json:"wallet_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	newTariffID := strings.TrimSpace(fmt.Sprint(body.TariffID))
	if newTariffID == "" || newTariffID == "<nil>" {
		writeError(w, http.StatusBadRequest, "tariff_id required")
		return
	}

	ctx := r.Context()
	row, err := h.loadServerTariffRow(ctx, serverID)
	if err != nil {
		writeError(w, http.StatusNotFound, "server not found")
		return
	}
	if row.expiresAt != nil && !row.expiresAt.After(time.Now()) {
		writeError(w, http.StatusBadRequest, "Срок аренды истёк")
		return
	}
	if row.tariffID == newTariffID {
		writeJSON(w, http.StatusOK, map[string]string{"status": "unchanged"})
		return
	}

	currentTariff, err := h.loadTariffLegacyJSON(ctx, row.tariffID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Тариф не найден")
		return
	}
	newTariff, err := h.loadTariffLegacyJSON(ctx, newTariffID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Тариф не найден")
		return
	}
	if !h.tariffAvailableForServer(ctx, newTariff, newTariffID, row.nodeID, row.gameID) {
		writeError(w, http.StatusBadRequest, "Этот тариф недоступен для сервера")
		return
	}
	curCurrency := tariffCurrency(currentTariff)
	if newCurrency := tariffCurrency(newTariff); curCurrency != newCurrency {
		writeError(w, http.StatusBadRequest, "Нельзя сменить тариф: разные валюты тарифов")
		return
	}

	order := limitsToRentOrder(row.limits)
	target := pricing.Resolve(newTariff, order)
	oldMonthly := pricing.MonthlyCost(currentTariff, order)
	newMonthly := pricing.MonthlyCost(newTariff, target)
	if oldMonthly <= 0 || newMonthly <= 0 {
		writeError(w, http.StatusBadRequest, "Не удалось рассчитать стоимость")
		return
	}

	daysLeft := serverRemainingDays(row.expiresAt)
	delta, newExpires := tariffChangeBilling(oldMonthly, newMonthly, daysLeft, row.expiresAt)
	if delta > 0 {
		if _, err := h.debitWalletForRentSource(r, claims, body.WalletID, curCurrency, delta,
			"Смена тарифа сервера", "server_tariff", serverID); err != nil {
			writeError(w, http.StatusPaymentRequired, err.Error())
			return
		}
	}

	limits := tariffLimits(newTariff, target, row.limits)
	limitsJSON, _ := json.Marshal(limits)
	_, err = h.dbOf(ctx).Exec(ctx, `
		UPDATE core.servers
		SET tariff_id = $2::uuid, limits = $3::jsonb, expires_at = $4
		WHERE id = $1
	`, serverID, newTariffID, limitsJSON, newExpires)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "update failed")
		return
	}
	audit(ctx, h.dbOf(ctx), claims.UserID, "server.tariff_change", "server:"+serverID, map[string]any{
		"from_tariff_id": row.tariffID, "to_tariff_id": newTariffID, "delta": delta,
	})
	writeJSON(w, http.StatusOK, map[string]any{
		"status":           "changed",
		"charged":          delta,
		"currency":         curCurrency,
		"restart_required": limitsDiffer(row.limits, limits),
	})
}

func (h *Handler) ServerTariffChangePreview(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	serverID := chi.URLParam(r, "id")
	access, err := h.resolveServerAccess(r.Context(), claims.UserID, claims.Role, serverID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "server not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	if !access.IsOwner {
		writeError(w, http.StatusForbidden, "only server owner can preview tariff")
		return
	}
	var body struct {
		TariffID any `json:"tariff_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	newTariffID := strings.TrimSpace(fmt.Sprint(body.TariffID))
	if newTariffID == "" || newTariffID == "<nil>" {
		writeError(w, http.StatusBadRequest, "tariff_id required")
		return
	}
	ctx := r.Context()
	row, err := h.loadServerTariffRow(ctx, serverID)
	if err != nil {
		writeError(w, http.StatusNotFound, "server not found")
		return
	}
	currentTariff, err := h.loadTariffLegacyJSON(ctx, row.tariffID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Тариф не найден")
		return
	}
	newTariff, err := h.loadTariffLegacyJSON(ctx, newTariffID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Тариф не найден")
		return
	}
	if !h.tariffAvailableForServer(ctx, newTariff, newTariffID, row.nodeID, row.gameID) {
		writeError(w, http.StatusBadRequest, "Этот тариф недоступен для сервера")
		return
	}
	order := limitsToRentOrder(row.limits)
	target := pricing.Resolve(newTariff, order)
	oldMonthly := pricing.MonthlyCost(currentTariff, order)
	newMonthly := pricing.MonthlyCost(newTariff, target)
	daysLeft := serverRemainingDays(row.expiresAt)
	delta, newExpires := tariffChangeBilling(oldMonthly, newMonthly, daysLeft, row.expiresAt)
	limits := tariffLimits(newTariff, target, row.limits)
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":                true,
		"from_tariff_id":    row.tariffID,
		"to_tariff_id":      newTariffID,
		"charged":           delta,
		"currency":          tariffCurrency(currentTariff),
		"days_left":         daysLeft,
		"new_expires_at":    newExpires,
		"target_resources":  limits,
		"current_resources": row.limits,
		"restart_required":  limitsDiffer(row.limits, limits),
	})
}

func (h *Handler) ServerTariffResources(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	serverID := chi.URLParam(r, "id")
	access, err := h.resolveServerAccess(r.Context(), claims.UserID, claims.Role, serverID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "server not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	if !access.IsOwner {
		writeError(w, http.StatusForbidden, "only server owner can change resources")
		return
	}
	if h.refuseWHMCSBilled(r.Context(), w, serverID) {
		return
	}

	var body struct {
		CPUCores float64 `json:"cpu_cores"`
		RAMGb    int     `json:"ram_gb"`
		DiskGb   int     `json:"disk_gb"`
		Slots    int     `json:"slots"`
		WalletID string  `json:"wallet_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if body.RAMGb < 1 || body.DiskGb < 1 || body.Slots < 1 {
		writeError(w, http.StatusBadRequest, "invalid resource values")
		return
	}

	ctx := r.Context()
	row, err := h.loadServerTariffRow(ctx, serverID)
	if err != nil {
		writeError(w, http.StatusNotFound, "server not found")
		return
	}
	if row.expiresAt != nil && !row.expiresAt.After(time.Now()) {
		writeError(w, http.StatusBadRequest, "Срок аренды истёк")
		return
	}

	tariffJSON, err := h.loadTariffLegacyJSON(ctx, row.tariffID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Тариф не найден")
		return
	}

	oldOrder := limitsToRentOrder(row.limits)
	oldMonthly := pricing.MonthlyCost(tariffJSON, oldOrder)
	if oldMonthly <= 0 {
		writeError(w, http.StatusBadRequest, "Не удалось рассчитать стоимость")
		return
	}

	target := pricing.Resolve(tariffJSON, pricing.RentOrder{
		Slots: body.Slots, CPUCores: int(math.Round(body.CPUCores)), RAMGb: body.RAMGb, DiskGb: body.DiskGb,
	})
	newMonthly := pricing.MonthlyCost(tariffJSON, target)
	if newMonthly <= 0 {
		writeError(w, http.StatusBadRequest, "Не удалось рассчитать стоимость")
		return
	}

	daysLeft := serverRemainingDays(row.expiresAt)
	delta, newExpires := tariffChangeBilling(oldMonthly, newMonthly, daysLeft, row.expiresAt)
	currency := tariffCurrency(tariffJSON)
	if delta > 0 {
		if _, err := h.debitWalletForRentSource(r, claims, body.WalletID, currency, delta,
			"Изменение ресурсов сервера", "server_resources", serverID); err != nil {
			writeError(w, http.StatusPaymentRequired, err.Error())
			return
		}
	}

	limits := tariffLimits(tariffJSON, target, row.limits)
	limitsJSON, _ := json.Marshal(limits)
	_, err = h.dbOf(ctx).Exec(ctx, `
		UPDATE core.servers SET limits = $2::jsonb, expires_at = $3 WHERE id = $1
	`, serverID, limitsJSON, newExpires)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "update failed")
		return
	}
	audit(ctx, h.dbOf(ctx), claims.UserID, "server.tariff_resources", "server:"+serverID, map[string]any{"delta": delta})
	writeJSON(w, http.StatusOK, map[string]any{
		"status":           "updated",
		"charged":          delta,
		"currency":         currency,
		"restart_required": limitsDiffer(row.limits, limits),
	})
}

func (h *Handler) ServerTariffResourcesPreview(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	serverID := chi.URLParam(r, "id")
	access, err := h.resolveServerAccess(r.Context(), claims.UserID, claims.Role, serverID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "server not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	if !access.IsOwner {
		writeError(w, http.StatusForbidden, "only server owner can preview resources")
		return
	}
	var body struct {
		CPUCores float64 `json:"cpu_cores"`
		RAMGb    int     `json:"ram_gb"`
		DiskGb   int     `json:"disk_gb"`
		Slots    int     `json:"slots"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	ctx := r.Context()
	row, err := h.loadServerTariffRow(ctx, serverID)
	if err != nil {
		writeError(w, http.StatusNotFound, "server not found")
		return
	}
	tariffJSON, err := h.loadTariffLegacyJSON(ctx, row.tariffID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Тариф не найден")
		return
	}
	oldOrder := limitsToRentOrder(row.limits)
	oldMonthly := pricing.MonthlyCost(tariffJSON, oldOrder)
	target := pricing.Resolve(tariffJSON, pricing.RentOrder{
		Slots: body.Slots, CPUCores: int(math.Round(body.CPUCores)), RAMGb: body.RAMGb, DiskGb: body.DiskGb,
	})
	newMonthly := pricing.MonthlyCost(tariffJSON, target)
	daysLeft := serverRemainingDays(row.expiresAt)
	delta, newExpires := tariffChangeBilling(oldMonthly, newMonthly, daysLeft, row.expiresAt)
	limits := tariffLimits(tariffJSON, target, row.limits)
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":                true,
		"charged":           delta,
		"currency":          tariffCurrency(tariffJSON),
		"days_left":         daysLeft,
		"new_expires_at":    newExpires,
		"target_resources":  limits,
		"current_resources": row.limits,
		"restart_required":  limitsDiffer(row.limits, limits),
	})
}

type serverTariffRow struct {
	tariffID  string
	nodeID    string
	gameID    string
	limits    map[string]any
	expiresAt *time.Time
}

func (h *Handler) loadServerTariffRow(ctx context.Context, serverID string) (*serverTariffRow, error) {
	var limitsRaw []byte
	row := &serverTariffRow{limits: map[string]any{}}
	var tariffID, nodeID, gameID *string
	err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT tariff_id::text, node_id::text, game_id, COALESCE(limits, '{}'::jsonb), expires_at
		FROM core.servers WHERE id = $1
	`, serverID).Scan(&tariffID, &nodeID, &gameID, &limitsRaw, &row.expiresAt)
	if err != nil {
		return nil, err
	}
	if tariffID != nil {
		row.tariffID = *tariffID
	}
	if nodeID != nil {
		row.nodeID = *nodeID
	}
	if gameID != nil {
		row.gameID = *gameID
	}
	_ = json.Unmarshal(limitsRaw, &row.limits)
	return row, nil
}

func limitsToRentOrder(limits map[string]any) pricing.RentOrder {
	order := pricing.RentOrder{
		Slots:    intFromAny(limits["slots"]),
		CPUCores: intFromAny(limits["cpu_cores"]),
		RAMGb:    intFromAny(limits["ram_gb"]),
		DiskGb:   intFromAny(limits["disk_gb"]),
	}
	if order.CPUCores <= 0 {
		order.CPUCores = int(math.Round(floatFromAny(limits["cpu"])))
	}
	if order.RAMGb <= 0 {
		if mb := intFromAny(limits["memory_mb"]); mb > 0 {
			order.RAMGb = mb / 1024
		}
	}
	if order.DiskGb <= 0 {
		if mb := intFromAny(limits["disk_mb"]); mb > 0 {
			order.DiskGb = mb / 1024
		}
	}
	return order
}

func tariffLimits(tariff map[string]any, order pricing.RentOrder, base map[string]any) map[string]any {
	limits := make(map[string]any, len(base)+8)
	for k, v := range base {
		limits[k] = v
	}
	limits["ram_gb"] = order.RAMGb
	limits["memory_mb"] = order.RAMGb * 1024
	limits["disk_gb"] = order.DiskGb
	limits["disk_mb"] = order.DiskGb * 1024
	shares := intFromAny(tariff["cpu_shares"])
	if intFromAny(tariff["cpu_cores"]) <= 0 && shares > 0 && !pricing.ResourceRange(tariff, "cpu").Configurable() {
		delete(limits, "cpu")
		delete(limits, "cpu_cores")
		limits["cpu_shares"] = shares
	} else {
		delete(limits, "cpu_shares")
		limits["cpu"] = order.CPUCores
		limits["cpu_cores"] = order.CPUCores
	}
	if order.Slots > 0 {
		limits["slots"] = order.Slots
	}
	return limits
}

func limitsDiffer(before, after map[string]any) bool {
	for _, key := range []string{"memory_mb", "disk_mb", "cpu", "cpu_shares", "slots"} {
		if floatFromAny(before[key]) != floatFromAny(after[key]) {
			return true
		}
	}
	return false
}

func serverRemainingDays(expiresAt *time.Time) float64 {
	if expiresAt == nil {
		return 0
	}
	sec := time.Until(*expiresAt).Seconds()
	if sec <= 0 {
		return 0
	}
	return sec / 86400.0
}

func tariffChangeBilling(oldMonthly, newMonthly, daysLeft float64, currentExpires *time.Time) (delta float64, newExpires *time.Time) {
	newExpires = currentExpires
	if daysLeft <= 0 {
		return 0, newExpires
	}
	if newMonthly > oldMonthly {
		delta = math.Round(((newMonthly-oldMonthly)/30.0)*daysLeft*100) / 100
		return delta, newExpires
	}
	remainingValue := (oldMonthly / 30.0) * daysLeft
	newDays := 0.0
	if newMonthly > 0 {
		newDays = remainingValue / (newMonthly / 30.0)
	}
	if newDays < 0 {
		newDays = 0
	}
	t := time.Now().Add(time.Duration(math.Round(newDays*86400)) * time.Second)
	newExpires = &t
	return 0, newExpires
}

func tariffCurrency(tariff map[string]any) string {
	if c, ok := tariff["currency"].(string); ok && c != "" {
		return c
	}
	return "RUB"
}

func (h *Handler) tariffAvailableForServer(ctx context.Context, tariff map[string]any, tariffID, nodeID, gameSlug string) bool {
	if !boolFromAny(tariff["active"]) {
		return false
	}
	if loc, _ := tariff["location_id"].(string); loc != "" && nodeID != "" && loc != nodeID {
		return false
	}
	slug := h.tariffGameSlug(ctx, tariffID)
	return slug == "" || gameSlug == "" || slug == gameSlug
}
