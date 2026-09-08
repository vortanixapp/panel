package handlers

import (
	"testing"

	"github.com/vortanix/vortanix/pkg/gamesettings"
)

func TestResolveSettingsTemplate(t *testing.T) {
	values := map[string]string{"servername": "myserv", "empty": ""}
	cases := map[string]string{
		"server.properties":                 "server.properties",
		"Zomboid/Server/{servername}.ini":   "Zomboid/Server/myserv.ini",
		"a/{servername}/b/{servername}.ini": "a/myserv/b/myserv.ini",
		// Незаполненную подстановку убираем целиком: путь с фигурными скобками
		// указывал бы на несуществующий каталог — ровно та ошибка, из-за которой
		// настройки Project Zomboid не работали никогда.
		"Zomboid/Server/{нетТакого}.ini": "Zomboid/Server/.ini",
		"{empty}config.cfg":              "config.cfg",
	}
	for in, want := range cases {
		if got := resolveSettingsTemplate(in, values); got != want {
			t.Errorf("resolveSettingsTemplate(%q) = %q, ожидалось %q", in, got, want)
		}
	}
}

// Путь собирается из профиля, но в него подставляется значение поля, которое
// задаёт клиент. Значит подстановку надо проверять так же, как если бы весь путь
// пришёл снаружи.
func TestSafeSettingsPathRejectsEscapes(t *testing.T) {
	bad := []string{
		"", "/etc/passwd", "../etc/passwd", "a/../../etc",
		"Zomboid/Server/../../../etc/passwd", "..", ".",
	}
	for _, p := range bad {
		if safeSettingsPath(p) {
			t.Errorf("путь %q должен быть отклонён", p)
		}
	}
	good := []string{
		"server.properties", "cstrike/server.cfg",
		"Zomboid/Server/servertest.ini",
		"Pal/Saved/Config/LinuxServer/PalWorldSettings.ini",
	}
	for _, p := range good {
		if !safeSettingsPath(p) {
			t.Errorf("путь %q должен быть разрешён", p)
		}
	}
}

// Подстановка не должна открывать дорогу наружу: значение поля -servername
// клиент задаёт сам.
func TestSafeSettingsPathAfterHostileSubstitution(t *testing.T) {
	hostile := resolveSettingsTemplate(
		"Zomboid/Server/{servername}.ini",
		map[string]string{"servername": "../../../../etc/passwd"},
	)
	if safeSettingsPath(hostile) {
		t.Fatalf("подстановка вывела за каталог сервера: %q", hostile)
	}
}

func TestContentDigestDetectsChange(t *testing.T) {
	a := contentDigest("hostname Сервер\n")
	if a == "" || a == contentDigest("hostname Другой\n") {
		t.Error("отпечаток не различает разное содержимое")
	}
	if a != contentDigest("hostname Сервер\n") {
		t.Error("отпечаток неустойчив")
	}
}

// Каждое поле каждого профиля обязано ссылаться на существующий файл: иначе
// сохранение молча уходило бы в пустоту.
func TestEveryProfileFieldResolvesToFile(t *testing.T) {
	for _, p := range gamesettings.All() {
		for _, f := range p.Fields {
			if _, ok := p.FileByID(f.File); !ok {
				t.Errorf("%s: поле %s ссылается на файл %q, которого нет", p.Key, f.Key, f.File)
			}
		}
	}
}

// Путь каждого файла профиля должен пережить проверку — иначе игра просто не
// сможет ничего сохранить.
func TestEveryProfileFilePathIsSafe(t *testing.T) {
	for _, p := range gamesettings.All() {
		for _, file := range p.Files {
			if file.IsStartup() {
				continue
			}
			resolved := resolveSettingsTemplate(file.Path, map[string]string{"servername": "servertest"})
			if !safeSettingsPath(resolved) {
				t.Errorf("%s: путь файла %s (%q → %q) не проходит проверку",
					p.Key, file.ID, file.Path, resolved)
			}
		}
	}
}
