package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

// listEnrichedServers отдаёт серверы владельца, а при allOwners — все серверы
// арендатора. Признак передаётся явно, потому что широкий список нужен разделу
// администратора, а не всякому, кто открыл кабинет с админской ролью.
func (h *Handler) listEnrichedServers(ctx context.Context, tenantID, userID string, allOwners bool, limit int) []map[string]any {
	q := `
		SELECT s.id::text, s.name, COALESCE(s.ip_address, ''), COALESCE(s.primary_port, 0),
		       s.status, COALESCE(s.runtime_status, ''), COALESCE(s.provisioning_status, 'pending'),
		       COALESCE(s.provisioning_error, ''), s.expires_at, s.game_id,
		       g.name, g.slug, g.image_url,
		       n.name, COALESCE(n.country, ''), COALESCE(n.city, ''),
		       t.name, COALESCE(u.email, ''), COALESCE(s.is_blocked, false), COALESCE(s.blocked_reason, ''),
		       COALESCE(n.maintenance_mode, false), COALESCE(n.maintenance_reason, ''), n.maintenance_until,
		       COALESCE(s.node_id::text, '')
		FROM core.servers s
		LEFT JOIN core.games g ON g.tenant_id = s.tenant_id AND g.slug = s.game_id
		LEFT JOIN core.nodes n ON n.id = s.node_id
		LEFT JOIN core.tariffs t ON t.id = s.tariff_id
		LEFT JOIN core.users u ON u.id = s.user_id
		WHERE s.tenant_id = $1`
	args := []any{tenantID}
	if !allOwners {
		q += ` AND s.user_id = $2`
		args = append(args, userID)
	}
	q += ` ORDER BY s.created_at DESC`
	if limit > 0 {
		q += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := h.readerOf(ctx).Query(ctx, q, args...)
	if err != nil {
		return []map[string]any{}
	}
	defer rows.Close()

	list := []map[string]any{}
	for rows.Next() {
		if item := scanEnrichedServerRow(ctx, h, rows); item != nil {
			list = append(list, item)
		}
	}
	return list
}

func (h *Handler) attachServerLoad(ctx context.Context, servers []map[string]any) {
	if len(servers) == 0 {
		return
	}
	ids := make([]string, 0, len(servers))
	byID := make(map[string]map[string]any, len(servers))
	for _, s := range servers {
		id, _ := s["id"].(string)
		if id == "" {
			continue
		}
		ids = append(ids, id)
		byID[id] = s
	}
	if len(ids) == 0 {
		return
	}

	rows, err := h.readerOf(ctx).Query(ctx, `
		SELECT DISTINCT ON (server_id)
			server_id::text, cpu_pct, mem_used_mb, mem_limit_mb, ts
		FROM core.server_metric_points
		WHERE server_id = ANY($1::uuid[]) AND ts >= now() - interval '15 minutes'
		ORDER BY server_id, ts DESC
	`, ids)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id string
		var cpu float64
		var memUsed, memLimit int
		var ts time.Time
		if err := rows.Scan(&id, &cpu, &memUsed, &memLimit, &ts); err != nil {
			continue
		}
		item, ok := byID[id]
		if !ok {
			continue
		}
		item["cpu_percent"] = cpu
		if memLimit > 0 {
			item["ram_percent"] = float64(memUsed) / float64(memLimit) * 100
		}
		item["mem_used_mb"] = memUsed
		item["mem_limit_mb"] = memLimit
		item["metrics_at"] = ts.Format(time.RFC3339)
	}
}

