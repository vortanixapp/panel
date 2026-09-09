package jobs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/vortanixapp/panel/pkg/gamecatalog"
	"github.com/vortanixapp/panel/pkg/sshclient"
)

const setupProgressFlushInterval = 700 * time.Millisecond

const setupLogLimit = 256 * 1024

type dbExec interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func (r *Runner) NodeSetupLoop(ctx context.Context, wake <-chan struct{}) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-wake:
			r.heartbeat.Beat(LoopNodeSetup)
			r.drainNodeSetup(ctx)
		case <-ticker.C:
			r.heartbeat.Beat(LoopNodeSetup)
			r.drainNodeSetup(ctx)
		}
	}
}

func (r *Runner) drainNodeSetup(ctx context.Context) {
	for r.processNodeSetup(ctx) {
	}
}

func (r *Runner) claimNodeSetup(ctx context.Context) (jobID string, payload []byte, ok bool) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", nil, false
	}
	defer tx.Rollback(ctx)

	err = tx.QueryRow(ctx, `
		SELECT id::text, payload
		FROM core.jobs
		WHERE type = 'node_setup' AND status = 'pending' AND attempts < 5
		ORDER BY created_at ASC LIMIT 1
		FOR UPDATE SKIP LOCKED
	`).Scan(&jobID, &payload)
	if err != nil {
		return "", nil, false
	}
	if _, err := tx.Exec(ctx, `UPDATE core.jobs SET status = 'running', attempts = attempts + 1 WHERE id = $1`, jobID); err != nil {
		return "", nil, false
	}
	if err := tx.Commit(ctx); err != nil {
		return "", nil, false
	}
	return jobID, payload, true
}

func (r *Runner) processNodeSetup(ctx context.Context) bool {
	jobID, payload, ok := r.claimNodeSetup(ctx)
	if !ok {
		return false
	}

	var pl struct {
		NodeID    string `json:"node_id"`
		Component string `json:"component"`
	}
	_ = json.Unmarshal(payload, &pl)
	if pl.NodeID == "" || pl.Component == "" {
		r.failJobGeneric(ctx, r.db, jobID, "missing node_id or component")
		return true
	}

	node, err := r.loadNodeSSH(ctx, r.db, pl.NodeID)
	if err != nil {
		r.failJobGeneric(ctx, r.db, jobID, err.Error())
		r.markSetupFailed(ctx, r.db, pl.NodeID, pl.Component, err.Error())
		r.completeSetupProgress(ctx, pl.NodeID, "\n❌ "+err.Error()+"\n", pl.Component)
		return true
	}

	relayURL := relayWebsocketURL(envOr("RELAY_PUBLIC_URL", envOr("RELAY_URL", "")))

	progress := r.newSetupProgressWriter(ctx, pl.NodeID, pl.Component)
	defer progress.Close()
	fmt.Fprintf(progress, "=== %s ===\n", pl.Component)

	if pl.Component == "daemon" && relayUnusableForNode(relayURL, node.SSHHost) {
		msg := "у воркера не задан RELAY_PUBLIC_URL: агент на ноде получил бы петлевой адрес " +
			relayURL + " и стучался бы сам в себя. Укажите внешний адрес relay, например wss://panel.example.com/relay"
		r.failNodeSetup(ctx, jobID, pl.NodeID, pl.Component, msg, progress)
		return true
	}

	var images []string
	if pl.Component == "images" {
		images = r.gamesToBuild(ctx)
	}
	commands, cmdErr := setupCommands(pl.Component, node.Meta, node.AgentToken, node.ID, relayURL, images)
	if cmdErr != nil {
		r.failNodeSetup(ctx, jobID, pl.NodeID, pl.Component, cmdErr.Error(), progress)
		return true
	}
	hubUser, hubToken := r.dockerHubCreds(ctx)
	logins := append(registryLoginCommands(pl.Component, r.licenseKey(ctx)),
		dockerHubLoginCommands(pl.Component, hubUser, hubToken)...)
	commands = append(logins, commands...)

	cfg := sshclient.Config{
		Host:     node.SSHHost,
		Port:     node.SSHPort,
		User:     node.SSHUser,
		Password: node.SSHPassword,
		Timeout:  60 * time.Second,
	}
	if pl.Component == "images" {
		r.markImagesBuilding(ctx, pl.NodeID, images)
	}
	if err := sshclient.Run(cfg, commands, progress); err != nil {
		if pl.Component == "images" {
			r.markImagesResult(ctx, pl.NodeID, images, err.Error())
		}
		r.failNodeSetup(ctx, jobID, pl.NodeID, pl.Component, err.Error(), progress)
		return true
	}
	if pl.Component == "images" {
		r.markImagesResult(ctx, pl.NodeID, images, "")
	}

	fmt.Fprintf(progress, "\n✅ %s installed\n", pl.Component)
	fullLog := progress.finish()
	r.markSetupInstalled(ctx, r.db, pl.NodeID, pl.Component)

	if pl.Component == "phpmyadmin" {
		_, _ = r.db.Exec(ctx, `
			UPDATE core.nodes
			SET meta = jsonb_set(COALESCE(meta, '{}'::jsonb), '{phpmyadmin_port}', '8081'::jsonb, true)
			WHERE id = $1
		`, pl.NodeID)
	}

	result, _ := json.Marshal(map[string]any{"ok": true, "log": fullLog})
	_, _ = r.db.Exec(ctx, `UPDATE core.jobs SET status = 'completed', result = $2::jsonb WHERE id = $1`, jobID, result)
	return true
}

