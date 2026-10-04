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
	Action   string `json:"action"`
	Source   string `json:"source"`
}

type FirewallLimit struct {
	ConnLimit  int   `json:"conn_limit"`
	LimitPorts []int `json:"limit_ports"`
}

const maxConnLimit = 100000

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

func SyncFirewall(ctx context.Context, serverID string, rules []FirewallRule, limit FirewallLimit) error {
	for _, r := range rules {
		if _, err := ruleProtocol(r); err != nil {
			return err
		}
		if r.PortFrom < 0 || r.PortFrom > 65535 || (r.PortTo != nil && (*r.PortTo < 0 || *r.PortTo > 65535)) {
			return errors.New("порты правила вне диапазона 0–65535")
		}
		if _, err := ruleSource(r); err != nil {
			return err
		}
	}
	if limit.ConnLimit < 0 || limit.ConnLimit > maxConnLimit {
		return errors.New("лимит подключений вне допустимого диапазона")
	}
	for _, p := range limit.LimitPorts {
		if p < 1 || p > 65535 {
			return errors.New("порт лимита подключений вне диапазона 1–65535")
		}
	}
	state := map[string]any{"rules": rules, "conn_limit": limit.ConnLimit, "limit_ports": limit.LimitPorts}
	if err := writeStateFile(serverID, stateFirewallFile, state); err != nil {
		return err
	}
	return ApplyFirewall(ctx, serverID)
}

func ApplyFirewall(ctx context.Context, serverID string) error {
	defer firewallLock(serverID)()
	rules, limit := readFirewallState(serverID)
	chain := firewallChainName(serverID)
	enabled := enabledRules(rules)
	if len(enabled) == 0 && !limit.active() {
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
	script, err := firewallApplyScript(chain, ip, enabled, limit)
	if err != nil {
		return err
	}
	_, err = HostShell(ctx, script)
	return err
}

func (l FirewallLimit) active() bool {
	return l.ConnLimit > 0 && len(l.LimitPorts) > 0
}

func HasFirewallRules(serverID string) bool {
	rules, limit := readFirewallState(serverID)
	return len(enabledRules(rules)) > 0 || limit.active()
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

func ruleSource(r FirewallRule) (string, error) {
	src := strings.TrimSpace(r.Source)
	if src == "" {
		if strings.EqualFold(strings.TrimSpace(r.Action), "allow") {
			return "", errors.New("правило «разрешить» требует адрес источника")
		}
		return "", nil
	}
	if ip := net.ParseIP(src); ip != nil {
		if ip.To4() == nil {
			return "", fmt.Errorf("поддерживаются только адреса IPv4: %q", r.Source)
		}
		return ip.To4().String() + "/32", nil
	}
	ip, network, err := net.ParseCIDR(src)
	if err != nil || ip.To4() == nil {
		return "", fmt.Errorf("неверный адрес источника %q", r.Source)
	}
	return network.String(), nil
}

func isAllowRule(r FirewallRule) bool {
	return strings.EqualFold(strings.TrimSpace(r.Action), "allow")
}

func firewallApplyScript(chain, ip string, rules []FirewallRule, limit FirewallLimit) (string, error) {
	parsed := net.ParseIP(ip)
	if parsed == nil || parsed.To4() == nil {
		return "", fmt.Errorf("неверный адрес контейнера %q", ip)
	}
	var b strings.Builder
	b.WriteString("set -e\n")
	b.WriteString("iptables -w -N DOCKER-USER 2>/dev/null || true\n")
	fmt.Fprintf(&b, "iptables -w -N %s 2>/dev/null || true\n", chain)
	fmt.Fprintf(&b, "iptables -w -F %s\n", chain)
	closed := map[string]bool{}
	var closeOrder []string
	for _, r := range rules {
		if !isAllowRule(r) {
			continue
		}
		proto, err := ruleProtocol(r)
		if err != nil {
			return "", err
		}
		src, err := ruleSource(r)
		if err != nil {
			return "", err
		}
		ports := formatPortRange(r.PortFrom, r.PortTo)
		fmt.Fprintf(&b, "iptables -w -A %s -s %s -p %s --dport %s -j RETURN\n", chain, src, proto, ports)
		key := proto + " " + ports
		if !closed[key] {
			closed[key] = true
			closeOrder = append(closeOrder, key)
		}
	}
	for _, key := range closeOrder {
		proto, ports, _ := strings.Cut(key, " ")
		fmt.Fprintf(&b, "iptables -w -A %s -p %s --dport %s -j DROP\n", chain, proto, ports)
	}
	for _, r := range rules {
		if isAllowRule(r) {
			continue
		}
		proto, err := ruleProtocol(r)
		if err != nil {
			return "", err
		}
		src, err := ruleSource(r)
		if err != nil {
			return "", err
		}
		match := ""
		if src != "" {
			match = " -s " + src
		}
		fmt.Fprintf(&b, "iptables -w -A %s%s -p %s --dport %s -j DROP\n", chain, match, proto, formatPortRange(r.PortFrom, r.PortTo))
	}
	if limit.active() {
		for _, p := range limit.LimitPorts {
			fmt.Fprintf(&b,
				"iptables -w -A %s -p tcp --syn --dport %d -m connlimit --connlimit-above %d --connlimit-mask 32 -j DROP\n",
				chain, p, limit.ConnLimit)
		}
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
	if info, err := engineInspect(ctx, cname); err == nil {
		if !info.Running {
			return "", errContainerNotRunning
		}
		if len(info.IPs) == 0 {
			return "", fmt.Errorf("у контейнера нет адреса в сети Docker")
		}
		return info.IPs[0], nil
	} else if errors.Is(err, errNoContainer) {
		return "", errContainerNotRunning
	}
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

func readFirewallState(serverID string) ([]FirewallRule, FirewallLimit) {
	var state struct {
		Rules []FirewallRule `json:"rules"`
		FirewallLimit
	}
	if !readStateFile(serverID, stateFirewallFile, &state) {
		return nil, FirewallLimit{}
	}
	return state.Rules, state.FirewallLimit
}

func ReadFirewallState(serverID string) ([]FirewallRule, error) {
	rules, _ := readFirewallState(serverID)
	return rules, nil
}

func DecodeFirewallLimit(payload map[string]any) FirewallLimit {
	limit := FirewallLimit{ConnLimit: IntFromPayload(payload["conn_limit"])}
	if list, ok := payload["limit_ports"].([]any); ok {
		for _, item := range list {
			limit.LimitPorts = append(limit.LimitPorts, IntFromPayload(item))
		}
	}
	return limit
}
