package docker

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/vortanixapp/panel/pkg/dockerapi"
)

const (
	KindServer        = "server"
	KindAgent         = "agent"
	KindAgentPrevious = "agent_previous"
	KindHelper        = "helper"
	KindMySQL         = "mysql"
	KindOther         = "other"

	labelHelper       = "vortanix.helper"
	labelAgentUpgrade = "app.vortanix.agent-upgrade"
)

type ContainerInfo struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	ServerID   string   `json:"server_id,omitempty"`
	Managed    bool     `json:"managed"`
	Kind       string   `json:"kind"`
	Image      string   `json:"image"`
	ImageID    string   `json:"image_id"`
	State      string   `json:"state"`
	Status     string   `json:"status"`
	ExitCode   int      `json:"exit_code"`
	CreatedAt  string   `json:"created_at"`
	StartedAt  string   `json:"started_at,omitempty"`
	FinishedAt string   `json:"finished_at,omitempty"`
	Restarts   int      `json:"restarts"`
	Health     string   `json:"health,omitempty"`
	CPUPct     float64  `json:"cpu_pct"`
	MemUsedMB  int      `json:"mem_used_mb"`
	MemLimitMB int      `json:"mem_limit_mb"`
	Ports      []string `json:"ports"`
}

type selfRef struct {
	id, name, image, repo string
}

func agentSelf(ctx context.Context, cli *dockerapi.Client) selfRef {
	id := dockerapi.SelfContainerID()
	if id == "" {
		return selfRef{}
	}
	c, err := cli.Inspect(ctx, id)
	if err != nil {
		return selfRef{id: id}
	}
	repo, _ := dockerapi.SplitRef(c.ConfigImage())
	return selfRef{id: c.ID, name: c.CleanName(), image: c.Image, repo: repo}
}

func sameID(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	return strings.HasPrefix(a, b) || strings.HasPrefix(b, a)
}

func classifyContainer(s dockerapi.Summary, self selfRef) (kind, serverID string) {
	name := s.CleanName()
	switch {
	case sameID(s.ID, self.id):
		return KindAgent, ""
	case self.name != "" && name == self.name+"-previous":
		return KindAgentPrevious, ""
	case s.Labels[labelHelper] != "", s.Labels[labelAgentUpgrade] != "", self.name != "" && name == self.name+"-upgrade":
		return KindHelper, ""
	}
	if id := ServerIDFromContainer(name); id != "" {
		return KindServer, id
	}
	if id := s.Labels["vortanix.server_id"]; id != "" && s.Labels["vortanix.managed"] == "true" {
		return KindServer, id
	}
	if strings.HasPrefix(name, "vortanix-mysql") || strings.HasPrefix(name, "vortanix-mariadb") {
		return KindMySQL, ""
	}
	return KindOther, ""
}

func formatPorts(ports []dockerapi.Port) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, p := range ports {
		if p.PublicPort == 0 {
			continue
		}
		s := strings.TrimSpace(p.IP)
		if s == "" || s == "0.0.0.0" || s == "::" {
			s = ""
		} else {
			s += ":"
		}
		s += strconv.Itoa(p.PublicPort) + "/" + p.Type
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	sort.Strings(out)
	return out
}

func dockerTime(s string) string {
	if s == "" || strings.HasPrefix(s, "0001-") {
		return ""
	}
	return s
}

func ListNodeContainers(ctx context.Context) ([]ContainerInfo, error) {
	cli := dockerapi.New()
	list, err := cli.List(ctx, true, nil)
	if err != nil {
		return nil, err
	}
	self := agentSelf(ctx, cli)
	stats := recentStats(ctx, 45*time.Second)
	out := make([]ContainerInfo, len(list))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 8)
	for i, s := range list {
		kind, serverID := classifyContainer(s, self)
		info := ContainerInfo{
			ID: s.ID, Name: s.CleanName(), ServerID: serverID, Kind: kind,
			Managed: kind != KindOther,
			Image:   s.Image, ImageID: s.ImageID, State: s.State, Status: s.Status,
			CreatedAt: time.Unix(s.Created, 0).UTC().Format(time.RFC3339),
			Ports:     formatPorts(s.Ports),
		}
		if st, ok := stats[info.Name]; ok && s.State == "running" {
			info.CPUPct, info.MemUsedMB, info.MemLimitMB = st.CPUPct, st.MemUsedMB, st.MemLimitMB
		}
		out[i] = info
		wg.Add(1)
		go func(i int, id string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			c, err := cli.Inspect(ctx, id)
			if err != nil {
				return
			}
			out[i].Restarts = c.RestartCount
			out[i].ExitCode = c.State.ExitCode
			out[i].StartedAt = dockerTime(c.State.StartedAt)
			out[i].FinishedAt = dockerTime(c.State.FinishedAt)
			if c.State.Health != nil {
				out[i].Health = c.State.Health.Status
			}
		}(i, s.ID)
	}
	wg.Wait()
	sort.Slice(out, func(a, b int) bool {
		if out[a].Kind != out[b].Kind {
			return kindOrder(out[a].Kind) < kindOrder(out[b].Kind)
		}
		return out[a].Name < out[b].Name
	})
	return out, nil
}

func kindOrder(kind string) int {
	switch kind {
	case KindServer:
		return 0
	case KindAgent:
		return 1
	case KindAgentPrevious:
		return 2
	case KindMySQL:
		return 3
	case KindHelper:
		return 4
	}
	return 5
}