type nodeSSH struct {
	ID          string
	AgentToken  string
	SSHHost     string
	SSHUser     string
	SSHPort     int
	SSHPassword string
	Meta        map[string]any
}

func (r *Runner) loadNodeSSH(ctx context.Context, q dbExec, nodeID string) (*nodeSSH, error) {
	var n nodeSSH
	var sshHost, sshUser, sshPassEnc *string
	var meta []byte
	err := q.QueryRow(ctx, `
		SELECT id::text, agent_token, ssh_host, ssh_user, COALESCE(ssh_port, 22), ssh_password_enc, COALESCE(meta, '{}')
		FROM core.nodes WHERE id = $1
	`, nodeID).Scan(&n.ID, &n.AgentToken, &sshHost, &sshUser, &n.SSHPort, &sshPassEnc, &meta)
	if err != nil {
		return nil, fmt.Errorf("node not found")
	}
	if sshHost == nil || *sshHost == "" || sshUser == nil || *sshUser == "" {
		return nil, fmt.Errorf("ssh not configured on location")
	}
	n.SSHHost = *sshHost
	n.SSHUser = *sshUser
	n.AgentToken = r.secrets.MustDecrypt(n.AgentToken)
	n.SSHPassword = ""
	if sshPassEnc != nil {
		n.SSHPassword = r.secrets.MustDecrypt(*sshPassEnc)
	}
	_ = json.Unmarshal(meta, &n.Meta)
	if n.SSHPassword == "" {
		if p, ok := n.Meta["ssh_password"].(string); ok {
			n.SSHPassword = p
		}
	}
	if n.SSHPassword == "" {
		return nil, fmt.Errorf("ssh password not set")
	}
	return &n, nil
}

type setupProgressWriter struct {
	ctx       context.Context
	r         *Runner
	nodeID    string
	component string

	mu        sync.Mutex
	buf       bytes.Buffer
	dirty     bool
	lastFlush time.Time

	stop     chan struct{}
	stopOnce sync.Once
}

func (r *Runner) newSetupProgressWriter(ctx context.Context, nodeID, component string) *setupProgressWriter {
	w := &setupProgressWriter{
		ctx:       ctx,
		r:         r,
		nodeID:    nodeID,
		component: component,
		stop:      make(chan struct{}),
	}
	if prev := r.currentSetupLog(ctx, nodeID); prev != "" {
		w.buf.WriteString(prev)
		if !strings.HasSuffix(prev, "\n") {
			w.buf.WriteString("\n")
		}
	}
	go w.loop()
	return w
}

func (w *setupProgressWriter) loop() {
	ticker := time.NewTicker(setupProgressFlushInterval)
	defer ticker.Stop()
	for {
		select {
		case <-w.stop:
			return
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			w.flush(false)
		}
	}
}

func (w *setupProgressWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	w.buf.Write(p)
	w.trimLocked()
	w.dirty = true
	due := time.Since(w.lastFlush) >= setupProgressFlushInterval
	w.mu.Unlock()
	if due {
		w.flush(false)
	}
	return len(p), nil
}

