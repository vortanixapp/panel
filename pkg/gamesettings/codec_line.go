package gamesettings

import "strings"

// Общая машинка для построчных форматов «ключ значение».
//
// Их в проекте три с половиной: server.cfg SA-MP (`ключ значение`), конфиги
// GoldSrc и Source (`ключ "значение"`), server.properties и rust.env
// (`ключ=значение`). Различаются они разделителем, знаком комментария и тем,
// когда значение берётся в кавычки, — а всё остальное у них общее: правка на
// месте, сохранение комментариев и порядка, дописывание недостающего в конец.
//
// Раньше каждый формат имел свой аппликатор, и повторяющиеся куски разъехались:
// один умел удалять ключи, другой сохранял регистр, третий терял хвостовые
// комментарии. Здесь эта часть одна.

// lineSyntax описывает конкретный формат.
type lineSyntax struct {
	// separator — чем ключ отделяется от значения при записи. Для форматов с
	// пробелом это " ", для properties и env — "=".
	separator string

	// equals — разделитель является знаком равенства. Тогда при разборе ключ
	// режется по первому "=", а не по пробельному промежутку.
	equals bool

	// comments — с чего начинается строка-комментарий.
	comments []string

	// inlineComments — что считается комментарием в конце строки. Отделяется от
	// значения обязательно пробелом: без этого условия `http://example.com`
	// обрезалось бы по `//`, а `key value;` — по точке с запятой.
	inlineComments []string

	// quote — когда значение оборачивать в кавычки.
	quote quotePolicy

	// trailingSemicolon — строка заканчивается точкой с запятой (serverDZ.cfg).
	trailingSemicolon bool
}

type quotePolicy int

const (
	// quoteNever — кавычки не ставим никогда (SA-MP, properties, env).
	quoteNever quotePolicy = iota
	// quoteAlways — кавычки ставим всегда (конфиги Source).
	quoteAlways
	// quoteWhenNeeded — только если без них значение распадётся или было в кавычках.
	quoteWhenNeeded
)

type lineCodec struct{ syn lineSyntax }

// parsedLine — разобранная строка конфига.
type parsedLine struct {
	key     string // как записан в файле
	value   string // без кавычек и без хвостового комментария
	comment string // хвостовой комментарий вместе с ведущими пробелами
	quoted  bool   // значение было в кавычках
	indent  string // отступ строки
}

// splitLine разбирает строку. Второе значение — удалось ли.
func (c lineCodec) splitLine(line string) (parsedLine, bool) {
	for _, m := range c.syn.comments {
		if strings.HasPrefix(strings.TrimSpace(line), m) {
			return parsedLine{}, false
		}
	}
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return parsedLine{}, false
	}
	indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]

	var key, rest string
	if c.syn.equals {
		i := strings.IndexByte(trimmed, '=')
		if i <= 0 {
			return parsedLine{}, false
		}
		key = strings.TrimSpace(trimmed[:i])
		rest = trimmed[i+1:]
	} else {
		i := strings.IndexAny(trimmed, " \t")
		if i <= 0 {
			// Ключ без значения — это законная запись: `password` в server.cfg
			// SA-MP означает пустой пароль. Прежняя регулярка требовала пробел
			// после ключа, и такая строка была парсеру не видна вовсе.
			return parsedLine{key: trimmed, indent: indent}, true
		}
		key = trimmed[:i]
		rest = trimmed[i+1:]
	}

	value, comment := c.splitInlineComment(rest)
	value = strings.TrimSpace(value)
	if c.syn.trailingSemicolon {
		value = strings.TrimRight(value, ";")
		value = strings.TrimSpace(value)
	}
	quoted := false
	if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
		value = value[1 : len(value)-1]
		quoted = true
	}
	return parsedLine{key: key, value: value, comment: comment, quoted: quoted, indent: indent}, true
}

// splitInlineComment отрезает хвостовой комментарий.
//
// Комментарием считается только знак, перед которым стоит пробел и который не
// попал внутрь кавычек. Иначе `sv_downloadurl "http://fastdl/"` обрезалось бы по
// двойной косой черте прямо в середине адреса.
func (c lineCodec) splitInlineComment(rest string) (value, comment string) {
	inQuote := false
	for i := 0; i < len(rest); i++ {
		ch := rest[i]
		if ch == '"' {
			inQuote = !inQuote
			continue
		}
		if inQuote || (ch != ' ' && ch != '\t') {
			continue
		}
		for _, m := range c.syn.inlineComments {
			if strings.HasPrefix(rest[i+1:], m) {
				return rest[:i], rest[i:]
			}
		}
	}
	return rest, ""
}

func (c lineCodec) formatValue(v string, wasQuoted bool) string {
	switch c.syn.quote {
	case quoteAlways:
		return `"` + strings.ReplaceAll(v, `"`, "'") + `"`
	case quoteWhenNeeded:
		if wasQuoted || v == "" || strings.ContainsAny(v, " \t\"") {
			return `"` + strings.ReplaceAll(v, `"`, "'") + `"`
		}
		return v
	default:
		return v
	}
}

func (c lineCodec) renderLine(p parsedLine, value string) string {
	sep := c.syn.separator
	out := p.indent + p.key + sep + c.formatValue(value, p.quoted)
	if c.syn.trailingSemicolon {
		out += ";"
	}
	// Хвостовой комментарий остаётся на месте: он объясняет именно эту строку, и
	// потерять его при правке значения — потерять единственную подсказку о том,
	// что параметр значит.
	return out + p.comment
}

func (c lineCodec) Parse(content string) map[string]string {
	out := map[string]string{}
	for _, line := range splitLines(content) {
		p, ok := c.splitLine(line)
		if !ok {
			continue
		}
		// Побеждает первое вхождение — его же правит Apply. Раньше эти две
		// стороны расходились: разбор брал последнее, запись меняла первое, и
		// клиент видел, что настройка «не сохраняется».
		if _, seen := out[p.key]; seen {
			continue
		}
		out[p.key] = p.value
	}
	return out
}

func (c lineCodec) Apply(content string, changes map[string]string) (string, error) {
	ending := lineEnding(content)
	lines := splitLines(content)

	pending := make(map[string]string, len(changes))
	original := make(map[string]string, len(changes))
	for k, v := range changes {
		lower := strings.ToLower(k)
		pending[lower] = v
		original[lower] = k
	}

	for i, line := range lines {
		p, ok := c.splitLine(line)
		if !ok {
			continue
		}
		lower := strings.ToLower(p.key)
		val, want := pending[lower]
		if !want {
			continue
		}
		delete(pending, lower)
		lines[i] = c.renderLine(p, val)
	}

	// Порядок дописывания устойчивый. Обход карты в Go случаен, и без сортировки
	// один и тот же набор изменений давал разный файл от запуска к запуску:
	// смысл тот же, а diff и резервные копии каждый раз разные.
	for _, lower := range sortedKeys(pending) {
		val := pending[lower]
		lines = appendConfigLine(lines, c.renderLine(parsedLine{key: original[lower]}, val))
	}

	return joinLines(lines, ending), nil
}

// appendConfigLine дописывает строку в конец, не плодя пустых.
//
// splitLines оставляет пустой элемент от завершающего перевода строки, и
// прежний код клал новый ключ после него — файл обрастал пустыми строками при
// каждом сохранении.
func appendConfigLine(lines []string, line string) []string {
	if n := len(lines); n > 0 && strings.TrimSpace(lines[n-1]) == "" {
		lines[n-1] = line
		return append(lines, "")
	}
	return append(lines, line, "")
}
