package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/vortanixapp/panel/internal/api/jobwake"
	"github.com/vortanixapp/panel/pkg/buildinfo"
	"github.com/vortanixapp/panel/pkg/nodeevents"
	"github.com/vortanixapp/panel/pkg/protocol"
	"github.com/vortanixapp/panel/pkg/sshclient"
	"github.com/vortanixapp/panel/pkg/updates"
)

const (
	agentDiskLowRatio  = 0.1
	sshExecTimeout     = 60 * time.Second
	sshExecOutputLimit = 256 << 10
)

type agentRow struct {
	ID, Name, Code, Country, Region string
	Host                            *string
	DaemonStatus, Version, Platform string
	RemoteAddr, DisconnectReason    string
	Proto                           int
	RTT                             *int
	LastSeen, StartedAt             *time.Time
	ConnectedAt, DisconnectedAt     *time.Time
	AutoUpdate, SSH, Maintenance    bool
	UpdateRaw, StatsRaw             []byte
	StatsAt                         *time.Time
	ServersTotal, ServersRunning    int
	Restarting                      bool
}

const agentRowQuery = `
	SELECT n.id::text, n.name, COALESCE(n.meta->>'code', n.fqdn, ''), COALESCE(n.country, ''),
		COALESCE(n.region, ''), n.ssh_host,
		COALESCE(d.status, 'unknown'), COALESCE(d.version, ''), COALESCE(d.platform, ''),
		COALESCE(d.remote_addr, ''), COALESCE(d.disconnect_reason, ''), COALESCE(d.proto, 1), d.rtt_ms,
		d.last_seen_at, d.started_at, d.connected_at, d.disconnected_at,
		n.agent_auto_update,
		(COALESCE(n.ssh_host, '') <> '' AND COALESCE(n.ssh_user, '') <> ''
			AND (COALESCE(n.ssh_password_enc, '') <> '' OR COALESCE(n.meta->>'ssh_password', '') <> '')),
		COALESCE(n.maintenance_mode, false),
		n.agent_update, d.stats, d.stats_at,
		(SELECT COUNT(*)::int FROM core.servers s WHERE s.node_id = n.id),
		(SELECT COUNT(*)::int FROM core.servers s WHERE s.node_id = n.id AND s.status IN ('running', 'starting')),
		EXISTS (SELECT 1 FROM core.node_tasks t WHERE t.node_id = n.id AND t.action = 'agent_restart'
			AND t.status IN ('queued', 'sent', 'running') AND t.deadline_at > now())
	FROM core.nodes n
	LEFT JOIN core.node_daemons d ON d.node_id = n.id`

func scanAgentRow(row pgx.Row) (agentRow, error) {
	var a agentRow
	err := row.Scan(&a.ID, &a.Name, &a.Code, &a.Country, &a.Region, &a.Host,
		&a.DaemonStatus, &a.Version, &a.Platform, &a.RemoteAddr, &a.DisconnectReason, &a.Proto, &a.RTT,
		&a.LastSeen, &a.StartedAt, &a.ConnectedAt, &a.DisconnectedAt,
		&a.AutoUpdate, &a.SSH, &a.Maintenance, &a.UpdateRaw, &a.StatsRaw, &a.StatsAt,
		&a.ServersTotal, &a.ServersRunning, &a.Restarting)
	a.Version = buildinfo.Normalize(a.Version)
	return a, err
}

func statsNumber(raw []byte, section, key string) (float64, bool) {
	var stats map[string]map[string]any
	if json.Unmarshal(raw, &stats) != nil {
		return 0, false
	}
	v, ok := stats[section][key].(float64)
	return v, ok
}

