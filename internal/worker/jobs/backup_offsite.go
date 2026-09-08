package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"

	"github.com/vortanix/vortanix/internal/worker/relay"
	"github.com/vortanix/vortanix/pkg/sshclient"
)

const (
	backupTransferTimeout = 3 * time.Hour
	nodeBackupsSubdir     = "backups"
)

func (r *Runner) BackupOffsiteLoop(ctx context.Context, wake <-chan struct{}) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-wake:
			r.heartbeat.Beat(LoopBackupOffsite)
			r.drainOffsite(ctx)
		case <-ticker.C:
			r.heartbeat.Beat(LoopBackupOffsite)
			r.drainOffsite(ctx)
		}
	}
}

func (r *Runner) drainOffsite(ctx context.Context) {
	for r.processOffsiteOne(ctx) {
	}
}

type offsitePayload struct {
	BackupID  string `json:"backup_id"`
	ServerID  string `json:"server_id"`
	Filename  string `json:"filename"`
	KeepCount int    `json:"keep_count"`
	Restore   bool   `json:"restore"`
}

func (r *Runner) processOffsiteOne(ctx context.Context) bool {
	var jobID, tenantID, jobType string
	var payload []byte
	err := r.db.QueryRow(ctx, `
		UPDATE core.jobs SET status = 'running', attempts = attempts + 1
		WHERE id = (
			SELECT id FROM core.jobs
			WHERE type IN ('backup_upload', 'backup_fetch') AND status = 'pending' AND attempts < 5
			ORDER BY created_at ASC
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		)
		RETURNING id::text, tenant_id::text, type, payload
	`).Scan(&jobID, &tenantID, &jobType, &payload)
	if err != nil {
		return false
	}

	var pl offsitePayload
	_ = json.Unmarshal(payload, &pl)
	if pl.ServerID == "" || pl.Filename == "" {
		r.failJobDirect(ctx, jobID, "missing server_id or filename in payload")
		return true
	}

	beatCtx, stopBeat := context.WithCancel(ctx)
	go func() {
		t := time.NewTicker(30 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-beatCtx.Done():
				return
			case <-t.C:
				r.heartbeat.Beat(LoopBackupOffsite)
			}
		}
	}()
	defer stopBeat()

	var runErr error
	if jobType == "backup_upload" {
		runErr = r.uploadBackup(ctx, tenantID, pl)
	} else {
		runErr = r.fetchBackup(ctx, tenantID, pl)
	}
	if runErr != nil {
		if pl.BackupID != "" {
			_, _ = r.db.Exec(ctx, `
				UPDATE core.server_backups SET remote_status = 'failed', remote_error = $2 WHERE id = $1
			`, pl.BackupID, runErr.Error())
		}
		r.failJobDirect(ctx, jobID, runErr.Error())
		return true
	}
	_, _ = r.db.Exec(ctx, `
		UPDATE core.jobs SET status = 'completed', result = '{"ok":true}'::jsonb WHERE id = $1
	`, jobID)
	return true
}

type s3Settings struct {
	Key          string
	Secret       string
	Region       string
	Bucket       string
	Endpoint     string
	UsePathStyle bool
}

func (r *Runner) s3SettingsFor(ctx context.Context, tenantID string) (s3Settings, error) {
	cfg := s3Settings{
		Key:      r.tenantSettingString(ctx, tenantID, "files.storage.s3.key"),
		Secret:   r.tenantSettingString(ctx, tenantID, "files.storage.s3.secret"),
		Region:   r.tenantSettingString(ctx, tenantID, "files.storage.s3.region"),
		Bucket:   r.tenantSettingString(ctx, tenantID, "files.storage.s3.bucket"),
		Endpoint: r.tenantSettingString(ctx, tenantID, "files.storage.s3.endpoint"),
	}
	cfg.Secret = r.secrets.MustDecrypt(cfg.Secret)
	cfg.UsePathStyle = truthySetting(r.tenantSettingString(ctx, tenantID, "files.storage.s3.use_path_style_endpoint"))
	if cfg.Bucket == "" || cfg.Key == "" || cfg.Secret == "" {
		return cfg, fmt.Errorf("хранилище S3 не настроено: заполните «Настройки → Файлы»")
	}
	if cfg.Region == "" {
		cfg.Region = "us-east-1"
	}
	return cfg, nil
}

