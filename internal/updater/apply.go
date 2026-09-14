package updater

import (
	"bufio"
	"compress/gzip"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/vortanixapp/panel/pkg/dockerapi"
)

const keepBackups = 5

type stringList []string

func (s *stringList) String() string { return strings.Join(*s, ",") }

func (s *stringList) Set(v string) error {
	*s = append(*s, v)
	return nil
}

type applier struct {
	version    string
	from       string
	project    string
	workDir    string
	root       string
	envFile    string
	files      []string
	docker     *dockerapi.Client
	backupPath string
}

func Apply(args []string) int {
	a := &applier{docker: dockerapi.New(), from: os.Getenv("VORTANIX_UPDATE_FROM")}
	var files stringList
	fs := flag.NewFlagSet("apply", flag.ContinueOnError)
	fs.StringVar(&a.version, "version", "", "")
	fs.StringVar(&a.project, "project", "", "")
	fs.StringVar(&a.workDir, "workdir", "", "")
	fs.StringVar(&a.root, "root", "", "")
	fs.StringVar(&a.envFile, "env-file", "", "")
	fs.Var(&files, "file", "")
	if err := fs.Parse(args); err != nil {
		fmt.Printf("%s%v\n", errorPrefix, err)
		return 2
	}
	a.files = files
	if a.version == "" || a.project == "" || a.workDir == "" || len(a.files) == 0 {
		fmt.Printf("%sне хватает параметров обновления\n", errorPrefix)
		return 2
	}
	if a.root == "" {
		a.root = path.Dir(a.workDir)
	}
	if a.envFile == "" {
		a.envFile = path.Join(a.workDir, ".env")
	}
	if err := a.run(context.Background()); err != nil {
		fmt.Printf("%s%s\n", errorPrefix, err)
		return 1
	}
	return 0
}

func say(format string, args ...any) {
	fmt.Printf("%s  %s\n", time.Now().Format("15:04:05"), fmt.Sprintf(format, args...))
}

func (a *applier) run(ctx context.Context) error {
	from := a.from
	if from == "" {
		from = "?"
	}
	say("Обновление панели %s → %s", from, a.version)
	for _, f := range append(append([]string{}, a.files...), a.envFile) {
		if _, err := os.Stat(f); err != nil {
			return fmt.Errorf("не найден %s: каталог панели должен быть доступен по тому же пути, что и на сервере", f)
		}
	}

	if err := a.backup(ctx); err != nil {
		return err
	}

	git := a.syncGit()
	restoreEnv, err := a.pinVersion()
	if err != nil {
		git.restore()
		return err
	}

	say("Загрузка образов %s", a.version)
	if out, err := a.compose(ctx, false, "pull", "--quiet"); err != nil {
		restoreEnv()
		git.restore()
		return fmt.Errorf("образы версии %s не загрузились, панель осталась на прежней версии: %s", a.version, tail(out, err))
	}

	say("Перезапуск служб")
	if out, err := a.compose(ctx, true, "up", "-d"); err != nil {
		return fmt.Errorf("службы не перезапустились: %s. %s", tail(out, err), a.backupHint())
	}
	if git.caddyChanged {
		say("Caddyfile изменился, пересоздаю caddy")
		if out, err := a.compose(ctx, true, "up", "-d", "--force-recreate", "caddy"); err != nil {
			say("caddy не пересоздан: %s", tail(out, err))
		}
	}

	if err := a.waitHealthy(ctx); err != nil {
		return err
	}
	say("Панель обновлена до %s", a.version)
	if a.backupPath != "" {
		say("Копия базы до обновления: %s", a.backupPath)
	}
	return nil
}

func (a *applier) backupHint() string {
	if a.backupPath == "" {
		return "Копия базы не снималась"
	}
	return "Копия базы до обновления: " + a.backupPath
}

func minimalEnv() []string {
	out := []string{"GIT_TERMINAL_PROMPT=0"}
	hasHome := false
	for _, kv := range os.Environ() {
		key, _, _ := strings.Cut(kv, "=")
		switch {
		case key == "PATH", strings.HasPrefix(key, "DOCKER_"):
			out = append(out, kv)
		case key == "HOME":
			hasHome = true
			out = append(out, kv)
		}
	}
	if !hasHome {
		out = append(out, "HOME=/root")
	}
	return out
}

