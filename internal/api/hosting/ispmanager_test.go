package hosting

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func ispmanagerCfg(apiURL string) ServerConfig {
	return ServerConfig{PanelType: "ispmanager", APIURL: apiURL, APIUsername: "root", APIToken: "s3cret"}
}

// ispmanagerRecorder — панель-заглушка: копит разобранные POST-формы и отдаёт
// заранее заданные ответы по имени функции.
type ispmanagerRecorder struct {
	t         *testing.T
	responses map[string]string
	forms     []url.Values
	paths     []string
	queries   []string
	methods   []string
}

func (rec *ispmanagerRecorder) handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			rec.t.Errorf("форма не разобралась: %v", err)
		}
		rec.paths = append(rec.paths, r.URL.Path)
		rec.queries = append(rec.queries, r.URL.RawQuery)
		rec.methods = append(rec.methods, r.Method)
		rec.forms = append(rec.forms, r.PostForm)
		fn := r.PostForm.Get("func")
		if body, ok := rec.responses[fn]; ok {
			w.Write([]byte(body))
			return
		}
		w.Write([]byte(`{"doc":{"ok":{"$":"done"}}}`))
	}
}

func (rec *ispmanagerRecorder) form(fn string) url.Values {
	for _, f := range rec.forms {
		if f.Get("func") == fn {
			return f
		}
	}
	rec.t.Fatalf("вызов %s не состоялся, были: %v", fn, rec.funcs())
	return nil
}

func (rec *ispmanagerRecorder) funcs() []string {
	var out []string
	for _, f := range rec.forms {
		out = append(out, f.Get("func"))
	}
	return out
}

func TestISPmanagerAdapter_CreateAccount(t *testing.T) {
	rec := &ispmanagerRecorder{t: t, responses: map[string]string{}}
	srv := httptest.NewServer(rec.handler())
	defer srv.Close()

	a := newISPmanagerAdapter(ispmanagerCfg(srv.URL))
	panelID, loginURL, err := a.CreateAccount(context.Background(), "vtx_user1", "example.com", "starter")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if panelID != "vtx_user1" {
		t.Fatalf("в ISPmanager идентификатор — имя пользователя, получено %q", panelID)
	}
	if loginURL != srv.URL {
		t.Fatalf("вход в панель идёт на её же хост, получено %q", loginURL)
	}
	if got := rec.funcs(); len(got) != 2 || got[0] != "user.edit" || got[1] != "webdomain.edit" {
		t.Fatalf("ожидались user.edit и webdomain.edit, получено %v", got)
	}
	for i, p := range rec.paths {
		if p != "/ispmgr" {
			t.Fatalf("вызов %d ушёл на %s вместо /ispmgr", i, p)
		}
		if rec.methods[i] != http.MethodPost {
			t.Fatalf("вызов %d ушёл методом %s вместо POST", i, rec.methods[i])
		}
		if strings.Contains(rec.queries[i], "authinfo") {
			t.Fatalf("пароль администратора попал в query: %s", rec.queries[i])
		}
	}
	user := rec.form("user.edit")
	if user.Get("authinfo") != "root:s3cret" {
		t.Fatalf("неожиданный authinfo: %q", user.Get("authinfo"))
	}
	if user.Get("out") != "json" || user.Get("sok") != "ok" {
		t.Fatalf("без out=json и sok=ok панель ничего не сохранит: %v", user)
	}
	if user.Get("name") != "vtx_user1" || user.Get("preset") != "starter" {
		t.Fatalf("неожиданные параметры пользователя: %v", user)
	}
	if user.Get("passwd") == "" || user.Get("passwd") != user.Get("confirm") {
		t.Fatalf("пароль и подтверждение должны совпадать: %v", user)
	}
	site := rec.form("webdomain.edit")
	if site.Get("name") != "example.com" || site.Get("owner") != "vtx_user1" {
		t.Fatalf("домен создан не для того владельца: %v", site)
	}
}

func TestISPmanagerAdapter_PanelErrorSurfaces(t *testing.T) {
	rec := &ispmanagerRecorder{t: t, responses: map[string]string{
		"user.edit": `{"doc":{"error":{"$type":"exists","$object":"name","msg":{"$":"Пользователь уже существует"}}}}`,
	}}
	srv := httptest.NewServer(rec.handler())
	defer srv.Close()

	a := newISPmanagerAdapter(ispmanagerCfg(srv.URL))
	_, _, err := a.CreateAccount(context.Background(), "u", "d.com", "")
	if err == nil {
		t.Fatal("ожидалась ошибка, панель отказала")
	}
	if !strings.Contains(err.Error(), "Пользователь уже существует") {
		t.Fatalf("причина от панели потерялась: %v", err)
	}
	if len(rec.forms) != 1 {
		t.Fatalf("после отказа домен создавать нельзя, вызовов: %v", rec.funcs())
	}
}

