package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) ServerPluginsList(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	serverID := chi.URLParam(r, "id")
	if !h.authorizeServerAction(w, r, claims, serverID, "settings_read") {
		return
	}
	gameSlug := h.serverGameSlug(r, serverID)
	rows, err := h.dbOf(r.Context()).Query(r.Context(), `
		SELECT p.id::text, p.slug, p.name, p.category, p.version, p.description, p.image_url, p.install_path,
		       COALESCE(sp.installed, false), COALESCE(sp.enabled, true), COALESCE(sp.last_error, '')
		FROM core.plugins p
		LEFT JOIN core.server_plugins sp ON sp.plugin_id = p.id AND sp.server_id = $1::uuid
		WHERE p.active = true
		  AND ($2 = '' OR $2 = ANY(p.supported_games) OR cardinality(p.supported_games) = 0)
		ORDER BY p.name ASC
	`, serverID, gameSlug)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"items": []any{}})
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id, slug, name, installPath string
		var category, version, desc, img, lastErr *string
		var installed, enabled bool
		if rows.Scan(&id, &slug, &name, &category, &version, &desc, &img, &installPath, &installed, &enabled, &lastErr) == nil {
			items = append(items, map[string]any{
				"plugin": map[string]any{
					"id": id, "slug": slug, "name": name, "category": derefStr(category, ""),
					"version": derefStr(version, ""), "description": derefStr(desc, ""),
					"image_url": derefStr(img, ""), "install_path": installPath,
				},
				"server_plugin": map[string]any{
					"installed": installed, "enabled": enabled, "last_error": derefStr(lastErr, ""),
				},
			})
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) ServerPluginInstall(w http.ResponseWriter, r *http.Request) {
	h.serverPluginMutate(w, r, true, true)
}

func (h *Handler) ServerPluginUninstall(w http.ResponseWriter, r *http.Request) {
	h.serverPluginMutate(w, r, false, false)
}

