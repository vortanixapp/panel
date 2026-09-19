package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"path"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
)

const siteFileMaxSize = 64 << 10

var siteFileNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

var siteFileTypes = map[string]string{
	".html": "text/html; charset=utf-8",
	".htm":  "text/html; charset=utf-8",
	".txt":  "text/plain; charset=utf-8",
	".xml":  "application/xml; charset=utf-8",
}

type siteFileView struct {
	Name      string    `json:"name"`
	Size      int       `json:"size"`
	UpdatedAt time.Time `json:"updated_at"`
}

func siteFileType(name string) (string, bool) {
	if !siteFileNamePattern.MatchString(name) || strings.Contains(name, "..") {
		return "", false
	}
	contentType, ok := siteFileTypes[strings.ToLower(path.Ext(name))]
	return contentType, ok
}

func (h *Handler) PublicSiteFile(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	contentType, ok := siteFileType(name)
	if !ok {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	ctx := r.Context()
	var content []byte
	if err := h.dbOf(ctx).QueryRow(ctx, `SELECT content FROM core.site_files WHERE name = $1`, name).Scan(&content); err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	setUploadHeaders(w, contentType)
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write(content)
}

func (h *Handler) AdminSiteFiles(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	rows, err := h.dbOf(ctx).Query(ctx, `SELECT name, octet_length(content), updated_at FROM core.site_files ORDER BY name`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load files")
		return
	}
	defer rows.Close()
	files := []siteFileView{}
	for rows.Next() {
		var f siteFileView
		if err := rows.Scan(&f.Name, &f.Size, &f.UpdatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to load files")
			return
		}
		files = append(files, f)
	}
	writeJSON(w, http.StatusOK, map[string]any{"files": files})
}

func (h *Handler) AdminSaveSiteFile(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ctx := r.Context()
	r.Body = http.MaxBytesReader(w, r.Body, siteFileMaxSize+(1<<20))
	name, content, err := readSiteFile(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Не удалось прочитать файл")
		return
	}
	if _, ok := siteFileType(name); !ok {
		writeError(w, http.StatusUnprocessableEntity, "Имя файла: латиница, цифры, точка, дефис или подчёркивание, расширение .html, .htm, .txt или .xml")
		return
	}
	if len(content) == 0 || len(content) > siteFileMaxSize {
		writeError(w, http.StatusUnprocessableEntity, "Файл пустой или больше 64 КБ")
		return
	}
	if !utf8.Valid(content) {
		writeError(w, http.StatusUnprocessableEntity, "Нужен текстовый файл в кодировке UTF-8")
		return
	}
	var view siteFileView
	err = h.dbOf(ctx).QueryRow(ctx, `
		INSERT INTO core.site_files (name, content, updated_at, updated_by)
		VALUES ($1, $2, now(), NULLIF($3, '')::uuid)
		ON CONFLICT (name) DO UPDATE
		SET content = EXCLUDED.content, updated_at = now(), updated_by = EXCLUDED.updated_by
		RETURNING name, octet_length(content), updated_at
	`, name, content, claims.UserID).Scan(&view.Name, &view.Size, &view.UpdatedAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save file")
		return
	}
	audit(ctx, h.dbOf(ctx), claims.UserID, "settings.site_file.save", name, nil)
	writeJSON(w, http.StatusOK, map[string]any{"file": view})
}

func (h *Handler) AdminDeleteSiteFile(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ctx := r.Context()
	name := chi.URLParam(r, "name")
	tag, err := h.dbOf(ctx).Exec(ctx, `DELETE FROM core.site_files WHERE name = $1`, name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete file")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	audit(ctx, h.dbOf(ctx), claims.UserID, "settings.site_file.delete", name, nil)
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func readSiteFile(r *http.Request) (string, []byte, error) {
	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		if err := r.ParseMultipartForm(siteFileMaxSize); err != nil {
			return "", nil, err
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			return "", nil, err
		}
		defer file.Close()
		content, err := io.ReadAll(io.LimitReader(file, siteFileMaxSize+1))
		if err != nil {
			return "", nil, err
		}
		name := strings.TrimSpace(r.FormValue("name"))
		if name == "" {
			name = path.Base(strings.ReplaceAll(header.Filename, "\\", "/"))
		}
		return name, content, nil
	}
	var body struct {
		Name    string `json:"name"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return "", nil, err
	}
	return strings.TrimSpace(body.Name), []byte(body.Content), nil
}
