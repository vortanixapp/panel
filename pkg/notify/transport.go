package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/smtp"
	"net/url"
	"strings"
	"time"
)

// Config — настройки доставки.
type Config struct {
	SMTPHost string
	SMTPPort string
	SMTPUser string
	SMTPPass string
	MailFrom string

	// TelegramBotToken — общий бот панели. Без него канал Telegram недоступен
	// целиком, о чём и надо сказать вслух, а не притворяться, что доставка идёт.
	TelegramBotToken string

	// HTTPClient используется для Telegram и Discord. Пусто — берётся клиент с
	// разумным таймаутом: без него зависший вебхук держал бы обработчик очереди.
	HTTPClient *http.Client
}

func (c Config) client() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return &http.Client{Timeout: 15 * time.Second}
}

// ErrChannelUnavailable — канал не настроен на этой установке.
//
// Отдельный тип ошибки нужен, чтобы отличить «некуда отправлять» от «не удалось
// отправить». Первое повторять бессмысленно: пока владелец не задаст SMTP или
// токен бота, следующая попытка провалится так же. Второе — сеть, лимит, —
// пройдёт само.
var ErrChannelUnavailable = errors.New("канал доставки не настроен")

// Delivery — одна запись очереди, готовая к отправке.
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

// Send отправляет одну запись выбранным каналом.
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

// sendEmail отправляет письмо.
//
// При ненастроенном SMTP возвращается ошибка, а НЕ успех. Прежний мейлер
// биллинга в этом случае писал строку в лог и возвращал nil, после чего письмо
// помечалось отправленным: на стандартной выкатке, где SMTP_HOST пуст, вся
// почта исчезала бесследно, а очередь выглядела разобранной.
func sendEmail(ctx context.Context, cfg Config, d Delivery) error {
	host := strings.TrimSpace(cfg.SMTPHost)
	if host == "" {
		return fmt.Errorf("%w: не задан SMTP_HOST", ErrChannelUnavailable)
	}
	from := strings.TrimSpace(cfg.MailFrom)
	if from == "" {
		from = "no-reply@" + host
	}
	port := strings.TrimSpace(cfg.SMTPPort)
	if port == "" {
		port = "587"
	}

	var msg bytes.Buffer
	msg.WriteString("From: " + from + "\r\n")
	msg.WriteString("To: " + d.Target + "\r\n")
	// Тема кодируется по RFC 2047: без этого кириллица в теме приезжает
	// вопросительными знаками у части почтовых клиентов.
	msg.WriteString("Subject: " + mime.QEncoding.Encode("utf-8", d.Subject) + "\r\n")
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	msg.WriteString("\r\n")
	msg.WriteString(d.Body)

	// Аутентификация только при непустом пользователе: net/smtp отказывается
	// передавать PLAIN по незашифрованному соединению, и безусловный PlainAuth
	// ломает работу с релеем без авторизации — ровно это делают почтовые модули
	// core-api и воркера.
	var auth smtp.Auth
	if u := strings.TrimSpace(cfg.SMTPUser); u != "" {
		auth = smtp.PlainAuth("", u, cfg.SMTPPass, host)
	}

	done := make(chan error, 1)
	go func() {
		done <- smtp.SendMail(host+":"+port, auth, from, []string{d.Target}, msg.Bytes())
	}()
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func sendTelegram(ctx context.Context, cfg Config, d Delivery) error {
	token := strings.TrimSpace(cfg.TelegramBotToken)
	if token == "" {
		return fmt.Errorf("%w: не задан токен бота Telegram", ErrChannelUnavailable)
	}

	text := severityPrefix(DefFor(d.Kind).Severity) + d.Subject
	if d.Body != "" {
		text += "\n\n" + d.Body
	}

	form := url.Values{}
	form.Set("chat_id", d.Target)
	form.Set("text", text)
	form.Set("disable_web_page_preview", "true")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.telegram.org/bot"+token+"/sendMessage",
		strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return doAndCheck(cfg.client(), req, "Telegram")
}

func sendDiscord(ctx context.Context, cfg Config, d Delivery) error {
	if !isDiscordWebhook(d.Target) {
		return fmt.Errorf("%w: адрес не похож на вебхук Discord", ErrChannelUnavailable)
	}

	content := severityPrefix(DefFor(d.Kind).Severity) + "**" + d.Subject + "**"
	if d.Body != "" {
		content += "\n" + d.Body
	}
	payload, err := json.Marshal(map[string]any{"content": content})
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

// doAndCheck выполняет запрос и превращает ответ в понятную ошибку.
func doAndCheck(client *http.Client, req *http.Request, who string) error {
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("%s: %w", who, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	// Текст ответа нужен в last_error: без него «ошибка 400» не говорит ничего,
	// а Telegram и Discord объясняют причину именно телом.
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	return fmt.Errorf("%s ответил %d: %s", who, resp.StatusCode,
		strings.TrimSpace(string(body)))
}
