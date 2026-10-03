package docker

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	errEngineUnavailable = errors.New("docker engine api unavailable")
	errNoContainer       = errors.New("no such container")
)

type engineContainer struct {
	Running   bool
	Status    string
	StartedAt string
	IPs       []string
}

var engineClient struct {
	once   sync.Once
	client *http.Client
}

func enginePath() string {
	host := strings.TrimSpace(os.Getenv("DOCKER_HOST"))
	if host == "" {
		return "/var/run/docker.sock"
	}
	if path, ok := strings.CutPrefix(host, "unix://"); ok {
		return path
	}
	return ""
}

func engineHTTP() *http.Client {
	engineClient.once.Do(func() {
		path := enginePath()
		if path == "" {
			return
		}
		engineClient.client = &http.Client{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
					var d net.Dialer
					return d.DialContext(ctx, "unix", path)
				},
				MaxIdleConns:        8,
				MaxIdleConnsPerHost: 8,
				IdleConnTimeout:     60 * time.Second,
			},
			Timeout: 10 * time.Second,
		}
	})
	return engineClient.client
}

func engineInspect(ctx context.Context, name string) (engineContainer, error) {
	client := engineHTTP()
	if client == nil {
		return engineContainer{}, errEngineUnavailable
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://docker/containers/"+url.PathEscape(name)+"/json", nil)
	if err != nil {
		return engineContainer{}, errEngineUnavailable
	}
	resp, err := client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return engineContainer{}, ctx.Err()
		}
		return engineContainer{}, errEngineUnavailable
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return engineContainer{}, errNoContainer
	}
	if resp.StatusCode != http.StatusOK {
		return engineContainer{}, errEngineUnavailable
	}
	var raw struct {
		State struct {
			Status    string
			Running   bool
			StartedAt string
		}
		NetworkSettings struct {
			Networks map[string]struct {
				IPAddress string
			}
		}
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return engineContainer{}, errEngineUnavailable
	}
	info := engineContainer{Running: raw.State.Running, Status: raw.State.Status, StartedAt: raw.State.StartedAt}
	for _, n := range raw.NetworkSettings.Networks {
		if n.IPAddress != "" {
			info.IPs = append(info.IPs, n.IPAddress)
		}
	}
	return info, nil
}
