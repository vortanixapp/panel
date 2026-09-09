package handlers

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

const monitoringSparkPoints = 24

type monitoringRow struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	GameID        string   `json:"game_id"`
	GameName      string   `json:"game_name"`
	Version       string   `json:"version"`
	Map           string   `json:"map"`
	Region        string   `json:"region"`
	Status        string   `json:"status"`
	Online        int      `json:"online"`
	Slots         int      `json:"slots"`
	Ping          int      `json:"ping"`
	CPU           int      `json:"cpu"`
	RAM           int      `json:"ram"`
	RAMLimitMB    int      `json:"ram_limit_mb"`
	TPS           float64  `json:"tps"`
	Uptime        float64  `json:"uptime"`
	IP            string   `json:"ip"`
	Spark         []int    `json:"spark"`
	PublicEnabled bool     `json:"public_enabled"`
	Description   string   `json:"description"`
	Tags          []string `json:"tags"`
	Discord       string   `json:"discord"`
	Website       string   `json:"website"`
	Votes         int      `json:"votes"`
}

func (h *Handler) MonitoringIndex(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ctx := r.Context()

	rows, err := h.loadMonitoringRows(ctx, monitoringScope{
		TenantID: claims.TenantID,
		UserID:   claims.UserID,
		Role:     claims.Role,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}

	online, players, slots, uptimeSum := 0, 0, 0, 0.0
	for _, row := range rows {
		if row.Status == "running" {
			online++
			players += row.Online
		}
		slots += row.Slots
		uptimeSum += row.Uptime
	}
	avgUptime := 0.0
	if len(rows) > 0 {
		avgUptime = round2(uptimeSum / float64(len(rows)))
	}

	incidents7d := 0
	_ = h.readerOf(ctx).QueryRow(ctx, `
		SELECT COUNT(*)
		FROM core.server_incidents i
		JOIN core.servers s ON s.id = i.server_id
		WHERE i.tenant_id = $1
		  AND i.started_at >= now() - INTERVAL '7 days'
		  AND ($2 OR s.user_id = $3)
	`, claims.TenantID, isStaffRole(claims.Role), claims.UserID).Scan(&incidents7d)

	writeJSON(w, http.StatusOK, map[string]any{
		"servers":      rows,
		"total":        len(rows),
		"online":       online,
		"players":      players,
		"slots":        slots,
		"avg_uptime":   avgUptime,
		"incidents_7d": incidents7d,
		"updated_at":   time.Now().UTC().Format(time.RFC3339),
	})
}

type monitoringScope struct {
	TenantID   string
	UserID     string
	Role       string
	ServerID   string
	PublicOnly bool
	AnyOwner   bool
	Limit      int
}

func (h *Handler) loadMonitoringRows(ctx context.Context, scope monitoringScope) ([]monitoringRow, error) {
	q := `
		SELECT s.id::text, s.name, s.game_id, COALESCE(g.name, ''),
		       COALESCE(gv.version, ''), COALESCE(s.status, 'stopped'), COALESCE(s.runtime_status, ''),
		       COALESCE(s.ip_address, ''), COALESCE(s.primary_port, 0),
		       COALESCE(s.limits, '{}'::jsonb), COALESCE(s.config, '{}'::jsonb),
		       COALESCE(n.city, ''), COALESCE(n.country, ''), COALESCE(n.name, ''),
		       COALESCE(m.public_enabled, false), COALESCE(m.description, ''),
		       COALESCE(m.tags, '{}'::text[]), COALESCE(m.discord_url, ''),
		       COALESCE(m.website_url, ''), COALESCE(m.votes, 0)
		FROM core.servers s
		LEFT JOIN core.games g ON g.tenant_id = s.tenant_id AND g.slug = s.game_id
		LEFT JOIN core.game_versions gv ON gv.id = s.game_version_id
		LEFT JOIN core.nodes n ON n.id = s.node_id
		LEFT JOIN core.server_monitoring m ON m.server_id = s.id
		WHERE s.tenant_id = $1`
	args := []any{scope.TenantID}
	if !scope.AnyOwner && !isStaffRole(scope.Role) {
		args = append(args, scope.UserID)
		q += ` AND (s.user_id = $` + strconv.Itoa(len(args)) + ` OR s.user_id IS NULL)`
	}
	if scope.ServerID != "" {
		args = append(args, scope.ServerID)
		q += ` AND s.id = $` + strconv.Itoa(len(args))
	}
	if scope.PublicOnly {
		q += ` AND COALESCE(m.public_enabled, false) = true`
	}
	q += ` ORDER BY s.name`
	if scope.Limit > 0 {
		q += ` LIMIT ` + strconv.Itoa(scope.Limit)
	}

	rows, err := h.readerOf(ctx).Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]monitoringRow, 0)
	ids := make([]string, 0)
	for rows.Next() {
		var (
			row                     monitoringRow
			status, runtime         string
			ip                      string
			port                    int
			limitsRaw, configRaw    []byte
			city, country, nodeName string
			tags                    []string
		)
		if err := rows.Scan(
			&row.ID, &row.Name, &row.GameID, &row.GameName,
			&row.Version, &status, &runtime,
			&ip, &port, &limitsRaw, &configRaw,
			&city, &country, &nodeName,
			&row.PublicEnabled, &row.Description,
			&tags, &row.Discord, &row.Website, &row.Votes,
		); err != nil {
			continue
		}

		if live, ok := h.cache.GetServerStatus(ctx, row.ID); ok {
			row.Status = resolveEffectiveStatus(status, runtime, &live)
		} else {
			row.Status = resolveEffectiveStatus(status, runtime, nil)
		}

		limits := map[string]any{}
		_ = json.Unmarshal(limitsRaw, &limits)
		config := map[string]any{}
		_ = json.Unmarshal(configRaw, &config)

		row.Slots = limitsInt(limits, "slots")
		row.RAMLimitMB = limitsInt(limits, "ram_mb")
		if row.RAMLimitMB == 0 {
			row.RAMLimitMB = limitsInt(limits, "memory_mb")
		}
		row.Map = firstNonEmpty(toString(config["map"]), toString(config["level_name"]), toString(config["world"]))
		if row.Version == "" {
			row.Version = toString(config["version"])
		}
		row.Region = monitoringRegion(city, country, nodeName)
		row.IP = monitoringAddress(ip, port)
		if tags == nil {
			tags = []string{}
		}
		row.Tags = tags
		row.Spark = make([]int, monitoringSparkPoints)

		out = append(out, row)
		ids = append(ids, row.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return out, nil
	}

	live := h.latestOnlineSamples(ctx, ids)
	sparks := h.onlineSparklines(ctx, ids)
	uptimes := h.uptime30d(ctx, ids)

	for i := range out {
		row := &out[i]
		if s, ok := live[row.ID]; ok {
			row.Online = s.Online
			row.Ping = s.Ping
			row.TPS = s.TPS
			if s.MaxPlayers > 0 {
				row.Slots = s.MaxPlayers
			}
		}
		if row.Status != "running" {
			row.Online, row.Ping, row.TPS = 0, 0, 0
		} else if cpu, ram, ok := h.latestResourceUsage(ctx, row.ID); ok {
			row.CPU, row.RAM = cpu, ram
		}
		if spark := sparks[row.ID]; spark != nil {
			row.Spark = spark
		}
		row.Uptime = uptimes[row.ID]
	}
	return out, nil
}

