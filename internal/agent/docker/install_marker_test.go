package docker

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Оборванная установка выглядела как успешная: агент падал посреди загрузки,
// в томе оставались файлы, и NeedsInstall решал, что ставить больше нечего.
// Сервер стартовал на битой сборке, а лечилось это только переустановкой
// с потерей данных.
func TestInstallMarkerForcesReinstall(t *testing.T) {
	src, err := os.ReadFile(filepath.Join(".", "install.go"))
	if err != nil {
		t.Fatalf("не прочитать install.go: %v", err)
	}
	text := string(src)

	needs := funcText(t, text, "func NeedsInstall(")
	if !strings.Contains(needs, "installMarkerPath") {
		t.Error("незавершённая установка не отличается от завершённой")
	}

	install := funcText(t, text, "func Install(")
	if !strings.Contains(install, "WipeData") {
		t.Error("повтор после обрыва не начинается с чистого тома: файлы смешаются")
	}
	if !strings.Contains(install, "os.WriteFile(marker") {
		t.Error("метка не ставится перед началом установки")
	}
	if !strings.Contains(install, "os.Remove(marker)") {
		t.Error("метка не снимается после успеха — установка будет повторяться вечно")
	}
	if !strings.Contains(install, "Метку оставляем") {
		t.Error("при ошибке метку нужно оставить, иначе следующая попытка примет том за целый")
	}
}

// funcText возвращает текст функции от её объявления до следующего
// объявления верхнего уровня.
func funcText(t *testing.T, src, decl string) string {
	t.Helper()
	i := strings.Index(src, decl)
	if i < 0 {
		t.Fatalf("не найдено объявление %q", decl)
	}
	rest := src[i+len(decl):]
	if j := strings.Index(rest, "\nfunc "); j >= 0 {
		return rest[:j]
	}
	return rest
}
