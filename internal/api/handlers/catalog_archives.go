package handlers

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-chi/chi/v5"
)

const (
	catalogArchiveMaxBytes = 512 * 1024 * 1024
	catalogImageMaxBytes   = 2 * 1024 * 1024
	catalogPlugins         = "plugins"
	catalogMaps            = "maps"
)

var catalogArchiveExt = map[string]string{
	"zip":   ".zip",
	"tar":   ".tar",
	"targz": ".tar.gz",
}

var catalogImageExt = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
	"image/gif":  ".gif",
}

func catalogSafeSlug(slug string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(slug)) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		}
		if b.Len() >= 64 {
			break
		}
	}
	if b.Len() == 0 {
		return "item"
	}
	return b.String()
}

func catalogRelPath(parts ...string) string {
	return strings.Join(parts, "/")
}

func (h *Handler) resolveCatalogPath(rel string) (string, error) {
	rel = strings.TrimSpace(rel)
	if rel == "" {
		return "", fmt.Errorf("пустой путь к файлу")
	}
	if strings.Contains(rel, "..") {
		return "", fmt.Errorf("недопустимый путь к файлу")
	}
	root, err := filepath.Abs(filepath.Join(h.uploadDir, "catalog"))
	if err != nil {
		return "", err
	}
	full, err := filepath.Abs(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		return "", err
	}
	if full != root && !strings.HasPrefix(full, root+string(os.PathSeparator)) {
		return "", fmt.Errorf("путь вне хранилища каталога")
	}
	return full, nil
}

func (h *Handler) saveCatalogFile(category, slug, prefix, ext string, src multipart.File, maxBytes int64) (string, int64, error) {
	dirRel := catalogRelPath(category, catalogSafeSlug(slug))
	dirAbs, err := h.resolveCatalogPath(dirRel)
	if err != nil {
		return "", 0, err
	}
	if err := os.MkdirAll(dirAbs, 0o755); err != nil {
		return "", 0, fmt.Errorf("хранилище недоступно: %w", err)
	}

	name := prefix + randomHex(8) + ext
	rel := catalogRelPath(dirRel, name)
	abs := filepath.Join(dirAbs, name)

	dst, err := os.Create(abs)
	if err != nil {
		return "", 0, fmt.Errorf("не удалось создать файл: %w", err)
	}
	written, copyErr := io.Copy(dst, io.LimitReader(src, maxBytes+1))
	closeErr := dst.Close()
	if copyErr == nil && closeErr != nil {
		copyErr = closeErr
	}
	if copyErr != nil {
		_ = os.Remove(abs)
		return "", 0, copyErr
	}
	if written > maxBytes {
		_ = os.Remove(abs)
		return "", 0, fmt.Errorf("файл больше допустимых %d МБ", maxBytes/(1024*1024))
	}
	return rel, written, nil
}

func (h *Handler) deleteCatalogFile(rel string) error {
	if strings.TrimSpace(rel) == "" {
		return nil
	}
	abs, err := h.resolveCatalogPath(rel)
	if err != nil {
		return err
	}
	if err := os.Remove(abs); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func catalogArchiveExtFor(archiveType, filename string) (string, string, error) {
	archiveType = strings.ToLower(strings.TrimSpace(archiveType))
	if ext, ok := catalogArchiveExt[archiveType]; ok {
		return archiveType, ext, nil
	}
	lower := strings.ToLower(filename)
	switch {
	case strings.HasSuffix(lower, ".tar.gz"), strings.HasSuffix(lower, ".tgz"):
		return "targz", ".tar.gz", nil
	case strings.HasSuffix(lower, ".tar"):
		return "tar", ".tar", nil
	case strings.HasSuffix(lower, ".zip"):
		return "zip", ".zip", nil
	}
	return "", "", fmt.Errorf("поддерживаются только zip, tar и tar.gz")
}

func zipFileList(path string) ([]string, error) {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("не удалось прочитать zip: %w", err)
	}
	defer zr.Close()

	seen := map[string]bool{}
	out := []string{}
	for _, f := range zr.File {
		name := strings.ReplaceAll(f.Name, "\\", "/")
		name = strings.TrimLeft(name, "/")
		if name == "" || strings.HasSuffix(name, "/") {
			continue
		}
		if strings.Contains(name, "..") {
			continue
		}
		if seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i]) < strings.ToLower(out[j])
	})
	return out, nil
}

func (h *Handler) UploadPluginArchive(w http.ResponseWriter, r *http.Request) {
	_, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")

	var slug, oldPath string
	if err := h.dbOf(r.Context()).QueryRow(r.Context(),
		`SELECT slug, COALESCE(archive_path, '') FROM core.plugins WHERE id = $1::uuid`,
		id).Scan(&slug, &oldPath); err != nil {
		writeError(w, http.StatusNotFound, "плагин не найден")
		return
	}

	if err := r.ParseMultipartForm(16 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "не удалось разобрать форму")
		return
	}
	file, header, err := r.FormFile("archive")
	if err != nil {
		writeError(w, http.StatusBadRequest, "файл архива обязателен")
		return
	}
	defer file.Close()

	archiveType, ext, err := catalogArchiveExtFor(r.FormValue("archive_type"), header.Filename)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	rel, size, err := h.saveCatalogFile(catalogPlugins, slug, "plugin_", ext, file, catalogArchiveMaxBytes)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	if _, err := h.dbOf(r.Context()).Exec(r.Context(), `
		UPDATE core.plugins
		SET archive_path = $2, archive_type = $3, archive_size = $4,
		    archive_location_id = NULL, updated_at = now()
		WHERE id = $1::uuid
	`, id, rel, archiveType, size); err != nil {
		_ = h.deleteCatalogFile(rel)
		writeError(w, http.StatusInternalServerError, "не удалось сохранить архив: "+err.Error())
		return
	}
	if oldPath != "" && oldPath != rel {
		_ = h.deleteCatalogFile(oldPath)
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "uploaded", "archive_type": archiveType, "size": size})
}

