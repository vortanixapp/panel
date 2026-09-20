package notify

import (
	"encoding/json"
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

func KnownChannel(c Channel) bool {
	return hasChannel(ExternalChannels(), c)
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

	Meta map[string]any

	DedupeKey string
}

type Recipient struct {
	UserID string
	Email  string
	Locale string

	Prefs Prefs
	Quiet Quiet

	StatusEmail bool
}

type Prefs struct {
	Email    bool
	Telegram bool
	Discord  bool

	TelegramChatID string
	DiscordWebhook string

	Routes Routes
}

type Routes map[Group]map[Channel]bool

func (r Routes) Allows(g Group, c Channel) bool {
	if on, ok := r[g][c]; ok {
		return on
	}
	return true
}

func (r Routes) Set(g Group, c Channel, on bool) Routes {
	if !KnownGroup(g) || !KnownChannel(c) {
		return r
	}
	out := r.clone()
	if on || GroupLocked(g, c) {
		delete(out[g], c)
		if len(out[g]) == 0 {
			delete(out, g)
		}
		return out
	}
	if out[g] == nil {
		out[g] = map[Channel]bool{}
	}
	out[g][c] = false
	return out
}

func (r Routes) clone() Routes {
	out := Routes{}
	for g, cells := range r {
		for c, on := range cells {
			if out[g] == nil {
				out[g] = map[Channel]bool{}
			}
			out[g][c] = on
		}
	}
	return out
}

func ParseRoutes(raw []byte) Routes {
	var stored map[string]map[string]bool
	if len(raw) == 0 || json.Unmarshal(raw, &stored) != nil {
		return Routes{}
	}
	out := Routes{}
	for g, cells := range stored {
		for c, on := range cells {
			if !on {
				out = out.Set(Group(g), Channel(c), false)
			}
		}
	}
	return out
}

func (r Routes) JSON() []byte {
	b, err := json.Marshal(r)
	if err != nil {
		return []byte("{}")
	}
	return b
}

func (r Recipient) Target(c Channel) (string, bool) {
	switch c {
	case ChannelEmail:
		if !r.Prefs.Email {
			return "", false
		}
	case ChannelTelegram:
		if !r.Prefs.Telegram {
			return "", false
		}
	case ChannelDiscord:
		if !r.Prefs.Discord {
			return "", false
		}
	default:
		return "", false
	}
	return r.address(c)
}

func (r Recipient) address(c Channel) (string, bool) {
	switch c {
	case ChannelEmail:
		v := strings.TrimSpace(r.Email)
		return v, v != ""
	case ChannelTelegram:
		v := strings.TrimSpace(r.Prefs.TelegramChatID)
		return v, v != ""
	case ChannelDiscord:
		v := strings.TrimSpace(r.Prefs.DiscordWebhook)
		return v, IsDiscordWebhook(v)
	}
	return "", false
}

func IsDiscordWebhook(v string) bool {
	return strings.HasPrefix(v, "https://discord.com/api/webhooks/") ||
		strings.HasPrefix(v, "https://discordapp.com/api/webhooks/") ||
		strings.HasPrefix(v, "https://ptb.discord.com/api/webhooks/") ||
		strings.HasPrefix(v, "https://canary.discord.com/api/webhooks/")
}

type route struct {
	channel Channel
	target  string
}

func statusEmailKind(kind Kind) bool {
	return kind == KindServerReady || kind == KindServerFailed
}

func routesFor(e Event, r Recipient) []route {
	def := DefFor(e.Kind)
	out := make([]route, 0, len(def.Channels))
	for _, c := range def.Channels {
		if c == ChannelEmail && !r.StatusEmail && statusEmailKind(e.Kind) {
			continue
		}
		if c == ChannelEmail && def.Required {
			if target, ok := r.address(c); ok {
				out = append(out, route{channel: c, target: target})
			}
			continue
		}
		if !r.Prefs.Routes.Allows(def.Group, c) {
			continue
		}
		if target, ok := r.Target(c); ok {
			out = append(out, route{channel: c, target: target})
		}
	}
	return out
}
