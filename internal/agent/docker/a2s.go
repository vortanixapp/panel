package docker

import (
	"encoding/binary"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/vortanixapp/panel/pkg/gamecatalog"
)

func isSourceGame(game string) bool {
	return gamecatalog.QueryProtocol(normalizeGame(game)) == gamecatalog.QueryA2S
}

func isSampGame(game string) bool {
	return gamecatalog.QueryProtocol(normalizeGame(game)) == gamecatalog.QuerySAMP
}

func isMcJavaGame(game string) bool {
	return gamecatalog.QueryProtocol(normalizeGame(game)) == gamecatalog.QueryMC &&
		strings.HasSuffix(gamecatalog.Repository(normalizeGame(game)), "/mcjava")
}

func isMcBedrockGame(game string) bool {
	key := normalizeGame(game)
	return gamecatalog.QueryProtocol(key) == gamecatalog.QueryMC &&
		strings.HasSuffix(gamecatalog.Repository(key), "/mcbedrock")
}

func readCString(buf []byte, offset int) (string, int) {
	end := -1
	for i := offset; i < len(buf); i++ {
		if buf[i] == 0 {
			end = i
			break
		}
	}
	if end < 0 {
		return "", len(buf)
	}
	return string(buf[offset:end]), end + 1
}

func recvA2SPacket(host string, port int, timeout time.Duration) ([]byte, error) {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return nil, err
	}
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(timeout))

	basePkt := []byte{0xFF, 0xFF, 0xFF, 0xFF, 'T', 'S', 'o', 'u', 'r', 'c', 'e', ' ', 'E', 'n', 'g', 'i', 'n', 'e', ' ', 'Q', 'u', 'e', 'r', 'y', 0}
	if _, err := conn.Write(basePkt); err != nil {
		return nil, err
	}
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		return nil, err
	}
	data := buf[:n]
	if len(data) >= 9 && data[0] == 0xFF && data[4] == 0x41 {
		challenge := data[5:9]
		pkt := append(append([]byte{}, basePkt...), challenge...)
		if _, err := conn.Write(pkt); err != nil {
			return nil, err
		}
		n, err = conn.Read(buf)
		if err != nil {
			return nil, err
		}
		data = buf[:n]
	}
	return data, nil
}

type a2sInfo struct {
	Players    int
	MaxPlayers int
	Map        string
}

func queryA2SInfo(host string, port int) (*a2sInfo, error) {
	data, err := recvA2SPacket(host, port, 1200*time.Millisecond)
	if err != nil || len(data) < 6 || data[0] != 0xFF || data[4] != 0x49 {
		return nil, err
	}
	off := 5
	off++
	_, off = readCString(data, off)
	mapName, off := readCString(data, off)
	_, off = readCString(data, off)
	_, off = readCString(data, off)
	if off+4 > len(data) {
		return &a2sInfo{Map: mapName}, nil
	}
	off += 2
	if off+2 > len(data) {
		return &a2sInfo{Map: mapName}, nil
	}
	players := int(data[off])
	maxPlayers := int(data[off+1])
	return &a2sInfo{Players: players, MaxPlayers: maxPlayers, Map: mapName}, nil
}

