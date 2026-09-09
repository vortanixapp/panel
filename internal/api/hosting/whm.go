package hosting

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type whmAdapter struct {
	cfg ServerConfig
}

func (w whmAdapter) Type() string { return "cpanel" }

func (w whmAdapter) whmGet(ctx context.Context, path string, q url.Values) ([]byte, error) {
	base := strings.TrimSuffix(w.cfg.APIURL, "/")
	full := base + path + "?" + q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, full, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "whm "+w.cfg.APIUsername+":"+w.cfg.APIToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("whm http %d: %s", resp.StatusCode, string(body))
	}
	return body, nil
}

func whmAPI1Result(body []byte) error {
	var parsed struct {
		Metadata struct {
			Result int    `json:"result"`
			Reason string `json:"reason"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return fmt.Errorf("whm: invalid response: %w", err)
	}
	if parsed.Metadata.Result != 1 {
		reason := parsed.Metadata.Reason
		if reason == "" {
			reason = "unknown error"
		}
		return fmt.Errorf("whm: %s", reason)
	}
	return nil
}

func parseUAPIResult(body []byte) error {
	type uapiEnvelope struct {
		Status int      `json:"status"`
		Errors []string `json:"errors"`
	}
	var direct uapiEnvelope
	var wrapped struct {
		Result uapiEnvelope `json:"result"`
	}
	_ = json.Unmarshal(body, &direct)
	_ = json.Unmarshal(body, &wrapped)

	env := direct
	if env.Status == 0 && wrapped.Result.Status != 0 {
		env = wrapped.Result
	}
	if env.Status == 1 {
		return nil
	}
	if len(env.Errors) > 0 {
		return fmt.Errorf("whm uapi: %s", strings.Join(env.Errors, "; "))
	}
	return fmt.Errorf("whm uapi: call did not report success (unrecognized response shape)")
}

func randomPassword(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func (w whmAdapter) CreateAccount(ctx context.Context, username, domain, planPackage string) (string, string, error) {
	q := url.Values{
		"api.version": {"1"},
		"username":    {username},
		"domain":      {domain},
		"plan":        {planPackage},
		"password":    {randomPassword(18)},
	}
	body, err := w.whmGet(ctx, "/json-api/createacct", q)
	if err != nil {
		return "", "", err
	}
	if err := whmAPI1Result(body); err != nil {
		return "", "", err
	}
	loginURL := "https://" + domain + ":2083"
	return username, loginURL, nil
}

func (w whmAdapter) AddDomain(ctx context.Context, panelAccountID, domain string) error {
	if domain == "" {
		return fmt.Errorf("domain required")
	}
	q := url.Values{
		"api.version": {"1"},
		"user":        {panelAccountID},
		"domain":      {domain},
	}
	body, err := w.whmGet(ctx, "/json-api/park", q)
	if err != nil {
		return err
	}
	return whmAPI1Result(body)
}

func (w whmAdapter) AddDatabase(ctx context.Context, panelAccountID, dbName, dbUser string) error {
	if dbName == "" {
		return fmt.Errorf("database name required")
	}
	q := url.Values{
		"cpanel_jsonapi_user":       {panelAccountID},
		"cpanel_jsonapi_apiversion": {"3"},
		"cpanel_jsonapi_module":     {"Mysql"},
		"cpanel_jsonapi_func":       {"create_database"},
		"name":                      {dbName},
	}
	body, err := w.whmGet(ctx, "/json-api/cpanel", q)
	if err != nil {
		return err
	}
	if err := parseUAPIResult(body); err != nil {
		return err
	}
	if dbUser == "" {
		return nil
	}
	userQ := url.Values{
		"cpanel_jsonapi_user":       {panelAccountID},
		"cpanel_jsonapi_apiversion": {"3"},
		"cpanel_jsonapi_module":     {"Mysql"},
		"cpanel_jsonapi_func":       {"create_user"},
		"name":                      {dbUser},
		"password":                  {randomPassword(18)},
	}
	userBody, err := w.whmGet(ctx, "/json-api/cpanel", userQ)
	if err != nil {
		return err
	}
	if err := parseUAPIResult(userBody); err != nil {
		return err
	}
	grantQ := url.Values{
		"cpanel_jsonapi_user":       {panelAccountID},
		"cpanel_jsonapi_apiversion": {"3"},
		"cpanel_jsonapi_module":     {"Mysql"},
		"cpanel_jsonapi_func":       {"set_privileges_on_database"},
		"user":                      {dbUser},
		"database":                  {dbName},
		"privileges":                {"ALL PRIVILEGES"},
	}
	grantBody, err := w.whmGet(ctx, "/json-api/cpanel", grantQ)
	if err != nil {
		return err
	}
	return parseUAPIResult(grantBody)
}

func (w whmAdapter) AddEmail(ctx context.Context, panelAccountID, address, password string) error {
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
	q := url.Values{
		"cpanel_jsonapi_user":       {panelAccountID},
		"cpanel_jsonapi_apiversion": {"3"},
		"cpanel_jsonapi_module":     {"Email"},
		"cpanel_jsonapi_func":       {"add_pop"},
		"email":                     {local},
		"domain":                    {domain},
		"password":                  {password},
		"quota":                     {"0"},
	}
	body, err := w.whmGet(ctx, "/json-api/cpanel", q)
	if err != nil {
		return err
	}
	return parseUAPIResult(body)
}

func (w whmAdapter) Renew(_ context.Context, _ string, days int) error {
	if days <= 0 {
		return fmt.Errorf("invalid period")
	}
	return nil
}

func (w whmAdapter) ChangePassword(ctx context.Context, panelAccountID, newPassword string) error {
	if len(newPassword) < 8 {
		return fmt.Errorf("password too short")
	}
	q := url.Values{
		"api.version": {"1"},
		"user":        {panelAccountID},
		"password":    {newPassword},
	}
	body, err := w.whmGet(ctx, "/json-api/passwd", q)
	if err != nil {
		return err
	}
	return whmAPI1Result(body)
}

func (w whmAdapter) Suspend(ctx context.Context, panelAccountID, reason string) error {
	if panelAccountID == "" {
		return fmt.Errorf("panel account id required")
	}
	q := url.Values{
		"api.version": {"1"},
		"user":        {panelAccountID},
	}
	if reason != "" {
		q.Set("reason", reason)
	}
	body, err := w.whmGet(ctx, "/json-api/suspendacct", q)
	if err != nil {
		return err
	}
	return whmAPI1Result(body)
}

func (w whmAdapter) Unsuspend(ctx context.Context, panelAccountID string) error {
	if panelAccountID == "" {
		return fmt.Errorf("panel account id required")
	}
	q := url.Values{
		"api.version": {"1"},
		"user":        {panelAccountID},
	}
	body, err := w.whmGet(ctx, "/json-api/unsuspendacct", q)
	if err != nil {
		return err
	}
	return whmAPI1Result(body)
}
