package jobs

import (
	"context"
	"fmt"
	"log"
	"os"
	"path"
	"strings"

	"github.com/vortanixapp/panel/pkg/gamecatalog"
)

func gameImagesRepo() string {
	return envOr("GAME_IMAGES_REPO", "vortanixapp/panel")
}

func gameImagesRef() string {
	if v := strings.TrimSpace(os.Getenv("GAME_IMAGES_REF")); v != "" {
		return v
	}
	version := strings.TrimSpace(os.Getenv("VORTANIX_VERSION"))
	if strings.HasPrefix(version, "v") {
		return version
	}
	return "main"
}

func gameBuildCommands(games []string) ([]string, error) {
	cmds := []string{"sudo systemctl start docker || true", hubMirrorCommand()}
	ref := gameImagesRef()
	url := fmt.Sprintf("https://codeload.github.com/%s/tar.gz/%s", gameImagesRepo(), ref)
	root := "/opt/vortanix/build"

	cmds = append(cmds,
		fmt.Sprintf("echo 'Рецепты образов: %s@%s'", gameImagesRepo(), ref),
		fmt.Sprintf("sudo rm -rf %s && sudo mkdir -p %s", root, root),
		fmt.Sprintf("curl -fsSL %s | sudo tar -xz -C %s --strip-components=1", shellQuote(url), root),
	)

	for _, rt := range gamecatalog.RuntimeBuilds() {
		dir := root + "/deploy/images/_runtime/" + rt.Kind
		cmds = append(cmds,
			fmt.Sprintf("echo '--- рантайм %s -> %s ---'", rt.Kind, rt.Image),
			fmt.Sprintf("sudo docker build -t %s %s", rt.Image, dir),
		)
	}

	builds := gameImageBuilds(games)
	if len(builds) > 0 {
		cmds = append(cmds, fmt.Sprintf("echo 'Отдельных образов игр: %d шт.'", len(builds)))
	}

	for _, b := range builds {
		dir := root + "/deploy/images/" + b.recipe
		cmds = append(cmds,
			fmt.Sprintf("echo '--- %s -> %s ---'", strings.Join(b.games, ", "), b.image),
			fmt.Sprintf("test -f %s/Dockerfile || { echo 'нет рецепта %s для %s' >&2; exit 1; }", dir, b.recipe, strings.Join(b.games, ", ")),
			fmt.Sprintf("sudo docker build -t %s %s", b.image, dir),
		)
	}
	cmds = append(cmds,
		fmt.Sprintf("sudo rm -rf %s", root),
		"sudo docker builder prune -f >/dev/null 2>&1 || true",
	)
	return cmds, nil
}

type gameImageBuild struct {
	image  string
	recipe string
	games  []string
}

func gameImageBuilds(games []string) []gameImageBuild {
	var out []gameImageBuild
	index := map[string]int{}
	for _, game := range games {
		image := gamecatalog.ImageWithTag(game, gamecatalog.DefaultTag(game))
		recipe := path.Base(gamecatalog.Repository(game))
		if image == "" || recipe == "." || recipe == "/" {
			image = "vortanix/" + game + ":latest"
			recipe = game
		}
		if i, ok := index[image]; ok {
			out[i].games = append(out[i].games, game)
			continue
		}
		index[image] = len(out)
		out = append(out, gameImageBuild{image: image, recipe: recipe, games: []string{game}})
	}
	return out
}

func (r *Runner) markImagesBuilding(ctx context.Context, nodeID string, games []string) {
	for _, game := range games {
		if _, err := r.db.Exec(ctx, `
			INSERT INTO core.node_game_images ( node_id, game_slug, status, error, updated_at)
			VALUES ( $1, $2, 'building', NULL, now())
			ON CONFLICT (node_id, game_slug)
			DO UPDATE SET status = 'building', error = NULL, updated_at = now()
		`, nodeID, game); err != nil {
			log.Printf("сборка образов: состояние %s не записано: %v", game, err)
		}
	}
}

func (r *Runner) markImagesResult(ctx context.Context, nodeID string, games []string, errMsg string) {
	status := "ready"
	var errVal any
	if errMsg != "" {
		status = "failed"
		errVal = errMsg
	}
	for _, game := range games {
		if _, err := r.db.Exec(ctx, `
			INSERT INTO core.node_game_images ( node_id, game_slug, status, error, built_at, updated_at)
			VALUES ( $1, $2, $3, $4, CASE WHEN $3 = 'ready' THEN now() END, now())
			ON CONFLICT (node_id, game_slug) DO UPDATE SET
				status   = EXCLUDED.status,
				error    = EXCLUDED.error,
				built_at = COALESCE(EXCLUDED.built_at, core.node_game_images.built_at),
				updated_at = now()
		`, nodeID, game, status, errVal); err != nil {
			log.Printf("сборка образов: результат %s не записан: %v", game, err)
		}
	}
}
