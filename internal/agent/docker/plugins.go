package docker

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func ApplyPlugin(ctx context.Context, serverID string, payload map[string]any) error {
	action := strings.ToLower(StringFromPayload(payload["action"]))
	if action == "" {
		if enabled, ok := payload["enabled"].(bool); ok {
			return applyPluginToggle(ctx, serverID, payload, enabled)
		}
		return fmt.Errorf("action required")
	}

	switch action {
	case "install":
		return installPlugin(ctx, serverID, payload)
	case "uninstall":
		return uninstallPlugin(ctx, serverID, payload)
	default:
		return fmt.Errorf("unknown plugin action: %s", action)
	}
}

func applyPluginToggle(ctx context.Context, serverID string, payload map[string]any, enabled bool) error {
	actions, _ := payload["file_actions"].([]any)
	if len(actions) == 0 {
		return nil
	}
	if !enabled {
		if ua, ok := payload["uninstall_actions"].([]any); ok && len(ua) > 0 {
			return applyFileActions(ctx, serverID, ua)
		}
	}
	return applyFileActions(ctx, serverID, actions)
}

func installPlugin(ctx context.Context, serverID string, payload map[string]any) error {
	installPath := StringFromPayload(payload["install_path"])
	cname := ContainerName(serverID)

	archiveURL := StringFromPayload(payload["download_url"])
	if archiveURL == "" {
		archiveURL = StringFromPayload(payload["archive_url"])
	}
	cachePath := StringFromPayload(payload["archive_cache_path"])
	archiveType := strings.ToLower(StringFromPayload(payload["archive_type"]))
	if archiveType == "" {
		archiveType = "zip"
	}

	if archiveURL != "" || cachePath != "" {
		localArchive, err := resolveArchive(ctx, archiveURL, cachePath)
		if err != nil {
			return err
		}
		targetDir := "/data"
		if installPath != "" {
			targetDir = filepath.Join("/data", strings.TrimPrefix(installPath, "/"))
		}
		if err := extractArchiveToContainer(ctx, cname, localArchive, archiveType, targetDir); err != nil {
			return err
		}
	}

	if actions, ok := payload["file_actions"].([]any); ok && len(actions) > 0 {
		if err := applyFileActions(ctx, serverID, actions); err != nil {
			return err
		}
	}
	return nil
}

