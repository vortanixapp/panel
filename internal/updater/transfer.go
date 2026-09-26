package updater

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/vortanixapp/panel/pkg/buildinfo"
	"github.com/vortanixapp/panel/pkg/dockerapi"
	"github.com/vortanixapp/panel/pkg/paneltransfer"
)

var transferEnvKeys = []string{
	"JWT_SECRET",
	"SECRETS_KEY",
	"VORTANIX_REGISTRY",
	"AGENT_IMAGE",
	"UPDATE_REPO",
	"UPDATE_GITHUB_TOKEN",
	"UPDATE_API_BASE",
	"SMTP_HOST",
	"SMTP_PORT",
	"SMTP_USER",
	"SMTP_PASS",
	"MAIL_FROM",
	"GOOGLE_CLIENT_ID",
	"GOOGLE_CLIENT_SECRET",
	"DISCORD_CLIENT_ID",
	"DISCORD_CLIENT_SECRET",
	"TELEGRAM_BOT_TOKEN",
}

const uploadsDir = "/app/uploads"

type transfer struct {
	applier *applier
	project *project
}

func (s *Server) transferContext(ctx context.Context) (*transfer, error) {
	p, err := s.discover(ctx)
	if err != nil {
		return nil, err
	}
	envFile := p.EnvFile
	if envFile == "" {
		envFile = path.Join(p.WorkingDir, ".env")
	}
	a := &applier{
		docker:  s.docker,
		project: p.Name,
		workDir: p.WorkingDir,
		root:    path.Dir(p.WorkingDir),
		envFile: envFile,
		files:   p.ConfigFiles,
		version: buildinfo.Current(),
	}
	return &transfer{applier: a, project: p}, nil
}

func (t *transfer) postgres(ctx context.Context) (*dockerapi.Container, string, string, error) {
	pg, err := t.applier.serviceContainer(ctx, "postgres", false)
	if err != nil {
		return nil, "", "", fmt.Errorf("docker недоступен: %w", err)
	}
	if pg == nil {
		return nil, "", "", errors.New("контейнер postgres в стеке не запущен")
	}
	user, db := "vortanix", ""
	if env, ok := pg.Config["Env"].([]any); ok {
		for _, item := range env {
			key, value, _ := strings.Cut(fmt.Sprint(item), "=")
			switch key {
			case "POSTGRES_USER":
				user = value
			case "POSTGRES_DB":
				db = value
			}
		}
	}
	if db == "" {
		db = user
	}
	return pg, user, db, nil
}

func (t *transfer) api(ctx context.Context) (*dockerapi.Container, error) {
	c, err := t.applier.serviceContainer(ctx, "api", true)
	if err != nil {
		return nil, fmt.Errorf("docker недоступен: %w", err)
	}
	if c == nil {
		return nil, errors.New("контейнер api в стеке не найден")
	}
	return c, nil
}

func output(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = minimalEnv()
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", errors.New(tail(stderr.String(), err))
	}
	return strings.TrimSpace(string(out)), nil
}

func (t *transfer) dbBytes(ctx context.Context) int64 {
	pg, user, db, err := t.postgres(ctx)
	if err != nil {
		return 0
	}
	out, err := output(ctx, "docker", "exec", pg.ID, "psql", "-U", user, "-d", db, "-tAc",
		"select pg_database_size(current_database())")
	if err != nil {
		return 0
	}
	n, _ := strconv.ParseInt(strings.TrimSpace(out), 10, 64)
	return n
}

func (t *transfer) uploadsBytes(ctx context.Context) int64 {
	api, err := t.api(ctx)
	if err != nil || !api.State.Running {
		return 0
	}
	out, err := output(ctx, "docker", "exec", api.ID, "du", "-sk", uploadsDir)
	if err != nil {
		return 0
	}
	field, _, _ := strings.Cut(strings.TrimSpace(out), "\t")
	n, _ := strconv.ParseInt(strings.TrimSpace(field), 10, 64)
	return n * 1024
}

