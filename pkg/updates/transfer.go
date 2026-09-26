package updates

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/vortanixapp/panel/pkg/paneltransfer"
)

func (u *Updater) stream(ctx context.Context, method, path string, body io.Reader) (io.ReadCloser, error) {
	if !u.Configured() {
		return nil, ErrUpdaterMissing
	}
	req, err := http.NewRequestWithContext(ctx, method, u.URL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Internal-Secret", u.Secret)
	if body != nil {
		req.Header.Set("Content-Type", "application/octet-stream")
	}
	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("служба обновления не отвечает: %w", err)
	}
	if resp.StatusCode >= 300 {
		var e struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(io.LimitReader(resp.Body, 1<<16)).Decode(&e)
		resp.Body.Close()
		if e.Error == "" {
			e.Error = resp.Status
		}
		return nil, errors.New(e.Error)
	}
	return resp.Body, nil
}

func (u *Updater) TransferProbe(ctx context.Context) (*paneltransfer.Probe, error) {
	var out paneltransfer.Probe
	if err := u.do(ctx, http.MethodGet, "/internal/v1/transfer/probe", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (u *Updater) TransferExport(ctx context.Context) (io.ReadCloser, error) {
	return u.stream(ctx, http.MethodGet, "/internal/v1/transfer/export", nil)
}

func (u *Updater) TransferImport(ctx context.Context, body io.Reader) (io.ReadCloser, error) {
	return u.stream(ctx, http.MethodPost, "/internal/v1/transfer/import", body)
}
