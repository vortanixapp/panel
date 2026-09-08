package docker

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/vortanix/vortanix/pkg/gamecatalog"
)

func runDocker(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, "docker", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("%w: %s", err, msg)
	}
	return nil
}

func runDockerStreaming(ctx context.Context, sink io.Writer, args ...string) error {
	cmd := exec.CommandContext(ctx, "docker", args...)
	tail := &tailBuffer{limit: 2048}
	out := io.MultiWriter(sink, tail)
	cmd.Stdout = out
	cmd.Stderr = out
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(tail.String())
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("%w: %s", err, msg)
	}
	return nil
}

type tailBuffer struct {
	limit int
	buf   []byte
}

func (t *tailBuffer) Write(b []byte) (int, error) {
	t.buf = append(t.buf, b...)
	if len(t.buf) > t.limit {
		t.buf = t.buf[len(t.buf)-t.limit:]
	}
	return len(b), nil
}

func (t *tailBuffer) String() string { return string(t.buf) }

func pullImage(ctx context.Context, image string) error {
	return runDocker(ctx, "pull", image)
}

func ContainerName(serverID string) string {
	return "vortanix-" + strings.ReplaceAll(serverID, "-", "")
}

func Start(ctx context.Context, serverID, name, gameID string, limits map[string]any, dockerImage string, primaryPort int, bindIP string) error {
	cname := ContainerName(serverID)
	if running, _ := isRunning(ctx, cname); running {
		return nil
	}
	if exists(ctx, cname) {
		_ = runDocker(ctx, "rm", "-f", cname)
	}

	image := strings.TrimSpace(dockerImage)
	if image == "" {
		image = resolveImage(ctx, gameID)
	}
	if gameID != "test" && gameID != "" && image != "" {
		if !imageExists(ctx, image) {
			if err := pullImage(ctx, image); err != nil {
				return err
			}
		}
	}
	// Квоту выставляем до запуска: иначе сервер успевает подняться и начать
	// писать раньше, чем у каталога появится предел.
	ApplyDiskQuota(ctx, serverID, diskLimitMB(limits))

	args := buildRunArgs(serverID, gameID, limits, image, primaryPort, bindIP)
	if gameID == "test" || gameID == "" {
		args = append(args, "sleep", "infinity")
	}
	return runDocker(ctx, args...)
}

func resolveImage(ctx context.Context, gameID string) string {
	key := normalizeGame(gameID)
	runtime := gamecatalog.RuntimeImage(key)
	legacy := gameImage(gameID)
	if runtime == "" {
		return legacy
	}
	if imageExists(ctx, runtime) {
		return runtime
	}
	if legacy != "" && legacy != runtime && imageExists(ctx, legacy) {
		return legacy
	}
	return runtime
}

func buildRunArgs(serverID, gameID string, limits map[string]any, image string, primaryPort int, bindIP string) []string {
	cname := ContainerName(serverID)
	mem := memoryLimit(limits)
	args := []string{
		"run", "-d", "--name", cname,
		"--label", "vortanix.server_id=" + serverID,
		"--label", "vortanix.managed=true",
		"-m", mem,
		"--restart", "unless-stopped",
		// Содержимое /data пишет клиент по SFTP, а entrypoint запускает бинарь
		// оттуда же — то есть внутри контейнера исполняется его код, и это по
		// замыслу: иначе не поставить ни плагин, ни свою сборку сервера. Но
		// запас прочности до сих пор был нулевой, поэтому:
		//
		// no-new-privileges — залитый setuid-бинарь не поднимет права внутри
		// контейнера. Игровым серверам setuid-помощники не нужны.
		"--security-opt", "no-new-privileges",
		// MKNOD входит в набор по умолчанию, а игре не нужен никогда. Доступ к
		// устройствам и так режет cgroup, но создавать узлы устройств в
		// клиентском каталоге незачем.
		"--cap-drop", "MKNOD",
		// SYS_CHROOT и AUDIT_WRITE тоже из набора по умолчанию и тоже лишние.
		"--cap-drop", "SYS_CHROOT",
		"--cap-drop", "AUDIT_WRITE",
		// SYS_RESOURCE даёт право писать мимо дисковой квоты. Docker его в
		// наборе по умолчанию не выдаёт, но дисковый лимит — обещание клиенту,
		// и держаться оно должно на нашем требовании, а не на чужом умолчании,
		// которое может измениться с версией демона.
		"--cap-drop", "SYS_RESOURCE",
	}
	if cpu := cpuLimit(limits); cpu != "" {
		args = append(args, "--cpus", cpu)
	}
	args = append(args, gameRunOptions(serverID, gameID, limits, primaryPort, bindIP)...)
	args = append(args, image)
	return args
}

