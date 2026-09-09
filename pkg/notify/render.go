package notify

import "strings"

func render(e Event) (subject, body string) {
	subject = strings.TrimSpace(e.Title)

	var b strings.Builder
	if t := strings.TrimSpace(e.Body); t != "" {
		b.WriteString(t)
	}
	if e.Action.valid() {
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(e.Action.Label + ": " + e.Action.Href)
	}
	return subject, b.String()
}

func severityPrefix(s Severity) string {
	switch s {
	case SeverityCritical:
		return "❗ "
	case SeverityWarning:
		return "⚠ "
	default:
		return ""
	}
}
