package jobs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/vortanixapp/panel/pkg/buildinfo"
	"github.com/vortanixapp/panel/pkg/sshclient"
	"github.com/vortanixapp/panel/pkg/updates"
)

const (
	panelTransferDir       = "/opt/vortanix"
	panelTransferReserve   = 5 << 30
	panelTransferHealthFor = 4 * time.Minute
)

func (r *Runner) runPanelTransfer(ctx context.Context, rec *panelTransferRecord) error {
	if rec.Mode != "ssh" && rec.Mode != "agents" {
		return errors.New("этот режим переноса выполняется панелью, а не воркером")
	}
	lg := r.newPanelTransferLog(ctx, rec.ID)
	defer lg.flush()

	if rec.Mode == "agents" {
		if err := r.switchAgents(ctx, rec, rec.sshConfig(), lg, strings.TrimSpace(rec.NewAddress), "images"); err != nil {
			return err
		}
		r.finishPanelTransfer(ctx, rec.ID, "done")
		return nil
	}

	upd := updates.NewUpdater()
	if !upd.Configured() {
		return errors.New("служба обновления не подключена, перенос невозможен")
	}

	r.setPanelTransferStage(ctx, rec.ID, "prepare")
	probe, err := upd.TransferProbe(ctx)
	if err != nil {
		return fmt.Errorf("состав данных не получен: %w", err)
	}
	total := probe.DBBytes + probe.UploadsBytes
	r.setPanelTransferBytes(ctx, rec.ID, 0, total)
	lg.say("База %s, файлы %s", humanBytes(probe.DBBytes), humanBytes(probe.UploadsBytes))

	address := strings.TrimSpace(rec.NewAddress)
	if rec.SameAddress {
		address = strings.TrimSpace(probe.SourceAddress)
	}
	if address == "" {
		return errors.New("не задан адрес новой панели")
	}

	if rec.FreezeWrites {
		r.setPanelTransferStage(ctx, rec.ID, "freeze")
		r.setPanelFreeze(ctx, true)
		lg.say("Панель переведена в режим только чтение")
	}

	cfg := rec.sshConfig()

	r.setPanelTransferStage(ctx, rec.ID, "probe_target")
	if err := r.probePanelTarget(cfg, lg, total); err != nil {
		return err
	}
	if r.cancelRequested(ctx, rec.ID) {
		return errors.New("перенос отменён")
	}

	r.setPanelTransferStage(ctx, rec.ID, "install_docker")
	lg.say("Устанавливаю Docker на новом сервере")
	if err := sshclient.Run(cfg, dockerCommands(), lg); err != nil {
		return fmt.Errorf("docker не установлен: %w", err)
	}

	version := buildinfo.Current()
	r.setPanelTransferStage(ctx, rec.ID, "clone")
	lg.say("Копирую панель версии %s в %s", version, panelTransferDir)
	if err := sshclient.Run(cfg, panelCloneCommands(version), lg); err != nil {
		return fmt.Errorf("файлы панели не скопированы: %w", err)
	}

	r.setPanelTransferStage(ctx, rec.ID, "init_env")
	lg.say("Готовлю настройки для адреса %s", address)
	if err := sshclient.Run(cfg, panelInitEnvCommands(address), lg); err != nil {
		return fmt.Errorf("настройки новой панели не созданы: %w", err)
	}

	r.setPanelTransferStage(ctx, rec.ID, "compose_up")
	lg.say("Поднимаю панель на новом сервере")
	if err := sshclient.Run(cfg, panelComposeUpCommands(probe.Mode), lg); err != nil {
		return fmt.Errorf("панель на новом сервере не поднялась: %w", err)
	}
	if err := r.waitPanelTargetHealth(cfg, lg, probe.Mode); err != nil {
		return err
	}

	if r.cancelRequested(ctx, rec.ID) {
		return errors.New("перенос отменён")
	}

	r.setPanelTransferStage(ctx, rec.ID, "transfer")
	lg.say("Передаю базу и файлы")
	if err := r.streamPanelData(ctx, rec, upd, cfg, lg, total, probe.Mode); err != nil {
		return err
	}

	r.setPanelTransferStage(ctx, rec.ID, "health")
	if err := r.waitPanelTargetHealth(cfg, lg, probe.Mode); err != nil {
		return err
	}
	lg.say("Новая панель отвечает по адресу %s", address)

	if rec.SameAddress {
		lg.say("Адрес панели не менялся: узлы подключатся к новой панели сами после переключения DNS")
	} else if err := r.switchAgents(ctx, rec, cfg, lg, address, probe.Mode); err != nil {
		return err
	}

	lg.say("Переключите DNS на новый сервер. Старая панель осталась в режиме только чтение — выключите её вручную, когда убедитесь, что всё на месте")

	r.finishPanelTransfer(ctx, rec.ID, "done")
	return nil
}

func humanBytes(n int64) string {
	switch {
	case n >= 1<<30:
		return strconv.FormatFloat(float64(n)/(1<<30), 'f', 1, 64) + " ГБ"
	case n >= 1<<20:
		return strconv.FormatFloat(float64(n)/(1<<20), 'f', 1, 64) + " МБ"
	default:
		return strconv.FormatInt(n/1024, 10) + " КБ"
	}
}