func freeBytes(ctx context.Context, dir string) int64 {
	out, err := output(ctx, "df", "-kP", dir)
	if err != nil {
		return 0
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
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

func (t *transfer) envLines() []string {
	raw, err := os.ReadFile(t.applier.envFile)
	if err != nil {
		return nil
	}
	return strings.Split(string(raw), "\n")
}

func (t *transfer) envValues() map[string]string {
	out := map[string]string{}
	want := map[string]bool{}
	for _, k := range transferEnvKeys {
		want[k] = true
	}
	for _, line := range t.envLines() {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		key = strings.TrimSpace(key)
		if !ok || !want[key] {
			continue
		}
		if value = strings.TrimSpace(value); value != "" {
			out[key] = value
		}
	}
	return out
}

func (t *transfer) siteAddress() string {
	for _, line := range t.envLines() {
		if key, value, ok := strings.Cut(strings.TrimSpace(line), "="); ok && strings.TrimSpace(key) == "SITE_ADDRESS" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func (s *Server) transferProbeHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	t, err := s.transferContext(ctx)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": err.Error()})
		return
	}
	env := t.envValues()
	keys := make([]string, 0, len(env))
	for _, k := range transferEnvKeys {
		if _, ok := env[k]; ok {
			keys = append(keys, k)
		}
	}
	writeJSON(w, http.StatusOK, paneltransfer.Probe{
		Project:       t.project.Name,
		Mode:          t.project.Mode,
		Version:       buildinfo.Current(),
		SourceAddress: t.siteAddress(),
		DBBytes:       t.dbBytes(ctx),
		UploadsBytes:  t.uploadsBytes(ctx),
		FreeBytes:     freeBytes(ctx, t.applier.workDir),
		EnvKeys:       keys,
	})
}

func (s *Server) transferExportHandler(w http.ResponseWriter, r *http.Request) {
	t, err := s.transferContext(r.Context())
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.WriteHeader(http.StatusOK)
	if err := t.export(r.Context(), w); err != nil {
		_, _ = fmt.Fprintf(w, "\n%s%s\n", errorPrefix, err)
	}
}

func (t *transfer) export(ctx context.Context, dst io.Writer) error {
	pg, user, db, err := t.postgres(ctx)
	if err != nil {
		return err
	}
	api, err := t.api(ctx)
	if err != nil {
		return err
	}
	if !api.State.Running {
		return errors.New("контейнер api остановлен, выгрузка файлов невозможна")
	}

	out := paneltransfer.NewWriter(dst)
	part, err := out.Part(paneltransfer.PartEnv)
	if err != nil {
		return err
	}
	if err := json.NewEncoder(part).Encode(t.envValues()); err != nil {
		return err
	}

	part, err = out.Part(paneltransfer.PartDB)
	if err != nil {
		return err
	}
	if err := pipe(ctx, part, nil, "docker", "exec", pg.ID, "pg_dump",
		"-U", user, "-d", db, "-Fc", "--no-owner", "--no-privileges"); err != nil {
		return fmt.Errorf("дамп базы не снят: %w", err)
	}

	part, err = out.Part(paneltransfer.PartUploads)
	if err != nil {
		return err
	}
	if err := pipe(ctx, part, nil, "docker", "exec", api.ID, "tar", "-czf", "-",
		"--exclude=./.vxnew", "-C", uploadsDir, "."); err != nil {
		return fmt.Errorf("файлы панели не упакованы: %w", err)
	}
	return out.Close()
}

func pipe(ctx context.Context, stdout io.Writer, stdin io.Reader, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = minimalEnv()
	cmd.Stdout = stdout
	cmd.Stdin = stdin
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return errors.New(tail(stderr.String(), err))
	}
	return nil
}

func (s *Server) transferImportHandler(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	flusher, _ := w.(http.Flusher)
	log := func(line string) {
		_, _ = fmt.Fprintln(w, line)
		if flusher != nil {
			flusher.Flush()
		}
	}
	if err := s.ImportStream(r.Context(), r.Body, log); err != nil {
		log(errorPrefix + err.Error())
	}
}

func (s *Server) ImportStream(ctx context.Context, src io.Reader, log func(string)) error {
	t, err := s.transferContext(ctx)
	if err != nil {
		return err
	}
	pg, user, db, err := t.postgres(ctx)
	if err != nil {
		return err
	}
	api, err := t.api(ctx)
	if err != nil {
		return err
	}

	log("Останавливаю службы панели")
	if out, err := t.applier.compose(ctx, false, "stop", "api", "worker", "relay", "console"); err != nil {
		return fmt.Errorf("службы не остановлены: %s", tail(out, err))
	}

	in := paneltransfer.NewReader(src)
	seen := map[string]bool{}
	for {
		name, body, err := in.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		seen[name] = true
		switch name {
		case paneltransfer.PartEnv:
			log("Переношу секреты в " + path.Base(t.applier.envFile))
			if err := t.mergeEnv(body); err != nil {
				return err
			}
		case paneltransfer.PartDB:
			log("Восстанавливаю базу")
			if err := t.restoreDB(ctx, pg, user, db, body); err != nil {
				return err
			}
		case paneltransfer.PartUploads:
			log("Распаковываю файлы панели")
			if err := t.restoreUploads(ctx, api, body); err != nil {
				return err
			}
		default:
			_, _ = io.Copy(io.Discard, body)
		}
	}
	if !seen[paneltransfer.PartDB] {
		return errors.New("в потоке нет дампа базы")
	}

	log("Поднимаю службы панели")
	if out, err := t.applier.compose(ctx, false, "up", "-d"); err != nil {
		return fmt.Errorf("службы не поднялись: %s", tail(out, err))
	}
	log("Ожидание готовности API")
	if err := t.applier.waitHealthy(ctx); err != nil {
		return err
	}
	log("Данные панели перенесены")
	return nil
}

func (t *transfer) mergeEnv(body io.Reader) error {
	var values map[string]string
	if err := json.NewDecoder(io.LimitReader(body, 1<<20)).Decode(&values); err != nil {
		return errors.New("список секретов не читается")
	}
	allowed := map[string]bool{}
	for _, k := range transferEnvKeys {
		allowed[k] = true
	}
	lines := t.envLines()
	if lines == nil {
		return fmt.Errorf("%s не читается", t.applier.envFile)
	}
	applied := map[string]bool{}
	for i, line := range lines {
		key, _, ok := strings.Cut(strings.TrimSpace(line), "=")
		key = strings.TrimSpace(key)
		if !ok || !allowed[key] {
			continue
		}
		value, found := values[key]
		if !found {
			continue
		}
		lines[i] = key + "=" + value
		applied[key] = true
	}
	for _, key := range transferEnvKeys {
		value, found := values[key]
		if !found || applied[key] {
			continue
		}
		if n := len(lines); n > 0 && lines[n-1] == "" {
			lines = append(lines[:n-1], key+"="+value, "")
		} else {
			lines = append(lines, key+"="+value)
		}
	}
	tmp := t.applier.envFile + ".vxtmp"
	if err := os.WriteFile(tmp, []byte(strings.Join(lines, "\n")), 0o600); err != nil {
		return fmt.Errorf("%s не записан: %w", tmp, err)
	}
	if err := os.Rename(tmp, t.applier.envFile); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("%s не обновлён: %w", t.applier.envFile, err)
	}
	return nil
}

