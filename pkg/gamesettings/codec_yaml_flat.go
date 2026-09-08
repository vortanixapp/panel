package gamesettings

import "strings"

func init() { registerCodec(FormatYAMLFlat, yamlFlatCodec{}) }

// yamlFlatCodec — плоский YAML в два уровня, dedicated.yaml Empyrion:
//
//	ServerConfig:
//	  Srv_Name: Мой сервер
//	  Srv_MaxPlayers: 8
//	GameConfig:
//	  GameName: Default
//
// Физический ключ — «секция/ключ».
//
// Библиотеки yaml в модуле нет, и добавлять её ради одного файла незачем: здесь
// не нужен ни один сложный приём формата — ни списки, ни якоря, ни многострочные
// значения. А правка на месте всё равно потребовала бы работы с текстом:
// разбор и пересборка библиотекой уничтожили бы комментарии и порядок.
type yamlFlatCodec struct{}

// yamlSplit разбирает строку на отступ, ключ и значение.
func yamlSplit(line string) (indent, key, value string, ok bool) {
	if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "#") {
		return "", "", "", false
	}
	indent = line[:len(line)-len(strings.TrimLeft(line, " \t"))]
	rest := line[len(indent):]
	i := strings.IndexByte(rest, ':')
	if i < 0 {
		return "", "", "", false
	}
	key = strings.TrimSpace(rest[:i])
	value = strings.TrimSpace(rest[i+1:])
	if key == "" {
		return "", "", "", false
	}
	// Хвостовой комментарий: отделяется пробелом, иначе решётка внутри значения
	// считалась бы началом комментария.
	if j := strings.Index(value, " #"); j >= 0 {
		value = strings.TrimSpace(value[:j])
	}
	return indent, key, value, true
}

func yamlUnquote(v string) string {
	if len(v) >= 2 {
		if (v[0] == '\'' && v[len(v)-1] == '\'') || (v[0] == '"' && v[len(v)-1] == '"') {
			return v[1 : len(v)-1]
		}
	}
	return v
}

// yamlQuote берёт значение в кавычки там, где без них YAML прочитает его иначе.
func yamlQuote(v string, wasQuoted bool) string {
	if v == "" {
		return "''"
	}
	if wasQuoted || strings.ContainsAny(v, ":#") || v != strings.TrimSpace(v) {
		return "'" + strings.ReplaceAll(v, "'", "''") + "'"
	}
	return v
}

func (yamlFlatCodec) Parse(content string) map[string]string {
	out := map[string]string{}
	section := ""
	for _, line := range splitLines(content) {
		indent, key, value, ok := yamlSplit(line)
		if !ok {
			continue
		}
		if indent == "" {
			section = key
			// У самой секции значения нет — только заголовок.
			if value == "" {
				continue
			}
			// Ключ верхнего уровня со значением тоже бывает.
			if _, seen := out[key]; !seen {
				out[key] = yamlUnquote(value)
			}
			continue
		}
		full := joinINIKey(section, key)
		if _, seen := out[full]; seen {
			continue
		}
		out[full] = yamlUnquote(value)
	}
	return out
}

func (yamlFlatCodec) Apply(content string, changes map[string]string) (string, error) {
	ending := lineEnding(content)
	lines := splitLines(content)

	pending := map[string]map[string]string{}
	for full, v := range changes {
		section, key := splitINIKey(full)
		if pending[section] == nil {
			pending[section] = map[string]string{}
		}
		pending[section][key] = v
	}

	section := ""
	lastLineOf := map[string]int{}
	indentOf := map[string]string{}

	for i, line := range lines {
		indent, key, value, ok := yamlSplit(line)
		if !ok {
			continue
		}
		if indent == "" {
			section = key
			lastLineOf[section] = i
			continue
		}
		lastLineOf[section] = i
		if _, has := indentOf[section]; !has {
			indentOf[section] = indent
		}
		want, has := pending[section]
		if !has {
			continue
		}
		val, need := want[key]
		if !need {
			continue
		}
		delete(want, key)
		wasQuoted := yamlUnquote(value) != value
		lines[i] = indent + key + ": " + yamlQuote(val, wasQuoted)
	}

	for _, sec := range sortedSectionNames(pending) {
		want := pending[sec]
		if len(want) == 0 {
			continue
		}
		indent := indentOf[sec]
		if indent == "" {
			indent = "  "
		}
		at, exists := lastLineOf[sec]
		if !exists {
			lines = appendConfigLine(lines, sec+":")
			for _, key := range sortedKeys(want) {
				lines = appendConfigLine(lines, indent+key+": "+yamlQuote(want[key], false))
			}
			continue
		}
		insert := make([]string, 0, len(want))
		for _, key := range sortedKeys(want) {
			insert = append(insert, indent+key+": "+yamlQuote(want[key], false))
		}
		lines = insertLines(lines, at+1, insert)
		for name, idx := range lastLineOf {
			if idx > at {
				lastLineOf[name] = idx + len(insert)
			}
		}
	}

	return joinLines(lines, ending), nil
}
