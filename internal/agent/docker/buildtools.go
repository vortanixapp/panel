package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	sourceBuildTools  = "buildtools"
	buildToolsURL     = "https://hub.spigotmc.org/jenkins/job/BuildTools/lastSuccessfulBuild/artifact/target/BuildTools.jar"
	spigotVersionsURL = "https://hub.spigotmc.org/versions/"
	buildToolsMemory  = "3g"
)

var spigotRevision = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,31}$`)

var buildJDKs = []int{25, 21, 17, 8}

func spigotJDK(ctx context.Context, rev string) int {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, spigotVersionsURL+rev+".json", nil)
	if err != nil {
		return buildJDKs[0]
	}
	req.Header.Set("User-Agent", downloadUserAgent)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return buildJDKs[0]
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return buildJDKs[0]
	}
	var info struct {
		JavaVersions []int `json:"javaVersions"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&info); err != nil {
		return buildJDKs[0]
	}
	if len(info.JavaVersions) < 2 {
		return 8
	}
	for _, jdk := range buildJDKs {
		class := jdk + 44
		if class >= info.JavaVersions[0] && class <= info.JavaVersions[1] {
			return jdk
		}
	}
	return buildJDKs[0]
}

func installBuildTools(ctx context.Context, dataDir, version string, report ProgressFunc) error {
	if report == nil {
		report = noopProgress
	}
	rev := strings.TrimSpace(version)
	if rev == "" {
		rev = "latest"
	}
	if !spigotRevision.MatchString(rev) {
		return fmt.Errorf("версия %q не подходит для BuildTools", version)
	}

	image := fmt.Sprintf("eclipse-temurin:%d-jdk", spigotJDK(ctx, rev))
	report(StageBuild, 0, "Подготовка сборки Spigot", image)
	if !imageExists(ctx, image) {
		if err := pullImage(ctx, image); err != nil {
			return fmt.Errorf("образ %s: %w", image, explainPullError(err))
		}
	}

	toolsDir := filepath.Join(dataDir, ".vtx", "buildtools")
	if err := os.MkdirAll(toolsDir, 0o755); err != nil {
		return fmt.Errorf("каталог BuildTools: %w", err)
	}
	defer os.RemoveAll(toolsDir)

	report(StageBuild, 5, "Скачивание BuildTools", "")
	if _, err := download(ctx, buildToolsURL, filepath.Join(toolsDir, "BuildTools.jar"), noopProgress); err != nil {
		return err
	}

	report(StageBuild, 10, "Сборка Spigot "+rev+", это занимает 5–15 минут", "")
	script := "set -e; apt-get update -qq; apt-get install -y -qq --no-install-recommends git >/dev/null; " +
		"mkdir -p /build; cd /build; java -jar /data/.vtx/buildtools/BuildTools.jar --rev \"$REV\" --output-dir /data --final-name server.jar"
	build := func() error {
		sink := newLineScanner(func(line string) {
			report(StageBuild, -1, "", line)
		})
		defer sink.Close()
		return runDockerStreaming(ctx, sink,
			"run", "--rm",
			"-m", buildToolsMemory,
			"-v", dataDir+":/data",
			"-e", "REV="+rev,
			image, "sh", "-c", script)
	}
	if err := build(); err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("сборка Spigot %s: %w", rev, err)
		}
		report(StageBuild, 10, "Повторная попытка сборки Spigot "+rev, "")
		if err := build(); err != nil {
			return fmt.Errorf("сборка Spigot %s: %w", rev, err)
		}
	}
	if _, err := os.Stat(filepath.Join(dataDir, "server.jar")); err != nil {
		return fmt.Errorf("BuildTools не собрал server.jar")
	}
	report(StageDone, 100, "Spigot собран", "")
	return nil
}
