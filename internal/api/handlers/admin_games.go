package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

type gameRow struct {
	ID          string
	Slug        string
	Name        string
	Description *string
	ImageURL    *string
	Code        *string
	Query       *string
	MinPort     int
	MaxPort     int
	Active      bool
	Meta        []byte
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (h *Handler) loadGameRow(ctx context.Context, tenantID, id string) (*gameRow, error) {
	var g gameRow
	err := h.readerOf(ctx).QueryRow(ctx, `
		SELECT id::text, slug, name, description, image_url, code, query_protocol,
			COALESCE(min_port, 1024), COALESCE(max_port, 65535), active,
			COALESCE(meta, '{}'::jsonb), created_at
		FROM core.games
		WHERE id = $1 AND tenant_id = $2
	`, id, tenantID).Scan(
		&g.ID, &g.Slug, &g.Name, &g.Description, &g.ImageURL, &g.Code, &g.Query,
		&g.MinPort, &g.MaxPort, &g.Active, &g.Meta, &g.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	g.UpdatedAt = g.CreatedAt
	return &g, nil
}

func gameMetaBool(meta map[string]any, key string, def bool) bool {
	if v, ok := meta[key]; ok {
		switch x := v.(type) {
		case bool:
			return x
		case float64:
			return x != 0
		case string:
			return x == "1" || x == "true"
		}
	}
	return def
}

func gameMetaString(meta map[string]any, key string) string {
	if v, ok := meta[key]; ok && v != nil {
		return fmt.Sprint(v)
	}
	return ""
}

func (g *gameRow) toJSON(includeCounts bool, serverCount, tariffCount int, image any) map[string]any {
	meta := parseMetaMap(g.Meta)
	status := gameMetaBool(meta, "status", g.Active)
	isVisible := gameMetaBool(meta, "is_visible", true)
	out := map[string]any{
		"id":                     g.ID,
		"slug":                   g.Slug,
		"name":                   g.Name,
		"description":            g.Description,
		"image":                  image,
		"code":                   strOrEmpty(g.Code),
		"query":                  strOrEmpty(g.Query),
		"minport":                g.MinPort,
		"maxport":                g.MaxPort,
		"default_startup_params": gameMetaString(meta, "default_startup_params"),
		"is_active":              g.Active,
		"active":                 g.Active,
		"status":                 status,
		"is_visible":             isVisible,
		"created_at":             g.CreatedAt.Format(time.RFC3339),
		"updated_at":             g.UpdatedAt.Format(time.RFC3339),
	}
	if includeCounts {
		out["server_count"] = serverCount
		out["tariff_count"] = tariffCount
	}
	return out
}

func strOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func bodyBoolVal(v any) (bool, bool) {
	switch x := v.(type) {
	case bool:
		return x, true
	case float64:
		return x != 0, true
	case int:
		return x != 0, true
	case string:
		return x == "1" || strings.EqualFold(x, "true"), true
	}
	return false, false
}

func bodyBool(body map[string]any, keys ...string) (bool, bool) {
	for _, k := range keys {
		if v, ok := body[k]; ok {
			return bodyBoolVal(v)
		}
	}
	return false, false
}

func bodyInt(body map[string]any, keys ...string) (int, bool) {
	for _, k := range keys {
		if v, ok := body[k]; ok {
			switch x := v.(type) {
			case float64:
				return int(x), true
			case int:
				return x, true
			case json.Number:
				n, _ := x.Int64()
				return int(n), true
			}
		}
	}
	return 0, false
}

func bodyString(body map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := body[k]; ok && v != nil {
			return strings.TrimSpace(fmt.Sprint(v))
		}
	}
	return ""
}

func mergeGameMeta(existing []byte, body map[string]any, active bool) ([]byte, error) {
	meta := parseMetaMap(existing)
	if v := bodyString(body, "default_startup_params"); v != "" || body["default_startup_params"] != nil {
		if v == "" {
			delete(meta, "default_startup_params")
		} else {
			meta["default_startup_params"] = v
		}
	}
	if b, ok := bodyBool(body, "status"); ok {
		meta["status"] = b
	} else {
		meta["status"] = active
	}
	if b, ok := bodyBool(body, "is_visible"); ok {
		meta["is_visible"] = b
	}
	raw, err := json.Marshal(meta)
	if err != nil {
		return nil, err
	}
	return raw, nil
}

func (h *Handler) ListAdminGames(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	rows, err := h.readerOf(r.Context()).Query(r.Context(), `
		SELECT g.id::text, g.slug, g.name, g.description, g.image_url, g.code, g.query_protocol,
			COALESCE(g.min_port, 1024), COALESCE(g.max_port, 65535), g.active,
			COALESCE(g.meta, '{}'::jsonb), g.created_at,
			(SELECT COUNT(*)::int FROM core.servers s WHERE s.tenant_id = g.tenant_id AND s.game_id = g.slug),
			(SELECT COUNT(*)::int FROM core.tariffs t WHERE t.tenant_id = g.tenant_id AND t.game_id = g.id)
		FROM core.games g
		WHERE g.tenant_id = $1
		ORDER BY g.name
	`, claims.TenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()

	list := []map[string]any{}
	for rows.Next() {
		var g gameRow
		var serverCount, tariffCount int
		if rows.Scan(
			&g.ID, &g.Slug, &g.Name, &g.Description, &g.ImageURL, &g.Code, &g.Query,
			&g.MinPort, &g.MaxPort, &g.Active, &g.Meta, &g.CreatedAt,
			&serverCount, &tariffCount,
		) != nil {
			continue
		}
		g.UpdatedAt = g.CreatedAt
		list = append(list, g.toJSON(true, serverCount, tariffCount, h.gameImageURL(g.ImageURL)))
	}
	writeJSON(w, http.StatusOK, map[string]any{"games": list})
}

func (h *Handler) GetAdminGame(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	g, err := h.loadGameRow(r.Context(), claims.TenantID, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "game not found")
		} else {
			writeError(w, http.StatusInternalServerError, "database error")
		}
		return
	}
	var versionCount int
	_ = h.readerOf(r.Context()).QueryRow(r.Context(), `
		SELECT COUNT(*)::int FROM core.game_versions WHERE game_id = $1 AND tenant_id = $2
	`, id, claims.TenantID).Scan(&versionCount)
	out := g.toJSON(false, 0, 0, h.gameImageURL(g.ImageURL))
	out["servers_count"] = 0
	out["versions"] = h.loadGameVersionsJSON(r, claims.TenantID, id)
	writeJSON(w, http.StatusOK, map[string]any{"game": out})
}

