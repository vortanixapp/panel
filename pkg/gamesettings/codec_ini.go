package gamesettings

import "strings"

func init() { registerCodec(FormatINI, iniCodec{}) }

// iniCodec — ini с секциями и без.
//
// Без секций живёт Project Zomboid (`PublicName=...` с первой строки), с
// секциями — все Unreal-игры: ARK, Conan, Icarus, Mordhau, Satisfactory.
//
// Физический ключ — «секция/ключ», а для ключа вне секций просто «ключ».
// Секции Unreal сами содержат косые черты (`[/Script/Engine.GameSession]`),
// поэтому имя ключа отделяется по ПОСЛЕДНЕЙ косой:
// `/Script/Engine.GameSession/MaxPlayers` — это секция `/Script/Engine.GameSession`
// и ключ `MaxPlayers`.
type iniCodec struct{}

func splitINIKey(full string) (section, key string) {
	if i := strings.LastIndexByte(full, '/'); i >= 0 {
		return full[:i], full[i+1:]
	}
	return "", full
}

func joinINIKey(section, key string) string {
	if section == "" {
		return key
	}
	return section + "/" + key
}

func iniSectionHeader(line string) (string, bool) {
	t := strings.TrimSpace(line)
	if len(t) >= 2 && t[0] == '[' && t[len(t)-1] == ']' {
		return t[1 : len(t)-1], true
	}
	return "", false
}

func iniSplitPair(line string) (key, value string, ok bool) {
	t := strings.TrimSpace(line)
	if t == "" || strings.HasPrefix(t, ";") || strings.HasPrefix(t, "#") {
		return "", "", false
	}
	i := strings.IndexByte(t, '=')
	if i <= 0 {
		return "", "", false
	}
	// Режем по ПЕРВОМУ знаку равенства: приветствие Project Zomboid и описание
	// сервера вполне содержат «=» внутри значения.
	return strings.TrimSpace(t[:i]), strings.TrimSpace(t[i+1:]), true
}

func (iniCodec) Parse(content string) map[string]string {
	out := map[string]string{}
	section := ""
	for _, line := range splitLines(content) {
		if name, ok := iniSectionHeader(line); ok {
			section = name
			continue
		}
		key, value, ok := iniSplitPair(line)
		if !ok {
			continue
		}
		full := joinINIKey(section, key)
		if _, seen := out[full]; seen {
			continue
		}
		out[full] = value
	}
	return out
}

func (iniCodec) Apply(content string, changes map[string]string) (string, error) {
	ending := lineEnding(content)
	lines := splitLines(content)

	// Что ещё не записано, разложенное по секциям.
	pending := map[string]map[string]string{}
	for full, v := range changes {
		section, key := splitINIKey(full)
		if pending[section] == nil {
			pending[section] = map[string]string{}
		}
		pending[section][key] = v
	}

	section := ""
	// Индекс последней содержательной строки каждой секции — туда дописываем
	// новый ключ, чтобы он попал внутрь своей секции, а не в конец файла под
	// чужой заголовок.
	lastLineOf := map[string]int{}

	for i, line := range lines {
		if name, ok := iniSectionHeader(line); ok {
			section = name
			lastLineOf[section] = i
			continue
		}
		key, _, ok := iniSplitPair(line)
		if !ok {
			continue
		}
		lastLineOf[section] = i
		want, has := pending[section]
		if !has {
			continue
		}
		val, need := want[key]
		if !need {
			continue
		}
		delete(want, key)
		indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
		lines[i] = indent + key + "=" + val
	}

	// Дописываем оставшееся: сначала в существующие секции, потом создаём новые.
	// Порядок обхода сортированный — иначе один и тот же набор изменений давал
	// бы разный файл от запуска к запуску.
	for _, sec := range sortedSectionNames(pending) {
		want := pending[sec]
		if len(want) == 0 {
			continue
		}
		at, exists := lastLineOf[sec]
		if !exists {
			lines = appendConfigLine(lines, "["+sec+"]")
			for _, key := range sortedKeys(want) {
				lines = appendConfigLine(lines, key+"="+want[key])
			}
			continue
		}
		insert := make([]string, 0, len(want))
		for _, key := range sortedKeys(want) {
			insert = append(insert, key+"="+want[key])
		}
		lines = insertLines(lines, at+1, insert)
		// Сдвигаем запомненные позиции последующих секций.
		for name, idx := range lastLineOf {
			if idx > at {
				lastLineOf[name] = idx + len(insert)
			}
		}
	}

	return joinLines(lines, ending), nil
}

func sortedSectionNames(m map[string]map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sortStrings(out)
	return out
}

func insertLines(lines []string, at int, extra []string) []string {
	if at > len(lines) {
		at = len(lines)
	}
	out := make([]string, 0, len(lines)+len(extra))
	out = append(out, lines[:at]...)
	out = append(out, extra...)
	out = append(out, lines[at:]...)
	return out
}
