package gamesettings

import (
	"sort"
	"strings"
)

func SplitArgs(s string) []string {
	var (
		out   []string
		cur   strings.Builder
		quote rune
		open  bool
	)
	flush := func() {
		if open {
			out = append(out, cur.String())
			cur.Reset()
			open = false
		}
	}
	for _, r := range s {
		switch {
		case quote != 0:
			cur.WriteRune(r)
			if r == quote {
				quote = 0
			}
		case r == '"' || r == '\'':
			quote = r
			cur.WriteRune(r)
			open = true
		case r == ' ' || r == '\t' || r == '\n' || r == '\r':
			flush()
		default:
			cur.WriteRune(r)
			open = true
		}
	}
	flush()
	return out
}

func Unquote(tok string) string {
	if len(tok) >= 2 {
		if (tok[0] == '"' && tok[len(tok)-1] == '"') || (tok[0] == '\'' && tok[len(tok)-1] == '\'') {
			return tok[1 : len(tok)-1]
		}
	}
	return tok
}

func QuoteArg(v string) string {
	if v == "" {
		return `""`
	}
	if !strings.ContainsAny(v, " \t\"'") {
		return v
	}
	return `"` + strings.ReplaceAll(v, `"`, `'`) + `"`
}

func IsFlag(tok string) bool {
	if len(tok) < 2 {
		return false
	}
	if tok[0] != '-' && tok[0] != '+' {
		return false
	}
	c := tok[1]
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '-'
}

func ParseArgs(s string, flags []string) map[string]string {
	tokens := SplitArgs(s)
	want := make(map[string]string, len(flags))
	for _, f := range flags {
		want[strings.ToLower(f)] = f
	}

	out := map[string]string{}
	for i, tok := range tokens {
		if !IsFlag(tok) {
			continue
		}
		name, ok := want[strings.ToLower(strings.TrimLeft(tok, "-+"))]
		if !ok {
			continue
		}
		if _, seen := out[name]; seen {
			continue
		}
		if i+1 < len(tokens) && !IsFlag(tokens[i+1]) {
			out[name] = Unquote(tokens[i+1])
		} else {
			out[name] = ""
		}
	}
	return out
}

func ApplyArgs(s string, changes map[string]string, prefix string) string {
	if prefix == "" {
		prefix = "-"
	}
	tokens := SplitArgs(s)
	pending := make(map[string]string, len(changes))
	original := make(map[string]string, len(changes))
	for k, v := range changes {
		lower := strings.ToLower(k)
		pending[lower] = v
		original[lower] = k
	}

	out := make([]string, 0, len(tokens)+2*len(changes))
	for i := 0; i < len(tokens); i++ {
		tok := tokens[i]
		if !IsFlag(tok) {
			out = append(out, tok)
			continue
		}
		key := strings.ToLower(strings.TrimLeft(tok, "-+"))
		val, ok := pending[key]
		if !ok {
			out = append(out, tok)
			continue
		}
		delete(pending, key)

		hasValue := i+1 < len(tokens) && !IsFlag(tokens[i+1])
		if val == "" {
			if hasValue {
				i++
			}
			continue
		}
		out = append(out, tok, QuoteArg(val))
		if hasValue {
			i++
		}
	}

	rest := make([]string, 0, len(pending))
	for k := range pending {
		rest = append(rest, k)
	}
	sort.Strings(rest)
	for _, k := range rest {
		if pending[k] == "" {
			continue
		}
		out = append(out, prefix+original[k], QuoteArg(pending[k]))
	}
	return strings.Join(out, " ")
}