func (h *Handler) GetAdminGameEdit(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	g, err := h.loadGameRow(r.Context(), claims.TenantID, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "game not found")
		} else {
			writeError(w, http.StatusInternalServerError, "database error")
		}
		return
	}
	out := g.toJSON(false, 0, 0, h.gameImageURL(g.ImageURL))
	out["versions"] = h.loadGameVersionsJSON(r, claims.TenantID, id)
	writeJSON(w, http.StatusOK, map[string]any{"game": out})
}

func (h *Handler) loadGameVersionsJSON(r *http.Request, tenantID, gameID string) []map[string]any {
	rows, err := h.readerOf(r.Context()).Query(r.Context(), `
		SELECT id::text, version, source_type,
			COALESCE(archive_url, ''), COALESCE(docker_image, ''), steam_app_id, steam_branch,
			active, sort_order
		FROM core.game_versions
		WHERE tenant_id = $1 AND game_id = $2
		ORDER BY sort_order ASC, created_at ASC
	`, tenantID, gameID)
	if err != nil {
		return []map[string]any{}
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
	return list
}

func (h *Handler) CreateAdminGame(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	name := bodyString(body, "name")
	slug := bodyString(body, "slug")
	code := bodyString(body, "code")
	query := bodyString(body, "query")
	if name == "" || slug == "" || code == "" || query == "" {
		writeError(w, http.StatusBadRequest, "name, slug, code and query are required")
		return
	}
	minPort, okMin := bodyInt(body, "minport", "min_port")
	if !okMin {
		minPort = 1024
	}
	maxPort, okMax := bodyInt(body, "maxport", "max_port")
	if !okMax {
		maxPort = 65535
	}
	active := true
	if b, ok := bodyBool(body, "is_active", "active"); ok {
		active = b
	}
	meta, err := mergeGameMeta(nil, body, active)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid meta")
		return
	}
	var desc *string
	if d := bodyString(body, "description"); d != "" {
		desc = &d
	}
	var image *string
	if img := bodyString(body, "image"); img != "" {
		image = &img
	}
	var id string
	err = h.dbOf(r.Context()).QueryRow(r.Context(), `
		INSERT INTO core.games (tenant_id, slug, name, description, image_url, code, query_protocol,
			min_port, max_port, active, meta)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11::jsonb)
		RETURNING id::text
	`, claims.TenantID, slug, name, desc, image, code, query, minPort, maxPort, active, meta).Scan(&id)
	if err != nil {
		writeError(w, http.StatusBadRequest, "create failed")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"ok": true, "id": id})
}

