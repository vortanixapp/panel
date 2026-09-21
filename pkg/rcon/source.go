package rcon

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

var ErrAuth = errors.New("rcon: неверный пароль")

const (
	packetAuth         = 3
	packetAuthResponse = 2
	packetExec         = 2
	packetResponse     = 0

	idAuth    = 1
	idCommand = 2
	idFence   = 3

	maxPacketSize = 1 << 20
)

type Options struct {
	DialTimeout time.Duration
	ReplyWait   time.Duration
	Settle      time.Duration
}

func (o Options) withDefaults() Options {
	if o.DialTimeout <= 0 {
		o.DialTimeout = 3 * time.Second
	}
	if o.ReplyWait <= 0 {
		o.ReplyWait = 5 * time.Second
	}
	if o.Settle <= 0 {
		o.Settle = 300 * time.Millisecond
	}
	return o
}

func Source(ctx context.Context, addr, password, command string, opts Options) (string, error) {
	opts = opts.withDefaults()
	dialer := net.Dialer{Timeout: opts.DialTimeout}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return "", err
	}
	defer conn.Close()
	stop := context.AfterFunc(ctx, func() { _ = conn.SetDeadline(time.Now()) })
	defer stop()

	_ = conn.SetDeadline(time.Now().Add(opts.ReplyWait))
	if err := writePacket(conn, idAuth, packetAuth, password); err != nil {
		return "", err
	}
	for {
		id, kind, _, err := readPacket(conn)
		if err != nil {
			return "", fmt.Errorf("rcon: нет ответа на вход: %w", err)
		}
		if kind != packetAuthResponse {
			continue
		}
		if id == -1 {
			return "", ErrAuth
		}
		break
	}

	_ = conn.SetDeadline(time.Now().Add(opts.ReplyWait))
	if err := writePacket(conn, idCommand, packetExec, command); err != nil {
		return "", err
	}
	if err := writePacket(conn, idFence, packetResponse, ""); err != nil {
		return "", err
	}

	var out strings.Builder
	for {
		id, _, body, err := readPacket(conn)
		if err != nil {
			var netErr net.Error
			if (errors.As(err, &netErr) && netErr.Timeout()) || errors.Is(err, io.EOF) {
				return clean(out.String()), nil
			}
			if out.Len() > 0 {
				return clean(out.String()), nil
			}
			return "", err
		}
		if id == idFence {
			return clean(out.String()), nil
		}
		if id == idCommand {
			out.WriteString(body)
			_ = conn.SetDeadline(time.Now().Add(opts.Settle))
		}
	}
}

func writePacket(w io.Writer, id int32, kind int32, body string) error {
	size := int32(len(body) + 10)
	buf := make([]byte, 0, size+4)
	buf = binary.LittleEndian.AppendUint32(buf, uint32(size))
	buf = binary.LittleEndian.AppendUint32(buf, uint32(id))
	buf = binary.LittleEndian.AppendUint32(buf, uint32(kind))
	buf = append(buf, body...)
	buf = append(buf, 0, 0)
	_, err := w.Write(buf)
	return err
}

func readPacket(r io.Reader) (int32, int32, string, error) {
	var head [4]byte
	if _, err := io.ReadFull(r, head[:]); err != nil {
		return 0, 0, "", err
	}
	size := int32(binary.LittleEndian.Uint32(head[:]))
	if size < 10 || size > maxPacketSize {
		return 0, 0, "", fmt.Errorf("rcon: пакет неверного размера %d", size)
	}
	data := make([]byte, size)
	if _, err := io.ReadFull(r, data); err != nil {
		return 0, 0, "", err
	}
	id := int32(binary.LittleEndian.Uint32(data[0:4]))
	kind := int32(binary.LittleEndian.Uint32(data[4:8]))
	body := strings.TrimRight(string(data[8:]), "\x00")
	return id, kind, body, nil
}

func clean(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	skip := false
	for _, r := range s {
		if skip {
			skip = false
			continue
		}
		if r == '§' {
			skip = true
			continue
		}
		b.WriteRune(r)
	}
	return strings.TrimRight(strings.ReplaceAll(b.String(), "\r\n", "\n"), "\n ")
}