func (a *applier) compose(ctx context.Context, stream bool, args ...string) (string, error) {
	full := []string{"compose", "--ansi", "never", "-p", a.project, "--project-directory", a.workDir}
	for _, f := range a.files {
		full = append(full, "-f", f)
	}
	full = append(full, "--env-file", a.envFile)
	full = append(full, args...)

	cmd := exec.CommandContext(ctx, "docker", full...)
	cmd.Dir = a.workDir
	cmd.Env = minimalEnv()
	pr, pw := io.Pipe()
	cmd.Stdout = pw
	cmd.Stderr = pw

	var buf strings.Builder
	done := make(chan struct{})
	go func() {
		defer close(done)
		scanner := bufio.NewScanner(pr)
		scanner.Buffer(make([]byte, 64*1024), 1<<20)
		for scanner.Scan() {
			line := scanner.Text()
			buf.WriteString(line)
			buf.WriteByte('\n')
			if stream {
				fmt.Println("   " + line)
			}
		}
		_, _ = io.Copy(io.Discard, pr)
	}()
	err := cmd.Run()
	_ = pw.Close()
	<-done
	return buf.String(), err
}

func tail(out string, err error) string {
	lines := []string{}
	for _, l := range strings.Split(strings.TrimSpace(out), "\n") {
		if l = strings.TrimSpace(l); l != "" {
			lines = append(lines, l)
		}
	}
	if len(lines) > 5 {
		lines = lines[len(lines)-5:]
	}
	if len(lines) == 0 && err != nil {
		return err.Error()
	}
	return strings.Join(lines, "; ")
}

func (a *applier) serviceContainer(ctx context.Context, service string, all bool) (*dockerapi.Container, error) {
	list, err := a.docker.List(ctx, all, map[string][]string{"label": {
		"com.docker.compose.project=" + a.project,
		"com.docker.compose.service=" + service,
	}})
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, nil
	}
	return a.docker.Inspect(ctx, list[0].ID)
}

var unsafeName = regexp.MustCompile(`[^0-9A-Za-z.\-]+`)

func (a *applier) backup(ctx context.Context) error {
	pg, err := a.serviceContainer(ctx, "postgres", false)
	if err != nil {
		return fmt.Errorf("docker недоступен: %w", err)
	}
	if pg == nil {
		say("Контейнер postgres в стеке не найден, копия базы пропущена")
		return nil
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

	dir := path.Join(a.workDir, "backups")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("каталог копий %s не создан: %w", dir, err)
	}
	label := unsafeName.ReplaceAllString(a.from, "_")
	if label == "" {
		label = "unknown"
	}
	file := path.Join(dir, fmt.Sprintf("vortanix-%s-%s.sql.gz", label, time.Now().Format("20060102-150405")))
	say("Копия базы: %s", file)

	f, err := os.OpenFile(file, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("файл копии не создан: %w", err)
	}
	gz := gzip.NewWriter(f)
	cmd := exec.CommandContext(ctx, "docker", "exec", pg.ID, "pg_dump", "-U", user, "-d", db)
	cmd.Env = minimalEnv()
	cmd.Stdout = gz
	var stderr strings.Builder
	cmd.Stderr = &stderr
	runErr := cmd.Run()
	gzErr := gz.Close()
	fileErr := f.Close()
	if runErr != nil || gzErr != nil || fileErr != nil {
		_ = os.Remove(file)
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = errors.Join(runErr, gzErr, fileErr).Error()
		}
		return fmt.Errorf("копия базы не снята, обновление остановлено: %s", msg)
	}
	if info, err := os.Stat(file); err == nil {
		say("Копия готова, %.1f МБ", float64(info.Size())/(1<<20))
	}
	a.backupPath = file
	pruneBackups(dir)
	return nil
}

func pruneBackups(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	type item struct {
		name string
		mod  time.Time
	}
	items := []item{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasPrefix(e.Name(), "vortanix-") || !strings.HasSuffix(e.Name(), ".sql.gz") {
			continue
		}
		if info, err := e.Info(); err == nil {
			items = append(items, item{e.Name(), info.ModTime()})
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].mod.After(items[j].mod) })
	for i := keepBackups; i < len(items); i++ {
		_ = os.Remove(path.Join(dir, items[i].name))
	}
}

type gitState struct {
	a            *applier
	moved        bool
	oldHead      string
	branch       string
	caddyChanged bool
}

func (a *applier) git(args ...string) (string, error) {
	full := append([]string{"-c", "safe.directory=*", "-C", a.root}, args...)
	cmd := exec.Command("git", full...)
	cmd.Env = minimalEnv()
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

func firstLine(s string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(s), "\n")
	return line
}

func (g *gitState) restore() {
	if g == nil || !g.moved {
		return
	}
	if g.branch != "" {
		_, _ = g.a.git("reset", "-q", "--keep", g.oldHead)
	} else {
		_, _ = g.a.git("-c", "advice.detachedHead=false", "checkout", "-q", g.oldHead)
	}
	say("Рабочая копия возвращена на %s", g.oldHead[:min(12, len(g.oldHead))])
}

