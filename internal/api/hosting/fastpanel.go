package hosting

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// FASTPANEL 2 отдаёт JSON-API на том же адресе, где живёт веб-панель (обычно
// порт 8888). Постоянного ключа, как в WHM, у панели нет: пара логин/пароль
// меняется на короткоживущий токен, поэтому токен держим в поле адаптера
// и перевыпускаем, когда панель отвечает 401. Адаптер создаётся на каждый
// вызов NewAdapter, так что кэш живёт ровно одну операцию — этого хватает,
// чтобы одна операция не логинилась по три раза.
type fastpanelAdapter struct {
	cfg    ServerConfig
	client *http.Client

	mu        sync.Mutex
	token     string
	loginPath string         // путь, на котором логин сработал: разные версии панели отвечают по-разному
	userIDs   map[string]int // логин -> id, чтобы не тянуть список пользователей на каждый шаг
}

func newFastpanelAdapter(cfg ServerConfig) *fastpanelAdapter {
	return &fastpanelAdapter{
		cfg: cfg,
		// Без таймаута зависшая панель держала бы горутину хендлера вечно:
		// провижининг идёт в запросе пользователя, ждать бесконечно нельзя.
		client:  &http.Client{Timeout: 30 * time.Second},
		userIDs: map[string]int{},
	}
}

func (f *fastpanelAdapter) Type() string { return "fastpanel" }

func (f *fastpanelAdapter) base() string { return strings.TrimSuffix(f.cfg.APIURL, "/") }

// panelURL — адрес, по которому клиент входит в саму панель. У FASTPANEL нет
// отдельного входа на домен (как :2083 у cPanel), клиент идёт на тот же хост,
// что и API, поэтому из APIURL берём только схему и хост.
func (f *fastpanelAdapter) panelURL() string {
	u, err := url.Parse(f.cfg.APIURL)
	if err != nil || u.Host == "" {
		return f.base()
	}
	return u.Scheme + "://" + u.Host
}

func (f *fastpanelAdapter) rawRequest(ctx context.Context, method, path string, payload []byte, token string) ([]byte, int, error) {
	var rdr io.Reader
	if payload != nil {
		rdr = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(ctx, method, f.base()+path, rdr)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		// Версии FASTPANEL расходятся в имени заголовка: одни ждут X-Auth-Token,
		// другие — обычный Bearer. Шлём оба: лишний заголовок панель игнорирует,
		// а иначе адаптер пришлось бы настраивать под версию панели у клиента.
		req.Header.Set("X-Auth-Token", token)
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := f.client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("fastpanel: не удалось дочитать ответ: %w", err)
	}
	return body, resp.StatusCode, nil
}

// fastpanelToken достаёт токен из ответа логина. Форма ответа у версий разная:
// плоский {"token":...}, обёрнутый {"data":{"token":...}} и OAuth-подобный
// {"access_token":...} встречаются все три.
func fastpanelToken(body []byte) string {
	var parsed struct {
		Token       string `json:"token"`
		AccessToken string `json:"access_token"`
		Data        struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return ""
	}
	for _, t := range []string{parsed.Token, parsed.Data.Token, parsed.AccessToken} {
		if t != "" {
			return t
		}
	}
	return ""
}

// fastpanelMessage вытаскивает человеческую причину отказа. Панель отвечает то
// {"message":...}, то {"error":...}, то {"detail":...} — берём первое непустое,
// чтобы в интерфейс хостера попал текст панели, а не голый код статуса.
func fastpanelMessage(body []byte, status int) string {
	var parsed struct {
		Message string `json:"message"`
		Error   string `json:"error"`
		Detail  string `json:"detail"`
	}
	if err := json.Unmarshal(body, &parsed); err == nil {
		for _, m := range []string{parsed.Message, parsed.Error, parsed.Detail} {
			if m != "" {
				return m
			}
		}
	}
	return fmt.Sprintf("http %d: %s", status, apiSnippet(body))
}

