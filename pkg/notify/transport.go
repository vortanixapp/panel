package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/vortanixapp/panel/pkg/mailer"
	"github.com/vortanixapp/panel/pkg/mailtpl"
)

type Config struct {
	Mail  mailer.Config
	Brand mailtpl.Brand

	TelegramBotToken string

	HTTPClient *http.Client
}

func (c Config) client() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return &http.Client{Timeout: 15 * time.Second}
}

var ErrChannelUnavailable = errors.New("канал доставки не настроен")

type Delivery struct {
	ID          string
	Kind        Kind
	Channel     Channel
	Target      string
	Subject     string
	Body        string
	ActionLabel string
	ActionHref  string
}

func Send(ctx context.Context, cfg Config, d Delivery) error {
	switch d.Channel {
	case ChannelEmail:
		return sendEmail(ctx, cfg, d)
	case ChannelTelegram:
		return sendTelegram(ctx, cfg, d)
	case ChannelDiscord:
		return sendDiscord(ctx, cfg, d)
	}
	return fmt.Errorf("неизвестный канал доставки %q", d.Channel)
}

func sendEmail(ctx context.Context, cfg Config, d Delivery) error {
	if !cfg.Mail.Configured() {
		return fmt.Errorf("%w: не задан адрес SMTP-сервера", ErrChannelUnavailable)
	}
	if cfg.Mail.Silent() {
		return fmt.Errorf("%w: выбран режим %q, письма не отправляются", ErrChannelUnavailable, cfg.Mail.MailerName())
	}
	body := mailtpl.Message{
		Title:       d.Subject,
		Body:        mailtpl.Paragraphs(d.Body),
		ActionLabel: d.ActionLabel,
		ActionURL:   d.ActionHref,
	}
	return cfg.Mail.Send(ctx, mailer.Message{
		To:      d.Target,
		Subject: d.Subject,
		HTML:    mailtpl.Render(cfg.Brand, body),
		Text:    mailtpl.PlainText(cfg.Brand, body),
	})
}

func sendTelegram(ctx context.Context, cfg Config, d Delivery) error {
	token := strings.TrimSpace(cfg.TelegramBotToken)
	if token == "" {
		return fmt.Errorf("%w: не задан токен бота Telegram", ErrChannelUnavailable)
	}

	text := "<b>" + html.EscapeString(severityPrefix(DefFor(d.Kind).Severity)+d.Subject) + "</b>"
	if body := strings.TrimSpace(d.Body); body != "" {
		text += "\n\n" + html.EscapeString(truncateRunes(body, 3500))
	}

	form := url.Values{}
	form.Set("chat_id", d.Target)
	form.Set("parse_mode", "HTML")
	form.Set("disable_web_page_preview", "true")

	if d.ActionLabel != "" && d.ActionHref != "" {
		if strings.HasPrefix(d.ActionHref, "https://") {
			markup, err := json.Marshal(map[string]any{
				"inline_keyboard": [][]map[string]string{{{"text": d.ActionLabel, "url": d.ActionHref}}},
			})
			if err != nil {
				return err
			}
			form.Set("reply_markup", string(markup))
		} else {
			text += "\n\n" + html.EscapeString(d.ActionLabel+": "+d.ActionHref)
		}
	}
	form.Set("text", text)

	return telegramCall(ctx, cfg.client(), token, "sendMessage", form, nil)
}

func sendDiscord(ctx context.Context, cfg Config, d Delivery) error {
	if !IsDiscordWebhook(d.Target) {
		return fmt.Errorf("%w: адрес не похож на вебхук Discord", ErrChannelUnavailable)
	}

	severity := DefFor(d.Kind).Severity
	embed := map[string]any{
		"title": truncateRunes(severityPrefix(severity)+d.Subject, 256),
		"color": severityColor(severity),
	}
	description := strings.TrimSpace(d.Body)
	if d.ActionLabel != "" && d.ActionHref != "" {
		if strings.HasPrefix(d.ActionHref, "https://") || strings.HasPrefix(d.ActionHref, "http://") {
			embed["url"] = d.ActionHref
			link := "[" + d.ActionLabel + "](" + d.ActionHref + ")"
			if description != "" {
				description += "\n\n"
			}
			description += link
		}
	}
	if description != "" {
		embed["description"] = truncateRunes(description, 4000)
	}

	payload, err := json.Marshal(map[string]any{"embeds": []any{embed}})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, d.Target, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	return doAndCheck(cfg.client(), req, "Discord")
}

func doAndCheck(client *http.Client, req *http.Request, who string) error {
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("%s: %w", who, withoutURL(err))
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	return fmt.Errorf("%s ответил %d: %s", who, resp.StatusCode,
		strings.TrimSpace(string(body)))
}