func uninstallPlugin(ctx context.Context, serverID string, payload map[string]any) error {
	if actions, ok := payload["uninstall_actions"].([]any); ok && len(actions) > 0 {
		return applyFileActions(ctx, serverID, actions)
	}
	installPath := StringFromPayload(payload["install_path"])
	if installPath == "" {
		return nil
	}
	cname := ContainerName(serverID)
	target := filepath.Join("/data", strings.TrimPrefix(installPath, "/"))
	script := "rm -rf " + shellQuote(target)
	out, err := exec.CommandContext(ctx, "docker", "exec", cname, "sh", "-c", script).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func ApplyMap(ctx context.Context, serverID, gameID string, payload map[string]any) error {
	action := strings.ToLower(StringFromPayload(payload["action"]))
	switch action {
	case "install":
		return installMap(ctx, serverID, payload)
	case "uninstall":
		return uninstallMap(ctx, serverID, payload)
	case "activate":
		mapName := StringFromPayload(payload["map_name"])
		if mapName == "" {
			mapName = StringFromPayload(payload["slug"])
		}
		if mapName == "" {
			return nil
		}
		_, err := ExecConsoleCommand(ctx, serverID, gameID, "changelevel "+mapName)
		return err
	default:
		return fmt.Errorf("unknown map action: %s", action)
	}
}

func installMap(ctx context.Context, serverID string, payload map[string]any) error {
	cachePath := StringFromPayload(payload["archive_cache_path"])
	downloadURL := StringFromPayload(payload["download_url"])
	archiveType := strings.ToLower(StringFromPayload(payload["archive_type"]))
	if archiveType == "" {
		archiveType = "zip"
	}
	if cachePath == "" && downloadURL == "" {
		if files, ok := payload["file_list"].([]any); ok && len(files) > 0 {
			return copyMapFiles(ctx, serverID, files)
		}
		return fmt.Errorf("archive required for map install")
	}
	localArchive, err := resolveArchive(ctx, downloadURL, cachePath)
	if err != nil {
		return err
	}
	cname := ContainerName(serverID)
	return extractArchiveToContainer(ctx, cname, localArchive, archiveType, "/data")
}

func uninstallMap(ctx context.Context, serverID string, payload map[string]any) error {
	paths, ok := payload["paths"].([]any)
	if !ok || len(paths) == 0 {
		if files, ok := payload["file_list"].([]any); ok {
			paths = files
		}
	}
	if len(paths) == 0 {
		return nil
	}
	cname := ContainerName(serverID)
	for _, p := range paths {
		rel := StringFromPayload(p)
		if rel == "" {
			continue
		}
		target := filepath.Join("/data", strings.TrimPrefix(rel, "/"))
		script := "rm -rf " + shellQuote(target)
		_, _ = exec.CommandContext(ctx, "docker", "exec", cname, "sh", "-c", script).CombinedOutput()
	}
	return nil
}

func copyMapFiles(ctx context.Context, serverID string, files []any) error {
	cname := ContainerName(serverID)
	for _, f := range files {
		m, ok := f.(map[string]any)
		if !ok {
			continue
		}
		src := StringFromPayload(m["src"])
		dst := StringFromPayload(m["dst"])
		if src == "" || dst == "" {
			continue
		}
		local, err := resolveArchive(ctx, "", src)
		if err != nil {
			return err
		}
		target := filepath.Join("/data", strings.TrimPrefix(dst, "/"))
		script := fmt.Sprintf("mkdir -p %s && cat > %s", shellQuote(filepath.Dir(target)), shellQuote(target))
		cmd := exec.CommandContext(ctx, "docker", "exec", "-i", cname, "sh", "-c", script)
		in, err := os.Open(local)
		if err != nil {
			return err
		}
		cmd.Stdin = in
		out, err := cmd.CombinedOutput()
		in.Close()
		if err != nil {
			return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
		}
	}
	return nil
}

// archiveCacheDir — где на ноде лежат архивы плагинов и карт.
func archiveCacheDir() string {
	return envOr("VORTANIX_PLUGIN_CACHE_DIR", "/opt/vortanix/plugin-cache")
}

// archiveCacheFile разворачивает путь в кэше и не выпускает за его пределы:
// значение приходит от панели, но по нему создаётся файл.
func archiveCacheFile(cachePath string) (string, error) {
	cachePath = strings.TrimSpace(cachePath)
	if cachePath == "" {
		return "", fmt.Errorf("cache_path required")
	}
	// Отказываем явно, а не полагаемся на filepath.Clean: тот схлопнул бы
	// «../../etc/passwd» в путь внутри кэша, и запись ушла бы по искажённому
	// адресу молча. Старый демон на такой путь отвечал ошибкой — и правильно.
	if strings.Contains(cachePath, "..") {
		return "", fmt.Errorf("invalid cache_path")
	}
	base, err := filepath.Abs(archiveCacheDir())
	if err != nil {
		return "", err
	}
	local, err := filepath.Abs(filepath.Join(base, filepath.Clean("/"+cachePath)))
	if err != nil {
		return "", err
	}
	if local != base && !strings.HasPrefix(local, base+string(os.PathSeparator)) {
		return "", fmt.Errorf("cache_path outside cache dir")
	}
	return local, nil
}

// FetchArchiveToCache кладёт архив каталога в кэш этой ноды.
//
// Панель зовёт это перед установкой плагина или карты: распаковка идёт здесь,
// на локации, поэтому архив нужен локально. Если файл уже на месте и сходится
// по размеру — не качаем: установка того же плагина на соседний сервер этой же
// ноды обходится без сети совсем.
func FetchArchiveToCache(ctx context.Context, payload map[string]any) (map[string]any, error) {
	cachePath := StringFromPayload(payload["cache_path"])
	url := StringFromPayload(payload["download_url"])
	size := int64(IntFromPayload(payload["size"]))

	local, err := archiveCacheFile(cachePath)
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return nil, fmt.Errorf("download_url required")
	}

	if info, err := os.Stat(local); err == nil && !info.IsDir() {
		if size <= 0 || info.Size() == size {
			return map[string]any{"cached": true, "reused": true, "path": cachePath}, nil
		}
	}
	if err := os.MkdirAll(filepath.Dir(local), 0o755); err != nil {
		return nil, err
	}

	// Качаем во временный файл рядом и переименовываем: оборванная закачка не
	// должна остаться в кэше под правильным именем — иначе следующая установка
	// возьмёт обрубок и распакует его молча.
	tmp := local + ".part"
	_ = os.Remove(tmp)
	out, err := exec.CommandContext(ctx, "curl", "-fsSL", "--retry", "2", "-o", tmp, url).CombinedOutput()
	if err != nil {
		_ = os.Remove(tmp)
		return nil, fmt.Errorf("download failed: %w: %s", err, strings.TrimSpace(string(out)))
	}
	if size > 0 {
		info, statErr := os.Stat(tmp)
		if statErr != nil || info.Size() != size {
			_ = os.Remove(tmp)
			return nil, fmt.Errorf("archive size mismatch: expected %d", size)
		}
	}
	if err := os.Rename(tmp, local); err != nil {
		_ = os.Remove(tmp)
		return nil, err
	}
	return map[string]any{"cached": true, "reused": false, "path": cachePath}, nil
}

