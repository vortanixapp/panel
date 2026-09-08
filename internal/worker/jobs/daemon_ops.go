package jobs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/vortanixapp/panel/pkg/sshclient"
)

func (r *Runner) DaemonLoop(ctx context.Context, wake <-chan struct{}) {
	ticker := time.NewTicker(15 * time.Second)
	staleTicker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()
	defer staleTicker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-wake:
			r.heartbeat.Beat(LoopDaemon)
			r.drainDaemonJobs(ctx)
		case <-ticker.C:
			r.heartbeat.Beat(LoopDaemon)
			r.drainDaemonJobs(ctx)
		case <-staleTicker.C:
			r.enqueueStaleDaemonPulls(ctx)
			r.drainDaemonJobs(ctx)
		}
	}
}

func (r *Runner) enqueueStaleDaemonPulls(ctx context.Context) {
	rows, err := r.db.Query(ctx, `
		SELECT n.tenant_id::text, n.id::text
		FROM core.nodes n
		WHERE COALESCE(n.ssh_host, '') <> ''
		  AND COALESCE(n.ssh_user, '') <> ''
		  AND COALESCE(n.ssh_password_enc, '') <> ''
		  AND NOT EXISTS (
		    SELECT 1 FROM core.jobs j
		    WHERE j.type = 'daemon_pull'
		      AND j.status IN ('pending', 'running')
		      AND j.payload->>'node_id' = n.id::text
		  )
		  AND NOT EXISTS (
		    SELECT 1 FROM core.node_metrics m
		    WHERE m.node_id = n.id AND m.measured_at > NOW() - INTERVAL '5 minutes'
		  )
		ORDER BY n.last_seen_at NULLS FIRST
		LIMIT 10
	`)
	if err != nil {
		log.Printf("daemon_pull: не удалось выбрать ноды для обхода: %v", err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var tenantID, nodeID string
		if rows.Scan(&tenantID, &nodeID) != nil {
			continue
		}
		payload, _ := json.Marshal(map[string]string{"node_id": nodeID})
		_, _ = r.db.Exec(ctx, `
			INSERT INTO core.jobs (tenant_id, type, status, payload)
			VALUES ($1, 'daemon_pull', 'pending', $2::jsonb)
		`, tenantID, payload)
		log.Printf("daemon_pull: scheduled stale sync for node %s", nodeID)
	}
}

func (r *Runner) drainDaemonJobs(ctx context.Context) {
	for r.processDaemonJob(ctx) {
	}
	for r.processDaemonPull(ctx) {
	}
}

func (r *Runner) processDaemonJob(ctx context.Context) bool {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return false
	}
	defer tx.Rollback(ctx)

	var jobID, tenantID string
	var payload []byte
	err = tx.QueryRow(ctx, `
		SELECT id::text, tenant_id::text, payload
		FROM core.jobs
		WHERE type = 'daemon_action' AND status = 'pending' AND attempts < 5
		ORDER BY created_at ASC LIMIT 1
		FOR UPDATE SKIP LOCKED
	`).Scan(&jobID, &tenantID, &payload)
	if err != nil {
		return false
	}
	_, _ = tx.Exec(ctx, `UPDATE core.jobs SET status = 'running', attempts = attempts + 1 WHERE id = $1`, jobID)

	var pl struct {
		NodeID string         `json:"node_id"`
		Action string         `json:"action"`
		Params map[string]any `json:"params"`
	}
	_ = json.Unmarshal(payload, &pl)

	node, err := r.loadNodeSSH(ctx, tx, tenantID, pl.NodeID)
	if err != nil {
		r.failJobGeneric(ctx, tx, jobID, err.Error())
		_ = tx.Commit(ctx)
		return true
	}
	cfg := sshclient.Config{Host: node.SSHHost, Port: node.SSHPort, User: node.SSHUser, Password: node.SSHPassword, Timeout: 30 * time.Second}

	var execErr error
	// Вывод ssh держим снаружи switch: без него в задании оставалось голое
	// «Process exited with status 1», а причина отказа — сообщение docker —
	// терялась вместе с буфером.
	var out bytes.Buffer
	switch pl.Action {
	case "restart":
		_, execErr = sshclient.RunCapture(cfg, "sudo docker restart vortanix-agent 2>/dev/null || true")
	case "install":
		relayURL := relayWebsocketURL(envOr("RELAY_PUBLIC_URL", envOr("RELAY_URL", "")))
		cmds, cmdErr := setupCommands("daemon", node.Meta, node.AgentToken, node.ID, relayURL, nil)
		if cmdErr != nil {
			execErr = cmdErr
			break
		}
		execErr = sshclient.Run(cfg, r.withRegistryLogin(ctx, cmds), &out)
	case "update":
		version, _ := pl.Params["version"].(string)
		// Адрес relay берём через relayWebsocketURL, как и при установке. Без
		// него сюда приезжал https://relay.vortanix.app, агент падал на
		// «malformed ws or wss URL», и кнопка обновления не обновляла ноду, а
		// отправляла её в офлайн.
		relayURL := relayWebsocketURL(envOr("RELAY_PUBLIC_URL", envOr("RELAY_URL", "")))
		cmds := daemonAgentCommands(node.AgentToken, node.ID, relayURL, version)
		execErr = sshclient.Run(cfg, r.withRegistryLogin(ctx, cmds), &out)
	case "refresh":
		execErr = r.collectNodeMetrics(ctx, tx, tenantID, pl.NodeID, node)
	default:
		execErr = fmt.Errorf("unknown daemon action: %s", pl.Action)
	}

	if execErr != nil {
		msg := execErr.Error()
		if tail := lastLines(out.String(), 12); tail != "" {
			msg = msg + "\n" + tail
		}
		result, _ := json.Marshal(map[string]string{"error": msg})
		_, _ = tx.Exec(ctx, `UPDATE core.jobs SET status = 'failed', result = $2::jsonb WHERE id = $1`, jobID, result)
		_ = tx.Commit(ctx)
		log.Printf("daemon_action %s на ноде %s: %s", pl.Action, pl.NodeID, msg)
		return true
	}

	result, _ := json.Marshal(map[string]any{"ok": true, "action": pl.Action})
	_, _ = tx.Exec(ctx, `UPDATE core.jobs SET status = 'completed', result = $2::jsonb WHERE id = $1`, jobID, result)
	_ = tx.Commit(ctx)
	return true
}

