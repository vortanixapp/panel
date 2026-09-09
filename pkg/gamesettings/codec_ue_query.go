package gamesettings

import "strings"

func init() { registerCodec(FormatUEQuery, ueQueryCodec{}) }

type ueQueryCodec struct{}

const MapKey = "@map"

func splitUEQuery(content string) (query, rest string) {
	trimmed := strings.TrimLeft(content, " \t")
	if trimmed == "" {
		return "", ""
	}
	quote := byte(0)
	for i := 0; i < len(trimmed); i++ {
		ch := trimmed[i]
		switch {
		case quote != 0:
			if ch == quote {
				quote = 0
			}
		case ch == '"' || ch == '\'':
			quote = ch
		case ch == ' ' || ch == '\t':
			return Unquote(trimmed[:i]), trimmed[i:]
		}
	}
	return Unquote(trimmed), ""
}

func (ueQueryCodec) Parse(content string) map[string]string {
	query, _ := splitUEQuery(content)
	query = strings.TrimSpace(query)
	if query == "" {
		return map[string]string{}
	}
	out := map[string]string{}
	parts := strings.Split(query, "?")
	if parts[0] != "" {
		out[MapKey] = parts[0]
	}
	for _, p := range parts[1:] {
		if p == "" {
			continue
		}
		key, value := p, ""
		if i := strings.IndexByte(p, '='); i >= 0 {
			key, value = p[:i], p[i+1:]
		}
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if _, seen := out[key]; seen {
			continue
		}
		out[key] = value
	}
	return out
}

func (ueQueryCodec) Apply(content string, changes map[string]string) (string, error) {
	query, rest := splitUEQuery(content)
	query = strings.TrimSpace(query)

	parts := []string{}
	if query != "" {
		parts = strings.Split(query, "?")
	}
	if len(parts) == 0 {
		parts = []string{""}
	}

	if m, ok := changes[MapKey]; ok {
		parts[0] = m
	}

	pending := make(map[string]string, len(changes))
	for k, v := range changes {
		if k == MapKey {
			continue
		}
		pending[k] = v
	}

	seen := map[string]bool{}
	for i := 1; i < len(parts); i++ {
		p := parts[i]
		key := p
		if j := strings.IndexByte(p, '='); j >= 0 {
			key = p[:j]
		}
		key = strings.TrimSpace(key)
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		val, want := pending[key]
		if !want {
			continue
		}
		delete(pending, key)
		if val == "" {
			parts[i] = ""
			continue
		}
		parts[i] = key + "=" + val
	}

	for _, key := range sortedKeys(pending) {
		if pending[key] == "" {
			continue
		}
		parts = append(parts, key+"="+pending[key])
	}

	kept := parts[:0]
	for i, p := range parts {
		if i > 0 && p == "" {
			continue
		}
		kept = append(kept, p)
	}
	return QuoteArg(strings.Join(kept, "?")) + rest, nil
}
