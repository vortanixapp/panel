package docker

import (
	"bytes"
	"context"
	"os/exec"
	"strconv"
	"strings"
	"unicode/utf8"
)

func TailLogs(ctx context.Context, serverID string, tail int) ([]string, error) {
	if tail <= 0 {
		tail = 100
	}
	if tail > 1000 {
		tail = 1000
	}
	cname := ContainerName(serverID)
	cmd := exec.CommandContext(
		ctx,
		"docker",
		"logs",
		"--tail",
		strconv.Itoa(tail),
		cname,
	)
	// Потоки разводим: раньше стоял CombinedOutput, и вывод самого docker
	// («No such container», предупреждения) попадал в журнал сервера наравне
	// с выводом игры, а при ошибке отдавался как её содержимое.
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil && stdout.Len() == 0 {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			return nil, err
		}
		return nil, errorWithOutput(err, msg)
	}

	// Игровой сервер пишет в оба потока: stderr обычно короче, поэтому
	// добавляем его после stdout, а не перемешиваем построчно — восстановить
	// исходный порядок всё равно нечем, но границу видно.
	lines := splitLogLines(stdout.String())
	if extra := splitLogLines(stderr.String()); len(extra) > 0 && err == nil {
		lines = append(lines, extra...)
	}
	return lines, nil
}

// splitLogLines приводит вывод к строкам, пригодным для показа: снимает
// управляющие последовательности и цветовые коды игры, разбивает по возврату
// каретки от прогресс-баров и чинит обрывки не в UTF-8 — иначе JSON заменял их
// на U+FFFD и строка приезжала в панель мусором.
func splitLogLines(raw string) []string {
	text := strings.TrimRight(raw, "\n\r")
	if text == "" {
		return nil
	}
	out := []string{}
	for _, line := range strings.Split(text, "\n") {
		// Прогресс-бары перезаписывают строку возвратом каретки: показываем
		// только последнее состояние, иначе вывод разъезжается пустыми строками.
		if i := strings.LastIndexByte(line, '\r'); i >= 0 {
			line = line[i+1:]
		}
		line = mcColorCodes.ReplaceAllString(stripANSI(line), "")
		line = strings.TrimRight(line, " \t")
		if !utf8.ValidString(line) {
			line = strings.ToValidUTF8(line, "")
		}
		out = append(out, line)
	}
	return out
}

type logCommandError struct {
	err    error
	output string
}

func (e logCommandError) Error() string { return e.err.Error() + ": " + e.output }
func (e logCommandError) Unwrap() error { return e.err }

func errorWithOutput(err error, output string) error {
	return logCommandError{err: err, output: output}
}
