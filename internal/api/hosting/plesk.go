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

// Plesk: работаем через REST API /api/v2 с Basic-авторизацией — сессию заводить
// не нужно, поэтому и кэшировать нечего. Клиенты и домены в REST есть, а базы,
// почта и приостановка — нет; для них у Plesk штатный шлюз к своим же CLI-утилитам
// (POST /api/v2/cli/<утилита>/call), им и пользуемся, вместо старого XML-API.
type pleskAdapter struct {
	cfg    ServerConfig
	client *http.Client

	mu sync.Mutex
	// Логин клиента -> его первый домен. Нужен только для создания БД:
	// plesk bin database требует -domain, а в интерфейс Vortanix домен не приходит.
	domains map[string]string
}

func newPleskAdapter(cfg ServerConfig) *pleskAdapter {
	return &pleskAdapter{
		cfg: cfg,
		// Plesk на занятом сервере отвечает не сразу, но ждать без предела нельзя:
		// провижининг идёт внутри запроса от панели хостера.
		client:  &http.Client{Timeout: 30 * time.Second},
		domains: map[string]string{},
	}
}

func (p *pleskAdapter) Type() string { return "plesk" }

func (p *pleskAdapter) base() string { return strings.TrimSuffix(p.cfg.APIURL, "/") }

// pleskMessage достаёт причину отказа: REST-ошибка Plesk выглядит как
// {"code":1013,"message":"..."} — без message в интерфейс уходил бы голый статус.
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
	// API-ключ Plesk тоже подошёл бы (X-API-Key), но в ServerConfig хостер
	// заводит пару логин/пароль, а её принимает только Basic.
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

// cli дёргает утилиту Plesk через REST-шлюз. Шлюз отдаёт 200 и на неуспешной
// команде, поэтому решает не статус, а code в теле; текст ошибки утилиты
// приходит в stderr, а часть утилит пишет отказ в stdout.
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

// primaryDomain находит первый домен клиента: id клиента по логину из списка
// клиентов, затем его домены. Дороже одного запроса, но иначе создать базу
// нечем — CLI Plesk привязывает базу к домену, а не к учётной записи.
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
	// Домен создаётся вторым вызовом и сразу с хостингом: подписка без
	// hosting_type=virtual заводится «пустой», без веб-хостинга.
	site := map[string]any{
		"name":         domain,
		"hosting_type": "virtual",
		"owner_client": map[string]any{"login": username},
	}
	if planPackage != "" {
		// Тарифный план указываем именем — так его задаёт хостер в Vortanix;
		// без плана Plesk возьмёт настройки по умолчанию.
		site["plan"] = map[string]any{"name": planPackage}
	}
	if _, err := p.request(ctx, http.MethodPost, "/api/v2/domains", site); err != nil {
		return "", "", err
	}
	p.rememberDomain(username, domain)
	// Идентификатор аккаунта — логин клиента: все дальнейшие операции
	// (пароль, блокировка) в CLI Plesk адресуются именно логином.
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
		// Пользователь БД заводится тем же вызовом: отдельным заходом
		// Plesk потребовал бы уже существующую базу и лишний круг ошибок.
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
	// Адрес уже содержит домен, поэтому владелец здесь не нужен: Plesk сам
	// находит подписку по домену почтового адреса.
	return p.cli(ctx, "mail", []string{"--create", address, "-passwd", password, "-mailbox", "true"})
}

// Renew к панели не ходит: Plesk умеет хранить дату окончания подписки, но
// источник правды по сроку — core.hosting_accounts, и переписывать дату из
// двух мест значит рано или поздно закрыть клиенту рабочий аккаунт.
// По истечении срока аккаунт блокируется через Suspend (см. hosting_expiry.go).
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

// Suspend гасит учётную запись клиента целиком, вместе со всеми его подписками:
// причину Plesk не принимает, она остаётся в core.hosting_accounts.
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
