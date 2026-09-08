package docker

import (
	"encoding/binary"
	"testing"
)

// Разбор ответов игровых серверов раньше не был покрыт ничем. Смещения в этих
// пакетах — единственное, что отделяет верный онлайн от нуля на живом сервере,
// и ошибиться в них можно молча: неверное смещение даёт правдоподобное число.

func TestParseBedrockMOTD(t *testing.T) {
	cases := []struct {
		name       string
		motd       string
		players    int
		maxPlayers int
		level      string
	}{
		{
			name:       "ванильный сервер",
			motd:       "MCPE;Мой сервер;800;1.21.4;3;20;123456;Bedrock level;Survival;1;19132;19133",
			players:    3,
			maxPlayers: 20,
			level:      "Bedrock level",
		},
		{
			// PocketMine и Nukkit обрезают строку после счётчиков. Обращение к
			// седьмому полю по фиксированному индексу здесь паниковало бы.
			name:       "укороченный ответ без имени мира",
			motd:       "MCPE;Сервер;800;1.21.4;5;10",
			players:    5,
			maxPlayers: 10,
			level:      "",
		},
		{
			name:       "Education Edition вместо MCPE",
			motd:       "MCEE;Класс;800;1.21.4;1;30;1;Урок;Creative",
			players:    1,
			maxPlayers: 30,
			level:      "Урок",
		},
		{
			name:       "нечисловые счётчики не роняют разбор",
			motd:       "MCPE;Сервер;800;1.21.4;много;;1;Мир",
			players:    0,
			maxPlayers: 0,
			level:      "Мир",
		},
		{
			name:       "пустая строка",
			motd:       "",
			players:    0,
			maxPlayers: 0,
			level:      "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseBedrockMOTD(tc.motd)
			if got.Players != tc.players {
				t.Errorf("игроков: получили %d, ожидали %d", got.Players, tc.players)
			}
			if got.MaxPlayers != tc.maxPlayers {
				t.Errorf("максимум: получили %d, ожидали %d", got.MaxPlayers, tc.maxPlayers)
			}
			if got.LevelName != tc.level {
				t.Errorf("мир: получили %q, ожидали %q", got.LevelName, tc.level)
			}
		})
	}
}

func TestSampStringНеВыходитЗаПределы(t *testing.T) {
	// Длина строки приходит по сети. Если ей верить без проверки, битый ответ
	// уводит срез за границу буфера и роняет агента.
	data := make([]byte, 8)
	binary.LittleEndian.PutUint32(data[0:], 0xFFFFFF)
	if s, off := sampString(data, 0); s != "" || off != len(data) {
		t.Fatalf("завышенная длина должна давать пустую строку, получили %q/%d", s, off)
	}
	if s, _ := sampString(data, 7); s != "" {
		t.Fatalf("обрезанный заголовок длины должен давать пустую строку, получили %q", s)
	}
}

func TestSampStringЧитаетКириллицу(t *testing.T) {
	value := "Русский режим"
	data := make([]byte, 4, 4+len(value))
	binary.LittleEndian.PutUint32(data[0:], uint32(len(value)))
	data = append(data, []byte(value)...)

	got, off := sampString(data, 0)
	if got != value {
		t.Fatalf("получили %q, ожидали %q", got, value)
	}
	if off != len(data) {
		t.Fatalf("смещение %d, ожидали %d", off, len(data))
	}
}

// buildSampInfoPacket собирает ответ SA-MP на запрос 'i' так, как его шлёт
// настоящий сервер.
func buildSampInfoPacket(players, max int, hostname, gamemode, language string) []byte {
	pkt := []byte("SAMP")
	pkt = append(pkt, 127, 0, 0, 1) // адрес
	pkt = append(pkt, 0x61, 0x1E)   // порт 7777
	pkt = append(pkt, 'i')          // код ответа
	pkt = append(pkt, 0)            // пароль
	pkt = binary.LittleEndian.AppendUint16(pkt, uint16(players))
	pkt = binary.LittleEndian.AppendUint16(pkt, uint16(max))
	for _, s := range []string{hostname, gamemode, language} {
		pkt = binary.LittleEndian.AppendUint32(pkt, uint32(len(s)))
		pkt = append(pkt, []byte(s)...)
	}
	return pkt
}

