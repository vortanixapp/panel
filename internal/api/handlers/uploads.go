package handlers

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) publicBaseURL(r *http.Request) string {
	base := strings.TrimRight(strings.TrimSpace(h.apiPublicURL), "/")
	if base != "" && !isLoopbackURL(base) {
		return base
	}
	if r == nil {
		return base
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if proto := firstListValue(r.Header.Get("X-Forwarded-Proto")); proto != "" {
		scheme = proto
	}
	host := firstListValue(r.Header.Get("X-Forwarded-Host"))
	if host == "" {
		host = r.Host
	}
	if host == "" {
		return base
	}
	return scheme + "://" + host
}

func (h *Handler) uploadPublicURL(r *http.Request, stored string) string {
	stored = strings.TrimSpace(stored)
	if stored == "" {
		return ""
	}
	path := stored
	if u, err := url.Parse(stored); err == nil && u.Host != "" {
		path = u.Path
	}
	if !strings.HasPrefix(path, "/v1/uploads/") {
		return stored
	}
	return h.publicBaseURL(r) + path
}

func (h *Handler) avatarPublicURL(r *http.Request, stored *string) *string {
	if stored == nil {
		return nil
	}
	fixed := h.uploadPublicURL(r, *stored)
	if fixed == "" {
		return nil
	}
	return &fixed
}

func isLoopbackURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	switch u.Hostname() {
	case "localhost", "127.0.0.1", "::1", "":
		return true
	}
	return false
}

func firstListValue(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	return strings.TrimSpace(strings.Split(v, ",")[0])
}

func (h *Handler) brandingPublicURL(r *http.Request, relative string) string {
	relative = strings.TrimSpace(relative)
	if relative == "" {
		return ""
	}
	if strings.HasPrefix(relative, "http://") || strings.HasPrefix(relative, "https://") {
		return relative
	}
	relative = strings.TrimPrefix(relative, "/")
	relative = strings.TrimPrefix(relative, "uploads/")
	name := strings.TrimPrefix(relative, "branding/")
	if name == "" || strings.Contains(name, "..") || strings.Contains(name, "/") {
		return ""
	}
	return h.publicBaseURL(r) + "/v1/uploads/branding/" + name
}

func setUploadHeaders(w http.ResponseWriter, contentType string) {
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; img-src data:; sandbox")
}

func (h *Handler) ServeBranding(w http.ResponseWriter, r *http.Request) {
	filename := chi.URLParam(r, "filename")
	if filename == "" || strings.Contains(filename, "..") || strings.Contains(filename, "/") {
		writeError(w, http.StatusBadRequest, "invalid filename")
		return
	}
	path := filepath.Join(h.uploadDir, "branding", filename)
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	ext := strings.ToLower(filepath.Ext(filename))
	contentType := "application/octet-stream"
	switch ext {
	case ".jpg", ".jpeg":
		contentType = "image/jpeg"
	case ".png":
		contentType = "image/png"
	case ".webp":
		contentType = "image/webp"
	case ".gif":
		contentType = "image/gif"
	case ".svg":
		contentType = "image/svg+xml"
	case ".ico":
		contentType = "image/x-icon"
	}
	setUploadHeaders(w, contentType)
	w.Header().Set("Cache-Control", "public, max-age=300")
	http.ServeFile(w, r, path)
}

// attachmentDisposition собирает заголовок Content-Disposition для отдачи файла.
// Имя приходит от пользователя (путь в файловом менеджере, имя вложения),
// поэтому в filename= уходит только безопасный набор символов, а полное имя —
// в filename* по RFC 5987. Без этого кавычка в имени закрывала параметр и
// позволяла дописать в заголовок свои значения.
func attachmentDisposition(name string) string {
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." {
		name = "download.bin"
	}
	var ascii strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			ascii.WriteRune(r)
		case r == '.', r == '-', r == '_', r == ' ', r == '(', r == ')', r == '[', r == ']':
			ascii.WriteRune(r)
		default:
			ascii.WriteByte('_')
		}
	}
	fallback := strings.TrimSpace(ascii.String())
	if fallback == "" || fallback == "." || fallback == ".." {
		fallback = "download.bin"
	}
	return `attachment; filename="` + fallback + `"; filename*=UTF-8''` + url.PathEscape(name)
}
