package hosting

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestWHMAPI1Result(t *testing.T) {
	if err := whmAPI1Result([]byte(`{"metadata":{"result":1,"reason":""}}`)); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if err := whmAPI1Result([]byte(`{"metadata":{"result":0,"reason":"Account already exists"}}`)); err == nil {
		t.Fatal("expected error on result 0")
	} else if err.Error() != "whm: Account already exists" {
		t.Fatalf("unexpected error message: %v", err)
	}
	if err := whmAPI1Result([]byte(`not json`)); err == nil {
		t.Fatal("expected error on invalid json")
	}
}

func TestParseUAPIResult(t *testing.T) {
	if err := parseUAPIResult([]byte(`{"status":1,"errors":null}`)); err != nil {
		t.Fatalf("expected success (top-level), got %v", err)
	}
	if err := parseUAPIResult([]byte(`{"result":{"status":1,"errors":null}}`)); err != nil {
		t.Fatalf("expected success (wrapped), got %v", err)
	}
	if err := parseUAPIResult([]byte(`{"status":0,"errors":["Database already exists."]}`)); err == nil {
		t.Fatal("expected error on status 0")
	}
	if err := parseUAPIResult([]byte(`{}`)); err == nil {
		t.Fatal("expected error on unrecognized shape")
	}
}

func TestWHMAdapter_CreateAccount(t *testing.T) {
	var gotPath string
	var gotQuery url.Values
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.Query()
		gotAuth = r.Header.Get("Authorization")
		w.Write([]byte(`{"metadata":{"result":1,"reason":""}}`))
	}))
	defer srv.Close()

	a := whmAdapter{cfg: ServerConfig{PanelType: "cpanel", APIURL: srv.URL, APIUsername: "root", APIToken: "tok123"}}
	panelID, loginURL, err := a.CreateAccount(context.Background(), "vtx_user1", "example.com", "starter")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if panelID != "vtx_user1" {
		t.Fatalf("unexpected panel account id: %s", panelID)
	}
	if loginURL != "https://example.com:2083" {
		t.Fatalf("unexpected login url: %s", loginURL)
	}
	if gotPath != "/json-api/createacct" {
		t.Fatalf("unexpected path: %s", gotPath)
	}
	if gotQuery.Get("username") != "vtx_user1" || gotQuery.Get("domain") != "example.com" || gotQuery.Get("plan") != "starter" {
		t.Fatalf("unexpected query: %v", gotQuery)
	}
	if gotAuth != "whm root:tok123" {
		t.Fatalf("unexpected Authorization header: %s", gotAuth)
	}
}

