package handlers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/vortanix/vortanix/internal/api/paneljwt"
)

func (h *Handler) SupportCreateForm(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	depts := defaultSupportDepartments()
	var raw []byte
	if err := h.dbOf(r.Context()).QueryRow(r.Context(), `
		SELECT value FROM core.tenant_settings WHERE tenant_id = $1 AND key = 'support.departments'
	`, claims.TenantID).Scan(&raw); err == nil && len(raw) > 0 {
		var custom []map[string]any
		if json.Unmarshal(raw, &custom) == nil && len(custom) > 0 {
			depts = custom
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"departments": depts,
		// Услуги клиента — чтобы обращение сразу привязывалось к серверу или
		// веб-аккаунту, а не выяснялось это первым ответом оператора.
		"services": h.userServices(r.Context(), claims.TenantID, claims.UserID),
		// Замеренное время первого ответа за неделю. Пусто, если отвечать
		// ещё не приходилось: обещать срок, которого не измеряли, нельзя.
		"sla": h.supportSLA(r.Context(), claims.TenantID),
	})
}

func defaultSupportDepartments() []map[string]any {
	return []map[string]any{
		{"id": "billing", "name": "Биллинг"},
		{"id": "technical", "name": "Техподдержка"},
		{"id": "hosting", "name": "Хостинг"},
		{"id": "other", "name": "Другое"},
	}
}

const (
	supportAttachmentMaxBytes = 8 << 20
	supportAttachmentDir      = "support"
	// Разумный предел на одну отправку: пять скриншотов по восемь мегабайт —
	// уже сорок, больше в одном сообщении смысла нет.
	supportAttachmentMaxCount = 5
)

var supportAttachmentTypes = map[string]string{
	".png":  "image/png",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".webp": "image/webp",
	".gif":  "image/gif",
	".txt":  "text/plain; charset=utf-8",
	".log":  "text/plain; charset=utf-8",
	".json": "application/json",
	".zip":  "application/zip",
	".pdf":  "application/pdf",
}

func (h *Handler) supportTicketAccess(r *http.Request, ticketID string) (*paneljwt.Claims, bool) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return nil, false
	}
	var ownerID string
	if h.dbOf(r.Context()).QueryRow(r.Context(), `
		SELECT user_id::text FROM core.support_tickets WHERE id = $1 AND tenant_id = $2
	`, ticketID, claims.TenantID).Scan(&ownerID) != nil {
		return nil, false
	}
	if !isStaffRole(claims.Role) && ownerID != claims.UserID {
		return nil, false
	}
	return claims, true
}

