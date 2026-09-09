package jobs

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/vortanixapp/panel/pkg/gamecatalog"
)

func gameImagesDir() string {
	return envOr("GAME_IMAGES_DIR", "images")
}

func buildContext(game string) (string, error) {
	dir := filepath.Join(gameImagesDir(), game)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("нет контекста сборки %s: %w", game, err)
	}

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		body, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return "", err
		}
		mode := int64(0o644)
		if strings.HasSuffix(e.Name(), ".sh") {
			mode = 0o755
		}
		if err := tw.WriteHeader(&tar.Header{
			Name: e.Name(), Mode: mode, Size: int64(len(body)),
		}); err != nil {
			return "", err
		}
		if _, err := tw.Write(body); err != nil {
			return "", err
		}
	}
	if err := tw.Close(); err != nil {
		return "", err
	}
	if err := gz.Close(); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

func gameBuildCommands(games []string) ([]string, error) {
	cmds := []string{"sudo systemctl start docker || true"}
	if len(games) == 0 {
		return append(cmds, "echo 'Ни один образ не отмечен к сборке'"), nil
	}

	cmds = append(cmds,
		"sudo mkdir -p /opt/vortanix/build",
		fmt.Sprintf("echo 'Сборка образов на этой ноде: %d шт.'", len(games)),
	)

	for _, game := range games {
		ctx, err := buildContext(game)
		if err != nil {
			return nil, err
		}
		dir := "/opt/vortanix/build/" + game

		image := gamecatalog.ImageWithTag(game, gamecatalog.DefaultTag(game))
		if image == "" {
			image = "vortanix/" + game + ":latest"
		}
		cmds = append(cmds,
			fmt.Sprintf("echo '--- %s -> %s ---'", game, image),
			fmt.Sprintf("sudo rm -rf %s && sudo mkdir -p %s", dir, dir),
			fmt.Sprintf("printf '%%s' %s | base64 -d | sudo tar -C %s -xzf -", shellQuote(ctx), dir),
			fmt.Sprintf("sudo docker build -t %s %s", image, dir),
		)
	}
	cmds = append(cmds, "sudo docker builder prune -f >/dev/null 2>&1 || true")
	return cmds, nil
}

func (r *Runner) markImagesBuilding(ctx context.Context, tenantID, nodeID string, games []string) {
	for _, game := range games {
		if _, err := r.db.Exec(ctx, `
			INSERT INTO core.node_game_images (tenant_id, node_id, game_slug, status, error, updated_at)
			VALUES ($1, $2, $3, 'building', NULL, now())
			ON CONFLICT (node_id, game_slug)
			DO UPDATE SET status = 'building', error = NULL, updated_at = now()
		`, tenantID, nodeID, game); err != nil {
			log.Printf("сборка образов: состояние %s не записано: %v", game, err)
		}
	}
}

func (r *Runner) markImagesResult(ctx context.Context, tenantID, nodeID string, games []string, errMsg string) {
	status := "ready"
	var errVal any
	if errMsg != "" {
		status = "failed"
		errVal = errMsg
	}
	for _, game := range games {
		if _, err := r.db.Exec(ctx, `
			INSERT INTO core.node_game_images (tenant_id, node_id, game_slug, status, error, built_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, CASE WHEN $4 = 'ready' THEN now() END, now())
			ON CONFLICT (node_id, game_slug) DO UPDATE SET
				status   = EXCLUDED.status,
				error    = EXCLUDED.error,
				built_at = COALESCE(EXCLUDED.built_at, core.node_game_images.built_at),
				updated_at = now()
		`, tenantID, nodeID, game, status, errVal); err != nil {
			log.Printf("сборка образов: результат %s не записан: %v", game, err)
		}
	}
}
