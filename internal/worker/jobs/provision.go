package jobs

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/vortanixapp/panel/pkg/gamecatalog"
	"github.com/vortanixapp/panel/pkg/portalloc"
	"github.com/vortanixapp/panel/pkg/secretbox"

	"github.com/vortanixapp/panel/internal/worker/mail"
	"github.com/vortanixapp/panel/internal/worker/relay"
)

type Runner struct {
	db        *pgxpool.Pool
	relay     *relay.Client
	mail      mail.Config
	secrets   *secretbox.Box
	heartbeat *Heartbeat

	telegramBotToken string
	panelURL         string
}

func New(db *pgxpool.Pool, relayClient *relay.Client, mailCfg mail.Config, secrets *secretbox.Box) *Runner {
	return &Runner{db: db, relay: relayClient, mail: mailCfg, secrets: secrets, heartbeat: newHeartbeat()}
}

func (r *Runner) WithNotify(telegramBotToken, panelURL string) *Runner {
	r.telegramBotToken = telegramBotToken
	r.panelURL = panelURL
	return r
}

func (r *Runner) Loop(ctx context.Context, wake <-chan struct{}) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-wake:
			r.heartbeat.Beat(LoopProvision)
			r.drainProvision(ctx)
		case <-ticker.C:
			r.heartbeat.Beat(LoopProvision)
			r.drainProvision(ctx)
		}
	}
}

func (r *Runner) drainProvision(ctx context.Context) {
	for r.processOne(ctx) {
	}
}

func (r *Runner) processOne(ctx context.Context) bool {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return false
	}
	defer tx.Rollback(ctx)

	var jobID string
	var payload []byte
	err = tx.QueryRow(ctx, `
		SELECT id::text, payload
		FROM core.jobs
		WHERE type = 'provision_server' AND status = 'pending' AND attempts < 5
		ORDER BY created_at ASC
		LIMIT 1
		FOR UPDATE SKIP LOCKED
	`).Scan(&jobID, &payload)
	if err != nil {
		return false
	}

	_, _ = tx.Exec(ctx, `UPDATE core.jobs SET status = 'running', attempts = attempts + 1 WHERE id = $1`, jobID)

	var pl struct {
		ServerID string `json:"server_id"`
	}
	_ = json.Unmarshal(payload, &pl)
	if pl.ServerID == "" {
		r.failJob(ctx, tx, jobID, pl.ServerID, "missing server_id in payload")
		_ = tx.Commit(ctx)
		return true
	}

	var nodeID, name, gameID, provStatus, status string
	var limits []byte
	err = tx.QueryRow(ctx, `
		SELECT node_id::text, name, game_id, provisioning_status, status, limits
		FROM core.servers WHERE id = $1
	`, pl.ServerID).Scan(&nodeID, &name, &gameID, &provStatus, &status, &limits)
	if err != nil {
		r.failJob(ctx, tx, jobID, pl.ServerID, "server not found")
		_ = tx.Commit(ctx)
		return true
	}

	if provStatus == "ready" && (status == "running" || status == "stopped") {
		r.completeJob(ctx, tx, jobID)
		_ = tx.Commit(ctx)
		return true
	}

	var lim map[string]any
	_ = json.Unmarshal(limits, &lim)
	dockerImage := r.resolveDockerImage(ctx, pl.ServerID)
	r.assignServerNetwork(ctx, tx, pl.ServerID, nodeID, gameID)
	var primaryPort int
	var startupParams string
	_ = tx.QueryRow(ctx, `
		SELECT COALESCE(primary_port, 0), COALESCE(config->>'startup_params', '')
		FROM core.servers WHERE id = $1
	`, pl.ServerID).Scan(&primaryPort, &startupParams)
	cmdPayload := map[string]any{
		"power_action":   "start",
		"name":           name,
		"game_id":        gameID,
		"limits":         lim,
		"startup_params": startupParams,
	}
	if primaryPort > 0 {
		cmdPayload["primary_port"] = primaryPort
	}
	if bindIP := r.serverBindIP(ctx, pl.ServerID); bindIP != "" {
		cmdPayload["bind_ip"] = bindIP
	}
	if dockerImage != "" {
		cmdPayload["docker_image"] = dockerImage
	}
	if spec := r.resolveInstallSpec(ctx, pl.ServerID); spec != nil {
		cmdPayload["install"] = spec
	}
	cmdErr := r.relay.SendCommand(ctx, nodeID, relay.CommandRequest{
		CommandID: uuid.NewString(),
		Action:    "power",
		ServerID:  pl.ServerID,
		Payload:   cmdPayload,
	})

	if cmdErr != nil {
		log.Printf("provision job %s: relay error: %v", jobID, cmdErr)
		r.failJob(ctx, tx, jobID, pl.ServerID, cmdErr.Error())
		_ = tx.Commit(ctx)
		return true
	}

	_, _ = tx.Exec(ctx, `
		UPDATE core.servers SET status = 'starting', provisioning_status = 'provisioning', provisioning_error = NULL WHERE id = $1
	`, pl.ServerID)
	r.completeJob(ctx, tx, jobID)
	_ = tx.Commit(ctx)
	return true
}

