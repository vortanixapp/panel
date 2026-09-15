package notify

import (
	"strings"

	"github.com/vortanixapp/panel/pkg/i18n"
)

type Channel string

const (
	ChannelPanel    Channel = "panel"
	ChannelEmail    Channel = "email"
	ChannelTelegram Channel = "telegram"
	ChannelDiscord  Channel = "discord"
)

func ExternalChannels() []Channel {
	return []Channel{ChannelEmail, ChannelTelegram, ChannelDiscord}
}

type Action struct {
	Label i18n.Msg
	Href  string
}

type Event struct {
	Kind Kind

	Title i18n.Msg
	Body  i18n.Msg
	Extra []i18n.Msg

	Action *Action

	Severity Severity

	Meta map[string]any

	DedupeKey string
}

func (e Event) severity() Severity {
	if e.Severity != "" {
		return e.Severity
	}
	return DefFor(e.Kind).Severity
}

type Recipient struct {
	UserID string
	Email  string
	Locale string

	Prefs Prefs
}

type Prefs struct {
	Email    bool
	Telegram bool
	Discord  bool

	TelegramChatID string
	DiscordWebhook string
}

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

func isDiscordWebhook(v string) bool {
	return strings.HasPrefix(v, "https://discord.com/api/webhooks/") ||
		strings.HasPrefix(v, "https://discordapp.com/api/webhooks/") ||
		strings.HasPrefix(v, "https://ptb.discord.com/api/webhooks/") ||
		strings.HasPrefix(v, "https://canary.discord.com/api/webhooks/")
}

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