func gameRunOptions(serverID, gameID string, limits map[string]any, primaryPort int, bindIP string) []string {
	dataPath := serverDataDir(serverID)
	_ = os.MkdirAll(dataPath, 0o755)

	opts := []string{"-v", dataPath + ":/data"}

	// Порты, добавленные во вкладке «Порты», публикуются независимо от того,
	// знаком ли каталогу этот сервер: клиент их завёл сам, и без публикации
	// они не работают.
	taken := map[int]bool{}

	key := normalizeGame(gameID)
	game, known := gamecatalog.Resolve(key)
	if !known || primaryPort <= 0 {
		opts = append(opts, extraPortArgs(serverID, bindIP, taken)...)
		return append(opts, memoryEnvFor(key, limits)...)
	}

	if game.PortEnv != "" {
		opts = append(opts, "-e", fmt.Sprintf("%s=%d", game.PortEnv, primaryPort))
	}
	for k, v := range gamecatalog.RuntimeEnv(key) {
		opts = append(opts, "-e", k+"="+v)
	}
	bindPrefix := ""
	if ip := strings.TrimSpace(bindIP); ip != "" {
		bindPrefix = ip + ":"
	}
	for _, spec := range gamecatalog.PortLayout(key) {
		port := primaryPort + spec.Offset
		if port < 1 || port > 65535 {
			continue
		}
		taken[port] = true
		opts = append(opts, "-p", fmt.Sprintf("%s%d:%d/%s", bindPrefix, port, port, spec.Protocol))
	}
	opts = append(opts, extraPortArgs(serverID, bindIP, taken)...)

	return append(opts, memoryEnvFor(key, limits)...)
}

func memoryEnvFor(key string, limits map[string]any) []string {
	if !strings.HasSuffix(gamecatalog.Repository(key), "/mcjava") {
		return nil
	}
	env := []string{"-e", "EULA=TRUE"}
	if mb := memoryMB(limits); mb > 0 {
		heap := mb - 256
		if heap < 512 {
			heap = 512
		}
		env = append(env, "-e", fmt.Sprintf("MEMORY=%dM", heap))
	}
	return env
}

// ReapplyOwnership возвращает каталог сервера клиенту после того, как его
// перепахали от root.
//
// SteamCMD, распаковка архива версии и восстановление копии работают в
// контейнерах от root и оставляют файлы root:root. То есть каждая установка и
// каждое обновление откатывают рекурсивный chown, который воркер делает при
// выдаче SFTP-доступа, и клиент снова видит свои файлы, но не может их править.
//
// Владельца не задаём, а считываем с самого каталога сервера: его выставил
// воркер, и агенту не нужно знать ни имени доступа, ни его uid. Если доступа
// ещё нет, каталог принадлежит root и вызов ничего не меняет.
func ReapplyOwnership(ctx context.Context, serverID string) {
	dir := serverDataDir(serverID)
	const script = `o=$(stat -c '%u:%g' "$1") || exit 0
chown -h -R "$o" "$1" || true
chmod -R g+rwX "$1" || true`
	_ = exec.CommandContext(ctx, "sh", "-c", script, "sh", dir).Run()
}

func serverDataDir(serverID string) string {
	base := os.Getenv("VORTANIX_DATA_DIR")
	if base == "" {
		base = "/var/lib/vortanix/servers"
	}
	return filepath.Join(base, serverID)
}

func normalizeGame(gameID string) string {
	return gamecatalog.Normalize(gameID)
}

// diskLimitMB — место из тарифа в мегабайтах.
func diskLimitMB(limits map[string]any) int {
	if limits == nil {
		return 0
	}
	return IntFromPayload(limits["disk_mb"])
}