func fastpanelID(body []byte) (int, error) {
	var parsed struct {
		ID   int `json:"id"`
		Data struct {
			ID int `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return 0, fmt.Errorf("нераспознанный ответ: %s", apiSnippet(body))
	}
	if parsed.ID > 0 {
		return parsed.ID, nil
	}
	if parsed.Data.ID > 0 {
		return parsed.Data.ID, nil
	}
	return 0, fmt.Errorf("панель не вернула id: %s", apiSnippet(body))
}

func (f *fastpanelAdapter) login(ctx context.Context) (string, error) {
	creds, err := json.Marshal(map[string]string{
		"username": f.cfg.APIUsername,
		"password": f.cfg.APIToken,
	})
	if err != nil {
		return "", err
	}
	paths := []string{"/login", "/api/v1/login"}
	if f.loginPath != "" {
		paths = []string{f.loginPath}
	}
	var lastErr error
	for _, path := range paths {
		body, status, err := f.rawRequest(ctx, http.MethodPost, path, creds, "")
		if err != nil {
			return "", err
		}
		// 404/405 — этой версии панели такой путь неизвестен, пробуем следующий.
		// Остальные коды это уже отказ в авторизации, его нужно показать как есть.
		if status == http.StatusNotFound || status == http.StatusMethodNotAllowed {
			lastErr = fmt.Errorf("fastpanel: %s: %s", path, fastpanelMessage(body, status))
			continue
		}
		if status >= 300 {
			return "", fmt.Errorf("fastpanel: вход не удался: %s", fastpanelMessage(body, status))
		}
		token := fastpanelToken(body)
		if token == "" {
			return "", fmt.Errorf("fastpanel: вход не вернул токен: %s", apiSnippet(body))
		}
		f.loginPath = path
		return token, nil
	}
	return "", fmt.Errorf("fastpanel: не найден эндпоинт входа (%w)", lastErr)
}

func (f *fastpanelAdapter) ensureToken(ctx context.Context) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.token != "" {
		return f.token, nil
	}
	token, err := f.login(ctx)
	if err != nil {
		return "", err
	}
	f.token = token
	return token, nil
}

// dropToken сбрасывает только тот токен, по которому пришёл 401: если параллельная
// операция уже успела перелогиниться, её свежий токен затирать нельзя.
func (f *fastpanelAdapter) dropToken(used string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.token == used {
		f.token = ""
	}
}

func (f *fastpanelAdapter) doJSON(ctx context.Context, method, path string, payload any) ([]byte, error) {
	var raw []byte
	if payload != nil {
		var err error
		raw, err = json.Marshal(payload)
		if err != nil {
			return nil, err
		}
	}
	token, err := f.ensureToken(ctx)
	if err != nil {
		return nil, err
	}
	body, status, err := f.rawRequest(ctx, method, path, raw, token)
	if err != nil {
		return nil, err
	}
	if status == http.StatusUnauthorized {
		// Токен FASTPANEL живёт минутами: между операциями одного аккаунта он
		// вполне может протухнуть, поэтому один раз логинимся заново и повторяем.
		f.dropToken(token)
		token, err = f.ensureToken(ctx)
		if err != nil {
			return nil, err
		}
		body, status, err = f.rawRequest(ctx, method, path, raw, token)
		if err != nil {
			return nil, err
		}
	}
	if status >= 300 {
		return nil, fmt.Errorf("fastpanel: %s", fastpanelMessage(body, status))
	}
	return body, nil
}

// ownerID приводит panelAccountID к числовому id пользователя панели.
// В core.hosting_accounts panel_account_id может быть пустым, и тогда хендлеры
// подставляют username (COALESCE), поэтому принимаем оба вида и логин
// доразрешаем через список пользователей.
func (f *fastpanelAdapter) ownerID(ctx context.Context, panelAccountID string) (int, error) {
	if panelAccountID == "" {
		return 0, fmt.Errorf("panel account id required")
	}
	if id, err := strconv.Atoi(panelAccountID); err == nil {
		return id, nil
	}
	f.mu.Lock()
	id, ok := f.userIDs[panelAccountID]
	f.mu.Unlock()
	if ok {
		return id, nil
	}
	body, err := f.doJSON(ctx, http.MethodGet, "/api/v1/users", nil)
	if err != nil {
		return 0, err
	}
	var list []struct {
		ID       int    `json:"id"`
		Username string `json:"username"`
	}
	if err := json.Unmarshal(body, &list); err != nil {
		var wrapped struct {
			Data []struct {
				ID       int    `json:"id"`
				Username string `json:"username"`
			} `json:"data"`
		}
		if err := json.Unmarshal(body, &wrapped); err != nil {
			return 0, fmt.Errorf("fastpanel: нераспознанный список пользователей: %s", apiSnippet(body))
		}
		for _, u := range wrapped.Data {
			if u.Username == panelAccountID {
				f.rememberUser(u.Username, u.ID)
				return u.ID, nil
			}
		}
		return 0, fmt.Errorf("fastpanel: пользователь %q не найден в панели", panelAccountID)
	}
	for _, u := range list {
		if u.Username == panelAccountID {
			f.rememberUser(u.Username, u.ID)
			return u.ID, nil
		}
	}
	return 0, fmt.Errorf("fastpanel: пользователь %q не найден в панели", panelAccountID)
}

func (f *fastpanelAdapter) rememberUser(username string, id int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.userIDs[username] = id
}

func (f *fastpanelAdapter) CreateAccount(ctx context.Context, username, domain, planPackage string) (string, string, error) {
	if username == "" || domain == "" {
		return "", "", fmt.Errorf("username and domain required")
	}
	user := map[string]any{
		"username": username,
		"password": randomPassword(18),
		"role":     "user",
	}
	if planPackage != "" {
		// Тариф передаём только когда хостер его задал: в редакциях без тарифов
		// панель на пустое поле отвечает отказом, а без поля просто заводит
		// пользователя с лимитами по умолчанию.
		user["plan"] = planPackage
	}
	body, err := f.doJSON(ctx, http.MethodPost, "/api/v1/users", user)
	if err != nil {
		return "", "", err
	}
	id, err := fastpanelID(body)
	if err != nil {
		return "", "", fmt.Errorf("fastpanel: создание пользователя: %w", err)
	}
	f.rememberUser(username, id)
	// Сайт заводится вторым вызовом: в FASTPANEL пользователь и сайт — разные
	// объекты, аккаунт без сайта хостеру бесполезен. Если этот шаг упадёт,
	// ошибка поднимается наверх, а пользователь остаётся — так видно, что
	// именно не доехало, и повтор не создаёт второго пользователя.
	if _, err := f.doJSON(ctx, http.MethodPost, "/api/v1/sites", map[string]any{
		"domain": domain,
		"owner":  id,
	}); err != nil {
		return "", "", err
	}
	return strconv.Itoa(id), f.panelURL(), nil
}

func (f *fastpanelAdapter) AddDomain(ctx context.Context, panelAccountID, domain string) error {
	if domain == "" {
		return fmt.Errorf("domain required")
	}
	id, err := f.ownerID(ctx, panelAccountID)
	if err != nil {
		return err
	}
	_, err = f.doJSON(ctx, http.MethodPost, "/api/v1/sites", map[string]any{
		"domain": domain,
		"owner":  id,
	})
	return err
}

func (f *fastpanelAdapter) AddDatabase(ctx context.Context, panelAccountID, dbName, dbUser string) error {
	if dbName == "" {
		return fmt.Errorf("database name required")
	}
	id, err := f.ownerID(ctx, panelAccountID)
	if err != nil {
		return err
	}
	db := map[string]any{
		"name":  dbName,
		"owner": id,
		// Локальный сервер БД заводится при установке панели и всегда идёт
		// первым; выбирать его хостеру в интерфейсе Vortanix пока негде.
		"server": 1,
	}
	if dbUser != "" {
		db["username"] = dbUser
		db["password"] = randomPassword(18)
	}
	_, err = f.doJSON(ctx, http.MethodPost, "/api/v1/databases", db)
	return err
}

func (f *fastpanelAdapter) AddEmail(ctx context.Context, panelAccountID, address, password string) error {
	if address == "" {
		return fmt.Errorf("email required")
	}
	local, domain, ok := strings.Cut(address, "@")
	if !ok || local == "" || domain == "" {
		return fmt.Errorf("invalid email address %q", address)
	}
	id, err := f.ownerID(ctx, panelAccountID)
	if err != nil {
		return err
	}
	if password == "" {
		password = randomPassword(18)
	}
	_, err = f.doJSON(ctx, http.MethodPost, "/api/v1/mailboxes", map[string]any{
		"name":     local,
		"domain":   domain,
		"password": password,
		"owner":    id,
	})
	return err
}

// Renew ничего не спрашивает у панели: срока аренды у пользователя FASTPANEL нет,
// дата окончания живёт в core.hosting_accounts, а по её истечении аккаунт
// блокируется через Suspend (см. hosting_expiry.go).
func (f *fastpanelAdapter) Renew(_ context.Context, _ string, days int) error {
	if days <= 0 {
		return fmt.Errorf("invalid period")
	}
	return nil
}

func (f *fastpanelAdapter) ChangePassword(ctx context.Context, panelAccountID, newPassword string) error {
	if len(newPassword) < 8 {
		return fmt.Errorf("password too short")
	}
	id, err := f.ownerID(ctx, panelAccountID)
	if err != nil {
		return err
	}
	_, err = f.doJSON(ctx, http.MethodPatch, "/api/v1/users/"+strconv.Itoa(id), map[string]any{
		"password": newPassword,
	})
	return err
}

// Suspend выключает пользователя целиком: причину панель хранить негде,
// она остаётся в core.hosting_accounts на стороне Vortanix.
func (f *fastpanelAdapter) Suspend(ctx context.Context, panelAccountID, _ string) error {
	return f.setEnabled(ctx, panelAccountID, false)
}

func (f *fastpanelAdapter) Unsuspend(ctx context.Context, panelAccountID string) error {
	return f.setEnabled(ctx, panelAccountID, true)
}

func (f *fastpanelAdapter) setEnabled(ctx context.Context, panelAccountID string, enabled bool) error {
	id, err := f.ownerID(ctx, panelAccountID)
	if err != nil {
		return err
	}
	_, err = f.doJSON(ctx, http.MethodPatch, "/api/v1/users/"+strconv.Itoa(id), map[string]any{
		"enabled": enabled,
	})
	return err
}