func TestWHMAdapter_CreateAccount_Failure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"metadata":{"result":0,"reason":"Account already exists"}}`))
	}))
	defer srv.Close()

	a := whmAdapter{cfg: ServerConfig{PanelType: "cpanel", APIURL: srv.URL, APIUsername: "root", APIToken: "tok"}}
	_, _, err := a.CreateAccount(context.Background(), "u", "d.com", "p")
	if err == nil {
		t.Fatal("expected error when WHM reports failure")
	}
}

func TestWHMAdapter_AddDatabase_CreatesUserAndGrants(t *testing.T) {
	var calls []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.URL.Query().Get("cpanel_jsonapi_func"))
		w.Write([]byte(`{"status":1,"errors":null}`))
	}))
	defer srv.Close()

	a := whmAdapter{cfg: ServerConfig{PanelType: "cpanel", APIURL: srv.URL, APIUsername: "root", APIToken: "tok"}}
	if err := a.AddDatabase(context.Background(), "vtx_user1", "vtx_db1", "vtx_dbuser1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"create_database", "create_user", "set_privileges_on_database"}
	if len(calls) != len(want) {
		t.Fatalf("expected calls %v, got %v", want, calls)
	}
	for i, w := range want {
		if calls[i] != w {
			t.Fatalf("call %d: expected %s, got %s", i, w, calls[i])
		}
	}
}

func TestWHMAdapter_AddEmail_SplitsAddress(t *testing.T) {
	var gotQuery url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		w.Write([]byte(`{"status":1,"errors":null}`))
	}))
	defer srv.Close()

	a := whmAdapter{cfg: ServerConfig{PanelType: "cpanel", APIURL: srv.URL, APIUsername: "root", APIToken: "tok"}}
	if err := a.AddEmail(context.Background(), "vtx_user1", "info@example.com", "s3cret!!"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotQuery.Get("email") != "info" || gotQuery.Get("domain") != "example.com" {
		t.Fatalf("unexpected email/domain split: email=%s domain=%s", gotQuery.Get("email"), gotQuery.Get("domain"))
	}
}

func TestWHMAdapter_AddEmail_InvalidAddress(t *testing.T) {
	a := whmAdapter{cfg: ServerConfig{PanelType: "cpanel", APIURL: "http://unused", APIUsername: "root", APIToken: "tok"}}
	if err := a.AddEmail(context.Background(), "u", "not-an-email", "pw"); err == nil {
		t.Fatal("expected error for address without @")
	}
}

func TestWHMAdapter_Renew_NoNetworkCall(t *testing.T) {
	a := whmAdapter{cfg: ServerConfig{PanelType: "cpanel", APIURL: "http://127.0.0.1:1", APIUsername: "root", APIToken: "tok"}}
	if err := a.Renew(context.Background(), "vtx_user1", 30); err != nil {
		t.Fatalf("expected renew to succeed without any panel call, got %v", err)
	}
	if err := a.Renew(context.Background(), "vtx_user1", 0); err == nil {
		t.Fatal("expected error for non-positive days")
	}
}

func TestNewAdapter_UnconfiguredCpanelReturnsHonestError(t *testing.T) {
	a := NewAdapter(ServerConfig{PanelType: "cpanel"})
	_, _, err := a.CreateAccount(context.Background(), "u", "d.com", "p")
	if err == nil {
		t.Fatal("expected an explicit error, not a silent fake success, when cpanel credentials are missing")
	}
}

func TestNewAdapter_UnconfiguredPanelReturnsHonestError(t *testing.T) {
	for _, panelType := range []string{"fastpanel", "ispmanager", "plesk"} {
		a := NewAdapter(ServerConfig{PanelType: panelType})
		_, _, err := a.CreateAccount(context.Background(), "u", "d.com", "p")
		if err == nil {
			t.Fatalf("expected an explicit error for %s without credentials, not a silent fake success", panelType)
		}
	}
}

func TestNewAdapter_ConfiguredPanelsGetRealAdapters(t *testing.T) {
	for _, panelType := range []string{"fastpanel", "ispmanager", "plesk"} {
		a := NewAdapter(ServerConfig{PanelType: panelType, APIURL: "https://panel.example", APIUsername: "u", APIToken: "t"})
		if _, stub := a.(stubAdapter); stub {
			t.Fatalf("%s: адаптер реализован, заглушка больше не нужна", panelType)
		}
		if a.Type() != panelType {
			t.Fatalf("expected adapter type %s, got %s", panelType, a.Type())
		}
	}
}

func TestWHMAdapter_SuspendUnsuspend(t *testing.T) {
	var gotPath string
	var gotQuery url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.Query()
		w.Write([]byte(`{"metadata":{"result":1,"reason":""}}`))
	}))
	defer srv.Close()

	a := whmAdapter{cfg: ServerConfig{PanelType: "cpanel", APIURL: srv.URL, APIUsername: "root", APIToken: "tok123"}}

	if err := a.Suspend(context.Background(), "vtx_user1", "expired"); err != nil {
		t.Fatalf("suspend: unexpected error: %v", err)
	}
	if gotPath != "/json-api/suspendacct" {
		t.Fatalf("unexpected suspend path: %s", gotPath)
	}
	if gotQuery.Get("user") != "vtx_user1" || gotQuery.Get("reason") != "expired" {
		t.Fatalf("unexpected suspend query: %v", gotQuery)
	}

	if err := a.Unsuspend(context.Background(), "vtx_user1"); err != nil {
		t.Fatalf("unsuspend: unexpected error: %v", err)
	}
	if gotPath != "/json-api/unsuspendacct" {
		t.Fatalf("unexpected unsuspend path: %s", gotPath)
	}
	if gotQuery.Get("user") != "vtx_user1" {
		t.Fatalf("unexpected unsuspend query: %v", gotQuery)
	}

	if err := a.Suspend(context.Background(), "", "expired"); err == nil {
		t.Fatal("expected error on empty panel account id")
	}
}

func TestStubAdapter_SuspendReportsError(t *testing.T) {
	// Заглушка осталась только для неизвестного типа панели: у fastpanel,
	// ispmanager и plesk теперь настоящие адаптеры.
	a := NewAdapter(ServerConfig{PanelType: "directadmin", APIURL: "https://x", APIUsername: "u", APIToken: "t"})
	if err := a.Suspend(context.Background(), "acc", "expired"); err == nil {
		t.Fatal("expected explicit error from stub adapter")
	}
	if err := a.Unsuspend(context.Background(), "acc"); err == nil {
		t.Fatal("expected explicit error from stub adapter")
	}
}
