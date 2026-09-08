package agent

import (
	"os"
	"strings"
	"testing"
)

func readSource(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("не прочитать %s: %v", path, err)
	}
	return string(raw)
}

// Команда обновления игры уходила агенту, а ветки для неё не было: команда
// падала мимо всех case, агент отвечал «готово» и слал статус по умолчанию —
// «остановлен». Панель показывала «обновляется», на ноде не происходило
// ничего, а работавший сервер оказывался в базе остановленным.
func TestAgentHandlesUpdateAction(t *testing.T) {
	src := readSource(t, "agent.go")

	if !strings.Contains(src, `case "update":`) {
		t.Fatal("вернулась потеря команды обновления: ветки update нет")
	}
	if !strings.Contains(src, "updateAndStart") {
		t.Error("обновление не запускает работу с файлами игры")
	}
	if !strings.Contains(src, "docker.Update(") {
		t.Error("обновление не доходит до файлов сервера")
	}
}

// Сервер, работавший до обновления, должен вернуться в работу: обновление идёт
// на остановленном контейнере, и без запуска обратно клиент получил бы
// молча остановленный сервер.
func TestUpdateRestoresRunningServer(t *testing.T) {
	src := readSource(t, "agent.go")
	i := strings.Index(src, "func (a *Agent) updateAndStart(")
	if i < 0 {
		t.Fatal("функция updateAndStart не найдена")
	}
	body := src[i:]
	if j := strings.Index(body[1:], "\nfunc "); j >= 0 {
		body = body[:j+1]
	}

	if !strings.Contains(body, "wasRunning") {
		t.Error("прежнее состояние сервера не запоминается")
	}
	if !strings.Contains(body, "docker.Stop(") {
		t.Error("обновление идёт на работающем контейнере, файлы заняты процессом")
	}
	if !strings.Contains(body, "docker.Start(") {
		t.Error("сервер не возвращается в работу после обновления")
	}
	if !strings.Contains(body, "ReapplyOwnership") {
		t.Error("после SteamCMD файлы остаются за root, клиент теряет доступ по SFTP")
	}
}

// Пустой источник файлов — честная ошибка, а не тихое «готово».
func TestUpdateRejectsMissingSource(t *testing.T) {
	src := readSource(t, "agent.go")
	i := strings.Index(src, `case "update":`)
	if i < 0 {
		t.Fatal("ветки update нет")
	}
	body := src[i:]
	if j := strings.Index(body, "\tcase "); j > 0 {
		body = body[:j]
	}
	if !strings.Contains(body, "HasSource()") {
		t.Error("отсутствие источника файлов не проверяется")
	}
	if !strings.Contains(body, "sendAck(cmd.ID, false") {
		t.Error("при отсутствии источника агент по-прежнему отвечает успехом")
	}
}
