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

// Хранилище архивов каталога — то, что в старой панели называлось файловым
// сервером (app/Services/FileStorageService.php).
//
// Архив плагина или карты живёт у панели: она единственная, кто видит его
// целиком и может отдать любой локации. На самой локации потом заводится копия
// в кэше агента — тянуть сотни мегабайт через панель на каждую установку
// незачем, — но источником истины остаётся этот каталог.
const (
	catalogArchiveMaxBytes = 512 * 1024 * 1024 // 500 МБ, как в старой панели
	catalogImageMaxBytes   = 2 * 1024 * 1024   // 2 МБ
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

// catalogSafeSlug оставляет от слага только то, из чего можно составить имя
// каталога. Слаг приходит от администратора и попадает в путь на диске.
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

// catalogRelPath — путь, который хранится в archive_path и image_path: он
// относительный, поэтому переезд панели на другой диск ничего не ломает.
func catalogRelPath(parts ...string) string {
	return strings.Join(parts, "/")
}

// resolveCatalogPath разворачивает относительный путь в абсолютный и не даёт
// выйти за пределы хранилища: значение приходит из базы, но попасть туда могло
// и вручную.
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

// saveCatalogFile кладёт загруженный файл в {uploadDir}/catalog/{категория}/{слаг}
// и возвращает относительный путь и размер.
//
// Размер считаем на лету и обрываем запись при превышении: заранее ему верить
// нельзя, Content-Length подделывается.
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

// deleteCatalogFile убирает артефакт с диска. Отсутствие файла ошибкой не
// считаем: в старой панели неудача удаления архива блокировала удаление самой
// записи, а уже удалённый файл блокировать нечему.
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

// catalogArchiveExtFor подбирает расширение по объявленному типу архива и, если
// тип не задан, по имени файла.
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

// zipFileList перечисляет файлы внутри zip — этот список потом используется,
// чтобы удалить карту с сервера. Без него удаление физически нечем выполнить:
// агент не знает, какие файлы принадлежат карте.
//
// Правила разбора взяты из старой панели (Admin/MapController.php): обратные
// слэши приводятся к прямым, ведущие слэши срезаются, каталоги и пути с '..'
// пропускаются, дубликаты убираются, сортировка без учёта регистра.
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

// UploadPluginArchive принимает архив плагина.
//
// Новый архив обнуляет archive_location_id: копии, разложенные по кэшам
// локаций, стали устаревшими, и перед следующей установкой архив надо
// доставить заново.
func (h *Handler) UploadPluginArchive(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")

	var slug, oldPath string
	if err := h.dbOf(r.Context()).QueryRow(r.Context(),
		`SELECT slug, COALESCE(archive_path, '') FROM core.plugins WHERE id = $1::uuid AND tenant_id = $2`,
		id, claims.TenantID).Scan(&slug, &oldPath); err != nil {
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
		SET archive_path = $3, archive_type = $4, archive_size = $5,
		    archive_location_id = NULL, updated_at = now()
		WHERE id = $1::uuid AND tenant_id = $2
	`, id, claims.TenantID, rel, archiveType, size); err != nil {
		_ = h.deleteCatalogFile(rel)
		writeError(w, http.StatusInternalServerError, "не удалось сохранить архив: "+err.Error())
		return
	}
	if oldPath != "" && oldPath != rel {
		_ = h.deleteCatalogFile(oldPath)
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "uploaded", "archive_type": archiveType, "size": size})
}

// UploadPluginImage принимает превью плагина.
func (h *Handler) UploadPluginImage(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")

	var slug, oldPath string
	if err := h.dbOf(r.Context()).QueryRow(r.Context(),
		`SELECT slug, COALESCE(image_path, '') FROM core.plugins WHERE id = $1::uuid AND tenant_id = $2`,
		id, claims.TenantID).Scan(&slug, &oldPath); err != nil {
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
		`UPDATE core.plugins SET image_path = $3, updated_at = now() WHERE id = $1::uuid AND tenant_id = $2`,
		id, claims.TenantID, rel); err != nil {
		_ = h.deleteCatalogFile(rel)
		writeError(w, http.StatusInternalServerError, "не удалось сохранить картинку: "+err.Error())
		return
	}
	if oldPath != "" && oldPath != rel {
		_ = h.deleteCatalogFile(oldPath)
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "uploaded", "image_url": h.pluginImageURL(r, id, rel)})
}

// ServePluginImage отдаёт превью без авторизации — как и в старой панели:
// картинку показывает клиент на вкладке плагинов своего сервера. У отключённого
// плагина превью не отдаётся.
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

// UploadMapArchive принимает zip карты и сразу разбирает его на список файлов.
//
// Список строится здесь, а не при установке: именно по нему потом удаляются
// файлы карты с сервера, и собрать его позже будет неоткуда — к тому моменту
// архив уже разложен по каталогам игры.
func (h *Handler) UploadMapArchive(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")

	var slug, oldPath string
	if err := h.dbOf(r.Context()).QueryRow(r.Context(),
		`SELECT slug, COALESCE(archive_path, '') FROM core.maps WHERE id = $1::uuid AND tenant_id = $2`,
		id, claims.TenantID).Scan(&slug, &oldPath); err != nil {
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

	// Только zip: список файлов внутри читается именно из него, а без списка
	// карту нечем удалить с сервера.
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
		SET archive_path = $3, archive_size = $4, file_list = $5::jsonb,
		    archive_location_id = NULL, updated_at = now()
		WHERE id = $1::uuid AND tenant_id = $2
	`, id, claims.TenantID, rel, size, encoded); err != nil {
		_ = h.deleteCatalogFile(rel)
		writeError(w, http.StatusInternalServerError, "не удалось сохранить архив: "+err.Error())
		return
	}
	if oldPath != "" && oldPath != rel {
		_ = h.deleteCatalogFile(oldPath)
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "uploaded", "size": size, "file_count": len(list)})
}