type onlineSample struct {
	Online     int
	MaxPlayers int
	Ping       int
	TPS        float64
	Up         bool
	TS         time.Time
}

func (h *Handler) latestOnlineSamples(ctx context.Context, ids []string) map[string]onlineSample {
	out := map[string]onlineSample{}
	rows, err := h.readerOf(ctx).Query(ctx, `
		SELECT DISTINCT ON (server_id)
		       server_id::text, online, max_players, ping_ms, tps, up, ts
		FROM core.server_online_points
		WHERE server_id = ANY($1::uuid[]) AND ts >= now() - INTERVAL '10 minutes'
		ORDER BY server_id, ts DESC
	`, ids)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var s onlineSample
		if rows.Scan(&id, &s.Online, &s.MaxPlayers, &s.Ping, &s.TPS, &s.Up, &s.TS) == nil {
			out[id] = s
		}
	}
	return out
}

func (h *Handler) onlineSparklines(ctx context.Context, ids []string) map[string][]int {
	out := map[string][]int{}
	rows, err := h.readerOf(ctx).Query(ctx, `
		SELECT server_id::text,
		       EXTRACT(EPOCH FROM date_trunc('hour', ts))::bigint AS bucket,
		       MAX(online)
		FROM core.server_online_points
		WHERE server_id = ANY($1::uuid[]) AND ts >= now() - INTERVAL '24 hours'
		GROUP BY 1, 2
	`, ids)
	if err != nil {
		return out
	}
	defer rows.Close()

	start := time.Now().UTC().Truncate(time.Hour).
		Add(-time.Duration(monitoringSparkPoints-1) * time.Hour).Unix()
	for rows.Next() {
		var id string
		var bucket int64
		var online int
		if rows.Scan(&id, &bucket, &online) != nil {
			continue
		}
		series, ok := out[id]
		if !ok {
			series = make([]int, monitoringSparkPoints)
			out[id] = series
		}
		idx := int((bucket - start) / 3600)
		if idx >= 0 && idx < monitoringSparkPoints {
			series[idx] = online
		}
	}
	return out
}