func (w *setupProgressWriter) trimLocked() {
	if w.buf.Len() <= setupLogLimit {
		return
	}
	s := w.buf.String()
	s = s[len(s)-setupLogLimit:]
	if i := strings.IndexByte(s, '\n'); i >= 0 && i < len(s)-1 {
		s = s[i+1:]
	}
	w.buf.Reset()
	w.buf.WriteString("… лог обрезан …\n")
	w.buf.WriteString(s)
}

func (w *setupProgressWriter) flush(completed bool) {
	w.mu.Lock()
	if !w.dirty && !completed {
		w.mu.Unlock()
		return
	}
	snapshot := w.buf.String()
	w.dirty = false
	w.lastFlush = time.Now()
	w.mu.Unlock()
	w.r.writeSetupProgress(w.ctx, w.nodeID, snapshot, completed, w.component)
}

func (w *setupProgressWriter) finish() string {
	w.Close()
	w.flush(true)
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.String()
}

func (w *setupProgressWriter) Close() {
	w.stopOnce.Do(func() { close(w.stop) })
}

func (r *Runner) currentSetupLog(ctx context.Context, nodeID string) string {
	var text string
	err := r.db.QueryRow(ctx, `
		SELECT COALESCE(meta -> 'setup_progress' ->> 'log', '')
		FROM core.nodes WHERE id = $1
	`, nodeID).Scan(&text)
	if err != nil {
		return ""
	}
	return text
}

func (r *Runner) writeSetupProgress(ctx context.Context, nodeID, logText string, completed bool, component string) {
	payload, err := json.Marshal(map[string]any{
		"log":       logText,
		"completed": completed,
		"component": component,
	})
	if err != nil {
		return
	}
	_, _ = r.db.Exec(ctx, `
		UPDATE core.nodes
		SET meta = jsonb_set(COALESCE(meta, '{}'::jsonb), '{setup_progress}', $2::jsonb, true)
		WHERE id = $1
	`, nodeID, payload)
}

func (r *Runner) completeSetupProgress(ctx context.Context, nodeID, logText, component string) {
	prev := r.currentSetupLog(ctx, nodeID)
	if prev != "" && !strings.HasSuffix(prev, "\n") {
		prev += "\n"
	}
	r.writeSetupProgress(ctx, nodeID, prev+logText, true, component)
}

func metaSetupComponent(component string) bool {
	return component == "phpmyadmin" || component == "images"
}

func (r *Runner) setMetaSetupStatus(ctx context.Context, q dbExec, nodeID, component, status string) {
	_, _ = q.Exec(ctx, `
		UPDATE core.nodes
		SET meta = COALESCE(meta, '{}'::jsonb) || jsonb_build_object(
			'setup_statuses',
			COALESCE(meta -> 'setup_statuses', '{}'::jsonb) || jsonb_build_object($2::text, $3::text)
		)
		WHERE id = $1
	`, nodeID, component, status)
}

func (r *Runner) markSetupInstalled(ctx context.Context, q dbExec, nodeID, component string) {
	if metaSetupComponent(component) {
		r.setMetaSetupStatus(ctx, q, nodeID, component, "installed")
		return
	}
	if _, err := q.Exec(ctx, `
		INSERT INTO core.node_setups ( node_id, component, status, installed_at)
		VALUES ( $1, $2, 'installed', now())
		ON CONFLICT (node_id, component) DO UPDATE SET status = 'installed', installed_at = now(), error_message = NULL, updated_at = now()
	`, nodeID, component); err != nil {
		log.Printf("статус шага %s для ноды %s не записан: %v", component, nodeID, err)
	}
}

func (r *Runner) markSetupFailed(ctx context.Context, q dbExec, nodeID, component, errMsg string) {
	if metaSetupComponent(component) {
		r.setMetaSetupStatus(ctx, q, nodeID, component, "failed")
		return
	}
	if _, err := q.Exec(ctx, `
		INSERT INTO core.node_setups ( node_id, component, status, error_message)
		VALUES ( $1, $2, 'failed', $3)
		ON CONFLICT (node_id, component) DO UPDATE SET status = 'failed', error_message = $3, updated_at = now()
	`, nodeID, component, errMsg); err != nil {
		log.Printf("статус шага %s для ноды %s не записан: %v", component, nodeID, err)
	}
}

