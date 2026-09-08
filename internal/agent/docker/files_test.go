package docker

import "testing"

func TestResolveServerPathJailsToData(t *testing.T) {
	cases := map[string]string{
		"":                 "/data",
		"/":                "/data",
		"logs":             "/data/logs",
		"/logs/latest.log": "/data/logs/latest.log",
		"/..":              "/data",
		"/../..":           "/data",
		"/../etc/passwd":   "/data/etc/passwd",
		"/logs/../../etc":  "/data/etc",
		"//etc//passwd":    "/data/etc/passwd",
		`\..\etc`:          "/data/etc",
	}
	for in, want := range cases {
		got, err := resolveServerPath(in)
		if err != nil {
			t.Fatalf("resolveServerPath(%q): %v", in, err)
		}
		if got != want {
			t.Errorf("resolveServerPath(%q) = %q, ожидалось %q", in, got, want)
		}
	}
}

func TestIsServerRoot(t *testing.T) {
	if !isServerRoot("/data") {
		t.Fatal("/data должен считаться корнем сервера")
	}
	if isServerRoot("/data/logs") {
		t.Fatal("подкаталог не корень")
	}
}

func TestGuardedScriptChecksRealpath(t *testing.T) {
	script := guardedScript("/data/x", `cat "$R"`)
	for _, want := range []string{"realpath -m", "/data|/data/*", "outside server data", `cat "$R"`} {
		if !contains(script, want) {
			t.Fatalf("в скрипте нет %q:\n%s", want, script)
		}
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (func() bool {
		for i := 0; i+len(needle) <= len(haystack); i++ {
			if haystack[i:i+len(needle)] == needle {
				return true
			}
		}
		return false
	})()
}
