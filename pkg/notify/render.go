package notify

import "strings"

func render(title, body string) (subject, text string) {
	return strings.TrimSpace(title), strings.TrimSpace(body)
}

func severityPrefix(s Severity) string {
	switch s {
	case SeverityCritical:
		return "❗ "
	case SeverityWarning:
		return "⚠️ "
	case SeveritySuccess:
		return "✅ "
	default:
		return ""
	}
}

func severityColor(s Severity) int {
	switch s {
	case SeverityCritical:
		return 0xE5484D
	case SeveritySuccess:
		return 0x10B981
	case SeverityWarning:
		return 0xF5A524
	default:
		return 0x3B82F6
	}
}

func truncateRunes(s string, limit int) string {
	r := []rune(s)
	if len(r) <= limit {
		return s
	}
	return string(r[:limit-1]) + "…"
}