func queryA2SPlayers(host string, port int) ([]Player, error) {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return nil, err
	}
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(1600 * time.Millisecond))

	challenge := int32(-1)
	for attempt := 0; attempt < 2; attempt++ {
		pkt := make([]byte, 9)
		copy(pkt, []byte{0xFF, 0xFF, 0xFF, 0xFF, 0x55})
		binary.LittleEndian.PutUint32(pkt[5:], uint32(challenge))
		if _, err := conn.Write(pkt); err != nil {
			return nil, err
		}
		buf := make([]byte, 14000)
		n, err := conn.Read(buf)
		if err != nil {
			return nil, err
		}
		data := buf[:n]
		if len(data) < 6 || data[0] != 0xFF {
			return nil, fmt.Errorf("invalid a2s player response")
		}
		switch data[4] {
		case 0x41:
			if len(data) < 9 {
				return nil, fmt.Errorf("invalid challenge")
			}
			challenge = int32(binary.LittleEndian.Uint32(data[5:9]))
			continue
		case 0x44:
			num := int(data[5])
			off := 6
			out := make([]Player, 0, num)
			for i := 0; i < num && off < len(data); i++ {
				off++
				name, noff := readCString(data, off)
				off = noff
				if off+8 > len(data) {
					break
				}
				score := int32(binary.LittleEndian.Uint32(data[off : off+4]))
				off += 8
				out = append(out, Player{Name: name, Score: float64(score)})
			}
			return out, nil
		default:
			return nil, fmt.Errorf("unexpected a2s packet type %d", data[4])
		}
	}
	return []Player{}, nil
}

type sampInfo struct {
	Players    int
	MaxPlayers int
	Hostname   string
	GameMode   string
}

func sampString(data []byte, off int) (string, int) {
	if off+4 > len(data) {
		return "", len(data)
	}
	n := int(binary.LittleEndian.Uint32(data[off:]))
	off += 4
	if n < 0 || off+n > len(data) {
		return "", len(data)
	}
	return string(data[off : off+n]), off + n
}

func querySampInfo(host string, port int) (online, max int, err error) {
	ip := net.ParseIP(host)
	if ip == nil || ip.To4() == nil {
		return 0, 0, fmt.Errorf("invalid host for samp")
	}
	v4 := ip.To4()
	pkt := append([]byte("SAMP"), v4[0], v4[1], v4[2], v4[3], byte(port&0xFF), byte(port>>8), 'i')
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return 0, 0, err
	}
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return 0, 0, err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(1200 * time.Millisecond))
	if _, err := conn.Write(pkt); err != nil {
		return 0, 0, err
	}
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil || n < 16 || !strings.HasPrefix(string(buf[:4]), "SAMP") {
		return 0, 0, err
	}
	data := buf[:n]
	if len(data) < 14 {
		return 0, 0, nil
	}
	off := 11
	players := int(binary.LittleEndian.Uint16(data[off+1:]))
	maxPlayers := int(binary.LittleEndian.Uint16(data[off+3:]))
	return players, maxPlayers, nil
}

func querySampFull(host string, port int) (*sampInfo, error) {
	ip := net.ParseIP(host)
	if ip == nil || ip.To4() == nil {
		return nil, fmt.Errorf("invalid host for samp")
	}
	v4 := ip.To4()
	pkt := append([]byte("SAMP"), v4[0], v4[1], v4[2], v4[3], byte(port&0xFF), byte(port>>8), 'i')
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return nil, err
	}
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(1200 * time.Millisecond))
	if _, err := conn.Write(pkt); err != nil {
		return nil, err
	}
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		return nil, err
	}
	data := buf[:n]
	if n < 16 || !strings.HasPrefix(string(data[:4]), "SAMP") {
		return nil, fmt.Errorf("не ответ SA-MP")
	}
	off := 12
	out := &sampInfo{
		Players:    int(binary.LittleEndian.Uint16(data[off:])),
		MaxPlayers: int(binary.LittleEndian.Uint16(data[off+2:])),
	}
	off += 4
	out.Hostname, off = sampString(data, off)
	out.GameMode, _ = sampString(data, off)
	return out, nil
}

func parseMcListOutput(text string) []Player {
	text = strings.TrimSpace(text)
	if text == "" || !strings.Contains(strings.ToLower(text), "players online") {
		return []Player{}
	}
	namesPart := ""
	if idx := strings.Index(text, ":"); idx >= 0 {
		namesPart = strings.TrimSpace(text[idx+1:])
	}
	if namesPart == "" {
		return []Player{}
	}
	out := []Player{}
	for _, n := range strings.Split(namesPart, ",") {
		n = strings.TrimSpace(n)
		if n != "" {
			out = append(out, Player{Name: n})
		}
	}
	return out
}
