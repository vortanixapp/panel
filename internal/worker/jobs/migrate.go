package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/vortanix/vortanix/pkg/notify"
	"github.com/vortanix/vortanix/pkg/portalloc"

	"github.com/vortanix/vortanix/internal/worker/relay"
	"github.com/vortanix/vortanix/pkg/sshclient"
)

const (
	nodeServersDir         = "/var/lib/vortanix/servers"
	migrateTransferTimeout = 6 * time.Hour
	migrateStopWait        = 90 * time.Second
)

func (r *Runner) MigrateLoop(ctx context.Context, wake <-chan struct{}) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-wake:
			r.heartbeat.Beat(LoopMigrate)
			r.drainMigrate(ctx)
		case <-ticker.C:
			r.heartbeat.Beat(LoopMigrate)
			r.drainMigrate(ctx)
		}
	}
}

func (r *Runner) drainMigrate(ctx context.Context) {
	for r.processMigrateOne(ctx) {
	}
}

type migratePayload struct {
	ServerID     string `json:"server_id"`
	ToNodeID     string `json:"to_node_id"`
	MigrationID  string `json:"migration_id"`
	RemoveSource bool   `json:"remove_source"`
}

func (r *Runner) processMigrateOne(ctx context.Context) bool {
	var jobID, tenantID string
	var payload []byte
	err := r.db.QueryRow(ctx, `
		UPDATE core.jobs SET status = 'running', attempts = attempts + 1
		WHERE id = (
			SELECT id FROM core.jobs
			WHERE type = 'migrate_server' AND status = 'pending' AND attempts < 5
			ORDER BY created_at ASC
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		)
		RETURNING id::text, tenant_id::text, payload
	`).Scan(&jobID, &tenantID, &payload)
	if err != nil {
		return false
	}

	var pl migratePayload
	_ = json.Unmarshal(payload, &pl)
	if pl.ServerID == "" || pl.ToNodeID == "" {
		r.failJobDirect(ctx, jobID, "missing server_id or to_node_id in payload")
		return true
	}

	beat, stopBeat := context.WithCancel(ctx)
	go func() {
		t := time.NewTicker(30 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-beat.Done():
				return
			case <-t.C:
				r.heartbeat.Beat(LoopMigrate)
			}
		}
	}()
	defer stopBeat()

	if err := r.runMigration(ctx, tenantID, pl); err != nil {
		r.failMigration(ctx, tenantID, pl, err.Error())
		r.failJobDirect(ctx, jobID, err.Error())
		return true
	}
	_, _ = r.db.Exec(ctx, `
		UPDATE core.jobs SET status = 'completed', result = '{"ok":true}'::jsonb WHERE id = $1
	`, jobID)
	return true
}

type migrateServer struct {
	Name         string
	GameID       string
	NodeID       string
	Limits       map[string]any
	PrimaryPort  int
	DockerImage  string
	PreviousName string
}