func (h *Handler) uptime30d(ctx context.Context, ids []string) map[string]float64 {
	out := map[string]float64{}
	rows, err := h.readerOf(ctx).Query(ctx, `
		SELECT server_id::text,
		       COUNT(*) FILTER (WHERE up)::float8 / NULLIF(COUNT(*), 0) * 100
		FROM core.server_online_points
		WHERE server_id = ANY($1::uuid[]) AND ts >= now() - INTERVAL '30 days'
		GROUP BY 1
	`, ids)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var pct *float64
		if rows.Scan(&id, &pct) == nil && pct != nil {
			out[id] = round2(*pct)
		}
	}
	return out
}

func (h *Handler) latestResourceUsage(ctx context.Context, serverID string) (cpu, ram int, ok bool) {
	var cpuPct float64
	var used, limit int
	err := h.readerOf(ctx).QueryRow(ctx, `
		SELECT cpu_pct, mem_used_mb, mem_limit_mb
		FROM core.server_metric_points
		WHERE server_id = $1 AND ts >= now() - INTERVAL '10 minutes'
		ORDER BY ts DESC LIMIT 1
	`, serverID).Scan(&cpuPct, &used, &limit)
	if err != nil {
		return 0, 0, false
	}
	ramPct := 0
	if limit > 0 {
		ramPct = int(math.Round(float64(used) / float64(limit) * 100))
	}
	return int(math.Round(cpuPct)), ramPct, true
}

func (h *Handler) MonitoringShow(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ctx := r.Context()
	serverID := chi.URLParam(r, "id")

	if _, err := h.resolveServerAccess(ctx, claims.TenantID, claims.UserID, claims.Role, serverID); err != nil {
		writeError(w, http.StatusNotFound, "server not found")
		return
	}

	rows, err := h.loadMonitoringRows(ctx, monitoringScope{
		TenantID: claims.TenantID, ServerID: serverID, AnyOwner: true,
	})
	if err != nil || len(rows) == 0 {
		writeError(w, http.StatusNotFound, "server not found")
		return
	}
	row := rows[0]

	players := []map[string]any{}
	if row.Status == "running" {
		if res, ok := h.monitoringGameQuery(ctx, claims.TenantID, serverID, row); ok {
			players = monitoringPlayers(res["players_online"])
			if m := toString(res["current_map"]); m != "" {
				row.Map = m
			}
			if mx := intFromAny(res["max_players"]); mx > 0 {
				row.Slots = mx
			}
			if n := intFromAny(res["online_players"]); n > 0 {
				row.Online = n
			}
		}
	}

	peak, avg := h.onlinePeakAvg(ctx, serverID, 24)

	writeJSON(w, http.StatusOK, map[string]any{
		"server":      row,
		"players":     players,
		"peak_24h":    peak,
		"avg_24h":     avg,
		"uptime_30d":  row.Uptime,
		"uptime_days": h.uptimeDays(ctx, serverID, 30),
		"incidents":   h.loadIncidents(ctx, serverID, 20),
		"settings":    h.loadMonitoringSettings(ctx, claims.TenantID, serverID),
		"visits":      h.visitSeries(ctx, serverID, 7),
		"public_url":  h.monitoringPublicURL(serverID),
		"banner_url":  h.monitoringBannerBase(serverID),
	})
}