func truthySetting(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

func (c s3Settings) client() *s3.Client {
	awsCfg := aws.Config{
		Region:      c.Region,
		Credentials: credentials.NewStaticCredentialsProvider(c.Key, c.Secret, ""),
	}
	if c.Endpoint != "" {
		endpoint := c.Endpoint
		if !strings.HasPrefix(endpoint, "http") {
			endpoint = "https://" + endpoint
		}
		awsCfg.BaseEndpoint = aws.String(endpoint)
	}
	return s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = c.UsePathStyle
	})
}

func backupObjectKey(tenantID, serverID, filename string) string {
	return "backups/" + tenantID + "/" + serverID + "/" + filename
}

func (r *Runner) uploadBackup(ctx context.Context, tenantID string, pl offsitePayload) error {
	cfg, err := r.s3SettingsFor(ctx, tenantID)
	if err != nil {
		return err
	}
	node, dir, err := r.serverNodeSSH(ctx, tenantID, pl.ServerID)
	if err != nil {
		return err
	}

	if pl.BackupID != "" {
		_, _ = r.db.Exec(ctx, `
			UPDATE core.server_backups SET remote_status = 'uploading', remote_error = NULL WHERE id = $1
		`, pl.BackupID)
	}

	path := dir + "/" + nodeBackupsSubdir + "/" + pl.Filename
	key := backupObjectKey(tenantID, pl.ServerID, pl.Filename)

	pr, pw := io.Pipe()
	uploader := manager.NewUploader(cfg.client(), func(u *manager.Uploader) {
		u.PartSize = 16 << 20
		u.Concurrency = 3
	})

	type sshResult struct {
		bytes int64
		err   error
	}
	done := make(chan sshResult, 1)
	go func() {
		sshCfg := sshConfigFor(node, backupTransferTimeout)
		n, err := sshclient.StreamFrom(sshCfg,
			fmt.Sprintf("test -f %s && cat %s", shellQuote(path), shellQuote(path)), pw)
		_ = pw.CloseWithError(err)
		done <- sshResult{bytes: n, err: err}
	}()

	uploadCtx, cancel := context.WithTimeout(ctx, backupTransferTimeout)
	defer cancel()
	_, upErr := uploader.Upload(uploadCtx, &s3.PutObjectInput{
		Bucket: aws.String(cfg.Bucket),
		Key:    aws.String(key),
		Body:   pr,
	})
	res := <-done
	if res.err != nil {
		return fmt.Errorf("чтение копии с ноды: %w", res.err)
	}
	if upErr != nil {
		return fmt.Errorf("загрузка в S3: %w", upErr)
	}
	if res.bytes == 0 {
		return fmt.Errorf("копия %s на ноде пуста или отсутствует", pl.Filename)
	}

	if pl.BackupID != "" {
		_, _ = r.db.Exec(ctx, `
			UPDATE core.server_backups
			SET remote_status = 'uploaded', remote_key = $2, remote_size = $3,
			    uploaded_at = now(), remote_error = NULL
			WHERE id = $1
		`, pl.BackupID, key, res.bytes)
	}
	log.Printf("backup offsite: %s → s3://%s/%s (%d байт)", pl.Filename, cfg.Bucket, key, res.bytes)

	r.pruneRemoteBackups(ctx, tenantID, pl.ServerID, pl.KeepCount, cfg)
	return nil
}

func (r *Runner) pruneRemoteBackups(ctx context.Context, tenantID, serverID string, keep int, cfg s3Settings) {
	if keep <= 0 {
		keep = 7
	}
	rows, err := r.db.Query(ctx, `
		SELECT id::text, remote_key FROM core.server_backups
		WHERE server_id = $1 AND remote_status = 'uploaded' AND remote_key IS NOT NULL
		ORDER BY COALESCE(uploaded_at, created_at) DESC
		OFFSET $2
	`, serverID, keep)
	if err != nil {
		return
	}
	type stale struct{ id, key string }
	list := []stale{}
	for rows.Next() {
		var s stale
		if rows.Scan(&s.id, &s.key) == nil {
			list = append(list, s)
		}
	}
	rows.Close()

	client := cfg.client()
	for _, s := range list {
		if _, err := client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(cfg.Bucket),
			Key:    aws.String(s.key),
		}); err != nil {
			log.Printf("backup offsite: не удалось удалить %s: %v", s.key, err)
			continue
		}
		_, _ = r.db.Exec(ctx, `
			UPDATE core.server_backups SET remote_status = 'deleted', remote_key = NULL WHERE id = $1
		`, s.id)
	}
}

