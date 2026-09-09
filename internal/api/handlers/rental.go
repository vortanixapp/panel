package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vortanixapp/panel/internal/api/jobwake"
	"github.com/vortanixapp/panel/internal/api/paneljwt"
	"github.com/vortanixapp/panel/internal/api/payments"
	"github.com/vortanixapp/panel/internal/api/pricing"
	"github.com/vortanixapp/panel/internal/api/relay"
	"github.com/vortanixapp/panel/pkg/gamecatalog"
	"github.com/vortanixapp/panel/pkg/portalloc"
)

func (h *Handler) RentServerForm(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ctx := r.Context()
	q := r.URL.Query()

	games := h.queryRentGames(ctx)
	tariffs := h.loadRentTariffs(ctx, q.Get("game_id"), q.Get("location_id"))
	nodes := h.queryRentNodes(ctx)

	resp := map[string]any{
		"games":         games,
		"tariffs":       tariffs,
		"nodes":         nodes,
		"locations":     nodes,
		"game_versions": h.queryGameVersions(ctx),
	}

	tariffID := q.Get("tariff_id")
	periodStr := q.Get("period")
	if tariffID != "" && periodStr != "" {
		periodDays, _ := strconv.Atoi(periodStr)
		if periodDays <= 0 {
			periodDays = 30
		}
		order := buildRentOrderFromQuery(q)
		tariffJSON, err := h.loadTariffLegacyJSON(ctx, tariffID)
		if err == nil {
			if msg := tariffMeetsGame(q.Get("game_id"), tariffJSON, order); msg != "" {
				resp["tariff_warning"] = msg
			}

			baseCost := pricing.CalculateRentCost(tariffJSON, order, periodDays)
			promoCode := q.Get("promo_code")
			promo, promoErr := payments.PickPromotion(
				ctx, h.dbOf(ctx), payments.ApplyRent, claims.UserID,
				promoCode, tariffID, q.Get("game_id"), q.Get("location_id"), baseCost,
			)
			preview := payments.ApplyRentDiscount(promo, baseCost)
			if promoErr != "" {
				preview.Valid = false
				preview.Error = &promoErr
				preview.FinalCost = baseCost
			}
			resp["calculated_cost"] = preview.FinalCost
			resp["base_cost"] = baseCost
			resp["order"] = rentOrderToMap(order)
			resp["promo_preview"] = preview
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) loadRentTariffs(ctx context.Context, gameID, locationID string) []map[string]any {
	q := tariffSelectSQL + ` WHERE t.active = true`
	args := []any{}
	argN := 1
	if gameID != "" {
		q += fmt.Sprintf(` AND (t.game_id IS NULL OR t.game_id::text = $%d OR EXISTS (
			SELECT 1 FROM core.games g WHERE g.id = t.game_id AND (g.id::text = $%d OR g.slug = $%d)
		))`, argN, argN, argN)
		args = append(args, gameID)
		argN++
	}
	if locationID != "" {
		q += fmt.Sprintf(` AND (t.node_id IS NULL OR t.node_id::text = $%d)`, argN)
		args = append(args, locationID)
	}
	q += ` ORDER BY t.position ASC, t.created_at ASC`
	rows, err := h.dbOf(ctx).Query(ctx, q, args...)
	if err != nil {
		return []map[string]any{}
	}
	defer rows.Close()
	list := []map[string]any{}
	for rows.Next() {
		row, err := scanTariffRow(rows)
		if err != nil {
			continue
		}
		list = append(list, tariffToLegacyJSON(row))
	}
	return list
}

func (h *Handler) loadTariffLegacyJSON(ctx context.Context, tariffID string) (map[string]any, error) {
	q := tariffSelectSQL + ` WHERE t.id = $1`
	row, err := scanTariffRow(h.dbOf(ctx).QueryRow(ctx, q, tariffID))
	if err != nil {
		return nil, err
	}
	return tariffToLegacyJSON(row), nil
}

func buildRentOrderFromQuery(q map[string][]string) pricing.RentOrder {
	get := func(key string) string {
		if v := q[key]; len(v) > 0 {
			return v[0]
		}
		return ""
	}
	slots, _ := strconv.Atoi(get("slots"))
	cpu, _ := strconv.Atoi(get("cpu_cores"))
	ram, _ := strconv.Atoi(get("ram_gb"))
	disk, _ := strconv.Atoi(get("disk_gb"))
	antiddos := get("antiddos_enabled") == "1" || get("antiddos_enabled") == "true"
	return pricing.RentOrder{
		Slots: slots, CPUCores: cpu, RAMGb: ram, DiskGb: disk, AntiddosEnabled: antiddos,
	}
}

func buildRentOrderFromBody(body map[string]any) pricing.RentOrder {
	return pricing.RentOrder{
		Slots:           intFromAny(body["slots"]),
		CPUCores:        intFromAny(body["cpu_cores"]),
		RAMGb:           intFromAny(body["ram_gb"]),
		DiskGb:          intFromAny(body["disk_gb"]),
		AntiddosEnabled: boolFromAny(body["antiddos_enabled"]),
	}
}

func rentOrderToMap(order pricing.RentOrder) map[string]any {
	return map[string]any{
		"slots": order.Slots, "cpu_cores": order.CPUCores,
		"ram_gb": order.RAMGb, "disk_gb": order.DiskGb,
		"antiddos_enabled": order.AntiddosEnabled,
	}
}

func boolFromAny(v any) bool {
	switch b := v.(type) {
	case bool:
		return b
	case float64:
		return b != 0
	case string:
		return b == "1" || b == "true"
	default:
		return false
	}
}

func queryPairs(ctx context.Context, h *Handler, q string) []map[string]any {
	rows, err := h.dbOf(ctx).Query(ctx, q)
	if err != nil {
		return []map[string]any{}
	}
	defer rows.Close()
	list := []map[string]any{}
	for rows.Next() {
		var id, name string
		if rows.Scan(&id, &name) == nil {
			list = append(list, map[string]any{"id": id, "name": name})
		}
	}
	return list
}

func (h *Handler) queryRentGames(ctx context.Context) []map[string]any {
	rows, err := h.dbOf(ctx).Query(ctx, `
		SELECT g.id::text, g.slug, g.name,
			(SELECT MIN(t.price_monthly) FROM core.tariffs t
			  WHERE t.active = true
			    AND (t.game_id IS NULL OR t.game_id = g.id)),
			(SELECT COUNT(*)::int FROM core.servers s
			  WHERE s.game_id = g.slug)
		FROM core.games g
		WHERE g.active = true
		ORDER BY g.name
	`)
	if err != nil {
		return []map[string]any{}
	}
	defer rows.Close()
	list := []map[string]any{}
	for rows.Next() {
		var id, slug, name string
		var minPrice *float64
		var serversCount int
		if rows.Scan(&id, &slug, &name, &minPrice, &serversCount) != nil {
			continue
		}
		item := map[string]any{"id": id, "slug": slug, "name": name, "servers_count": serversCount}
		if minPrice != nil {
			item["min_price"] = *minPrice
		}
		if g, ok := gamecatalog.Resolve(gamecatalog.Normalize(slug)); ok {
			if g.Install.SourceType == gamecatalog.SourceDocker {
				item["manual_install"] = true
				item["manual_install_note"] = g.Install.Note
			}
			item["min_ram_mb"] = g.MinRAMMB
			item["min_disk_mb"] = g.MinDiskMB
			item["min_cpu"] = g.MinCPU
			if g.MaxSlots > 0 {
				item["max_slots"] = g.MaxSlots
			}
		}
		list = append(list, item)
	}
	return list
}

func (h *Handler) queryRentNodes(ctx context.Context) []map[string]any {
	rows, err := h.dbOf(ctx).Query(ctx, `
		SELECT n.id::text, n.name, COALESCE(n.country, ''),
			COALESCE(n.meta->>'code', n.fqdn, ''),
			COALESCE(d.status, 'unknown'), d.last_seen_at,
			(SELECT COUNT(*)::int FROM core.servers s WHERE s.node_id = n.id),
			COALESCE(n.maintenance_mode, false), COALESCE(n.maintenance_reason, '')
		FROM core.nodes n
		LEFT JOIN core.node_daemons d ON d.node_id = n.id
		WHERE COALESCE(n.is_active, n.active, true) = true
		ORDER BY COALESCE(n.sort_order, 0), n.name
	`)
	if err != nil {
		return []map[string]any{}
	}
	defer rows.Close()
	list := []map[string]any{}
	for rows.Next() {
		var id, name, country, code, daemonStatus string
		var daemonLastSeen *time.Time
		var serversCount int
		var maintenance bool
		var maintenanceReason string
		if rows.Scan(&id, &name, &country, &code, &daemonStatus, &daemonLastSeen, &serversCount,
			&maintenance, &maintenanceReason) != nil {
			continue
		}
		list = append(list, map[string]any{
			"id": id, "name": name, "country": country, "code": code,
			"is_online":          agentDaemonOnline(daemonStatus, daemonLastSeen),
			"servers_count":      serversCount,
			"maintenance_mode":   maintenance,
			"maintenance_reason": maintenanceReason,
		})
	}
	return list
}

func (h *Handler) queryGameVersions(ctx context.Context) []map[string]any {
	rows, err := h.dbOf(ctx).Query(ctx, `
		SELECT gv.id::text, gv.game_id::text, COALESCE(g.slug, ''), COALESCE(gv.version, ''), COALESCE(gv.source_type, '')
		FROM core.game_versions gv
		JOIN core.games g ON g.id = gv.game_id
		WHERE gv.active = true AND g.active = true
		ORDER BY g.slug ASC, gv.sort_order ASC, gv.created_at DESC
	`)
	if err != nil {
		return []map[string]any{}
	}
	defer rows.Close()
	list := []map[string]any{}
	for rows.Next() {
		var id, gameID, gameSlug, version, sourceType string
		if rows.Scan(&id, &gameID, &gameSlug, &version, &sourceType) == nil {
			name := strings.TrimSpace(version)
			if name == "" {
				name = "default"
			}
			list = append(list, map[string]any{
				"id":          id,
				"game_id":     gameID,
				"game_slug":   gameSlug,
				"name":        name,
				"version":     version,
				"source_type": sourceType,
			})
		}
	}
	return list
}

func querySlugPairs(ctx context.Context, h *Handler, q string) []map[string]any {
	return queryPairs(ctx, h, q)
}

func (h *Handler) RentServerSubmit(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var body struct {
		NodeID        string `json:"node_id"`
		LocationID    string `json:"location_id"`
		GameID        string `json:"game_id"`
		GameVersionID string `json:"game_version_id"`
		TariffID      string `json:"tariff_id"`
		Name          string `json:"name"`
		Period        int    `json:"period"`
		Slots         int    `json:"slots"`
		CPUCores      int    `json:"cpu_cores"`
		RAMGb         int    `json:"ram_gb"`
		DiskGb        int    `json:"disk_gb"`
		Antiddos      bool   `json:"antiddos_enabled"`
		PromoCode     string `json:"promo_code"`
		WalletID      string `json:"wallet_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	nodeID := body.NodeID
	if nodeID == "" {
		nodeID = body.LocationID
	}
	if nodeID == "" || body.Name == "" {
		writeError(w, http.StatusBadRequest, "node_id and name required")
		return
	}
	if h.nodeInMaintenance(r.Context(), nodeID) {
		writeCodedError(w, http.StatusConflict, "node_maintenance",
			"на этой локации идут технические работы, выберите другую")
		return
	}
	if reason := h.nodeCapacityReason(r.Context(), nodeID, body.GameID); reason != "" {
		writeCodedError(w, http.StatusConflict, "node_capacity", reason+", выберите другую")
		return
	}

	periodDays := 30
	if body.Period > 0 {
		periodDays = body.Period
	}

	gameID, gameOK := resolveGameSlug(r.Context(), h, body.GameID)
	if !gameOK {
		writeCodedError(w, http.StatusBadRequest, "game_unknown",
			"игра не найдена в каталоге")
		return
	}

	if body.TariffID == "" {
		writeCodedError(w, http.StatusBadRequest, "tariff_required",
			"выберите тариф: без него нельзя определить ни цену, ни ресурсы сервера")
		return
	}

	limits := gamecatalog.DefaultLimits(gameID)
	var rentPrice float64
	var currency string = "RUB"
	var promoID string

	order := pricing.RentOrder{
		Slots: body.Slots, CPUCores: body.CPUCores, RAMGb: body.RAMGb,
		DiskGb: body.DiskGb, AntiddosEnabled: body.Antiddos,
	}

	{
		tariffJSON, err := h.loadTariffLegacyJSON(r.Context(), body.TariffID)
		if err != nil {
			writeCodedError(w, http.StatusBadRequest, "tariff_unknown", "тариф не найден")
			return
		}
		{
			if msg := tariffMeetsGame(gameID, tariffJSON, order); msg != "" {
				writeCodedError(w, http.StatusBadRequest, "tariff_too_small", msg)
				return
			}

			baseCost := pricing.CalculateRentCost(tariffJSON, order, periodDays)
			promo, promoErr := payments.PickPromotion(
				r.Context(), h.dbOf(r.Context()), payments.ApplyRent, claims.UserID,
				body.PromoCode, body.TariffID, body.GameID, nodeID, baseCost,
			)
			if promoErr != "" {
				writeError(w, http.StatusBadRequest, promoErr)
				return
			}
			preview := payments.ApplyRentDiscount(promo, baseCost)
			rentPrice = preview.FinalCost
			promoID = preview.PromoID
			currency = fmt.Sprint(tariffJSON["currency"])
			if currency == "" {
				currency = "RUB"
			}
			if ram := intFromAny(tariffJSON["ram_gb"]); ram > 0 {
				limits["memory_mb"] = ram * 1024
			}
			if disk := intFromAny(tariffJSON["disk_gb"]); disk > 0 {
				limits["disk_mb"] = disk * 1024
			}
			if cpu := intFromAny(tariffJSON["cpu_cores"]); cpu > 0 {
				limits["cpu"] = cpu
			}
		}
	}
	if order.Slots > 0 {
		limits["slots"] = order.Slots
	} else if body.Slots > 0 {
		limits["slots"] = body.Slots
	}

	rentTxID := ""
	if rentPrice > 0 {
		var debitErr error
		rentTxID, debitErr = h.debitWalletForRentSource(r, claims, body.WalletID, currency,
			rentPrice, "Server rent: "+body.Name, "server_rent", "")
		if debitErr != nil {
			writeError(w, http.StatusPaymentRequired, debitErr.Error())
			return
		}
		if promoID != "" {
			_ = payments.IncrementPromoUsage(r.Context(), h.dbOf(r.Context()), promoID)
		}
	}
	limitsJSON, _ := json.Marshal(limits)
	expires := time.Now().Add(time.Duration(periodDays) * 24 * time.Hour)

	var id string
	err := h.dbOf(r.Context()).QueryRow(r.Context(), `
		INSERT INTO core.servers ( node_id, game_id, game_version_id, name, limits, user_id, tariff_id, provisioning_status, expires_at, rental_period_days, config)
		VALUES ( $1::uuid, $2, NULLIF($3, '')::uuid, $4, $5::jsonb, $6::uuid, NULLIF($7, '')::uuid, 'provisioning', $8, $9,
		        jsonb_build_object('startup_params', $10::text))
		RETURNING id::text
	`, nodeID, gameID, body.GameVersionID, body.Name, limitsJSON, claims.UserID, body.TariffID, expires, periodDays,
		h.defaultStartupParams(r.Context(), gameID)).Scan(&id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create server")
		return
	}
	if rentTxID != "" {
		_, _ = h.dbOf(r.Context()).Exec(r.Context(), `
			UPDATE core.transactions SET source_id = $2::uuid WHERE id = $1
		`, rentTxID, id)
	}
	_, _ = h.dbOf(r.Context()).Exec(r.Context(), `
		INSERT INTO core.jobs ( type, status, payload)
		VALUES ( 'provision_server', 'pending', $1::jsonb)
	`, mustJSON(map[string]string{"server_id": id}))
	jobwake.Notify("provision_server")
	h.emitWebhook(r.Context(), "server.created", map[string]any{
		"server_id": id, "server_name": body.Name, "game_id": gameID,
		"node_id": nodeID, "user_id": claims.UserID,
	})

	var lim map[string]any
	_ = json.Unmarshal(limitsJSON, &lim)
	_, _ = h.dbOf(r.Context()).Exec(r.Context(), `
		UPDATE core.servers s SET ip_address = n.fqdn
		FROM core.nodes n
		WHERE s.id = $1 AND n.id = s.node_id AND COALESCE(s.ip_address, '') = ''
	`, id)
	port := 0
	if gameID != "test" {
		assigned, err := portalloc.Assign(r.Context(), h.dbOf(r.Context()), nodeID, id, gameID)
		if err != nil {
			log.Printf("rental: не удалось выдать порт серверу %s (%s): %v", id, gameID, err)
		} else {
			port = assigned
		}
	}
	payload := map[string]any{
		"power_action": "start",
		"name":         body.Name,
		"game_id":      gameID,
		"limits":       lim,
	}
	if port > 0 {
		payload["primary_port"] = port
	}
	if bindIP := h.serverBindIP(r.Context(), id); bindIP != "" {
		payload["bind_ip"] = bindIP
	}
	if img := h.resolveDockerImage(r.Context(), id); img != "" {
		payload["docker_image"] = img
	}
	if spec := h.resolveInstallSpec(r.Context(), id); spec != nil {
		payload["install"] = spec
	}
	cmdID := uuid.NewString()
	startErr := h.relay.SendCommand(r.Context(), nodeID, relay.CommandRequest{
		CommandID: cmdID,
		Action:    "power",
		ServerID:  id,
		Payload:   payload,
	})
	if startErr == nil {
		_, _ = h.dbOf(r.Context()).Exec(r.Context(), `
			UPDATE core.servers SET status = 'starting', provisioning_status = 'provisioning', provisioning_error = NULL WHERE id = $1
		`, id)
		_, _ = h.dbOf(r.Context()).Exec(r.Context(), `
			UPDATE core.jobs SET status = 'completed', result = '{"ok":true}'::jsonb
			WHERE type = 'provision_server' AND status = 'pending'
			  AND payload->>'server_id' = $1
		`, id)
	} else {
		_, _ = h.dbOf(r.Context()).Exec(r.Context(), `
			UPDATE core.servers SET provisioning_status = 'failed', provisioning_error = $2 WHERE id = $1
		`, id, startErr.Error())
	}

	_ = h.cache.InvalidateTenantServers(r.Context())
	writeJSON(w, http.StatusCreated, map[string]any{
		"id":          id,
		"server_id":   id,
		"status":      "provisioning",
		"period_days": periodDays,
		"expires_at":  expires.Format(time.RFC3339),
		"cost":        rentPrice,
	})
}

func resolveGameSlug(ctx context.Context, h *Handler, raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}

	var slug string
	err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT slug FROM core.games
		WHERE active AND (id::text = $1 OR slug = $1)
		LIMIT 1
	`, raw).Scan(&slug)
	if err == nil && slug != "" {
		return slug, true
	}

	if g, ok := gamecatalog.Resolve(gamecatalog.Normalize(raw)); ok {
		return g.Key, true
	}
	return "", false
}

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

func (h *Handler) debitWalletForRent(r *http.Request, claims *paneljwt.Claims, walletID, currency string, amount float64, desc string) error {
	_, err := h.debitWalletForRentSource(r, claims, walletID, currency, amount, desc, "", "")
	return err
}

func (h *Handler) debitWalletForRentSource(
	r *http.Request, claims *paneljwt.Claims,
	walletID, currency string, amount float64, desc, sourceType, sourceID string,
) (string, error) {
	ctx := r.Context()
	if walletID == "" {
		_ = h.dbOf(ctx).QueryRow(ctx, `
			SELECT id::text FROM core.wallets WHERE user_id = $1 AND currency = $2 LIMIT 1
		`, claims.UserID, currency).Scan(&walletID)
	}
	if walletID == "" {
		return "", fmt.Errorf("wallet not found")
	}
	tx, err := h.dbOf(ctx).Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)
	var balance float64
	if err := tx.QueryRow(ctx, `
		SELECT balance FROM core.wallets
		WHERE id = $1 AND user_id = $2 AND currency = $3
		FOR UPDATE
	`, walletID, claims.UserID, currency).Scan(&balance); err != nil {
		return "", fmt.Errorf("кошелёк в валюте %s не найден", currency)
	}
	if balance < amount {
		return "", fmt.Errorf("insufficient balance")
	}
	if _, err := tx.Exec(ctx, `UPDATE core.wallets SET balance = balance - $2, updated_at = now() WHERE id = $1`, walletID, amount); err != nil {
		return "", err
	}
	var txID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO core.transactions ( wallet_id, type, amount, description, source_type, source_id)
		VALUES ( $1, 'debit', $2, $3, NULLIF($4, ''), $5::uuid)
		RETURNING id::text
	`, walletID, -amount, desc, sourceType, nullableUUID(sourceID)).Scan(&txID); err != nil {
		return "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return txID, nil
}

func tariffMeetsGame(gameSlug string, tariff map[string]any, order pricing.RentOrder) string {
	g, ok := gamecatalog.Resolve(gamecatalog.Normalize(gameSlug))
	if !ok {
		return ""
	}

	ramMB := intFromAny(tariff["ram_gb"]) * 1024
	if order.RAMGb*1024 > ramMB {
		ramMB = order.RAMGb * 1024
	}
	diskMB := intFromAny(tariff["disk_gb"]) * 1024
	if order.DiskGb*1024 > diskMB {
		diskMB = order.DiskGb * 1024
	}
	cpu := intFromAny(tariff["cpu_cores"])
	if order.CPUCores > cpu {
		cpu = order.CPUCores
	}

	if ramMB > 0 && g.MinRAMMB > 0 && ramMB < g.MinRAMMB {
		return fmt.Sprintf("%s требует минимум %d ГБ оперативной памяти, в тарифе — %d ГБ",
			g.Name, g.MinRAMMB/1024, ramMB/1024)
	}
	if diskMB > 0 && g.MinDiskMB > 0 && diskMB < g.MinDiskMB {
		return fmt.Sprintf("%s требует минимум %d ГБ дискового пространства, в тарифе — %d ГБ",
			g.Name, g.MinDiskMB/1024, diskMB/1024)
	}
	if cpu > 0 && g.MinCPU > 0 && float64(cpu) < g.MinCPU {
		return fmt.Sprintf("%s требует минимум %.0f ядра процессора, в тарифе — %d",
			g.Name, g.MinCPU, cpu)
	}

	if g.MaxSlots > 0 && order.Slots > g.MaxSlots {
		return fmt.Sprintf("%s поддерживает не больше %d слотов", g.Name, g.MaxSlots)
	}
	return ""
}
