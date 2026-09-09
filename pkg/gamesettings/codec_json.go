package gamesettings

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

func init() { registerCodec(FormatJSON, jsonCodec{}) }

type jsonCodec struct{}

type jsonSpan struct{ start, end int }

func scanJSON(src string) (map[string]jsonSpan, map[string]string, error) {
	spans := map[string]jsonSpan{}
	vals := map[string]string{}
	dec := json.NewDecoder(strings.NewReader(src))
	dec.UseNumber()
	if err := jsonWalk(dec, src, "", spans, vals); err != nil {
		return nil, nil, err
	}
	return spans, vals, nil
}

func jsonChild(path, key string) string {
	if path == "" {
		return key
	}
	return path + "." + key
}

func skipToValue(src string, from int) int {
	for from < len(src) {
		switch src[from] {
		case ' ', '\t', '\n', '\r', ':', ',':
			from++
		default:
			return from
		}
	}
	return from
}

func jsonWalk(dec *json.Decoder, src, path string, spans map[string]jsonSpan, vals map[string]string) error {
	before := int(dec.InputOffset())
	tok, err := dec.Token()
	if err != nil {
		return err
	}
	start := skipToValue(src, before)

	if delim, ok := tok.(json.Delim); ok {
		switch delim {
		case '{':
			for dec.More() {
				keyTok, err := dec.Token()
				if err != nil {
					return err
				}
				key, _ := keyTok.(string)
				if err := jsonWalk(dec, src, jsonChild(path, key), spans, vals); err != nil {
					return err
				}
			}
			if _, err := dec.Token(); err != nil {
				return err
			}
		case '[':
			i := 0
			for dec.More() {
				if err := jsonWalk(dec, src, fmt.Sprintf("%s[%d]", path, i), spans, vals); err != nil {
					return err
				}
				i++
			}
			if _, err := dec.Token(); err != nil {
				return err
			}
		}
		spans[path] = jsonSpan{start: start, end: int(dec.InputOffset())}
		return nil
	}

	spans[path] = jsonSpan{start: start, end: int(dec.InputOffset())}
	vals[path] = jsonTokenString(tok)
	return nil
}

func jsonTokenString(tok any) string {
	switch v := tok.(type) {
	case nil:
		return ""
	case string:
		return v
	case bool:
		if v {
			return "true"
		}
		return "false"
	case json.Number:
		return v.String()
	default:
		return fmt.Sprint(v)
	}
}

func resolveSelector(path string, vals map[string]string) string {
	open := strings.IndexByte(path, '[')
	if open < 0 {
		return path
	}
	close := strings.IndexByte(path[open:], ']')
	if close < 0 {
		return path
	}
	close += open
	inner := path[open+1 : close]
	eq := strings.IndexByte(inner, '=')
	if eq < 0 {
		rest := resolveSelector(path[close+1:], vals)
		return path[:close+1] + rest
	}
	arr := path[:open]
	field, want := inner[:eq], inner[eq+1:]
	for i := 0; ; i++ {
		probe := fmt.Sprintf("%s[%d].%s", arr, i, field)
		got, ok := vals[probe]
		if !ok {
			return path
		}
		if got == want {
			rest := resolveSelector(path[close+1:], vals)
			return fmt.Sprintf("%s[%d]%s", arr, i, rest)
		}
	}
}

func (jsonCodec) Parse(content string) map[string]string {
	if strings.TrimSpace(content) == "" {
		return map[string]string{}
	}
	_, vals, err := scanJSON(content)
	if err != nil {
		return map[string]string{}
	}
	delete(vals, "")
	return vals
}

func jsonEncode(value, previous string, known bool) string {
	if known {
		switch {
		case previous == "true" || previous == "false":
			if b, ok := ParseBool(value); ok {
				return strconv.FormatBool(b)
			}
		case previous != "" && isJSONNumber(previous):
			if isJSONNumber(value) {
				return value
			}
		}
		b, _ := json.Marshal(value)
		return string(b)
	}
	if b, ok := ParseBool(value); ok && (value == "true" || value == "false") {
		return strconv.FormatBool(b)
	}
	if isJSONNumber(value) {
		return value
	}
	b, _ := json.Marshal(value)
	return string(b)
}

func isJSONNumber(s string) bool {
	if s == "" {
		return false
	}
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

func (jsonCodec) Apply(content string, changes map[string]string) (string, error) {
	if strings.TrimSpace(content) == "" {
		content = "{}"
	}
	spans, vals, err := scanJSON(content)
	if err != nil {
		return "", fmt.Errorf("файл не разбирается как JSON: %w", err)
	}

	type edit struct {
		span jsonSpan
		text string
	}
	var edits []edit
	missing := map[string]string{}

	for _, path := range sortedKeys(changes) {
		resolved := resolveSelector(path, vals)
		span, ok := spans[resolved]
		if !ok {
			missing[resolved] = changes[path]
			continue
		}
		prev, known := vals[resolved]
		edits = append(edits, edit{span: span, text: jsonEncode(changes[path], prev, known)})
	}

	for i := len(edits) - 1; i >= 0; i-- {
		for j := 0; j < i; j++ {
			if edits[j].span.start > edits[j+1].span.start {
				edits[j], edits[j+1] = edits[j+1], edits[j]
			}
		}
	}
	for i := len(edits) - 1; i >= 0; i-- {
		e := edits[i]
		content = content[:e.span.start] + e.text + content[e.span.end:]
	}

	if len(missing) == 0 {
		return content, nil
	}
	for _, path := range sortedKeys(missing) {
		var insertErr error
		content, insertErr = insertJSONPath(content, path, missing[path])
		if insertErr != nil {
			return "", insertErr
		}
	}
	return content, nil
}

func insertJSONPath(content, path, value string) (string, error) {
	spans, vals, err := scanJSON(content)
	if err != nil {
		return "", err
	}
	dot := strings.LastIndexByte(path, '.')
	parent, key := "", path
	if dot >= 0 {
		parent, key = path[:dot], path[dot+1:]
	}
	span, ok := spans[parent]
	if !ok {
		return "", fmt.Errorf("в конфиге нет раздела %q для параметра %q", parent, key)
	}
	body := content[span.start:span.end]
	if len(body) < 2 || body[0] != '{' {
		return "", fmt.Errorf("раздел %q не является объектом", parent)
	}

	encoded := jsonEncode(value, "", false)
	keyJSON, _ := json.Marshal(key)
	entry := string(keyJSON) + ": " + encoded

	inner := strings.TrimSpace(body[1 : len(body)-1])
	closing := span.end - 1
	if inner == "" {
		return content[:span.start] + "{" + entry + "}" + content[span.end:], nil
	}

	indent := jsonIndentOf(content, span.start)
	insertion := ",\n" + indent + entry
	at := closing
	for at > span.start && isJSONSpace(content[at-1]) {
		at--
	}
	_ = vals
	return content[:at] + insertion + content[at:], nil
}

func isJSONSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}

func jsonIndentOf(content string, objectStart int) string {
	nl := strings.IndexByte(content[objectStart:], '\n')
	if nl < 0 {
		return "  "
	}
	rest := content[objectStart+nl+1:]
	indent := rest[:len(rest)-len(strings.TrimLeft(rest, " \t"))]
	if indent == "" {
		return "  "
	}
	return indent
}
