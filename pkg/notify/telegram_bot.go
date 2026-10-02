package notify

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	telegramMessageLimit = 4000
	telegramCallbackMax  = 64
)

type TelegramUser struct {
	ID           int64  `json:"id"`
	IsBot        bool   `json:"is_bot"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	Username     string `json:"username"`
	LanguageCode string `json:"language_code"`
}

type TelegramChatInfo struct {
	ID        int64  `json:"id"`
	Type      string `json:"type"`
	Title     string `json:"title"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

func (c TelegramChatInfo) Private() bool {
	return c.Type == "private"
}

func (c TelegramChatInfo) DisplayName() string {
	return telegramChatTitle(c.Title, c.Username, c.FirstName, c.LastName)
}

type TelegramMessage struct {
	MessageID int64            `json:"message_id"`
	From      *TelegramUser    `json:"from"`
	Chat      TelegramChatInfo `json:"chat"`
	Date      int64            `json:"date"`
	Text      string           `json:"text"`
}

type TelegramCallback struct {
	ID      string           `json:"id"`
	From    TelegramUser     `json:"from"`
	Message *TelegramMessage `json:"message"`
	Data    string           `json:"data"`
}

type TelegramUpdate struct {
	UpdateID      int64             `json:"update_id"`
	Message       *TelegramMessage  `json:"message"`
	CallbackQuery *TelegramCallback `json:"callback_query"`
}

type TelegramButton struct {
	Text string
	Data string
	URL  string
}

type TelegramKeyboard [][]TelegramButton

type TelegramCommand struct {
	Command     string `json:"command"`
	Description string `json:"description"`
}

type TelegramBot struct {
	Token  string
	Client *http.Client
}

func NewTelegramBot(token string) TelegramBot {
	return TelegramBot{Token: strings.TrimSpace(token), Client: &http.Client{Timeout: 45 * time.Second}}
}

func (b TelegramBot) client() *http.Client {
	if b.Client != nil {
		return b.Client
	}
	return &http.Client{Timeout: 45 * time.Second}
}

func (b TelegramBot) call(ctx context.Context, method string, form url.Values, result any) error {
	return telegramCall(ctx, b.client(), b.Token, method, form, result)
}

func (k TelegramKeyboard) markup() (string, bool) {
	rows := make([][]map[string]string, 0, len(k))
	for _, row := range k {
		cells := make([]map[string]string, 0, len(row))
		for _, btn := range row {
			if btn.Text == "" {
				continue
			}
			switch {
			case btn.URL != "":
				cells = append(cells, map[string]string{"text": btn.Text, "url": btn.URL})
			case btn.Data != "" && len(btn.Data) <= telegramCallbackMax:
				cells = append(cells, map[string]string{"text": btn.Text, "callback_data": btn.Data})
			}
		}
		if len(cells) > 0 {
			rows = append(rows, cells)
		}
	}
	if len(rows) == 0 {
		return "", false
	}
	raw, err := json.Marshal(map[string]any{"inline_keyboard": rows})
	if err != nil {
		return "", false
	}
	return string(raw), true
}

func (b TelegramBot) GetUpdates(ctx context.Context, offset int64, timeoutSeconds int) ([]TelegramUpdate, error) {
	form := url.Values{}
	form.Set("timeout", strconv.Itoa(timeoutSeconds))
	form.Set("limit", "50")
	form.Set("allowed_updates", `["message","callback_query"]`)
	if offset > 0 {
		form.Set("offset", strconv.FormatInt(offset, 10))
	}
	var updates []TelegramUpdate
	if err := b.call(ctx, "getUpdates", form, &updates); err != nil {
		return nil, err
	}
	return updates, nil
}

func (b TelegramBot) SendMessage(ctx context.Context, chatID int64, text string, kb TelegramKeyboard) (int64, error) {
	form := url.Values{}
	form.Set("chat_id", strconv.FormatInt(chatID, 10))
	form.Set("parse_mode", "HTML")
	form.Set("disable_web_page_preview", "true")
	form.Set("text", truncateRunes(text, telegramMessageLimit))
	if markup, ok := kb.markup(); ok {
		form.Set("reply_markup", markup)
	}
	var sent struct {
		MessageID int64 `json:"message_id"`
	}
	if err := b.call(ctx, "sendMessage", form, &sent); err != nil {
		return 0, err
	}
	return sent.MessageID, nil
}

func (b TelegramBot) EditMessage(ctx context.Context, chatID, messageID int64, text string, kb TelegramKeyboard) error {
	form := url.Values{}
	form.Set("chat_id", strconv.FormatInt(chatID, 10))
	form.Set("message_id", strconv.FormatInt(messageID, 10))
	form.Set("parse_mode", "HTML")
	form.Set("disable_web_page_preview", "true")
	form.Set("text", truncateRunes(text, telegramMessageLimit))
	if markup, ok := kb.markup(); ok {
		form.Set("reply_markup", markup)
	} else {
		form.Set("reply_markup", `{"inline_keyboard":[]}`)
	}
	err := b.call(ctx, "editMessageText", form, nil)
	if IsTelegramNotModified(err) {
		return nil
	}
	return err
}

func (b TelegramBot) AnswerCallback(ctx context.Context, id, text string, alert bool) error {
	form := url.Values{}
	form.Set("callback_query_id", id)
	if text != "" {
		form.Set("text", truncateRunes(text, 190))
	}
	if alert {
		form.Set("show_alert", "true")
	}
	return b.call(ctx, "answerCallbackQuery", form, nil)
}

func (b TelegramBot) SendChatAction(ctx context.Context, chatID int64, action string) {
	form := url.Values{}
	form.Set("chat_id", strconv.FormatInt(chatID, 10))
	form.Set("action", action)
	_ = b.call(ctx, "sendChatAction", form, nil)
}

func (b TelegramBot) SetCommands(ctx context.Context, language string, commands []TelegramCommand) error {
	raw, err := json.Marshal(commands)
	if err != nil {
		return err
	}
	form := url.Values{}
	form.Set("commands", string(raw))
	form.Set("scope", `{"type":"all_private_chats"}`)
	if language != "" {
		form.Set("language_code", language)
	}
	return b.call(ctx, "setMyCommands", form, nil)
}

func (b TelegramBot) DeleteWebhook(ctx context.Context) error {
	return b.call(ctx, "deleteWebhook", url.Values{}, nil)
}

func IsTelegramNotModified(err error) bool {
	var te *TelegramError
	return errors.As(err, &te) && strings.Contains(strings.ToLower(te.Description), "message is not modified")
}

func IsTelegramConflict(err error) bool {
	var te *TelegramError
	return errors.As(err, &te) && te.Code == http.StatusConflict
}

func IsTelegramWebhookConflict(err error) bool {
	var te *TelegramError
	return errors.As(err, &te) && te.Code == http.StatusConflict &&
		strings.Contains(strings.ToLower(te.Description), "webhook")
}

func IsTelegramUnauthorized(err error) bool {
	var te *TelegramError
	return errors.As(err, &te) && (te.Code == http.StatusUnauthorized || te.Code == http.StatusNotFound)
}

func IsTelegramBlocked(err error) bool {
	var te *TelegramError
	return errors.As(err, &te) && te.Code == http.StatusForbidden
}
