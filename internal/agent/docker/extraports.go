package docker

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type ExtraPort struct {
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
	Purpose  string `json:"purpose"`
}

func extraPortsPath(serverID string) string {
	return filepath.Join(serverDataDir(serverID), ".vortanix_ports.json")
}

func SyncExtraPorts(serverID string, ports []ExtraPort) (bool, error) {
	cleaned := make([]ExtraPort, 0, len(ports))
	for _, p := range ports {
		if p.Port < 1 || p.Port > 65535 {
			continue
		}
		proto := strings.ToLower(strings.TrimSpace(p.Protocol))
		if proto != "tcp" && proto != "udp" && proto != "both" {
			proto = "udp"
		}
		cleaned = append(cleaned, ExtraPort{Port: p.Port, Protocol: proto, Purpose: p.Purpose})
	}

	if samePorts(ExtraPortsFor(serverID), cleaned) {
		return false, nil
	}
	if err := os.MkdirAll(serverDataDir(serverID), 0o755); err != nil {
		return false, err
	}
	if err := writeJSONFile(extraPortsPath(serverID), map[string]any{"ports": cleaned}); err != nil {
		return false, err
	}
	return true, nil
}

func ExtraPortsFor(serverID string) []ExtraPort {
	raw, err := os.ReadFile(extraPortsPath(serverID))
	if err != nil {
		return nil
	}
	var parsed struct {
		Ports []ExtraPort `json:"ports"`
	}
	if json.Unmarshal(raw, &parsed) != nil {
		return nil
	}
	return parsed.Ports
}

func DecodeExtraPorts(v any) []ExtraPort {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	var out []ExtraPort
	if json.Unmarshal(raw, &out) != nil {
		return nil
	}
	return out
}

func samePorts(a, b []ExtraPort) bool {
	if len(a) != len(b) {
		return false
	}
	type key struct {
		port  int
		proto string
	}
	seen := map[key]int{}
	for _, p := range a {
		seen[key{p.Port, p.Protocol}]++
	}
	for _, p := range b {
		k := key{p.Port, p.Protocol}
		seen[k]--
		if seen[k] < 0 {
			return false
		}
	}
	return true
}

func extraPortArgs(serverID, bindIP string, taken map[int]bool) []string {
	ports := ExtraPortsFor(serverID)
	if len(ports) == 0 {
		return nil
	}
	bindPrefix := ""
	if ip := strings.TrimSpace(bindIP); ip != "" {
		bindPrefix = ip + ":"
	}
	args := make([]string, 0, len(ports)*2)
	for _, p := range ports {
		if p.Port < 1 || p.Port > 65535 || taken[p.Port] {
			continue
		}
		protocols := []string{p.Protocol}
		if p.Protocol == "both" {
			protocols = []string{"tcp", "udp"}
		}
		for _, proto := range protocols {
			port := strconv.Itoa(p.Port)
			args = append(args, "-p", bindPrefix+port+":"+port+"/"+proto)
		}
		taken[p.Port] = true
	}
	return args
}
