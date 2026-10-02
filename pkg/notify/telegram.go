package notify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

var telegramAPI = "https://api.telegram.org"

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

func telegramChatTitle(title, username, first, last string) string {
	if t := strings.TrimSpace(title); t != "" {
		return t
	}
	if u := strings.TrimSpace(username); u != "" {
		return "@" + u
	}
	return strings.TrimSpace(first + " " + last)
}
