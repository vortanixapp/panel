package docker

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
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
	return execInServer(ctx, serverID, target, `mkdir -p "$R"`).Run()
}

func DeletePath(ctx context.Context, serverID, p string) error {
	target, err := resolveServerPath(p)
	if err != nil {
		return err
	}
	if isServerRoot(target) {
		return fmt.Errorf("нельзя удалить корень данных сервера")
	}
	return execInServer(ctx, serverID, target, `rm -rf "$R"`).Run()
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
	if err := exec.CommandContext(ctx, "docker", "exec", cname, "sh", "-c", script).Run(); err != nil {
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
	// Распаковка шла от root и вернула файлам владельца root — возвращаем
	// каталог клиенту, иначе после восстановления копии он теряет доступ к
	// собственным файлам.
	ReapplyOwnership(ctx, serverID)
	if wasRunning {
		return exec.CommandContext(ctx, "docker", "start", cname).Run()
	}
	return nil
}

// extractBackup распаковывает копию, считая её содержимое ЧУЖИМ.
//
// Архивы лежат в /data/backups, а в /data клиент пишет по SFTP — значит он
// может положить туда собственный tar.gz и попросить восстановление. Раньше
// распаковка шла на хосте от root: tar в этом случае переносит владельца из
// архива, сохраняет setuid-биты, а выход через ../ и симлинк наружу ограничен
// только добросовестностью самого tar — у агента это busybox из alpine, где на
// такую добросовестность рассчитывать не стоит.
//
// Поэтому распаковываем внутри одноразового контейнера с примонтированным
// каталогом сервера: и обход каталогов, и симлинк упираются в его собственную
// файловую систему, которая тут же исчезает. Плюс -o, чтобы владелец брался не
// из архива.
func extractBackup(ctx context.Context, serverID, cname, name string) error {
	dataDir := serverDataDir(serverID)
	archive := "/data/backups/" + name

	if img := containerImage(ctx, cname); img != "" {
		script := fmt.Sprintf(
			"cd /data && tar --exclude=backups -xzof %s && "+
				// Снимаем setuid и setgid с файлов из архива. Каталогов не
				// касаемся: setgid на самом /data нужен, чтобы новые файлы
				// получали группу сервера.
				"find /data -type f -perm /6000 -exec chmod a-s {} + 2>/dev/null || true",
			shellQuote(archive),
		)
		return exec.CommandContext(ctx, "docker", "run", "--rm",
			"--network", "none",
			"--entrypoint", "sh",
			"-v", dataDir+":/data",
			img, "-c", script,
		).Run()
	}

	// Контейнера ещё нет — распаковываем на хосте, но хотя бы без переноса
	// владельца. Снятие setuid здесь не делаем: find у busybox не понимает
	// -perm /MODE, а без своего владельца setuid-файл в каталоге сервера ничего
	// не даёт — с хоста его исполнять некому, у файловых учёток nologin.
	return exec.CommandContext(ctx, "tar", "--exclude=backups", "-xzof",
		filepath.Join(dataDir, "backups", name), "-C", dataDir).Run()
}

// containerImage отдаёт образ, в котором работает сервер. Остановленный
// контейнер inspect тоже видит, поэтому образ находится и после docker stop.
func containerImage(ctx context.Context, cname string) string {
	out, err := exec.CommandContext(ctx, "docker", "inspect", "-f", "{{.Config.Image}}", cname).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
