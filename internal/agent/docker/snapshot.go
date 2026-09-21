package docker

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/vortanixapp/panel/pkg/protocol"
)

var exitCodePattern = regexp.MustCompile(`Exited \((\d+)\)`)

func SnapshotServers(ctx context.Context) ([]protocol.SnapshotServer, error) {
	out, err := exec.CommandContext(ctx, "docker", "ps", "-a",
		"--filter", "name=^vortanix-",
		"--format", "{{.Names}}|{{.State}}|{{.Status}}").Output()
	if err != nil {
		return nil, err
	}
	var list []protocol.SnapshotServer
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		parts := strings.SplitN(strings.TrimSpace(line), "|", 3)
		if len(parts) < 2 {
			continue
		}
		id := ServerIDFromContainer(parts[0])
		if id == "" {
			continue
		}
		human := ""
		if len(parts) == 3 {
			human = parts[2]
		}
		status, errMsg := snapshotStatus(parts[1], human)
		if status == "" {
			continue
		}
		list = append(list, protocol.SnapshotServer{ServerID: id, Status: status, Error: errMsg})
	}
	return list, nil
}

func snapshotStatus(state, human string) (string, string) {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "running":
		return "running", ""
	case "created", "paused":
		return "stopped", ""
	case "restarting":
		return "error", "контейнер перезапускается после сбоя"
	case "dead", "removing":
		return "error", "контейнер в состоянии " + state
	case "exited":
		if m := exitCodePattern.FindStringSubmatch(human); m != nil {
			code, _ := strconv.Atoi(m[1])
			switch code {
			case 0, 130, 137, 143:
				return "stopped", ""
			}
			return "error", fmt.Sprintf("процесс сервера завершился с кодом %d", code)
		}
		return "stopped", ""
	}
	return "", ""
}
