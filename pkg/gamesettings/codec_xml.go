package gamesettings

import (
	"errors"
	"strings"
)

func init() {
	registerCodec(FormatXMLProps, xmlPropsCodec{})
	registerCodec(FormatXMLTags, xmlTagsCodec{})
}

// Правка XML текстом, а не через encoding/xml.
//
// Прежний код 7 Days to Die разбирал файл в структуру и пересобирал через
// xml.Marshal. Это уничтожало XML-декларацию, комментарии и отступы, превращало
// `<property .../>` в `<property ...></property>` и складывало весь конфиг в одну
// строку. Заодно у структуры не было поля XMLName, и Marshal писал корневым тегом
// имя Go-типа — `<sevenDaysXML>` вместо `<ServerSettings>`, после чего игра
// вообще отказывалась стартовать.
//
// Разбирать XML для чтения можно чем угодно, но запись обязана быть точечной
// заменой в исходном тексте: контракт требует, чтобы всё постороннее осталось
// нетронутым.

// commentMask отмечает байты, попавшие внутрь <!-- ... -->.
//
// Без этой маски комментарий вида `<!-- <password>секрет</password> -->`
// принимался за настоящую настройку: именно так и вело себя разборщик MTA,
// собранный на регулярных выражениях.
func commentMask(s string) []bool {
	mask := make([]bool, len(s))
	for i := 0; i < len(s); {
		if strings.HasPrefix(s[i:], "<!--") {
			end := strings.Index(s[i:], "-->")
			if end < 0 {
				for ; i < len(s); i++ {
					mask[i] = true
				}
				break
			}
			for j := i; j < i+end+3; j++ {
				mask[j] = true
			}
			i += end + 3
			continue
		}
		i++
	}
	return mask
}

func xmlUnescape(s string) string {
	r := strings.NewReplacer("&lt;", "<", "&gt;", ">", "&quot;", `"`, "&#34;", `"`, "&apos;", "'", "&#39;", "'", "&amp;", "&")
	return r.Replace(s)
}

func xmlEscapeAttr(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")
	return r.Replace(s)
}

func xmlEscapeText(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
	return r.Replace(s)
}

// findAttr ищет значение атрибута внутри куска разметки тега.
// Возвращает границы значения (без кавычек) и сам текст.
func findAttr(tag string, name string) (start, end int, value string, ok bool) {
	i := 0
	for {
		j := strings.Index(tag[i:], name)
		if j < 0 {
			return 0, 0, "", false
		}
		j += i
		// Слева должен быть пробел, иначе `name` совпадёт с хвостом другого
		// атрибута — например `nickname` содержит `name`.
		if j == 0 || (tag[j-1] != ' ' && tag[j-1] != '\t' && tag[j-1] != '\n') {
			i = j + len(name)
			continue
		}
		k := j + len(name)
		for k < len(tag) && (tag[k] == ' ' || tag[k] == '\t') {
			k++
		}
		if k >= len(tag) || tag[k] != '=' {
			i = j + len(name)
			continue
		}
		k++
		for k < len(tag) && (tag[k] == ' ' || tag[k] == '\t') {
			k++
		}
		if k >= len(tag) || (tag[k] != '"' && tag[k] != '\'') {
			i = j + len(name)
			continue
		}
		quote := tag[k]
		k++
		e := strings.IndexByte(tag[k:], quote)
		if e < 0 {
			return 0, 0, "", false
		}
		return k, k + e, tag[k : k+e], true
	}
}

// xmlPropsCodec — 7 Days to Die: <property name="X" value="Y"/>.
type xmlPropsCodec struct{}

// eachProperty обходит теги property вне комментариев.
// cb получает границы всего тега в исходной строке и его текст.
func eachProperty(content string, cb func(tagStart, tagEnd int, tag string) bool) {
	mask := commentMask(content)
	for i := 0; i < len(content); i++ {
		if mask[i] || content[i] != '<' {
			continue
		}
		if !strings.HasPrefix(content[i:], "<property") {
			continue
		}
		end := strings.IndexByte(content[i:], '>')
		if end < 0 {
			return
		}
		end += i + 1
		if !cb(i, end, content[i:end]) {
			return
		}
		i = end - 1
	}
}

func (xmlPropsCodec) Parse(content string) map[string]string {
	out := map[string]string{}
	eachProperty(content, func(_, _ int, tag string) bool {
		_, _, name, ok := findAttr(tag, "name")
		if !ok {
			return true
		}
		name = strings.TrimSpace(xmlUnescape(name))
		if name == "" {
			return true
		}
		// Побеждает первое вхождение — его же правит Apply. Раньше разбор брал
		// последнее, а запись меняла первое: записали одно, прочитали другое.
		if _, seen := out[name]; seen {
			return true
		}
		_, _, value, _ := findAttr(tag, "value")
		out[name] = xmlUnescape(value)
		return true
	})
	return out
}