func scanEnrichedServerRow(ctx context.Context, h *Handler, rows interface {
	Scan(dest ...any) error
}) map[string]any {
	var id, name, ip, status, runtime, prov, provError, gameID string
	var gameName, gameSlug, gameImage, locName, country, city, tariffName *string
	var ownerEmail, blockedReason string
	var isBlocked bool
	var port int
	var expiresAt *time.Time
	var maintenance bool
	var maintenanceReason string
	var maintenanceUntil *time.Time
	var nodeID string
	if err := rows.Scan(
		&id, &name, &ip, &port, &status, &runtime, &prov, &provError, &expiresAt, &gameID,
		&gameName, &gameSlug, &gameImage,
		&locName, &country, &city, &tariffName, &ownerEmail, &isBlocked, &blockedReason,
		&maintenance, &maintenanceReason, &maintenanceUntil, &nodeID,
	); err != nil {
		return nil
	}
	if live, ok := h.cache.GetServerStatus(ctx, id); ok {
		status = resolveEffectiveStatus(status, runtime, &live)
	} else {
		status = resolveEffectiveStatus(status, runtime, nil)
	}
	item := map[string]any{
		"id":                  id,
		"name":                name,
		"ip_address":          ip,
		"port":                port,
		"status":              status,
		"runtime_status":      runtime,
		"provisioning_status": prov,
		"provisioning_error":  nilIfEmpty(provError),
		"expires_at":          nil,
		"game":                nil,
		"location":            nil,
		"tariff":              nil,
		"owner_email":         ownerEmail,
		"is_blocked":          isBlocked,
		"blocked_reason":      nilIfEmpty(blockedReason),
		"node_id":             nodeID,
	}
	if expiresAt != nil {
		item["expires_at"] = expiresAt.Format(time.RFC3339)
	}
	if gameName != nil && *gameName != "" {
		img := (*string)(nil)
		if gameImage != nil && *gameImage != "" {
			img = gameImage
		}
		item["game"] = map[string]any{"name": *gameName, "slug": derefStr(gameSlug, gameID), "image": img}
	}
	if locName != nil && *locName != "" {
		location := map[string]any{"name": *locName, "country": derefStr(country, ""), "city": derefStr(city, "")}
		if maintenance {
			location["maintenance"] = map[string]any{
				"enabled": true,
				"reason":  maintenanceReason,
				"until":   timeOrNil(maintenanceUntil),
			}
		}
		item["location"] = location
	}
	if tariffName != nil && *tariffName != "" {
		item["tariff"] = map[string]any{"name": *tariffName}
	}
	return item
}

