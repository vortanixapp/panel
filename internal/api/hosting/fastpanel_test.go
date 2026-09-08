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

func fastpanelCfg(apiURL string) ServerConfig {
	return ServerConfig{PanelType: "fastpanel", APIURL: apiURL, APIUsername: "admin", APIToken: "adminpass"}
}

func TestFastpanelAdapter_CreateAccount(t *testing.T) {
	var loginBody map[string]string
	var authToken, authBearer, userMethod string
	var userPayload, sitePayload map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			if err := json.NewDecoder(r.Body).Decode(&loginBody); err != nil {
				t.Errorf("login: тело не разобралось: %v", err)
			}
			fmt.Fprint(w, `{"token":"tok-1"}`)
		case "/api/v1/users":
			userMethod = r.Method
			authToken = r.Header.Get("X-Auth-Token")
			authBearer = r.Header.Get("Authorization")
			if err := json.NewDecoder(r.Body).Decode(&userPayload); err != nil {
				t.Errorf("users: тело не разобралось: %v", err)
			}
			fmt.Fprint(w, `{"id":7}`)
		case "/api/v1/sites":
			if err := json.NewDecoder(r.Body).Decode(&sitePayload); err != nil {
				t.Errorf("sites: тело не разобралось: %v", err)
			}
			fmt.Fprint(w, `{"id":11}`)
		default:
			t.Errorf("неожиданный путь: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	a := newFastpanelAdapter(fastpanelCfg(srv.URL))
	panelID, loginURL, err := a.CreateAccount(context.Background(), "vtx_user1", "example.com", "starter")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if panelID != "7" {
		t.Fatalf("ожидался id пользователя панели, получено %q", panelID)
	}
	if loginURL != srv.URL {
		t.Fatalf("вход в FASTPANEL идёт на сам хост панели, получено %q", loginURL)
	}
	if loginBody["username"] != "admin" || loginBody["password"] != "adminpass" {
		t.Fatalf("логин ушёл не с теми доступами: %v", loginBody)
	}
	if userMethod != http.MethodPost {
		t.Fatalf("создание пользователя должно идти POST, а не %s", userMethod)
	}
	if authToken != "tok-1" || authBearer != "Bearer tok-1" {
		t.Fatalf("токен не доехал в заголовках: X-Auth-Token=%q Authorization=%q", authToken, authBearer)
	}
	if userPayload["username"] != "vtx_user1" || userPayload["plan"] != "starter" {
		t.Fatalf("неожиданное тело создания пользователя: %v", userPayload)
	}
	if userPayload["password"] == nil || userPayload["password"] == "" {
		t.Fatal("пользователь заводится без пароля")
	}
	if sitePayload["domain"] != "example.com" || sitePayload["owner"] != float64(7) {
		t.Fatalf("сайт создан не для того владельца: %v", sitePayload)
	}
}

func TestFastpanelAdapter_CreateAccount_WithoutPlan(t *testing.T) {
	var userPayload map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			fmt.Fprint(w, `{"token":"t"}`)
		case "/api/v1/users":
			if err := json.NewDecoder(r.Body).Decode(&userPayload); err != nil {
				t.Errorf("users: тело не разобралось: %v", err)
			}
			fmt.Fprint(w, `{"id":3}`)
		default:
			fmt.Fprint(w, `{"id":1}`)
		}
	}))
	defer srv.Close()

	a := newFastpanelAdapter(fastpanelCfg(srv.URL))
	if _, _, err := a.CreateAccount(context.Background(), "u", "d.com", ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := userPayload["plan"]; ok {
		t.Fatalf("пустой тариф не должен уезжать в панель: %v", userPayload)
	}
}

