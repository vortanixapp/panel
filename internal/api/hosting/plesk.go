package hosting

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type pleskAdapter struct {
	cfg    ServerConfig
	client *http.Client

	mu      sync.Mutex
	domains map[string]string
}

func newPleskAdapter(cfg ServerConfig) *pleskAdapter {
	return &pleskAdapter{
		cfg:     cfg,
		client:  &http.Client{Timeout: 30 * time.Second},
		domains: map[string]string{},
	}
}

func (p *pleskAdapter) Type() string { return "plesk" }

func (p *pleskAdapter) base() string { return strings.TrimSuffix(p.cfg.APIURL, "/") }

func pleskMessage(body []byte, status int) string {
	var parsed struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &parsed); err == nil && parsed.Message != "" {
		return parsed.Message
	}
	return fmt.Sprintf("http %d: %s", status, apiSnippet(body))
}

func (p *pleskAdapter) request(ctx context.Context, method, path string, payload any) ([]byte, error) {
	var rdr io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		rdr = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, p.base()+path, rdr)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.SetBasicAuth(p.cfg.APIUsername, p.cfg.APIToken)
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("plesk: не удалось дочитать ответ: %w", err)
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("plesk: %s", pleskMessage(body, resp.StatusCode))
	}
	return body, nil
}

func (p *pleskAdapter) cli(ctx context.Context, utility string, params []string) error {
	body, err := p.request(ctx, http.MethodPost, "/api/v2/cli/"+utility+"/call", map[string]any{
		"params": params,
	})
	if err != nil {
		return err
	}
	var parsed struct {
		Code   int    `json:"code"`
		Stdout string `json:"stdout"`
		Stderr string `json:"stderr"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return fmt.Errorf("plesk: нераспознанный ответ %s: %s", utility, apiSnippet(body))
	}
	if parsed.Code == 0 {
		return nil
	}
	reason := strings.TrimSpace(parsed.Stderr)
	if reason == "" {
		reason = strings.TrimSpace(parsed.Stdout)
	}
	if reason == "" {
		reason = fmt.Sprintf("утилита %s завершилась с кодом %d", utility, parsed.Code)
	}
	return fmt.Errorf("plesk: %s", reason)
}

func (p *pleskAdapter) rememberDomain(login, domain string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.domains[login] = domain
}

func (p *pleskAdapter) primaryDomain(ctx context.Context, login string) (string, error) {
	p.mu.Lock()
	domain, ok := p.domains[login]
	p.mu.Unlock()
	if ok {
		return domain, nil
	}
	id, err := p.clientID(ctx, login)
	if err != nil {
		return "", err
	}
	body, err := p.request(ctx, http.MethodGet, "/api/v2/clients/"+strconv.Itoa(id)+"/domains", nil)
	if err != nil {
		return "", err
	}
	var list []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(body, &list); err != nil {
		return "", fmt.Errorf("plesk: нераспознанный список доменов: %s", apiSnippet(body))
	}
	for _, d := range list {
		if d.Name != "" {
			p.rememberDomain(login, d.Name)
			return d.Name, nil
		}
	}
	return "", fmt.Errorf("plesk: у клиента %q нет ни одного домена", login)
}

func (p *pleskAdapter) clientID(ctx context.Context, login string) (int, error) {
	body, err := p.request(ctx, http.MethodGet, "/api/v2/clients", nil)
	if err != nil {
		return 0, err
	}
	var list []struct {
		ID    int    `json:"id"`
		Login string `json:"login"`
	}
	if err := json.Unmarshal(body, &list); err != nil {
		return 0, fmt.Errorf("plesk: нераспознанный список клиентов: %s", apiSnippet(body))
	}
	for _, c := range list {
		if c.Login == login {
			return c.ID, nil
		}
	}
	return 0, fmt.Errorf("plesk: клиент %q не найден в панели", login)
}

func (p *pleskAdapter) CreateAccount(ctx context.Context, username, domain, planPackage string) (string, string, error) {
	if username == "" || domain == "" {
		return "", "", fmt.Errorf("username and domain required")
	}
	if _, err := p.request(ctx, http.MethodPost, "/api/v2/clients", map[string]any{
		"name":     username,
		"login":    username,
		"password": randomPassword(18),
		"type":     "customer",
	}); err != nil {
		return "", "", err
	}
	site := map[string]any{
		"name":         domain,
		"hosting_type": "virtual",
		"owner_client": map[string]any{"login": username},
	}
	if planPackage != "" {
		site["plan"] = map[string]any{"name": planPackage}
	}
	if _, err := p.request(ctx, http.MethodPost, "/api/v2/domains", site); err != nil {
		return "", "", err
	}
	p.rememberDomain(username, domain)
	return username, "https://" + domain + ":8443", nil
}

func (p *pleskAdapter) AddDomain(ctx context.Context, panelAccountID, domain string) error {
	if domain == "" {
		return fmt.Errorf("domain required")
	}
	if panelAccountID == "" {
		return fmt.Errorf("panel account id required")
	}
	_, err := p.request(ctx, http.MethodPost, "/api/v2/domains", map[string]any{
		"name":         domain,
		"hosting_type": "virtual",
		"owner_client": map[string]any{"login": panelAccountID},
	})
	return err
}

func (p *pleskAdapter) AddDatabase(ctx context.Context, panelAccountID, dbName, dbUser string) error {
	if dbName == "" {
		return fmt.Errorf("database name required")
	}
	if panelAccountID == "" {
		return fmt.Errorf("panel account id required")
	}
	domain, err := p.primaryDomain(ctx, panelAccountID)
	if err != nil {
		return err
	}
	params := []string{"--create", dbName, "-domain", domain, "-type", "mysql"}
	if dbUser != "" {
		params = append(params, "-add_user", dbUser, "-passwd", randomPassword(18))
	}
	return p.cli(ctx, "database", params)
}

func (p *pleskAdapter) AddEmail(ctx context.Context, _, address, password string) error {
	if address == "" {
		return fmt.Errorf("email required")
	}
	local, domain, ok := strings.Cut(address, "@")
	if !ok || local == "" || domain == "" {
		return fmt.Errorf("invalid email address %q", address)
	}
	if password == "" {
		password = randomPassword(18)
	}
	return p.cli(ctx, "mail", []string{"--create", address, "-passwd", password, "-mailbox", "true"})
}

func (p *pleskAdapter) Renew(_ context.Context, _ string, days int) error {
	if days <= 0 {
		return fmt.Errorf("invalid period")
	}
	return nil
}

func (p *pleskAdapter) ChangePassword(ctx context.Context, panelAccountID, newPassword string) error {
	if len(newPassword) < 8 {
		return fmt.Errorf("password too short")
	}
	if panelAccountID == "" {
		return fmt.Errorf("panel account id required")
	}
	return p.cli(ctx, "customer", []string{"--update", panelAccountID, "-passwd", newPassword})
}

func (p *pleskAdapter) Suspend(ctx context.Context, panelAccountID, _ string) error {
	if panelAccountID == "" {
		return fmt.Errorf("panel account id required")
	}
	return p.cli(ctx, "customer", []string{"--suspend", panelAccountID})
}

func (p *pleskAdapter) Unsuspend(ctx context.Context, panelAccountID string) error {
	if panelAccountID == "" {
		return fmt.Errorf("panel account id required")
	}
	return p.cli(ctx, "customer", []string{"--activate", panelAccountID})
}
