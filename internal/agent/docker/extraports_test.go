package docker

import (
	"testing"
)

// Порты со вкладки «Порты» доходили до базы панели и там останавливались.
// Публикует их docker run, поэтому проверяем именно аргументы запуска.
func TestBuildRunArgsPublishesExtraPorts(t *testing.T) {
	t.Setenv("VORTANIX_DATA_DIR", t.TempDir())
	const serverID = "srv-extra-ports"

	changed, err := SyncExtraPorts(serverID, []ExtraPort{
		{Port: 27020, Protocol: "udp", Purpose: "rcon"},
		{Port: 8080, Protocol: "tcp", Purpose: "web"},
	})
	if err != nil {
		t.Fatalf("сохранение портов: %v", err)
	}
	if !changed {
		t.Fatal("первое сохранение должно считаться изменением")
	}

	args := buildRunArgs(serverID, "cs16", nil, "img", 27015, "")
	if !hasArgValue(args, "-p", "27020:27020/udp") {
		t.Error("дополнительный порт 27020/udp не опубликован")
	}
	if !hasArgValue(args, "-p", "8080:8080/tcp") {
		t.Error("дополнительный порт 8080/tcp не опубликован")
	}
}

// Протокол «both» означает две публикации, иначе половина трафика не пройдёт.
func TestExtraPortBothProtocols(t *testing.T) {
	t.Setenv("VORTANIX_DATA_DIR", t.TempDir())
	const serverID = "srv-both"

	if _, err := SyncExtraPorts(serverID, []ExtraPort{{Port: 7777, Protocol: "both"}}); err != nil {
		t.Fatalf("сохранение портов: %v", err)
	}
	args := buildRunArgs(serverID, "unknown-game", nil, "img", 0, "")
	if !hasArgValue(args, "-p", "7777:7777/tcp") || !hasArgValue(args, "-p", "7777:7777/udp") {
		t.Error("для «both» нужны обе публикации, tcp и udp")
	}
}

// Порт, уже занятый раскладкой игры, второй раз публиковать нельзя: docker
// откажется поднимать контейнер целиком, и сервер просто не запустится.
func TestExtraPortDoesNotDuplicateCatalogPort(t *testing.T) {
	t.Setenv("VORTANIX_DATA_DIR", t.TempDir())
	const serverID = "srv-dup"

	if _, err := SyncExtraPorts(serverID, []ExtraPort{{Port: 27015, Protocol: "udp"}}); err != nil {
		t.Fatalf("сохранение портов: %v", err)
	}
	args := buildRunArgs(serverID, "cs16", nil, "img", 27015, "")

	count := 0
	for _, v := range argValues(args, "-p") {
		if v == "27015:27015/udp" {
			count++
		}
	}
	if count > 1 {
		t.Errorf("порт 27015/udp опубликован %d раз", count)
	}
}

// Повторная синхронизация тем же списком не должна считаться изменением:
// иначе агент пересоздавал бы контейнер на каждую правку соседнего порта.
func TestSyncExtraPortsReportsNoChangeForSameList(t *testing.T) {
	t.Setenv("VORTANIX_DATA_DIR", t.TempDir())
	const serverID = "srv-same"

	ports := []ExtraPort{{Port: 30000, Protocol: "tcp"}, {Port: 30001, Protocol: "udp"}}
	if _, err := SyncExtraPorts(serverID, ports); err != nil {
		t.Fatalf("сохранение портов: %v", err)
	}
	changed, err := SyncExtraPorts(serverID, []ExtraPort{ports[1], ports[0]})
	if err != nil {
		t.Fatalf("повторное сохранение: %v", err)
	}
	if changed {
		t.Error("тот же список в другом порядке — не изменение")
	}

	changed, err = SyncExtraPorts(serverID, []ExtraPort{ports[0]})
	if err != nil {
		t.Fatalf("сохранение после удаления: %v", err)
	}
	if !changed {
		t.Error("удаление порта должно считаться изменением")
	}
}

// Мусор в списке не должен доезжать до docker run.
func TestSyncExtraPortsDropsInvalid(t *testing.T) {
	t.Setenv("VORTANIX_DATA_DIR", t.TempDir())
	const serverID = "srv-invalid"

	if _, err := SyncExtraPorts(serverID, []ExtraPort{
		{Port: 0, Protocol: "tcp"},
		{Port: 70000, Protocol: "tcp"},
		{Port: 25565, Protocol: "sctp"},
	}); err != nil {
		t.Fatalf("сохранение портов: %v", err)
	}

	saved := ExtraPortsFor(serverID)
	if len(saved) != 1 {
		t.Fatalf("ожидался один порт, получено %d", len(saved))
	}
	if saved[0].Port != 25565 || saved[0].Protocol != "udp" {
		t.Errorf("неизвестный протокол должен стать udp, получено %+v", saved[0])
	}
}