func TestFastpanelAdapter_LoginFallbackPath(t *testing.T) {
	var tried []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			tried = append(tried, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		case "/api/v1/login":
			tried = append(tried, r.URL.Path)
			fmt.Fprint(w, `{"data":{"token":"tok-v1"}}`)
		case "/api/v1/users/4":
			if got := r.Header.Get("X-Auth-Token"); got != "tok-v1" {
				t.Errorf("ожидался токен tok-v1, получен %q", got)
			}
			fmt.Fprint(w, `{"id":4}`)
		default:
			t.Errorf("неожиданный путь: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	a := newFastpanelAdapter(fastpanelCfg(srv.URL))
	if err := a.ChangePassword(context.Background(), "4", "supersecret"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tried) != 2 || tried[0] != "/login" || tried[1] != "/api/v1/login" {
		t.Fatalf("ожидался перебор путей входа, получено %v", tried)
	}
}

func TestFastpanelAdapter_RelogsOnExpiredToken(t *testing.T) {
	logins := 0
	var seen []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			logins++
			fmt.Fprintf(w, `{"token":"tok-%d"}`, logins)
		case "/api/v1/users/5":
			seen = append(seen, r.Header.Get("X-Auth-Token"))
			if len(seen) == 1 {
				w.WriteHeader(http.StatusUnauthorized)
				fmt.Fprint(w, `{"message":"token expired"}`)
				return
			}
			fmt.Fprint(w, `{"id":5}`)
		default:
			t.Errorf("неожиданный путь: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	a := newFastpanelAdapter(fastpanelCfg(srv.URL))
	if err := a.Suspend(context.Background(), "5", "expired"); err != nil {
		t.Fatalf("после перелогина операция должна пройти, получено: %v", err)
	}
	if logins != 2 {
		t.Fatalf("ожидался повторный вход после 401, входов: %d", logins)
	}
	if len(seen) != 2 || seen[0] != "tok-1" || seen[1] != "tok-2" {
		t.Fatalf("повтор ушёл не со свежим токеном: %v", seen)
	}
}

func TestFastpanelAdapter_PanelErrorSurfaces(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/login" {
			fmt.Fprint(w, `{"token":"t"}`)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"message":"User already exists"}`)
	}))
	defer srv.Close()

	a := newFastpanelAdapter(fastpanelCfg(srv.URL))
	_, _, err := a.CreateAccount(context.Background(), "u", "d.com", "")
	if err == nil {
		t.Fatal("ожидалась ошибка, панель отказала")
	}
	if !strings.Contains(err.Error(), "User already exists") {
		t.Fatalf("причина от панели потерялась: %v", err)
	}
}

func TestFastpanelAdapter_BadLoginSurfaces(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"message":"Invalid credentials"}`)
	}))
	defer srv.Close()

	a := newFastpanelAdapter(fastpanelCfg(srv.URL))
	err := a.AddDomain(context.Background(), "3", "example.com")
	if err == nil || !strings.Contains(err.Error(), "Invalid credentials") {
		t.Fatalf("ожидалась ошибка входа с текстом панели, получено: %v", err)
	}
}

func TestFastpanelAdapter_ResolvesUsernameToID(t *testing.T) {
	var mailbox map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			fmt.Fprint(w, `{"token":"t"}`)
		case "/api/v1/users":
			fmt.Fprint(w, `[{"id":2,"username":"other"},{"id":9,"username":"vtx_user1"}]`)
		case "/api/v1/mailboxes":
			if err := json.NewDecoder(r.Body).Decode(&mailbox); err != nil {
				t.Errorf("mailboxes: тело не разобралось: %v", err)
			}
			fmt.Fprint(w, `{"id":1}`)
		default:
			t.Errorf("неожиданный путь: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	a := newFastpanelAdapter(fastpanelCfg(srv.URL))
	if err := a.AddEmail(context.Background(), "vtx_user1", "info@example.com", "s3cret!!"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mailbox["name"] != "info" || mailbox["domain"] != "example.com" || mailbox["owner"] != float64(9) {
		t.Fatalf("ящик создан не по тем данным: %v", mailbox)
	}
}

func TestFastpanelAdapter_AddDatabaseSendsUser(t *testing.T) {
	var db map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			fmt.Fprint(w, `{"token":"t"}`)
		case "/api/v1/databases":
			if err := json.NewDecoder(r.Body).Decode(&db); err != nil {
				t.Errorf("databases: тело не разобралось: %v", err)
			}
			fmt.Fprint(w, `{"id":1}`)
		default:
			t.Errorf("неожиданный путь: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	a := newFastpanelAdapter(fastpanelCfg(srv.URL))
	if err := a.AddDatabase(context.Background(), "9", "vtx_db1", "vtx_dbuser1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if db["name"] != "vtx_db1" || db["owner"] != float64(9) || db["username"] != "vtx_dbuser1" {
		t.Fatalf("неожиданное тело создания базы: %v", db)
	}
	if db["password"] == nil || db["password"] == "" {
		t.Fatal("пользователь БД заведён без пароля")
	}
}

func TestFastpanelAdapter_Renew_NoNetworkCall(t *testing.T) {
	a := newFastpanelAdapter(fastpanelCfg("http://127.0.0.1:1"))
	if err := a.Renew(context.Background(), "7", 30); err != nil {
		t.Fatalf("продление не должно ходить в панель, получено: %v", err)
	}
	if err := a.Renew(context.Background(), "7", 0); err == nil {
		t.Fatal("expected error for non-positive days")
	}
}

func TestFastpanelAdapter_AddEmail_InvalidAddress(t *testing.T) {
	a := newFastpanelAdapter(fastpanelCfg("http://127.0.0.1:1"))
	if err := a.AddEmail(context.Background(), "7", "not-an-email", "pw"); err == nil {
		t.Fatal("expected error for address without @")
	}
}