// lastLines отдаёт хвост вывода: причина отказа docker всегда в последних
// строках, а весь журнал установки в карточке задания не нужен.
func lastLines(s string, n int) string {
	lines := []string{}
	for _, line := range strings.Split(s, "\n") {
		if line = strings.TrimRight(line, "\r "); strings.TrimSpace(line) != "" {
			lines = append(lines, line)
		}
	}
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}

// withRegistryLogin добавляет вход в наш реестр перед командами, которые тянут
// образ агента.
//
// Установка ноды это делала, а действия демона — нет: образ агента лежит в
// закрытом реестре, и docker pull без ключа лицензии отвечал отказом. Кнопка
// «Обновить агента» из-за этого либо не меняла версию, либо оставляла ноду без
// контейнера вовсе.
func (r *Runner) withRegistryLogin(ctx context.Context, cmds []string) []string {
	logins := registryLoginCommands("daemon", r.licenseKey(ctx))
	if len(logins) == 0 {
		return cmds
	}
	return append(logins, cmds...)
}

func (r *Runner) processDaemonPull(ctx context.Context) bool {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return false
	}
	defer tx.Rollback(ctx)

	var jobID, tenantID string
	var payload []byte
	err = tx.QueryRow(ctx, `
		SELECT id::text, tenant_id::text, payload
		FROM core.jobs
		WHERE type = 'daemon_pull' AND status = 'pending' AND attempts < 5
		ORDER BY created_at ASC LIMIT 1
		FOR UPDATE SKIP LOCKED
	`).Scan(&jobID, &tenantID, &payload)
	if err != nil {
		return false
	}
	_, _ = tx.Exec(ctx, `UPDATE core.jobs SET status = 'running', attempts = attempts + 1 WHERE id = $1`, jobID)

	var pl struct {
		NodeID string `json:"node_id"`
	}
	_ = json.Unmarshal(payload, &pl)
	node, err := r.loadNodeSSH(ctx, tx, tenantID, pl.NodeID)
	if err != nil {
		r.failJobGeneric(ctx, tx, jobID, err.Error())
		_ = tx.Commit(ctx)
		return true
	}
	if err := r.collectNodeMetrics(ctx, tx, tenantID, pl.NodeID, node); err != nil {
		result, _ := json.Marshal(map[string]string{"error": err.Error()})
		_, _ = tx.Exec(ctx, `UPDATE core.jobs SET status = 'failed', result = $2::jsonb WHERE id = $1`, jobID, result)
		_ = tx.Commit(ctx)
		return true
	}
	result, _ := json.Marshal(map[string]string{"ok": "true"})
	_, _ = tx.Exec(ctx, `UPDATE core.jobs SET status = 'completed', result = $2::jsonb WHERE id = $1`, jobID, result)
	_ = tx.Commit(ctx)
	return true
}

