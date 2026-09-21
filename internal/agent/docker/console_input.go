package docker

import (
	"context"
	"encoding/json"
	"errors"
	"os/exec"
	"strings"
)

var (
	ErrConsoleStopped = errors.New("сервер не запущен")
	ErrConsoleNoStdin = errors.New("сервер запущен без канала для команд, перезапустите его")
)

const consoleMaxLines = 20

type ConsoleReply struct {
	Via    string
	Output string
}

type containerInfo struct {
	State struct {
		Status    string `json:"Status"`
		StartedAt string `json:"StartedAt"`
	} `json:"State"`
	Config struct {
		OpenStdin bool     `json:"OpenStdin"`
		Env       []string `json:"Env"`
	} `json:"Config"`
	NetworkSettings struct {
		IPAddress string `json:"IPAddress"`
		Networks  map[string]struct {
			IPAddress string `json:"IPAddress"`
		} `json:"Networks"`
		Ports map[string][]struct {
			HostIP   string `json:"HostIp"`
			HostPort string `json:"HostPort"`
		} `json:"Ports"`
	} `json:"NetworkSettings"`
}

func inspectContainer(ctx context.Context, cname string) (*containerInfo, error) {
	out, err := exec.CommandContext(ctx, "docker", "inspect", "--type", "container", cname).Output()
	if err != nil {
		return nil, err
	}
	var list []containerInfo
	if err := json.Unmarshal(out, &list); err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, errors.New("контейнер не найден")
	}
	return &list[0], nil
}

func consoleLines(input string) []string {
	var lines []string
	for _, line := range strings.Split(strings.ReplaceAll(input, "\r", "\n"), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			lines = append(lines, line)
		}
		if len(lines) == consoleMaxLines {
			break
		}
	}
	return lines
}

func SendConsoleCommand(ctx context.Context, serverID, gameID, input string) (ConsoleReply, error) {
	lines := consoleLines(input)
	if len(lines) == 0 {
		return ConsoleReply{}, nil
	}
	cname := ContainerName(serverID)
	info, err := inspectContainer(ctx, cname)
	if err != nil || info.State.Status != "running" {
		return ConsoleReply{}, ErrConsoleStopped
	}

	rest := lines
	var rconErr error
	var outputs []string
	if target, ok := findRcon(ctx, serverID, gameID, cname, info); ok {
		for i, line := range lines {
			out, err := target.run(ctx, info, line)
			if err != nil {
				rconErr = err
				rest = lines[i:]
				forgetRcon(cname)
				break
			}
			if out != "" {
				outputs = append(outputs, out)
			}
		}
		if rconErr == nil {
			return ConsoleReply{Via: target.via(), Output: strings.Join(outputs, "\n")}, nil
		}
	}

	if info.Config.OpenStdin {
		if err := writeContainerStdin(ctx, cname, rest); err != nil {
			return ConsoleReply{}, err
		}
		return ConsoleReply{Via: "stdin", Output: strings.Join(outputs, "\n")}, nil
	}
	if rconErr != nil {
		return ConsoleReply{}, rconErr
	}
	return ConsoleReply{}, ErrConsoleNoStdin
}

func writeContainerStdin(ctx context.Context, cname string, lines []string) error {
	cmd := exec.CommandContext(ctx, "docker", "exec", "-i", cname, "sh", "-c", "cat > /proc/1/fd/0")
	cmd.Stdin = strings.NewReader(strings.Join(lines, "\n") + "\n")
	out, err := cmd.CombinedOutput()
	if err != nil {
		if msg := strings.TrimSpace(string(out)); msg != "" {
			return errors.New(msg)
		}
		return err
	}
	return nil
}