func (h *Handler) UpdateAdminGame(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	existing, err := h.loadGameRow(r.Context(), claims.TenantID, id)
	if err != nil {
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
	name := bodyString(body, "name")
	if name == "" {
		name = existing.Name
	}
	slug := bodyString(body, "slug")
	if slug == "" {
		slug = existing.Slug
	}
	code := bodyString(body, "code")
	if code == "" {
		code = strOrEmpty(existing.Code)
	}
	query := bodyString(body, "query")
	if query == "" {
		query = strOrEmpty(existing.Query)
	}
	minPort := existing.MinPort
	if v, ok := bodyInt(body, "minport", "min_port"); ok {
		minPort = v
	}
	maxPort := existing.MaxPort
	if v, ok := bodyInt(body, "maxport", "max_port"); ok {
		maxPort = v
	}
	active := existing.Active
	if b, ok := bodyBool(body, "is_active", "active"); ok {
		active = b
	}
	meta, err := mergeGameMeta(existing.Meta, body, active)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid meta")
		return
	}
	desc := existing.Description
	if _, ok := body["description"]; ok {
		d := bodyString(body, "description")
		if d == "" {
			desc = nil
		} else {
			desc = &d
		}
	}
	image := existing.ImageURL
	if _, ok := body["image"]; ok {
		img := bodyString(body, "image")
		if img == "" {
			image = nil
		} else {
			image = &img
		}
	}
	tag, err := h.dbOf(r.Context()).Exec(r.Context(), `
		UPDATE core.games SET
			name = $3, slug = $4, description = $5, image_url = $6, code = $7, query_protocol = $8,
			min_port = $9, max_port = $10, active = $11, meta = $12::jsonb
		WHERE id = $1 AND tenant_id = $2
	`, id, claims.TenantID, name, slug, desc, image, code, query, minPort, maxPort, active, meta)
	if err != nil || tag.RowsAffected() == 0 {
		writeError(w, http.StatusBadRequest, "update failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) DeleteAdminGame(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	tag, err := h.dbOf(r.Context()).Exec(r.Context(), `DELETE FROM core.games WHERE id = $1 AND tenant_id = $2`, id, claims.TenantID)
	if err != nil || tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "game not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) AdminGameToggle(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	var active bool
	err := h.dbOf(r.Context()).QueryRow(r.Context(), `SELECT active FROM core.games WHERE id = $1 AND tenant_id = $2`, id, claims.TenantID).Scan(&active)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "game not found")
		} else {
			writeError(w, http.StatusInternalServerError, "database error")
		}
		return
	}
	newActive := !active
	_, _ = h.dbOf(r.Context()).Exec(r.Context(), `UPDATE core.games SET active = $3 WHERE id = $1 AND tenant_id = $2`, id, claims.TenantID, newActive)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "is_active": newActive, "active": newActive})
}
