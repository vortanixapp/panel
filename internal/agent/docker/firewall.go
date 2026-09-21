package docker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os/exec"
	"strings"
	"sync"
)

type FirewallRule struct {
	ID       string `json:"id"`
	Protocol string `json:"protocol"`
	PortFrom int    `json:"port_from"`
	PortTo   *int   `json:"port_to"`
	Enabled  bool   `json:"enabled"`
}

var errContainerNotRunning = errors.New("контейнер сервера не запущен")

var (
	firewallLocksMu sync.Mutex
	firewallLocks   = map[string]*sync.Mutex{}
)

func firewallLock(serverID string) func() {
	firewallLocksMu.Lock()
	mu, ok := firewallLocks[serverID]
	if !ok {
		mu = &sync.Mutex{}
		firewallLocks[serverID] = mu
	}
	firewallLocksMu.Unlock()
	mu.Lock()
	return mu.Unlock
}

func SyncFirewall(ctx context.Context, serverID string, rules []FirewallRule) error {
	for _, r := range rules {
		if _, err := ruleProtocol(r); err != nil {
			return err
		}
		if r.PortFrom < 0 || r.PortFrom > 65535 || (r.PortTo != nil && (*r.PortTo < 0 || *r.PortTo > 65535)) {
			return errors.New("порты правила вне диапазона 0–65535")
		}
	}
	if err := writeStateFile(serverID, stateFirewallFile, map[string]any{"rules": rules}); err != nil {
		return err
	}
	return ApplyFirewall(ctx, serverID)
}

func ApplyFirewall(ctx context.Context, serverID string) error {
	defer firewallLock(serverID)()
	rules, _ := ReadFirewallState(serverID)
	chain := firewallChainName(serverID)
	enabled := enabledRules(rules)
	if len(enabled) == 0 {
		_, err := HostShell(ctx, firewallRemoveScript(chain))
		return err
	}
	ip, err := containerIP(ctx, serverID)
	if errors.Is(err, errContainerNotRunning) {
		_, err = HostShell(ctx, dropJumpsScript(chain))
		return err
	}
	if err != nil {
		return err
	}
	script, err := firewallApplyScript(chain, ip, enabled)
	if err != nil {
		return err
	}
	_, err = HostShell(ctx, script)
	return err
}

func HasFirewallRules(serverID string) bool {
	rules, _ := ReadFirewallState(serverID)
	return len(enabledRules(rules)) > 0
}

func enabledRules(rules []FirewallRule) []FirewallRule {
	out := make([]FirewallRule, 0, len(rules))
	for _, r := range rules {
		if r.Enabled && r.PortFrom > 0 {
			out = append(out, r)
		}
	}
	return out
}

func ruleProtocol(r FirewallRule) (string, error) {
	proto := strings.ToLower(strings.TrimSpace(r.Protocol))
	if proto == "" {
		proto = "tcp"
	}
	if proto != "tcp" && proto != "udp" {
		return "", fmt.Errorf("неподдерживаемый протокол %q", r.Protocol)
	}
	return proto, nil
}

func firewallApplyScript(chain, ip string, rules []FirewallRule) (string, error) {
	parsed := net.ParseIP(ip)
	if parsed == nil || parsed.To4() == nil {
		return "", fmt.Errorf("неверный адрес контейнера %q", ip)
	}
	var b strings.Builder
	b.WriteString("set -e\n")
	b.WriteString("iptables -w -N DOCKER-USER 2>/dev/null || true\n")
	fmt.Fprintf(&b, "iptables -w -N %s 2>/dev/null || true\n", chain)
	fmt.Fprintf(&b, "iptables -w -F %s\n", chain)
	for _, r := range rules {
		proto, err := ruleProtocol(r)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&b, "iptables -w -A %s -p %s --dport %s -j DROP\n", chain, proto, formatPortRange(r.PortFrom, r.PortTo))
	}
	b.WriteString(dropJumpsScript(chain))
	fmt.Fprintf(&b, "iptables -w -I DOCKER-USER 1 -d %s/32 -j %s\n", parsed.To4().String(), chain)
	return b.String(), nil
}

func firewallRemoveScript(chain string) string {
	return dropJumpsScript(chain) +
		fmt.Sprintf("iptables -w -F %s 2>/dev/null || true\niptables -w -X %s 2>/dev/null || true\n", chain, chain)
}

func dropJumpsScript(chain string) string {
	return fmt.Sprintf(
		"iptables -w -S DOCKER-USER 2>/dev/null | grep -e ' -j %s$' | sed 's/^-A /-D /' | while read -r rule; do iptables -w $rule; done\n",
		chain,
	)
}

func containerIP(ctx context.Context, serverID string) (string, error) {
	cname := ContainerName(serverID)
	out, err := exec.CommandContext(ctx, "docker", "inspect", "-f",
		"{{.State.Running}} {{range .NetworkSettings.Networks}}{{.IPAddress}} {{end}}", cname).Output()
	if err != nil {
		return "", errContainerNotRunning
	}
	fields := strings.Fields(string(out))
	if len(fields) == 0 || fields[0] != "true" {
		return "", errContainerNotRunning
	}
	for _, ip := range fields[1:] {
		if ip != "" {
			return ip, nil
		}
	}
	return "", fmt.Errorf("у контейнера нет адреса в сети Docker")
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

func ReadFirewallState(serverID string) ([]FirewallRule, error) {
	var state struct {
		Rules []FirewallRule `json:"rules"`
	}
	if !readStateFile(serverID, stateFirewallFile, &state) {
		return nil, nil
	}
	return state.Rules, nil
}
