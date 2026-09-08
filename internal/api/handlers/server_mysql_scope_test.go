package handlers

import "testing"

const testServerID = "53f5e577-1341-4b93-9253-ba6637c9e8a3"

func TestForbidsTouchingSystemAndForeignNames(t *testing.T) {
	forbidden := []string{
		"mysql",
		"information_schema",
		"performance_schema",
		"sys",
		"system",
		"root",
		"srv_deadbeef",
		"srv_deadbeef_shop",
		"other_db",
	}
	for _, name := range forbidden {
		if msg := validateMysqlName(testServerID, "имя базы", name); msg == "" {
			t.Errorf("%q разрешено к изменению, а не должно", name)
		}
	}
}

func TestAllowsOwnNamespace(t *testing.T) {
	allowed := []string{
		"srv_53f5e577",
		"srv_53f5e577_shop",
		"srv_53f5e577_stats2",
	}
	for _, name := range allowed {
		if msg := validateMysqlName(testServerID, "имя базы", name); msg != "" {
			t.Errorf("%q запрещено, хотя принадлежит серверу: %s", name, msg)
		}
	}
}

func TestRejectsInjectionAttempts(t *testing.T) {
	nasty := []string{
		"srv_53f5e577`; DROP DATABASE mysql; --",
		"srv_53f5e577' OR '1'='1",
		"srv_53f5e577 mysql",
		"srv_53f5e577;",
		"",
		"   ",
	}
	for _, name := range nasty {
		if msg := validateMysqlName(testServerID, "имя базы", name); msg == "" {
			t.Errorf("%q принято, хотя содержит недопустимые символы", name)
		}
	}
}

func TestCatalogHidesForeignEntries(t *testing.T) {
	catalog := map[string]any{
		"container": "vortanix-mysql80-3306",
		"databases": []any{
			"information_schema", "mysql", "performance_schema", "sys", "system",
			"srv_53f5e577", "srv_53f5e577_shop", "srv_deadbeef",
		},
		"users": []any{
			map[string]any{"username": "root", "host": "%"},
			map[string]any{"username": "root", "host": "localhost"},
			map[string]any{"username": "mysql.sys", "host": "localhost"},
			map[string]any{"username": "srv_53f5e577", "host": "%"},
			map[string]any{"username": "srv_deadbeef", "host": "%"},
		},
	}

	out := filterMysqlCatalog(testServerID, catalog)

	dbs, _ := out["databases"].([]string)
	if len(dbs) != 2 {
		t.Fatalf("в каталоге %d баз, ожидалось 2: %v", len(dbs), dbs)
	}
	for _, db := range dbs {
		if !ownsMysqlName(testServerID, db) {
			t.Errorf("чужая база %q попала в каталог", db)
		}
	}

	users, _ := out["users"].([]map[string]string)
	if len(users) != 1 {
		t.Fatalf("в каталоге %d пользователей, ожидался 1: %v", len(users), users)
	}
	if users[0]["username"] != "srv_53f5e577" {
		t.Errorf("в каталоге чужой пользователь %q", users[0]["username"])
	}

	if _, leaked := out["container"]; leaked {
		t.Error("имя контейнера СУБД раскрыто клиенту")
	}
}

func TestNamespaceDerivation(t *testing.T) {
	if got := mysqlNamespace(testServerID); got != "srv_53f5e577" {
		t.Errorf("пространство имён %q", got)
	}
	if got := mysqlNamespace("abc"); got != "" {
		t.Errorf("короткий id дал пространство имён %q", got)
	}
	if msg := validateMysqlName("abc", "имя базы", "srv_abc"); msg == "" {
		t.Error("при отсутствии пространства имён операция разрешена")
	}
}