func (r *Runner) failNodeSetup(ctx context.Context, jobID, nodeID, component, errMsg string, progress *setupProgressWriter) {
	fmt.Fprintf(progress, "\n❌ %s\n", errMsg)
	progress.finish()
	r.markSetupFailed(ctx, r.db, nodeID, component, errMsg)
	result, _ := json.Marshal(map[string]string{"error": errMsg})
	_, _ = r.db.Exec(ctx, `UPDATE core.jobs SET status = 'failed', result = $2::jsonb WHERE id = $1`, jobID, result)
	log.Printf("node_setup failed: %s", errMsg)
}

func (r *Runner) failJobGeneric(ctx context.Context, q dbExec, jobID, msg string) {
	result, _ := json.Marshal(map[string]string{"error": msg})
	_, _ = q.Exec(ctx, `UPDATE core.jobs SET status = 'failed', result = $2::jsonb WHERE id = $1`, jobID, result)
}

func (r *Runner) tenantGameImages(ctx context.Context) []string {
	rows, err := r.db.Query(ctx, `
		SELECT g.slug, COALESCE(gv.docker_image, '')
		FROM core.games g
		LEFT JOIN core.game_versions gv
		       ON gv.game_id = g.id AND gv.active = true
		WHERE g.active = true
	`)
	if err != nil {
		return nil
	}
	defer rows.Close()

	seen := map[string]bool{}
	var out []string
	for rows.Next() {
		var slug, image string
		if rows.Scan(&slug, &image) != nil {
			continue
		}
		key := gamecatalog.Normalize(slug)
		if _, known := gamecatalog.Resolve(key); !known {
			continue
		}
		tag := gamecatalog.DefaultTag(key)
		if image != "" && gamecatalog.BelongsTo(key, image) {
			tag = gamecatalog.TagOf(image)
		}
		full := gamecatalog.ImageWithTag(key, tag)
		if full != "" && !seen[full] {
			seen[full] = true
			out = append(out, full)
		}
	}
	return out
}

func (r *Runner) claimJob(ctx context.Context, jobType string) (jobID string, payload []byte, ok bool) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", nil, false
	}
	defer tx.Rollback(ctx)

	err = tx.QueryRow(ctx, `
		SELECT id::text, payload
		FROM core.jobs
		WHERE type = $1 AND status = 'pending' AND attempts < 5
		ORDER BY created_at ASC LIMIT 1
		FOR UPDATE SKIP LOCKED
	`, jobType).Scan(&jobID, &payload)
	if err != nil {
		return "", nil, false
	}
	if _, err := tx.Exec(ctx, `UPDATE core.jobs SET status = 'running', attempts = attempts + 1 WHERE id = $1`, jobID); err != nil {
		return "", nil, false
	}
	if err := tx.Commit(ctx); err != nil {
		return "", nil, false
	}
	return jobID, payload, true
}

func (r *Runner) licenseKey(ctx context.Context) string {
	var key string
	if r.db.QueryRow(ctx, `SELECT COALESCE(license_key, '') FROM core.installation LIMIT 1`).Scan(&key) != nil {
		return ""
	}
	return strings.TrimSpace(key)
}

func (r *Runner) dockerHubCreds(ctx context.Context) (user, token string) {
	rows, err := r.db.Query(ctx, `
		SELECT key, value FROM core.tenant_settings
		WHERE key IN ('dockerhub.username', 'dockerhub.token')
	`)
	if err != nil {
		return "", ""
	}
	defer rows.Close()

	for rows.Next() {
		var key string
		var raw []byte
		if rows.Scan(&key, &raw) != nil {
			continue
		}
		var value string
		if json.Unmarshal(raw, &value) != nil {
			continue
		}
		switch key {
		case "dockerhub.username":
			user = strings.TrimSpace(value)
		case "dockerhub.token":
			token = strings.TrimSpace(r.secrets.MustDecrypt(value))
		}
	}
	return user, token
}

func (r *Runner) gamesToBuild(ctx context.Context) []string {
	rows, err := r.db.Query(ctx, `
		SELECT slug FROM core.games
		WHERE build_image = true
		ORDER BY slug
	`)
	if err != nil {
		return nil
	}
	defer rows.Close()

	seen := map[string]bool{}
	var out []string
	for rows.Next() {
		var slug string
		if rows.Scan(&slug) != nil {
			continue
		}
		key := gamecatalog.Normalize(slug)
		if _, known := gamecatalog.Resolve(key); !known || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, key)
	}
	return out
}
