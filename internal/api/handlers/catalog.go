package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) Features(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"features": []map[string]string{
			{"title": "Игровые серверы", "description": "Docker-based hosting"},
			{"title": "Консоль", "description": "Live WebSocket console"},
			{"title": "Биллинг", "description": "Кошельки и пополнение"},
		},
	})
}

func (h *Handler) listGames(w http.ResponseWriter, r *http.Request, admin bool) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	q := `SELECT id::text, slug, name, description, image_url, active FROM core.games WHERE tenant_id = $1`
	if !admin {
		q += ` AND active = true`
	}
	rows, err := h.dbOf(r.Context()).Query(r.Context(), q, claims.TenantID)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"games": []any{}})
		return
	}
	defer rows.Close()
	list := []map[string]any{}
	for rows.Next() {
		var id, slug, name string
		var desc, img *string
		var active bool
		if rows.Scan(&id, &slug, &name, &desc, &img, &active) == nil {
			list = append(list, map[string]any{
				"id": id, "slug": slug, "name": name, "description": desc,
				"image": h.gameImageURL(img), "active": active,
			})
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"games": list})
}

func (h *Handler) gameImageURL(img *string) any {
	if h.eggCDN != nil {
		return h.eggCDN.URL(img)
	}
	if img == nil {
		return nil
	}
	return *img
}

func (h *Handler) ListGamesPublic(w http.ResponseWriter, r *http.Request) {
	h.listGames(w, r, false)
}

func (h *Handler) GetGamePublic(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	slug := chi.URLParam(r, "slug")
	var id, name string
	var desc, img *string
	err := h.dbOf(r.Context()).QueryRow(r.Context(), `
		SELECT id::text, name, description, image_url FROM core.games
		WHERE tenant_id = $1 AND slug = $2 AND active = true
	`, claims.TenantID, slug).Scan(&id, &name, &desc, &img)
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id": id, "slug": slug, "name": name, "description": desc, "image": h.gameImageURL(img),
	})
}