func (r *Runner) runMigration(ctx context.Context, tenantID string, pl migratePayload) error {
	var srv migrateServer
	var limits []byte
	err := r.db.QueryRow(ctx, `
		SELECT name, game_id, node_id::text, limits
		FROM core.servers WHERE id = $1 AND tenant_id = $2
	`, pl.ServerID, tenantID).Scan(&srv.Name, &srv.GameID, &srv.NodeID, &limits)
	if err != nil {
		return fmt.Errorf("сервер не найден")
	}
	_ = json.Unmarshal(limits, &srv.Limits)
	if srv.NodeID == pl.ToNodeID {
		return fmt.Errorf("сервер уже на этой ноде")
	}

	source, err := r.loadNodeSSH(ctx, r.db, tenantID, srv.NodeID)
	if err != nil {
		return fmt.Errorf("исходная нода: %w", err)
	}
	target, err := r.loadNodeSSH(ctx, r.db, tenantID, pl.ToNodeID)
	if err != nil {
		return fmt.Errorf("нода назначения: %w", err)
	}

	r.migrationStage(ctx, pl.MigrationID, "running", "stopping")
	_, _ = r.db.Exec(ctx, `
		UPDATE core.servers SET provisioning_status = 'migrating' WHERE id = $1 AND tenant_id = $2
	`, pl.ServerID, tenantID)

	srcCfg := sshConfigFor(source, 0)

	if err := r.stopServerForMigration(ctx, srv.NodeID, pl.ServerID, srcCfg); err != nil {
		r.restoreAfterFailure(ctx, tenantID, pl.ServerID, srv, srv.NodeID)
		return err
	}

	r.migrationStage(ctx, pl.MigrationID, "running", "transferring")
	dir := serverDirOnNode(pl.ServerID)
	srcCmd := fmt.Sprintf("tar -C %s -czf - .", shellQuote(dir))
	dstCmd := fmt.Sprintf("mkdir -p %s && tar -C %s -xzf -", shellQuote(dir), shellQuote(dir))
	bytes, err := sshclient.StreamBetween(
		sshConfigFor(source, migrateTransferTimeout), srcCmd,
		sshConfigFor(target, migrateTransferTimeout), dstCmd,
		func(total int64) { r.migrationBytes(ctx, pl.MigrationID, total) },
	)
	r.migrationBytes(ctx, pl.MigrationID, bytes)
	if err != nil {
		r.restoreAfterFailure(ctx, tenantID, pl.ServerID, srv, srv.NodeID)
		return fmt.Errorf("перенос файлов: %w", err)
	}

	r.migrationStage(ctx, pl.MigrationID, "running", "switching")
	if err := r.switchServerNode(ctx, tenantID, pl.ServerID, pl.ToNodeID, srv.GameID); err != nil {
		r.restoreAfterFailure(ctx, tenantID, pl.ServerID, srv, srv.NodeID)
		return fmt.Errorf("переключение ноды: %w", err)
	}

	r.migrationStage(ctx, pl.MigrationID, "running", "starting")
	if err := r.startServerOnNode(ctx, tenantID, pl.ServerID, pl.ToNodeID, srv); err != nil {
		return fmt.Errorf("запуск на новой ноде: %w", err)
	}

	r.migrationStage(ctx, pl.MigrationID, "running", "cleanup")
	r.cleanupSource(ctx, srcCfg, pl.ServerID, pl.RemoveSource)

	_, _ = r.db.Exec(ctx, `
		UPDATE core.server_migrations
		SET status = 'completed', stage = 'done', finished_at = now()
		WHERE id = $1
	`, nullableUUIDText(pl.MigrationID))
	log.Printf("migrate: сервер %s перенесён на ноду %s (%d байт)", pl.ServerID, pl.ToNodeID, bytes)
	r.notifyServerMigrated(ctx, tenantID, pl.ServerID, srv.Name, pl.ToNodeID)
	return nil
}

// notifyServerMigrated сообщает владельцу о переезде сервера.
//
// Событие server.migrated было объявлено, но не отправлялось: у сервера
// менялись нода и адрес, а владелец узнавал об этом, только когда старый адрес
// переставал отвечать.
func (r *Runner) notifyServerMigrated(ctx context.Context, tenantID, serverID, serverName, toNodeID string) {
	var nodeName, ip string
	_ = r.db.QueryRow(ctx, `
		SELECT COALESCE(n.fqdn, ''), COALESCE(s.ip_address, '')
		FROM core.servers s
		LEFT JOIN core.nodes n ON n.id = $3::uuid
		WHERE s.id = $1 AND s.tenant_id = $2
	`, serverID, tenantID, toNodeID).Scan(&nodeName, &ip)

	if serverName == "" {
		serverName = "сервер"
	}
	body := "Сервер «" + serverName + "» перенесён на другую ноду"
	if nodeName != "" {
		body += " (" + nodeName + ")"
	}
	body += "."
	if ip != "" {
		body += " Новый адрес подключения: " + ip + "."
	}

	rec, err := notify.LoadServerOwner(ctx, r.db, tenantID, serverID)
	if err != nil {
		log.Printf("оповещение о переносе: не найден владелец сервера %s: %v", serverID, err)
		return
	}
	if _, err := notify.Dispatch(ctx, r.db, tenantID, rec, notify.Event{
		Kind:   notify.KindServerMigrated,
		Title:  "Сервер переехал",
		Body:   body,
		Action: r.serverAction("Открыть сервер", serverID, ""),
		Meta:   map[string]any{"server_id": serverID, "node_id": toNodeID},
	}); err != nil {
		log.Printf("оповещение о переносе сервера %s: %v", serverID, err)
	}
}

