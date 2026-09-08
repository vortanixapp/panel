package gamesettings

import (
	"fmt"
	"sort"
	"strings"
)

func sortStrings(s []string) { sort.Strings(s) }

// Codec — чтение и правка одного формата конфигурации.
//
// Кодек говорит на физических ключах файла, а не на ключах API. Раньше эти два
// словаря путались: разбор server.properties возвращал `max-players`, а запись
// ждала `max_players`, и панель докручивала соответствие у себя третьим списком
// (normalizeLiveSettings). Из-за этого значение, прочитанное из файла, нельзя
// было записать обратно тем же именем.
//
// Теперь перевод один и живёт в Field: Key — имя в API, Prop — имя в файле.
// Кодек про Key не знает вовсе, поэтому разойтись им негде.
type Codec interface {
	// Parse достаёт значения по физическим ключам. Незнакомые ключи тоже
	// возвращаются: панель показывает их в сыром редакторе, а профиль решает,
	// какие из них показать формой.
	Parse(content string) map[string]string

	// Apply правит значения на месте.
	//
	// Всё, чего нет в changes, обязано остаться нетронутым — включая
	// комментарии, порядок строк и ключи, которых профиль не знает. Клиент мог
	// править файл руками, и сохранение настроек не должно это вычищать.
	//
	// Ошибка означает, что содержимое не удалось разобрать. Раньше такие случаи
	// возвращали просто false, вызывающий его игнорировал, и панель отвечала
	// «Настройки сохранены», ничего не сохранив.
	Apply(content string, changes map[string]string) (string, error)
}

// codecs — реализации по форматам. Заполняется файлами codec_*.go в init.
var codecs = map[Format]Codec{}

func registerCodec(f Format, c Codec) {
	if _, busy := codecs[f]; busy {
		panic("gamesettings: кодек формата " + string(f) + " уже зарегистрирован")
	}
	codecs[f] = c
}

// CodecFor возвращает кодек формата.
func CodecFor(f Format) (Codec, bool) {
	c, ok := codecs[f]
	return c, ok
}

// ErrUnsupportedFormat — для формата нет кодека.
type ErrUnsupportedFormat struct{ Format Format }

func (e *ErrUnsupportedFormat) Error() string {
	return "неизвестный формат конфигурации: " + string(e.Format)
}

// ParseFile разбирает содержимое файла его кодеком.
func ParseFile(f ConfigFile, content string) (map[string]string, error) {
	c, ok := CodecFor(f.Format)
	if !ok {
		return nil, &ErrUnsupportedFormat{Format: f.Format}
	}
	return c.Parse(content), nil
}

// ApplyFile правит содержимое файла его кодеком.
func ApplyFile(f ConfigFile, content string, changes map[string]string) (string, error) {
	c, ok := CodecFor(f.Format)
	if !ok {
		return "", &ErrUnsupportedFormat{Format: f.Format}
	}
	if len(changes) == 0 {
		return content, nil
	}
	// Пустой файл — обычное дело: большинство образов конфиг не создают, он
	// появляется только после первого запуска игры. Тогда пишем поверх
	// заготовки, чтобы не выдумывать структуру формата с нуля.
	if strings.TrimSpace(content) == "" && f.Template != "" {
		content = f.Template
	}
	out, err := c.Apply(content, changes)
	if err != nil {
		return "", fmt.Errorf("%s: %w", f.Path, err)
	}
	return out, nil
}

// splitLines режет содержимое на строки, не теряя завершающую пустую.
//
// CRLF приводится к LF при разборе, но при записи строки собираются обратно тем
// же переводом, каким пришли: конфиг, отредактированный в Windows, не должен
// молча менять весь файл при правке одного поля.
func splitLines(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.Split(s, "\n")
}

// lineEnding определяет, каким переводом строки пользуется файл.
func lineEnding(s string) string {
	if strings.Contains(s, "\r\n") {
		return "\r\n"
	}
	return "\n"
}

// joinLines собирает строки обратно исходным переводом.
func joinLines(lines []string, ending string) string {
	return strings.Join(lines, ending)
}

// isCommentLine сообщает, что строка — комментарий или пустая.
func isCommentLine(line string, markers ...string) bool {
	t := strings.TrimSpace(line)
	if t == "" {
		return true
	}
	for _, m := range markers {
		if strings.HasPrefix(t, m) {
			return true
		}
	}
	return false
}

// sortedKeys — устойчивый порядок обхода изменений.
//
// Нужен везде, где ключи дописываются в конец файла: обход карты в Go случаен,
// и без сортировки один и тот же набор изменений давал разный файл от запуска к
// запуску. Заметить это трудно — смысл-то не меняется, — а разница видна в
// каждом diff и в каждой резервной копии.
func sortedKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sortStrings(out)
	return out
}