func (a agentRow) view(target string, res *nodeResources) map[string]any {
	online := agentDaemonOnline(a.DaemonStatus, a.LastSeen)
	state := agentPresenceState(a.DaemonStatus, a.LastSeen)
	update := agentUpdateView(a.UpdateRaw)
	outdated := a.Version != "" && updates.AgentOutdated(a.Version, target)
	labels := []string{}
	updateStatus := ""
	if update != nil {
		updateStatus, _ = update["status"].(string)
	}
	switch updateStatus {
	case "pending", "pulling", "restarting":
		labels = append(labels, "updating")
	case "failed":
		labels = append(labels, "update_failed")
	}
	if a.Restarting {
		labels = append(labels, "restarting")
	}
	if outdated {
		labels = append(labels, "outdated")
	}
	if online && a.Proto < protocol.ProtoVersion {
		labels = append(labels, "legacy")
	}
	resources := map[string]any{}
	if res != nil && res.known() {
		resources["source"] = res.Source
		if res.MeasuredAt != nil {
			resources["measured_at"] = res.MeasuredAt.UTC().Format(time.RFC3339)
		}
		resources["cpu_percent"] = res.CPUPercent
		resources["ram_percent"] = res.RAMPercent
		resources["disk_percent"] = res.DiskPercent
		resources["ram_total_mb"] = res.RAMTotalMB
		resources["ram_used_mb"] = res.RAMUsedMB
		resources["disk_total_mb"] = res.DiskTotalMB
		resources["disk_free_mb"] = res.DiskFreeMB
		if res.DiskTotalMB > 0 && res.DiskFreeMB < res.DiskTotalMB*agentDiskLowRatio {
			labels = append(labels, "disk_low")
		}
	}
	item := map[string]any{
		"id": a.ID, "node_id": a.ID, "location_id": a.ID,
		"name": a.Name, "code": a.Code, "country": a.Country, "region": a.Region,
		"host": strPtr(a.Host), "status": a.DaemonStatus, "is_online": online, "state": state,
		"labels": labels, "version": a.Version, "outdated": outdated, "proto": a.Proto,
		"platform": a.Platform, "auto_update": a.AutoUpdate, "ssh": a.SSH, "maintenance": a.Maintenance,
		"resources": resources,
		"servers":   map[string]any{"total": a.ServersTotal, "running": a.ServersRunning},
		"location":  map[string]any{"id": a.ID, "name": a.Name, "code": a.Code, "region": a.Region},
	}
	if update != nil {
		item["update"] = update
	}
	if secs, ok := agentUptimeSeconds(online, a.StartedAt, a.ConnectedAt); ok {
		item["uptime_sec"] = secs
	}
	if v, ok := statsNumber(a.StatsRaw, "host", "uptime_sec"); ok && online {
		item["host_uptime_sec"] = v
	}
	if a.LastSeen != nil {
		item["last_seen"] = a.LastSeen.UTC().Format(time.RFC3339)
		item["last_seen_human"] = formatLastSeenHuman(*a.LastSeen)
	}
	if a.ConnectedAt != nil {
		item["connected_at"] = a.ConnectedAt.UTC().Format(time.RFC3339)
	}
	if a.RemoteAddr != "" {
		item["remote_addr"] = a.RemoteAddr
	}
	if a.RTT != nil {
		item["rtt_ms"] = *a.RTT
	}
	return item
}

func hasLabel(item map[string]any, label string) bool {
	labels, _ := item["labels"].([]string)
	for _, l := range labels {
		if l == label {
			return true
		}
	}
	return false
}

func (h *Handler) ListAdminAgents(w http.ResponseWriter, r *http.Request) {
	if _, ok := tenantClaims(r.Context()); !ok {
		return
	}
	ctx := r.Context()
	rows, err := h.readerOf(ctx).Query(ctx, agentRowQuery+` ORDER BY COALESCE(n.sort_order, 0), n.name`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	var list []agentRow
	for rows.Next() {
		if a, err := scanAgentRow(rows); err == nil {
			list = append(list, a)
		}
	}
	rows.Close()
	ids := make([]string, len(list))
	for i, a := range list {
		ids[i] = a.ID
	}
	resources := h.loadNodeResources(ctx, ids)
	target := buildinfo.Current()
	items := []map[string]any{}
	summary := map[string]int{
		"total": len(list), "online": 0, "offline": 0, "never_connected": 0, "outdated": 0,
		"updating": 0, "update_failed": 0, "disk_low": 0, "legacy": 0, "restarting": 0,
	}
	for _, a := range list {
		item := a.view(target, resources[a.ID])
		summary[item["state"].(string)]++
		for _, l := range []string{"outdated", "updating", "update_failed", "disk_low", "legacy", "restarting"} {
			if hasLabel(item, l) {
				summary[l]++
			}
		}
		items = append(items, item)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"summary":        summary,
		"target_version": target,
		"auto_enabled":   h.tenantSettingString(ctx, updates.SettingAgentsAuto) == "1",
		"agents":         items,
		"daemons":        items,
	})
}

func (h *Handler) agentViewer(r *http.Request) map[string]any {
	claims, _ := tenantClaims(r.Context())
	_, isKey := apiKeyFromContext(r.Context())
	return map[string]any{
		"can_write":    h.viewerCan(r, "admin.daemons.write"),
		"can_power":    h.viewerCan(r, "admin.servers.write"),
		"can_ssh_exec": claims != nil && claims.Role == "owner" && !isKey,
	}
}

