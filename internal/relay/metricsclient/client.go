package metricsclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	baseURL string
	secret  string
	http    *http.Client
}

func New(baseURL, secret string) *Client {
	return &Client{
		baseURL: baseURL,
		secret:  secret,
		http:    &http.Client{Timeout: 5 * time.Second},
	}
}

type IngestRequest struct {
	TenantID   string  `json:"tenant_id"`
	ServerID   string  `json:"server_id"`
	CPUPct     float64 `json:"cpu_pct"`
	MemUsedMB  int     `json:"mem_used_mb"`
	MemLimitMB int     `json:"mem_limit_mb"`
	TS         *int64  `json:"ts,omitempty"`
}

func (c *Client) Ingest(ctx context.Context, req IngestRequest) error {
	body, err := json.Marshal(req)
	if err != nil {
		return err
	}
	url := c.baseURL + "/internal/v1/ingest"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Internal-Secret", c.secret)
	resp, err := c.http.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		var errBody map[string]string
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		msg := errBody["error"]
		if msg == "" {
			msg = resp.Status
		}
		return fmt.Errorf("metrics-ingest: %s", msg)
	}
	return nil
}

func (c *Client) IngestAsync(req IngestRequest) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = c.Ingest(ctx, req)
	}()
}