func (a *applier) syncGit() *gitState {
	g := &gitState{a: a}
	if _, err := os.Stat(path.Join(a.root, ".git")); err != nil {
		say("%s не git-репозиторий, файлы развёртывания остаются как есть", a.root)
		return g
	}
	say("Получение выпуска из git")
	if out, err := a.git("fetch", "--quiet", "--force", "--tags", "origin"); err != nil {
		say("git fetch не прошёл (%s), файлы развёртывания остаются как есть", firstLine(out))
		return g
	}
	tag := ""
	for _, candidate := range []string{"v" + a.version, a.version} {
		if _, err := a.git("rev-parse", "-q", "--verify", "refs/tags/"+candidate+"^{commit}"); err == nil {
			tag = candidate
			break
		}
	}
	if tag == "" {
		say("В репозитории нет тега v%s, файлы развёртывания остаются как есть", a.version)
		return g
	}
	if out, _ := a.git("status", "--porcelain", "--untracked-files=no"); out != "" {
		say("В рабочей копии есть правки, файлы развёртывания остаются как есть:")
		for _, line := range strings.Split(out, "\n") {
			say("   %s", line)
		}
		return g
	}
	head, err := a.git("rev-parse", "HEAD")
	if err != nil {
		say("Не удалось прочитать HEAD (%s), файлы развёртывания остаются как есть", firstLine(head))
		return g
	}
	if _, err := a.git("merge-base", "--is-ancestor", tag, "HEAD"); err == nil {
		say("Рабочая копия уже содержит %s", tag)
		return g
	}
	branch, _ := a.git("symbolic-ref", "-q", "--short", "HEAD")
	if branch != "" {
		if out, err := a.git("merge", "--ff-only", "-q", tag); err != nil {
			say("Ветка %s разошлась с %s (%s), файлы развёртывания остаются как есть", branch, tag, firstLine(out))
			return g
		}
	} else if out, err := a.git("-c", "advice.detachedHead=false", "checkout", "-q", tag); err != nil {
		say("Переход на %s не удался (%s), файлы развёртывания остаются как есть", tag, firstLine(out))
		return g
	}
	g.moved, g.oldHead, g.branch = true, head, branch
	if changed, err := a.git("diff", "--name-only", head, "HEAD"); err == nil {
		for _, f := range strings.Split(changed, "\n") {
			if path.Base(strings.TrimSpace(f)) == "Caddyfile" {
				g.caddyChanged = true
			}
		}
	}
	say("Рабочая копия переведена на %s", tag)
	return g
}

func (a *applier) pinVersion() (func(), error) {
	info, err := os.Stat(a.envFile)
	if err != nil {
		return nil, fmt.Errorf("%s недоступен: %w", a.envFile, err)
	}
	original, err := os.ReadFile(a.envFile)
	if err != nil {
		return nil, fmt.Errorf("%s не читается: %w", a.envFile, err)
	}
	entry := "VORTANIX_VERSION=" + a.version
	lines := strings.Split(string(original), "\n")
	found := false
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "VORTANIX_VERSION=") {
			lines[i] = entry
			found = true
		}
	}
	if !found {
		if n := len(lines); n > 0 && lines[n-1] == "" {
			lines = append(lines[:n-1], entry, "")
		} else {
			lines = append(lines, entry)
		}
	}
	if err := os.WriteFile(a.envFile, []byte(strings.Join(lines, "\n")), info.Mode().Perm()); err != nil {
		return nil, fmt.Errorf("%s не записан: %w", a.envFile, err)
	}
	say("В %s закреплена версия: %s", path.Base(a.envFile), entry)
	return func() {
		if err := os.WriteFile(a.envFile, original, info.Mode().Perm()); err == nil {
			say("%s возвращён к прежнему виду", path.Base(a.envFile))
		}
	}, nil
}

func (a *applier) waitHealthy(ctx context.Context) error {
	say("Ожидание готовности API")
	deadline := time.Now().Add(5 * time.Minute)
	for {
		c, err := a.serviceContainer(ctx, "api", true)
		if err == nil && c != nil && c.State.Running && (c.State.Health == nil || c.State.Health.Status == "healthy") {
			say("API отвечает, образ %s", c.ConfigImage())
			return nil
		}
		if time.Now().After(deadline) {
			if c != nil {
				if lines, err := a.docker.Logs(ctx, c.ID, 30); err == nil {
					say("Последние строки журнала API:")
					for _, line := range lines {
						fmt.Println("   " + line)
					}
				}
			}
			return fmt.Errorf("API не поднялся за 5 минут. %s", a.backupHint())
		}
		time.Sleep(3 * time.Second)
	}
}