func (h *Handler) monitoringGameQuery(
	ctx context.Context, tenantID, serverID string, row monitoringRow,
) (map[string]any, bool) {
	nodeID, err := h.serverNodeID(ctx, tenantID, serverID)
	if err != nil || nodeID == "" {
		return nil, false
	}
	res, err := h.agentCommand(ctx, nodeID, serverID, "game_query", map[string]any{
		"game_id": row.GameID,
		"limits":  map[string]any{"slots": row.Slots},
		"port":    monitoringPortOf(row.IP),
	})
	if err != nil || res == nil {
		return nil, false
	}
	return res, true
}

func (h *Handler) onlinePeakAvg(ctx context.Context, serverID string, hours int) (peak, avg int) {
	var p, a *float64
	_ = h.readerOf(ctx).QueryRow(ctx, `
		SELECT MAX(online)::float8, AVG(online)::float8
		FROM core.server_online_points
		WHERE server_id = $1 AND ts >= now() - ($2::text || ' hours')::interval
	`, serverID, hours).Scan(&p, &a)
	if p != nil {
		peak = int(math.Round(*p))
	}
	if a != nil {
		avg = int(math.Round(*a))
	}
	return
}

type uptimeDay struct {
	Day     string  `json:"day"`
	Uptime  float64 `json:"uptime"`
	HasData bool    `json:"has_data"`
}

func (h *Handler) uptimeDays(ctx context.Context, serverID string, days int) []uptimeDay {
	byDay := map[string]float64{}
	rows, err := h.readerOf(ctx).Query(ctx, `
		SELECT to_char(date_trunc('day', ts), 'YYYY-MM-DD'),
		       COUNT(*) FILTER (WHERE up)::float8 / NULLIF(COUNT(*), 0) * 100
		FROM core.server_online_points
		WHERE server_id = $1 AND ts >= now() - ($2::text || ' days')::interval
		GROUP BY 1
	`, serverID, days)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var day string
			var pct *float64
			if rows.Scan(&day, &pct) == nil && pct != nil {
				byDay[day] = round2(*pct)
			}
		}
	}

	out := make([]uptimeDay, 0, days)
	today := time.Now().UTC().Truncate(24 * time.Hour)
	for i := days - 1; i >= 0; i-- {
		key := today.AddDate(0, 0, -i).Format("2006-01-02")
		pct, has := byDay[key]
		out = append(out, uptimeDay{Day: key, Uptime: pct, HasData: has})
	}
	return out
}

type incidentItem struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	Level       string `json:"level"`
	Body        string `json:"body"`
	StartedAt   string `json:"started_at"`
	DurationSec int    `json:"duration_sec"`
	Resolved    bool   `json:"resolved"`
}

