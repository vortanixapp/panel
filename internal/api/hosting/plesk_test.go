package hosting

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func pleskCfg(apiURL string) ServerConfig {
	return ServerConfig{PanelType: "plesk", APIURL: apiURL, APIUsername: "admin", APIToken: "adminpass"}
}

// pleskCLIParams достаёт аргументы утилиты из тела вызова CLI-шлюза.
func pleskCLIParams(t *testing.T, r *http.Request) []string {
	t.Helper()
	var body struct {
		Params []string `json:"params"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		t.Errorf("cli: тело не разобралось: %v", err)
	}
	return body.Params
}

func TestPleskAdapter_CreateAccount(t *testing.T) {
	var clientPayload, domainPayload map[string]any
	var basicUser, basicPass string
	var methods []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		basicUser, basicPass, _ = r.BasicAuth()
		methods = append(methods, r.Method+" "+r.URL.Path)
		switch r.URL.Path {
		case "/api/v2/clients":
			if err := json.NewDecoder(r.Body).Decode(&clientPayload); err != nil {
				t.Errorf("clients: тело не разобралось: %v", err)
			}
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, `{"id":5,"guid":"c-guid"}`)
		case "/api/v2/domains":
			if err := json.NewDecoder(r.Body).Decode(&domainPayload); err != nil {
				t.Errorf("domains: тело не разобралось: %v", err)
			}
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, `{"id":12,"guid":"d-guid"}`)
		default:
			t.Errorf("неожиданный путь: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	a := newPleskAdapter(pleskCfg(srv.URL))
	panelID, loginURL, err := a.CreateAccount(context.Background(), "vtx_user1", "example.com", "Unlimited")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if panelID != "vtx_user1" {
		t.Fatalf("идентификатор аккаунта Plesk — логин клиента, получено %q", panelID)
	}
	if loginURL != "https://example.com:8443" {
		t.Fatalf("unexpected login url: %s", loginURL)
	}
	if basicUser != "admin" || basicPass != "adminpass" {
		t.Fatalf("Basic-авторизация ушла не с теми доступами: %s/%s", basicUser, basicPass)
	}
	want := []string{"POST /api/v2/clients", "POST /api/v2/domains"}
	if len(methods) != 2 || methods[0] != want[0] || methods[1] != want[1] {
		t.Fatalf("ожидались %v, получено %v", want, methods)
	}
	if clientPayload["login"] != "vtx_user1" || clientPayload["type"] != "customer" {
		t.Fatalf("неожиданное тело создания клиента: %v", clientPayload)
	}
	if clientPayload["password"] == nil || clientPayload["password"] == "" {
		t.Fatal("клиент заводится без пароля")
	}
	if domainPayload["name"] != "example.com" || domainPayload["hosting_type"] != "virtual" {
		t.Fatalf("неожиданное тело создания домена: %v", domainPayload)
	}
	owner, _ := domainPayload["owner_client"].(map[string]any)
	if owner == nil || owner["login"] != "vtx_user1" {
		t.Fatalf("домен создан не для того клиента: %v", domainPayload)
	}
	plan, _ := domainPayload["plan"].(map[string]any)
	if plan == nil || plan["name"] != "Unlimited" {
		t.Fatalf("тариф не доехал: %v", domainPayload)
	}
}

func TestPleskAdapter_RESTErrorSurfaces(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"code":1013,"message":"Client with this login already exists"}`)
	}))
	defer srv.Close()

	a := newPleskAdapter(pleskCfg(srv.URL))
	_, _, err := a.CreateAccount(context.Background(), "u", "d.com", "")
	if err == nil {
		t.Fatal("ожидалась ошибка, панель отказала")
	}
	if !strings.Contains(err.Error(), "Client with this login already exists") {
		t.Fatalf("причина от панели потерялась: %v", err)
	}
}