func TestISPmanagerResult(t *testing.T) {
	if err := ispmanagerResult([]byte(`{"doc":{"ok":{"$":"done"}}}`)); err != nil {
		t.Fatalf("успешный ответ принят за ошибку: %v", err)
	}
	err := ispmanagerResult([]byte(`{"doc":{"error":{"$type":"exists","$object":"name"}}}`))
	if err == nil {
		t.Fatal("ошибка без msg должна оставаться ошибкой")
	}
	if !strings.Contains(err.Error(), "exists") || !strings.Contains(err.Error(), "name") {
		t.Fatalf("из ошибки без msg потерялись $type/$object: %v", err)
	}
	if err := ispmanagerResult([]byte(`<html>502</html>`)); err == nil {
		t.Fatal("нераспознанный ответ должен подниматься как ошибка")
	}
}

func TestISPmanagerAdapter_SuspendResume(t *testing.T) {
	rec := &ispmanagerRecorder{t: t, responses: map[string]string{}}
	srv := httptest.NewServer(rec.handler())
	defer srv.Close()

	a := newISPmanagerAdapter(ispmanagerCfg(srv.URL))
	if err := a.Suspend(context.Background(), "vtx_user1", "expired"); err != nil {
		t.Fatalf("suspend: unexpected error: %v", err)
	}
	if err := a.Unsuspend(context.Background(), "vtx_user1"); err != nil {
		t.Fatalf("resume: unexpected error: %v", err)
	}
	if got := rec.funcs(); len(got) != 2 || got[0] != "user.suspend" || got[1] != "user.resume" {
		t.Fatalf("ожидались user.suspend и user.resume, получено %v", got)
	}
	if rec.form("user.suspend").Get("elid") != "vtx_user1" {
		t.Fatalf("блокировка ушла не тому пользователю: %v", rec.form("user.suspend"))
	}
	if err := a.Suspend(context.Background(), "", "expired"); err == nil {
		t.Fatal("expected error on empty panel account id")
	}
}

func TestISPmanagerAdapter_AddEmailSplitsAddress(t *testing.T) {
	rec := &ispmanagerRecorder{t: t, responses: map[string]string{}}
	srv := httptest.NewServer(rec.handler())
	defer srv.Close()

	a := newISPmanagerAdapter(ispmanagerCfg(srv.URL))
	if err := a.AddEmail(context.Background(), "vtx_user1", "info@example.com", "s3cret!!"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	box := rec.form("email.box.edit")
	if box.Get("plid") != "example.com" || box.Get("name") != "info" {
		t.Fatalf("ящик разложен неверно: %v", box)
	}
	if box.Get("passwd") != "s3cret!!" || box.Get("confirm") != "s3cret!!" {
		t.Fatalf("пароль ящика не доехал: %v", box)
	}
	if err := a.AddEmail(context.Background(), "vtx_user1", "not-an-email", "pw"); err == nil {
		t.Fatal("expected error for address without @")
	}
}

func TestISPmanagerAdapter_ChangePasswordAndDatabase(t *testing.T) {
	rec := &ispmanagerRecorder{t: t, responses: map[string]string{}}
	srv := httptest.NewServer(rec.handler())
	defer srv.Close()

	a := newISPmanagerAdapter(ispmanagerCfg(srv.URL))
	if err := a.ChangePassword(context.Background(), "vtx_user1", "supersecret"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	pass := rec.form("user.edit")
	if pass.Get("elid") != "vtx_user1" || pass.Get("passwd") != "supersecret" {
		t.Fatalf("смена пароля ушла не тому пользователю: %v", pass)
	}
	if err := a.AddDatabase(context.Background(), "vtx_user1", "vtx_db1", "vtx_dbuser1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	db := rec.form("db.edit")
	if db.Get("name") != "vtx_db1" || db.Get("owner") != "vtx_user1" || db.Get("username") != "vtx_dbuser1" {
		t.Fatalf("неожиданные параметры базы: %v", db)
	}
	if db.Get("password") == "" || db.Get("password") != db.Get("confirm") {
		t.Fatalf("пароль пользователя БД не задан: %v", db)
	}
	if err := a.ChangePassword(context.Background(), "vtx_user1", "short"); err == nil {
		t.Fatal("expected error for short password")
	}
}

func TestISPmanagerAdapter_EndpointNotDoubled(t *testing.T) {
	rec := &ispmanagerRecorder{t: t, responses: map[string]string{}}
	srv := httptest.NewServer(rec.handler())
	defer srv.Close()

	a := newISPmanagerAdapter(ispmanagerCfg(srv.URL + "/ispmgr"))
	if err := a.Unsuspend(context.Background(), "vtx_user1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.paths[0] != "/ispmgr" {
		t.Fatalf("api_url с /ispmgr дал путь %s", rec.paths[0])
	}
}

func TestISPmanagerAdapter_Renew_NoNetworkCall(t *testing.T) {
	a := newISPmanagerAdapter(ispmanagerCfg("http://127.0.0.1:1"))
	if err := a.Renew(context.Background(), "vtx_user1", 30); err != nil {
		t.Fatalf("продление не должно ходить в панель, получено: %v", err)
	}
	if err := a.Renew(context.Background(), "vtx_user1", 0); err == nil {
		t.Fatal("expected error for non-positive days")
	}
}
