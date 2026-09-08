package notify

import "strings"

// Channel — способ доставки.
type Channel string

const (
	ChannelPanel    Channel = "panel" // запись в самой панели, есть всегда
	ChannelEmail    Channel = "email"
	ChannelTelegram Channel = "telegram"
	ChannelDiscord  Channel = "discord"
)

// ExternalChannels — каналы, которые уходят наружу и требуют доставки.
func ExternalChannels() []Channel {
	return []Channel{ChannelEmail, ChannelTelegram, ChannelDiscord}
}

// Action — кнопка в уведомлении.
//
// Отдельный тип, а не карта в meta. Раньше действие клали в meta объектом
// {"label":…, "url":…}, а читали строкой через fmt.Sprint — в панель приезжало
// `map[label:Продлить url:/servers/…]`, а ссылка оставалась пустой, поэтому
// кнопки «Продлить» не было в самых важных уведомлениях об истечении аренды.
type Action struct {
	Label string `json:"label"`
	Href  string `json:"href"`
}

func (a *Action) valid() bool {
	return a != nil && strings.TrimSpace(a.Label) != "" && strings.TrimSpace(a.Href) != ""
}

// Event — что произошло.
type Event struct {
	Kind Kind

	// Title и Body — то, что увидит человек. Body может быть пустым.
	Title string
	Body  string

	// Action — необязательная кнопка.
	Action *Action

	// Severity переопределяет важность из справочника. Пусто — берётся оттуда.
	Severity Severity

	// Meta — дополнительные сведения для интерфейса: server_id, node_id, сумма.
	// В текст письма не попадает.
	Meta map[string]any

	// DedupeKey — если задан, повторное событие с тем же ключом у того же
	// получателя не создаётся.
	//
	// Нужен там, где источник может сработать дважды: остановку за неоплату
	// сейчас шлют и воркер, и подметатель в core-api, и клиент получает два
	// разных уведомления об одном и том же.
	DedupeKey string
}

// severity возвращает важность события с учётом справочника.
func (e Event) severity() Severity {
	if e.Severity != "" {
		return e.Severity
	}
	return DefFor(e.Kind).Severity
}

// Recipient — кому доставлять.
type Recipient struct {
	UserID string
	Email  string

	// Prefs — что включил сам получатель. Нулевое значение означает «только
	// почта», как и умолчание в базе.
	Prefs Prefs
}

// Prefs — настройки каналов получателя, зеркало core.user_notification_channels.
type Prefs struct {
	Email    bool
	Telegram bool
	Discord  bool

	TelegramChatID string
	DiscordWebhook string
}

// Target возвращает адрес доставки для канала и признак, что канал пригоден.
//
// Включённый тумблер без адреса каналом не считается: раньше переключатель на
// странице оповещений позволял включить Telegram вообще без chat id, и клиент
// оставался в уверенности, что подписался.
func (r Recipient) Target(c Channel) (string, bool) {
	switch c {
	case ChannelEmail:
		if !r.Prefs.Email {
			return "", false
		}
		return strings.TrimSpace(r.Email), strings.TrimSpace(r.Email) != ""
	case ChannelTelegram:
		if !r.Prefs.Telegram {
			return "", false
		}
		id := strings.TrimSpace(r.Prefs.TelegramChatID)
		return id, id != ""
	case ChannelDiscord:
		if !r.Prefs.Discord {
			return "", false
		}
		hook := strings.TrimSpace(r.Prefs.DiscordWebhook)
		return hook, isDiscordWebhook(hook)
	}
	return "", false
}

// isDiscordWebhook отсеивает всё, что не похоже на адрес вебхука.
//
// Проверка нужна не ради строгости, а чтобы не отправлять запрос на
// произвольный адрес, введённый в поле: значение приходит от пользователя, а
// запрос уйдёт с нашего сервера.
func isDiscordWebhook(v string) bool {
	return strings.HasPrefix(v, "https://discord.com/api/webhooks/") ||
		strings.HasPrefix(v, "https://discordapp.com/api/webhooks/") ||
		strings.HasPrefix(v, "https://ptb.discord.com/api/webhooks/") ||
		strings.HasPrefix(v, "https://canary.discord.com/api/webhooks/")
}

// channelsFor решает, какими внешними каналами уйдёт событие этому получателю.
func channelsFor(e Event, r Recipient) []Channel {
	def := DefFor(e.Kind)
	out := make([]Channel, 0, len(def.Channels))
	for _, c := range def.Channels {
		if _, ok := r.Target(c); ok {
			out = append(out, c)
		}
	}
	return out
}
