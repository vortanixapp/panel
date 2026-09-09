package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/vortanixapp/panel/pkg/gamesettings"
)

type settingsSaveRequest struct {
	Values map[string]string `json:"values"`
}

func (h *Handler) ServerSettingsSave(w http.ResponseWriter, r *http.Request) {
	serverID := chi.URLParam(r, "id")
	claims, ok := h.ensureServerForTenant(w, r, serverID)
	if !ok {
		return
	}
	if !h.authorizeServerAction(w, r, claims, serverID, "settings_write") {
		return
	}
	ctx := r.Context()

	var body settingsSaveRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid json"})
		return
	}
	if len(body.Values) == 0 {
		writeJSON(w, http.StatusOK, map[string]any{
			"ok": true, "saved": []string{}, "files_written": []string{},
			"restart_required": false, "message": "Изменений нет",
		})
		return
	}

	row, err := h.loadServerGameSettingsRow(ctx, serverID)
	if err != nil {
		writeError(w, http.StatusNotFound, "server not found")
		return
	}
	profile, ok := gamesettings.For(row.GameID)
	if !ok {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"ok": false, "error": "Для этой игры настройки не описаны",
		})
		return
	}

	st := h.loadSettingsState(ctx, serverID, profile, row)
	current, _ := st.fieldValues()
	slots := st.slotsLimit()

	unknown := []string{}
	validated := map[string]gamesettings.Field{}
	normalized := map[string]string{}
	for key, raw := range body.Values {
		field, has := profile.FieldByKey(key)
		if !has {
			unknown = append(unknown, key)
			continue
		}
		if field.Secret && raw == secretSentinel {
			continue
		}
		if field.Slots && slots > 0 {
			raw = strconv.Itoa(slots)
		}
		if field.ReadOnly && !(field.Slots && slots > 0) {
			continue
		}
		value := raw
		if !(field.Slots && slots > 0) {
			checked, err := gamesettings.Validate(field, raw)
			if err != nil {
				var ve *gamesettings.ValidationError
				status := http.StatusUnprocessableEntity
				payload := map[string]any{"ok": false, "error": err.Error()}
				if errors.As(err, &ve) {
					payload["error"] = ve.Message
					payload["field"] = ve.Key
				}
				writeJSON(w, status, payload)
				return
			}
			value = checked
		}
		validated[key] = field
		normalized[key] = value
	}
	if len(unknown) > 0 {
		sort.Strings(unknown)
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"ok": false, "error": "неизвестные поля: " + strings.Join(unknown, ", "),
		})
		return
	}
	if len(normalized) == 0 {
		writeJSON(w, http.StatusOK, map[string]any{
			"ok": true, "saved": []string{}, "files_written": []string{},
			"restart_required": false, "message": "Изменений нет",
		})
		return
	}

	byFile := map[string]map[string]string{}
	for key, value := range normalized {
		field := validated[key]
		fileID := field.File
		if fileID == "" && len(profile.Files) > 0 {
			fileID = profile.Files[0].ID
		}
		if byFile[fileID] == nil {
			byFile[fileID] = map[string]string{}
		}
		byFile[fileID][field.Prop] = value
	}

	written := []string{}
	failed := []string{}
	newStartup := st.startup

	for _, fileID := range sortedFileIDs(byFile) {
		file, has := profile.FileByID(fileID)
		if !has {
			continue
		}
		content := st.contents[fileID]
		updated, applyErr := gamesettings.ApplyFile(file, content, byFile[fileID])
		if applyErr != nil {
			failed = append(failed, st.paths[fileID]+": "+applyErr.Error())
			continue
		}
		if file.IsStartup() {
			newStartup = updated
			written = append(written, fileID)
			continue
		}
		if h.agentFilesWriteQuiet(ctx, serverID, st.paths[fileID], updated) {
			written = append(written, fileID)
			continue
		}
		failed = append(failed, st.paths[fileID])
	}

	if len(failed) > 0 {
		writeJSON(w, http.StatusBadGateway, map[string]any{
			"ok": false,
			"error": "Не удалось записать настройки на ноду: " + strings.Join(failed, ", ") +
				". Проверьте, что нода и агент на связи.",
			"files_written": written,
		})
		return
	}

	saved := map[string]any{}
	savedKeys := []string{}
	for key := range normalized {
		saved[key] = normalized[key]
		savedKeys = append(savedKeys, key)
	}
	sort.Strings(savedKeys)

	if newStartup != st.startup {
		if err := h.updateServerStartupParamsString(ctx, serverID, newStartup); err != nil {
			writeError(w, http.StatusInternalServerError, "не удалось сохранить параметры запуска")
			return
		}
	}
	if err := h.mergeGameSettingsConfig(ctx, serverID, profile.Key, saved); err != nil {
		writeError(w, http.StatusInternalServerError, "не удалось сохранить настройки")
		return
	}

	audit(ctx, h.dbOf(ctx), claims.UserID,
		"server.settings.update", "server:"+serverID, map[string]any{
			"game": profile.Key, "fields": savedKeys, "files": written,
		})

	restartFields := []string{}
	for _, key := range savedKeys {
		field := validated[key]
		if field.AppliesLive {
			continue
		}
		if current[key] == normalized[key] {
			continue
		}
		restartFields = append(restartFields, field.Label)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok": true, "saved": savedKeys, "files_written": written,
		"restart_required": len(restartFields) > 0,
		"restart_fields":   restartFields,
		"server_running":   h.serverIsRunning(ctx, serverID),
		"message":          "Настройки сохранены",
	})
}

func sortedFileIDs(m map[string]map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func (h *Handler) serverIsRunning(ctx context.Context, serverID string) bool {
	var status string
	err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT COALESCE(NULLIF(runtime_status, ''), status)
		FROM core.servers WHERE id = $1
	`, serverID).Scan(&status)
	if err != nil {
		return false
	}
	if live, ok := h.cache.GetServerStatus(ctx, serverID); ok {
		status = live
	}
	return strings.EqualFold(status, "running")
}
