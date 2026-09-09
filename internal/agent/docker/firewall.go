package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type FirewallRule struct {
	ID       string `json:"id"`
	Protocol string `json:"protocol"`
	PortFrom int    `json:"port_from"`
	PortTo   *int   `json:"port_to"`
	Enabled  bool   `json:"enabled"`
}

func SyncFirewall(ctx context.Context, serverID string, rules []FirewallRule) error {
	if err := ensureIptables(ctx); err != nil {
		return err
	}

	statePath := filepath.Join(serverDataDir(serverID), ".vortanix_firewall.json")
	if err := writeJSONFile(statePath, map[string]any{"rules": rules}); err != nil {
		return err
	}

	ip, err := containerIP(ctx, serverID)
	if err != nil {
		return err
	}

	chain := firewallChainName(serverID)
	if err := iptables(ctx, "-N", chain); err != nil {
		return err
	}
	if err := iptables(ctx, "-F", chain); err != nil {
		return err
	}

	enabled := make([]FirewallRule, 0, len(rules))
	for _, r := range rules {
		if r.Enabled && r.PortFrom > 0 {
			enabled = append(enabled, r)
		}
	}

	if len(enabled) == 0 {
		removeFirewallJump(ctx, ip, chain)
		_ = iptables(ctx, "-X", chain)
		return nil
	}

	for _, r := range enabled {
		proto := strings.ToLower(strings.TrimSpace(r.Protocol))
		if proto == "" {
			proto = "tcp"
		}
		if proto != "tcp" && proto != "udp" {
			return fmt.Errorf("unsupported protocol %q", r.Protocol)
		}
		dport := formatPortRange(r.PortFrom, r.PortTo)
		if err := iptables(ctx, "-A", chain, "-p", proto, "--dport", dport, "-j", "DROP"); err != nil {
			return err
		}
	}

	return ensureFirewallJump(ctx, ip, chain)
}

func containerIP(ctx context.Context, serverID string) (string, error) {
	cname := ContainerName(serverID)
	out, err := exec.CommandContext(ctx, "docker", "inspect", "-f",
		"{{range .NetworkSettings.Networks}}{{.IPAddress}} {{end}}", cname).Output()
	if err != nil {
		return "", fmt.Errorf("container not found")
	}
	for _, ip := range strings.Fields(string(out)) {
		if ip != "" {
			return ip, nil
		}
	}
	return "", fmt.Errorf("container has no IP (is it running?)")
}

func firewallChainName(serverID string) string {
	safe := strings.ReplaceAll(serverID, "-", "")
	name := "VRTX-SRV-" + safe
	if len(name) > 28 {
		return name[:28]
	}
	return name
}

func formatPortRange(from int, to *int) string {
	if to != nil && *to > from {
		return fmt.Sprintf("%d:%d", from, *to)
	}
	return fmt.Sprintf("%d", from)
}

func ensureIptables(ctx context.Context) error {
	if err := exec.CommandContext(ctx, "sh", "-c", "command -v iptables >/dev/null 2>&1").Run(); err != nil {
		return fmt.Errorf("iptables not available")
	}
	_ = exec.CommandContext(ctx, "sh", "-c", "iptables -L DOCKER-USER >/dev/null 2>&1 || iptables -N DOCKER-USER >/dev/null 2>&1 || true").Run()
	return nil
}

func iptables(ctx context.Context, args ...string) error {
	cmd := append([]string{"iptables"}, args...)
	err := exec.CommandContext(ctx, cmd[0], cmd[1:]...).Run()
	if err == nil {
		return nil
	}
	if len(args) >= 2 && args[0] == "-N" {
		return nil
	}
	return err
}

func ensureFirewallJump(ctx context.Context, ip, chain string) error {
	check := fmt.Sprintf("iptables -C DOCKER-USER -d %s -j %s >/dev/null 2>&1", ip, chain)
	insert := fmt.Sprintf("iptables -I DOCKER-USER 1 -d %s -j %s", ip, chain)
	return exec.CommandContext(ctx, "sh", "-c", check+" || "+insert).Run()
}

func removeFirewallJump(ctx context.Context, ip, chain string) {
	script := fmt.Sprintf(
		"while iptables -C DOCKER-USER -d %s -j %s >/dev/null 2>&1; do iptables -D DOCKER-USER -d %s -j %s; done",
		ip, chain, ip, chain,
	)
	_ = exec.CommandContext(ctx, "sh", "-c", script).Run()
}

func DecodeFirewallRules(raw any) ([]FirewallRule, error) {
	if raw == nil {
		return []FirewallRule{}, nil
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}
	var rules []FirewallRule
	if err := json.Unmarshal(b, &rules); err != nil {
		return nil, err
	}
	return rules, nil
}

func FirewallChainForServer(serverID string) string {
	return firewallChainName(serverID)
}

func ReadFirewallState(serverID string) ([]FirewallRule, error) {
	path := filepath.Join(serverDataDir(serverID), ".vortanix_firewall.json")
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var state struct {
		Rules []FirewallRule `json:"rules"`
	}
	if err := json.Unmarshal(b, &state); err != nil {
		return nil, err
	}
	return state.Rules, nil
}