func (h *Handler) ListTariffsPublic(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	rows, _ := h.dbOf(r.Context()).Query(r.Context(), `
		SELECT id::text, slug, name, price_monthly::float8, currency, slots_min, slots_max,
		       billing_type, ram_mb, disk_mb
		FROM core.tariffs WHERE tenant_id = $1 AND active = true
	`, claims.TenantID)
	defer func() {
		if rows != nil {
			rows.Close()
		}
	}()
	list := []map[string]any{}
	if rows != nil {
		for rows.Next() {
			var id, slug, name, cur, billingType string
			var price float64
			var slotsMin, slotsMax, ram, disk *int
			if rows.Scan(&id, &slug, &name, &price, &cur, &slotsMin, &slotsMax, &billingType, &ram, &disk) == nil {
				list = append(list, map[string]any{
					"id": id, "slug": slug, "name": name, "price_monthly": price, "currency": cur,
					"slots_min": slotsMin, "slots_max": slotsMax, "billing_type": billingType,
					"ram_mb": ram, "disk_mb": disk,
				})
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"tariffs": list})
}

func (h *Handler) ListPlugins(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	rows, err := h.dbOf(r.Context()).Query(r.Context(), `
		SELECT id::text, slug, name, COALESCE(category, ''), COALESCE(version, ''),
		       COALESCE(description, ''), COALESCE(archive_type, ''), COALESCE(archive_path, ''),
		       COALESCE(archive_size, 0), COALESCE(image_path, ''), COALESCE(install_path, ''),
		       supported_games, COALESCE(file_actions, '[]'::jsonb), COALESCE(uninstall_actions, '[]'::jsonb),
		       restart_required, active
		FROM core.plugins
		WHERE tenant_id = $1
		ORDER BY COALESCE(category, '') ASC, name ASC
	`, claims.TenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "не удалось получить список плагинов: "+err.Error())
		return
	}
	defer rows.Close()
	list := []map[string]any{}
	for rows.Next() {
		var id, slug, name, category, version, description, archiveType, archivePath, imagePath, installPath string
		var archiveSize int64
		var games []string
		var fileActions, uninstallActions []byte
		var restart, active bool
		if rows.Scan(&id, &slug, &name, &category, &version, &description, &archiveType, &archivePath,
			&archiveSize, &imagePath, &installPath, &games, &fileActions, &uninstallActions,
			&restart, &active) != nil {
			continue
		}
		if games == nil {
			games = []string{}
		}
		list = append(list, map[string]any{
			"id": id, "slug": slug, "name": name,
			"category": category, "version": version, "description": description,
			"archive_type": archiveType, "has_archive": archivePath != "", "archive_size": archiveSize,
			"image_url":         h.pluginImageURL(r, id, imagePath),
			"install_path":      installPath,
			"supported_games":   games,
			"all_games":         len(games) == 0 || (len(games) == 1 && games[0] == "*"),
			"file_actions":      jsonOrEmptyArray(fileActions),
			"uninstall_actions": jsonOrEmptyArray(uninstallActions),
			"restart_required":  restart,
			"active":            active,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"plugins": list})
}

func jsonOrEmptyArray(raw []byte) any {
	if len(raw) == 0 {
		return []any{}
	}
	var out any
	if json.Unmarshal(raw, &out) != nil || out == nil {
		return []any{}
	}
	return out
}

func (h *Handler) pluginImageURL(r *http.Request, pluginID, imagePath string) string {
	if strings.TrimSpace(imagePath) == "" {
		return ""
	}
	return h.publicBaseURL(r) + "/v1/plugins/images/" + pluginID
}

func (h *Handler) CreatePlugin(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	var body map[string]any
	_ = json.NewDecoder(r.Body).Decode(&body)

	slug := strings.TrimSpace(stringFromBody(body["slug"]))
	name := strings.TrimSpace(stringFromBody(body["name"]))
	category := strings.TrimSpace(stringFromBody(body["category"]))
	version := strings.TrimSpace(stringFromBody(body["version"]))
	description := strings.TrimSpace(stringFromBody(body["description"]))
	restart := boolFromBody(body["restart_required"], false)
	active := boolFromBody(body["active"], true)

	if name == "" {
		writeError(w, http.StatusUnprocessableEntity, "название плагина обязательно")
		return
	}
	if len(category) > 64 {
		writeError(w, http.StatusUnprocessableEntity, "категория не длиннее 64 символов")
		return
	}

	installPath, err := normalizeInstallPath(stringFromBody(body["install_path"]))
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	games, err := normalizeSupportedGames(boolFromBody(body["all_games"], false), body["supported_games"])
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	fileActions := normalizePluginActions(body["file_actions"])
	uninstallActions := normalizePluginActions(body["uninstall_actions"])

	id := strings.TrimSpace(stringFromBody(body["id"]))
	if id == "" && slug != "" {
		_ = h.dbOf(r.Context()).QueryRow(r.Context(),
			`SELECT id::text FROM core.plugins WHERE tenant_id=$1 AND slug=$2`, claims.TenantID, slug).Scan(&id)
	}
	if id != "" {
		tag, err := h.dbOf(r.Context()).Exec(r.Context(), `
			UPDATE core.plugins
			SET name = $3, category = NULLIF($4, ''), version = NULLIF($5, ''),
			    description = NULLIF($6, ''), install_path = $7, supported_games = $8,
			    file_actions = $9::jsonb, uninstall_actions = $10::jsonb,
			    restart_required = $11, active = $12, updated_at = now()
			WHERE id = $1::uuid AND tenant_id = $2
		`, id, claims.TenantID, name, category, version, description, installPath, games,
			fileActions, uninstallActions, restart, active)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "не удалось сохранить плагин: "+err.Error())
			return
		}
		if tag.RowsAffected() == 0 {
			writeError(w, http.StatusNotFound, "плагин не найден")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "updated", "id": id})
		return
	}

	if slug == "" {
		writeError(w, http.StatusUnprocessableEntity, "slug плагина обязателен")
		return
	}
	var newID string
	err = h.dbOf(r.Context()).QueryRow(r.Context(), `
		INSERT INTO core.plugins (tenant_id, slug, name, category, version, description,
		                          install_path, supported_games, file_actions, uninstall_actions,
		                          restart_required, active)
		VALUES ($1, $2, $3, NULLIF($4, ''), NULLIF($5, ''), NULLIF($6, ''),
		        $7, $8, $9::jsonb, $10::jsonb, $11, $12)
		RETURNING id::text
	`, claims.TenantID, slug, name, category, version, description, installPath, games,
		fileActions, uninstallActions, restart, active).Scan(&newID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "не удалось создать плагин: "+err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"id": newID})
}

