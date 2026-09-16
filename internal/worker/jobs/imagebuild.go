package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/vortanixapp/panel/pkg/buildinfo"
	"github.com/vortanixapp/panel/pkg/gamecatalog"
	"github.com/vortanixapp/panel/pkg/sshclient"
)

const (
	imageBuildRoot    = "/opt/vortanix/build"
	imageBuildTimeout = 45 * time.Minute
	imageErrorLimit   = 2000
)

func gameImagesRepo() string {
	return envOr("GAME_IMAGES_REPO", "vortanixapp/panel")
}

func gameImagesRef() string {
	if v := strings.TrimSpace(os.Getenv("GAME_IMAGES_REF")); v != "" {
		return v
	}
	if v := buildinfo.Current(); v != "dev" {
		return "v" + v
	}
	return "main"
}

func imagePrepareCommands(ref string) []string {
	url := fmt.Sprintf("https://codeload.github.com/%s/tar.gz/%s", gameImagesRepo(), ref)
	return []string{
		"sudo systemctl start docker || true",
		hubMirrorCommand(),
		fmt.Sprintf("echo 'Рецепты образов: %s@%s'", gameImagesRepo(), ref),
		fmt.Sprintf("sudo rm -rf %s && sudo mkdir -p %s", imageBuildRoot, imageBuildRoot),
		fmt.Sprintf("curl -fsSL %s | sudo tar -xz -C %s --strip-components=1", shellQuote(url), imageBuildRoot),
	}
}

func imageBuildCommands(t gamecatalog.BuildTarget) []string {
	dir := imageBuildRoot + "/deploy/images/" + t.Recipe
	return []string{
		fmt.Sprintf("test -f %s/Dockerfile || { echo 'нет рецепта %s' >&2; exit 1; }", dir, t.Recipe),
		fmt.Sprintf("sudo docker build -t %s %s", t.Image, dir),
	}
}

func imageCleanupCommands() []string {
	return []string{
		fmt.Sprintf("sudo rm -rf %s", imageBuildRoot),
		"sudo docker builder prune -f >/dev/null 2>&1 || true",
	}
}

func (r *Runner) defaultImageKeys(ctx context.Context) []string {
	return append(gamecatalog.RuntimeKeys(), r.gamesToBuild(ctx)...)
}

func (r *Runner) buildNodeImages(ctx context.Context, jobID string, node *nodeSSH, requested []string, progress *setupProgressWriter) {
	keys := requested
	if len(keys) == 0 {
		keys = r.defaultImageKeys(ctx)
	}
	targets := gamecatalog.BuildTargets(keys)
	if len(targets) == 0 {
		r.failNodeSetup(ctx, jobID, node.ID, "images", "нет известных образов для сборки", progress)
		return
	}

	ref := gameImagesRef()
	for _, t := range targets {
		r.markImageState(ctx, node.ID, t, "queued", "", ref)
	}

	cfg := r.sshConfig(node, imageBuildTimeout)

	hubUser, hubToken := r.dockerHubCreds(ctx)
	prepare := append(registryLoginCommands("images", r.licenseKey(ctx)),
		dockerHubLoginCommands("images", hubUser, hubToken)...)
	prepare = append(prepare, imagePrepareCommands(ref)...)
	fmt.Fprintf(progress, "Образов к сборке: %d\n", len(targets))
	if err := sshclient.Run(cfg, prepare, progress); err != nil {
		for _, t := range targets {
			r.markImageState(ctx, node.ID, t, "failed", err.Error(), ref)
		}
		r.failNodeSetup(ctx, jobID, node.ID, "images", err.Error(), progress)
		return
	}

	var failed []string
	for i, t := range targets {
		fmt.Fprintf(progress, "\n--- [%d/%d] %s -> %s ---\n", i+1, len(targets), t.Label, t.Image)
		r.markImageState(ctx, node.ID, t, "building", "", ref)
		started := time.Now()
		if err := sshclient.Run(cfg, imageBuildCommands(t), progress); err != nil {
			r.markImageState(ctx, node.ID, t, "failed", err.Error(), ref)
			failed = append(failed, t.Label)
			fmt.Fprintf(progress, "✗ %s не собран\n", t.Label)
			continue
		}
		r.markImageState(ctx, node.ID, t, "ready", "", ref)
		fmt.Fprintf(progress, "✓ %s собран за %s\n", t.Label, time.Since(started).Round(time.Second))
	}
	if err := sshclient.Run(cfg, imageCleanupCommands(), progress); err != nil {
		fmt.Fprintf(progress, "⚠ временные файлы сборки не удалены: %v\n", err)
	}

	if len(failed) > 0 {
		r.failNodeSetup(ctx, jobID, node.ID, "images",
			fmt.Sprintf("не собрано %d из %d: %s", len(failed), len(targets), strings.Join(failed, "; ")), progress)
		return
	}

	fmt.Fprintf(progress, "\n✅ images installed\n")
	fullLog := progress.finish()
	r.markSetupInstalled(ctx, r.db, node.ID, "images")
	result, _ := json.Marshal(map[string]any{"ok": true, "log": fullLog})
	_, _ = r.db.Exec(ctx, `UPDATE core.jobs SET status = 'completed', result = $2::jsonb WHERE id = $1`, jobID, result)
}

func (r *Runner) markImageState(ctx context.Context, nodeID string, t gamecatalog.BuildTarget, status, errMsg, ref string) {
	if runes := []rune(errMsg); len(runes) > imageErrorLimit {
		errMsg = string(runes[len(runes)-imageErrorLimit:])
	}
	for _, key := range t.Keys {
		if _, err := r.db.Exec(ctx, `
			INSERT INTO core.node_images
				(node_id, image_key, image, status, error, recipe_ref, queued_at, started_at, built_at, updated_at)
			VALUES ($1, $2, $3, $4, $5,
			        CASE WHEN $4 = 'ready' THEN $6 ELSE '' END,
			        CASE WHEN $4 = 'queued' THEN now() END,
			        CASE WHEN $4 = 'building' THEN now() END,
			        CASE WHEN $4 = 'ready' THEN now() END,
			        now())
			ON CONFLICT (node_id, image_key) DO UPDATE SET
				image      = EXCLUDED.image,
				status     = CASE
				                 WHEN EXCLUDED.status = 'queued' AND core.node_images.status = 'building'
				                 THEN core.node_images.status
				                 ELSE EXCLUDED.status
				             END,
				error      = EXCLUDED.error,
				recipe_ref = CASE WHEN EXCLUDED.status = 'ready' THEN EXCLUDED.recipe_ref ELSE core.node_images.recipe_ref END,
				queued_at  = COALESCE(EXCLUDED.queued_at, core.node_images.queued_at),
				started_at = COALESCE(EXCLUDED.started_at, core.node_images.started_at),
				built_at   = COALESCE(EXCLUDED.built_at, core.node_images.built_at),
				updated_at = now()
		`, nodeID, key, t.Image, status, errMsg, ref); err != nil {
			log.Printf("сборка образов: состояние %s на ноде %s не записано: %v", key, nodeID, err)
		}
	}
}