// resolveArchive выбирает, откуда брать архив для установки.
//
// Кэш проверяется ПЕРВЫМ: панель доставляет архив на ноду отдельной командой,
// и качать его повторно на каждую установку — ровно то, ради чего кэш и
// заводился. Ссылка остаётся запасным путём.
func resolveArchive(ctx context.Context, url, cachePath string) (string, error) {
	if cachePath != "" {
		if local, err := archiveCacheFile(cachePath); err == nil {
			if info, statErr := os.Stat(local); statErr == nil && !info.IsDir() {
				return local, nil
			}
		}
	}
	if url != "" && (strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")) {
		tmp := filepath.Join(os.TempDir(), "vtx-plugin-"+filepath.Base(url))
		out, err := exec.CommandContext(ctx, "curl", "-fsSL", "-o", tmp, url).CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("download failed: %w: %s", err, strings.TrimSpace(string(out)))
		}
		return tmp, nil
	}
	if cachePath == "" {
		return "", fmt.Errorf("archive path required")
	}
	return "", fmt.Errorf("archive not found: %s", cachePath)
}

func extractArchiveToContainer(ctx context.Context, cname, localArchive, archiveType, targetDir string) error {
	hostTmp := filepath.Join(os.TempDir(), "vtx-extract-"+filepath.Base(localArchive))
	_ = os.RemoveAll(hostTmp)
	if err := os.MkdirAll(hostTmp, 0o755); err != nil {
		return err
	}
	defer os.RemoveAll(hostTmp)

	var extractCmd string
	switch archiveType {
	case "targz", "tar.gz":
		extractCmd = fmt.Sprintf("tar -xzf %s -C %s", shellQuote(localArchive), shellQuote(hostTmp))
	case "tar":
		extractCmd = fmt.Sprintf("tar -xf %s -C %s", shellQuote(localArchive), shellQuote(hostTmp))
	default:
		extractCmd = fmt.Sprintf("unzip -q %s -d %s", shellQuote(localArchive), shellQuote(hostTmp))
	}
	if out, err := exec.CommandContext(ctx, "sh", "-c", extractCmd).CombinedOutput(); err != nil {
		return fmt.Errorf("extract: %w: %s", err, strings.TrimSpace(string(out)))
	}

	tarPath := filepath.Join(os.TempDir(), "vtx-bundle.tar")
	_ = os.Remove(tarPath)
	if out, err := exec.CommandContext(ctx, "tar", "-cf", tarPath, "-C", hostTmp, ".").CombinedOutput(); err != nil {
		return fmt.Errorf("bundle: %w: %s", err, strings.TrimSpace(string(out)))
	}
	defer os.Remove(tarPath)

	f, err := os.Open(tarPath)
	if err != nil {
		return err
	}
	defer f.Close()
	script := "mkdir -p " + shellQuote(targetDir) + " && tar -xf - -C " + shellQuote(targetDir)
	cmd := exec.CommandContext(ctx, "docker", "exec", "-i", cname, "sh", "-c", script)
	cmd.Stdin = f
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("container extract: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func applyFileActions(ctx context.Context, serverID string, actions []any) error {
	cname := ContainerName(serverID)
	for _, item := range actions {
		a, ok := item.(map[string]any)
		if !ok {
			continue
		}
		action := strings.ToLower(StringFromPayload(a["action"]))
		path := StringFromPayload(a["path"])
		if path == "" {
			path = StringFromPayload(a["file"])
		}
		if path == "" {
			continue
		}
		target := filepath.Join("/data", strings.TrimPrefix(path, "/"))
		switch action {
		case "write", "create":
			content := fmt.Sprint(a["content"])
			script := "mkdir -p " + shellQuote(filepath.Dir(target)) + " && cat > " + shellQuote(target)
			cmd := exec.CommandContext(ctx, "docker", "exec", "-i", cname, "sh", "-c", script)
			cmd.Stdin = strings.NewReader(content)
			if out, err := cmd.CombinedOutput(); err != nil {
				return fmt.Errorf("write %s: %w: %s", path, err, strings.TrimSpace(string(out)))
			}
		case "delete", "remove":
			script := "rm -f " + shellQuote(target)
			if out, err := exec.CommandContext(ctx, "docker", "exec", cname, "sh", "-c", script).CombinedOutput(); err != nil {
				return fmt.Errorf("delete %s: %w: %s", path, err, strings.TrimSpace(string(out)))
			}
		case "append_lines", "prepend_lines":
			block := fmt.Sprint(a["content"])
			if lines, ok := a["lines"].([]any); ok {
				parts := make([]string, 0, len(lines))
				for _, l := range lines {
					parts = append(parts, fmt.Sprint(l))
				}
				block = strings.Join(parts, "\n")
			}
			readScript := "cat " + shellQuote(target) + " 2>/dev/null || true"
			raw, _ := exec.CommandContext(ctx, "docker", "exec", cname, "sh", "-c", readScript).Output()
			var out string
			if action == "append_lines" {
				out = strings.TrimRight(string(raw), "\n") + "\n" + block
			} else {
				out = block + "\n" + string(raw)
			}
			writeScript := "mkdir -p " + shellQuote(filepath.Dir(target)) + " && cat > " + shellQuote(target)
			cmd := exec.CommandContext(ctx, "docker", "exec", "-i", cname, "sh", "-c", writeScript)
			cmd.Stdin = strings.NewReader(out)
			if wo, err := cmd.CombinedOutput(); err != nil {
				return fmt.Errorf("append %s: %w: %s", path, err, strings.TrimSpace(string(wo)))
			}
		}
	}
	return nil
}