func (h *Handler) UploadSupportAttachment(w http.ResponseWriter, r *http.Request) {
	ticketID := chi.URLParam(r, "id")
	if ticketID == "" {
		writeError(w, http.StatusBadRequest, "ticket id required")
		return
	}
	claims, ok := h.supportTicketAccess(r, ticketID)
	if !ok {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, supportAttachmentMaxBytes+(1<<20))
	if err := r.ParseMultipartForm(supportAttachmentMaxBytes); err != nil {
		writeError(w, http.StatusBadRequest, "файл больше 8 МБ или повреждённая форма")
		return
	}
	// Файлов может быть несколько: раньше на сообщение приходилось ровно одно
	// вложение, потому что оно описывалось полем meta самого сообщения.
	headers := r.MultipartForm.File["file"]
	if len(headers) == 0 {
		headers = r.MultipartForm.File["files"]
	}
	if len(headers) == 0 {
		writeError(w, http.StatusBadRequest, "file required")
		return
	}
	if len(headers) > supportAttachmentMaxCount {
		writeError(w, http.StatusBadRequest, "не больше 5 файлов за раз")
		return
	}

	type pending struct {
		id, name, contentType, path string
		size                        int64
	}

	messageID := uuid.NewString()
	dir := filepath.Join(h.uploadDir, supportAttachmentDir, ticketID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		writeError(w, http.StatusInternalServerError, "storage unavailable")
		return
	}

	saved := make([]pending, 0, len(headers))
	// Ошибка на середине не должна оставлять половину файлов на диске: пишем
	// все, и при неудаче убираем уже записанные.
	cleanup := func() {
		for _, p := range saved {
			_ = os.Remove(p.path)
		}
	}

	for _, header := range headers {
		if header.Size > supportAttachmentMaxBytes {
			cleanup()
			writeError(w, http.StatusRequestEntityTooLarge, "файл больше 8 МБ")
			return
		}
		ext := strings.ToLower(filepath.Ext(header.Filename))
		contentType, allowed := supportAttachmentTypes[ext]
		if !allowed {
			cleanup()
			writeError(w, http.StatusBadRequest, "допустимы png, jpg, webp, gif, txt, log, json, zip, pdf")
			return
		}

		file, err := header.Open()
		if err != nil {
			cleanup()
			writeError(w, http.StatusBadRequest, "не удалось прочитать файл")
			return
		}
		attachmentID := uuid.NewString()
		destPath := filepath.Join(dir, attachmentID+ext)
		out, err := os.Create(destPath)
		if err != nil {
			_ = file.Close()
			cleanup()
			writeError(w, http.StatusInternalServerError, "failed to save file")
			return
		}
		written, copyErr := io.Copy(out, io.LimitReader(file, supportAttachmentMaxBytes))
		closeErr := out.Close()
		_ = file.Close()
		if copyErr != nil || closeErr != nil {
			_ = os.Remove(destPath)
			cleanup()
			writeError(w, http.StatusInternalServerError, "failed to save file")
			return
		}
		saved = append(saved, pending{
			id:          attachmentID,
			name:        filepath.Base(header.Filename),
			contentType: contentType,
			path:        destPath,
			size:        written,
		})
	}

	isStaff := isStaffRole(claims.Role)
	label := "Вложение: " + saved[0].name
	if len(saved) > 1 {
		label = "Вложения: " + strconv.Itoa(len(saved)) + " файла"
	}

	ctx := r.Context()
	tx, err := h.dbOf(ctx).Begin(ctx)
	if err != nil {
		cleanup()
		writeError(w, http.StatusInternalServerError, "failed")
		return
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		INSERT INTO core.support_messages (id, tenant_id, ticket_id, user_id, is_staff, message, body)
		SELECT $1::uuid, tenant_id, $2::uuid, $3::uuid, $4, $5, $5
		FROM core.support_tickets WHERE id = $2::uuid AND tenant_id = $6
	`, messageID, ticketID, claims.UserID, isStaff, label, claims.TenantID); err != nil {
		cleanup()
		writeError(w, http.StatusInternalServerError, "failed")
		return
	}

	out := make([]map[string]any, 0, len(saved))
	for _, p := range saved {
		// Путь храним относительным: каталог загрузок задаётся настройкой и на
		// разных установках разный, а переносить файлы между ними придётся.
		relPath := filepath.ToSlash(filepath.Join(supportAttachmentDir, ticketID, filepath.Base(p.path)))
		if _, err := tx.Exec(ctx, `
			INSERT INTO core.support_message_attachments (
				id, tenant_id, ticket_id, message_id, user_id, is_staff,
				disk, path, original_name, mime_type, size_bytes
			) VALUES ($1::uuid, $2, $3::uuid, $4::uuid, $5::uuid, $6, 'local', $7, $8, $9, $10)
		`, p.id, claims.TenantID, ticketID, messageID, claims.UserID, isStaff,
			relPath, p.name, p.contentType, p.size); err != nil {
			cleanup()
			writeError(w, http.StatusInternalServerError, "failed")
			return
		}
		out = append(out, map[string]any{
			"id":           p.id,
			"name":         p.name,
			"size":         p.size,
			"content_type": p.contentType,
			"url":          supportAttachmentURL(ticketID, p.id),
		})
	}

	if _, err := tx.Exec(ctx, `
		UPDATE core.support_tickets SET last_message_at = now(), updated_at = now()
		WHERE id = $1 AND tenant_id = $2
	`, ticketID, claims.TenantID); err != nil {
		cleanup()
		writeError(w, http.StatusInternalServerError, "failed")
		return
	}
	if err := tx.Commit(ctx); err != nil {
		cleanup()
		writeError(w, http.StatusInternalServerError, "failed")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"status":      "uploaded",
		"message_id":  messageID,
		"attachments": out,
		// Прежние поля оставлены: панель до сих пор показывает имя и ссылку
		// первого файла сразу после загрузки.
		"filename": saved[0].name,
		"size":     saved[0].size,
		"url":      supportAttachmentURL(ticketID, saved[0].id),
	})
}

func supportAttachmentURL(ticketID, attachmentID string) string {
	return "/v1/support/" + ticketID + "/attachments/" + attachmentID
}

func (h *Handler) DownloadSupportAttachment(w http.ResponseWriter, r *http.Request) {
	ticketID := chi.URLParam(r, "id")
	messageID := chi.URLParam(r, "messageId")
	claims, ok := h.supportTicketAccess(r, ticketID)
	if !ok {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if _, err := uuid.Parse(messageID); err != nil {
		writeError(w, http.StatusBadRequest, "invalid message id")
		return
	}

	// Сначала ищем вложение в таблице: там лежат все новые файлы, и на одно
	// сообщение их может быть несколько. Ссылки на старые вложения указывают на
	// id сообщения, поэтому ниже остаётся прежний путь через meta.
	var relPath, name, mime string
	if h.dbOf(r.Context()).QueryRow(r.Context(), `
		SELECT path, original_name, COALESCE(mime_type, '')
		FROM core.support_message_attachments
		WHERE id = $1::uuid AND ticket_id = $2::uuid AND tenant_id = $3
	`, messageID, ticketID, claims.TenantID).Scan(&relPath, &name, &mime) == nil {
		h.serveSupportFile(w, r, filepath.Join(h.uploadDir, filepath.FromSlash(relPath)), name, mime)
		return
	}

	var metaRaw []byte
	if h.dbOf(r.Context()).QueryRow(r.Context(), `
		SELECT meta FROM core.support_messages WHERE id = $1 AND ticket_id = $2
	`, messageID, ticketID).Scan(&metaRaw) != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	var meta struct {
		Attachment struct {
			Name        string `json:"name"`
			ContentType string `json:"content_type"`
			Ext         string `json:"ext"`
		} `json:"attachment"`
	}
	if json.Unmarshal(metaRaw, &meta) != nil || meta.Attachment.Ext == "" {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if _, allowed := supportAttachmentTypes[meta.Attachment.Ext]; !allowed {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	h.serveSupportFile(w, r,
		filepath.Join(h.uploadDir, supportAttachmentDir, ticketID, messageID+meta.Attachment.Ext),
		meta.Attachment.Name, meta.Attachment.ContentType)
}

func (h *Handler) serveSupportFile(w http.ResponseWriter, r *http.Request, path, name, contentType string) {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	if name == "" {
		name = filepath.Base(path)
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", "attachment; filename*=UTF-8''"+url.PathEscape(name))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "private, no-store")
	http.ServeFile(w, r, path)
}

func itoa64(n int64) string {
	if n <= 0 {
		return "0"
	}
	buf := make([]byte, 0, 20)
	for n > 0 {
		buf = append([]byte{byte('0' + n%10)}, buf...)
		n /= 10
	}
	return string(buf)
}

// supportAttachmentsFor — вложения сообщения из таблицы.
func (h *Handler) supportAttachmentsFor(ctx context.Context, tenantID, ticketID, messageID string) []map[string]any {
	rows, err := h.dbOf(ctx).Query(ctx, `
		SELECT id::text, original_name, COALESCE(mime_type, ''), size_bytes
		FROM core.support_message_attachments
		WHERE tenant_id = $1 AND ticket_id = $2::uuid AND message_id = $3::uuid
		ORDER BY created_at
	`, tenantID, ticketID, messageID)
	if err != nil {
		return nil
	}
	defer rows.Close()

	out := []map[string]any{}
	for rows.Next() {
		var id, name, mime string
		var size int64
		if rows.Scan(&id, &name, &mime, &size) != nil {
			continue
		}
		out = append(out, map[string]any{
			"id":           id,
			"name":         name,
			"size":         size,
			"content_type": mime,
			"url":          supportAttachmentURL(ticketID, id),
		})
	}
	return out
}

func supportAttachmentInfo(metaRaw []byte, ticketID, messageID string) map[string]any {
	if len(metaRaw) == 0 {
		return nil
	}
	var meta struct {
		Attachment struct {
			Name        string `json:"name"`
			Size        any    `json:"size"`
			ContentType string `json:"content_type"`
			Ext         string `json:"ext"`
		} `json:"attachment"`
	}
	if json.Unmarshal(metaRaw, &meta) != nil || meta.Attachment.Name == "" {
		return nil
	}
	info := map[string]any{
		"name":         meta.Attachment.Name,
		"size":         attachmentSize(meta.Attachment.Size),
		"content_type": meta.Attachment.ContentType,
	}
	if meta.Attachment.Ext != "" {
		info["url"] = supportAttachmentURL(ticketID, messageID)
	}
	return info
}

func attachmentSize(v any) int64 {
	switch n := v.(type) {
	case float64:
		return int64(n)
	case string:
		parsed, err := strconv.ParseInt(n, 10, 64)
		if err != nil {
			return 0
		}
		return parsed
	default:
		return 0
	}
}
