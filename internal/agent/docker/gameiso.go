package docker

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
)

// Игровые контейнеры стоят в одной сети Docker и по умолчанию видят друг друга
// по внутренним адресам. Публикация порта через -p ограничивает только хост, а
// внутри сети сосед достаёт любой порт чужого контейнера — в том числе RCON и
// query, которые наружу не публикуются. Здесь ставится запрет на обмен между
// игровыми контейнерами: всё остальное (интернет, MySQL ноды, обращение к
// собственному адресу) остаётся доступным.

const (
	gameIsoChain     = "VTX-GAMEISO"
	cloudMetadataNet = "169.254.0.0/16"
	// Метка нужна, чтобы обойтись 2n правилами вместо n². Пакет из игрового
	// контейнера помечается, а дальше проверяется только адрес получателя.
	gameIsoMark = "0x4a000000/0x4a000000"
)

var gameIsoMu sync.Mutex

// GameIsolationEnabled — запрет обмена между игровыми контейнерами. Выключается
// переменной VORTANIX_GAME_ISOLATION=0, если на ноде своя схема сети.
func GameIsolationEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("VORTANIX_GAME_ISOLATION"))) {
	case "0", "false", "no", "off":
		return false
	}
	return true
}

type gameEndpoint struct {
	serverID string
	ip       string
}

func runningGameEndpoints(ctx context.Context) ([]gameEndpoint, error) {
	out, err := exec.CommandContext(ctx, "docker", "ps",
		"--filter", "label=vortanix.managed=true",
		"--format", "{{.Names}}").Output()
	if err != nil {
		return nil, err
	}
	var list []gameEndpoint
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		name := strings.TrimSpace(line)
		if name == "" {
			continue
		}
		serverID := ServerIDFromContainer(name)
		if serverID == "" {
			continue
		}
		ip, ipErr := containerIP(ctx, serverID)
		if ipErr != nil {
			continue
		}
		if parsed := net.ParseIP(ip); parsed == nil || parsed.To4() == nil {
			continue
		}
		list = append(list, gameEndpoint{serverID: serverID, ip: ip})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].ip < list[j].ip })
	return list, nil
}

func gameIsolationScript(endpoints []gameEndpoint) string {
	var b strings.Builder
	b.WriteString("set -e\n")
	b.WriteString("iptables -w -N DOCKER-USER 2>/dev/null || true\n")
	fmt.Fprintf(&b, "iptables -w -N %s 2>/dev/null || true\n", gameIsoChain)
	fmt.Fprintf(&b, "iptables -w -F %s\n", gameIsoChain)

	for _, e := range endpoints {
		fmt.Fprintf(&b, "iptables -w -A %s -s %s/32 -j MARK --set-xmark %s\n",
			gameIsoChain, e.ip, gameIsoMark)
	}
	// Обращение контейнера к собственному опубликованному порту — нормальный
	// случай у части игр, его не режем.
	for _, e := range endpoints {
		fmt.Fprintf(&b, "iptables -w -A %s -s %s/32 -d %s/32 -j RETURN\n",
			gameIsoChain, e.ip, e.ip)
	}
	for _, e := range endpoints {
		fmt.Fprintf(&b, "iptables -w -A %s -m mark --mark %s -d %s/32 -j DROP\n",
			gameIsoChain, gameIsoMark, e.ip)
	}
	fmt.Fprintf(&b, "iptables -w -A %s -m mark --mark %s -d %s -j DROP\n",
		gameIsoChain, gameIsoMark, cloudMetadataNet)
	fmt.Fprintf(&b, "iptables -w -A %s -j MARK --set-xmark 0x0/0x4a000000\n", gameIsoChain)

	b.WriteString(gameIsoDropJumps())
	if len(endpoints) > 0 {
		fmt.Fprintf(&b, "iptables -w -A DOCKER-USER -j %s\n", gameIsoChain)
	}
	return b.String()
}

func gameIsoDropJumps() string {
	return fmt.Sprintf(
		"iptables -w -S DOCKER-USER 2>/dev/null | grep -e ' -j %s$' | sed 's/^-A /-D /' | while read -r rule; do iptables -w $rule; done\n",
		gameIsoChain,
	)
}

func gameIsolationRemoveScript() string {
	return gameIsoDropJumps() +
		fmt.Sprintf("iptables -w -F %s 2>/dev/null || true\niptables -w -X %s 2>/dev/null || true\n",
			gameIsoChain, gameIsoChain)
}

// ApplyGameIsolation пересобирает запрет обмена между игровыми контейнерами.
// Вызывается на каждое событие запуска и остановки контейнера: адреса Docker
// выдаёт заново, поэтому список нужно держать свежим.
func ApplyGameIsolation(ctx context.Context) error {
	gameIsoMu.Lock()
	defer gameIsoMu.Unlock()

	if !GameIsolationEnabled() {
		_, err := HostShell(ctx, gameIsolationRemoveScript())
		return err
	}
	endpoints, err := runningGameEndpoints(ctx)
	if err != nil {
		return fmt.Errorf("список игровых контейнеров: %w", err)
	}
	_, err = HostShell(ctx, gameIsolationScript(endpoints))
	return err
}
