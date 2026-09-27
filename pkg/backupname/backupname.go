// Package backupname проверяет имена файлов резервных копий.
//
// Имя копии приходит от клиента и попадает в путь трижды: агент кладёт архив в
// /data/backups внутри контейнера, панель хранит имя в core.server_backups, а
// worker подставляет его в команды SSH на ноде (cat и cat > для выгрузки в S3 и
// обратно). Поэтому имя должно оставаться простым именем файла: любое звено
// пути превращает выгрузку копии в чтение и запись произвольного файла на ноде
// от имени пользователя SSH.
package backupname

import (
	"errors"
	"strings"
)

// MaxLen — предел длины имени архива.
const MaxLen = 128

// ErrInvalid возвращается, когда имя нельзя использовать как имя файла.
var ErrInvalid = errors.New("имя копии должно быть именем файла без косых черт, " +
	"переводов строк и длиной до 128 символов")

func badByte(c byte) bool {
	return c == '/' || c == '\x5c' || c < 0x20 || c == 0x7f
}

// Valid сообщает, годится ли имя как имя файла архива внутри каталога копий.
// Буквы любого алфавита и пробелы разрешены: опасны только звенья пути и
// управляющие символы.
func Valid(name string) bool {
	if name == "" || len(name) > MaxLen {
		return false
	}
	if name == "." || name == ".." {
		return false
	}
	for i := 0; i < len(name); i++ {
		if badByte(name[i]) {
			return false
		}
	}
	return true
}

// Check возвращает ошибку, если имя не годится.
func Check(name string) error {
	if !Valid(name) {
		return ErrInvalid
	}
	return nil
}

// Sanitize приводит имя к безопасному: выбрасывает звенья пути и управляющие
// символы. Пустой результат означает, что от имени ничего не осталось и вызвать
// нужно поведение по умолчанию.
func Sanitize(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, "\x5c", "/")
	if i := strings.LastIndexByte(name, '/'); i >= 0 {
		name = name[i+1:]
	}
	var b strings.Builder
	for i := 0; i < len(name); i++ {
		if c := name[i]; !badByte(c) {
			b.WriteByte(c)
		}
	}
	out := strings.TrimSpace(b.String())
	if len(out) > MaxLen {
		out = out[:MaxLen]
	}
	if out == "." || out == ".." {
		return ""
	}
	return out
}