func (h *Handler) getEnrichedServer(ctx context.Context, tenantID, userID, role, serverID string) (map[string]any, *serverAccess, bool) {
	access, err := h.resolveServerAccess(ctx, tenantID, userID, role, serverID)
	if err != nil {
		return nil, nil, false
	}
	q := `
		SELECT s.id::text, s.name, COALESCE(s.ip_address, ''), COALESCE(s.primary_port, 0),
		       s.status, COALESCE(s.runtime_status, ''), COALESCE(s.provisioning_status, 'pending'),
		       COALESCE(s.provisioning_error, ''), s.expires_at, s.game_id,
		       g.name, g.slug, g.image_url,
		       n.name, COALESCE(n.country, ''), COALESCE(n.city, ''),
		       t.name, COALESCE(u.email, ''), COALESCE(s.is_blocked, false), COALESCE(s.blocked_reason, ''),
		       COALESCE(n.maintenance_mode, false), COALESCE(n.maintenance_reason, ''), n.maintenance_until,
		       COALESCE(s.node_id::text, ''),
		       COALESCE(s.auto_renew, false), COALESCE(s.rental_period_days, 30)
		FROM core.servers s
		LEFT JOIN core.games g ON g.tenant_id = s.tenant_id AND g.slug = s.game_id
		LEFT JOIN core.nodes n ON n.id = s.node_id
		LEFT JOIN core.tariffs t ON t.id = s.tariff_id
		LEFT JOIN core.users u ON u.id = s.user_id
		WHERE s.tenant_id = $1 AND s.id = $2::uuid`
	row := h.dbOf(ctx).QueryRow(ctx, q, tenantID, serverID)
	var id, name, ip, status, runtime, prov, provError, gameID string
	var gameName, gameSlug, gameImage, locName, country, city, tariffName *string
	var ownerEmail, blockedReason string
	var isBlocked bool
	var port int
	var expiresAt *time.Time
	var maintenance bool
	var maintenanceReason string
	var maintenanceUntil *time.Time
	var autoRenew bool
	var periodDays int
	var detailNodeID string
	if err := row.Scan(
		&id, &name, &ip, &port, &status, &runtime, &prov, &provError, &expiresAt, &gameID,
		&gameName, &gameSlug, &gameImage,
		&locName, &country, &city, &tariffName, &ownerEmail, &isBlocked, &blockedReason,
		&maintenance, &maintenanceReason, &maintenanceUntil, &detailNodeID, &autoRenew, &periodDays,
	); err != nil {
		return nil, nil, false
	}
	if live, ok := h.cache.GetServerStatus(ctx, id); ok {
		status = resolveEffectiveStatus(status, runtime, &live)
	} else {
		status = resolveEffectiveStatus(status, runtime, nil)
	}
	item := map[string]any{
		"id": id, "name": name, "ip_address": ip, "port": port,
		"status": status, "runtime_status": runtime, "provisioning_status": prov,
		"provisioning_error": nilIfEmpty(provError),
		"expires_at":         nil, "game": nil, "location": nil, "tariff": nil, "game_id": gameID,
		"owner_email": ownerEmail, "is_blocked": isBlocked, "blocked_reason": nilIfEmpty(blockedReason),
		"auto_renew": autoRenew, "rental_period_days": periodDays,
		"node_id": detailNodeID,
	}
	if expiresAt != nil {
		item["expires_at"] = expiresAt.Format(time.RFC3339)
	}
	if gameName != nil && *gameName != "" {
		img := (*string)(nil)
		if gameImage != nil && *gameImage != "" {
			img = gameImage
		}
		item["game"] = map[string]any{"name": *gameName, "slug": derefStr(gameSlug, gameID), "image": img}
	}
	if locName != nil && *locName != "" {
		location := map[string]any{"name": *locName, "country": derefStr(country, ""), "city": derefStr(city, "")}
		if maintenance {
			location["maintenance"] = map[string]any{
				"enabled": true,
				"reason":  maintenanceReason,
				"until":   timeOrNil(maintenanceUntil),
			}
		}
		item["location"] = location
	}
	if tariffName != nil && *tariffName != "" {
		item["tariff"] = map[string]any{"name": *tariffName}
	}
	return item, access, true
}

func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func (h *Handler) GetServerDetail(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := chi.URLParam(r, "id")
	item, access, found := h.getEnrichedServer(r.Context(), claims.TenantID, claims.UserID, claims.Role, id)
	if !found {
		writeError(w, http.StatusNotFound, "server not found")
		return
	}
	h.enrichServerDetail(r.Context(), claims.TenantID, id, item, access)
	writeJSON(w, http.StatusOK, item)
}