func memoryMB(limits map[string]any) int {
	if limits == nil {
		return 0
	}
	switch v := limits["memory_mb"].(type) {
	case float64:
		return int(v)
	case int:
		return v
	case int64:
		return int(v)
	default:
		return 0
	}
}

func cpuLimit(limits map[string]any) string {
	if limits == nil {
		return ""
	}
	var cores float64
	switch v := limits["cpu"].(type) {
	case float64:
		cores = v
	case int:
		cores = float64(v)
	case int64:
		cores = float64(v)
	}
	if cores <= 0 {
		return ""
	}
	return strconv.FormatFloat(cores, 'f', -1, 64)
}

func memoryLimit(limits map[string]any) string {
	mem := "512m"
	if mb := memoryMB(limits); mb > 0 {
		mem = fmt.Sprintf("%dm", mb)
	}
	return mem
}

func Stop(ctx context.Context, serverID string) error {
	return exec.CommandContext(ctx, "docker", "stop", ContainerName(serverID)).Run()
}

func Kill(ctx context.Context, serverID string) error {
	return exec.CommandContext(ctx, "docker", "kill", ContainerName(serverID)).Run()
}

func Restart(ctx context.Context, serverID, name, gameID string, limits map[string]any, dockerImage string, primaryPort int, bindIP string) error {
	_ = Stop(ctx, serverID)
	return Start(ctx, serverID, name, gameID, limits, dockerImage, primaryPort, bindIP)
}

func ContainerError(ctx context.Context, serverID string) string {
	cname := ContainerName(serverID)
	out, err := exec.CommandContext(ctx, "docker", "inspect", "-f",
		"{{if .State.Error}}{{.State.Error}}{{else if eq .State.Status \"exited\"}}exited with code {{.State.ExitCode}}{{end}}",
		cname).Output()
	if err != nil {
		return ""
	}
	if msg := strings.TrimSpace(string(out)); msg != "" {
		return msg
	}
	if msg := lastLogLines(ctx, cname); msg != "" {
		return msg
	}
	return ""
}

func lastLogLines(ctx context.Context, cname string) string {
	out, err := exec.CommandContext(ctx, "docker", "logs", "--tail", "20", cname).CombinedOutput()
	if err != nil && len(out) == 0 {
		return ""
	}
	seen := map[string]bool{}
	var uniq []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || seen[line] {
			continue
		}
		seen[line] = true
		uniq = append(uniq, line)
	}
	if len(uniq) == 0 {
		return ""
	}
	if len(uniq) > 5 {
		uniq = uniq[len(uniq)-5:]
	}
	return strings.Join(uniq, "; ")
}

func Destroy(ctx context.Context, serverID string) error {
	cname := ContainerName(serverID)
	_ = exec.CommandContext(ctx, "docker", "stop", cname).Run()
	return exec.CommandContext(ctx, "docker", "rm", "-f", cname).Run()
}

func Status(ctx context.Context, serverID string) string {
	return DetailedStatus(ctx, serverID)
}

func DetailedStatus(ctx context.Context, serverID string) string {
	cname := ContainerName(serverID)
	out, err := exec.CommandContext(ctx, "docker", "inspect", "-f", "{{.State.Status}}", cname).Output()
	if err != nil {
		return "stopped"
	}
	switch strings.TrimSpace(string(out)) {
	case "running":
		return "running"
	case "restarting", "dead", "exited":
		return "error"
	default:
		return "stopped"
	}
}

func isRunning(ctx context.Context, name string) (bool, error) {
	out, err := exec.CommandContext(ctx, "docker", "inspect", "-f", "{{.State.Running}}", name).Output()
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(out)) == "true", nil
}

func imageExists(ctx context.Context, image string) bool {
	return exec.CommandContext(ctx, "docker", "image", "inspect", image).Run() == nil
}

func exists(ctx context.Context, name string) bool {
	err := exec.CommandContext(ctx, "docker", "inspect", name).Run()
	return err == nil
}

func gameImage(gameID string) string {
	if img := gamecatalog.Image(gameID); img != "" {
		return img
	}
	return "alpine:3.20"
}