func (r *Runner) probePanelTarget(cfg sshclient.Config, lg *panelTransferLog, total int64) error {
	out, err := sshclient.Probe(cfg, "cat /etc/os-release; echo ---; uname -m; echo ---; df -kP /; echo ---; (ss -ltn || netstat -ltn) 2>/dev/null")
	if err != nil {
		if sshclient.IsHostKeyError(err) {
			return err
		}
		return fmt.Errorf("новый сервер не отвечает: %w", err)
	}
	blocks := strings.Split(out, "---")
	if len(blocks) < 4 {
		return errors.New("новый сервер ответил непонятно, проверьте доступ")
	}
	osInfo := strings.ToLower(blocks[0])
	if !strings.Contains(osInfo, "debian") && !strings.Contains(osInfo, "ubuntu") {
		return errors.New("на новом сервере нужна Debian или Ubuntu")
	}
	arch := strings.TrimSpace(blocks[1])
	if arch != "x86_64" && arch != "aarch64" {
		return fmt.Errorf("архитектура %s не поддерживается", arch)
	}
	need := total*3 + panelTransferReserve
	free := freeFromDF(blocks[2])
	if free > 0 && free < need {
		return fmt.Errorf("на новом сервере свободно %s, а нужно не меньше %s", humanBytes(free), humanBytes(need))
	}
	for _, port := range []string{":80", ":443"} {
		if strings.Contains(blocks[3], port+" ") {
			return fmt.Errorf("порт %s на новом сервере занят", strings.TrimPrefix(port, ":"))
		}
	}
	lg.say("Новый сервер проверен: %s, свободно %s", arch, humanBytes(free))
	return nil
}

func freeFromDF(block string) int64 {
	lines := strings.Split(strings.TrimSpace(block), "\n")
	if len(lines) < 2 {
		return 0
	}
	fields := strings.Fields(lines[len(lines)-1])
	if len(fields) < 4 {
		return 0
	}
	n, _ := strconv.ParseInt(fields[3], 10, 64)
	return n * 1024
}

func panelCloneCommands(version string) []string {
	repo := "https://github.com/" + updates.Repo() + ".git"
	tag := "v" + buildinfo.Normalize(version)
	return []string{
		"sudo apt-get update -y",
		"sudo apt-get install -y git curl openssl",
		fmt.Sprintf("sudo test -d %s/.git || sudo git clone --depth 1 --branch %s %s %s",
			panelTransferDir, tag, repo, panelTransferDir),
		fmt.Sprintf("cd %s && sudo git fetch --depth 1 origin tag %s && sudo git checkout -q %s", panelTransferDir, tag, tag),
	}
}

func panelAddressHost(address string) string {
	host := address
	for _, scheme := range []string{"https://", "http://", "wss://", "ws://"} {
		host = strings.TrimPrefix(host, scheme)
	}
	if idx := strings.IndexAny(host, "/"); idx >= 0 {
		host = host[:idx]
	}
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	return strings.Trim(host, "[]")
}

func panelAddressIsIP(address string) bool {
	return net.ParseIP(panelAddressHost(address)) != nil
}

func panelInitEnvCommands(address string) []string {
	arg := panelAddressHost(address)
	if arg == "" || panelAddressIsIP(address) {
		arg = "--ip"
	}
	return []string{
		fmt.Sprintf("cd %s && sudo sh scripts/init-env.sh %s", panelTransferDir, shellArg(arg)),
	}
}

func panelComposeCommand(mode string) string {
	if mode == "images" {
		return "sudo docker compose -f deploy/docker-compose.yml -f deploy/docker-compose.images.yml"
	}
	return "sudo docker compose -f deploy/docker-compose.yml"
}

func panelComposeUpCommands(mode string) []string {
	return []string{
		fmt.Sprintf("cd %s && %s up -d", panelTransferDir, panelComposeCommand(mode)),
	}
}

func shellArg(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func (r *Runner) waitPanelTargetHealth(cfg sshclient.Config, lg *panelTransferLog, mode string) error {
	check := fmt.Sprintf("cd %s && %s exec -T api wget -qO- http://127.0.0.1:8080/health 2>/dev/null || curl -fsS -m 5 http://127.0.0.1/health 2>/dev/null",
		panelTransferDir, panelComposeCommand(mode))
	deadline := time.Now().Add(panelTransferHealthFor)
	for {
		out, err := sshclient.RunCapture(cfg, check)
		if err == nil && strings.Contains(out, "ok") {
			return nil
		}
		if time.Now().After(deadline) {
			lg.say("Новая панель не ответила на проверку готовности")
			return errors.New("новая панель не поднялась за отведённое время")
		}
		time.Sleep(5 * time.Second)
	}
}

func (r *Runner) streamPanelData(ctx context.Context, rec *panelTransferRecord, upd *updates.Updater,
	cfg sshclient.Config, lg *panelTransferLog, total int64, mode string) error {
	body, err := upd.TransferExport(ctx)
	if err != nil {
		return fmt.Errorf("данные панели не выгружены: %w", err)
	}
	defer body.Close()

	last := time.Now()
	counted := &countingReader{src: body, onCount: func(n int64) {
		if time.Since(last) < 2*time.Second {
			return
		}
		last = time.Now()
		r.setPanelTransferBytes(ctx, rec.ID, n, total)
	}}

	cmd := fmt.Sprintf("cd %s && %s exec -T updater vortanix transfer-import", panelTransferDir, panelComposeCommand(mode))
	sent, err := sshclient.StreamToLog(cfg, cmd, counted, lg)
	r.setPanelTransferBytes(ctx, rec.ID, sent, total)
	if err != nil {
		return fmt.Errorf("данные не доехали: %w", err)
	}
	lg.say("Передано %s", humanBytes(sent))
	return nil
}

type countingReader struct {
	src     io.Reader
	total   int64
	onCount func(int64)
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.src.Read(p)
	if n > 0 {
		c.total += int64(n)
		if c.onCount != nil {
			c.onCount(c.total)
		}
	}
	return n, err
}