func (h *Handler) UploadPluginImage(w http.ResponseWriter, r *http.Request) {
	_, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")

	var slug, oldPath string
	if err := h.dbOf(r.Context()).QueryRow(r.Context(),
		`SELECT slug, COALESCE(image_path, '') FROM core.plugins WHERE id = $1::uuid`,
		id).Scan(&slug, &oldPath); err != nil {
		writeError(w, http.StatusNotFound, "плагин не найден")
		return
	}

	if err := r.ParseMultipartForm(4 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "не удалось разобрать форму")
		return
	}
	file, header, err := r.FormFile("image")
	if err != nil {
		writeError(w, http.StatusBadRequest, "файл картинки обязателен")
		return
	}
	defer file.Close()

	ext, allowed := catalogImageExt[header.Header.Get("Content-Type")]
	if !allowed {
		lower := strings.ToLower(header.Filename)
		switch {
		case strings.HasSuffix(lower, ".jpg"), strings.HasSuffix(lower, ".jpeg"):
			ext = ".jpg"
		case strings.HasSuffix(lower, ".png"):
			ext = ".png"
		case strings.HasSuffix(lower, ".webp"):
			ext = ".webp"
		case strings.HasSuffix(lower, ".gif"):
			ext = ".gif"
		default:
			writeError(w, http.StatusUnprocessableEntity, "картинка должна быть jpg, png, webp или gif")
			return
		}
	}
	rel, _, err := h.saveCatalogFile(catalogPlugins, slug+"/images", "img_", ext, file, catalogImageMaxBytes)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	if _, err := h.dbOf(r.Context()).Exec(r.Context(),
		`UPDATE core.plugins SET image_path = $2, updated_at = now() WHERE id = $1::uuid`,
		id, rel); err != nil {
		_ = h.deleteCatalogFile(rel)
		writeError(w, http.StatusInternalServerError, "не удалось сохранить картинку: "+err.Error())
		return
	}
	if oldPath != "" && oldPath != rel {
		_ = h.deleteCatalogFile(oldPath)
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "uploaded", "image_url": h.pluginImageURL(r, id, rel)})
}

func (h *Handler) ServePluginImage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var rel string
	var active bool
	if err := h.db.QueryRow(r.Context(),
		`SELECT COALESCE(image_path, ''), active FROM core.plugins WHERE id = $1::uuid`, id).Scan(&rel, &active); err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if !active || rel == "" {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	abs, err := h.resolveCatalogPath(rel)
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	info, err := os.Stat(abs)
	if err != nil || info.IsDir() {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	switch strings.ToLower(filepath.Ext(abs)) {
	case ".jpg", ".jpeg":
		w.Header().Set("Content-Type", "image/jpeg")
	case ".png":
		w.Header().Set("Content-Type", "image/png")
	case ".webp":
		w.Header().Set("Content-Type", "image/webp")
	case ".gif":
		w.Header().Set("Content-Type", "image/gif")
	}
	w.Header().Set("Cache-Control", "public, max-age=300")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeFile(w, r, abs)
}

func (h *Handler) UploadMapArchive(w http.ResponseWriter, r *http.Request) {
	_, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")

	var slug, oldPath string
	if err := h.dbOf(r.Context()).QueryRow(r.Context(),
		`SELECT slug, COALESCE(archive_path, '') FROM core.maps WHERE id = $1::uuid`,
		id).Scan(&slug, &oldPath); err != nil {
		writeError(w, http.StatusNotFound, "карта не найдена")
		return
	}

	if err := r.ParseMultipartForm(16 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "не удалось разобрать форму")
		return
	}
	file, header, err := r.FormFile("archive")
	if err != nil {
		writeError(w, http.StatusBadRequest, "файл архива обязателен")
		return
	}
	defer file.Close()

	if !strings.HasSuffix(strings.ToLower(header.Filename), ".zip") {
		writeError(w, http.StatusUnprocessableEntity, "карта загружается только архивом zip")
		return
	}
	rel, size, err := h.saveCatalogFile(catalogMaps, slug, "map_", ".zip", file, catalogArchiveMaxBytes)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	abs, err := h.resolveCatalogPath(rel)
	if err != nil {
		_ = h.deleteCatalogFile(rel)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	list, err := zipFileList(abs)
	if err != nil {
		_ = h.deleteCatalogFile(rel)
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	if len(list) == 0 {
		_ = h.deleteCatalogFile(rel)
		writeError(w, http.StatusUnprocessableEntity, "в архиве нет файлов")
		return
	}
	encoded, err := json.Marshal(list)
	if err != nil {
		_ = h.deleteCatalogFile(rel)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if _, err := h.dbOf(r.Context()).Exec(r.Context(), `
		UPDATE core.maps
		SET archive_path = $2, archive_size = $3, file_list = $4::jsonb,
		    archive_location_id = NULL, updated_at = now()
		WHERE id = $1::uuid
	`, id, rel, size, encoded); err != nil {
		_ = h.deleteCatalogFile(rel)
		writeError(w, http.StatusInternalServerError, "не удалось сохранить архив: "+err.Error())
		return
	}
	if oldPath != "" && oldPath != rel {
		_ = h.deleteCatalogFile(oldPath)
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "uploaded", "size": size, "file_count": len(list)})
}
