package hosting

import (
	"context"
	"fmt"
	"strings"
)

type ServerConfig struct {
	PanelType   string
	APIURL      string
	APIUsername string
	APIToken    string
}

func (c ServerConfig) configured() bool {
	return c.APIURL != "" && c.APIUsername != "" && c.APIToken != ""
}

type PanelAdapter interface {
	Type() string
	CreateAccount(ctx context.Context, username, domain, planPackage string) (panelAccountID, loginURL string, err error)
	AddDomain(ctx context.Context, panelAccountID, domain string) error
	AddDatabase(ctx context.Context, panelAccountID, dbName, dbUser string) error
	AddEmail(ctx context.Context, panelAccountID, address, password string) error
	Renew(ctx context.Context, panelAccountID string, days int) error
	ChangePassword(ctx context.Context, panelAccountID, newPassword string) error
	Suspend(ctx context.Context, panelAccountID, reason string) error
	Unsuspend(ctx context.Context, panelAccountID string) error
}

type stubAdapter struct {
	panelType string
	reason    string
}

func (s stubAdapter) Type() string { return s.panelType }

func (s stubAdapter) err() error {
	return fmt.Errorf("%s: %s", s.panelType, s.reason)
}

func (s stubAdapter) CreateAccount(_ context.Context, _, _, _ string) (string, string, error) {
	return "", "", s.err()
}
func (s stubAdapter) AddDomain(_ context.Context, _, _ string) error      { return s.err() }
func (s stubAdapter) AddDatabase(_ context.Context, _, _, _ string) error { return s.err() }
func (s stubAdapter) AddEmail(_ context.Context, _, _, _ string) error    { return s.err() }
func (s stubAdapter) Renew(_ context.Context, _ string, _ int) error      { return s.err() }
func (s stubAdapter) ChangePassword(_ context.Context, _, _ string) error { return s.err() }
func (s stubAdapter) Suspend(_ context.Context, _, _ string) error        { return s.err() }
func (s stubAdapter) Unsuspend(_ context.Context, _ string) error         { return s.err() }

func apiSnippet(body []byte) string {
	const limit = 300
	s := strings.TrimSpace(string(body))
	r := []rune(s)
	if len(r) > limit {
		return string(r[:limit]) + "…"
	}
	return s
}

func NewAdapter(cfg ServerConfig) PanelAdapter {
	unconfigured := stubAdapter{panelType: cfg.PanelType, reason: "hosting server has no api_url/api_username/api_token configured"}
	switch cfg.PanelType {
	case "cpanel":
		if !cfg.configured() {
			return unconfigured
		}
		return whmAdapter{cfg: cfg}
	case "fastpanel":
		if !cfg.configured() {
			return unconfigured
		}
		return newFastpanelAdapter(cfg)
	case "ispmanager":
		if !cfg.configured() {
			return unconfigured
		}
		return newISPmanagerAdapter(cfg)
	case "plesk":
		if !cfg.configured() {
			return unconfigured
		}
		return newPleskAdapter(cfg)
	default:
		return stubAdapter{panelType: "generic", reason: "unknown panel type"}
	}
}
