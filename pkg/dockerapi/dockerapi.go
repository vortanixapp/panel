package dockerapi

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type Client struct {
	http *http.Client
}

func New() *Client {
	sock := "/var/run/docker.sock"
	if host := os.Getenv("DOCKER_HOST"); strings.HasPrefix(host, "unix://") {
		sock = strings.TrimPrefix(host, "unix://")
	}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			var d net.Dialer
			return d.DialContext(ctx, "unix", sock)
		},
	}
	return &Client{http: &http.Client{Transport: transport}}
}

type Error struct {
	Status  int
	Message string
}

func (e *Error) Error() string { return e.Message }

func IsNotFound(err error) bool {
	var e *Error
	return errors.As(err, &e) && e.Status == http.StatusNotFound
}

type ContainerState struct {
	Status     string `json:"Status"`
	Running    bool   `json:"Running"`
	Restarting bool   `json:"Restarting"`
	ExitCode   int    `json:"ExitCode"`
	Error      string `json:"Error"`
	StartedAt  string `json:"StartedAt"`
	FinishedAt string `json:"FinishedAt"`
	Health     *struct {
		Status string `json:"Status"`
	} `json:"Health"`
}

type Container struct {
	ID              string         `json:"Id"`
	Name            string         `json:"Name"`
	Image           string         `json:"Image"`
	RestartCount    int            `json:"RestartCount"`
	State           ContainerState `json:"State"`
	Config          map[string]any `json:"Config"`
	HostConfig      map[string]any `json:"HostConfig"`
	NetworkSettings struct {
		Networks map[string]map[string]any `json:"Networks"`
	} `json:"NetworkSettings"`
}

func (c *Container) CleanName() string {
	return strings.TrimPrefix(c.Name, "/")
}

func (c *Container) ConfigImage() string {
	s, _ := c.Config["Image"].(string)
	return s
}

func (c *Container) Labels() map[string]string {
	out := map[string]string{}
	raw, _ := c.Config["Labels"].(map[string]any)
	for k, v := range raw {
		if s, ok := v.(string); ok {
			out[k] = s
		}
	}
	return out
}

type Image struct {
	ID     string         `json:"Id"`
	Config map[string]any `json:"Config"`
}

type Summary struct {
	ID     string            `json:"Id"`
	Names  []string          `json:"Names"`
	Image  string            `json:"Image"`
	State  string            `json:"State"`
	Labels map[string]string `json:"Labels"`
}

func (c *Client) request(ctx context.Context, method, path string, query url.Values, body any) (*http.Response, error) {
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(raw)
	}
	target := "http://docker" + path
	if len(query) > 0 {
		target += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, method, target, reader)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("docker недоступен: %w", err)
	}
	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
		var payload struct {
			Message string `json:"message"`
		}
		_ = json.Unmarshal(raw, &payload)
		msg := strings.TrimSpace(payload.Message)
		if msg == "" {
			msg = strings.TrimSpace(string(raw))
		}
		if msg == "" {
			msg = resp.Status
		}
		return nil, &Error{Status: resp.StatusCode, Message: msg}
	}
	return resp, nil
}

