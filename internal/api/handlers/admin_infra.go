package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/vortanixapp/panel/internal/api/jobwake"
	"github.com/vortanixapp/panel/pkg/gamecatalog"
)

func (h *Handler) LocationDaemonAction(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	nodeID := chi.URLParam(r, "id")
	action := chi.URLParam(r, "action")
	if action != "restart" && action != "logs" && action != "exec" {
		writeError(w, http.StatusBadRequest, "invalid action")
		return
	}
	var body map[string]any
	_ = json.NewDecoder(r.Body).Decode(&body)
	payload, _ := json.Marshal(map[string]any{"node_id": nodeID, "action": action, "params": body})
	_, _ = h.dbOf(r.Context()).Exec(r.Context(), `
		INSERT INTO core.jobs (tenant_id, type, status, payload)
		VALUES ($1, 'daemon_action', 'pending', $2::jsonb)
	`, claims.TenantID, payload)
	jobwake.Notify("daemon_action")
	writeJSON(w, http.StatusAccepted, map[string]string{"status": action})
}

func (h *Handler) ListNodeImages(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}

	type dbGame struct {
		name    string
		version string
		image   string
		active  bool
		build   bool
	}
	enabled := map[string]dbGame{}
	rows, _ := h.dbOf(r.Context()).Query(r.Context(), `
		SELECT g.slug, g.name, g.active, g.build_image, COALESCE(gv.version, ''), COALESCE(gv.docker_image, '')
		FROM core.games g
		LEFT JOIN core.game_versions gv
		       ON gv.game_id = g.id AND gv.tenant_id = g.tenant_id AND gv.active = true
		WHERE g.tenant_id = $1
		ORDER BY g.name, gv.sort_order
	`, claims.TenantID)
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var slug, name, version, image string
			var active, build bool
			if rows.Scan(&slug, &name, &active, &build, &version, &image) == nil {
				key := gamecatalog.Normalize(slug)
				if _, seen := enabled[key]; !seen {
					enabled[key] = dbGame{name: name, version: version, image: image, active: active, build: build}
				}
			}
		}
	}

	// Состояние сборки по нодам: один образ на разных нодах может быть собран,
	// собираться или упасть.
	type buildState struct{ ready, failed, building, total int }
	states := map[string]*buildState{}
	if br, err := h.dbOf(r.Context()).Query(r.Context(), `
		SELECT game_slug, status, count(*) FROM core.node_game_images
		WHERE tenant_id = $1 GROUP BY game_slug, status
	`, claims.TenantID); err == nil {
		defer br.Close()
		for br.Next() {
			var slug, status string
			var n int
			if br.Scan(&slug, &status, &n) != nil {
				continue
			}
			key := gamecatalog.Normalize(slug)
			st, ok := states[key]
			if !ok {
				st = &buildState{}
				states[key] = st
			}
			st.total += n
			switch status {
			case "ready":
				st.ready += n
			case "failed":
				st.failed += n
			case "building":
				st.building += n
			}
		}
	}

	list := []map[string]any{}
	for _, g := range gamecatalog.All() {
		db, inDB := enabled[g.Key]
		name := db.name
		if name == "" {
			name = g.Key
		}
		tag := gamecatalog.DefaultTag(g.Key)
		if db.image != "" && gamecatalog.BelongsTo(g.Key, db.image) {
			tag = gamecatalog.TagOf(db.image)
		}
		st := states[g.Key]
		if st == nil {
			st = &buildState{}
		}
		list = append(list, map[string]any{
			"game":         g.Key,
			"name":         name,
			"version":      db.version,
			"docker_image": gamecatalog.ImageWithTag(g.Key, tag),
			"repository":   gamecatalog.Repository(g.Key),
			"tag":          tag,
			"default_tag":  gamecatalog.DefaultTag(g.Key),
			"image_env":    g.ImageEnv,
			"enabled":      inDB && db.active,
			"in_catalog":   inDB,
			"build_image":  inDB && db.build,
			"build_ready":  st.ready,
			"build_failed": st.failed,
			"building":     st.building,
			"build_nodes":  st.total,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"images": list})
}

