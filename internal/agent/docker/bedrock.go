package docker

import (
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

// Опрос Minecraft Bedrock по RakNet.
//
// Ветки для Bedrock не было вовсе: в каталоге у него QueryProto = mc, но
// разбор шёл через isMcJavaGame, который дополнительно требует образ
// .../mcjava, а у Bedrock образ свой. Запрос проваливался мимо всех веток, и
// живой сервер всегда показывал ноль игроков и пустую карту.
//
// Java и Bedrock объединены в каталоге одним значением mc, хотя протоколы
// несовместимы: у Java это TCP, у Bedrock — UDP RakNet. Отличаем по образу.

// magic —OFFLINE_MESSAGE_DATA_ID, обязательная подпись RakNet.
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

// queryBedrock шлёт Unconnected Ping (0x01) и разбирает Unconnected Pong (0x1C).
//
// Все числа в RakNet — big-endian, в отличие от A2S и SA-MP, где little-endian.
// Челленджа нет, хватает одного обмена.
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
	// Идентификатор клиента: серверу безразличен, лишь бы поле присутствовало.
	guid := make([]byte, 8)
	binary.BigEndian.PutUint64(guid, 0x564F5254414E4958) // "VORTANIX"
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
	// 1 байт кода + 8 времени + 8 GUID + 16 подписи + 2 длины строки.
	if n < 35 || data[0] != 0x1C {
		return nil, fmt.Errorf("не ответ RakNet")
	}
	strLen := int(binary.BigEndian.Uint16(data[33:35]))
	if strLen <= 0 || 35+strLen > n {
		// Часть сборок объявляет длину неверно; берём остаток пакета.
		strLen = n - 35
	}
	return parseBedrockMOTD(string(data[35 : 35+strLen])), nil
}

// parseBedrockMOTD разбирает строку вида
// MCPE;имя;протокол;версия;онлайн;максимум;GUID;мир;режим;…
//
// Полей может быть меньше двенадцати: PocketMine, Nukkit и старые сборки
// обрезают строку после шестого. Поэтому каждое поле берётся с проверкой длины,
// а не по фиксированному индексу.
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
		// Вторая строка MOTD: ванильный bedrock_server кладёт туда level-name
		// из server.properties — ровно то, что нужно показать как карту.
		LevelName: at(7),
	}
}
