package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/vortanixapp/panel/internal/api/jobwake"
	"github.com/vortanixapp/panel/pkg/buildinfo"
	"github.com/vortanixapp/panel/pkg/gamecatalog"
)

func (h *Handler) LocationDaemonAction(w http.ResponseWriter, r *http.Request) {
	_, ok := tenantClaims(r.Context())
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
		INSERT INTO core.jobs ( type, status, payload)
		VALUES ( 'daemon_action', 'pending', $1::jsonb)
	`, payload)
	jobwake.Notify("daemon_action")
	writeJSON(w, http.StatusAccepted, map[string]string{"status": action})
}

type imageState struct {
	Status      string `json:"status"`
	Error       string `json:"error,omitempty"`
	Image       string `json:"image,omitempty"`
	RecipeRef   string `json:"recipe_ref,omitempty"`
	Outdated    bool   `json:"outdated,omitempty"`
	Interrupted bool   `json:"interrupted,omitempty"`
	QueuedAt    string `json:"queued_at,omitempty"`
	StartedAt   string `json:"started_at,omitempty"`
	BuiltAt     string `json:"built_at,omitempty"`
	UpdatedAt   string `json:"updated_at,omitempty"`
}

type imageNode struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Code     string `json:"code"`
	Country  string `json:"country"`
	Active   bool   `json:"active"`
	Online   bool   `json:"online"`
	CanBuild bool   `json:"can_build"`
	Reason   string `json:"reason,omitempty"`
	Building bool   `json:"building"`
}

func imageRecipeRef() string {
	if v := strings.TrimSpace(os.Getenv("GAME_IMAGES_REF")); v != "" {
		return v
	}
	if v := buildinfo.Current(); v != "dev" {
		return "v" + v
	}
	return ""
}

func formatImageTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func (h *Handler) imageNodes(ctx context.Context) ([]imageNode, error) {
	rows, err := h.dbOf(ctx).Query(ctx, `
		SELECT n.id::text, n.name, COALESCE(n.meta->>'code', n.fqdn, ''), COALESCE(n.country, ''),
		       COALESCE(n.is_active, n.active, true), COALESCE(n.status, ''),
		       COALESCE(n.ssh_host, '') <> '' AND COALESCE(n.ssh_user, '') <> ''
		           AND (COALESCE(n.ssh_password_enc, '') <> '' OR COALESCE(n.meta->>'ssh_password', '') <> ''),
		       COALESCE((SELECT s.status FROM core.node_setups s
		                 WHERE s.node_id = n.id AND s.component = 'docker'), ''),
		       EXISTS (SELECT 1 FROM core.jobs j
		               WHERE j.type = 'node_setup' AND j.status IN ('pending', 'running')
		                 AND j.payload->>'node_id' = n.id::text
		                 AND j.payload->>'component' = 'images')
		FROM core.nodes n
		ORDER BY COALESCE(n.sort_order, 0), n.name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []imageNode{}
	for rows.Next() {
		var n imageNode
		var status, docker string
		var ssh bool
		if err := rows.Scan(&n.ID, &n.Name, &n.Code, &n.Country, &n.Active, &status, &ssh, &docker, &n.Building); err != nil {
			return nil, err
		}
		n.Online = status == "online"
		switch {
		case !n.Active:
			n.Reason = "inactive"
		case !ssh:
			n.Reason = "ssh"
		case docker != "installed":
			n.Reason = "docker"
		default:
			n.CanBuild = true
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func (h *Handler) imageStates(ctx context.Context, nodes []imageNode) (map[string]map[string]imageState, error) {
	building := map[string]bool{}
	for _, n := range nodes {
		building[n.ID] = n.Building
	}
	ref := imageRecipeRef()

	rows, err := h.dbOf(ctx).Query(ctx, `
		SELECT node_id::text, image_key, image, status, error, recipe_ref,
		       queued_at, started_at, built_at, updated_at
		FROM core.node_images
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	states := map[string]map[string]imageState{}
	for rows.Next() {
		var nodeID, key string
		var st imageState
		var queued, started, built *time.Time
		var updated time.Time
		if err := rows.Scan(&nodeID, &key, &st.Image, &st.Status, &st.Error, &st.RecipeRef,
			&queued, &started, &built, &updated); err != nil {
			return nil, err
		}
		if (st.Status == "queued" || st.Status == "building") && !building[nodeID] {
			st.Status = "failed"
			st.Interrupted = true
		}
		st.Outdated = st.Status == "ready" && ref != "" && st.RecipeRef != "" && st.RecipeRef != ref
		st.QueuedAt = formatImageTime(queued)
		st.StartedAt = formatImageTime(started)
		st.BuiltAt = formatImageTime(built)
		st.UpdatedAt = updated.UTC().Format(time.RFC3339)
		if states[key] == nil {
			states[key] = map[string]imageState{}
		}
		states[key][nodeID] = st
	}
	return states, rows.Err()
}

func statesFor(states map[string]map[string]imageState, key string) map[string]imageState {
	if s := states[key]; s != nil {
		return s
	}
	return map[string]imageState{}
}

func (h *Handler) ListNodeImages(w http.ResponseWriter, r *http.Request) {
	_, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	ctx := r.Context()

	type dbGame struct {
		name    string
		version string
		image   string
		active  bool
		build   bool
	}
	enabled := map[string]dbGame{}
	rows, err := h.dbOf(ctx).Query(ctx, `
		SELECT g.slug, g.name, g.active, g.build_image, COALESCE(gv.version, ''), COALESCE(gv.docker_image, '')
		FROM core.games g
		LEFT JOIN core.game_versions gv
		       ON gv.game_id = g.id AND gv.active = true
		ORDER BY g.name, gv.sort_order
	`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
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
	rows.Close()

	nodes, err := h.imageNodes(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	states, err := h.imageStates(ctx, nodes)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}

	catalog := gamecatalog.All()
	images := []map[string]any{}
	runtimeGames := map[string]int{}
	runtimeEnabled := map[string]int{}
	for _, g := range catalog {
		db, inDB := enabled[g.Key]
		runtime := gamecatalog.RuntimeOf(g.Key)
		runtimeGames[runtime]++
		if inDB && db.active {
			runtimeEnabled[runtime]++
		}
		name := db.name
		if name == "" {
			name = g.Name
		}
		if name == "" {
			name = g.Key
		}
		tag := gamecatalog.DefaultTag(g.Key)
		if db.image != "" && gamecatalog.BelongsTo(g.Key, db.image) {
			tag = gamecatalog.TagOf(db.image)
		}
		shared := []string{}
		for _, key := range gamecatalog.SharedImageKeys(g.Key) {
			if key != g.Key {
				shared = append(shared, key)
			}
		}
		images = append(images, map[string]any{
			"game":         g.Key,
			"name":         name,
			"version":      db.version,
			"docker_image": gamecatalog.ImageWithTag(g.Key, tag),
			"repository":   gamecatalog.Repository(g.Key),
			"tag":          tag,
			"runtime":      runtime,
			"shared_with":  shared,
			"enabled":      inDB && db.active,
			"in_catalog":   inDB,
			"build_image":  inDB && db.build,
			"states":       statesFor(states, g.Key),
		})
	}

	runtimes := []map[string]any{}
	for _, rt := range gamecatalog.RuntimeBuilds() {
		key := gamecatalog.RuntimeKey(rt.Kind)
		runtimes = append(runtimes, map[string]any{
			"key":           key,
			"kind":          rt.Kind,
			"image":         rt.Image,
			"games":         runtimeGames[rt.Kind],
			"enabled_games": runtimeEnabled[rt.Kind],
			"states":        statesFor(states, key),
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ref":      imageRecipeRef(),
		"nodes":    nodes,
		"runtimes": runtimes,
		"images":   images,
	})
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
		SELECT id::text FROM core.games WHERE slug = ANY($1::text[])
		ORDER BY slug = $2 DESC LIMIT 1
	`, gamecatalog.Slugs(game.Key), game.Key).Scan(&gameID); err != nil {
		writeError(w, http.StatusNotFound, "игра не подключена в этом тенанте")
		return
	}

	image := gamecatalog.ImageWithTag(game.Key, tag)
	_, err := h.dbOf(r.Context()).Exec(r.Context(), `
		INSERT INTO core.game_versions ( game_id, version, source_type, docker_image, active)
		VALUES ( $1, $2, 'docker', $3, true)
		ON CONFLICT (game_id, version) DO UPDATE SET docker_image = EXCLUDED.docker_image, active = true
	`, gameID, tag, image)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "не удалось сохранить тег")
		return
	}
	audit(r.Context(), h.dbOf(r.Context()), claims.UserID, "image.tag_set", "game:"+game.Key,
		map[string]any{"tag": tag, "image": image})
	writeJSON(w, http.StatusOK, map[string]any{"status": "saved", "docker_image": image, "tag": tag})
}

func (h *Handler) SetImageBuildFlag(w http.ResponseWriter, r *http.Request) {
	_, ok := tenantClaims(r.Context())
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
		UPDATE core.games SET build_image = $2
		WHERE slug = $1
	`, slug, body.Build)
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

func (h *Handler) defaultImageKeys(ctx context.Context) []string {
	keys := gamecatalog.RuntimeKeys()
	rows, err := h.dbOf(ctx).Query(ctx, `SELECT slug FROM core.games WHERE build_image = true ORDER BY slug`)
	if err != nil {
		return keys
	}
	defer rows.Close()
	seen := map[string]bool{}
	for rows.Next() {
		var slug string
		if rows.Scan(&slug) != nil {
			continue
		}
		key, known := gamecatalog.NormalizeImageKey(slug)
		if !known || seen[key] {
			continue
		}
		seen[key] = true
		keys = append(keys, key)
	}
	return keys
}

func mergeImageKeys(a, b []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, list := range [][]string{a, b} {
		for _, key := range list {
			if !seen[key] {
				seen[key] = true
				out = append(out, key)
			}
		}
	}
	return out
}

func (h *Handler) enqueueImageBuild(ctx context.Context, nodeID string, keys []string) error {
	tx, err := h.dbOf(ctx).Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var busy bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM core.jobs
			WHERE type = 'node_setup' AND status IN ('pending', 'running')
			  AND payload->>'node_id' = $1
		)
	`, nodeID).Scan(&busy); err != nil {
		return err
	}

	var jobID string
	var raw []byte
	err = tx.QueryRow(ctx, `
		SELECT id::text, payload FROM core.jobs
		WHERE type = 'node_setup' AND status = 'pending'
		  AND payload->>'node_id' = $1 AND payload->>'component' = 'images'
		ORDER BY created_at
		LIMIT 1
		FOR UPDATE
	`, nodeID).Scan(&jobID, &raw)
	switch {
	case err == nil:
		var pending struct {
			Images []string `json:"images"`
		}
		_ = json.Unmarshal(raw, &pending)
		existing := pending.Images
		if len(existing) == 0 {
			existing = h.defaultImageKeys(ctx)
		}
		keys = mergeImageKeys(existing, keys)
		payload, _ := json.Marshal(map[string]any{"node_id": nodeID, "component": "images", "images": keys})
		if _, err := tx.Exec(ctx, `UPDATE core.jobs SET payload = $2::jsonb WHERE id = $1`, jobID, payload); err != nil {
			return err
		}
	case errors.Is(err, pgx.ErrNoRows):
		payload, _ := json.Marshal(map[string]any{"node_id": nodeID, "component": "images", "images": keys})
		if _, err := tx.Exec(ctx, `
			INSERT INTO core.jobs (type, status, payload)
			VALUES ('node_setup', 'pending', $1::jsonb)
		`, payload); err != nil {
			return err
		}
	default:
		return err
	}

	for _, t := range gamecatalog.BuildTargets(keys) {
		for _, key := range t.Keys {
			if _, err := tx.Exec(ctx, `
				INSERT INTO core.node_images (node_id, image_key, image, status, queued_at, updated_at)
				VALUES ($1, $2, $3, 'queued', now(), now())
				ON CONFLICT (node_id, image_key) DO UPDATE SET
					image     = EXCLUDED.image,
					status    = CASE WHEN core.node_images.status = 'building' THEN 'building' ELSE 'queued' END,
					error     = CASE WHEN core.node_images.status = 'building' THEN core.node_images.error ELSE '' END,
					queued_at = now(),
					updated_at = now()
			`, nodeID, key, t.Image); err != nil {
				return err
			}
		}
	}

	message := fmt.Sprintf("Сборка образов поставлена в очередь: %d шт.", len(gamecatalog.BuildTargets(keys)))
	if _, err := tx.Exec(ctx, `
		UPDATE core.nodes
		SET meta = jsonb_set(
			jsonb_set(
				COALESCE(meta, '{}'::jsonb), '{setup_statuses}',
				COALESCE(meta->'setup_statuses', '{}'::jsonb) || '{"images": "installing"}'::jsonb, true
			),
			'{setup_progress}',
			CASE WHEN $2 THEN COALESCE(meta->'setup_progress', '{}'::jsonb)
			     ELSE jsonb_build_object('log', $3::text, 'completed', false, 'component', 'images')
			END,
			true
		)
		WHERE id = $1
	`, nodeID, busy, message); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (h *Handler) BuildNodeImages(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	var body struct {
		NodeID  string   `json:"node_id"`
		NodeIDs []string `json:"node_ids"`
		Images  []string `json:"images"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	ctx := r.Context()

	keys := []string{}
	for _, raw := range body.Images {
		key, known := gamecatalog.NormalizeImageKey(raw)
		if !known {
			writeError(w, http.StatusBadRequest, "unknown_image")
			return
		}
		keys = mergeImageKeys(keys, []string{key})
	}
	if len(keys) == 0 {
		keys = h.defaultImageKeys(ctx)
	}

	nodes, err := h.imageNodes(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	wanted := map[string]bool{}
	for _, id := range append(body.NodeIDs, body.NodeID) {
		if id = strings.TrimSpace(id); id != "" {
			wanted[id] = true
		}
	}

	queued := []map[string]any{}
	skipped := []map[string]any{}
	for _, n := range nodes {
		if len(wanted) > 0 && !wanted[n.ID] {
			continue
		}
		if len(wanted) == 0 && !n.Active {
			continue
		}
		delete(wanted, n.ID)
		if !n.CanBuild {
			skipped = append(skipped, map[string]any{"node_id": n.ID, "name": n.Name, "reason": n.Reason})
			continue
		}
		if err := h.enqueueImageBuild(ctx, n.ID, keys); err != nil {
			writeError(w, http.StatusInternalServerError, "не удалось поставить сборку в очередь: "+err.Error())
			return
		}
		queued = append(queued, map[string]any{"node_id": n.ID, "name": n.Name})
	}
	for id := range wanted {
		skipped = append(skipped, map[string]any{"node_id": id, "name": "", "reason": "not_found"})
	}

	if len(queued) == 0 {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"error": "no_nodes", "queued": queued, "skipped": skipped,
		})
		return
	}

	jobwake.Notify("node_setup")
	nodeIDs := make([]string, 0, len(queued))
	for _, q := range queued {
		nodeIDs = append(nodeIDs, q["node_id"].(string))
	}
	audit(ctx, h.dbOf(ctx), claims.UserID, "images.build", "images",
		map[string]any{"nodes": nodeIDs, "images": keys})
	writeJSON(w, http.StatusAccepted, map[string]any{
		"ok": true, "images": len(gamecatalog.BuildTargets(keys)), "queued": queued, "skipped": skipped,
	})
}

func (h *Handler) ImageBuildLog(w http.ResponseWriter, r *http.Request) {
	_, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	payload, found := h.locationSetupStatus(r, strings.TrimSpace(r.URL.Query().Get("node_id")))
	if !found {
		writeError(w, http.StatusNotFound, "location not found")
		return
	}
	writeJSON(w, http.StatusOK, payload)
}
