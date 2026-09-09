package docker

import (
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

var raknetMagic = []byte{
	0x00, 0xFF, 0xFF, 0x00, 0xFE, 0xFE, 0xFE, 0xFE,
	0xFD, 0xFD, 0xFD, 0xFD, 0x12, 0x34, 0x56, 0x78,
}

type bedrockInfo struct {
	Players    int
	MaxPlayers int
	LevelName  string
	Hostname   string
}

func queryBedrock(host string, port int) (*bedrockInfo, error) {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return nil, err
	}
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(1500 * time.Millisecond))

	pkt := make([]byte, 0, 33)
	pkt = append(pkt, 0x01)
	ts := make([]byte, 8)
	binary.BigEndian.PutUint64(ts, uint64(time.Now().UnixMilli()))
	pkt = append(pkt, ts...)
	pkt = append(pkt, raknetMagic...)
	guid := make([]byte, 8)
	binary.BigEndian.PutUint64(guid, 0x564F5254414E4958)
	pkt = append(pkt, guid...)

	if _, err := conn.Write(pkt); err != nil {
		return nil, err
	}
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		return nil, err
	}
	data := buf[:n]
	if n < 35 || data[0] != 0x1C {
		return nil, fmt.Errorf("не ответ RakNet")
	}
	strLen := int(binary.BigEndian.Uint16(data[33:35]))
	if strLen <= 0 || 35+strLen > n {
		strLen = n - 35
	}
	return parseBedrockMOTD(string(data[35 : 35+strLen])), nil
}

func parseBedrockMOTD(motd string) *bedrockInfo {
	parts := strings.Split(motd, ";")
	at := func(i int) string {
		if i < len(parts) {
			return strings.TrimSpace(parts[i])
		}
		return ""
	}
	num := func(i int) int {
		n, err := strconv.Atoi(at(i))
		if err != nil {
			return 0
		}
		return n
	}
	return &bedrockInfo{
		Hostname:   at(1),
		Players:    num(4),
		MaxPlayers: num(5),
		LevelName:  at(7),
	}
}
