package licenseclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL string
	http    *http.Client
}

func New(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: 10 * time.Second},
	}
}

type ActivateRequest struct {
	LicenseKey     string `json:"license_key"`
	InstallationID string `json:"installation_id"`
	Domain         string `json:"domain"`
	Fingerprint    string `json:"fingerprint"`
}

type ActivateResponse struct {
	LicenseToken string `json:"license_token"`
	ActivationID string `json:"activation_id"`
	TenantID     string `json:"tenant_id"`
	TenantSlug   string `json:"tenant_slug"`
	Plan         string `json:"plan"`
	Status       string `json:"status"`
	Revision     int64  `json:"revision"`
	KeyHint      string `json:"key_hint"`
}

func (c *Client) Activate(ctx context.Context, req ActivateRequest) (ActivateResponse, error) {
	var out ActivateResponse
	err := c.do(ctx, http.MethodPost, "/v1/activate", "", req, &out)
	return out, err
}

type RefreshResponse struct {
	LicenseToken string `json:"license_token"`
	Status       string `json:"status"`
	Revision     int64  `json:"revision"`
}

func (c *Client) Refresh(ctx context.Context, token string) (RefreshResponse, error) {
	var out RefreshResponse
	err := c.do(ctx, http.MethodPost, "/v1/refresh", token, map[string]string{}, &out)
	return out, err
}

type StateRequest struct {
	RequestID        string `json:"request_id"`
	Version          string `json:"version"`
	Revision         int64  `json:"revision"`
	ServersUsed      *int   `json:"servers_used,omitempty"`
	NodesUsed        *int   `json:"nodes_used,omitempty"`
	AdminsUsed       *int   `json:"admins_used,omitempty"`
	CPULoadPercent   *int   `json:"cpu_load_percent,omitempty"`
	Domain           string `json:"domain,omitempty"`
	IssuedAt         int64  `json:"issued_at"`
	UpdateAction     string `json:"update_action,omitempty"`
	DeferUpdateUntil int64  `json:"defer_update_until,omitempty"`
	UpdateComponent  string `json:"update_component,omitempty"`
}

func (c *Client) State(ctx context.Context, token string, req StateRequest) (string, error) {
	var out struct {
		State string `json:"state"`
	}
	req.IssuedAt = time.Now().Unix()
	if err := c.do(ctx, http.MethodPost, "/v1/state", token, req, &out); err != nil {
		return "", err
	}
	return out.State, nil
}

func (c *Client) PublicKey(ctx context.Context) (string, error) {
	var out struct {
		PublicKey string `json:"public_key"`
	}
	if err := c.do(ctx, http.MethodGet, "/v1/public-key", "", nil, &out); err != nil {
		return "", err
	}
	return out.PublicKey, nil
}

type Error struct {
	StatusCode int
	Code       string
	Message    string
}

func (e *Error) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("сервис лицензий вернул %d: %s", e.StatusCode, e.Code)
	}
	return fmt.Sprintf("сервис лицензий вернул %d: %s", e.StatusCode, e.Message)
}

func (e *Error) Rejected() bool {
	return e.StatusCode == http.StatusForbidden ||
		e.StatusCode == http.StatusUnauthorized ||
		e.StatusCode == http.StatusNotFound
}

func (c *Client) do(ctx context.Context, method, path, bearer string, body, out any) error {
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(raw)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("сервис лицензий недоступен: %w", err)
	}
	defer resp.Body.Close()

	payload, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 300 {
		var errBody struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(payload, &errBody)
		return &Error{StatusCode: resp.StatusCode, Code: errBody.Error, Message: strings.TrimSpace(string(payload))}
	}
	if out != nil {
		return json.Unmarshal(payload, out)
	}
	return nil
}

type UpdateEvent struct {
	Stage     string    `json:"stage"`
	Message   string    `json:"message"`
	Version   string    `json:"version"`
	Component string    `json:"component"`
	CreatedAt time.Time `json:"created_at"`
}

// UpdateRelease — версия компонента с описанием. Из неё собирается список
// релизов на вкладке обновлений: без него владелец видит только номер
// доступной версии и не знает ни что в ней, ни что он пропустил.
type UpdateRelease struct {
	Version        string     `json:"version"`
	Channel        string     `json:"channel"`
	PublishedAt    time.Time  `json:"published_at"`
	Notes          string     `json:"notes"`
	MandatoryAfter *time.Time `json:"mandatory_after"`
	IsTarget       bool       `json:"is_target"`
	IsCurrent      bool       `json:"is_current"`
}

// UpdateEvents отдаёт журнал обновления панели. Ходит по ключу лицензии — тем
// же, которым пользуется служба обновления на машине клиента.
func (c *Client) UpdateEvents(ctx context.Context, licenseKey, component string) ([]UpdateEvent, error) {
	var out struct {
		Events []UpdateEvent `json:"events"`
	}
	err := c.do(ctx, http.MethodPost, "/v1/update/events", "",
		map[string]string{"license_key": licenseKey, "component": component}, &out)
	return out.Events, err
}

// ComponentReleases — история версий компонента вместе с тем, на чём панель
// сейчас и куда её ведут. Версия компонента известна только сервису лицензий:
// у панели своей записи о версии службы обновления нет.
type ComponentReleases struct {
	Releases       []UpdateRelease `json:"releases"`
	Component      string          `json:"component"`
	CurrentVersion string          `json:"current_version"`
	TargetVersion  string          `json:"target_version"`
}

// UpdateReleases отдаёт историю версий компонента.
func (c *Client) UpdateReleases(ctx context.Context, licenseKey, component string) (ComponentReleases, error) {
	var out ComponentReleases
	err := c.do(ctx, http.MethodPost, "/v1/update/releases", "",
		map[string]string{"license_key": licenseKey, "component": component}, &out)
	return out, err
}

// BugReport — отчёт об ошибке панели, уходящий разработчику.
type BugReport struct {
	LicenseKey  string `json:"license_key"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Steps       string `json:"steps,omitempty"`
	Expected    string `json:"expected,omitempty"`
	Actual      string `json:"actual,omitempty"`
	Severity    string `json:"severity,omitempty"`
	AppVersion  string `json:"app_version,omitempty"`
	Component   string `json:"component,omitempty"`
	Scope       string `json:"scope,omitempty"`
}

// ReportBug отправляет отчёт в консоль разработчика через сервис лицензий:
// биллинг наружу не опубликован, а сервис лицензий панель и так знает.
func (c *Client) ReportBug(ctx context.Context, report BugReport) (int64, error) {
	var out struct {
		Number int64 `json:"number"`
	}
	err := c.do(ctx, http.MethodPost, "/v1/panel/bug-report", "", report, &out)
	return out.Number, err
}
