package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/go-chi/chi/v5"

	"github.com/vortanix/vortanix/internal/api/jobwake"
)

func (h *Handler) remoteBackupsConfigured(ctx context.Context, tenantID string) bool {
	bucket := h.tenantSettingString(ctx, tenantID, "files.storage.s3.bucket")
	key := h.tenantSettingString(ctx, tenantID, "files.storage.s3.key")
	return strings.TrimSpace(bucket) != "" && strings.TrimSpace(key) != ""
}

func (h *Handler) enqueueBackupUpload(ctx context.Context, tenantID, serverID, backupID, filename string) {
	if !h.remoteBackupsConfigured(ctx, tenantID) {
		return
	}
	var keep int
	_ = h.dbOf(ctx).QueryRow(ctx, `
		SELECT COALESCE(keep_count, 0) FROM core.server_backup_schedules WHERE server_id = $1
	`, serverID).Scan(&keep)

	payload, _ := json.Marshal(map[string]any{
		"backup_id": backupID, "server_id": serverID,
		"filename": filename, "keep_count": keep,
	})
	if _, err := h.dbOf(ctx).Exec(ctx, `
		INSERT INTO core.jobs (tenant_id, type, status, payload)
		VALUES ($1, 'backup_upload', 'pending', $2::jsonb)
	`, tenantID, payload); err != nil {
		return
	}
	_, _ = h.dbOf(ctx).Exec(ctx, `UPDATE core.server_backups SET remote_status = 'pending' WHERE id = $1`, backupID)
	jobwake.Notify("backup_upload")
}

