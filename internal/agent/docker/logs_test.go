package docker

import (
	"strings"
	"testing"
)

// Прогресс-бары перезаписывают строку возвратом каретки. В <pre> он трактуется
// как разрыв, и журнал разъезжался пустыми строками.
func TestCarriageReturnKeepsLastState(t *testing.T) {
	lines := splitLogLines("Downloading 10%\rDownloading 60%\rDownloading 100%\nDone")

	if len(lines) != 2 {
		t.Fatalf("ожидалось 2 строки, получено %d: %q", len(lines), lines)
	}
	if lines[0] != "Downloading 100%" {
		t.Errorf("осталось не последнее состояние строки: %q", lines[0])
	}
}

// Цветовые коды игры и управляющие последовательности терминала печатались в
// панели буквально: снимаем их тем же способом, что и в опросе сервера.
func TestColorCodesStripped(t *testing.T) {
	lines := splitLogLines("\x1b[32mINFO\x1b[0m server ready\n§aПривет§r мир")

	if strings.Contains(lines[0], "\x1b") {
		t.Errorf("остались ANSI-последовательности: %q", lines[0])
	}
	if lines[0] != "INFO server ready" {
		t.Errorf("строка испорчена при очистке: %q", lines[0])
	}
	if strings.ContainsRune(lines[1], '§') {
		t.Errorf("остались цветовые коды игры: %q", lines[1])
	}
	if lines[1] != "Привет мир" {
		t.Errorf("кириллица не пережила очистку: %q", lines[1])
	}
}

// Сборки игр пишут не в UTF-8. Такие байты JSON заменяет на U+FFFD молча,
// и до панели доезжал мусор вместо текста.
func TestInvalidUTF8Dropped(t *testing.T) {
	lines := splitLogLines("ok \xff\xfe tail")

	if len(lines) != 1 {
		t.Fatalf("ожидалась одна строка, получено %d", len(lines))
	}
	if !strings.HasPrefix(lines[0], "ok ") || !strings.HasSuffix(lines[0], "tail") {
		t.Errorf("годный текст потерялся вместе с битыми байтами: %q", lines[0])
	}
}

// Пустой вывод — это отсутствие строк, а не строка из пустоты.
func TestEmptyOutputGivesNoLines(t *testing.T) {
	if got := splitLogLines("\n\n"); len(got) != 0 {
		t.Errorf("пустой вывод дал строки: %q", got)
	}
}
