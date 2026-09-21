package relayclient

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
		http:    &http.Client{Timeout: 15 * time.Second},
	}
}

type ConsoleSessionRequest struct {
	SessionID string `json:"session_id"`
	ServerID  string `json:"server_id"`
	NodeID    string `json:"node_id"`
}

type ConsoleInputRequest struct {
	SessionID string `json:"session_id"`
	ServerID  string `json:"server_id"`
	NodeID    string `json:"node_id"`
	GameID    string `json:"game_id"`
	Data      string `json:"data"`
}

func (c *Client) StartConsole(ctx context.Context, req ConsoleSessionRequest) error {
	return c.post(ctx, "/internal/v1/console/start", req)
}

func (c *Client) StopConsole(ctx context.Context, req ConsoleSessionRequest) error {
	return c.post(ctx, "/internal/v1/console/stop", req)
}

func (c *Client) ConsoleInput(ctx context.Context, req ConsoleInputRequest) error {
	return c.post(ctx, "/internal/v1/console/input", req)
}

func (c *Client) post(ctx context.Context, path string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
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
		return fmt.Errorf("relay error: %s", resp.Status)
	}
	return nil
}
