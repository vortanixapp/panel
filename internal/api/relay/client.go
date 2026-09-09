package relay

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
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
	return c.post(ctx, url, body)
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
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(httpReq)
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

func (c *Client) CommandSyncBinary(
	ctx context.Context,
	nodeID string,
	req CommandRequest,
	binaryPayload []byte,
) (*CommandSyncResponse, error) {
	reqJSON, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	if err := writer.WriteField("request", string(reqJSON)); err != nil {
		return nil, err
	}
	part, err := writer.CreateFormFile("binary", "payload.bin")
	if err != nil {
		return nil, err
	}
	if _, err := part.Write(binaryPayload); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/internal/v1/nodes/%s/command/sync-binary", c.baseURL, nodeID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())
	httpReq.Header.Set("X-Internal-Secret", c.secret)
	resp, err := (&http.Client{Timeout: 25 * time.Second}).Do(httpReq)
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

type ConsoleStartRequest struct {
	SessionID string `json:"session_id"`
	ServerID  string `json:"server_id"`
	NodeID    string `json:"node_id"`
}

func (c *Client) StartConsole(ctx context.Context, req ConsoleStartRequest) error {
	body, err := json.Marshal(req)
	if err != nil {
		return err
	}
	return c.post(ctx, c.baseURL+"/internal/v1/console/start", body)
}

type ConsoleInputRequest struct {
	SessionID string `json:"session_id"`
	ServerID  string `json:"server_id"`
	NodeID    string `json:"node_id"`
	Data      string `json:"data"`
}

func (c *Client) ConsoleInput(ctx context.Context, req ConsoleInputRequest) error {
	body, err := json.Marshal(req)
	if err != nil {
		return err
	}
	return c.post(ctx, c.baseURL+"/internal/v1/console/input", body)
}

func (c *Client) post(ctx context.Context, url string, body []byte) error {
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
