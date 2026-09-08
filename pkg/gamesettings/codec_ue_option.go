package gamesettings

import (
	"errors"
	"strings"
)

func init() { registerCodec(FormatUEOption, ueOptionCodec{}) }

// ueOptionCodec — формат Palworld, самый необычный в проекте.
//
// Весь конфиг это одна строка внутри секции:
//
//	[/Script/Pal.PalGameWorldSettings]
//	OptionSettings=(Difficulty=None,ServerName="Мой сервер",ServerPlayerMaxNum=32,bIsPvP=False)
//
// Физический ключ — имя параметра внутри скобок.
//
// Прежняя реализация вырезала содержимое регулярным выражением
// `OptionSettings=\(([^)]+)\)`, которое обрывалось на ПЕРВОЙ закрывающей скобке.
// Имя сервера вида «Сервер (RU)» ломало и разбор, и запись: половина параметров
// пропадала. Скобки и кавычки требуют разбора состоянием, а не шаблоном.
type ueOptionCodec struct{}

const ueOptionPrefix = "OptionSettings=("

// findOptionSettings возвращает границы содержимого скобок.
func findOptionSettings(content string) (start, end int, ok bool) {
	idx := strings.Index(content, ueOptionPrefix)
	if idx < 0 {
		return 0, 0, false
	}
	start = idx + len(ueOptionPrefix)
	depth := 1
	inQuote := false
	for i := start; i < len(content); i++ {
		switch content[i] {
		case '"':
			inQuote = !inQuote
		case '(':
			if !inQuote {
				depth++
			}
		case ')':
			if inQuote {
				continue
			}
			depth--
			if depth == 0 {
				return start, i, true
			}
		}
	}
	return 0, 0, false
}

// splitOptionPairs режет содержимое по запятым верхнего уровня.
func splitOptionPairs(body string) []string {
	var (
		out   []string
		cur   strings.Builder
		depth int
		quote bool
	)
	for i := 0; i < len(body); i++ {
		ch := body[i]
		switch ch {
		case '"':
			quote = !quote
		case '(':
			if !quote {
				depth++
			}
		case ')':
			if !quote {
				depth--
			}
		case ',':
			if !quote && depth == 0 {
				out = append(out, cur.String())
				cur.Reset()
				continue
			}
		}
		cur.WriteByte(ch)
	}
	if strings.TrimSpace(cur.String()) != "" {
		out = append(out, cur.String())
	}
	return out
}

func splitOptionPair(pair string) (key, value string, ok bool) {
	i := strings.IndexByte(pair, '=')
	if i <= 0 {
		return "", "", false
	}
	key = strings.TrimSpace(pair[:i])
	value = strings.TrimSpace(pair[i+1:])
	if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
		value = value[1 : len(value)-1]
	}
	return key, value, key != ""
}

// quotedInSource — значение записано в кавычках. Кавычки в Palworld ставятся у
// строк и не ставятся у чисел и булевых, и менять этот выбор при правке нельзя:
// число в кавычках игра не примет.
func ueNeedsQuotes(original string, hadQuotes bool) bool {
	if hadQuotes {
		return true
	}
	return strings.ContainsAny(original, " ,()")
}

func (ueOptionCodec) Parse(content string) map[string]string {
	start, end, ok := findOptionSettings(content)
	if !ok {
		return map[string]string{}
	}
	out := map[string]string{}
	for _, pair := range splitOptionPairs(content[start:end]) {
		key, value, ok := splitOptionPair(pair)
		if !ok {
			continue
		}
		if _, seen := out[key]; seen {
			continue
		}
		out[key] = value
	}
	return out
}

func (ueOptionCodec) Apply(content string, changes map[string]string) (string, error) {
	start, end, ok := findOptionSettings(content)
	if !ok {
		// Файла ещё нет или он без нужной секции — собираем минимальный.
		//
		// Раньше в этом случае выписывались ВСЕ известные ключи со значениями по
		// умолчанию, то есть нулями: bUseAuth=0, bEnablePlayerToPlayerDamage=0,
		// ExpRate=0. Palworld ждёт там True/False, а нулевой множитель опыта
		// останавливает прогресс — сервер поднимался с настройками, которых никто
		// не задавал. Пишем только то, что действительно попросили изменить.
		if len(changes) == 0 {
			return content, nil
		}
		var b strings.Builder
		b.WriteString("[/Script/Pal.PalGameWorldSettings]\n")
		b.WriteString(ueOptionPrefix)
		for i, key := range sortedKeys(changes) {
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteString(key + "=" + ueFormat(changes[key], false))
		}
		b.WriteString(")\n")
		return b.String(), nil
	}

	pairs := splitOptionPairs(content[start:end])
	seen := map[string]bool{}
	for i, pair := range pairs {
		key, _, ok := splitOptionPair(pair)
		if !ok {
			continue
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		val, want := changes[key]
		if !want {
			continue
		}
		raw := strings.TrimSpace(strings.SplitN(pair, "=", 2)[1])
		hadQuotes := len(raw) >= 2 && raw[0] == '"' && raw[len(raw)-1] == '"'
		pairs[i] = key + "=" + ueFormat(val, hadQuotes)
	}

	// Незнакомые параметры остаются на своих местах: их мог выставить сам
	// клиент, и правка одного поля не должна вычищать остальные.
	for _, key := range sortedKeys(changes) {
		if seen[key] {
			continue
		}
		pairs = append(pairs, key+"="+ueFormat(changes[key], false))
	}

	if len(pairs) == 0 {
		return content, errors.New("не удалось разобрать OptionSettings")
	}
	return content[:start] + strings.Join(pairs, ",") + content[end:], nil
}

func ueFormat(value string, hadQuotes bool) string {
	if ueNeedsQuotes(value, hadQuotes) || value == "" {
		return `"` + strings.ReplaceAll(value, `"`, "'") + `"`
	}
	return value
}
