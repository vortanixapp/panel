package rcon

import (
	"context"
	"errors"
	"net"
	"regexp"
	"strings"
	"time"
)

var telnetLogLine = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\s`)

func Telnet(ctx context.Context, addr, password, command string, opts Options) (string, error) {
	opts = opts.withDefaults()
	dialer := net.Dialer{Timeout: opts.DialTimeout}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return "", err
	}
	defer conn.Close()
	stop := context.AfterFunc(ctx, func() { _ = conn.SetDeadline(time.Now()) })
	defer stop()

	greeting, err := readIdle(conn, opts.ReplyWait, opts.Settle)
	if err != nil {
		return "", err
	}
	if strings.Contains(strings.ToLower(greeting), "password") {
		if err := writeLine(conn, password, opts.DialTimeout); err != nil {
			return "", err
		}
		answer, err := readIdle(conn, opts.ReplyWait, opts.Settle)
		if err != nil {
			return "", err
		}
		lower := strings.ToLower(answer)
		if strings.Contains(lower, "incorrect") || !strings.Contains(lower, "logon successful") {
			return "", ErrAuth
		}
	}
	if err := writeLine(conn, command, opts.DialTimeout); err != nil {
		return "", err
	}
	out, err := readIdle(conn, opts.ReplyWait, opts.Settle)
	_ = writeLine(conn, "exit", opts.DialTimeout)
	if err != nil {
		return "", err
	}
	var rows []string
	for _, row := range strings.Split(strings.ReplaceAll(out, "\r", ""), "\n") {
		row = strings.TrimRight(row, " ")
		if row == "" || telnetLogLine.MatchString(row) {
			continue
		}
		rows = append(rows, row)
	}
	return strings.Join(rows, "\n"), nil
}

func writeLine(conn net.Conn, text string, wait time.Duration) error {
	_ = conn.SetWriteDeadline(time.Now().Add(wait))
	_, err := conn.Write([]byte(text + "\r\n"))
	return err
}

func readIdle(conn net.Conn, first, settle time.Duration) (string, error) {
	var out []byte
	buf := make([]byte, 4096)
	_ = conn.SetReadDeadline(time.Now().Add(first))
	for {
		n, err := conn.Read(buf)
		if n > 0 {
			out = append(out, buf[:n]...)
			_ = conn.SetReadDeadline(time.Now().Add(settle))
		}
		if err != nil {
			var netErr net.Error
			if errors.As(err, &netErr) && netErr.Timeout() {
				return stripTelnet(out), nil
			}
			if len(out) > 0 {
				return stripTelnet(out), nil
			}
			return "", err
		}
	}
}

func stripTelnet(data []byte) string {
	out := make([]byte, 0, len(data))
	for i := 0; i < len(data); i++ {
		if data[i] == 0xff && i+2 < len(data) {
			i += 2
			continue
		}
		out = append(out, data[i])
	}
	return string(out)
}
