package updater

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/vortanixapp/panel/pkg/buildinfo"
	"github.com/vortanixapp/panel/pkg/dockerapi"
	"github.com/vortanixapp/panel/pkg/updates"
)

const (
	labelJob     = "app.vortanix.update"
	labelProject = "app.vortanix.update.project"
	labelTarget  = "app.vortanix.update.target"
	labelFrom    = "app.vortanix.update.from"

	errorPrefix = "ОШИБКА: "
)

type Server struct {
	docker *dockerapi.Client
	secret string
	mu     sync.Mutex
}

func NewServer(secret string) *Server {
	return &Server{docker: dockerapi.New(), secret: secret}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /internal/v1/status", s.guard(s.status))
	mux.HandleFunc("POST /internal/v1/update", s.guard(s.update))
	return mux
}

func (s *Server) guard(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		got := r.Header.Get("X-Internal-Secret")
		if s.secret == "" || subtle.ConstantTimeCompare([]byte(got), []byte(s.secret)) != 1 {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
		next(w, r)
	}
}

type project struct {
	Name        string
	WorkingDir  string
	ConfigFiles []string
	EnvFile     string
	Image       string
	Socket      string
	Mode        string
}

func (s *Server) discover(ctx context.Context) (*project, error) {
	id := dockerapi.SelfContainerID()
	if id == "" {
		return nil, errors.New("служба обновления запущена не в контейнере")
	}
	self, err := s.docker.Inspect(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("нет доступа к docker: %w", err)
	}
	labels := self.Labels()
	p := &project{
		Name:       labels["com.docker.compose.project"],
		WorkingDir: labels["com.docker.compose.project.working_dir"],
		Image:      self.Image,
		Socket:     "/var/run/docker.sock",
		Mode:       "source",
	}
	for _, f := range strings.Split(labels["com.docker.compose.project.config_files"], ",") {
		if f = strings.TrimSpace(f); f != "" {
			p.ConfigFiles = append(p.ConfigFiles, f)
		}
	}
	if env := strings.Split(labels["com.docker.compose.project.environment_file"], ","); strings.TrimSpace(env[0]) != "" {
		p.EnvFile = strings.TrimSpace(env[0])
	}
	if binds, ok := self.HostConfig["Binds"].([]any); ok {
		for _, b := range binds {
			parts := strings.Split(fmt.Sprint(b), ":")
			if len(parts) >= 2 && parts[1] == "/var/run/docker.sock" {
				p.Socket = parts[0]
			}
		}
	}
	if p.Name == "" || p.WorkingDir == "" || len(p.ConfigFiles) == 0 {
		return p, errors.New("служба обновления запущена не через docker compose")
	}
	for _, f := range p.ConfigFiles {
		if path.Base(f) == "docker-compose.images.yml" {
			p.Mode = "images"
		}
	}
	return p, nil
}

func helperName(p *project) string {
	return p.Name + "-update"
}

func (s *Server) job(ctx context.Context, p *project) *updates.Job {
	c, err := s.docker.Inspect(ctx, helperName(p))
	if err != nil {
		return nil
	}
	labels := c.Labels()
	job := &updates.Job{
		ID:        c.ID[:12],
		Target:    labels[labelTarget],
		From:      labels[labelFrom],
		StartedAt: c.State.StartedAt,
		ExitCode:  c.State.ExitCode,
		Log:       []string{},
	}
	switch {
	case c.State.Running || c.State.Restarting || c.State.Status == "created":
		job.State = "running"
	case c.State.ExitCode == 0:
		job.State = "succeeded"
		job.FinishedAt = c.State.FinishedAt
	default:
		job.State = "failed"
		job.FinishedAt = c.State.FinishedAt
	}
	if lines, err := s.docker.Logs(ctx, c.ID, 400); err == nil {
		job.Log = lines
	}
	if job.State == "failed" {
		job.Error = c.State.Error
		for i := len(job.Log) - 1; i >= 0; i-- {
			if strings.HasPrefix(job.Log[i], errorPrefix) {
				job.Error = strings.TrimPrefix(job.Log[i], errorPrefix)
				break
			}
		}
		if job.Error == "" {
			job.Error = fmt.Sprintf("обновление завершилось с кодом %d", job.ExitCode)
		}
	}
	return job
}

func (s *Server) status(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	st := updates.Status{Mode: "unknown"}
	p, err := s.discover(ctx)
	switch {
	case err != nil:
		st.Reason = err.Error()
	case p.Mode != "images":
		st.Mode = p.Mode
		st.Project = p.Name
		st.Reason = "панель собрана из исходников: по кнопке обновляется только установка из готовых образов"
	default:
		st.Mode = p.Mode
		st.Project = p.Name
		st.Available = true
	}
	if p != nil && p.Name != "" {
		st.Job = s.job(ctx, p)
	}
	writeJSON(w, http.StatusOK, st)
}

func (s *Server) update(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	var body struct {
		Version string `json:"version"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	version := buildinfo.Normalize(body.Version)
	if !updates.IsSemver(version) {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "укажите версию выпуска, например 0.1.16"})
		return
	}

	p, err := s.discover(ctx)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": err.Error()})
		return
	}
	if p.Mode != "images" {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "панель собрана из исходников: обновите её командами на сервере"})
		return
	}
	if j := s.job(ctx, p); j != nil && j.State == "running" {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "обновление уже идёт"})
		return
	}

	name := helperName(p)
	if err := s.docker.Remove(ctx, name, true); err != nil && !dockerapi.IsNotFound(err) {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "прошлое обновление не убрано: " + err.Error()})
		return
	}

	from := buildinfo.Current()
	root := path.Dir(p.WorkingDir)
	cmd := []string{"apply", "--version", version, "--project", p.Name, "--workdir", p.WorkingDir, "--root", root}
	for _, f := range p.ConfigFiles {
		cmd = append(cmd, "--file", f)
	}
	if p.EnvFile != "" {
		cmd = append(cmd, "--env-file", p.EnvFile)
	}
	spec := map[string]any{
		"Image":      p.Image,
		"Cmd":        cmd,
		"User":       "0:0",
		"WorkingDir": p.WorkingDir,
		"Env":        []string{"VORTANIX_UPDATE_FROM=" + from},
		"Labels": map[string]string{
			labelJob: "1", labelProject: p.Name, labelTarget: version, labelFrom: from,
		},
		"HostConfig": map[string]any{
			"Binds":         []string{p.Socket + ":/var/run/docker.sock", root + ":" + root},
			"RestartPolicy": map[string]any{"Name": "no"},
		},
	}
	id, err := s.docker.Create(ctx, name, spec)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "обновление не запущено: " + err.Error()})
		return
	}
	if err := s.docker.Start(ctx, id); err != nil {
		_ = s.docker.Remove(ctx, id, true)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "обновление не запущено: " + err.Error()})
		return
	}
	job := s.job(ctx, p)
	if job == nil {
		job = &updates.Job{ID: id[:12], Target: version, From: from, State: "running", Log: []string{}}
	}
	writeJSON(w, http.StatusAccepted, job)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
