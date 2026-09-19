package updates

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/vortanixapp/panel/pkg/buildinfo"
)

const (
	SettingPanelAuto        = "updates.panel.auto"
	SettingPanelWindowStart = "updates.panel.window_start"
	SettingPanelWindowEnd   = "updates.panel.window_end"
	SettingPanelSkipVersion = "updates.panel.auto_failed_version"
	SettingAgentsAuto       = "updates.agents.auto"

	DefaultAgentImage = "ghcr.io/vortanixapp/vortanix-agent:latest"

	AgentUpdateTimeout = 15 * time.Minute
)

func Env(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func Repo() string {
	if v, ok := os.LookupEnv("UPDATE_REPO"); ok {
		return strings.Trim(strings.TrimSpace(v), "/")
	}
	return "vortanixapp/panel"
}

type Release struct {
	Version     string `json:"version"`
	Notes       string `json:"notes"`
	URL         string `json:"url"`
	PublishedAt string `json:"published_at"`
	Prerelease  bool   `json:"prerelease"`
	CheckedAt   string `json:"checked_at"`
}

type githubRelease struct {
	TagName     string `json:"tag_name"`
	Body        string `json:"body"`
	HTMLURL     string `json:"html_url"`
	PublishedAt string `json:"published_at"`
	Prerelease  bool   `json:"prerelease"`
	Draft       bool   `json:"draft"`
}

func (g githubRelease) release(checkedAt string) Release {
	return Release{
		Version:     buildinfo.Normalize(g.TagName),
		Notes:       g.Body,
		URL:         g.HTMLURL,
		PublishedAt: g.PublishedAt,
		Prerelease:  g.Prerelease,
		CheckedAt:   checkedAt,
	}
}

func githubGet(ctx context.Context, repo, path string, out any) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	base := strings.TrimRight(Env("UPDATE_API_BASE", "https://api.github.com"), "/")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/repos/"+repo+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "vortanix-panel")
	if token := strings.TrimSpace(os.Getenv("UPDATE_GITHUB_TOKEN")); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("github недоступен: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound:
		return fmt.Errorf("у репозитория %s нет ни одного выпуска", repo)
	case http.StatusForbidden, http.StatusTooManyRequests:
		return fmt.Errorf("github ограничил частоту запросов, попробуйте позже")
	default:
		return fmt.Errorf("github ответил %s", resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func LatestRelease(ctx context.Context, repo string) (*Release, error) {
	var payload githubRelease
	if err := githubGet(ctx, repo, "/releases/latest", &payload); err != nil {
		return nil, err
	}
	rel := payload.release(time.Now().UTC().Format(time.RFC3339))
	return &rel, nil
}

func Releases(ctx context.Context, repo string, limit int) ([]Release, error) {
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	var payload []githubRelease
	if err := githubGet(ctx, repo, "/releases?per_page="+strconv.Itoa(limit), &payload); err != nil {
		return nil, err
	}
	checkedAt := time.Now().UTC().Format(time.RFC3339)
	out := make([]Release, 0, len(payload))
	for _, item := range payload {
		rel := item.release(checkedAt)
		if item.Draft || !IsSemver(rel.Version) {
			continue
		}
		out = append(out, rel)
	}
	sort.SliceStable(out, func(i, j int) bool { return IsNewer(out[i].Version, out[j].Version) })
	return out, nil
}

func SameVersion(a, b string) bool {
	return IsSemver(a) && IsSemver(b) && !IsNewer(a, b) && !IsNewer(b, a)
}

func Parse(v string) ([3]int, bool) {
	var out [3]int
	v = buildinfo.Normalize(v)
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i]
	}
	parts := strings.Split(v, ".")
	if v == "" || len(parts) > 3 {
		return out, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return out, false
		}
		out[i] = n
	}
	return out, true
}

func IsSemver(v string) bool {
	_, ok := Parse(v)
	return ok
}

func IsNewer(latest, current string) bool {
	l, okL := Parse(latest)
	c, okC := Parse(current)
	if !okL || !okC {
		return false
	}
	for i := 0; i < 3; i++ {
		if l[i] != c[i] {
			return l[i] > c[i]
		}
	}
	return false
}

func AgentImage(version string) string {
	image := Env("AGENT_IMAGE", DefaultAgentImage)
	if !IsSemver(version) {
		return image
	}
	name := image
	if i := strings.LastIndex(image, ":"); i > strings.LastIndex(image, "/") {
		name = image[:i]
	}
	return name + ":" + buildinfo.Normalize(version)
}

func AgentOutdated(agentVersion, target string) bool {
	if !IsSemver(target) {
		return false
	}
	if !IsSemver(agentVersion) {
		return true
	}
	return IsNewer(target, agentVersion)
}

func InWindow(now time.Time, start, end int) bool {
	if start < 0 || end < 0 || start > 23 || end > 23 || start == end {
		return true
	}
	h := now.Hour()
	if start < end {
		return h >= start && h < end
	}
	return h >= start || h < end
}

type Job struct {
	ID         string   `json:"id"`
	Target     string   `json:"target"`
	From       string   `json:"from"`
	State      string   `json:"state"`
	StartedAt  string   `json:"started_at,omitempty"`
	FinishedAt string   `json:"finished_at,omitempty"`
	ExitCode   int      `json:"exit_code"`
	Error      string   `json:"error,omitempty"`
	Log        []string `json:"log"`
}

type Status struct {
	Available bool   `json:"available"`
	Mode      string `json:"mode"`
	Reason    string `json:"reason,omitempty"`
	Project   string `json:"project,omitempty"`
	Job       *Job   `json:"job,omitempty"`
}

type Updater struct {
	URL    string
	Secret string
	http   *http.Client
}

var ErrUpdaterMissing = errors.New("служба обновления не подключена")

func NewUpdater() *Updater {
	return &Updater{
		URL:    strings.TrimRight(Env("UPDATER_URL", ""), "/"),
		Secret: Env("INTERNAL_SECRET", ""),
		http:   &http.Client{Timeout: 20 * time.Second},
	}
}

func (u *Updater) Configured() bool {
	return u != nil && u.URL != ""
}

func (u *Updater) do(ctx context.Context, method, path string, body, out any) error {
	if !u.Configured() {
		return ErrUpdaterMissing
	}
	var reader *bytes.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		reader = bytes.NewReader(raw)
	} else {
		reader = bytes.NewReader(nil)
	}
	req, err := http.NewRequestWithContext(ctx, method, u.URL+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Secret", u.Secret)
	resp, err := u.http.Do(req)
	if err != nil {
		return fmt.Errorf("служба обновления не отвечает: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		var e struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&e)
		if e.Error == "" {
			e.Error = resp.Status
		}
		return errors.New(e.Error)
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (u *Updater) Status(ctx context.Context) (*Status, error) {
	var out Status
	if err := u.do(ctx, http.MethodGet, "/internal/v1/status", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (u *Updater) Start(ctx context.Context, version string) (*Job, error) {
	var out Job
	if err := u.do(ctx, http.MethodPost, "/internal/v1/update", map[string]string{"version": version}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