func (t *transfer) restoreDB(ctx context.Context, pg *dockerapi.Container, user, db string, body io.Reader) error {
	kill := "select pg_terminate_backend(pid) from pg_stat_activity where datname = current_database() and pid <> pg_backend_pid()"
	if err := pipe(ctx, io.Discard, nil, "docker", "exec", pg.ID, "psql", "-U", user, "-d", db, "-tAc", kill); err != nil {
		return fmt.Errorf("подключения к базе не закрыты: %w", err)
	}
	if err := pipe(ctx, io.Discard, body, "docker", "exec", "-i", pg.ID, "pg_restore",
		"--clean", "--if-exists", "--no-owner", "--no-privileges", "--single-transaction",
		"-U", user, "-d", db); err != nil {
		return fmt.Errorf("база не восстановлена: %w", err)
	}
	return nil
}

const uploadsRestoreScript = `set -e
rm -rf ` + uploadsDir + `/.vxnew
mkdir -p ` + uploadsDir + `/.vxnew
tar -xzf - -C ` + uploadsDir + `/.vxnew
cd ` + uploadsDir + `
for item in .vxnew/* .vxnew/.[!.]*; do
  [ -e "$item" ] || continue
  name=$(basename "$item")
  rm -rf "./$name"
  mv "$item" "./$name"
done
rm -rf ` + uploadsDir + `/.vxnew
`

func (t *transfer) restoreUploads(ctx context.Context, api *dockerapi.Container, body io.Reader) error {
	args := []string{"run", "--rm", "-i", "--volumes-from", api.ID,
		"--entrypoint", "sh", api.ConfigImage(), "-c", uploadsRestoreScript}
	if err := pipe(ctx, io.Discard, body, "docker", args...); err != nil {
		return fmt.Errorf("файлы панели не распакованы: %w", err)
	}
	return nil
}

func TransferImport() int {
	s := NewServer(os.Getenv("INTERNAL_SECRET"))
	stdout := bufio.NewWriter(os.Stdout)
	defer stdout.Flush()
	log := func(line string) {
		_, _ = fmt.Fprintln(stdout, line)
		_ = stdout.Flush()
	}
	if err := s.ImportStream(context.Background(), bufio.NewReaderSize(os.Stdin, 1<<20), log); err != nil {
		log(errorPrefix + err.Error())
		return 1
	}
	return 0
}
