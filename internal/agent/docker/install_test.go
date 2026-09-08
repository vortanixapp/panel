package docker

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func zipWith(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("zip create %s: %v", name, err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatalf("zip write %s: %v", name, err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("zip close: %v", err)
	}
	return buf.Bytes()
}

func serveZip(t *testing.T, data []byte) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(data)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestInstallArchiveUnpacksIntoDataDir(t *testing.T) {
	srv := serveZip(t, zipWith(t, map[string]string{
		"server.jar":               "JAR",
		"config/server.properties": "server-port=25565",
	}))
	dataDir := t.TempDir()

	if err := installArchive(context.Background(), dataDir, srv.URL+"/pack.zip", nil); err != nil {
		t.Fatalf("installArchive: %v", err)
	}

	jar, err := os.ReadFile(filepath.Join(dataDir, "server.jar"))
	if err != nil || string(jar) != "JAR" {
		t.Fatalf("server.jar не распакован: %v %q", err, jar)
	}
	if _, err := os.Stat(filepath.Join(dataDir, "config", "server.properties")); err != nil {
		t.Fatalf("вложенный файл не распакован: %v", err)
	}
}

func TestInstallArchiveFlattensSingleRootDir(t *testing.T) {
	srv := serveZip(t, zipWith(t, map[string]string{
		"minecraft-server-1.20/server.jar": "JAR",
		"minecraft-server-1.20/eula.txt":   "eula=true",
	}))
	dataDir := t.TempDir()

	if err := installArchive(context.Background(), dataDir, srv.URL+"/pack.zip", nil); err != nil {
		t.Fatalf("installArchive: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dataDir, "server.jar")); err != nil {
		t.Fatalf("корневая папка не развёрнута: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dataDir, "minecraft-server-1.20")); err == nil {
		t.Fatal("корневая папка осталась в томе")
	}
}

func TestInstallArchiveRejectsBadURLs(t *testing.T) {
	dataDir := t.TempDir()
	for _, url := range []string{
		"file:///etc/passwd",
		"https://example.com/server.exe",
		"ftp://example.com/pack.zip",
	} {
		if err := installArchive(context.Background(), dataDir, url, nil); err == nil {
			t.Fatalf("ожидалась ошибка для %q", url)
		}
	}
}

func TestExtractZipRejectsPathTraversal(t *testing.T) {
	data := zipWith(t, map[string]string{"../../evil.txt": "pwned"})
	tmp := t.TempDir()
	archive := filepath.Join(tmp, "a.zip")
	if err := os.WriteFile(archive, data, 0o644); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(tmp, "extract")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := extractZip(archive, dest); err == nil {
		t.Fatal("path traversal должен отклоняться")
	}
}

func TestNeedsInstall(t *testing.T) {
	base := t.TempDir()
	t.Setenv("VORTANIX_DATA_DIR", base)

	if !NeedsInstall("srv1") {
		t.Fatal("отсутствующий том должен требовать установки")
	}

	dir := filepath.Join(base, "srv1")
	if err := os.MkdirAll(filepath.Join(dir, ".vtx"), 0o755); err != nil {
		t.Fatal(err)
	}
	if !NeedsInstall("srv1") {
		t.Fatal("том только со служебным .vtx должен требовать установки")
	}

	if err := os.WriteFile(filepath.Join(dir, "server.jar"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if NeedsInstall("srv1") {
		t.Fatal("установленный сервер не должен переустанавливаться")
	}
}

func TestInstallSpecFromPayload(t *testing.T) {
	spec := InstallSpecFromPayload(map[string]any{
		"source_type":  "steam",
		"steam_app_id": float64(730),
		"steam_branch": "public",
	})
	if spec.SteamAppID != 730 || spec.SteamBranch != "public" || !spec.HasSource() {
		t.Fatalf("steam spec разобран неверно: %+v", spec)
	}

	empty := InstallSpecFromPayload(nil)
	if empty.HasSource() {
		t.Fatal("пустой payload не должен давать источник")
	}
}

func TestSteamcmdArgsModConfigPrecedesAppUpdate(t *testing.T) {
	args := steamcmdArgs("/srv/data", InstallSpec{SteamAppID: 90, SteamModConfig: "cstrike"})

	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "+app_set_config 90 mod cstrike") {
		t.Fatalf("нет app_set_config: %s", joined)
	}
	setIdx := indexOf(args, "+app_set_config")
	updIdx := indexOf(args, "+app_update")
	if setIdx < 0 || updIdx < 0 || setIdx > updIdx {
		t.Fatalf("app_set_config должен идти до app_update: %s", joined)
	}
	if args[len(args)-1] != "+quit" {
		t.Fatalf("последним аргументом должен быть +quit: %s", joined)
	}
}

func TestSteamcmdArgsWithoutModConfig(t *testing.T) {
	args := steamcmdArgs("/srv/data", InstallSpec{SteamAppID: 740})
	if indexOf(args, "+app_set_config") >= 0 {
		t.Fatalf("app_set_config не нужен без мода: %v", args)
	}
	if !strings.Contains(strings.Join(args, " "), "+app_update 740 validate") {
		t.Fatalf("нет app_update: %v", args)
	}
}

func TestSteamcmdArgsBranch(t *testing.T) {
	args := steamcmdArgs("/srv/data", InstallSpec{SteamAppID: 412680, SteamBranch: "evrima"})
	if !strings.Contains(strings.Join(args, " "), "+app_update 412680 -beta evrima validate") {
		t.Fatalf("ветка не прокинута: %v", args)
	}
}

func indexOf(list []string, want string) int {
	for i, v := range list {
		if v == want {
			return i
		}
	}
	return -1
}

func TestExplainPullError(t *testing.T) {
	rate := errors.New(`Error response from daemon: unexpected status from HEAD request: 429 Too Many Requests`)
	got := explainPullError(rate).Error()
	if !strings.Contains(got, "лимит") || !strings.Contains(got, "docker login") {
		t.Errorf("лимит Docker Hub не объяснён: %s", got)
	}

	denied := errors.New(`Error response from daemon: pull access denied for vortanix/cs16, repository does not exist`)
	got = explainPullError(denied).Error()
	if !strings.Contains(got, "соберите его на ноде") {
		t.Errorf("отсутствующий образ не объяснён: %s", got)
	}

	other := errors.New("no space left on device")
	if explainPullError(other).Error() != other.Error() {
		t.Error("неизвестная ошибка не должна подменяться")
	}
	if explainPullError(nil) != nil {
		t.Error("nil должен оставаться nil")
	}
}

func TestInstallArchiveReportsProgress(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/java-archive")
		_, _ = w.Write([]byte("JAR"))
	}))
	defer srv.Close()

	var stages []string
	var sawDone bool
	report := func(stage string, percent int, message, line string) {
		stages = append(stages, stage)
		if stage == StageDone && percent == 100 {
			sawDone = true
		}
	}

	if err := installArchive(context.Background(), t.TempDir(), srv.URL+"/server.jar", report); err != nil {
		t.Fatalf("installArchive: %v", err)
	}

	if len(stages) == 0 {
		t.Fatal("установка прошла молча — панель показала бы выдуманный прогресс")
	}
	if stages[0] != StageDownload {
		t.Errorf("первый этап %q, ожидался %q", stages[0], StageDownload)
	}
	if !sawDone {
		t.Error("нет завершающего отчёта: полоса прогресса застынет на середине")
	}
}
