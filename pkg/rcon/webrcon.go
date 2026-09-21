package rcon

import (
	"context"
	"errors"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

type webFrame struct {
	Identifier int    `json:"Identifier"`
	Message    string `json:"Message"`
	Name       string `json:"Name,omitempty"`
	Type       string `json:"Type,omitempty"`
}

const webIdentifier = 41007

func WebRcon(ctx context.Context, addr, password, command string, opts Options) (string, error) {
	opts = opts.withDefaults()
	dialer := websocket.Dialer{HandshakeTimeout: opts.DialTimeout}
	target := url.URL{Scheme: "ws", Host: addr, Path: "/" + password}
	conn, resp, err := dialer.DialContext(ctx, target.String(), nil)
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
	if err != nil {
		if resp != nil && (resp.StatusCode == 401 || resp.StatusCode == 403) {
			return "", ErrAuth
		}
		return "", err
	}
	defer conn.Close()
	stop := context.AfterFunc(ctx, func() { _ = conn.SetReadDeadline(time.Now()) })
	defer stop()

	_ = conn.SetWriteDeadline(time.Now().Add(opts.DialTimeout))
	if err := conn.WriteJSON(webFrame{Identifier: webIdentifier, Message: command, Name: "WebRcon"}); err != nil {
		return "", err
	}
	_ = conn.SetReadDeadline(time.Now().Add(opts.ReplyWait))
	for {
		var frame webFrame
		if err := conn.ReadJSON(&frame); err != nil {
			var netErr net.Error
			if errors.As(err, &netErr) && netErr.Timeout() {
				return "", nil
			}
			return "", err
		}
		if frame.Identifier == webIdentifier {
			return strings.TrimRight(frame.Message, "\r\n "), nil
		}
	}
}
