package notify

import "strings"

// render готовит тему и тело для внешних каналов.
//
// Шаблонизатора здесь намеренно нет. Тексты собираются в месте, где событие
// происходит: только там известно имя сервера, сумма и срок. Задача этого файла
// — привести их к виду, пригодному для канала, и добавить то, что человеку
// нужно, а событию знать не обязательно: подпись действия и ссылку.
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

// severityPrefix — короткая пометка важности для каналов без оформления.
//
// В письме важность видна по теме, а в Telegram и Discord сообщения идут
// сплошным потоком, и «сервер упал» должно отличаться от «копия готова» с
// первого взгляда.
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
