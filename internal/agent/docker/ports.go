package docker

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

func QueryTarget(ctx context.Context, serverID string, containerPort int, proto string) (host string, port int) {
	if containerPort <= 0 {
		return "127.0.0.1", 0
	}
	if ip, err := containerIP(ctx, serverID); err == nil && ip != "" {
		return ip, containerPort
	}
	return "127.0.0.1", ResolveHostPort(ctx, serverID, containerPort, proto)
}

func ResolveHostPort(ctx context.Context, serverID string, containerPort int, proto string) int {
	if containerPort <= 0 {
		return 0
	}
	if proto == "" {
		proto = "udp"
	}
	cname := ContainerName(serverID)
	key := fmt.Sprintf("%d/%s", containerPort, proto)
	tmpl := fmt.Sprintf(`{{with index .NetworkSettings.Ports "%s"}}{{if .}}{{(index . 0).HostPort}}{{end}}{{end}}`, key)
	out, err := exec.CommandContext(ctx, "docker", "inspect", "-f", tmpl, cname).Output()
	if err != nil {
		return containerPort
	}
	hp := strings.TrimSpace(string(out))
	if hp == "" {
		return containerPort
	}
	n, err := strconv.Atoi(hp)
	if err != nil || n <= 0 {
		return containerPort
	}
	return n
}

func IntFromPayload(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	default:
		return 0
	}
}

func StringFromPayload(v any) string {
	switch s := v.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(s)
	case fmt.Stringer:
		return strings.TrimSpace(s.String())
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}
