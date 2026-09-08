package docker

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFirstLineMatchesEntrypointReading(t *testing.T) {
	// Ровно то, что делает entrypoint: head -n 1 плюс обрезка пробелов.
	cases := map[string]string{
		"":                        "",
		"   ":                     "",
		"-Xmx2G":                  "-Xmx2G",
		"  -Xmx2G  ":              "-Xmx2G",
		"-Xmx2G\n-Xms1G":          "-Xmx2G",
		"-Xmx2G\r\n-Xms1G":        "-Xmx2G",
		"-Xmx2G\r-Xms1G":          "-Xmx2G",
		"\n-Xms1G":                "",
		"?SessionName=Мой сервер": "?SessionName=Мой сервер",
	}
	for in, want := range cases {
		if got := firstLine(in); got != want {
			t.Errorf("firstLine(%q) = %q, ожидалось %q", in, got, want)
		}
	}
}

func TestWriteStartupParamsRoundTrip(t *testing.T) {
	base := t.TempDir()
	t.Setenv("VORTANIX_DATA_DIR", base)
	const serverID = "srv-1"
	path := filepath.Join(base, serverID, startupParamsRel)

	if err := WriteStartupParams(serverID, "-name Мир -world Дом"); err != nil {
		t.Fatalf("запись: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("чтение: %v", err)
	}
	// Перевод строки обязателен: entrypoint читает через head -n 1.
	if string(got) != "-name Мир -world Дом\n" {
		t.Errorf("в файле %q", string(got))
	}
}

// Однострочный файл подставлялся в entrypoint без кавычек, и оболочка делила
// его на слова, не снимая кавычек: -name "Мой мир" приходило игре тремя
// аргументами. Значение с пробелом не доезжало никогда, поэтому рядом пишется
// файл с одним аргументом на строку.
func TestWriteStartupParamsSplitsArgvForShell(t *testing.T) {
	base := t.TempDir()
	t.Setenv("VORTANIX_DATA_DIR", base)
	const serverID = "srv-argv"

	if err := WriteStartupParams(serverID, `-name "Мой мир" -world Дом -password 'па роль'`); err != nil {
		t.Fatalf("запись: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(base, serverID, startupArgvRel))
	if err != nil {
		t.Fatalf("чтение argv: %v", err)
	}
	// Кавычки снимаются здесь: дальше строку читает `read -r` и отдаёт как есть.
	want := "-name\nМой мир\n-world\nДом\n-password\nпа роль\n"
	if string(got) != want {
		t.Errorf("argv:\n%q\nожидалось:\n%q", string(got), want)
	}

	// Однострочный файл остаётся ради уже собранных образов.
	line, err := os.ReadFile(filepath.Join(base, serverID, startupParamsRel))
	if err != nil {
		t.Fatalf("чтение params: %v", err)
	}
	if string(line) != `-name "Мой мир" -world Дом -password 'па роль'`+"\n" {
		t.Errorf("однострочный файл изменился: %q", string(line))
	}
}

func TestWriteStartupParamsRemovesBothFilesWhenEmpty(t *testing.T) {
	base := t.TempDir()
	t.Setenv("VORTANIX_DATA_DIR", base)
	const serverID = "srv-clear"

	if err := WriteStartupParams(serverID, "-name Мир"); err != nil {
		t.Fatalf("подготовка: %v", err)
	}
	if err := WriteStartupParams(serverID, ""); err != nil {
		t.Fatalf("очистка: %v", err)
	}
	for _, rel := range []string{startupParamsRel, startupArgvRel} {
		if _, err := os.Stat(filepath.Join(base, serverID, rel)); !os.IsNotExist(err) {
			t.Errorf("%s должен быть удалён, получено %v", rel, err)
		}
	}
}

func TestWriteStartupParamsRemovesFileWhenEmpty(t *testing.T) {
	base := t.TempDir()
	t.Setenv("VORTANIX_DATA_DIR", base)
	const serverID = "srv-2"
	path := filepath.Join(base, serverID, startupParamsRel)

	if err := WriteStartupParams(serverID, "-old"); err != nil {
		t.Fatalf("подготовка: %v", err)
	}
	// Очистка поля в панели должна убирать параметры, а не оставлять прежние.
	if err := WriteStartupParams(serverID, "   "); err != nil {
		t.Fatalf("очистка: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("файл должен быть удалён, получено %v", err)
	}
}

func TestWriteStartupParamsEmptyOnFreshServerIsNotAnError(t *testing.T) {
	base := t.TempDir()
	t.Setenv("VORTANIX_DATA_DIR", base)
	// Каталога сервера ещё нет: команда power приходит раньше установки.
	if err := WriteStartupParams("srv-3", ""); err != nil {
		t.Fatalf("пустые параметры на новом сервере не должны падать: %v", err)
	}
}
