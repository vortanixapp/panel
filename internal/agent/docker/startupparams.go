package docker

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/vortanix/vortanix/pkg/gamesettings"
)

// Параметры запуска для entrypoint контейнера.
//
// Все образы игр читают одну строку из /data/.vtx/startup_params и дописывают её
// к команде запуска (см. deploy/images/_runtime/*/entrypoint.sh). Читающая сторона
// пережила переезд с legacy-демона на Go, а пишущая — нет: файл не создавал никто,
// и поле «Параметры запуска» в панели много месяцев не влияло ни на что. Заодно
// молчал профиль Valheim, у которого настройки живут только в этой строке.
//
// Пишем на стороне хоста, а не через docker exec: параметры нужны до старта
// контейнера, когда исполнять в нём ещё нечего.
const (
	startupParamsRel = ".vtx/startup_params"
	startupArgvRel   = ".vtx/startup_argv"
)

// WriteStartupParams кладёт параметры запуска в каталог сервера — в двух видах.
//
// startup_params — одна строка, как было раньше. Её читают уже собранные образы,
// и убирать её нельзя, пока они не обновлены.
//
// startup_argv — по одному аргументу на строку. Это исправление: однострочный
// файл подставлялся в entrypoint как $STARTUP_LINE без кавычек, а оболочка в
// таком случае делит строку на слова, но кавычки не снимает. Аргумент
// -name "Мой мир" приходил игре тремя кусками: -name, «"Мой» и «мир"». То есть
// любое значение с пробелом — а это в первую очередь имя сервера — доехать не
// могло в принципе, сколько его ни экранируй. Разбор кавычек должен произойти
// на нашей стороне, а до игры аргументы обязаны дойти уже разделёнными.
func WriteStartupParams(serverID, params string) error {
	dir := serverDataDir(serverID)
	paramsPath := filepath.Join(dir, startupParamsRel)
	argvPath := filepath.Join(dir, startupArgvRel)

	line := firstLine(params)
	if line == "" {
		return removeAll(paramsPath, argvPath)
	}
	if err := os.MkdirAll(filepath.Join(dir, ".vtx"), 0o755); err != nil {
		return err
	}

	argv := gamesettings.SplitArgs(line)
	if len(argv) == 0 {
		return removeAll(paramsPath, argvPath)
	}
	var buf strings.Builder
	for _, tok := range argv {
		// Кавычки снимаем здесь: дальше строку читает `read -r`, который отдаёт
		// её игре как есть, без разбора кавычек.
		buf.WriteString(gamesettings.Unquote(tok))
		buf.WriteByte('\n')
	}
	if err := os.WriteFile(argvPath, []byte(buf.String()), 0o644); err != nil {
		return err
	}
	return os.WriteFile(paramsPath, []byte(line+"\n"), 0o644)
}

// removeAll убирает файлы параметров.
//
// Пустое значение означает «параметров нет», и тогда файлы именно удаляются, а
// не переписываются пустыми: entrypoint проверяет существование файла раньше
// содержимого, и очистка поля в панели должна снимать прежние аргументы, а не
// оставлять их в силе.
func removeAll(paths ...string) error {
	for _, p := range paths {
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

// firstLine сводит значение к тому, что entrypoint всё равно прочитает: он берёт
// `head -n 1` и обрезает пробелы по краям. Хранить больше одной строки незачем —
// вторая никогда не будет исполнена, но при чтении конфига создаст впечатление,
// будто она учитывается.
func firstLine(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}
