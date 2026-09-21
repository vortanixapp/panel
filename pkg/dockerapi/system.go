package dockerapi

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Info struct {
	Name              string `json:"Name"`
	OperatingSystem   string `json:"OperatingSystem"`
	OSType            string `json:"OSType"`
	KernelVersion     string `json:"KernelVersion"`
	Architecture      string `json:"Architecture"`
	NCPU              int    `json:"NCPU"`
	MemTotal          int64  `json:"MemTotal"`
	DockerRootDir     string `json:"DockerRootDir"`
	Driver            string `json:"Driver"`
	CgroupDriver      string `json:"CgroupDriver"`
	CgroupVersion     string `json:"CgroupVersion"`
	ServerVersion     string `json:"ServerVersion"`
	Containers        int    `json:"Containers"`
	ContainersRunning int    `json:"ContainersRunning"`
	Images            int    `json:"Images"`
	MemoryLimit       bool   `json:"MemoryLimit"`
	SwapLimit         bool   `json:"SwapLimit"`
}

type Version struct {
	Version    string `json:"Version"`
	APIVersion string `json:"ApiVersion"`
}

func (c *Client) Info(ctx context.Context) (*Info, error) {
	var out Info
	if err := c.call(ctx, http.MethodGet, "/info", nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) Version(ctx context.Context) (*Version, error) {
	var out Version
	if err := c.call(ctx, http.MethodGet, "/version", nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type LogLine struct {
	TS   time.Time
	Text string
}

func (c *Client) LogsRange(ctx context.Context, id string, since, until time.Time, tail int, limitBytes int64) ([]LogLine, bool, error) {
	query := url.Values{"stdout": {"1"}, "stderr": {"1"}, "timestamps": {"1"}}
	if tail > 0 {
		query.Set("tail", strconv.Itoa(tail))
	} else {
		query.Set("tail", "all")
	}
	if !since.IsZero() {
		query.Set("since", unixNano(since))
	}
	if !until.IsZero() {
		query.Set("until", unixNano(until))
	}
	resp, err := c.request(ctx, http.MethodGet, "/containers/"+id+"/logs", query, nil)
	if err != nil {
		return nil, false, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, limitBytes+1))
	if err != nil {
		return nil, false, err
	}
	truncated := int64(len(raw)) > limitBytes
	if truncated {
		raw = raw[:limitBytes]
	}
	text := demux(raw)
	var lines []LogLine
	for _, row := range strings.Split(text, "\n") {
		row = strings.TrimRight(row, "\r")
		if row == "" {
			continue
		}
		stamp, rest, ok := strings.Cut(row, " ")
		if ts, err := time.Parse(time.RFC3339Nano, stamp); ok && err == nil {
			lines = append(lines, LogLine{TS: ts, Text: rest})
			continue
		}
		lines = append(lines, LogLine{Text: row})
	}
	return lines, truncated, nil
}

func unixNano(t time.Time) string {
	return fmt.Sprintf("%d.%09d", t.Unix(), t.Nanosecond())
}
