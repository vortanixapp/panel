package checks

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	StatusOperational = "operational"
	StatusDegraded    = "degraded"
	StatusDown        = "down"
)

func Severity(status string) int {
	switch status {
	case StatusDown:
		return 2
	case StatusDegraded:
		return 1
	default:
		return 0
	}
}

func Worse(a, b string) string {
	if Severity(b) > Severity(a) {
		return b
	}
	return a
}

type Target struct {
	Key    string `json:"key"`
	Name   string `json:"name"`
	URL    string `json:"-"`
	Path   string `json:"-"`
	Public bool   `json:"-"`
}

type Result struct {
	Key       string `json:"key"`
	Name      string `json:"name"`
	URL       string `json:"url,omitempty"`
	Status    string `json:"status"`
	LatencyMS int64  `json:"latency_ms"`
	Detail    string `json:"detail,omitempty"`
}

type Prober struct {
	targets       []Target
	client        *http.Client
	degradedAfter time.Duration
}

func NewProber(targets []Target, timeout, degradedAfter time.Duration) *Prober {
	return &Prober{
		targets:       targets,
		client:        &http.Client{Timeout: timeout},
		degradedAfter: degradedAfter,
	}
}

func (p *Prober) Targets() []Target { return p.targets }

func (p *Prober) Run(ctx context.Context) []Result {
	results := make([]Result, len(p.targets))
	var wg sync.WaitGroup
	for i, t := range p.targets {
		wg.Add(1)
		go func(i int, t Target) {
			defer wg.Done()
			results[i] = p.probe(ctx, t)
		}(i, t)
	}
	wg.Wait()
	return results
}

func (p *Prober) probe(ctx context.Context, t Target) Result {
	start := time.Now()
	out := Result{Key: t.Key, Name: t.Name}
	if t.Public {
		out.URL = t.URL
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, t.URL+t.Path, nil)
	if err != nil {
		out.Status = StatusDown
		out.Detail = err.Error()
		return out
	}

	resp, err := p.client.Do(req)
	out.LatencyMS = time.Since(start).Milliseconds()
	if err != nil {
		out.Status = StatusDown
		out.Detail = "сервис недоступен"
		return out
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode >= 500:
		out.Status = StatusDown
		out.Detail = fmt.Sprintf("HTTP %d", resp.StatusCode)
	case resp.StatusCode >= 400:
		out.Status = StatusDegraded
		out.Detail = fmt.Sprintf("HTTP %d", resp.StatusCode)
	case p.degradedAfter > 0 && time.Since(start) > p.degradedAfter:
		out.Status = StatusDegraded
		out.Detail = "медленный ответ"
	default:
		out.Status = StatusOperational
	}
	return out
}

func Overall(results []Result) string {
	overall := StatusOperational
	for _, r := range results {
		overall = Worse(overall, r.Status)
	}
	return overall
}

var defaults = []Target{
	{Key: "api", Name: "API панели", URL: "https://api.vortanix.app", Path: "/health", Public: true},
	{Key: "license", Name: "Сервис лицензий", URL: "https://license.vortanix.app", Path: "/health", Public: true},
	{Key: "billing", Name: "Биллинг", URL: "https://vortanix.app", Path: "/api/health", Public: true},
	{Key: "relay", Name: "Связь с нодами", URL: "https://relay.vortanix.app", Path: "/health", Public: true},
	{Key: "console", Name: "Консоль серверов", URL: "https://console.vortanix.app", Path: "/health", Public: true},
	{Key: "panel", Name: "Веб-интерфейс", URL: "https://panel.vortanix.app", Path: "/", Public: true},
}

func LoadTargets() []Target {
	out := make([]Target, 0, len(defaults))
	for _, t := range defaults {
		raw, set := os.LookupEnv("STATUS_" + strings.ToUpper(t.Key) + "_URL")
		if set {
			raw = strings.TrimSpace(raw)
			if raw == "" {
				continue
			}
			t.URL = strings.TrimRight(raw, "/")
		}
		out = append(out, t)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}