func (h *Handler) loadIncidents(ctx context.Context, serverID string, limit int) []incidentItem {
	out := []incidentItem{}
	rows, err := h.readerOf(ctx).Query(ctx, `
		SELECT id, title, level, body, started_at, resolved_at
		FROM core.server_incidents
		WHERE server_id = $1
		ORDER BY started_at DESC
		LIMIT $2
	`, serverID, limit)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var it incidentItem
		var started time.Time
		var resolved *time.Time
		if rows.Scan(&it.ID, &it.Title, &it.Level, &it.Body, &started, &resolved) != nil {
			continue
		}
		it.StartedAt = started.UTC().Format(time.RFC3339)
		if resolved != nil {
			it.Resolved = true
			it.DurationSec = int(resolved.Sub(started).Seconds())
		} else {
			it.DurationSec = int(time.Since(started).Seconds())
		}
		out = append(out, it)
	}
	return out
}

func (h *Handler) MonitoringIncidents(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ctx := r.Context()
	serverID := chi.URLParam(r, "id")
	if _, err := h.resolveServerAccess(ctx, claims.TenantID, claims.UserID, claims.Role, serverID); err != nil {
		writeError(w, http.StatusNotFound, "server not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"items":       h.loadIncidents(ctx, serverID, 50),
		"uptime_days": h.uptimeDays(ctx, serverID, 30),
	})
}

type monitoringSettings struct {
	PublicEnabled bool     `json:"public_enabled"`
	Title         string   `json:"title"`
	Description   string   `json:"description"`
	Tags          []string `json:"tags"`
	Discord       string   `json:"discord"`
	Website       string   `json:"website"`
	ShowPlayers   bool     `json:"show_players"`
	ShowChart     bool     `json:"show_chart"`
	ShowIncidents bool     `json:"show_incidents"`
	ShowAddress   bool     `json:"show_address"`
	ShowVersion   bool     `json:"show_version"`
	Votes         int      `json:"votes"`
}

func defaultMonitoringSettings() monitoringSettings {
	return monitoringSettings{
		Tags:        []string{},
		ShowPlayers: true, ShowChart: true, ShowIncidents: true, ShowAddress: true,
	}
}

func (h *Handler) loadMonitoringSettings(ctx context.Context, tenantID, serverID string) monitoringSettings {
	s := defaultMonitoringSettings()
	var tags []string
	err := h.readerOf(ctx).QueryRow(ctx, `
		SELECT public_enabled, title, description, tags, discord_url, website_url,
		       show_players, show_chart, show_incidents, show_address, show_version, votes
		FROM core.server_monitoring
		WHERE server_id = $1 AND tenant_id = $2
	`, serverID, tenantID).Scan(
		&s.PublicEnabled, &s.Title, &s.Description, &tags, &s.Discord, &s.Website,
		&s.ShowPlayers, &s.ShowChart, &s.ShowIncidents, &s.ShowAddress, &s.ShowVersion, &s.Votes,
	)
	if err != nil {
		return defaultMonitoringSettings()
	}
	if tags == nil {
		tags = []string{}
	}
	s.Tags = tags
	return s
}

func (h *Handler) MonitoringSettingsShow(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ctx := r.Context()
	serverID := chi.URLParam(r, "id")
	if _, err := h.resolveServerAccess(ctx, claims.TenantID, claims.UserID, claims.Role, serverID); err != nil {
		writeError(w, http.StatusNotFound, "server not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"settings":   h.loadMonitoringSettings(ctx, claims.TenantID, serverID),
		"public_url": h.monitoringPublicURL(serverID),
		"banner_url": h.monitoringBannerBase(serverID),
	})
}

