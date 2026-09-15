package notify

import "strings"

func render(title, body, label, href string) (subject, text string) {
	subject = strings.TrimSpace(title)

	var b strings.Builder
	if t := strings.TrimSpace(body); t != "" {
		b.WriteString(t)
	}
	if label != "" && href != "" {
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(label + ": " + href)
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
