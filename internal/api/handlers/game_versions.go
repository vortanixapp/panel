package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/vortanix/vortanix/pkg/gamecatalog"
)

func versionLegacyJSON(id, name, sourceType, archiveURL, dockerImage string, steamAppID *int64, steamBranch *string, active bool, sortOrder int) map[string]any {
	url := archiveURL
	if sourceType == "steam" && steamAppID != nil && *steamAppID > 0 {
		url = fmt.Sprintf("steam:%d", *steamAppID)
	}
	out := map[string]any{
		"id":           id,
		"name":         name,
		"version":      name,
		"source_type":  sourceType,
		"archive_url":  archiveURL,
		"url":          url,
		"docker_image": dockerImage,
		"is_active":    active,
		"active":       active,
		"sort_order":   sortOrder,
	}
	if steamAppID != nil && *steamAppID > 0 {
		out["steam_app_id"] = *steamAppID
	}
	if steamBranch != nil && *steamBranch != "" {
		out["steam_branch"] = *steamBranch
	}
	return out
}

func (h *Handler) resolveDockerImage(ctx context.Context, tenantID, serverID string) string {
	var gameSlug, img string
	err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT s.game_id, COALESCE(gv.docker_image, '')
		FROM core.servers s
		LEFT JOIN core.game_versions gv ON gv.id = s.game_version_id
		WHERE s.id = $1 AND s.tenant_id = $2
	`, serverID, tenantID).Scan(&gameSlug, &img)
	if err != nil {
		return ""
	}
	if img == "" {
		_ = h.dbOf(ctx).QueryRow(ctx, `
			SELECT COALESCE(gv.docker_image, '')
			FROM core.servers s
			JOIN core.games g ON g.tenant_id = s.tenant_id AND g.slug = s.game_id
			JOIN core.game_versions gv ON gv.game_id = g.id AND gv.tenant_id = s.tenant_id AND gv.active = true
			WHERE s.id = $1 AND s.tenant_id = $2
			ORDER BY gv.sort_order ASC, gv.created_at DESC
			LIMIT 1
		`, serverID, tenantID).Scan(&img)
	}

	key := gamecatalog.Normalize(gameSlug)
	if _, known := gamecatalog.Resolve(key); !known {
		return ""
	}
	tag := gamecatalog.DefaultTag(key)
	if img != "" && gamecatalog.BelongsTo(key, img) {
		tag = gamecatalog.TagOf(img)
	}
	return gamecatalog.ImageWithTag(key, tag)
}

func (h *Handler) resolveInstallSpec(ctx context.Context, tenantID, serverID string) map[string]any {
	var sourceType, archiveURL, steamBranch, steamModConfig string
	var steamAppID *int64
	err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT COALESCE(gv.source_type, ''), COALESCE(gv.archive_url, ''),
		       gv.steam_app_id, COALESCE(gv.steam_branch, ''), COALESCE(gv.steam_mod_config, '')
		FROM core.servers s
		JOIN core.game_versions gv ON gv.id = s.game_version_id
		WHERE s.id = $1 AND s.tenant_id = $2
	`, serverID, tenantID).Scan(&sourceType, &archiveURL, &steamAppID, &steamBranch, &steamModConfig)
	if err != nil {
		err = h.dbOf(ctx).QueryRow(ctx, `
			SELECT COALESCE(gv.source_type, ''), COALESCE(gv.archive_url, ''),
			       gv.steam_app_id, COALESCE(gv.steam_branch, ''), COALESCE(gv.steam_mod_config, '')
			FROM core.servers s
			JOIN core.games g ON g.tenant_id = s.tenant_id AND g.slug = s.game_id
			JOIN core.game_versions gv ON gv.game_id = g.id AND gv.tenant_id = s.tenant_id AND gv.active = true
			WHERE s.id = $1 AND s.tenant_id = $2
			ORDER BY gv.sort_order ASC, gv.created_at DESC
			LIMIT 1
		`, serverID, tenantID).Scan(&sourceType, &archiveURL, &steamAppID, &steamBranch, &steamModConfig)
		if err != nil {
			return nil
		}
	}

	spec := map[string]any{"source_type": sourceType}
	switch {
	case archiveURL != "":
		spec["archive_url"] = archiveURL
	case steamAppID != nil && *steamAppID > 0:
		spec["steam_app_id"] = *steamAppID
		if steamBranch != "" {
			spec["steam_branch"] = steamBranch
		}
		if steamModConfig != "" {
			spec["steam_mod_config"] = steamModConfig
		}
	default:
		return nil
	}
	return spec
}

func (h *Handler) catalogImageForGame(ctx context.Context, tenantID, gameID, image string) (string, error) {
	image = strings.TrimSpace(image)
	if image == "" {
		return "", nil
	}
	var slug string
	if err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT slug FROM core.games WHERE id = $1 AND tenant_id = $2
	`, gameID, tenantID).Scan(&slug); err != nil {
		return "", nil
	}
	key := gamecatalog.Normalize(slug)
	if _, known := gamecatalog.Resolve(key); !known {
		return "", nil
	}
	if !gamecatalog.BelongsTo(key, image) {
		return "", fmt.Errorf("сторонние образы запрещены: допустим только %s:<tag>", gamecatalog.Repository(key))
	}
	tag := gamecatalog.TagOf(image)
	if !gamecatalog.ValidTag(tag) {
		return "", fmt.Errorf("недопустимый тег образа")
	}
	return gamecatalog.ImageWithTag(key, tag), nil
}

func (h *Handler) verifyGameTenant(ctx context.Context, tenantID, gameID string) error {
	var ok bool
	err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM core.games WHERE id = $1 AND tenant_id = $2)
	`, gameID, tenantID).Scan(&ok)
	if err != nil {
		return err
	}
	if !ok {
		return pgx.ErrNoRows
	}
	return nil
}

