package relay

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
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

type CommandRequest struct {
	CommandID string         `json:"command_id"`
	Action    string         `json:"action"`
	ServerID  string         `json:"server_id"`
	Payload   map[string]any `json:"payload"`
}

func (c *Client) SendCommand(ctx context.Context, nodeID string, req CommandRequest) error {
	body, err := json.Marshal(req)
	if err != nil {
		return err
	}
	url := fmt.Sprintf("%s/internal/v1/nodes/%s/command", c.baseURL, nodeID)
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
		return fmt.Errorf("relay: %s", msg)
	}
	return nil
}

type CommandSyncResponse struct {
	CommandID string         `json:"command_id"`
	OK        bool           `json:"ok"`
	Result    map[string]any `json:"result"`
}

func (c *Client) CommandSync(ctx context.Context, nodeID string, req CommandRequest) (*CommandSyncResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/internal/v1/nodes/%s/command/sync", c.baseURL, nodeID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Internal-Secret", c.secret)
	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		var errBody map[string]string
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		msg := errBody["error"]
		if msg == "" {
			msg = resp.Status
		}
		return nil, fmt.Errorf("relay: %s", msg)
	}
	var out CommandSyncResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}