func (h *Handler) enrichServerDetail(ctx context.Context, tenantID, serverID string, item map[string]any, access *serverAccess) {
	var limitsRaw, configRaw []byte
	var autoStart bool
	var gameVersionID *string
	var priceMonthly *float64
	var tariffCurrency *string
	var ramMB, diskMB, slotsMax *int
	var tariffID, tariffBilling *string
	var tariffRenewal []byte
	var tariffMeta []byte
	var createdAt time.Time
	var ownerID *string
	var isBlocked bool
	var blockedReason string

	err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT COALESCE(s.limits, '{}'::jsonb), COALESCE(s.config, '{}'::jsonb),
		       COALESCE(s.auto_start, false), s.game_version_id::text,
		       t.price_monthly, t.currency, t.ram_mb, t.disk_mb, t.slots_max,
		       s.tariff_id::text, t.billing_type, COALESCE(t.renewal_periods, '[]'::jsonb),
		       COALESCE(t.meta, '{}'::jsonb), s.created_at, s.user_id::text,
		       COALESCE(s.is_blocked, false), COALESCE(s.blocked_reason, '')
		FROM core.servers s
		LEFT JOIN core.tariffs t ON t.id = s.tariff_id
		WHERE s.id = $1 AND s.tenant_id = $2
	`, serverID, tenantID).Scan(
		&limitsRaw, &configRaw, &autoStart, &gameVersionID,
		&priceMonthly, &tariffCurrency, &ramMB, &diskMB, &slotsMax,
		&tariffID, &tariffBilling, &tariffRenewal, &tariffMeta, &createdAt, &ownerID,
		&isBlocked, &blockedReason,
	)
	if err != nil {
		return
	}

	limits := map[string]any{}
	_ = json.Unmarshal(limitsRaw, &limits)
	config := map[string]any{}
	_ = json.Unmarshal(configRaw, &config)

	item["limits"] = limits
	item["auto_start_enabled"] = autoStart
	item["created_at"] = createdAt.Format(time.RFC3339)
	item["is_blocked"] = isBlocked
	item["blocked_reason"] = nilIfEmpty(blockedReason)
	if ownerID != nil && *ownerID != "" {
		item["user_id"] = *ownerID
	}
	if gameVersionID != nil && *gameVersionID != "" {
		item["game_version_id"] = *gameVersionID
	}

	startupParams := ""
	if v, ok := config["startup_params"]; ok {
		switch t := v.(type) {
		case string:
			startupParams = t
		case map[string]any:
			if s, ok := t["startup_params"].(string); ok {
				startupParams = s
			} else {
				b, _ := json.Marshal(t)
				startupParams = string(b)
			}
		default:
			b, _ := json.Marshal(t)
			startupParams = string(b)
		}
	}
	item["startup_params"] = startupParams

	if tariff, ok := item["tariff"].(map[string]any); ok && tariff != nil {
		if tariffID != nil && *tariffID != "" {
			tariff["id"] = *tariffID
		}
		if tariffBilling != nil && *tariffBilling != "" {
			tariff["billing_type"] = *tariffBilling
		}
		if priceMonthly != nil {
			tariff["price_monthly"] = *priceMonthly
			tariff["base_price_monthly"] = *priceMonthly
		}
		if tariffCurrency != nil && *tariffCurrency != "" {
			tariff["currency"] = *tariffCurrency
		}
		if ramMB != nil {
			tariff["ram_mb"] = *ramMB
		}
		if diskMB != nil {
			tariff["disk_mb"] = *diskMB
		}
		if slotsMax != nil {
			tariff["slots"] = *slotsMax
		}
		var renewal []int
		_ = json.Unmarshal(tariffRenewal, &renewal)
		if len(renewal) > 0 {
			tariff["renewal_periods"] = renewal
		}
		meta := map[string]any{}
		_ = json.Unmarshal(tariffMeta, &meta)
		for _, key := range []string{
			"price_per_slot", "price_per_cpu_core", "price_per_ram_gb", "price_per_disk_gb", "antiddos_price",
		} {
			if v, ok := meta[key]; ok {
				tariff[key] = v
			}
		}
		item["tariff"] = tariff
	}

	gameID, _ := item["game_id"].(string)
	item["available_game_versions"] = h.listActiveGameVersions(ctx, tenantID, gameID)
	item["steam_updatable"] = h.gameSteamUpdatable(ctx, tenantID, gameID)
	item["viewer_permissions"] = viewerPermissionsForAccess(access)
	h.enrichServerLiveFields(ctx, tenantID, serverID, item)
	if mysql, ok := config["mysql"].(map[string]any); ok {
		item["mysql_host"] = mysql["host"]
		item["mysql_port"] = mysql["port"]
		item["mysql_database"] = mysql["database"]
		item["mysql_username"] = mysql["username"]
		item["mysql_password_decrypted"] = mysql["password"]
		item["mysql_instance_key"] = mysql["mysql_instance_key"]
	}
	rows, err := h.dbOf(ctx).Query(ctx, `
		SELECT id::text, port, protocol, purpose, created_at::text
		FROM core.server_ports
		WHERE tenant_id = $1 AND server_id = $2
		ORDER BY port ASC, created_at ASC
	`, tenantID, serverID)
	if err == nil {
		defer rows.Close()
		ports := make([]map[string]any, 0)
		for rows.Next() {
			var pid, proto, purpose, created string
			var port int
			if scanErr := rows.Scan(&pid, &port, &proto, &purpose, &created); scanErr == nil {
				ports = append(ports, map[string]any{
					"id":         pid,
					"port":       port,
					"protocol":   proto,
					"purpose":    purpose,
					"is_primary": port == intFromAny(item["port"]),
					"created_at": created,
				})
			}
		}
		item["extra_ports"] = ports
	}
}

func (h *Handler) listActiveGameVersions(ctx context.Context, tenantID, gameSlug string) []map[string]any {
	if gameSlug == "" {
		return []map[string]any{}
	}
	rows, err := h.dbOf(ctx).Query(ctx, `
		SELECT gv.id::text, gv.version, COALESCE(gv.source_type, ''), COALESCE(gv.meta->>'note', '')
		FROM core.game_versions gv
		JOIN core.games g ON g.id = gv.game_id AND g.tenant_id = gv.tenant_id
		WHERE gv.tenant_id = $1 AND g.slug = $2 AND gv.active = true
		ORDER BY gv.sort_order ASC, gv.created_at DESC
	`, tenantID, gameSlug)
	if err != nil {
		return []map[string]any{}
	}
	defer rows.Close()
	list := []map[string]any{}
	for rows.Next() {
		var id, version, sourceType, note string
		if rows.Scan(&id, &version, &sourceType, &note) == nil {
			item := map[string]any{
				"id": id, "name": version, "version": version,
				"source_type":    sourceType,
				"manual_install": sourceType == "docker",
			}
			if note != "" {
				item["install_note"] = note
			}
			list = append(list, item)
		}
	}
	return list
}

func (h *Handler) gameSteamUpdatable(ctx context.Context, tenantID, gameSlug string) bool {
	if gameSlug == "" {
		return false
	}
	var meta []byte
	if err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT COALESCE(meta, '{}'::jsonb) FROM core.games
		WHERE tenant_id = $1 AND slug = $2
	`, tenantID, gameSlug).Scan(&meta); err != nil {
		return false
	}
	var m map[string]any
	if json.Unmarshal(meta, &m) != nil {
		return false
	}
	v, ok := m["steam_updatable"]
	if !ok {
		return false
	}
	switch t := v.(type) {
	case bool:
		return t
	case float64:
		return t != 0
	default:
		return fmt.Sprint(t) == "true" || fmt.Sprint(t) == "1"
	}
}