func TestPleskAdapter_AddDatabaseResolvesDomain(t *testing.T) {
	var params []string
	var calls []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		switch r.URL.Path {
		case "/api/v2/clients":
			fmt.Fprint(w, `[{"id":4,"login":"other"},{"id":5,"login":"vtx_user1"}]`)
		case "/api/v2/clients/5/domains":
			fmt.Fprint(w, `[{"id":12,"name":"example.com"}]`)
		case "/api/v2/cli/database/call":
			params = pleskCLIParams(t, r)
			fmt.Fprint(w, `{"code":0,"stdout":"SUCCESS: Creation of database vtx_db1 complete","stderr":""}`)
		default:
			t.Errorf("неожиданный путь: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	a := newPleskAdapter(pleskCfg(srv.URL))
	if err := a.AddDatabase(context.Background(), "vtx_user1", "vtx_db1", "vtx_dbuser1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	joined := strings.Join(params, " ")
	if !strings.HasPrefix(joined, "--create vtx_db1 -domain example.com") {
		t.Fatalf("база создаётся не на том домене: %v", params)
	}
	if !strings.Contains(joined, "-add_user vtx_dbuser1") {
		t.Fatalf("пользователь БД не создаётся тем же вызовом: %v", params)
	}
	if len(calls) != 3 {
		t.Fatalf("ожидались поиск клиента, его домены и вызов CLI, получено %v", calls)
	}

	// Домен клиента запомнен: второй базе повторный поиск уже не нужен.
	if err := a.AddDatabase(context.Background(), "vtx_user1", "vtx_db2", ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(calls) != 4 {
		t.Fatalf("домен клиента должен браться из кэша, вызовы: %v", calls)
	}
	if strings.Contains(strings.Join(params, " "), "-add_user") {
		t.Fatalf("без имени пользователя -add_user слать нельзя: %v", params)
	}
}

func TestPleskAdapter_CLIFailureSurfaces(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Шлюз отдаёт 200 даже на неуспешной команде — решает code в теле.
		fmt.Fprint(w, `{"code":1,"stdout":"","stderr":"Database already exists"}`)
	}))
	defer srv.Close()

	a := newPleskAdapter(pleskCfg(srv.URL))
	err := a.AddEmail(context.Background(), "vtx_user1", "info@example.com", "s3cret!!")
	if err == nil {
		t.Fatal("ненулевой код утилиты должен подниматься как ошибка")
	}
	if !strings.Contains(err.Error(), "Database already exists") {
		t.Fatalf("stderr утилиты потерялся: %v", err)
	}
}

func TestPleskAdapter_AddEmail(t *testing.T) {
	var params []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/cli/mail/call" {
			t.Errorf("почта заводится утилитой mail, а вызвано %s", r.URL.Path)
		}
		params = pleskCLIParams(t, r)
		fmt.Fprint(w, `{"code":0,"stdout":"SUCCESS","stderr":""}`)
	}))
	defer srv.Close()

	a := newPleskAdapter(pleskCfg(srv.URL))
	if err := a.AddEmail(context.Background(), "vtx_user1", "info@example.com", "s3cret!!"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Join(params, " ") != "--create info@example.com -passwd s3cret!! -mailbox true" {
		t.Fatalf("неожиданные аргументы утилиты mail: %v", params)
	}
	if err := a.AddEmail(context.Background(), "vtx_user1", "not-an-email", "pw"); err == nil {
		t.Fatal("expected error for address without @")
	}
}

func TestPleskAdapter_SuspendUnsuspendAndPassword(t *testing.T) {
	var seen [][]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/cli/customer/call" {
			t.Errorf("операции над клиентом идут утилитой customer, а вызвано %s", r.URL.Path)
		}
		seen = append(seen, pleskCLIParams(t, r))
		fmt.Fprint(w, `{"code":0,"stdout":"SUCCESS","stderr":""}`)
	}))
	defer srv.Close()

	a := newPleskAdapter(pleskCfg(srv.URL))
	if err := a.Suspend(context.Background(), "vtx_user1", "expired"); err != nil {
		t.Fatalf("suspend: unexpected error: %v", err)
	}
	if err := a.Unsuspend(context.Background(), "vtx_user1"); err != nil {
		t.Fatalf("unsuspend: unexpected error: %v", err)
	}
	if err := a.ChangePassword(context.Background(), "vtx_user1", "supersecret"); err != nil {
		t.Fatalf("password: unexpected error: %v", err)
	}
	want := []string{
		"--suspend vtx_user1",
		"--activate vtx_user1",
		"--update vtx_user1 -passwd supersecret",
	}
	if len(seen) != len(want) {
		t.Fatalf("ожидалось %d вызовов, получено %v", len(want), seen)
	}
	for i, w := range want {
		if got := strings.Join(seen[i], " "); got != w {
			t.Fatalf("вызов %d: ожидалось %q, получено %q", i, w, got)
		}
	}
	if err := a.Suspend(context.Background(), "", "expired"); err == nil {
		t.Fatal("expected error on empty panel account id")
	}
	if err := a.ChangePassword(context.Background(), "vtx_user1", "short"); err == nil {
		t.Fatal("expected error for short password")
	}
}

func TestPleskAdapter_UnknownClientSurfaces(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[{"id":4,"login":"other"}]`)
	}))
	defer srv.Close()

	a := newPleskAdapter(pleskCfg(srv.URL))
	err := a.AddDatabase(context.Background(), "vtx_user1", "vtx_db1", "")
	if err == nil || !strings.Contains(err.Error(), "vtx_user1") {
		t.Fatalf("ожидалась внятная ошибка о ненайденном клиенте, получено: %v", err)
	}
}

func TestPleskAdapter_Renew_NoNetworkCall(t *testing.T) {
	a := newPleskAdapter(pleskCfg("http://127.0.0.1:1"))
	if err := a.Renew(context.Background(), "vtx_user1", 30); err != nil {
		t.Fatalf("продление не должно ходить в панель, получено: %v", err)
	}
	if err := a.Renew(context.Background(), "vtx_user1", 0); err == nil {
		t.Fatal("expected error for non-positive days")
	}
}