func normalizeInstallPath(raw string) (string, error) {
	p := strings.Trim(strings.TrimSpace(raw), "/")
	if strings.Contains(p, "..") {
		return "", fmt.Errorf("путь установки не может содержать «..»")
	}
	if len(p) > 255 {
		return "", fmt.Errorf("путь установки не длиннее 255 символов")
	}
	return p, nil
}

func normalizeSupportedGames(allGames bool, raw any) ([]string, error) {
	if allGames {
		return []string{}, nil
	}
	items, _ := raw.([]any)
	seen := map[string]bool{}
	out := []string{}
	for _, it := range items {
		code := strings.ToLower(strings.TrimSpace(stringFromBody(it)))
		if code == "" || code == "*" || seen[code] {
			continue
		}
		seen[code] = true
		out = append(out, code)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("выберите хотя бы одну игру или включите «Все игры»")
	}
	return out, nil
}

var pluginActionKinds = map[string]bool{
	"ensure_contains": true,
	"append_lines":    true,
	"prepend_lines":   true,
	"remove_lines":    true,
	"write_file":      true,
	"replace_regex":   true,
}

func normalizePluginActions(raw any) []byte {
	items, _ := raw.([]any)
	out := []map[string]any{}
	for _, it := range items {
		m, ok := it.(map[string]any)
		if !ok {
			continue
		}
		path := strings.TrimSpace(stringFromBody(m["path"]))
		if path == "" || strings.Contains(path, "..") {
			continue
		}
		kind := strings.ToLower(strings.TrimSpace(stringFromBody(m["action"])))
		if !pluginActionKinds[kind] {
			kind = "ensure_contains"
		}
		action := map[string]any{
			"path":              path,
			"action":            kind,
			"create_if_missing": truthyFlag(m["create_if_missing"]),
		}
		if kind == "replace_regex" {
			pattern := strings.TrimSpace(stringFromBody(m["pattern"]))
			if pattern == "" {
				continue
			}
			action["pattern"] = pattern
			action["replacement"] = stringFromBody(m["replacement"])
		} else {
			lines := splitActionLines(m["lines"])
			if len(lines) == 0 {
				continue
			}
			action["lines"] = lines
		}
		out = append(out, action)
	}
	encoded, err := json.Marshal(out)
	if err != nil {
		return []byte("[]")
	}
	return encoded
}

func truthyFlag(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		switch strings.ToLower(strings.TrimSpace(t)) {
		case "1", "true", "on":
			return true
		}
	}
	return false
}

func splitActionLines(v any) []string {
	var raw []string
	switch t := v.(type) {
	case string:
		normalized := strings.ReplaceAll(strings.ReplaceAll(t, "\r\n", "\n"), "\r", "\n")
		raw = strings.Split(normalized, "\n")
	case []any:
		for _, item := range t {
			raw = append(raw, stringFromBody(item))
		}
	}
	out := []string{}
	for _, line := range raw {
		if strings.TrimSpace(line) == "" {
			continue
		}
		out = append(out, line)
	}
	return out
}

