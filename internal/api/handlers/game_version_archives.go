package handlers

import (
	"io"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

const (
	gameArchiveMaxBytes  = 4 << 30
	gameArchiveCategory  = "game-versions"
	gameArchiveTransfer  = 2 * time.Hour
	gameArchiveTokenSize = 32
)

var gameArchiveExts = []string{".tar.gz", ".tgz", ".zip", ".jar"}

func gameArchiveExt(filename string) (string, bool) {
	lower := strings.ToLower(strings.TrimSpace(filename))
	for _, ext := range gameArchiveExts {
		if strings.HasSuffix(lower, ext) {
			return ext, true
		}
	}
	return "", false
}

func gameArchiveName(filename, ext string) string {
	base := path.Base(strings.ReplaceAll(filename, "\\", "/"))
	base = base[:len(base)-len(ext)]
	var b strings.Builder
	for _, r := range base {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '.', r == '-', r == '_':
			b.WriteRune(r)
		}
		if b.Len() >= 80 {
			break
		}
	}
	name := strings.Trim(b.String(), ".")
	if name == "" {
		name = "server"
	}
	return name + ext
}

func extendTransfer(w http.ResponseWriter) {
	rc := http.NewResponseController(w)
	deadline := time.Now().Add(gameArchiveTransfer)
	_ = rc.SetReadDeadline(deadline)
	_ = rc.SetWriteDeadline(deadline)
}

func (h *Handler) UploadGameVersionArchive(w http.ResponseWriter, r *http.Request) {
	if _, ok := tenantClaims(r.Context()); !ok {
		return
	}
	extendTransfer(w)
	gameID := chi.URLParam(r, "gameId")
	var slug string
	if err := h.dbOf(r.Context()).QueryRow(r.Context(),
		`SELECT slug FROM core.games WHERE id = $1`, gameID).Scan(&slug); err != nil {
		writeError(w, http.StatusNotFound, "game not found")
		return
	}

	reader, err := r.MultipartReader()
	if err != nil {
		writeError(w, http.StatusBadRequest, "Не удалось разобрать форму")
		return
	}

	fields := map[string]string{}
	var rel, archiveName string
	var size int64
	fail := func(status int, message string) {
		if rel != "" {
			_ = h.deleteCatalogFile(rel)
		}
		writeJSON(w, status, map[string]any{"ok": false, "error": message})
	}

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			fail(http.StatusBadRequest, "Не удалось разобрать форму")
			return
		}
		if part.FormName() != "archive" {
			value, _ := io.ReadAll(io.LimitReader(part, 4096))
			_ = part.Close()
			fields[part.FormName()] = strings.TrimSpace(string(value))
			continue
		}
		if rel != "" {
			_ = part.Close()
			continue
		}
		ext, ok := gameArchiveExt(part.FileName())
		if !ok {
			_ = part.Close()
			fail(http.StatusUnprocessableEntity, "Поддерживаются архивы zip, tar.gz и файлы jar")
			return
		}
		if name := fields["name"]; name != "" && h.gameVersionExists(r, gameID, name) {
			_ = part.Close()
			fail(http.StatusConflict, "Версия с таким названием уже есть")
			return
		}
		saved, written, err := h.saveCatalogFile(gameArchiveCategory, slug, "version_", ext, part, gameArchiveMaxBytes)
		_ = part.Close()
		if err != nil {
			fail(http.StatusUnprocessableEntity, err.Error())
			return
		}
		rel, size, archiveName = saved, written, gameArchiveName(part.FileName(), ext)
	}

	name := fields["name"]
	switch {
	case rel == "":
		fail(http.StatusUnprocessableEntity, "Файл архива обязателен")
		return
	case name == "":
		fail(http.StatusBadRequest, "name is required")
		return
	case h.gameVersionExists(r, gameID, name):
		fail(http.StatusConflict, "Версия с таким названием уже есть")
		return
	}
	active := fields["is_active"] != "false"
	sortOrder, _ := strconv.Atoi(fields["sort_order"])
	token := randomHex(gameArchiveTokenSize)
	archiveURL := h.publicBaseURL(r) + "/v1/internal/game-archive/" + token + "/" + archiveName

	var id string
	if err := h.dbOf(r.Context()).QueryRow(r.Context(), `
		INSERT INTO core.game_versions (game_id, version, source_type, archive_url,
			archive_path, archive_name, archive_size, archive_token, active, sort_order)
		VALUES ($1, $2, 'archive', $3, $4, $5, $6, $7, $8, $9)
		RETURNING id::text
	`, gameID, name, archiveURL, rel, archiveName, size, token, active, sortOrder).Scan(&id); err != nil {
		fail(http.StatusInternalServerError, "Не удалось сохранить версию")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"ok": true, "id": id, "archive_name": archiveName, "archive_size": size,
	})
}

func (h *Handler) gameVersionExists(r *http.Request, gameID, name string) bool {
	var exists bool
	_ = h.dbOf(r.Context()).QueryRow(r.Context(), `
		SELECT EXISTS(SELECT 1 FROM core.game_versions WHERE game_id = $1 AND version = $2)
	`, gameID, name).Scan(&exists)
	return exists
}

func (h *Handler) ServeGameArchive(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	if len(token) != gameArchiveTokenSize*2 {
		writeError(w, http.StatusNotFound, "Архив не найден")
		return
	}
	var rel, name string
	if err := h.db.QueryRow(r.Context(), `
		SELECT COALESCE(archive_path, ''), COALESCE(archive_name, '')
		FROM core.game_versions WHERE archive_token = $1
	`, token).Scan(&rel, &name); err != nil || rel == "" {
		writeError(w, http.StatusNotFound, "Архив не найден")
		return
	}
	abs, err := h.resolveCatalogPath(rel)
	if err != nil {
		writeError(w, http.StatusNotFound, "Архив не найден")
		return
	}
	if info, err := os.Stat(abs); err != nil || info.IsDir() {
		writeError(w, http.StatusNotFound, "Архив не найден")
		return
	}
	extendTransfer(w)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeFile(w, r, abs)
}
