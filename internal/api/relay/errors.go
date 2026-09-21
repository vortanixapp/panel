package relay

import (
	"encoding/json"
	"errors"
	"net/http"
)

type Error struct {
	Status  int
	Code    string
	Message string
}

func (e *Error) Error() string { return "relay: " + e.Message }

func decodeError(resp *http.Response) error {
	var body struct {
		Error string `json:"error"`
		Code  string `json:"code"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&body)
	msg := body.Error
	if msg == "" {
		msg = resp.Status
	}
	return &Error{Status: resp.StatusCode, Code: body.Code, Message: msg}
}

func ErrorCode(err error) string {
	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}
	return ""
}