func (h *Handler) ListGameVersions(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	gameID := chi.URLParam(r, "gameId")
	if err := h.verifyGameTenant(r.Context(), claims.TenantID, gameID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "game not found")
		} else {
			writeError(w, http.StatusInternalServerError, "database error")
		}
		return
	}
	rows, err := h.dbOf(r.Context()).Query(r.Context(), `
		SELECT id::text, version, source_type,
			COALESCE(archive_url, ''), COALESCE(docker_image, ''), steam_app_id, steam_branch,
			active, sort_order
		FROM core.game_versions
		WHERE tenant_id = $1 AND game_id = $2
		ORDER BY sort_order ASC, created_at ASC
	`, claims.TenantID, gameID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()
	list := []map[string]any{}
	for rows.Next() {
		var id, name, sourceType, archiveURL, dockerImage string
		var steamAppID *int64
		var steamBranch *string
		var active bool
		var sortOrder int
		if rows.Scan(&id, &name, &sourceType, &archiveURL, &dockerImage, &steamAppID, &steamBranch, &active, &sortOrder) == nil {
			list = append(list, versionLegacyJSON(id, name, sourceType, archiveURL, dockerImage, steamAppID, steamBranch, active, sortOrder))
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"versions": list})
}

func (h *Handler) CreateGameVersion(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	gameID := chi.URLParam(r, "gameId")
	if err := h.verifyGameTenant(r.Context(), claims.TenantID, gameID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "game not found")
		} else {
			writeError(w, http.StatusInternalServerError, "database error")
		}
		return
	}
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	name := bodyString(body, "name", "version")
	if name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	sourceType := bodyString(body, "source_type")
	if sourceType == "" {
		sourceType = "archive"
	}
	archiveURL := bodyString(body, "url", "archive_url")
	dockerImage := bodyString(body, "docker_image")
	steamBranch := bodyString(body, "steam_branch")
	var steamAppID *int64
	if v, ok := bodyInt(body, "steam_app_id"); ok && v > 0 {
		id64 := int64(v)
		steamAppID = &id64
	}
	active := true
	if b, ok := bodyBool(body, "is_active", "active"); ok {
		active = b
	}
	sortOrder := 0
	if v, ok := bodyInt(body, "sort_order"); ok {
		sortOrder = v
	}

	switch sourceType {
	case "archive":
		if archiveURL == "" {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"ok": false, "error": "Для archive версии нужна ссылка (archive_url)"})
			return
		}
		if !strings.HasPrefix(strings.ToLower(archiveURL), "http://") &&
			!strings.HasPrefix(strings.ToLower(archiveURL), "https://") {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"ok": false, "error": "archive_url должен быть http(s) ссылкой"})
			return
		}
	case "steam":
		if steamAppID == nil || *steamAppID <= 0 {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"ok": false, "error": "Для steam версии нужен Steam App ID"})
			return
		}
	case "docker":
	default:
		writeError(w, http.StatusBadRequest, "invalid source_type")
		return
	}

	if img, err := h.catalogImageForGame(r.Context(), claims.TenantID, gameID, dockerImage); err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"ok": false, "error": err.Error()})
		return
	} else if img != "" {
		dockerImage = img
	}

	var archivePtr, dockerPtr, branchPtr *string
	if archiveURL != "" {
		archivePtr = &archiveURL
	}
	if dockerImage != "" {
		dockerPtr = &dockerImage
	}
	if steamBranch != "" {
		branchPtr = &steamBranch
	}

	var id string
	err := h.dbOf(r.Context()).QueryRow(r.Context(), `
		INSERT INTO core.game_versions (tenant_id, game_id, version, source_type, archive_url, docker_image, steam_app_id, steam_branch, active, sort_order)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id::text
	`, claims.TenantID, gameID, name, sourceType, archivePtr, dockerPtr, steamAppID, branchPtr, active, sortOrder).Scan(&id)
	if err != nil {
		writeError(w, http.StatusBadRequest, "create failed")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"ok": true, "id": id})
}

func (h *Handler) UpdateGameVersion(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	gameID := chi.URLParam(r, "gameId")
	versionID := chi.URLParam(r, "id")
	var body map[string]any
	_ = json.NewDecoder(r.Body).Decode(&body)
	if raw := bodyString(body, "docker_image"); raw != "" {
		img, err := h.catalogImageForGame(r.Context(), claims.TenantID, gameID, raw)
		if err != nil {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		if img != "" {
			body["docker_image"] = img
		}
	}
	tag, err := h.dbOf(r.Context()).Exec(r.Context(), `
		UPDATE core.game_versions SET
			version = COALESCE($4, version),
			source_type = COALESCE($5, source_type),
			docker_image = COALESCE($6, docker_image),
			steam_app_id = COALESCE($7::bigint, steam_app_id),
			active = COALESCE($8::boolean, active),
			sort_order = COALESCE($9::int, sort_order),
			meta = COALESCE($10::jsonb, meta)
		WHERE id = $1 AND game_id = $2 AND tenant_id = $3
	`, versionID, gameID, claims.TenantID, body["version"], body["source_type"],
		body["docker_image"], body["steam_app_id"], body["active"], body["sort_order"], body["meta"])
	if err != nil || tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "version not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (h *Handler) DeleteGameVersion(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	gameID := chi.URLParam(r, "gameId")
	versionID := chi.URLParam(r, "id")
	tag, err := h.dbOf(r.Context()).Exec(r.Context(), `
		DELETE FROM core.game_versions WHERE id = $1 AND game_id = $2 AND tenant_id = $3
	`, versionID, gameID, claims.TenantID)
	if err != nil || tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "version not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
