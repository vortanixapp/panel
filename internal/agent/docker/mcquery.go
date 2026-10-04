package docker

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
	"time"
)

type mcQueryInfo struct {
	Players    []string
	NumPlayers int
	MaxPlayers int
	Map        string
}

var mcQuerySession = []byte{0x01, 0x02, 0x03, 0x04}

func queryMinecraftJava(host string, port int) (*mcQueryInfo, error) {
	addr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(host, strconv.Itoa(port)))
	if err != nil {
		return nil, err
	}
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(2 * time.Second))

	handshake := append([]byte{0xFE, 0xFD, 0x09}, mcQuerySession...)
	if _, err := conn.Write(handshake); err != nil {
		return nil, err
	}
	buf := make([]byte, 8192)
	n, err := conn.Read(buf)
	if err != nil {
		return nil, err
	}
	if n < 6 || buf[0] != 0x09 {
		return nil, fmt.Errorf("не ответ Minecraft Query")
	}
	token, err := strconv.ParseInt(string(bytes.TrimRight(buf[5:n], "\x00")), 10, 32)
	if err != nil {
		return nil, fmt.Errorf("токен Minecraft Query: %w", err)
	}

	stat := append([]byte{0xFE, 0xFD, 0x00}, mcQuerySession...)
	challenge := make([]byte, 4)
	binary.BigEndian.PutUint32(challenge, uint32(int32(token)))
	stat = append(stat, challenge...)
	stat = append(stat, 0x00, 0x00, 0x00, 0x00)
	if _, err := conn.Write(stat); err != nil {
		return nil, err
	}
	n, err = conn.Read(buf)
	if err != nil {
		return nil, err
	}
	return parseMcQueryFull(buf[:n])
}

func parseMcQueryFull(data []byte) (*mcQueryInfo, error) {
	if len(data) < 16 || data[0] != 0x00 {
		return nil, fmt.Errorf("не ответ Minecraft Query")
	}
	body := data[5:]
	const kvPadding = 11
	if len(body) < kvPadding {
		return nil, fmt.Errorf("короткий ответ Minecraft Query")
	}
	body = body[kvPadding:]

	fields := map[string]string{}
	for {
		key, rest, ok := cutNul(body)
		if !ok {
			return nil, fmt.Errorf("обрыв ответа Minecraft Query")
		}
		body = rest
		if key == "" {
			break
		}
		value, rest, ok := cutNul(body)
		if !ok {
			return nil, fmt.Errorf("обрыв ответа Minecraft Query")
		}
		body = rest
		fields[key] = value
	}

	info := &mcQueryInfo{Map: fields["map"], Players: []string{}}
	info.NumPlayers, _ = strconv.Atoi(fields["numplayers"])
	info.MaxPlayers, _ = strconv.Atoi(fields["maxplayers"])

	const playerPadding = 10
	if len(body) >= playerPadding {
		body = body[playerPadding:]
		for {
			name, rest, ok := cutNul(body)
			if !ok || name == "" {
				break
			}
			info.Players = append(info.Players, name)
			body = rest
		}
	}
	return info, nil
}

func cutNul(b []byte) (string, []byte, bool) {
	i := bytes.IndexByte(b, 0)
	if i < 0 {
		return "", nil, false
	}
	return string(b[:i]), b[i+1:], true
}