func TestSampПакетРазбираетсяПоТемЖеСмещениям(t *testing.T) {
	// Проверяем согласованность двух читателей одного пакета: querySampInfo
	// берёт счётчики по смещению 12/14, querySampFull — оттуда же плюс строки.
	// Разъехавшись, они показывали бы разный онлайн в разных местах панели.
	pkt := buildSampInfoPacket(7, 250, "Мой сервер", "Role Play", "Russian")

	players := int(binary.LittleEndian.Uint16(pkt[12:]))
	maxPlayers := int(binary.LittleEndian.Uint16(pkt[14:]))
	if players != 7 || maxPlayers != 250 {
		t.Fatalf("счётчики: получили %d/%d, ожидали 7/250", players, maxPlayers)
	}

	off := 16
	hostname, off := sampString(pkt, off)
	gamemode, _ := sampString(pkt, off)
	if hostname != "Мой сервер" {
		t.Errorf("имя: получили %q", hostname)
	}
	if gamemode != "Role Play" {
		t.Errorf("режим: получили %q", gamemode)
	}
}

// buildA2SInfoPacket собирает ответ A2S_INFO движка Source.
func buildA2SInfoPacket(name, mapName string, players, max int) []byte {
	pkt := []byte{0xFF, 0xFF, 0xFF, 0xFF, 0x49}
	pkt = append(pkt, 17) // версия протокола
	for _, s := range []string{name, mapName, "cstrike", "Counter-Strike"} {
		pkt = append(pkt, []byte(s)...)
		pkt = append(pkt, 0)
	}
	pkt = binary.LittleEndian.AppendUint16(pkt, 10) // AppID
	pkt = append(pkt, byte(players), byte(max), 0)
	return pkt
}

func TestA2SInfoСмещенияПолей(t *testing.T) {
	// Разбор идёт через queryA2SInfo, который сам ходит в сеть, поэтому здесь
	// повторяется его арифметика смещений — она и есть предмет проверки.
	pkt := buildA2SInfoPacket("Мой сервер", "de_dust2", 12, 32)

	off := 5
	off++ // версия протокола
	_, off = readCString(pkt, off)
	mapName, off := readCString(pkt, off)
	_, off = readCString(pkt, off)
	_, off = readCString(pkt, off)
	off += 2 // AppID

	if mapName != "de_dust2" {
		t.Errorf("карта: получили %q, ожидали de_dust2", mapName)
	}
	if int(pkt[off]) != 12 {
		t.Errorf("игроков: получили %d, ожидали 12", pkt[off])
	}
	if int(pkt[off+1]) != 32 {
		t.Errorf("максимум: получили %d, ожидали 32", pkt[off+1])
	}
}

func TestBedrockИSampРазныеСемейства(t *testing.T) {
	// Bedrock и Java делят в каталоге одно значение QueryProto, и раньше
	// Bedrock проваливался мимо всех веток опроса.
	if !isMcBedrockGame("mcbedrock") {
		t.Error("mcbedrock должен опознаваться как Bedrock")
	}
	if isMcJavaGame("mcbedrock") {
		t.Error("mcbedrock не должен попадать в ветку Java")
	}
	for _, java := range []string{"mcjava", "mcpaper", "mcspigot", "mcforge", "mcfabric"} {
		if !isMcJavaGame(java) {
			t.Errorf("%s должен опознаваться как Java", java)
		}
		if isMcBedrockGame(java) {
			t.Errorf("%s не должен попадать в ветку Bedrock", java)
		}
	}
}