func (h *Handler) MyServers(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ctx := r.Context()

	// «Мои серверы» — это именно свои: у сотрудника раздел показывал серверы
	// всех клиентов арендатора, хотя для них есть раздел администратора.
	ownerFilter := ` AND user_id = $2`
	args := []any{claims.TenantID, claims.UserID}

	var total, active int
	_ = h.dbOf(ctx).QueryRow(ctx, `SELECT COUNT(*) FROM core.servers WHERE tenant_id = $1`+ownerFilter, args...).Scan(&total)
	_ = h.dbOf(ctx).QueryRow(ctx, `
		SELECT COUNT(*) FROM core.servers
		WHERE tenant_id = $1`+ownerFilter+` AND (status = 'running' OR runtime_status = 'running')
	`, args...).Scan(&active)

	var expiringSoon int
	_ = h.dbOf(ctx).QueryRow(ctx, `
		SELECT COUNT(*) FROM core.servers
		WHERE tenant_id = $1`+ownerFilter+`
		  AND expires_at IS NOT NULL
		  AND expires_at >= now()
		  AND expires_at < now() + interval '7 days'
	`, args...).Scan(&expiringSoon)

	servers := h.listEnrichedServers(ctx, claims.TenantID, claims.UserID, false, 0)
	h.attachServerLoad(ctx, servers)
	writeJSON(w, http.StatusOK, map[string]any{
		"servers":       servers,
		"total":         total,
		"active_count":  active,
		"expiring_soon": expiringSoon,
	})
}
