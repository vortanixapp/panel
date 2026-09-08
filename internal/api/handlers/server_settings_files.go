package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"path"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/vortanixapp/panel/pkg/gamesettings"
)

// Правка файла настроек как текста.
//
// Доступны только те файлы, которые объявил профиль игры. Произвольный путь —
// это вкладка SFTP со своим отдельным правом; заводить здесь второй, более
// слабый способ добраться до любого файла не нужно.

type settingsFilePutRequest struct {
	Content    string `json:"content"`
	SHA256Base string `json:"sha256_base"`
}

// settingsFileTarget находит файл профиля и его настоящий путь.
func (h *Handler) settingsFileTarget(
	ctx context.Context, tenantID, serverID, fileID string,
) (gamesettings.ConfigFile, *settingsState, string, bool) {
	row, err := h.loadServerGameSettingsRow(ctx, tenantID, serverID)
	if err != nil {
		return gamesettings.ConfigFile{}, nil, "", false
	}
	profile, ok := gamesettings.For(row.GameID)
	if !ok {
		return gamesettings.ConfigFile{}, nil, "", false
	}
	file, ok := profile.FileByID(fileID)
	if !ok || file.IsStartup() {
		// Аргументы запуска правятся своей вкладкой, а не как файл.
		return gamesettings.ConfigFile{}, nil, "", false
	}
	st := h.loadSettingsState(ctx, tenantID, serverID, profile, row)
	resolved := st.paths[file.ID]
	if !safeSettingsPath(resolved) {
		return gamesettings.ConfigFile{}, nil, "", false
	}
	return file, st, resolved, true
}

// safeSettingsPath — последняя проверка перед отправкой пути агенту.
//
// Путь приходит из профиля, а не от клиента, но в него подставляются значения
// полей (имя ini-файла Project Zomboid берётся из аргументов запуска). Значение
// клиент задаёт сам, поэтому подстановка проверяется так же, как если бы весь
// путь пришёл снаружи.
func safeSettingsPath(p string) bool {
	if p == "" || strings.HasPrefix(p, "/") {
		return false
	}
	clean := path.Clean(p)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return false
	}
	return !strings.Contains(clean, "..")
}

func (h *Handler) ServerSettingsFileGet(w http.ResponseWriter, r *http.Request) {
	serverID := chi.URLParam(r, "id")
	claims, ok := h.ensureServerForTenant(w, r, serverID)
	if !ok {
		return
	}
	if !h.authorizeServerAction(w, r, claims, serverID, "settings_read") {
		return
	}
	ctx := r.Context()
	file, st, resolved, ok := h.settingsFileTarget(ctx, claims.TenantID, serverID, chi.URLParam(r, "fileId"))
	if !ok {
		writeError(w, http.StatusNotFound, "файл настроек не найден")
		return
	}

	content, present := st.contents[file.ID]
	truncated := len(content) > file.MaxFileBytes()
	if truncated {
		// Огромный файл не открываем: он положил бы вкладку, а править его
		// удобнее по SFTP.
		content = ""
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok": true, "id": file.ID, "path": resolved, "content": content,
		"exists": present, "size": len(st.contents[file.ID]),
		"sha256":    contentDigest(st.contents[file.ID]),
		"truncated": truncated, "editable": !truncated,
	})
}

func (h *Handler) ServerSettingsFilePut(w http.ResponseWriter, r *http.Request) {
	serverID := chi.URLParam(r, "id")
	claims, ok := h.ensureServerForTenant(w, r, serverID)
	if !ok {
		return
	}
	if !h.authorizeServerAction(w, r, claims, serverID, "settings_raw_write") {
		return
	}
	ctx := r.Context()

	var body settingsFilePutRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid json"})
		return
	}

	file, st, resolved, ok := h.settingsFileTarget(ctx, claims.TenantID, serverID, chi.URLParam(r, "fileId"))
	if !ok {
		writeError(w, http.StatusNotFound, "файл настроек не найден")
		return
	}
	if len(body.Content) > file.MaxFileBytes() {
		writeJSON(w, http.StatusRequestEntityTooLarge, map[string]any{
			"ok": false, "error": "Файл слишком велик для редактора настроек",
		})
		return
	}

	previous := st.contents[file.ID]
	// Игра переписывает свой конфиг при запуске. Если это случилось, пока вкладка
	// была открыта, слепое сохранение затёрло бы чужие правки — а заметил бы это
	// клиент много позже и по совсем другим признакам.
	if body.SHA256Base != "" && body.SHA256Base != contentDigest(previous) {
		writeJSON(w, http.StatusConflict, map[string]any{
			"ok": false, "code": "stale",
			"error": "Файл изменился на сервере, обновите страницу и перенесите правки заново",
		})
		return
	}

	backup := ""
	if strings.TrimSpace(previous) != "" {
		backup = ".vtx/config-backups/" + file.ID + ".bak"
		if !h.agentFilesWriteQuiet(ctx, claims.TenantID, serverID, backup, previous) {
			// Копию считаем обязательной: сырой редактор — единственное место, где
			// можно одним движением сделать конфиг нечитаемым для игры.
			writeJSON(w, http.StatusBadGateway, map[string]any{
				"ok": false, "error": "Не удалось сохранить резервную копию — правка отменена",
			})
			return
		}
	}

	if !h.agentFilesWriteQuiet(ctx, claims.TenantID, serverID, resolved, body.Content) {
		writeJSON(w, http.StatusBadGateway, map[string]any{
			"ok": false, "error": "Не удалось записать файл на ноду. Проверьте, что нода и агент на связи.",
		})
		return
	}

	audit(ctx, h.dbOf(ctx), claims.TenantID, claims.UserID,
		"server.settings.file", "server:"+serverID, map[string]any{
			"file": file.ID, "path": resolved, "backup": backup,
		})

	writeJSON(w, http.StatusOK, map[string]any{
		"ok": true, "sha256": contentDigest(body.Content), "backup": backup,
		"restart_required": true,
		"server_running":   h.serverIsRunning(ctx, claims.TenantID, serverID),
	})
}
