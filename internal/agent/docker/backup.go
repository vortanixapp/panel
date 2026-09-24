package docker

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

func Mkdir(ctx context.Context, serverID, dir string) error {
	target, err := resolveServerPath(dir)
	if err != nil {
		return err
	}
	if isServerRoot(target) {
		return nil
	}
	return mkdirOnHost(ctx, serverID, target)
}

func DeletePath(ctx context.Context, serverID, p string) error {
	target, err := resolveServerPath(p)
	if err != nil {
		return err
	}
	if isServerRoot(target) {
		return fmt.Errorf("нельзя удалить корень данных сервера")
	}
	return deleteOnHost(serverID, target)
}

func CreateBackup(ctx context.Context, serverID, name string) (string, int64, error) {
	cname := ContainerName(serverID)
	filename := strings.TrimSpace(name)
	if filename == "" {
		filename = fmt.Sprintf("backup-%d.tar.gz", time.Now().Unix())
	}
	if !strings.HasSuffix(filename, ".tar.gz") && !strings.HasSuffix(filename, ".zip") {
		filename += ".tar.gz"
	}
	archive := "/data/backups/" + filename
	script := fmt.Sprintf(
		"mkdir -p /data/backups && cd /data && tar -czf %s --exclude=backups .",
		shellQuote(archive),
	)
	if err := runCommand(exec.CommandContext(ctx, "docker", "exec", cname, "sh", "-c", script)); err != nil {
		return "", 0, err
	}
	sizeOut, err := exec.CommandContext(ctx, "docker", "exec", cname, "sh", "-c",
		fmt.Sprintf("wc -c < %s", shellQuote(archive))).Output()
	if err != nil {
		return filename, 0, nil
	}
	var size int64
	fmt.Sscanf(strings.TrimSpace(string(sizeOut)), "%d", &size)
	return filename, size, nil
}

func RestoreBackup(ctx context.Context, serverID, name string) error {
	name = strings.TrimSpace(name)
	if name == "" || strings.Contains(name, "/") || strings.Contains(name, "..") {
		return fmt.Errorf("invalid backup name")
	}
	cname := ContainerName(serverID)
	wasRunning, _ := isRunning(ctx, cname)
	if wasRunning {
		_ = exec.CommandContext(ctx, "docker", "stop", cname).Run()
	}
	if err := extractBackup(ctx, serverID, cname, name); err != nil {
		return err
	}
	ReapplyOwnership(ctx, serverID)
	if wasRunning {
		return exec.CommandContext(ctx, "docker", "start", cname).Run()
	}
	return nil
}

func extractBackup(ctx context.Context, serverID, cname, name string) error {
	dataDir := serverDataDir(serverID)
	archive := "/data/backups/" + name

	img := containerImage(ctx, cname)
	if img == "" {
		img = resolveImage(ctx, "")
	}
	if img == "" {
		return fmt.Errorf("не найден образ сервера для распаковки резервной копии")
	}
	script := fmt.Sprintf(
		"cd /data && tar --exclude=backups -xzof %s && "+
			"find /data -type f -perm /6000 -exec chmod a-s {} + 2>/dev/null || true",
		shellQuote(archive),
	)
	return runCommand(exec.CommandContext(ctx, "docker", "run", "--rm",
		"--network", "none",
		"--entrypoint", "sh",
		"-v", dataDir+":/data",
		img, "-c", script,
	))
}

func containerImage(ctx context.Context, cname string) string {
	out, err := exec.CommandContext(ctx, "docker", "inspect", "-f", "{{.Config.Image}}", cname).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
