package docker

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func hostFilesFixture(t *testing.T) (base, serverID string) {
	t.Helper()
	base = t.TempDir()
	t.Setenv("VORTANIX_DATA_DIR", base)
	serverID = "srv"
	if err := os.MkdirAll(filepath.Join(base, serverID), 0o755); err != nil {
		t.Fatalf("подготовка каталога: %v", err)
	}
	return base, serverID
}

func TestHostServerPathStaysUnderServerDir(t *testing.T) {
	base, serverID := hostFilesFixture(t)
	root := filepath.Join(base, serverID)

	cases := map[string]string{
		"/data/server.properties":     filepath.Join(root, "server.properties"),
		"/data/cstrike/server.cfg":    filepath.Join(root, "cstrike", "server.cfg"),
		"/data/Zomboid/Server/a.ini":  filepath.Join(root, "Zomboid", "Server", "a.ini"),
		"/data/.vtx/startup_params":   filepath.Join(root, ".vtx", "startup_params"),
		"/data/имя с пробелом/к.conf": filepath.Join(root, "имя с пробелом", "к.conf"),
	}
	for in, want := range cases {
		got, err := hostServerPath(serverID, in)
		if err != nil {
			t.Fatalf("hostServerPath(%q): %v", in, err)
		}
		if got != want {
			t.Errorf("hostServerPath(%q) = %q, ожидалось %q", in, got, want)
		}
	}
}

// Джейл в контейнере обеспечивал `realpath -m` внутри guardedScript. На хосте
// его нет, а содержимое каталога принадлежит клиенту: по SFTP он может положить
// туда ссылку наружу. Проверяем, что подмена каталога ссылкой не даёт выйти.
func TestHostServerPathRejectsSymlinkEscape(t *testing.T) {
	base, serverID := hostFilesFixture(t)
	root := filepath.Join(base, serverID)

	outside := filepath.Join(base, "снаружи")
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatalf("подготовка: %v", err)
	}
	link := filepath.Join(root, "escape")
	if err := os.Symlink(outside, link); err != nil {
		if runtime.GOOS == "windows" {
			t.Skip("создание символических ссылок недоступно без соответствующих прав")
		}
		t.Fatalf("символическая ссылка: %v", err)
	}

	if _, err := hostServerPath(serverID, "/data/escape/секрет.txt"); err == nil {
		t.Fatal("путь через ссылку наружу должен быть отклонён")
	}
	// Обычный подкаталог рядом при этом остаётся доступным.
	if _, err := hostServerPath(serverID, "/data/обычный/файл.cfg"); err != nil {
		t.Fatalf("обычный путь не должен отклоняться: %v", err)
	}
}

func TestWriteAndReadFileRoundTrip(t *testing.T) {
	_, serverID := hostFilesFixture(t)
	ctx := context.Background()
	const body = "hostname \"Мой сервер\"\n// комментарий\nmp_timelimit 30\n"

	// Вложенного каталога ещё нет — запись должна его создать.
	if err := WriteFile(ctx, serverID, "cstrike/server.cfg", body); err != nil {
		t.Fatalf("запись: %v", err)
	}
	got, err := ReadFile(ctx, serverID, "cstrike/server.cfg")
	if err != nil {
		t.Fatalf("чтение: %v", err)
	}
	if got != body {
		t.Errorf("прочитано %q, ожидалось %q", got, body)
	}
}

func TestWriteFileKeepsModeOfExistingFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("права доступа на Windows не совпадают с unix-моделью")
	}
	base, serverID := hostFilesFixture(t)
	ctx := context.Background()
	path := filepath.Join(base, serverID, "server.cfg")

	if err := os.WriteFile(path, []byte("старое\n"), 0o600); err != nil {
		t.Fatalf("подготовка: %v", err)
	}
	if err := WriteFile(ctx, serverID, "server.cfg", "новое\n"); err != nil {
		t.Fatalf("запись: %v", err)
	}
	st, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	// Иначе перезапись настроек снимала бы с файла выданные ему права.
	if st.Mode().Perm() != 0o600 {
		t.Errorf("права стали %v, ожидалось 0600", st.Mode().Perm())
	}
}

func TestWriteFileLeavesNoTempFilesBehind(t *testing.T) {
	base, serverID := hostFilesFixture(t)
	ctx := context.Background()

	if err := WriteFile(ctx, serverID, "server.cfg", "тело\n"); err != nil {
		t.Fatalf("запись: %v", err)
	}
	entries, err := os.ReadDir(filepath.Join(base, serverID))
	if err != nil {
		t.Fatalf("чтение каталога: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "server.cfg" {
		names := []string{}
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("в каталоге %v, ожидался только server.cfg", names)
	}
}

func TestWriteToServerRootIsRefused(t *testing.T) {
	_, serverID := hostFilesFixture(t)
	if err := WriteFile(context.Background(), serverID, "/", "что угодно"); err == nil {
		t.Fatal("запись в корень данных сервера должна быть отклонена")
	}
}

func TestListFilesOnHostSortsDirsFirst(t *testing.T) {
	base, serverID := hostFilesFixture(t)
	root := filepath.Join(base, serverID)
	if err := os.MkdirAll(filepath.Join(root, "cstrike"), 0o755); err != nil {
		t.Fatalf("подготовка: %v", err)
	}
	for _, name := range []string{"server.cfg", "имя с пробелом.txt", "a.log"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("x"), 0o644); err != nil {
			t.Fatalf("подготовка %s: %v", name, err)
		}
	}

	files, err := ListFiles(context.Background(), serverID, "/")
	if err != nil {
		t.Fatalf("список: %v", err)
	}
	if len(files) != 4 {
		t.Fatalf("получено %d записей, ожидалось 4", len(files))
	}
	if !files[0].IsDir || files[0].Name != "cstrike" {
		t.Errorf("первым должен идти каталог cstrike, получено %+v", files[0])
	}
	// Разбор вывода `ls -la` ломался на именах с пробелами — прямое чтение нет.
	found := false
	for _, f := range files {
		if f.Name == "имя с пробелом.txt" {
			found = true
		}
	}
	if !found {
		t.Error("файл с пробелом в имени потерялся")
	}
}