func (h *Handler) ServerPluginToggle(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	serverID := chi.URLParam(r, "id")
	pluginID := chi.URLParam(r, "pluginId")
	var body struct {
		Enabled int `json:"enabled"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	enabled := body.Enabled != 0

	if !h.authorizeServerAction(w, r, claims, serverID, "plugins_apply") {
		return
	}
	state, err := h.loadServerOperableState(r.Context(), serverID)
	if err != nil {
		writeError(w, http.StatusNotFound, "сервер не найден")
		return
	}
	if err := state.ensureAcceptsChanges(); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	var alreadyInstalled bool
	if err := h.dbOf(r.Context()).QueryRow(r.Context(), `
		SELECT installed FROM core.server_plugins WHERE server_id = $1::uuid AND plugin_id = $2::uuid
	`, serverID, pluginID).Scan(&alreadyInstalled); err != nil || !alreadyInstalled {
		writeError(w, http.StatusUnprocessableEntity, "сначала установите плагин на сервер")
		return
	}

	if _, err := h.dbOf(r.Context()).Exec(r.Context(), `
		UPDATE core.server_plugins SET enabled = $3, updated_at = now()
		WHERE server_id = $1::uuid AND plugin_id = $2::uuid
	`, serverID, pluginID, enabled); err != nil {
		writeError(w, http.StatusInternalServerError, "не удалось переключить плагин")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "enabled": enabled})
}

func (h *Handler) serverPluginMutate(w http.ResponseWriter, r *http.Request, installed, enabled bool) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	serverID := chi.URLParam(r, "id")
	pluginID := chi.URLParam(r, "pluginId")
	action := "install"
	if !installed {
		action = "uninstall"
	}

	if !h.authorizeServerAction(w, r, claims, serverID, "plugins_apply") {
		return
	}

	state, err := h.loadServerOperableState(r.Context(), serverID)
	if err != nil {
		writeError(w, http.StatusNotFound, "сервер не найден")
		return
	}
	if err := state.ensureAcceptsChanges(); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	spec, err := h.loadPluginInstallSpec(r.Context(), pluginID)
	if err != nil {
		writeError(w, http.StatusNotFound, "плагин не найден")
		return
	}
	if installed {
		err = spec.ensureInstallable(state.GameSlug)
	} else {
		err = spec.ensureUninstallable()
	}
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	payload := h.pluginAgentPayload(r, serverID, pluginID)
	payload["action"] = action

	if action == "install" {
		cachePath, delErr := h.deliverPluginArchive(w, r, serverID, pluginID)
		if delErr != nil {
			h.recordPluginError(r.Context(), serverID, pluginID, delErr.Error())
			return
		}
		if cachePath != "" {
			payload["archive_cache_path"] = cachePath
		}
	}

	nodeID, err := h.serverNodeID(r.Context(), serverID)
	if err != nil {
		writeError(w, http.StatusNotFound, "сервер не найден")
		return
	}
	if _, err := h.agentCommand(r.Context(), nodeID, serverID, "plugins_apply", payload); err != nil {
		code, msg := explainAgentError("plugins_apply", err)
		h.recordPluginError(r.Context(), serverID, pluginID, msg)
		writeCodedError(w, http.StatusBadGateway, code, msg)
		return
	}

	now := time.Now()
	if _, err := h.dbOf(r.Context()).Exec(r.Context(), `
		INSERT INTO core.server_plugins (server_id, plugin_id, installed, enabled, installed_at, last_error)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5, NULL)
		ON CONFLICT (server_id, plugin_id) DO UPDATE
		SET installed = $3, enabled = $4, installed_at = $5, last_error = NULL, updated_at = now()
	`, serverID, pluginID, installed, enabled, now); err != nil {
		writeError(w, http.StatusInternalServerError, "не удалось сохранить состояние плагина")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) recordPluginError(ctx context.Context, serverID, pluginID, msg string) {
	_, _ = h.dbOf(ctx).Exec(ctx, `
		INSERT INTO core.server_plugins (server_id, plugin_id, installed, enabled, last_error)
		VALUES ($1::uuid, $2::uuid, false, true, $3)
		ON CONFLICT (server_id, plugin_id) DO UPDATE SET last_error = $3, updated_at = now()
	`, serverID, pluginID, msg)
}

func (h *Handler) deliverPluginArchive(w http.ResponseWriter, r *http.Request, serverID, pluginID string) (string, error) {
	nodeID, err := h.serverNodeID(r.Context(), serverID)
	if err != nil {
		writeError(w, http.StatusNotFound, "сервер не найден")
		return "", err
	}
	cachePath, err := h.deliverCatalogArchive(r, nodeID, catalogPlugins, pluginID)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return "", err
	}
	return cachePath, nil
}

func (h *Handler) pluginAgentPayload(r *http.Request, serverID, pluginID string) map[string]any {
	payload := map[string]any{"plugin_id": pluginID}
	gameSlug := h.serverGameSlug(r, serverID)
	if gameSlug != "" {
		payload["game"] = gameSlug
	}
	var installPath, archivePath, archiveType string
	var fileActions, meta []byte
	_ = h.dbOf(r.Context()).QueryRow(r.Context(), `
		SELECT COALESCE(install_path, ''), COALESCE(archive_path, ''), COALESCE(archive_type, 'zip'),
		       COALESCE(file_actions, '[]'::jsonb), COALESCE(meta, '{}'::jsonb)
		FROM core.plugins WHERE id = $1::uuid
	`, pluginID).Scan(&installPath, &archivePath, &archiveType, &fileActions, &meta)
	if installPath != "" {
		payload["install_path"] = installPath
	}
	if archivePath != "" {
		payload["has_archive"] = true
	}
	if archiveType != "" {
		payload["archive_type"] = archiveType
	}
	if len(fileActions) > 0 {
		var actions any
		if json.Unmarshal(fileActions, &actions) == nil {
			payload["file_actions"] = actions
		}
	}
	if len(meta) > 0 {
		var metaMap map[string]any
		if json.Unmarshal(meta, &metaMap) == nil {
			if ua, ok := metaMap["uninstall_actions"]; ok {
				payload["uninstall_actions"] = ua
			}
			if url, ok := metaMap["download_url"].(string); ok && url != "" {
				payload["download_url"] = url
			}
		}
	}
	if instances := h.nodeMysqlInstances(r, serverID); len(instances) > 0 {
		payload["mysql_instances"] = instances
	}
	return payload
}

func (h *Handler) ServerMapsList(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	serverID := chi.URLParam(r, "id")
	if !h.authorizeServerAction(w, r, claims, serverID, "settings_read") {
		return
	}
	gameSlug := h.serverGameSlug(r, serverID)
	rows, err := h.dbOf(r.Context()).Query(r.Context(), `
		SELECT m.id::text, m.slug, m.name, m.category, m.version,
		       COALESCE(sm.installed, false), COALESCE(sm.is_active, false), COALESCE(sm.last_error, '')
		FROM core.maps m
		LEFT JOIN core.games g ON g.id = m.game_id
		LEFT JOIN core.server_maps sm ON sm.map_id = m.id AND sm.server_id = $1::uuid
		WHERE m.active = true
		  AND ($2 = '' OR g.slug = $2 OR m.game_id IS NULL)
		ORDER BY m.name ASC
	`, serverID, gameSlug)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"items": []any{}})
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id, slug, name string
		var category, version, lastErr *string
		var installed, isActive bool
		if rows.Scan(&id, &slug, &name, &category, &version, &installed, &isActive, &lastErr) == nil {
			items = append(items, map[string]any{
				"map": map[string]any{
					"id": id, "slug": slug, "name": name,
					"category": derefStr(category, ""), "version": derefStr(version, ""),
				},
				"server_map": map[string]any{
					"installed": installed, "is_active": isActive, "last_error": derefStr(lastErr, ""),
				},
				"is_active": isActive,
			})
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) ServerMapInstall(w http.ResponseWriter, r *http.Request) {
	h.serverMapMutate(w, r, true, false)
}

func (h *Handler) ServerMapUninstall(w http.ResponseWriter, r *http.Request) {
	h.serverMapMutate(w, r, false, false)
}

func (h *Handler) ServerMapActivate(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	serverID := chi.URLParam(r, "id")
	mapID := chi.URLParam(r, "mapId")

	if !h.authorizeServerAction(w, r, claims, serverID, "maps_apply") {
		return
	}
	state, err := h.loadServerOperableState(r.Context(), serverID)
	if err != nil {
		writeError(w, http.StatusNotFound, "сервер не найден")
		return
	}
	if err := state.ensureAcceptsChanges(); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	var installed bool
	if err := h.dbOf(r.Context()).QueryRow(r.Context(), `
		SELECT installed FROM core.server_maps WHERE server_id = $1::uuid AND map_id = $2::uuid
	`, serverID, mapID).Scan(&installed); err != nil || !installed {
		writeError(w, http.StatusUnprocessableEntity, "сначала установите карту на сервер")
		return
	}

	payload := h.mapAgentPayload(r, serverID, mapID)
	payload["action"] = "activate"
	nodeID, err := h.serverNodeID(r.Context(), serverID)
	if err != nil {
		writeError(w, http.StatusNotFound, "сервер не найден")
		return
	}
	if _, err := h.agentCommand(r.Context(), nodeID, serverID, "maps_apply", payload); err != nil {
		code, msg := explainAgentError("maps_apply", err)
		h.recordMapError(r.Context(), serverID, mapID, msg)
		writeCodedError(w, http.StatusBadGateway, code, msg)
		return
	}

	_, _ = h.dbOf(r.Context()).Exec(r.Context(),
		`UPDATE core.server_maps SET is_active = (map_id = $2::uuid), last_error = NULL, updated_at = now()
		 WHERE server_id = $1::uuid`, serverID, mapID)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) serverMapMutate(w http.ResponseWriter, r *http.Request, installed, isActive bool) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	serverID := chi.URLParam(r, "id")
	mapID := chi.URLParam(r, "mapId")
	action := "install"
	if !installed {
		action = "uninstall"
	}

	if !h.authorizeServerAction(w, r, claims, serverID, "maps_apply") {
		return
	}
	state, err := h.loadServerOperableState(r.Context(), serverID)
	if err != nil {
		writeError(w, http.StatusNotFound, "сервер не найден")
		return
	}
	if err := state.ensureAcceptsChanges(); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	spec, err := h.loadMapInstallSpec(r.Context(), mapID)
	if err != nil {
		writeError(w, http.StatusNotFound, "карта не найдена")
		return
	}
	if installed {
		if err := spec.ensureInstallable(state.GameSlug); err != nil {
			writeError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
	}

	payload := h.mapAgentPayload(r, serverID, mapID)
	payload["action"] = action

	nodeID, err := h.serverNodeID(r.Context(), serverID)
	if err != nil {
		writeError(w, http.StatusNotFound, "сервер не найден")
		return
	}
	if action == "install" {
		cachePath, delErr := h.deliverCatalogArchive(r, nodeID, catalogMaps, mapID)
		if delErr != nil {
			h.recordMapError(r.Context(), serverID, mapID, delErr.Error())
			writeError(w, http.StatusBadGateway, delErr.Error())
			return
		}
		if cachePath != "" {
			payload["archive_cache_path"] = cachePath
		}
	}

	if _, err := h.agentCommand(r.Context(), nodeID, serverID, "maps_apply", payload); err != nil {
		code, msg := explainAgentError("maps_apply", err)
		h.recordMapError(r.Context(), serverID, mapID, msg)
		writeCodedError(w, http.StatusBadGateway, code, msg)
		return
	}

	now := time.Now()
	if _, err := h.dbOf(r.Context()).Exec(r.Context(), `
		INSERT INTO core.server_maps (server_id, map_id, installed, is_active, installed_at, last_error)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5, NULL)
		ON CONFLICT (server_id, map_id) DO UPDATE
		SET installed = $3, is_active = $4, last_error = NULL, updated_at = now()
	`, serverID, mapID, installed, isActive, now); err != nil {
		writeError(w, http.StatusInternalServerError, "не удалось сохранить состояние карты")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) recordMapError(ctx context.Context, serverID, mapID, msg string) {
	_, _ = h.dbOf(ctx).Exec(ctx, `
		INSERT INTO core.server_maps (server_id, map_id, installed, is_active, last_error)
		VALUES ($1::uuid, $2::uuid, false, false, $3)
		ON CONFLICT (server_id, map_id) DO UPDATE SET last_error = $3, updated_at = now()
	`, serverID, mapID, msg)
}

func (h *Handler) mapAgentPayload(r *http.Request, serverID, mapID string) map[string]any {
	payload := map[string]any{"map_id": mapID}
	gameSlug := h.serverGameSlug(r, serverID)
	if gameSlug != "" {
		payload["game_id"] = gameSlug
	}
	var slug, archivePath string
	var fileList, meta []byte
	_ = h.dbOf(r.Context()).QueryRow(r.Context(), `
		SELECT COALESCE(slug, ''), COALESCE(archive_path, ''), COALESCE(file_list, '[]'::jsonb), COALESCE(meta, '{}'::jsonb)
		FROM core.maps WHERE id = $1::uuid
	`, mapID).Scan(&slug, &archivePath, &fileList, &meta)
	if slug != "" {
		payload["slug"] = slug
		payload["map_name"] = slug
	}
	if archivePath != "" {
		payload["has_archive"] = true
	}
	if len(fileList) > 0 {
		var files any
		if json.Unmarshal(fileList, &files) == nil {
			payload["file_list"] = files
			if paths, ok := files.([]any); ok {
				payload["paths"] = paths
			}
		}
	}
	if len(meta) > 0 {
		var metaMap map[string]any
		if json.Unmarshal(meta, &metaMap) == nil {
			if url, ok := metaMap["download_url"].(string); ok && url != "" {
				payload["download_url"] = url
			}
			if at, ok := metaMap["archive_type"].(string); ok && at != "" {
				payload["archive_type"] = at
			}
		}
	}
	return payload
}

func (h *Handler) nodeMysqlInstances(r *http.Request, serverID string) []any {
	var nodeMeta []byte
	_ = h.dbOf(r.Context()).QueryRow(r.Context(), `
		SELECT COALESCE(n.meta, '{}'::jsonb)
		FROM core.servers s
		LEFT JOIN core.nodes n ON n.id = s.node_id
		WHERE s.id = $1
	`, serverID).Scan(&nodeMeta)
	meta := map[string]any{}
	_ = json.Unmarshal(nodeMeta, &meta)
	if raw, ok := meta["mysql_instances"]; ok {
		if arr, ok := raw.([]any); ok && len(arr) > 0 {
			out := make([]any, 0, len(arr))
			for _, item := range arr {
				if m, ok := item.(map[string]any); ok {
					out = append(out, mysqlInstanceWithPassword(m))
					continue
				}
				out = append(out, item)
			}
			return out
		}
	}
	return defaultMysqlInstances()
}

func defaultMysqlInstances() []any {
	return []any{
		mysqlInstanceWithPassword(map[string]any{
			"key": "mysql80-3306", "engine": "mysql", "version": "8.0",
			"port": 3306, "container": "vortanix-mysql80-3306", "enabled": true,
		}),
		mysqlInstanceWithPassword(map[string]any{
			"key": "mysql57-3307", "engine": "mysql", "version": "5.7",
			"port": 3307, "container": "vortanix-mysql57-3307", "enabled": true,
		}),
	}
}

func mysqlInstanceWithPassword(inst map[string]any) map[string]any {
	if pw, ok := inst["root_password"].(string); ok && strings.TrimSpace(pw) != "" {
		return inst
	}
	container, _ := inst["container"].(string)
	port := 0
	switch v := inst["port"].(type) {
	case int:
		port = v
	case float64:
		port = int(v)
	}
	if container == "" || port <= 0 {
		inst["root_password"] = ""
		return inst
	}
	inst["root_password"] = fmt.Sprintf("vtx_%s_%d", strings.ReplaceAll(container, "-", "_"), port)
	return inst
}

func (h *Handler) serverGameSlug(r *http.Request, serverID string) string {
	var gameID string
	_ = h.dbOf(r.Context()).QueryRow(r.Context(), `
		SELECT COALESCE(game_id, '') FROM core.servers WHERE id = $1
	`, serverID).Scan(&gameID)
	return gameID
}
