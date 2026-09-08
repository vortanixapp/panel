package hosting

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ISPmanager 5/6 говорит одним эндпоинтом: функция выбирается параметром func,
// формат ответа — out, авторизация — authinfo=логин:пароль в каждом запросе.
// Сессию панель тоже умеет выдавать, но она короткая и её всё равно пришлось бы
// обновлять, поэтому проще каждый раз слать authinfo — как и рекомендует ISPsystem.
type ispmanagerAdapter struct {
	cfg    ServerConfig
	client *http.Client
}

func newISPmanagerAdapter(cfg ServerConfig) *ispmanagerAdapter {
	return &ispmanagerAdapter{
		cfg: cfg,
		// Панель на нагруженной ноде отвечает медленно, но не бесконечно:
		// без таймаута повисший запрос держал бы хендлер провижининга.
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (i *ispmanagerAdapter) Type() string { return "ispmanager" }

// endpoint терпит оба вида настройки: и «https://host:1500», и «https://host:1500/ispmgr»
// — хостеры вписывают в api_url то одно, то другое, а второй /ispmgr в пути
// панель уже не понимает.
func (i *ispmanagerAdapter) endpoint() string {
	base := strings.TrimSuffix(i.cfg.APIURL, "/")
	if strings.HasSuffix(base, "/ispmgr") {
		return base
	}
	return base + "/ispmgr"
}

// panelURL — адрес входа клиента в панель: у ISPmanager это тот же хост, что и API.
func (i *ispmanagerAdapter) panelURL() string {
	u, err := url.Parse(i.cfg.APIURL)
	if err != nil || u.Host == "" {
		return strings.TrimSuffix(i.cfg.APIURL, "/")
	}
	return u.Scheme + "://" + u.Host
}

// call шлёт вызов POST-формой, а не через query. Пароль администратора в
// authinfo иначе оседал бы в access-логе nginx панели и в логах прокси.
func (i *ispmanagerAdapter) call(ctx context.Context, fn string, params url.Values) ([]byte, error) {
	form := url.Values{}
	for k, v := range params {
		form[k] = v
	}
	form.Set("func", fn)
	form.Set("out", "json")
	form.Set("authinfo", i.cfg.APIUsername+":"+i.cfg.APIToken)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, i.endpoint(), strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	resp, err := i.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ispmanager: не удалось дочитать ответ: %w", err)
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("ispmanager http %d: %s", resp.StatusCode, apiSnippet(body))
	}
	// Отказ ISPmanager отдаёт с кодом 200 и телом doc.error, поэтому статуса мало.
	if err := ispmanagerResult(body); err != nil {
		return nil, err
	}
	return body, nil
}

// ispmanagerResult разбирает doc.error. Текст ошибки лежит в msg.$, но у части
// функций его нет и остаются только $type/$object («exists», «name») — тогда
// собираем причину из них, иначе хостер увидел бы пустое сообщение.
func ispmanagerResult(body []byte) error {
	var parsed struct {
		Doc struct {
			Error *struct {
				Type   string `json:"$type"`
				Object string `json:"$object"`
				Detail string `json:"$detail"`
				Value  string `json:"$value"`
				Msg    struct {
					Text string `json:"$"`
				} `json:"msg"`
			} `json:"error"`
		} `json:"doc"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return fmt.Errorf("ispmanager: нераспознанный ответ: %s", apiSnippet(body))
	}
	e := parsed.Doc.Error
	if e == nil {
		return nil
	}
	if e.Msg.Text != "" {
		return fmt.Errorf("ispmanager: %s", e.Msg.Text)
	}
	parts := make([]string, 0, 3)
	for _, p := range []string{e.Type, e.Object, e.Value, e.Detail} {
		if p != "" {
			parts = append(parts, p)
		}
	}
	if len(parts) == 0 {
		return fmt.Errorf("ispmanager: панель вернула ошибку без описания")
	}
	return fmt.Errorf("ispmanager: %s", strings.Join(parts, " "))
}

func (i *ispmanagerAdapter) CreateAccount(ctx context.Context, username, domain, planPackage string) (string, string, error) {
	if username == "" || domain == "" {
		return "", "", fmt.Errorf("username and domain required")
	}
	password := randomPassword(18)
	// В API ISPsystem создание и правка — одна и та же функция <объект>.edit:
	// без elid это создание, с elid — изменение. sok=ok означает «форма
	// отправлена», без него панель вернёт описание формы и ничего не сохранит.
	user := url.Values{
		"sok":     {"ok"},
		"name":    {username},
		"passwd":  {password},
		"confirm": {password},
	}
	if planPackage != "" {
		user.Set("preset", planPackage)
	}
	if _, err := i.call(ctx, "user.edit", user); err != nil {
		return "", "", err
	}
	// Сайт заводится отдельно: пользователь ISPmanager сам по себе доменов не имеет.
	if err := i.addWebDomain(ctx, username, domain); err != nil {
		return "", "", err
	}
	// Идентификатор пользователя в ISPmanager — его имя, отдельного id нет.
	return username, i.panelURL(), nil
}

func (i *ispmanagerAdapter) addWebDomain(ctx context.Context, owner, domain string) error {
	_, err := i.call(ctx, "webdomain.edit", url.Values{
		"sok":   {"ok"},
		"name":  {domain},
		"owner": {owner},
	})
	return err
}

func (i *ispmanagerAdapter) AddDomain(ctx context.Context, panelAccountID, domain string) error {
	if domain == "" {
		return fmt.Errorf("domain required")
	}
	if panelAccountID == "" {
		return fmt.Errorf("panel account id required")
	}
	return i.addWebDomain(ctx, panelAccountID, domain)
}

func (i *ispmanagerAdapter) AddDatabase(ctx context.Context, panelAccountID, dbName, dbUser string) error {
	if dbName == "" {
		return fmt.Errorf("database name required")
	}
	if panelAccountID == "" {
		return fmt.Errorf("panel account id required")
	}
	params := url.Values{
		"sok":   {"ok"},
		"name":  {dbName},
		"owner": {panelAccountID},
	}
	// Сервер БД не указываем: панель подставит сервер по умолчанию, а угадывать
	// его имя за хостера значило бы ломаться на любой нестандартной установке.
	if dbUser != "" {
		password := randomPassword(18)
		params.Set("username", dbUser)
		params.Set("password", password)
		params.Set("confirm", password)
	}
	_, err := i.call(ctx, "db.edit", params)
	return err
}

func (i *ispmanagerAdapter) AddEmail(ctx context.Context, _, address, password string) error {
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
	// Ящики в ISPmanager вложены в почтовый домен, поэтому родитель (plid) —
	// сам домен; создавать сюда же почтовый домен нельзя: если он уже заведён,
	// панель ответит отказом и настоящая ошибка ящика потеряется.
	_, err := i.call(ctx, "email.box.edit", url.Values{
		"sok":     {"ok"},
		"plid":    {domain},
		"name":    {local},
		"passwd":  {password},
		"confirm": {password},
	})
	return err
}

// Renew к панели не ходит: срок аренды ведётся в core.hosting_accounts, а
// ISPmanager по истечении срока ничего сам не делает — блокировка идёт
// отдельной операцией Suspend (см. hosting_expiry.go).
func (i *ispmanagerAdapter) Renew(_ context.Context, _ string, days int) error {
	if days <= 0 {
		return fmt.Errorf("invalid period")
	}
	return nil
}

func (i *ispmanagerAdapter) ChangePassword(ctx context.Context, panelAccountID, newPassword string) error {
	if len(newPassword) < 8 {
		return fmt.Errorf("password too short")
	}
	if panelAccountID == "" {
		return fmt.Errorf("panel account id required")
	}
	// Здесь та же user.edit, но с elid — правка существующего пользователя.
	_, err := i.call(ctx, "user.edit", url.Values{
		"sok":     {"ok"},
		"elid":    {panelAccountID},
		"passwd":  {newPassword},
		"confirm": {newPassword},
	})
	return err
}

// Suspend: причину панель не принимает, она остаётся в core.hosting_accounts.
func (i *ispmanagerAdapter) Suspend(ctx context.Context, panelAccountID, _ string) error {
	if panelAccountID == "" {
		return fmt.Errorf("panel account id required")
	}
	_, err := i.call(ctx, "user.suspend", url.Values{"elid": {panelAccountID}})
	return err
}

func (i *ispmanagerAdapter) Unsuspend(ctx context.Context, panelAccountID string) error {
	if panelAccountID == "" {
		return fmt.Errorf("panel account id required")
	}
	_, err := i.call(ctx, "user.resume", url.Values{"elid": {panelAccountID}})
	return err
}