func (c *Client) call(ctx context.Context, method, path string, query url.Values, body, out any) error {
	resp, err := c.request(ctx, method, path, query, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if out == nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) Inspect(ctx context.Context, id string) (*Container, error) {
	var out Container
	if err := c.call(ctx, http.MethodGet, "/containers/"+id+"/json", nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) InspectImage(ctx context.Context, ref string) (*Image, error) {
	var out Image
	if err := c.call(ctx, http.MethodGet, "/images/"+ref+"/json", nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) List(ctx context.Context, all bool, filters map[string][]string) ([]Summary, error) {
	query := url.Values{}
	if all {
		query.Set("all", "1")
	}
	if len(filters) > 0 {
		raw, _ := json.Marshal(filters)
		query.Set("filters", string(raw))
	}
	var out []Summary
	if err := c.call(ctx, http.MethodGet, "/containers/json", query, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func SplitRef(ref string) (name, tag string) {
	if strings.Contains(ref, "@") {
		return ref, ""
	}
	if i := strings.LastIndex(ref, ":"); i > strings.LastIndex(ref, "/") {
		return ref[:i], ref[i+1:]
	}
	return ref, "latest"
}

func (c *Client) Pull(ctx context.Context, ref string, onStatus func(string)) error {
	name, tag := SplitRef(ref)
	query := url.Values{"fromImage": {name}}
	if tag != "" {
		query.Set("tag", tag)
	}
	resp, err := c.request(ctx, http.MethodPost, "/images/create", query, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	dec := json.NewDecoder(resp.Body)
	last := ""
	for {
		var msg struct {
			Status      string `json:"status"`
			ID          string `json:"id"`
			Error       string `json:"error"`
			ErrorDetail struct {
				Message string `json:"message"`
			} `json:"errorDetail"`
		}
		if err := dec.Decode(&msg); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		if msg.Error != "" || msg.ErrorDetail.Message != "" {
			if msg.ErrorDetail.Message != "" {
				return errors.New(msg.ErrorDetail.Message)
			}
			return errors.New(msg.Error)
		}
		if onStatus != nil && msg.ID == "" && msg.Status != "" && msg.Status != last {
			last = msg.Status
			onStatus(msg.Status)
		}
	}
}

func (c *Client) Create(ctx context.Context, name string, body map[string]any) (string, error) {
	query := url.Values{}
	if name != "" {
		query.Set("name", name)
	}
	var out struct {
		ID string `json:"Id"`
	}
	if err := c.call(ctx, http.MethodPost, "/containers/create", query, body, &out); err != nil {
		return "", err
	}
	return out.ID, nil
}

func (c *Client) Start(ctx context.Context, id string) error {
	return c.call(ctx, http.MethodPost, "/containers/"+id+"/start", nil, nil, nil)
}

func (c *Client) Stop(ctx context.Context, id string, timeoutSec int) error {
	query := url.Values{"t": {strconv.Itoa(timeoutSec)}}
	return c.call(ctx, http.MethodPost, "/containers/"+id+"/stop", query, nil, nil)
}

func (c *Client) Rename(ctx context.Context, id, name string) error {
	return c.call(ctx, http.MethodPost, "/containers/"+id+"/rename", url.Values{"name": {name}}, nil, nil)
}

func (c *Client) Remove(ctx context.Context, id string, force bool) error {
	query := url.Values{}
	if force {
		query.Set("force", "1")
	}
	return c.call(ctx, http.MethodDelete, "/containers/"+id, query, nil, nil)
}

func (c *Client) ConnectNetwork(ctx context.Context, network, containerID string, endpoint map[string]any) error {
	body := map[string]any{"Container": containerID}
	if endpoint != nil {
		body["EndpointConfig"] = endpoint
	}
	return c.call(ctx, http.MethodPost, "/networks/"+network+"/connect", nil, body, nil)
}

func (c *Client) Logs(ctx context.Context, id string, tail int) ([]string, error) {
	query := url.Values{"stdout": {"1"}, "stderr": {"1"}, "tail": {strconv.Itoa(tail)}}
	resp, err := c.request(ctx, http.MethodGet, "/containers/"+id+"/logs", query, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	text := demux(raw)
	lines := []string{}
	scanner := bufio.NewScanner(strings.NewReader(text))
	scanner.Buffer(make([]byte, 64*1024), 1<<20)
	for scanner.Scan() {
		lines = append(lines, strings.TrimRight(scanner.Text(), "\r"))
	}
	return lines, nil
}

func demux(raw []byte) string {
	if len(raw) < 8 || raw[0] > 2 || raw[1] != 0 || raw[2] != 0 || raw[3] != 0 {
		return string(raw)
	}
	var out bytes.Buffer
	for len(raw) >= 8 {
		size := int(binary.BigEndian.Uint32(raw[4:8]))
		raw = raw[8:]
		if size > len(raw) {
			size = len(raw)
		}
		out.Write(raw[:size])
		raw = raw[size:]
	}
	return out.String()
}

var containerIDPattern = regexp.MustCompile(`/containers/([0-9a-f]{64})/`)
var cgroupIDPattern = regexp.MustCompile(`[0-9a-f]{64}`)

func SelfContainerID() string {
	if raw, err := os.ReadFile("/proc/self/mountinfo"); err == nil {
		if m := containerIDPattern.FindSubmatch(raw); m != nil {
			return string(m[1])
		}
	}
	if raw, err := os.ReadFile("/proc/self/cgroup"); err == nil {
		if m := cgroupIDPattern.Find(raw); m != nil {
			return string(m)
		}
	}
	if host, err := os.Hostname(); err == nil && len(host) == 12 {
		return host
	}
	return ""
}
