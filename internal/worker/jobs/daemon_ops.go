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

	"github.com/vortanixapp/panel/pkg/nodeevents"
	"github.com/vortanixapp/panel/pkg/sshclient"
)

func (r *Runner) DaemonLoop(ctx context.Context, wake <-chan struct{}) {
	ticker := time.NewTicker(15 * time.Second)
	staleTicker := time.NewTicker(2 * time.Minute)
	purgeTicker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	defer staleTicker.Stop()
	defer purgeTicker.Stop()
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
		case <-purgeTicker.C:
			r.purgeDaemonJobs(ctx)
		}
	}
}

func (r *Runner) enqueueStaleDaemonPulls(ctx context.Context) {
	rows, err := r.db.Query(ctx, `
		SELECT n.id::text
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
		    WHERE m.node_id = n.id AND m.metric_type = 'cpu_usage'
		      AND m.measured_at > NOW() - INTERVAL '5 minutes'
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
		var nodeID string
		if rows.Scan(&nodeID) != nil {
			continue
		}
		payload, _ := json.Marshal(map[string]string{"node_id": nodeID})
		_, _ = r.db.Exec(ctx, `
			INSERT INTO core.jobs ( type, status, payload)
			VALUES ( 'daemon_pull', 'pending', $1::jsonb)
		`, payload)
		log.Printf("daemon_pull: scheduled stale sync for node %s", nodeID)
	}
}

func (r *Runner) drainDaemonJobs(ctx context.Context) {
	for r.processDaemonJob(ctx) {
	}
	for r.processDaemonPull(ctx) {
	}
	r.processServerDestroys(ctx)
}

const (
	daemonActionTimeout = 20 * time.Minute
	daemonPullTimeout   = 2 * time.Minute
)

func (r *Runner) claimDaemonJob(ctx context.Context, jobType string) (string, []byte, bool) {
	var jobID string
	var payload []byte
	err := r.db.QueryRow(ctx, `
		UPDATE core.jobs SET status = 'running', attempts = attempts + 1
		WHERE id = (
			SELECT id FROM core.jobs
			WHERE type = $1 AND status = 'pending' AND attempts < 5
			ORDER BY created_at ASC
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		)
		RETURNING id::text, payload
	`, jobType).Scan(&jobID, &payload)
	if err != nil {
		return "", nil, false
	}
	return jobID, payload, true
}

func (r *Runner) settleSSHTask(ctx context.Context, jobID string, execErr error) {
	status, msg := "done", ""
	if execErr != nil {
		status, msg = "failed", execErr.Error()
	}
	_, _ = r.db.Exec(ctx, `
		UPDATE core.node_tasks
		SET status = $2, error = NULLIF($3, ''), finished_at = now(), updated_at = now()
		WHERE job_id = $1::uuid AND status IN ('queued', 'sent', 'running')
	`, jobID, status, msg)
}

func (r *Runner) finishDaemonJob(ctx context.Context, jobID string, execErr error, ok map[string]any) {
	r.settleSSHTask(ctx, jobID, execErr)
	if execErr != nil {
		result, _ := json.Marshal(map[string]string{"error": execErr.Error()})
		_, _ = r.db.Exec(ctx, `UPDATE core.jobs SET status = 'failed', result = $2::jsonb WHERE id = $1`, jobID, result)
		return
	}
	result, _ := json.Marshal(ok)
	_, _ = r.db.Exec(ctx, `UPDATE core.jobs SET status = 'completed', result = $2::jsonb WHERE id = $1`, jobID, result)
}

func (r *Runner) processDaemonJob(ctx context.Context) bool {
	jobID, payload, ok := r.claimDaemonJob(ctx, "daemon_action")
	if !ok {
		return false
	}
	_, _ = r.db.Exec(ctx, `
		UPDATE core.node_tasks SET status = 'running', updated_at = now()
		WHERE job_id = $1::uuid AND status IN ('queued', 'sent')
	`, jobID)

	var pl struct {
		NodeID string         `json:"node_id"`
		Action string         `json:"action"`
		Params map[string]any `json:"params"`
	}
	_ = json.Unmarshal(payload, &pl)

	node, err := r.loadNodeSSH(ctx, r.db, pl.NodeID)
	if err != nil {
		r.failJobGeneric(ctx, r.db, jobID, err.Error())
		r.settleSSHTask(ctx, jobID, err)
		if pl.Action == "update" {
			r.failAgentUpdate(ctx, pl.NodeID, err.Error())
		}
		return true
	}
	cfg := r.sshConfig(node, daemonActionTimeout)

	var execErr error
	var out bytes.Buffer
	switch pl.Action {
	case "restart":
		_, execErr = sshclient.RunCapture(cfg, "sudo docker restart vortanix-agent")
		if execErr == nil {
			_ = nodeevents.Record(ctx, r.db, pl.NodeID, nodeevents.RestartRequested, nodeevents.Info,
				map[string]any{"method": "ssh"}, "")
		}
	case "install":
		relayURL, relayPin := r.relayTarget(ctx)
		cmds, cmdErr := setupCommands("daemon", node.Meta, node.ID, relayURL, relayPin)
		if cmdErr != nil {
			execErr = cmdErr
			break
		}
		cmds, secrets := r.withRegistryLogin(ctx, cmds)
		execErr = sshclient.RunInput(cfg, cmds, mergeSecrets(secrets, agentSecrets(node.AgentToken)), &out)
		if execErr != nil {
			_ = nodeevents.Record(ctx, r.db, pl.NodeID, nodeevents.ReinstallFailed, nodeevents.Error,
				map[string]any{"error": execErr.Error()}, "")
		} else {
			_ = nodeevents.Record(ctx, r.db, pl.NodeID, nodeevents.ReinstallDone, nodeevents.Success, nil, "")
		}
	case "update":
		version, _ := pl.Params["version"].(string)
		relayURL, relayPin := r.relayTarget(ctx)
		cmds, secrets := r.withRegistryLogin(ctx, daemonAgentCommands(node.ID, relayURL, relayPin, version))
		execErr = sshclient.RunInput(cfg, cmds, mergeSecrets(secrets, agentSecrets(node.AgentToken)), &out)
		if execErr != nil {
			r.failAgentUpdate(ctx, pl.NodeID, execErr.Error())
		}
	case "refresh":
		execErr = r.collectNodeMetrics(ctx, pl.NodeID, node)
	default:
		execErr = fmt.Errorf("unknown daemon action: %s", pl.Action)
	}

	if execErr != nil {
		log.Printf("daemon_action %s на ноде %s: %s", pl.Action, pl.NodeID, execErr)
	}
	r.finishDaemonJob(ctx, jobID, execErr, map[string]any{"ok": true, "action": pl.Action})
	return true
}

func (r *Runner) failAgentUpdate(ctx context.Context, nodeID, msg string) {
	tag, err := r.db.Exec(ctx, `
		UPDATE core.nodes
		SET agent_update = agent_update || jsonb_build_object(
			'status', 'failed', 'error', $2::text, 'updated_at', now(), 'finished_at', now())
		WHERE id = $1 AND agent_update->>'status' IN ('pending', 'pulling', 'restarting')
	`, nodeID, msg)
	if err != nil || tag.RowsAffected() == 0 {
		return
	}
	_ = nodeevents.Record(ctx, r.db, nodeID, nodeevents.UpdateFailed, nodeevents.Error,
		map[string]any{"error": msg, "method": "ssh"}, "")
}

func (r *Runner) purgeDaemonJobs(ctx context.Context) {
	tag, err := r.db.Exec(ctx, `
		DELETE FROM core.jobs
		WHERE status IN ('completed', 'failed', 'cancelled')
		  AND (
			(type = 'daemon_pull' AND created_at < now() - interval '3 days')
			OR (type = 'daemon_action' AND created_at < now() - interval '30 days')
		  )
	`)
	if err != nil {
		log.Printf("daemon: очистка старых задач: %v", err)
		return
	}
	if n := tag.RowsAffected(); n > 0 {
		log.Printf("daemon: удалено %d старых задач обхода нод", n)
	}
}

func (r *Runner) withRegistryLogin(ctx context.Context, cmds []string) ([]string, map[string]string) {
	logins, secrets := registryLoginCommands("daemon", r.licenseKey(ctx))
	if len(logins) == 0 {
		return cmds, nil
	}
	return append(logins, cmds...), secrets
}

func (r *Runner) processDaemonPull(ctx context.Context) bool {
	jobID, payload, ok := r.claimDaemonJob(ctx, "daemon_pull")
	if !ok {
		return false
	}
	var pl struct {
		NodeID string `json:"node_id"`
	}
	_ = json.Unmarshal(payload, &pl)
	node, err := r.loadNodeSSH(ctx, r.db, pl.NodeID)
	if err != nil {
		r.failJobGeneric(ctx, r.db, jobID, err.Error())
		return true
	}
	err = r.collectNodeMetrics(ctx, pl.NodeID, node)
	r.finishDaemonJob(ctx, jobID, err, map[string]any{"ok": "true"})
	return true
}

func (r *Runner) collectNodeMetrics(ctx context.Context, nodeID string, node *nodeSSH) error {
	cfg := r.sshConfig(node, daemonPullTimeout)

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
  if [ "$u" = "mysql" ] && docker ps --format '{{.Names}}' 2>/dev/null | grep -q '^vortanix-mysql'; then st=active; fi
  echo "SVC:$u:$st"
done
if docker ps --format '{{.Names}}' 2>/dev/null | grep -qx vortanix-agent; then
  docker stats vortanix-agent --no-stream --format 'AGENT_CPU:{{.CPUPerc}}' 2>/dev/null | head -1
  docker stats vortanix-agent --no-stream --format 'AGENT_RAM:{{.MemPerc}}' 2>/dev/null | head -1
  docker inspect vortanix-agent --format 'AGENT_IMAGE:{{.Config.Image}}' 2>/dev/null | head -1
  docker inspect vortanix-agent --format 'AGENT_STARTED:{{.State.StartedAt}}' 2>/dev/null | head -1
  docker inspect vortanix-agent --format 'AGENT_PID:{{.State.Pid}}' 2>/dev/null | head -1
fi
printf 'MYSQL_INSTANCES:%s\n' "$( (cat /opt/vortanix/mysql-instances.json 2>/dev/null || sudo -n cat /opt/vortanix/mysql-instances.json 2>/dev/null) | tr -d '\n')"
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

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	now := time.Now()
	insertMetric := func(metricType string, value float64, text *string) {
		_, _ = tx.Exec(ctx, `
			INSERT INTO core.node_metrics ( node_id, metric_type, value, text_value, measured_at)
			VALUES ( $1, $2, $3, $4, $5)
		`, nodeID, metricType, value, text, now)
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
		SET meta = COALESCE(meta, '{}'::jsonb) || jsonb_build_object('service_statuses', $2::jsonb)
		WHERE id = $1
	`, nodeID, b)
	if raw := textMetrics["MYSQL_INSTANCES"]; raw != "" {
		if _, err := r.syncMySQLInstances(ctx, tx, nodeID, raw); err != nil {
			log.Printf("синхронизация ноды %s: инстансы MySQL не записаны: %v", nodeID, err)
		}
	}

	var pidVal *int
	if agentPID != "" {
		if p, err := strconv.Atoi(agentPID); err == nil {
			pidVal = &p
		}
	}
	platform := textMetrics["OS"]
	_, _ = tx.Exec(ctx, `
		INSERT INTO core.node_daemons (node_id, status, platform, pid, host)
		VALUES ($1, 'offline', NULLIF($2, ''), $3,
			CASE WHEN $4::text = '' THEN '{}'::jsonb
			     ELSE jsonb_build_object('agent', jsonb_build_object('image', $4::text)) END)
		ON CONFLICT (node_id) DO UPDATE SET
			platform = COALESCE(NULLIF(EXCLUDED.platform, ''), core.node_daemons.platform),
			pid = COALESCE(EXCLUDED.pid, core.node_daemons.pid),
			host = CASE WHEN $4::text = '' THEN core.node_daemons.host
			            ELSE core.node_daemons.host || jsonb_build_object('agent',
			                COALESCE(core.node_daemons.host->'agent', '{}'::jsonb)
			                || jsonb_build_object('image', $4::text)) END,
			updated_at = now()
	`, nodeID, platform, pidVal, agentImage)

	if err := tx.Commit(ctx); err != nil {
		return err
	}
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