func (r *Runner) stopServerForMigration(ctx context.Context, nodeID, serverID string, ssh sshclient.Config) error {
	_ = r.relay.SendCommand(ctx, nodeID, relay.CommandRequest{
		CommandID: uuid.NewString(),
		Action:    "power",
		ServerID:  serverID,
		Payload:   map[string]any{"power_action": "stop"},
	})

	cname := containerNameFor(serverID)
	deadline := time.Now().Add(migrateStopWait)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
		}
		out, err := sshclient.RunCapture(ssh, fmt.Sprintf(
			"docker inspect -f '{{.State.Running}}' %s 2>/dev/null || echo false", shellQuote(cname)))
		if err != nil {
			continue
		}
		if !strings.Contains(out, "true") {
			return nil
		}
	}

	if _, err := sshclient.RunCapture(ssh, fmt.Sprintf("docker stop -t 30 %s", shellQuote(cname))); err != nil {
		return fmt.Errorf("не удалось остановить сервер на исходной ноде: %w", err)
	}
	return nil
}

func (r *Runner) switchServerNode(ctx context.Context, tenantID, serverID, toNodeID, gameID string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var fqdn string
	if err := tx.QueryRow(ctx, `SELECT COALESCE(fqdn, '') FROM core.nodes WHERE id = $1 AND tenant_id = $2`,
		toNodeID, tenantID).Scan(&fqdn); err != nil {
		return fmt.Errorf("нода назначения не найдена")
	}
	if _, err := tx.Exec(ctx, `
		UPDATE core.servers
		SET node_id = $3, ip_address = $4, primary_port = NULL,
		    provisioning_status = 'provisioning', provisioning_error = NULL
		WHERE id = $1 AND tenant_id = $2
	`, serverID, tenantID, toNodeID, fqdn); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM core.server_ports WHERE server_id = $1 AND tenant_id = $2`,
		serverID, tenantID); err != nil {
		return err
	}
	if gameID != "" && gameID != "test" {
		if _, err := portalloc.Assign(ctx, tx, tenantID, toNodeID, serverID, gameID); err != nil {
			return fmt.Errorf("выдача порта на новой ноде: %w", err)
		}
	}
	return tx.Commit(ctx)
}

func (r *Runner) startServerOnNode(ctx context.Context, tenantID, serverID, nodeID string, srv migrateServer) error {
	var primaryPort int
	var startupParams string
	_ = r.db.QueryRow(ctx, `
		SELECT COALESCE(primary_port, 0), COALESCE(config->>'startup_params', '')
		FROM core.servers WHERE id = $1
	`, serverID).Scan(&primaryPort, &startupParams)

	// На новой ноде каталога .vtx может не быть: перенос копирует данные игры, а
	// параметры запуска — состояние панели, и источник для них один, база.
	cmdPayload := map[string]any{
		"power_action":   "start",
		"name":           srv.Name,
		"game_id":        srv.GameID,
		"limits":         srv.Limits,
		"startup_params": startupParams,
	}
	if primaryPort > 0 {
		cmdPayload["primary_port"] = primaryPort
	}
	if bindIP := r.serverBindIP(ctx, tenantID, serverID); bindIP != "" {
		cmdPayload["bind_ip"] = bindIP
	}
	if img := r.resolveDockerImage(ctx, tenantID, serverID); img != "" {
		cmdPayload["docker_image"] = img
	}
	if err := r.relay.SendCommand(ctx, nodeID, relay.CommandRequest{
		CommandID: uuid.NewString(),
		Action:    "power",
		ServerID:  serverID,
		Payload:   cmdPayload,
	}); err != nil {
		return err
	}
	_, _ = r.db.Exec(ctx, `
		UPDATE core.servers SET status = 'starting', provisioning_status = 'ready' WHERE id = $1 AND tenant_id = $2
	`, serverID, tenantID)
	return nil
}

func (r *Runner) cleanupSource(ctx context.Context, ssh sshclient.Config, serverID string, removeData bool) {
	cname := containerNameFor(serverID)
	if _, err := sshclient.RunCapture(ssh, fmt.Sprintf("docker rm -f %s 2>/dev/null || true", shellQuote(cname))); err != nil {
		log.Printf("migrate: не удалось удалить контейнер %s на исходной ноде: %v", cname, err)
	}
	if !removeData {
		return
	}
	dir := serverDirOnNode(serverID)
	if _, err := sshclient.RunCapture(ssh, fmt.Sprintf("rm -rf %s", shellQuote(dir))); err != nil {
		log.Printf("migrate: не удалось удалить данные %s на исходной ноде: %v", dir, err)
	}
}

func (r *Runner) restoreAfterFailure(ctx context.Context, tenantID, serverID string, srv migrateServer, nodeID string) {
	_, _ = r.db.Exec(ctx, `
		UPDATE core.servers SET provisioning_status = 'ready' WHERE id = $1 AND tenant_id = $2
	`, serverID, tenantID)
	if err := r.startServerOnNode(ctx, tenantID, serverID, nodeID, srv); err != nil {
		log.Printf("migrate: не удалось вернуть сервер %s на ноду %s: %v", serverID, nodeID, err)
	}
}

func (r *Runner) failMigration(ctx context.Context, tenantID string, pl migratePayload, msg string) {
	_, _ = r.db.Exec(ctx, `
		UPDATE core.server_migrations
		SET status = 'failed', error = $2, finished_at = now()
		WHERE id = $1
	`, nullableUUIDText(pl.MigrationID), msg)
	_, _ = r.db.Exec(ctx, `
		UPDATE core.servers SET provisioning_status = 'ready'
		WHERE id = $1 AND tenant_id = $2 AND provisioning_status = 'migrating'
	`, pl.ServerID, tenantID)
}

func (r *Runner) migrationStage(ctx context.Context, migrationID, status, stage string) {
	_, _ = r.db.Exec(ctx, `
		UPDATE core.server_migrations
		SET status = $2, stage = $3,
		    started_at = COALESCE(started_at, now())
		WHERE id = $1
	`, nullableUUIDText(migrationID), status, stage)
}

func (r *Runner) migrationBytes(ctx context.Context, migrationID string, total int64) {
	_, _ = r.db.Exec(ctx, `UPDATE core.server_migrations SET bytes = $2 WHERE id = $1`,
		nullableUUIDText(migrationID), total)
}

func sshConfigFor(n *nodeSSH, execTimeout time.Duration) sshclient.Config {
	return sshclient.Config{
		Host:        n.SSHHost,
		Port:        n.SSHPort,
		User:        n.SSHUser,
		Password:    n.SSHPassword,
		Timeout:     60 * time.Second,
		ExecTimeout: execTimeout,
	}
}

func serverDirOnNode(serverID string) string {
	return nodeServersDir + "/" + serverID
}

func containerNameFor(serverID string) string {
	return "vortanix-" + strings.ReplaceAll(serverID, "-", "")
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}

func nullableUUIDText(v string) any {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	return v
}
