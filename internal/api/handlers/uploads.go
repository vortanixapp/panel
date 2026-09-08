package handlers

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
)

// publicBaseURL — публичный адрес API, каким его видит браузер клиента.
//
// Адреса загруженных файлов складывались из API_PUBLIC_URL, а её значение по
// умолчанию — http://localhost:<порт> самого сервера. Переменную легко забыть
// задать, и тогда панель отдаёт клиенту ссылку на его собственную машину:
// аватар не грузится, а причина по виду страницы не читается совсем.
//
// Поэтому петлевой адрес считаем «не настроено» и берём адрес из самого
// запроса — браузер пришёл именно по нему. В разработке это то же самое
// localhost, так что поведение не меняется.
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

// uploadPublicURL приводит сохранённый адрес нашего файла к текущему адресу
// API. Так чинятся и записи, сделанные когда-то с localhost: от сохранённого
// значения берётся только путь.
//
// Чужие адреса не трогаем: аватар мог приехать от Google или быть указан
// пользователем вручную.
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

// avatarPublicURL — то же самое для необязательного поля профиля.
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
	case ".svg":
		contentType = "image/svg+xml"
	case ".ico":
		contentType = "image/x-icon"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=300")
	http.ServeFile(w, r, path)
}
