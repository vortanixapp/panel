package gamesettings

import "strings"

type lineSyntax struct {
	separator string

	equals bool

	comments []string

	inlineComments []string

	quote quotePolicy

	trailingSemicolon bool
}

type quotePolicy int

const (
	quoteNever quotePolicy = iota
	quoteAlways
	quoteWhenNeeded
)

type lineCodec struct{ syn lineSyntax }

type parsedLine struct {
	key     string
	value   string
	comment string
	quoted  bool
	indent  string
}

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
	return out + p.comment
}

func (c lineCodec) Parse(content string) map[string]string {
	out := map[string]string{}
	for _, line := range splitLines(content) {
		p, ok := c.splitLine(line)
		if !ok {
			continue
		}
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

	for _, lower := range sortedKeys(pending) {
		val := pending[lower]
		lines = appendConfigLine(lines, c.renderLine(parsedLine{key: original[lower]}, val))
	}

	return joinLines(lines, ending), nil
}

func appendConfigLine(lines []string, line string) []string {
	if n := len(lines); n > 0 && strings.TrimSpace(lines[n-1]) == "" {
		lines[n-1] = line
		return append(lines, "")
	}
	return append(lines, line, "")
}