func (h *Handler) ServerRemoteBackupList(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	serverID := chi.URLParam(r, "id")
	if !h.requireServerTenant(w, r, claims.TenantID, serverID) {
		return
	}
	ctx := r.Context()

	rows, err := h.readerOf(ctx).Query(ctx, `
		SELECT id::text, COALESCE(filename, ''), COALESCE(size_bytes, 0),
		       remote_status, COALESCE(remote_error, ''), COALESCE(remote_size, 0),
		       COALESCE(source, 'manual'), created_at, uploaded_at
		FROM core.server_backups
		WHERE server_id = $1 AND remote_status <> 'none'
		ORDER BY created_at DESC
		LIMIT 100
	`, serverID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()

	list := []map[string]any{}
	for rows.Next() {
		var id, filename, status, errText, source string
		var size, remoteSize int64
		var createdAt time.Time
		var uploadedAt *time.Time
		if rows.Scan(&id, &filename, &size, &status, &errText, &remoteSize,
			&source, &createdAt, &uploadedAt) != nil {
			continue
		}
		list = append(list, map[string]any{
			"id": id, "filename": filename,
			"size_bytes": size, "remote_size": remoteSize,
			"status": status, "error": nilIfEmpty(errText),
			"source":      source,
			"created_at":  createdAt.Format(time.RFC3339),
			"uploaded_at": timeOrNil(uploadedAt),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"backups":   list,
		"available": h.remoteBackupsConfigured(ctx, claims.TenantID),
	})
}

type remoteBackupBody struct {
	BackupID string `json:"backup_id"`
	Restore  *bool  `json:"restore"`
}

func (h *Handler) ServerRemoteBackupRestore(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	serverID := chi.URLParam(r, "id")
	if !h.requireServerTenant(w, r, claims.TenantID, serverID) {
		return
	}
	var body remoteBackupBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.BackupID) == "" {
		writeError(w, http.StatusBadRequest, "backup_id required")
		return
	}
	ctx := r.Context()

	var filename, status string
	if h.dbOf(ctx).QueryRow(ctx, `
		SELECT COALESCE(filename, ''), remote_status FROM core.server_backups
		WHERE id = $1 AND server_id = $2
	`, body.BackupID, serverID).Scan(&filename, &status) != nil {
		writeError(w, http.StatusNotFound, "backup not found")
		return
	}
	if status != "uploaded" {
		writeError(w, http.StatusConflict, "копия не выгружена в S3 (статус "+status+")")
		return
	}

	restore := true
	if body.Restore != nil {
		restore = *body.Restore
	}
	payload, _ := json.Marshal(map[string]any{
		"backup_id": body.BackupID, "server_id": serverID,
		"filename": filename, "restore": restore,
	})
	if _, err := h.dbOf(ctx).Exec(ctx, `
		INSERT INTO core.jobs (tenant_id, type, status, payload)
		VALUES ($1, 'backup_fetch', 'pending', $2::jsonb)
	`, claims.TenantID, payload); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	jobwake.Notify("backup_fetch")
	audit(ctx, h.dbOf(ctx), claims.TenantID, claims.UserID, "server.backup.restore_remote", "server:"+serverID,
		map[string]any{"filename": filename, "restore": restore})
	writeJSON(w, http.StatusAccepted, map[string]any{"status": "queued", "restore": restore})
}

func (h *Handler) ServerRemoteBackupDelete(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	serverID := chi.URLParam(r, "id")
	if !h.requireServerTenant(w, r, claims.TenantID, serverID) {
		return
	}
	var body remoteBackupBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.BackupID) == "" {
		writeError(w, http.StatusBadRequest, "backup_id required")
		return
	}
	ctx := r.Context()

	var remoteKey *string
	if h.dbOf(ctx).QueryRow(ctx, `
		SELECT remote_key FROM core.server_backups WHERE id = $1 AND server_id = $2
	`, body.BackupID, serverID).Scan(&remoteKey) != nil {
		writeError(w, http.StatusNotFound, "backup not found")
		return
	}
	if remoteKey == nil || *remoteKey == "" {
		writeError(w, http.StatusConflict, "у копии нет объекта в S3")
		return
	}

	client, bucket, err := h.tenantS3Client(ctx, claims.TenantID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	delCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if _, err := client.DeleteObject(delCtx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(*remoteKey),
	}); err != nil {
		writeError(w, http.StatusBadGateway, "S3: "+err.Error())
		return
	}
	_, _ = h.dbOf(ctx).Exec(ctx, `
		UPDATE core.server_backups SET remote_status = 'deleted', remote_key = NULL WHERE id = $1
	`, body.BackupID)
	audit(ctx, h.dbOf(ctx), claims.TenantID, claims.UserID, "server.backup.delete_remote", "server:"+serverID,
		map[string]any{"key": *remoteKey})
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) tenantS3Client(ctx context.Context, tenantID string) (*s3.Client, string, error) {
	bucket := strings.TrimSpace(h.tenantSettingString(ctx, tenantID, "files.storage.s3.bucket"))
	key := strings.TrimSpace(h.tenantSettingString(ctx, tenantID, "files.storage.s3.key"))
	secret := strings.TrimSpace(h.tenantSettingString(ctx, tenantID, "files.storage.s3.secret"))
	region := strings.TrimSpace(h.tenantSettingString(ctx, tenantID, "files.storage.s3.region"))
	endpoint := strings.TrimSpace(h.tenantSettingString(ctx, tenantID, "files.storage.s3.endpoint"))
	pathStyle := formTruthy(h.tenantSettingString(ctx, tenantID, "files.storage.s3.use_path_style_endpoint"))
	if bucket == "" || key == "" || secret == "" {
		return nil, "", errS3NotConfigured
	}
	if region == "" {
		region = "us-east-1"
	}
	cfg := aws.Config{
		Region:      region,
		Credentials: credentials.NewStaticCredentialsProvider(key, secret, ""),
	}
	if endpoint != "" {
		if !strings.HasPrefix(endpoint, "http") {
			endpoint = "https://" + endpoint
		}
		cfg.BaseEndpoint = aws.String(endpoint)
	}
	return s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = pathStyle
	}), bucket, nil
}

var errS3NotConfigured = errS3{}

type errS3 struct{}

func (errS3) Error() string {
	return "хранилище S3 не настроено: заполните «Настройки → Файлы»"
}