func (h *Handler) ListMaps(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	rows, err := h.dbOf(r.Context()).Query(r.Context(), `
		SELECT m.id::text, m.slug, m.name, COALESCE(m.category, ''), COALESCE(m.version, ''),
		       COALESCE(g.slug, ''), COALESCE(m.archive_path, ''),
		       COALESCE(jsonb_array_length(m.file_list), 0), m.restart_required, m.active
		FROM core.maps m
		LEFT JOIN core.games g ON g.id = m.game_id
		WHERE m.tenant_id = $1
		ORDER BY COALESCE(m.category, '') ASC, m.name ASC
	`, claims.TenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "не удалось получить список карт: "+err.Error())
		return
	}
	defer rows.Close()
	list := []map[string]any{}
	for rows.Next() {
		var id, slug, name, category, version, game, archive string
		var files int
		var restart, active bool
		if rows.Scan(&id, &slug, &name, &category, &version, &game, &archive, &files, &restart, &active) == nil {
			list = append(list, map[string]any{
				"id": id, "slug": slug, "name": name,
				"category": category, "version": version, "game_slug": game,
				"archive_path": archive, "file_count": files,
				"restart_required": restart, "active": active,
			})
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"maps": list})
}

func (h *Handler) CreateMap(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	var body map[string]any
	_ = json.NewDecoder(r.Body).Decode(&body)
	slug := strings.TrimSpace(stringFromBody(body["slug"]))
	name := strings.TrimSpace(stringFromBody(body["name"]))
	gameSlug := strings.TrimSpace(stringFromBody(body["game_slug"]))
	category := strings.TrimSpace(stringFromBody(body["category"]))
	version := strings.TrimSpace(stringFromBody(body["version"]))
	restart := boolFromBody(body["restart_required"], false)
	active := boolFromBody(body["active"], true)

	if name == "" {
		writeError(w, http.StatusUnprocessableEntity, "название карты обязательно")
		return
	}

	var gameID *string
	if gameSlug != "" {
		var found string
		err := h.dbOf(r.Context()).QueryRow(r.Context(),
			`SELECT id::text FROM core.games WHERE tenant_id = $1 AND slug = $2`, claims.TenantID, gameSlug).Scan(&found)
		if err != nil {
			writeError(w, http.StatusUnprocessableEntity, "игра не найдена: "+gameSlug)
			return
		}
		gameID = &found
	}

	id := strings.TrimSpace(stringFromBody(body["id"]))
	if id == "" && slug != "" {
		_ = h.dbOf(r.Context()).QueryRow(r.Context(),
			`SELECT id::text FROM core.maps WHERE tenant_id=$1 AND slug=$2`, claims.TenantID, slug).Scan(&id)
	}
	if id != "" {
		tag, err := h.dbOf(r.Context()).Exec(r.Context(), `
			UPDATE core.maps
			SET name = $3, category = NULLIF($4, ''), version = NULLIF($5, ''),
			    game_id = $6::uuid, restart_required = $7, active = $8
			WHERE id = $1::uuid AND tenant_id = $2
		`, id, claims.TenantID, name, category, version, gameID, restart, active)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "не удалось сохранить карту: "+err.Error())
			return
		}
		if tag.RowsAffected() == 0 {
			writeError(w, http.StatusNotFound, "карта не найдена")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "updated", "id": id})
		return
	}

	if slug == "" {
		writeError(w, http.StatusUnprocessableEntity, "slug карты обязателен")
		return
	}
	var newID string
	err := h.dbOf(r.Context()).QueryRow(r.Context(), `
		INSERT INTO core.maps (tenant_id, slug, name, category, version, game_id, restart_required, active)
		VALUES ($1, $2, $3, NULLIF($4, ''), NULLIF($5, ''), $6::uuid, $7, $8)
		RETURNING id::text
	`, claims.TenantID, slug, name, category, version, gameID, restart, active).Scan(&newID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "не удалось создать карту: "+err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"id": newID})
}

func stringFromBody(v any) string {
	s, _ := v.(string)
	return s
}

func boolFromBody(v any, def bool) bool {
	if b, ok := v.(bool); ok {
		return b
	}
	return def
}