func (c xmlPropsCodec) Apply(content string, changes map[string]string) (string, error) {
	if strings.TrimSpace(content) == "" {
		content = "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<ServerSettings>\n</ServerSettings>\n"
	}
	if !strings.Contains(content, "<ServerSettings") {
		return "", errors.New("не найден корневой элемент ServerSettings")
	}

	remaining := make(map[string]string, len(changes))
	for k, v := range changes {
		remaining[k] = v
	}

	// Идём с конца, чтобы уже вычисленные границы не съезжали после замены.
	type edit struct {
		start, end int
		text       string
	}
	var edits []edit
	seen := map[string]bool{}
	eachProperty(content, func(ts, te int, tag string) bool {
		_, _, name, ok := findAttr(tag, "name")
		if !ok {
			return true
		}
		name = strings.TrimSpace(xmlUnescape(name))
		if seen[name] {
			return true
		}
		seen[name] = true
		val, want := remaining[name]
		if !want {
			return true
		}
		delete(remaining, name)
		vs, ve, _, hasValue := findAttr(tag, "value")
		if !hasValue {
			return true
		}
		edits = append(edits, edit{start: ts + vs, end: ts + ve, text: xmlEscapeAttr(val)})
		return true
	})

	for i := len(edits) - 1; i >= 0; i-- {
		e := edits[i]
		content = content[:e.start] + e.text + content[e.end:]
	}

	if len(remaining) == 0 {
		return content, nil
	}

	// Дописываем недостающие перед закрывающим тегом, повторяя отступ соседей.
	closing := strings.LastIndex(content, "</ServerSettings>")
	if closing < 0 {
		return "", errors.New("не найден закрывающий тег ServerSettings")
	}
	indent := detectIndent(content, "<property")
	var b strings.Builder
	for _, name := range sortedKeys(remaining) {
		b.WriteString(indent + `<property name="` + xmlEscapeAttr(name) +
			`" value="` + xmlEscapeAttr(remaining[name]) + `"/>` + "\n")
	}
	return content[:closing] + b.String() + content[closing:], nil
}

// detectIndent подбирает отступ по первому такому же тегу в файле.
func detectIndent(content, tag string) string {
	i := strings.Index(content, tag)
	if i < 0 {
		return "    "
	}
	start := strings.LastIndexByte(content[:i], '\n') + 1
	indent := content[start:i]
	if strings.TrimSpace(indent) != "" {
		return "    "
	}
	return indent
}

// xmlTagsCodec — MTA:SA: <servername>значение</servername>.
type xmlTagsCodec struct{}

// eachTag обходит парные теги вне комментариев.
// cb получает имя, границы внутреннего текста и границы всего элемента.
func eachTag(content string, cb func(name string, innerStart, innerEnd int) bool) {
	mask := commentMask(content)
	for i := 0; i < len(content); i++ {
		if mask[i] || content[i] != '<' {
			continue
		}
		if i+1 < len(content) && (content[i+1] == '/' || content[i+1] == '?' || content[i+1] == '!') {
			continue
		}
		gt := strings.IndexByte(content[i:], '>')
		if gt < 0 {
			return
		}
		gt += i
		inner := content[i+1 : gt]
		if strings.HasSuffix(inner, "/") {
			i = gt
			continue
		}
		name := inner
		// Тег может нести атрибуты: <servername lang="ru">. Прежний разбор их
		// не понимал и такую строку просто не находил.
		if sp := strings.IndexAny(name, " \t\n"); sp >= 0 {
			name = name[:sp]
		}
		if name == "" {
			i = gt
			continue
		}
		closeTag := "</" + name + ">"
		ce := strings.Index(content[gt:], closeTag)
		if ce < 0 {
			i = gt
			continue
		}
		ce += gt
		if !cb(name, gt+1, ce) {
			return
		}
		i = gt
	}
}

func (xmlTagsCodec) Parse(content string) map[string]string {
	out := map[string]string{}
	eachTag(content, func(name string, s, e int) bool {
		value := content[s:e]
		// Вложенные элементы не значение: <config> содержит остальные теги.
		if strings.Contains(value, "<") {
			return true
		}
		if _, seen := out[name]; seen {
			return true
		}
		out[name] = xmlUnescape(strings.TrimSpace(value))
		return true
	})
	return out
}

func (xmlTagsCodec) Apply(content string, changes map[string]string) (string, error) {
	if strings.TrimSpace(content) == "" {
		content = "<config>\n</config>\n"
	}
	remaining := make(map[string]string, len(changes))
	for k, v := range changes {
		remaining[k] = v
	}

	type edit struct {
		start, end int
		text       string
	}
	var edits []edit
	seen := map[string]bool{}
	eachTag(content, func(name string, s, e int) bool {
		if strings.Contains(content[s:e], "<") || seen[name] {
			return true
		}
		seen[name] = true
		val, want := remaining[name]
		if !want {
			return true
		}
		delete(remaining, name)
		edits = append(edits, edit{start: s, end: e, text: xmlEscapeText(val)})
		return true
	})
	for i := len(edits) - 1; i >= 0; i-- {
		e := edits[i]
		content = content[:e.start] + e.text + content[e.end:]
	}

	if len(remaining) == 0 {
		return content, nil
	}
	closing := strings.LastIndex(content, "</config>")
	if closing < 0 {
		return "", errors.New("не найден закрывающий тег config")
	}
	indent := detectIndent(content, "<servername")
	var b strings.Builder
	for _, name := range sortedKeys(remaining) {
		b.WriteString(indent + "<" + name + ">" + xmlEscapeText(remaining[name]) + "</" + name + ">\n")
	}
	return content[:closing] + b.String() + content[closing:], nil
}