func (h *Handler) MonitoringSettingsUpdate(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ctx := r.Context()
	serverID := chi.URLParam(r, "id")

	access, err := h.resolveServerAccess(ctx, claims.TenantID, claims.UserID, claims.Role, serverID)
	if err != nil {
		writeError(w, http.StatusNotFound, "server not found")
		return
	}
	if !access.IsOwner && !access.IsStaff {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	current := h.loadMonitoringSettings(ctx, claims.TenantID, serverID)
	var body struct {
		PublicEnabled *bool     `json:"public_enabled"`
		Title         *string   `json:"title"`
		Description   *string   `json:"description"`
		Tags          *[]string `json:"tags"`
		Discord       *string   `json:"discord"`
		Website       *string   `json:"website"`
		ShowPlayers   *bool     `json:"show_players"`
		ShowChart     *bool     `json:"show_chart"`
		ShowIncidents *bool     `json:"show_incidents"`
		ShowAddress   *bool     `json:"show_address"`
		ShowVersion   *bool     `json:"show_version"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	if body.PublicEnabled != nil {
		current.PublicEnabled = *body.PublicEnabled
	}
	if body.Title != nil {
		current.Title = clampText(*body.Title, 120)
	}
	if body.Description != nil {
		current.Description = clampText(*body.Description, 1000)
	}
	if body.Tags != nil {
		current.Tags = normalizeTags(*body.Tags)
	}
	if body.Discord != nil {
		current.Discord = clampText(*body.Discord, 200)
	}
	if body.Website != nil {
		current.Website = clampText(*body.Website, 200)
	}
	if body.ShowPlayers != nil {
		current.ShowPlayers = *body.ShowPlayers
	}
	if body.ShowChart != nil {
		current.ShowChart = *body.ShowChart
	}
	if body.ShowIncidents != nil {
		current.ShowIncidents = *body.ShowIncidents
	}
	if body.ShowAddress != nil {
		current.ShowAddress = *body.ShowAddress
	}
	if body.ShowVersion != nil {
		current.ShowVersion = *body.ShowVersion
	}

	_, err = h.dbOf(ctx).Exec(ctx, `
		INSERT INTO core.server_monitoring
		    (server_id, tenant_id, public_enabled, title, description, tags,
		     discord_url, website_url, show_players, show_chart, show_incidents,
		     show_address, show_version, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, now())
		ON CONFLICT (server_id) DO UPDATE SET
		    public_enabled = EXCLUDED.public_enabled,
		    title          = EXCLUDED.title,
		    description    = EXCLUDED.description,
		    tags           = EXCLUDED.tags,
		    discord_url    = EXCLUDED.discord_url,
		    website_url    = EXCLUDED.website_url,
		    show_players   = EXCLUDED.show_players,
		    show_chart     = EXCLUDED.show_chart,
		    show_incidents = EXCLUDED.show_incidents,
		    show_address   = EXCLUDED.show_address,
		    show_version   = EXCLUDED.show_version,
		    updated_at     = now()
	`, serverID, claims.TenantID, current.PublicEnabled, current.Title, current.Description,
		current.Tags, current.Discord, current.Website, current.ShowPlayers, current.ShowChart,
		current.ShowIncidents, current.ShowAddress, current.ShowVersion)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":       true,
		"settings": h.loadMonitoringSettings(ctx, claims.TenantID, serverID),
	})
}

type topItem struct {
	Rank     int      `json:"rank"`
	ServerID string   `json:"server_id"`
	Name     string   `json:"name"`
	Tagline  string   `json:"tagline"`
	GameID   string   `json:"game_id"`
	GameName string   `json:"game_name"`
	Online   int      `json:"online"`
	Slots    int      `json:"slots"`
	Uptime   float64  `json:"uptime"`
	Votes    int      `json:"votes"`
	IP       string   `json:"ip"`
	Spark    []int    `json:"spark"`
	Tags     []string `json:"tags"`
}

func (h *Handler) MonitoringTop(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID, ok := h.monitoringTenant(w, r)
	if !ok {
		return
	}

	rows, err := h.loadMonitoringRows(ctx, monitoringScope{
		TenantID: tenantID, AnyOwner: true, PublicOnly: true, Limit: 200,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}

	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].Votes != rows[j].Votes {
			return rows[i].Votes > rows[j].Votes
		}
		if rows[i].Online != rows[j].Online {
			return rows[i].Online > rows[j].Online
		}
		return rows[i].Uptime > rows[j].Uptime
	})

	gameFilter := strings.TrimSpace(r.URL.Query().Get("game"))
	games := []string{}
	seen := map[string]bool{}
	items := make([]topItem, 0, len(rows))
	rank := 0
	for _, row := range rows {
		gameLabel := firstNonEmpty(row.GameName, row.GameID)
		if !seen[gameLabel] {
			seen[gameLabel] = true
			games = append(games, gameLabel)
		}
		if gameFilter != "" && gameFilter != gameLabel {
			continue
		}
		rank++
		items = append(items, topItem{
			Rank: rank, ServerID: row.ID, Name: row.Name,
			Tagline: monitoringTagline(row), GameID: row.GameID, GameName: gameLabel,
			Online: row.Online, Slots: row.Slots, Uptime: row.Uptime, Votes: row.Votes,
			IP: row.IP, Spark: row.Spark, Tags: row.Tags,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": items, "games": games})
}

func monitoringTagline(row monitoringRow) string {
	if len(row.Tags) > 0 {
		return strings.Join(row.Tags, " · ")
	}
	return firstNonEmpty(row.GameName, row.GameID)
}

func (h *Handler) monitoringTenant(w http.ResponseWriter, r *http.Request) (string, bool) {
	if claims, ok := tenantClaims(r.Context()); ok {
		return claims.TenantID, true
	}
	if tenantID, ok := h.resolvePublicTenantID(r.Context(), r); ok {
		return tenantID, true
	}
	writeError(w, http.StatusBadRequest, "tenant not resolved")
	return "", false
}

func (h *Handler) MonitoringPublic(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	serverID := chi.URLParam(r, "id")

	tenantID, ok := h.resolveMonitoringPublicTenant(w, ctx, r, serverID)
	if !ok {
		return
	}

	rows, err := h.loadMonitoringRows(ctx, monitoringScope{
		TenantID: tenantID, AnyOwner: true, ServerID: serverID,
	})
	if err != nil || len(rows) == 0 {
		writeError(w, http.StatusNotFound, "server not found")
		return
	}
	row := rows[0]

	settings := h.loadMonitoringSettings(ctx, tenantID, serverID)
	if !settings.PublicEnabled {
		writeError(w, http.StatusNotFound, "public page disabled")
		return
	}

	h.recordMonitoringVisit(ctx, serverID)

	if !settings.ShowAddress {
		row.IP = ""
	}
	if !settings.ShowVersion {
		row.Version = ""
		row.Map = ""
	}
	players := []map[string]any{}
	if settings.ShowPlayers && row.Status == "running" {
		if res, ok := h.monitoringGameQuery(ctx, tenantID, serverID, row); ok {
			players = monitoringPlayers(res["players_online"])
		}
	}
	incidents := []incidentItem{}
	days := []uptimeDay{}
	if settings.ShowIncidents {
		incidents = h.loadIncidents(ctx, serverID, 10)
		days = h.uptimeDays(ctx, serverID, 30)
	}

	peak, avg := h.onlinePeakAvg(ctx, serverID, 24)
	if settings.Title != "" {
		row.Name = settings.Title
	}
	row.Description = settings.Description
	row.Tags = settings.Tags
	row.Discord = settings.Discord
	row.Website = settings.Website

	writeJSON(w, http.StatusOK, map[string]any{
		"server":      row,
		"settings":    settings,
		"players":     players,
		"peak_24h":    peak,
		"avg_24h":     avg,
		"incidents":   incidents,
		"uptime_days": days,
		"show_chart":  settings.ShowChart,
		"banner_url":  h.monitoringBannerBase(serverID),
		"public_url":  h.monitoringPublicURL(serverID),
	})
}

func (h *Handler) MonitoringPublicVote(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	serverID := chi.URLParam(r, "id")
	tenantID, ok := h.resolveMonitoringPublicTenant(w, ctx, r, serverID)
	if !ok {
		return
	}

	var enabled bool
	if h.readerOf(ctx).QueryRow(ctx, `
		SELECT public_enabled FROM core.server_monitoring WHERE server_id = $1 AND tenant_id = $2
	`, serverID, tenantID).Scan(&enabled) != nil || !enabled {
		writeError(w, http.StatusNotFound, "public page disabled")
		return
	}

	claimed, err := h.cache.Claim(ctx, "monitoring:vote:"+serverID+":"+clientIP(r), 24*time.Hour)
	if err == nil && !claimed {
		writeError(w, http.StatusTooManyRequests, "already voted today")
		return
	}

	var votes int
	err = h.dbOf(ctx).QueryRow(ctx, `
		UPDATE core.server_monitoring SET votes = votes + 1
		WHERE server_id = $1 AND tenant_id = $2
		RETURNING votes
	`, serverID, tenantID).Scan(&votes)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "votes": votes})
}

func (h *Handler) recordMonitoringVisit(ctx context.Context, serverID string) {
	_, _ = h.dbOf(ctx).Exec(ctx, `
		INSERT INTO core.server_monitoring_visits (server_id, day, visits)
		VALUES ($1, CURRENT_DATE, 1)
		ON CONFLICT (server_id, day) DO UPDATE
		SET visits = core.server_monitoring_visits.visits + 1
	`, serverID)
}

type visitSeries struct {
	Total int   `json:"total"`
	Days  []int `json:"days"`
}

func (h *Handler) visitSeries(ctx context.Context, serverID string, days int) visitSeries {
	byDay := map[string]int{}
	rows, err := h.readerOf(ctx).Query(ctx, `
		SELECT to_char(day, 'YYYY-MM-DD'), visits
		FROM core.server_monitoring_visits
		WHERE server_id = $1 AND day >= CURRENT_DATE - ($2::int - 1)
	`, serverID, days)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var day string
			var n int
			if rows.Scan(&day, &n) == nil {
				byDay[day] = n
			}
		}
	}
	out := visitSeries{Days: make([]int, 0, days)}
	today := time.Now().UTC().Truncate(24 * time.Hour)
	for i := days - 1; i >= 0; i-- {
		n := byDay[today.AddDate(0, 0, -i).Format("2006-01-02")]
		out.Days = append(out.Days, n)
		out.Total += n
	}
	return out
}

func (h *Handler) monitoringPublicURL(serverID string) string {
	return strings.TrimRight(h.frontendURL, "/") + "/monitoring/public/" + serverID
}

func (h *Handler) monitoringBannerBase(serverID string) string {
	return strings.TrimRight(h.apiPublicURL, "/") + "/v1/monitoring/public/" + serverID + "/banner"
}

func monitoringRegion(city, country, nodeName string) string {
	parts := []string{}
	if city != "" {
		parts = append(parts, city)
	}
	if country != "" {
		parts = append(parts, country)
	}
	if len(parts) == 0 {
		return nodeName
	}
	return strings.Join(parts, " · ")
}

func monitoringAddress(ip string, port int) string {
	if ip == "" {
		return ""
	}
	if port > 0 {
		return ip + ":" + strconv.Itoa(port)
	}
	return ip
}

func monitoringPortOf(address string) int {
	idx := strings.LastIndex(address, ":")
	if idx < 0 {
		return 0
	}
	n, err := strconv.Atoi(address[idx+1:])
	if err != nil {
		return 0
	}
	return n
}

func monitoringPlayers(raw any) []map[string]any {
	out := []map[string]any{}
	arr, ok := raw.([]any)
	if !ok {
		return out
	}
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		name := toString(m["name"])
		if name == "" {
			continue
		}
		out = append(out, map[string]any{
			"name":         name,
			"score":        intFromAny(m["score"]),
			"ping":         intFromAny(m["ping"]),
			"duration_sec": intFromAny(m["duration_sec"]),
		})
	}
	return out
}

func normalizeTags(in []string) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, tag := range in {
		t := clampText(tag, 32)
		if t == "" || seen[strings.ToLower(t)] {
			continue
		}
		seen[strings.ToLower(t)] = true
		out = append(out, t)
		if len(out) >= 12 {
			break
		}
	}
	return out
}

func clampText(s string, max int) string {
	s = strings.TrimSpace(s)
	runes := []rune(s)
	if len(runes) > max {
		return string(runes[:max])
	}
	return s
}

func round2(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	return math.Round(v*100) / 100
}
