package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/vortanixapp/panel/pkg/protocol"
	"github.com/vortanixapp/panel/pkg/sshclient"
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

func (h *Handler) GetAdminAgentLogs(w http.ResponseWriter, r *http.Request) {
	claims, id, loc, ok := h.adminLocationContext(w, r)
	if !ok {
		return
	}
	var relayErr error
	if link, err := h.loadAgentLink(r.Context(), id); err == nil && link.can(protocol.CapAgentLogs) {
		res, err := h.agentLogsViaRelay(r, id)
		if err == nil {
			res["location_id"] = id
			writeJSON(w, http.StatusOK, res)
			return
		}
		relayErr = err
	}
	meta := parseMetaMap(loc.Meta)
	if !hasSSHConfigured(loc, meta) {
		if relayErr != nil {
			writeRelayError(w, relayErr)
			return
		}
		writeCodedError(w, http.StatusConflict, "ssh_not_configured", "Журнал недоступен: агент не в сети или устарел, а SSH для ноды не настроен")
		return
	}
	tail := 200
	if t := r.URL.Query().Get("tail"); t != "" {
		if n, err := strconv.Atoi(t); err == nil && n > 0 {
			tail = n
		}
	}
	cfg, err := h.nodeSSHConfigTOFU(loc, meta)
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
		"source":      "ssh",
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

func (h *Handler) loadAgentInfo(r *http.Request, nodeID string) map[string]any {
	var nodeStatus string
	var nodeLastSeen *time.Time
	var metaRaw []byte
	_ = h.readerOf(r.Context()).QueryRow(r.Context(), `
		SELECT status, last_seen_at, COALESCE(meta, '{}'::jsonb) FROM core.nodes WHERE id = $1
	`, nodeID).Scan(&nodeStatus, &nodeLastSeen, &metaRaw)
	meta := parseMetaMap(metaRaw)
	containerRunning, containerKnown := agentContainerRunning(meta)

	var daemonStatus, version, platform, remoteAddr, disconnectReason string
	var pid, rtt *int
	var daemonLastSeen, startedAt, connectedAt, disconnectedAt *time.Time
	err := h.readerOf(r.Context()).QueryRow(r.Context(), `
		SELECT status, COALESCE(version,''), COALESCE(platform,''), pid, last_seen_at,
			started_at, connected_at, disconnected_at, COALESCE(disconnect_reason, ''),
			COALESCE(remote_addr, ''), rtt_ms
		FROM core.node_daemons WHERE node_id = $1
	`, nodeID).Scan(&daemonStatus, &version, &platform, &pid, &daemonLastSeen,
		&startedAt, &connectedAt, &disconnectedAt, &disconnectReason, &remoteAddr, &rtt)
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
		"state":             agentPresenceState(daemonStatus, daemonLastSeen),
		"version":           version,
		"platform":          platform,
		"container":         agentContainerName,
		"relay":             relayHost(),
	}
	if pid != nil {
		out["pid"] = *pid
	}
	if secs, ok := agentUptimeSeconds(isOnline, startedAt, connectedAt); ok {
		out["uptime_sec"] = secs
	}
	if connectedAt != nil {
		out["connected_at"] = connectedAt.Format(time.RFC3339)
	}
	if disconnectedAt != nil {
		out["disconnected_at"] = disconnectedAt.Format(time.RFC3339)
		out["disconnect_reason"] = disconnectReason
	}
	if remoteAddr != "" {
		out["remote_addr"] = remoteAddr
	}
	if rtt != nil {
		out["rtt_ms"] = *rtt
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

func (h *Handler) loadAgentChartMetrics(r *http.Request, nodeID string) ([]map[string]any, []map[string]any) {
	return h.loadMetricSeries(r, nodeID, "agent_cpu_usage"), h.loadMetricSeries(r, nodeID, "agent_ram_usage")
}

func (h *Handler) loadMetricSeries(r *http.Request, nodeID, metricType string) []map[string]any {
	rows, err := h.readerOf(r.Context()).Query(r.Context(), `
		WITH bounds AS (
			SELECT min(measured_at) AS since FROM core.node_metrics
			WHERE node_id = $1 AND metric_type = $2 AND measured_at > now() - interval '48 hours'
		), step AS (
			SELECT since, GREATEST(interval '30 seconds', (now() - since) / 120) AS width
			FROM bounds WHERE since IS NOT NULL
		)
		SELECT avg(m.value)::float8, date_bin(step.width, m.measured_at, step.since) AS bucket
		FROM core.node_metrics m, step
		WHERE m.node_id = $1 AND m.metric_type = $2 AND m.measured_at >= step.since
		GROUP BY bucket
		ORDER BY bucket
	`, nodeID, metricType)
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
		KnownHostKey: loc.SSHHostKey,
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