func (h *Handler) SetNodeImageTag(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	var body struct {
		GameSlug    string `json:"game_slug"`
		Tag         string `json:"tag"`
		DockerImage string `json:"docker_image"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	slug := gamecatalog.Normalize(strings.TrimSpace(body.GameSlug))
	game, known := gamecatalog.Resolve(slug)
	if !known {
		writeError(w, http.StatusBadRequest, "неизвестная игра: образы задаются только для игр из каталога Vortanix")
		return
	}

	tag := strings.TrimSpace(body.Tag)
	if tag == "" && body.DockerImage != "" {
		if !gamecatalog.BelongsTo(game.Key, body.DockerImage) {
			writeError(w, http.StatusUnprocessableEntity,
				"сторонние образы запрещены: допустим только "+gamecatalog.Repository(game.Key)+":<tag>")
			return
		}
		tag = gamecatalog.TagOf(body.DockerImage)
	}
	if tag == "" {
		tag = gamecatalog.DefaultTag(game.Key)
	}
	if !gamecatalog.ValidTag(tag) {
		writeError(w, http.StatusUnprocessableEntity, "недопустимый тег образа")
		return
	}

	var gameID string
	if err := h.dbOf(r.Context()).QueryRow(r.Context(), `
		SELECT id::text FROM core.games WHERE tenant_id = $1 AND slug = ANY($2::text[])
		ORDER BY slug = $3 DESC LIMIT 1
	`, claims.TenantID, gamecatalog.Slugs(game.Key), game.Key).Scan(&gameID); err != nil {
		writeError(w, http.StatusNotFound, "игра не подключена в этом тенанте")
		return
	}

	image := gamecatalog.ImageWithTag(game.Key, tag)
	_, err := h.dbOf(r.Context()).Exec(r.Context(), `
		INSERT INTO core.game_versions (tenant_id, game_id, version, source_type, docker_image, active)
		VALUES ($1, $2, $3, 'docker', $4, true)
		ON CONFLICT (game_id, version) DO UPDATE SET docker_image = EXCLUDED.docker_image, active = true
	`, claims.TenantID, gameID, tag, image)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "не удалось сохранить тег")
		return
	}
	audit(r.Context(), h.dbOf(r.Context()), claims.TenantID, claims.UserID, "image.tag_set", "game:"+game.Key,
		map[string]any{"tag": tag, "image": image})
	writeJSON(w, http.StatusOK, map[string]any{"status": "saved", "docker_image": image, "tag": tag})
}

// SetImageBuildFlag отмечает игру к сборке образа на нодах.
//
// Флаг отдельный от «игра подключена»: подключить к продаже и собрать образ —
// разные решения, и держать их в одном поле означало бы собирать лишнее либо
// продавать несобранное.
func (h *Handler) SetImageBuildFlag(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	var body struct {
		GameSlug string `json:"game_slug"`
		Build    bool   `json:"build"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	slug := gamecatalog.Normalize(strings.TrimSpace(body.GameSlug))
	if _, known := gamecatalog.Resolve(slug); !known {
		writeError(w, http.StatusBadRequest, "неизвестная игра")
		return
	}

	tag, err := h.dbOf(r.Context()).Exec(r.Context(), `
		UPDATE core.games SET build_image = $3
		WHERE tenant_id = $1 AND slug = $2
	`, claims.TenantID, slug, body.Build)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "не удалось сохранить: "+err.Error())
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "игра не заведена в каталоге этой панели")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "game": slug, "build_image": body.Build})
}

// BuildNodeImages ставит сборку отмеченных образов в очередь. Без неё новая
// игра требовала бы переустановки локации целиком.
func (h *Handler) BuildNodeImages(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	var body struct {
		NodeID string `json:"node_id"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	db := h.dbOf(r.Context())
	nodes := []string{}
	if strings.TrimSpace(body.NodeID) != "" {
		nodes = append(nodes, strings.TrimSpace(body.NodeID))
	} else {
		rows, err := db.Query(r.Context(),
			`SELECT id::text FROM core.nodes WHERE tenant_id = $1 AND is_active = true`, claims.TenantID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "не удалось получить список локаций")
			return
		}
		defer rows.Close()
		for rows.Next() {
			var id string
			if rows.Scan(&id) == nil {
				nodes = append(nodes, id)
			}
		}
	}
	if len(nodes) == 0 {
		writeError(w, http.StatusUnprocessableEntity, "нет активных локаций — собирать образы негде")
		return
	}

	for _, id := range nodes {
		payload, _ := json.Marshal(map[string]string{"node_id": id, "component": "images"})
		if _, err := db.Exec(r.Context(), `
			INSERT INTO core.jobs (tenant_id, type, status, payload)
			VALUES ($1, 'node_setup', 'pending', $2::jsonb)
		`, claims.TenantID, payload); err != nil {
			writeError(w, http.StatusInternalServerError, "не удалось поставить задачу: "+err.Error())
			return
		}
	}
	jobwake.Notify("node_setup")
	writeJSON(w, http.StatusAccepted, map[string]any{"ok": true, "nodes": len(nodes)})
}