func (h *Handler) GetAdminAgentShow(w http.ResponseWriter, r *http.Request) {
	if _, ok := tenantClaims(r.Context()); !ok {
		return
	}
	ctx := r.Context()
	id := chi.URLParam(r, "id")
	a, err := scanAgentRow(h.readerOf(ctx).QueryRow(ctx, agentRowQuery+` WHERE n.id::text = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "location not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	var hostRaw []byte
	var caps []string
	var bootID string
	_ = h.readerOf(ctx).QueryRow(ctx, `
		SELECT COALESCE(host, '{}'::jsonb), COALESCE(caps, '{}'), COALESCE(boot_id, '')
		FROM core.node_daemons WHERE node_id::text = $1
	`, id).Scan(&hostRaw, &caps, &bootID)
	target := buildinfo.Current()
	item := a.view(target, h.loadNodeResources(ctx, []string{id})[id])
	item["target_version"] = target
	item["caps"] = caps
	item["boot_id"] = bootID
	if len(hostRaw) > 0 {
		item["facts"] = json.RawMessage(hostRaw)
	}
	if len(a.StatsRaw) > 0 {
		item["stats"] = json.RawMessage(a.StatsRaw)
	}
	connection := map[string]any{"rtt_ms": a.RTT, "remote_addr": a.RemoteAddr}
	if a.ConnectedAt != nil {
		connection["connected_at"] = a.ConnectedAt.UTC().Format(time.RFC3339)
	}
	if a.DisconnectedAt != nil {
		connection["disconnected_at"] = a.DisconnectedAt.UTC().Format(time.RFC3339)
		connection["disconnect_reason"] = a.DisconnectReason
	}
	item["connection"] = connection

	daemon := h.loadAgentInfo(r, id)
	cpuMetrics, ramMetrics := h.loadAgentChartMetrics(r, id)
	writeJSON(w, http.StatusOK, map[string]any{
		"agent_view": item,
		"viewer":     h.agentViewer(r),
		"location": map[string]any{
			"id": id, "name": a.Name, "code": a.Code, "ssh_host": strPtr(a.Host),
			"country": a.Country, "region": a.Region,
		},
		"agent":  daemon,
		"daemon": daemon,
		"metrics": map[string]any{
			"agent_cpu_usage": cpuMetrics, "agent_ram_usage": ramMetrics,
			"daemon_cpu_usage": cpuMetrics, "daemon_ram_usage": ramMetrics,
		},
	})
}

var metricRanges = map[string]struct {
	window, bucket time.Duration
}{
	"1h":  {time.Hour, 15 * time.Second},
	"24h": {24 * time.Hour, 6 * time.Minute},
	"7d":  {7 * 24 * time.Hour, 42 * time.Minute},
}

var agentMetricTypes = []string{"cpu_usage", "ram_usage", "disk_usage", "agent_cpu_usage", "agent_ram_usage", "agent_ram_mb"}

func (h *Handler) GetAdminAgentMetrics(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	key := r.URL.Query().Get("range")
	rg, ok := metricRanges[key]
	if !ok {
		key, rg = "24h", metricRanges["24h"]
	}
	ctx := r.Context()
	rows, err := h.readerOf(ctx).Query(ctx, `
		WITH buckets AS (
			SELECT generate_series(
				date_bin(make_interval(secs => $3), now() - make_interval(secs => $2), 'epoch'::timestamptz),
				date_bin(make_interval(secs => $3), now(), 'epoch'::timestamptz),
				make_interval(secs => $3)) AS at
		), agg AS (
			SELECT metric_type, date_bin(make_interval(secs => $3), measured_at, 'epoch'::timestamptz) AS at,
				AVG(value)::float8 AS avg, MAX(value)::float8 AS max
			FROM core.node_metrics
			WHERE node_id::text = $1 AND metric_type = ANY($4::text[])
			  AND measured_at > now() - make_interval(secs => $2)
			GROUP BY 1, 2
		)
		SELECT t.metric_type, b.at, a.avg, a.max
		FROM buckets b
		CROSS JOIN unnest($4::text[]) AS t(metric_type)
		LEFT JOIN agg a ON a.metric_type = t.metric_type AND a.at = b.at
		ORDER BY t.metric_type, b.at
	`, id, rg.window.Seconds(), rg.bucket.Seconds(), agentMetricTypes)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()
	series := map[string][]map[string]any{}
	for _, t := range agentMetricTypes {
		series[t] = []map[string]any{}
	}
	for rows.Next() {
		var metric string
		var at time.Time
		var avg, max *float64
		if rows.Scan(&metric, &at, &avg, &max) != nil {
			continue
		}
		series[metric] = append(series[metric], map[string]any{"t": at.UTC().Format(time.RFC3339), "v": avg, "max": max})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"range": key, "bucket_sec": int(rg.bucket.Seconds()), "series": series,
	})
}

var agentEventGroups = map[string][]string{
	"connection":  {string(nodeevents.Connect), string(nodeevents.Disconnect), string(nodeevents.AgentStarted)},
	"updates":     {string(nodeevents.VersionChanged), string(nodeevents.UpdateStarted), string(nodeevents.UpdateStage), string(nodeevents.UpdateDone), string(nodeevents.UpdateFailed)},
	"maintenance": {string(nodeevents.RestartRequested), string(nodeevents.ReinstallStarted), string(nodeevents.ReinstallDone), string(nodeevents.ReinstallFailed), string(nodeevents.Diagnostics), string(nodeevents.Cleanup), string(nodeevents.SSHExec)},
	"problems":    {string(nodeevents.TaskFailed), string(nodeevents.Alert), string(nodeevents.FirewallFailed), string(nodeevents.CronFailed), string(nodeevents.DockerUnreachable), string(nodeevents.DockerRecovered)},
}

func (h *Handler) GetAdminAgentEvents(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	before, _ := strconv.ParseInt(q.Get("before"), 10, 64)
	kinds := agentEventGroups[q.Get("group")]
	ctx := r.Context()
	rows, err := h.readerOf(ctx).Query(ctx, `
		SELECT e.id, e.kind, e.level, e.data, e.created_at, COALESCE(u.email, '')
		FROM core.node_events e
		LEFT JOIN core.users u ON u.id = e.actor_id
		WHERE e.node_id::text = $1
		  AND ($2::bigint = 0 OR e.id < $2)
		  AND (COALESCE(cardinality($3::text[]), 0) = 0 OR e.kind = ANY($3::text[]))
		ORDER BY e.id DESC
		LIMIT $4
	`, id, before, kinds, limit+1)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()
	events := []map[string]any{}
	for rows.Next() {
		var eid int64
		var kind, level, actor string
		var data []byte
		var at time.Time
		if rows.Scan(&eid, &kind, &level, &data, &at, &actor) != nil {
			continue
		}
		ev := map[string]any{"id": eid, "kind": kind, "level": level, "data": json.RawMessage(data), "created_at": at.UTC().Format(time.RFC3339)}
		if actor != "" {
			ev["actor"] = actor
		}
		events = append(events, ev)
	}
	resp := map[string]any{"events": events, "has_more": false}
	if len(events) > limit {
		events = events[:limit]
		resp["events"] = events
		resp["has_more"] = true
	}
	if len(events) > 0 {
		resp["next_before"] = events[len(events)-1]["id"]
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) startAgentUpdates(ctx context.Context, userID string, nodeIDs []string) ([]map[string]any, int) {
	target := buildinfo.Current()
	image := updates.AgentImage(target)
	results := []map[string]any{}
	started := 0
	for _, id := range nodeIDs {
		id = strings.TrimSpace(id)
		method, err := h.startAgentUpdate(ctx, id, target, image, "manual")
		item := map[string]any{"id": id, "ok": err == nil, "method": method}
		if err != nil {
			item["error"] = err.Error()
		} else {
			started++
			_ = nodeevents.Record(ctx, h.dbOf(ctx), id, nodeevents.UpdateStarted, nodeevents.Info,
				map[string]any{"target": target, "method": method}, userID)
		}
		results = append(results, item)
	}
	audit(ctx, h.dbOf(ctx), userID, "agent.update", "nodes", map[string]any{"nodes": nodeIDs, "target": target, "started": started})
	return results, started
}

func (h *Handler) PostAdminAgentUpdate(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	results, started := h.startAgentUpdates(r.Context(), claims.UserID, []string{chi.URLParam(r, "id")})
	if started == 0 {
		msg, _ := results[0]["error"].(string)
		writeError(w, http.StatusConflict, msg)
		return
	}
	writeJSON(w, http.StatusAccepted, results[0])
}

func (h *Handler) PostAdminAgentsUpdate(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	var body struct {
		NodeIDs  []string `json:"node_ids"`
		Outdated bool     `json:"outdated"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	ids := body.NodeIDs
	if body.Outdated {
		ids = h.outdatedAgentIDs(r.Context())
	}
	if len(ids) == 0 {
		writeError(w, http.StatusUnprocessableEntity, "Нет агентов для обновления")
		return
	}
	results, started := h.startAgentUpdates(r.Context(), claims.UserID, ids)
	writeJSON(w, http.StatusOK, map[string]any{"results": results, "started": started})
}

func (h *Handler) outdatedAgentIDs(ctx context.Context) []string {
	target := buildinfo.Current()
	rows, err := h.readerOf(ctx).Query(ctx, `
		SELECT n.id::text, COALESCE(d.version, ''), COALESCE(n.agent_update->>'status', '')
		FROM core.nodes n JOIN core.node_daemons d ON d.node_id = n.id
	`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id, version, status string
		if rows.Scan(&id, &version, &status) != nil {
			continue
		}
		version = buildinfo.Normalize(version)
		if version == "" || !updates.AgentOutdated(version, target) {
			continue
		}
		switch status {
		case "pending", "pulling", "restarting":
			continue
		}
		ids = append(ids, id)
	}
	return ids
}

func (h *Handler) setAgentsAutoUpdate(w http.ResponseWriter, r *http.Request, ids []string, value bool) {
	claims, _ := tenantClaims(r.Context())
	ctx := r.Context()
	var err error
	if ids == nil {
		_, err = h.dbOf(ctx).Exec(ctx, `UPDATE core.nodes SET agent_auto_update = $1`, value)
	} else {
		_, err = h.dbOf(ctx).Exec(ctx, `UPDATE core.nodes SET agent_auto_update = $1 WHERE id::text = ANY($2::text[])`, value, ids)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	audit(ctx, h.dbOf(ctx), claims.UserID, "agent.auto_update", "nodes", map[string]any{"nodes": ids, "auto_update": value})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) PatchAdminAgent(w http.ResponseWriter, r *http.Request) {
	var body struct {
		AutoUpdate *bool `json:"auto_update"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.AutoUpdate == nil {
		writeError(w, http.StatusUnprocessableEntity, "Укажите auto_update")
		return
	}
	h.setAgentsAutoUpdate(w, r, []string{chi.URLParam(r, "id")}, *body.AutoUpdate)
}

func (h *Handler) PatchAdminAgents(w http.ResponseWriter, r *http.Request) {
	var body struct {
		NodeIDs    []string `json:"node_ids"`
		AutoUpdate *bool    `json:"auto_update"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.AutoUpdate == nil || len(body.NodeIDs) == 0 {
		writeError(w, http.StatusUnprocessableEntity, "Укажите агентов и auto_update")
		return
	}
	h.setAgentsAutoUpdate(w, r, body.NodeIDs, *body.AutoUpdate)
}

func (h *Handler) PatchAdminAgentsSettings(w http.ResponseWriter, r *http.Request) {
	claims, _ := tenantClaims(r.Context())
	var body struct {
		AutoEnabled *bool `json:"auto_enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.AutoEnabled == nil {
		writeError(w, http.StatusUnprocessableEntity, "Укажите auto_enabled")
		return
	}
	value := "0"
	if *body.AutoEnabled {
		value = "1"
	}
	h.setTenantSettingString(r.Context(), updates.SettingAgentsAuto, value)
	audit(r.Context(), h.dbOf(r.Context()), claims.UserID, "agent.auto_update", "settings", map[string]any{"auto_enabled": *body.AutoEnabled})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "auto_enabled": *body.AutoEnabled})
}

func (h *Handler) startSSHTask(ctx context.Context, nodeID, action, userID string) (string, error) {
	payload, _ := json.Marshal(map[string]any{"node_id": nodeID, "action": action, "params": map[string]any{}})
	var jobID string
	if err := h.dbOf(ctx).QueryRow(ctx, `
		INSERT INTO core.jobs (type, status, payload) VALUES ('daemon_action', 'pending', $1::jsonb) RETURNING id::text
	`, payload).Scan(&jobID); err != nil {
		return "", err
	}
	taskID := uuid.NewString()
	if _, err := h.dbOf(ctx).Exec(ctx, `
		INSERT INTO core.node_tasks (id, node_id, action, method, status, params, job_id, requested_by, deadline_at)
		VALUES ($1::uuid, $2::uuid, $3, 'ssh', 'queued', '{}'::jsonb, $4::uuid, NULLIF($5, '')::uuid, now() + interval '40 minutes')
	`, taskID, nodeID, "ssh_"+action, jobID, userID); err != nil {
		return "", err
	}
	jobwake.Notify("daemon_action")
	return taskID, nil
}

func (h *Handler) sshTaskHTTP(w http.ResponseWriter, r *http.Request, action, auditAction string) {
	claims, id, loc, ok := h.adminLocationContext(w, r)
	if !ok {
		return
	}
	if !hasSSHConfigured(loc, parseMetaMap(loc.Meta)) {
		writeCodedError(w, http.StatusConflict, "ssh_not_configured", "SSH для ноды не настроен")
		return
	}
	taskID, err := h.startSSHTask(r.Context(), id, action, claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "задача не поставлена в очередь")
		return
	}
	if action == "install" {
		_ = nodeevents.Record(r.Context(), h.dbOf(r.Context()), id, nodeevents.ReinstallStarted, nodeevents.Info, map[string]any{"method": "ssh"}, claims.UserID)
	}
	audit(r.Context(), h.dbOf(r.Context()), claims.UserID, auditAction, "node:"+id, map[string]any{"task_id": taskID})
	writeJSON(w, http.StatusAccepted, map[string]any{"task_id": taskID, "method": "ssh"})
}

func (h *Handler) PostAdminAgentSSHInstall(w http.ResponseWriter, r *http.Request) {
	h.sshTaskHTTP(w, r, "install", "agent.reinstall")
}

func (h *Handler) PostAdminAgentSSHCheck(w http.ResponseWriter, r *http.Request) {
	h.sshTaskHTTP(w, r, "refresh", "agent.ssh_check")
}

func (h *Handler) PostAdminAgentInstallScript(w http.ResponseWriter, r *http.Request) {
	h.GetAdminLocationInstallScript(w, r)
}

func limitOutput(s string, limit int) (string, bool) {
	if len(s) <= limit {
		return s, false
	}
	return s[len(s)-limit:], true
}

func (h *Handler) PostAdminAgentSSHExec(w http.ResponseWriter, r *http.Request) {
	claims, id, loc, ok := h.adminLocationContext(w, r)
	if !ok {
		return
	}
	if _, isKey := apiKeyFromContext(r.Context()); isKey || claims.Role != "owner" {
		writeCodedError(w, http.StatusForbidden, "owner_only", "Произвольные команды на ноде может выполнять только владелец панели")
		return
	}
	var body struct {
		Cmd string `json:"cmd"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	cmd := strings.TrimSpace(body.Cmd)
	if cmd == "" || len(cmd) > 4000 {
		writeError(w, http.StatusUnprocessableEntity, "Команда пустая или длиннее 4000 символов")
		return
	}
	meta := parseMetaMap(loc.Meta)
	if !hasSSHConfigured(loc, meta) {
		writeCodedError(w, http.StatusConflict, "ssh_not_configured", "SSH для ноды не настроен")
		return
	}
	cfg, err := h.nodeSSHConfigTOFU(loc, meta)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	cfg.ExecTimeout = sshExecTimeout
	started := time.Now()
	out, runErr := sshclient.RunCapture(cfg, cmd)
	duration := time.Since(started)
	exitCode := 0
	if runErr != nil {
		exitCode = -1
		var ee interface{ ExitStatus() int }
		if errors.As(runErr, &ee) {
			exitCode = ee.ExitStatus()
		}
	}
	out, truncated := limitOutput(out, sshExecOutputLimit)
	audit(r.Context(), h.dbOf(r.Context()), claims.UserID, "agent.ssh_exec", "node:"+id, map[string]any{
		"cmd": cmd, "exit_code": exitCode, "duration_ms": duration.Milliseconds(),
	})
	_ = nodeevents.Record(r.Context(), h.dbOf(r.Context()), id, nodeevents.SSHExec, nodeevents.Warn,
		map[string]any{"exit_code": exitCode, "duration_ms": duration.Milliseconds()}, claims.UserID)
	resp := map[string]any{
		"stdout": out, "output": out, "ok": runErr == nil, "exit_code": exitCode,
		"duration_ms": duration.Milliseconds(), "truncated": truncated,
	}
	if runErr != nil {
		resp["error"] = runErr.Error()
	}
	writeJSON(w, http.StatusOK, resp)
}
