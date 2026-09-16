package notify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var telegramAPI = "https://api.telegram.org"

var ErrTelegramWebhook = errors.New("у бота настроен вебхук, сообщения ему можно прочитать только через него")

const telegramLinkWindow = 15 * time.Minute

type TelegramError struct {
	Code        int
	Description string
}

func (e *TelegramError) Error() string {
	if e.Description != "" {
		return "Telegram: " + e.Description
	}
	return fmt.Sprintf("Telegram ответил %d", e.Code)
}

type TelegramChat struct {
	ID    string
	Title string
}

func telegramCall(ctx context.Context, client *http.Client, token, method string, form url.Values, result any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		telegramAPI+"/bot"+token+"/"+method, strings.NewReader(form.Encode()))
	if err != nil {
		return errors.New("Telegram: не удалось собрать запрос")
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("Telegram: %w", withoutURL(err))
	}
	defer resp.Body.Close()

	var parsed struct {
		OK          bool            `json:"ok"`
		ErrorCode   int             `json:"error_code"`
		Description string          `json:"description"`
		Result      json.RawMessage `json:"result"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&parsed); err != nil {
		return &TelegramError{Code: resp.StatusCode}
	}
	if !parsed.OK {
		code := parsed.ErrorCode
		if code == 0 {
			code = resp.StatusCode
		}
		return &TelegramError{Code: code, Description: parsed.Description}
	}
	if result != nil && len(parsed.Result) > 0 {
		return json.Unmarshal(parsed.Result, result)
	}
	return nil
}

func withoutURL(err error) error {
	var ue *url.Error
	if errors.As(err, &ue) && ue.Err != nil {
		return ue.Err
	}
	return err
}

func TelegramBotUsername(ctx context.Context, client *http.Client, token string) (string, error) {
	var me struct {
		Username string `json:"username"`
	}
	if err := telegramCall(ctx, client, token, "getMe", url.Values{}, &me); err != nil {
		return "", err
	}
	return me.Username, nil
}

type telegramUpdate struct {
	UpdateID int64 `json:"update_id"`
	Message  *struct {
		Date int64  `json:"date"`
		Text string `json:"text"`
		Chat struct {
			ID        int64  `json:"id"`
			Title     string `json:"title"`
			Username  string `json:"username"`
			FirstName string `json:"first_name"`
			LastName  string `json:"last_name"`
		} `json:"chat"`
	} `json:"message"`
}

func TelegramFindStart(ctx context.Context, client *http.Client, token, code string, now time.Time) (TelegramChat, bool, error) {
	form := url.Values{}
	form.Set("timeout", "0")
	form.Set("limit", "100")
	form.Set("allowed_updates", `["message"]`)

	var updates []telegramUpdate
	if err := telegramCall(ctx, client, token, "getUpdates", form, &updates); err != nil {
		var te *TelegramError
		if errors.As(err, &te) && te.Code == http.StatusConflict {
			if strings.Contains(strings.ToLower(te.Description), "webhook") {
				return TelegramChat{}, false, ErrTelegramWebhook
			}
			return TelegramChat{}, false, nil
		}
		return TelegramChat{}, false, err
	}

	var chat TelegramChat
	found := false
	var stale int64
	leading := true
	for _, u := range updates {
		old := u.Message == nil || now.Sub(time.Unix(u.Message.Date, 0)) > telegramLinkWindow
		if leading && old {
			stale = u.UpdateID
		} else {
			leading = false
		}
		if found || u.Message == nil || !isStartWithCode(u.Message.Text, code) {
			continue
		}
		m := u.Message
		chat = TelegramChat{
			ID:    strconv.FormatInt(m.Chat.ID, 10),
			Title: telegramChatTitle(m.Chat.Title, m.Chat.Username, m.Chat.FirstName, m.Chat.LastName),
		}
		found = true
	}

	if stale > 0 {
		confirm := url.Values{}
		confirm.Set("offset", strconv.FormatInt(stale+1, 10))
		confirm.Set("limit", "1")
		confirm.Set("timeout", "0")
		confirm.Set("allowed_updates", `["message"]`)
		_ = telegramCall(ctx, client, token, "getUpdates", confirm, nil)
	}
	return chat, found, nil
}

func isStartWithCode(text, code string) bool {
	fields := strings.Fields(text)
	if len(fields) != 2 || code == "" || fields[1] != code {
		return false
	}
	return fields[0] == "/start" || strings.HasPrefix(fields[0], "/start@")
}

func telegramChatTitle(title, username, first, last string) string {
	if t := strings.TrimSpace(title); t != "" {
		return t
	}
	if u := strings.TrimSpace(username); u != "" {
		return "@" + u
	}
	return strings.TrimSpace(first + " " + last)
}
