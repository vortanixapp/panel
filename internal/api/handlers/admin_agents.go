package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/vortanixapp/panel/internal/api/sshclient"
)

const agentContainerName = "vortanix-agent"

func relayHost() string {
	raw := strings.TrimSpace(envOr("RELAY_PUBLIC_URL", ""))
	if raw == "" {
		return ""
	}
	for _, prefix := range []string{"wss://", "ws://", "https://", "http://"} {
		raw = strings.TrimPrefix(raw, prefix)
	}
	if idx := strings.IndexAny(raw, "/?"); idx >= 0 {
		raw = raw[:idx]
	}
	return raw
}

func (h *Handler) ListAdminAgents(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	rows, err := h.readerOf(r.Context()).Query(r.Context(), `
		SELECT n.id::text, n.name, COALESCE(n.meta->>'code', n.fqdn, ''), COALESCE(n.country, ''),
			COALESCE(n.region, ''), n.ssh_host, n.status, n.last_seen_at,
			COALESCE(d.status, 'unknown'), COALESCE(d.version, ''), COALESCE(d.platform, ''),
			d.pid, d.uptime_sec, d.last_seen_at
		FROM core.nodes n
		LEFT JOIN core.node_daemons d ON d.node_id = n.id
		WHERE n.tenant_id = $1
		ORDER BY COALESCE(n.sort_order, 0), n.name
	`, claims.TenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()

	list := []map[string]any{}
	for rows.Next() {
		var id, name, code, country, region string
		var sshHost *string
		var nodeStatus string
		var nodeLastSeen *time.Time
		var daemonStatus, version, platform string
		var pid *int
		var uptime *float64
		var daemonLastSeen *time.Time
		if rows.Scan(&id, &name, &code, &country, &region, &sshHost, &nodeStatus, &nodeLastSeen,
			&daemonStatus, &version, &platform, &pid, &uptime, &daemonLastSeen) != nil {
			continue
		}
		lastSeen := nodeLastSeen
		if daemonLastSeen != nil && (lastSeen == nil || daemonLastSeen.After(*lastSeen)) {
			lastSeen = daemonLastSeen
		}
		isOnline := agentDaemonOnline(daemonStatus, daemonLastSeen)
		item := map[string]any{
			"id": id, "location_id": id, "node_id": id,
			"name": name, "code": code, "country": country, "region": region,
			"host": strPtr(sshHost), "status": daemonStatus, "is_online": isOnline,
			"version": version, "platform": platform,
			"location": map[string]any{
				"id": id, "name": name, "code": code, "region": region,
			},
		}
		if pid != nil {
			item["pid"] = *pid
		}
		if uptime != nil {
			item["uptime_sec"] = *uptime
		}
		if lastSeen != nil {
			item["last_seen"] = lastSeen.Format(time.RFC3339)
			item["last_seen_human"] = formatLastSeenHuman(*lastSeen)
		}
		list = append(list, item)
	}
	writeJSON(w, http.StatusOK, map[string]any{"agents": list, "daemons": list})
}

func (h *Handler) GetAdminAgentShow(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	loc, err := h.loadLocationRow(r, claims.TenantID, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "location not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	meta := parseMetaMap(loc.Meta)
	daemon := h.loadAgentInfo(r, claims.TenantID, id)
	cpuMetrics, ramMetrics := h.loadAgentChartMetrics(r, claims.TenantID, id)
	location := map[string]any{
		"id": id, "name": loc.Name, "code": metaString(meta, "code"),
		"ssh_host": strPtr(loc.SSHHost), "country": strPtr(loc.Country),
		"region": strPtr(loc.Region), "ip_address": strPtr(loc.IPAddress),
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"location": location,
		"agent":    daemon,
		"daemon":   daemon,
		"metrics": map[string]any{
			"agent_cpu_usage":  cpuMetrics,
			"agent_ram_usage":  ramMetrics,
			"daemon_cpu_usage": cpuMetrics,
			"daemon_ram_usage": ramMetrics,
		},
	})
}

func (h *Handler) GetAdminAgentLogs(w http.ResponseWriter, r *http.Request) {
	claims, id, loc, ok := h.adminLocationContext(w, r)
	if !ok {
		return
	}
	meta := parseMetaMap(loc.Meta)
	if !hasSSHConfigured(loc, meta) {
		writeError(w, http.StatusBadRequest, "SSH is not configured for this location")
		return
	}
	tail := 200
	if t := r.URL.Query().Get("tail"); t != "" {
		if n, err := strconv.Atoi(t); err == nil && n > 0 {
			tail = n
		}
	}
	cfg, err := nodeSSHConfig(loc, meta)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	out, runErr := sshclient.RunCapture(cfg, fmt.Sprintf(
		"sudo docker logs %s --tail %d 2>&1 || echo 'Agent container not found'",
		agentContainerName, tail,
	))
	lines := []string{}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimRight(line, "\r")
		if line != "" || len(lines) == 0 {
			lines = append(lines, line)
		}
	}
	resp := map[string]any{
		"location_id": id,
		"hostname":    cfg.Host,
		"logs":        lines,
		"container_found": !strings.Contains(out, "No such container") &&
			!strings.Contains(out, "Agent container not found"),
	}
	if runErr != nil {
		resp["error"] = runErr.Error()
	}
	_ = claims
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) GetAdminAgentServers(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	rows, err := h.readerOf(r.Context()).Query(r.Context(), `
		SELECT s.id::text, s.name, s.status, COALESCE(s.runtime_status, ''), COALESCE(s.provisioning_status, ''),
			COALESCE(s.game_id, ''), COALESCE(g.name, ''), COALESCE(g.slug, g.code, ''),
			COALESCE(u.email, ''), COALESCE(p.display_name, u.email, '')
		FROM core.servers s
		LEFT JOIN core.games g ON g.tenant_id = s.tenant_id
			AND (g.id::text = s.game_id OR g.slug = s.game_id OR g.code = s.game_id)
		LEFT JOIN core.users u ON u.id = s.user_id
		LEFT JOIN core.user_profiles p ON p.user_id = u.id
		WHERE s.tenant_id = $1 AND s.node_id = $2
		ORDER BY s.created_at DESC
	`, claims.TenantID, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()

	servers := []map[string]any{}
	for rows.Next() {
		var sid, name, status, runtime, prov, gameID, gameName, gameSlug, userEmail, userName string
		if rows.Scan(&sid, &name, &status, &runtime, &prov, &gameID, &gameName, &gameSlug, &userEmail, &userName) != nil {
			continue
		}
		servers = append(servers, map[string]any{
			"id": sid, "name": name, "status": status,
			"runtime_status": runtime, "provisioning_status": prov,
			"container_name": "vortanix-" + sid,
			"game":           map[string]any{"code": gameSlug, "name": gameName, "id": gameID},
			"user":           map[string]any{"email": userEmail, "name": userName},
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"servers": servers})
}

func (h *Handler) PostAdminAgentExec(w http.ResponseWriter, r *http.Request) {
	claims, id, loc, ok := h.adminLocationContext(w, r)
	if !ok {
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
	if cmd == "" {
		writeError(w, http.StatusUnprocessableEntity, "Command is required")
		return
	}
	meta := parseMetaMap(loc.Meta)
	if !hasSSHConfigured(loc, meta) {
		writeError(w, http.StatusBadRequest, "SSH is not configured for this location")
		return
	}
	cfg, err := nodeSSHConfig(loc, meta)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	out, runErr := sshclient.RunCapture(cfg, cmd)
	audit(r.Context(), h.dbOf(r.Context()), claims.TenantID, claims.UserID, "agent.exec", "node:"+id, map[string]any{"cmd": cmd})
	resp := map[string]any{"stdout": out, "output": out, "ok": runErr == nil}
	if runErr != nil {
		resp["error"] = runErr.Error()
		writeJSON(w, http.StatusOK, resp)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) loadAgentInfo(r *http.Request, tenantID, nodeID string) map[string]any {
	var nodeStatus string
	var nodeLastSeen *time.Time
	var metaRaw []byte
	_ = h.readerOf(r.Context()).QueryRow(r.Context(), `
		SELECT status, last_seen_at, COALESCE(meta, '{}'::jsonb) FROM core.nodes WHERE id = $1 AND tenant_id = $2
	`, nodeID, tenantID).Scan(&nodeStatus, &nodeLastSeen, &metaRaw)
	meta := parseMetaMap(metaRaw)
	containerRunning, containerKnown := agentContainerRunning(meta)

	var daemonStatus, version, platform string
	var pid *int
	var uptime *float64
	var daemonLastSeen *time.Time
	err := h.readerOf(r.Context()).QueryRow(r.Context(), `
		SELECT status, COALESCE(version,''), COALESCE(platform,''), pid, uptime_sec, last_seen_at
		FROM core.node_daemons WHERE tenant_id = $1 AND node_id = $2
	`, tenantID, nodeID).Scan(&daemonStatus, &version, &platform, &pid, &uptime, &daemonLastSeen)
	if errors.Is(err, pgx.ErrNoRows) {
		daemonStatus = "unknown"
	} else if err != nil {
		return map[string]any{"status": "unknown", "is_online": false}
	}

	isOnline := agentDaemonOnline(daemonStatus, daemonLastSeen)
	agentStatus := resolveAgentStatus(isOnline, containerKnown, containerRunning, daemonStatus)

	lastSeen := daemonLastSeen
	if lastSeen == nil {
		lastSeen = nodeLastSeen
	}
	nodeReachable := nodeLastSeen != nil && time.Since(*nodeLastSeen) < 10*time.Minute

	out := map[string]any{
		"location_id":       nodeID,
		"node_id":           nodeID,
		"status":            agentStatus,
		"daemon_status":     daemonStatus,
		"node_status":       nodeStatus,
		"node_reachable":    nodeReachable,
		"container_running": containerRunning,
		"container_known":   containerKnown,
		"is_online":         isOnline,
		"version":           version,
		"platform":          platform,
		"container":         agentContainerName,
		"relay":             relayHost(),
	}
	if pid != nil {
		out["pid"] = *pid
	}
	if uptime != nil {
		out["uptime_sec"] = *uptime
	}
	if lastSeen != nil {
		out["last_seen"] = lastSeen.Format(time.RFC3339)
		out["last_seen_at"] = lastSeen.Format(time.RFC3339)
		out["last_seen_human"] = formatLastSeenHuman(*lastSeen)
	}
	if nodeLastSeen != nil {
		out["node_last_seen_human"] = formatLastSeenHuman(*nodeLastSeen)
	}
	return out
}

func agentContainerRunning(meta map[string]any) (running bool, known bool) {
	statuses, ok := meta["service_statuses"].(map[string]any)
	if !ok {
		return false, false
	}
	agent, ok := statuses["vortanix-agent"].(map[string]any)
	if !ok {
		return false, false
	}
	state, _ := agent["state"].(string)
	return state == "active", true
}

func resolveAgentStatus(isOnline, containerKnown, containerRunning bool, daemonStatus string) string {
	switch {
	case isOnline:
		return "online"
	case containerKnown && !containerRunning:
		return "not_installed"
	case containerKnown && containerRunning:
		return "offline"
	case daemonStatus == "offline":
		return "offline"
	case daemonStatus == "online":
		return "offline"
	default:
		return "unknown"
	}
}

func (h *Handler) loadAgentChartMetrics(r *http.Request, tenantID, nodeID string) ([]map[string]any, []map[string]any) {
	cpu := h.loadMetricSeries(r, tenantID, nodeID, "agent_cpu_usage")
	if len(cpu) == 0 {
		cpu = h.loadMetricSeries(r, tenantID, nodeID, "cpu_usage")
	}
	ram := h.loadMetricSeries(r, tenantID, nodeID, "agent_ram_usage")
	if len(ram) == 0 {
		ram = h.loadMetricSeries(r, tenantID, nodeID, "ram_usage")
	}
	return cpu, ram
}

func (h *Handler) loadMetricSeries(r *http.Request, tenantID, nodeID, metricType string) []map[string]any {
	rows, err := h.readerOf(r.Context()).Query(r.Context(), `
		SELECT value, measured_at FROM core.node_metrics
		WHERE tenant_id = $1 AND node_id = $2 AND metric_type = $3
		ORDER BY measured_at DESC LIMIT 50
	`, tenantID, nodeID, metricType)
	if err != nil {
		return nil
	}
	defer rows.Close()
	points := make([]map[string]any, 0)
	for rows.Next() {
		var value float64
		var measuredAt time.Time
		if rows.Scan(&value, &measuredAt) == nil {
			ts := measuredAt.Format(time.RFC3339)
			points = append(points, map[string]any{
				"measured_at": ts, "value": value, "t": ts, "v": value,
			})
		}
	}
	for i, j := 0, len(points)-1; i < j; i, j = i+1, j-1 {
		points[i], points[j] = points[j], points[i]
	}
	return points
}

func nodeSSHConfig(loc *locationRow, meta map[string]any) (sshclient.Config, error) {
	if loc.SSHHost == nil || *loc.SSHHost == "" || loc.SSHUser == nil || *loc.SSHUser == "" {
		return sshclient.Config{}, fmt.Errorf("ssh not configured")
	}
	pass := ""
	if loc.SSHPassword != nil {
		pass = *loc.SSHPassword
	}
	if pass == "" {
		pass = metaString(meta, "ssh_password")
	}
	if pass == "" {
		return sshclient.Config{}, fmt.Errorf("ssh password not set")
	}
	return sshclient.Config{
		Host: *loc.SSHHost, Port: loc.SSHPort, User: *loc.SSHUser, Password: pass,
	}, nil
}

func agentDaemonOnline(daemonStatus string, lastSeen *time.Time) bool {
	if daemonStatus != "online" || lastSeen == nil {
		return false
	}
	return time.Since(*lastSeen) < 90*time.Second
}

func formatLastSeenHuman(t time.Time) string {
	diff := time.Since(t)
	if diff < time.Minute {
		return "только что"
	}
	if diff < time.Hour {
		return fmt.Sprintf("%d мин назад", int(diff.Minutes()))
	}
	if diff < 24*time.Hour {
		return fmt.Sprintf("%d ч назад", int(diff.Hours()))
	}
	return t.Format("02.01.2006 15:04")
}

func strPtr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