func (r *Runner) fetchBackup(ctx context.Context, tenantID string, pl offsitePayload) error {
	cfg, err := r.s3SettingsFor(ctx, tenantID)
	if err != nil {
		return err
	}
	node, dir, err := r.serverNodeSSH(ctx, tenantID, pl.ServerID)
	if err != nil {
		return err
	}

	key := backupObjectKey(tenantID, pl.ServerID, pl.Filename)
	if pl.BackupID != "" {
		var stored *string
		if r.db.QueryRow(ctx, `SELECT remote_key FROM core.server_backups WHERE id = $1`, pl.BackupID).
			Scan(&stored) == nil && stored != nil && *stored != "" {
			key = *stored
		}
	}

	obj, err := cfg.client().GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(cfg.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("чтение из S3: %w", err)
	}
	defer obj.Body.Close()

	backupsDir := dir + "/" + nodeBackupsSubdir
	path := backupsDir + "/" + pl.Filename
	cmd := fmt.Sprintf("mkdir -p %s && cat > %s", shellQuote(backupsDir), shellQuote(path))
	written, err := sshclient.StreamTo(sshConfigFor(node, backupTransferTimeout), cmd, obj.Body)
	if err != nil {
		return fmt.Errorf("запись копии на ноду: %w", err)
	}
	log.Printf("backup offsite: s3://%s/%s → нода (%d байт)", cfg.Bucket, key, written)

	if !pl.Restore {
		return nil
	}
	var nodeID string
	if err := r.db.QueryRow(ctx, `SELECT node_id::text FROM core.servers WHERE id = $1`, pl.ServerID).
		Scan(&nodeID); err != nil {
		return fmt.Errorf("сервер не найден")
	}
	resp, cmdErr := r.relay.CommandSync(ctx, nodeID, relay.CommandRequest{
		CommandID: uuid.NewString(),
		Action:    "backup_restore",
		ServerID:  pl.ServerID,
		Payload:   map[string]any{"name": pl.Filename},
	})
	if cmdErr != nil {
		return fmt.Errorf("восстановление на ноде: %w", cmdErr)
	}
	if resp == nil || !resp.OK {
		msg := "восстановление не удалось"
		if resp != nil && resp.Result != nil {
			if e, _ := resp.Result["error"].(string); e != "" {
				msg = e
			}
		}
		return fmt.Errorf("%s", msg)
	}
	return nil
}

func (r *Runner) serverNodeSSH(ctx context.Context, tenantID, serverID string) (*nodeSSH, string, error) {
	var nodeID string
	if err := r.db.QueryRow(ctx, `
		SELECT node_id::text FROM core.servers WHERE id = $1 AND tenant_id = $2
	`, serverID, tenantID).Scan(&nodeID); err != nil {
		return nil, "", fmt.Errorf("сервер не найден")
	}
	node, err := r.loadNodeSSH(ctx, r.db, tenantID, nodeID)
	if err != nil {
		return nil, "", fmt.Errorf("нода сервера: %w", err)
	}
	return node, serverDirOnNode(serverID), nil
}

func (r *Runner) EnqueueBackupUpload(ctx context.Context, tenantID, serverID, backupID, filename string, keep int) {
	payload, _ := json.Marshal(map[string]any{
		"backup_id": backupID, "server_id": serverID,
		"filename": filename, "keep_count": keep,
	})
	_, _ = r.db.Exec(ctx, `
		INSERT INTO core.jobs (tenant_id, type, status, payload)
		VALUES ($1, 'backup_upload', 'pending', $2::jsonb)
	`, tenantID, payload)
	if backupID != "" {
		_, _ = r.db.Exec(ctx, `
			UPDATE core.server_backups SET remote_status = 'pending' WHERE id = $1
		`, backupID)
	}
}

func (r *Runner) remoteBackupsEnabled(ctx context.Context, tenantID string) bool {
	_, err := r.s3SettingsFor(ctx, tenantID)
	return err == nil
}

func (r *Runner) scheduleKeepCount(ctx context.Context, serverID string) int {
	var keep int
	if r.db.QueryRow(ctx, `
		SELECT COALESCE(keep_count, 0) FROM core.server_backup_schedules WHERE server_id = $1
	`, serverID).Scan(&keep) != nil {
		return 0
	}
	return keep
}