func (r *Runner) collectNodeMetrics(ctx context.Context, tx pgx.Tx, tenantID, nodeID string, node *nodeSSH) error {
	cfg := sshclient.Config{Host: node.SSHHost, Port: node.SSHPort, User: node.SSHUser, Password: node.SSHPassword, Timeout: 45 * time.Second}

	script := `
set +e
echo "OS:$(uname -s) $(uname -r)"
grep -m1 'model name' /proc/cpuinfo 2>/dev/null | cut -d: -f2 | sed 's/^ *//' | awk '{print "CPU:" $0}'
free -b 2>/dev/null | awk '/Mem:/ {print "RAM:" $2}'
df -B1 / 2>/dev/null | tail -1 | awk '{print "DISK_TOTAL:" $2 "\nDISK_USED:" $3 "\nDISK_AVAIL:" $4}'
awk '{print int($1)}' /proc/uptime 2>/dev/null | awk '{print "UPTIME:" $1 "s"}'
top -bn1 2>/dev/null | grep -E 'Cpu\\(s\\)|%Cpu' | head -1 | awk -F'id,' '{print $1}' | grep -oE '[0-9.]+' | head -1 | awk '{print "CPU_USAGE:" (100-$1)}'
free 2>/dev/null | awk '/Mem:/ {printf "RAM_USAGE:%.2f\n", ($3/$2)*100}'
for u in docker mysql vortanix-sftp vortanix-agent; do
  st=$(systemctl is-active $u 2>/dev/null || echo inactive)
  if [ "$u" = "docker" ] && docker info >/dev/null 2>&1; then st=active; fi
  if [ "$u" = "vortanix-agent" ] && docker ps --format '{{.Names}}' 2>/dev/null | grep -qx vortanix-agent; then st=active; fi
  # MySQL ставится контейнерами vortanix-mysql*, systemd-юнита на ноде нет.
  if [ "$u" = "mysql" ] && docker ps --format '{{.Names}}' 2>/dev/null | grep -q '^vortanix-mysql'; then st=active; fi
  echo "SVC:$u:$st"
done
# Agent container stats
if docker ps --format '{{.Names}}' 2>/dev/null | grep -qx vortanix-agent; then
  docker stats vortanix-agent --no-stream --format 'AGENT_CPU:{{.CPUPerc}}' 2>/dev/null | head -1
  docker stats vortanix-agent --no-stream --format 'AGENT_RAM:{{.MemPerc}}' 2>/dev/null | head -1
  docker inspect vortanix-agent --format 'AGENT_IMAGE:{{.Config.Image}}' 2>/dev/null | head -1
  docker inspect vortanix-agent --format 'AGENT_STARTED:{{.State.StartedAt}}' 2>/dev/null | head -1
  docker inspect vortanix-agent --format 'AGENT_PID:{{.State.Pid}}' 2>/dev/null | head -1
fi
`
	out, err := sshclient.RunCapture(cfg, script)
	if err != nil && strings.TrimSpace(out) == "" {
		return err
	}

	textMetrics := map[string]string{}
	var cpuUsage, ramUsage, agentCPU, agentRAM float64
	var agentImage, agentPID string
	services := map[string]map[string]any{}

	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "SVC:") {
			parts := strings.SplitN(strings.TrimPrefix(line, "SVC:"), ":", 2)
			if len(parts) == 2 {
				state := parts[1]
				if state != "active" {
					state = "inactive"
				}
				if state == "active" {
					state = "active"
				} else if state == "inactive" || state == "failed" {
					state = "inactive"
				}
				label := parts[0]
				switch parts[0] {
				case "vortanix-sftp":
					label = "SFTP"
				case "vortanix-agent":
					label = "Vortanix Agent"
				case "docker":
					label = "Docker"
				case "mysql":
					label = "MySQL"
				}
				services[parts[0]] = map[string]any{"state": mapServiceState(parts[1]), "error": nil, "label": label}
			}
			continue
		}
		if i := strings.Index(line, ":"); i > 0 {
			key := line[:i]
			val := strings.TrimSpace(line[i+1:])
			textMetrics[key] = val
			switch key {
			case "CPU_USAGE":
				cpuUsage, _ = strconv.ParseFloat(strings.TrimSuffix(val, "%"), 64)
			case "RAM_USAGE":
				ramUsage, _ = strconv.ParseFloat(strings.TrimSuffix(val, "%"), 64)
			case "AGENT_CPU":
				agentCPU, _ = strconv.ParseFloat(strings.TrimSuffix(val, "%"), 64)
			case "AGENT_RAM":
				agentRAM, _ = strconv.ParseFloat(strings.TrimSuffix(val, "%"), 64)
			case "AGENT_IMAGE":
				agentImage = val
			case "AGENT_PID":
				agentPID = val
			}
		}
	}

	now := time.Now()
	insertMetric := func(metricType string, value float64, text *string) {
		_, _ = tx.Exec(ctx, `
			INSERT INTO core.node_metrics (tenant_id, node_id, metric_type, value, text_value, measured_at)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, tenantID, nodeID, metricType, value, text, now)
	}

	if v := textMetrics["OS"]; v != "" {
		insertMetric("os_info", 0, &v)
	}
	if v := textMetrics["CPU"]; v != "" {
		insertMetric("cpu_model", 0, &v)
	}
	if v := textMetrics["RAM"]; v != "" {
		insertMetric("ram_total", 0, &v)
	}
	if v := textMetrics["DISK_TOTAL"]; v != "" {
		insertMetric("disk_total", 0, &v)
	}
	if v := textMetrics["DISK_USED"]; v != "" {
		insertMetric("disk_used", 0, &v)
	}
	if v := textMetrics["DISK_AVAIL"]; v != "" {
		insertMetric("disk_available", 0, &v)
	}
	if v := textMetrics["UPTIME"]; v != "" {
		insertMetric("uptime", 0, &v)
	}
	insertMetric("cpu_usage", cpuUsage, nil)
	insertMetric("ram_usage", ramUsage, nil)
	if agentCPU > 0 {
		insertMetric("agent_cpu_usage", agentCPU, nil)
	}
	if agentRAM > 0 {
		insertMetric("agent_ram_usage", agentRAM, nil)
	}

	b, _ := json.Marshal(services)
	_, _ = tx.Exec(ctx, `
		UPDATE core.nodes
		SET meta = COALESCE(meta, '{}'::jsonb) || jsonb_build_object('service_statuses', $3::jsonb),
		    last_seen_at = now()
		WHERE id = $1 AND tenant_id = $2
	`, nodeID, tenantID, b)

	agentActive := false
	if svc, ok := services["vortanix-agent"]; ok {
		state, _ := svc["state"].(string)
		agentActive = state == "active"
	}
	status := "offline"
	if agentActive {
		status = "online"
	}
	var pidVal *int
	if agentPID != "" {
		if p, err := strconv.Atoi(agentPID); err == nil {
			pidVal = &p
		}
	}
	platform := textMetrics["OS"]
	_, _ = tx.Exec(ctx, `
		INSERT INTO core.node_daemons (tenant_id, node_id, status, version, platform, pid, last_seen_at)
		VALUES ($1, $2, $3, NULLIF($4, ''), NULLIF($5, ''), $6, now())
		ON CONFLICT (node_id) DO UPDATE SET
			status = EXCLUDED.status,
			version = COALESCE(NULLIF(EXCLUDED.version, ''), core.node_daemons.version),
			platform = COALESCE(NULLIF(EXCLUDED.platform, ''), core.node_daemons.platform),
			pid = COALESCE(EXCLUDED.pid, core.node_daemons.pid),
			last_seen_at = now(), updated_at = now()
	`, tenantID, nodeID, status, agentImage, platform, pidVal)

	log.Printf("daemon_pull: node %s cpu=%.1f ram=%.1f", nodeID, cpuUsage, ramUsage)
	return nil
}

func mapServiceState(systemdState string) string {
	switch strings.TrimSpace(systemdState) {
	case "active":
		return "active"
	case "inactive", "failed":
		return "inactive"
	default:
		return "unknown"
	}
}