func (r *Runner) completeJob(ctx context.Context, tx pgx.Tx, jobID string) {
	_, _ = tx.Exec(ctx, `
		UPDATE core.jobs SET status = 'completed', result = '{"ok":true}'::jsonb WHERE id = $1
	`, jobID)
}

func (r *Runner) failJob(ctx context.Context, tx pgx.Tx, jobID, serverID, msg string) {
	result, _ := json.Marshal(map[string]string{"error": msg})
	_, _ = tx.Exec(ctx, `UPDATE core.jobs SET status = 'failed', result = $2::jsonb WHERE id = $1`, jobID, result)
	r.EmitWebhook(ctx, "job.failed", map[string]any{
		"job_id": jobID, "type": "provision_server",
		"server_id": serverID, "error": msg,
	})
	if serverID != "" {
		_, _ = tx.Exec(ctx, `
			UPDATE core.servers SET provisioning_status = 'failed', provisioning_error = $2 WHERE id = $1
		`, serverID, msg)
	}
}

func (r *Runner) assignServerNetwork(ctx context.Context, tx pgx.Tx, serverID, nodeID, gameID string) {
	var fqdn string
	if err := tx.QueryRow(ctx, `SELECT COALESCE(fqdn, '') FROM core.nodes WHERE id = $1`, nodeID).Scan(&fqdn); err != nil || fqdn == "" {
		return
	}
	_, _ = tx.Exec(ctx, `
		UPDATE core.servers SET ip_address = $2 WHERE id = $1 AND (ip_address IS NULL OR ip_address = '')
	`, serverID, fqdn)

	if gameID == "test" || gameID == "" {
		return
	}
	if _, err := portalloc.Assign(ctx, tx, nodeID, serverID, gameID); err != nil {
		log.Printf("provision: не удалось выдать порт серверу %s (%s): %v", serverID, gameID, err)
	}
}

func (r *Runner) resolveDockerImage(ctx context.Context, serverID string) string {
	var gameSlug, img string
	err := r.db.QueryRow(ctx, `
		SELECT s.game_id, COALESCE(gv.docker_image, '')
		FROM core.servers s
		LEFT JOIN core.game_versions gv ON gv.id = s.game_version_id
		WHERE s.id = $1
	`, serverID).Scan(&gameSlug, &img)
	if err != nil {
		return ""
	}
	if img == "" {
		_ = r.db.QueryRow(ctx, `
			SELECT COALESCE(gv.docker_image, '')
			FROM core.servers s
			JOIN core.games g ON g.slug = s.game_id
			JOIN core.game_versions gv ON gv.game_id = g.id AND gv.active = true
			WHERE s.id = $1
			ORDER BY gv.sort_order ASC, gv.created_at DESC
			LIMIT 1
		`, serverID).Scan(&img)
	}
	key := gamecatalog.Normalize(gameSlug)
	if _, known := gamecatalog.Resolve(key); !known {
		return ""
	}
	tag := gamecatalog.DefaultTag(key)
	if img != "" && gamecatalog.BelongsTo(key, img) {
		tag = gamecatalog.TagOf(img)
	}
	return gamecatalog.ImageWithTag(key, tag)
}

func (r *Runner) resolveInstallSpec(ctx context.Context, serverID string) map[string]any {
	var sourceType, archiveURL, steamBranch, steamModConfig string
	var steamAppID *int64
	err := r.db.QueryRow(ctx, `
		SELECT COALESCE(gv.source_type, ''), COALESCE(gv.archive_url, ''),
		       gv.steam_app_id, COALESCE(gv.steam_branch, ''), COALESCE(gv.steam_mod_config, '')
		FROM core.servers s
		JOIN core.game_versions gv ON gv.id = s.game_version_id
		WHERE s.id = $1
	`, serverID).Scan(&sourceType, &archiveURL, &steamAppID, &steamBranch, &steamModConfig)
	if err != nil {
		err = r.db.QueryRow(ctx, `
			SELECT COALESCE(gv.source_type, ''), COALESCE(gv.archive_url, ''),
			       gv.steam_app_id, COALESCE(gv.steam_branch, ''), COALESCE(gv.steam_mod_config, '')
			FROM core.servers s
			JOIN core.games g ON g.slug = s.game_id
			JOIN core.game_versions gv ON gv.game_id = g.id AND gv.active = true
			WHERE s.id = $1
			ORDER BY gv.sort_order ASC, gv.created_at DESC
			LIMIT 1
		`, serverID).Scan(&sourceType, &archiveURL, &steamAppID, &steamBranch, &steamModConfig)
		if err != nil {
			return nil
		}
	}
	spec := map[string]any{"source_type": sourceType}
	switch {
	case archiveURL != "":
		spec["archive_url"] = archiveURL
	case steamAppID != nil && *steamAppID > 0:
		spec["steam_app_id"] = *steamAppID
		if steamBranch != "" {
			spec["steam_branch"] = steamBranch
		}
		if steamModConfig != "" {
			spec["steam_mod_config"] = steamModConfig
		}
	default:
		return nil
	}
	return spec
}

func (r *Runner) serverBindIP(ctx context.Context, serverID string) string {
	var address string
	if r.db.QueryRow(ctx, `
		SELECT host(address) FROM core.ip_pools
		WHERE server_id = $1::uuid
		LIMIT 1
	`, serverID).Scan(&address) != nil {
		return ""
	}
	return address
}
