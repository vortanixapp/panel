package docker

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	stateCronFile     = "cron.json"
	stateFirewallFile = "firewall.json"
	statePortsFile    = "ports.json"
)

var legacyStateFiles = map[string]string{
	stateCronFile:     ".vortanix_cron.json",
	stateFirewallFile: ".vortanix_firewall.json",
	statePortsFile:    ".vortanix_ports.json",
}

func safeServerID(serverID string) bool {
	if serverID == "" || len(serverID) > 64 {
		return false
	}
	for _, r := range serverID {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
		default:
			return false
		}
	}
	return true
}

func stateDir(serverID string) (string, error) {
	if !safeServerID(serverID) {
		return "", fmt.Errorf("некорректный идентификатор сервера")
	}
	base := envOr("VORTANIX_STATE_DIR", "/opt/vortanix/state")
	return filepath.Join(base, serverID), nil
}

func writeStateFile(serverID, name string, v any) error {
	dir, err := stateDir(serverID)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return err
	}
	tmp := filepath.Join(dir, name+".tmp")
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, filepath.Join(dir, name)); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	removeLegacyState(serverID, name)
	return nil
}

func readStateFile(serverID, name string, dest any) bool {
	dir, err := stateDir(serverID)
	if err != nil {
		return false
	}
	if raw, readErr := os.ReadFile(filepath.Join(dir, name)); readErr == nil {
		return json.Unmarshal(raw, dest) == nil
	}
	legacy, ok := legacyStateFiles[name]
	if !ok {
		return false
	}
	root, err := serverRootFor(serverID)
	if err != nil {
		return false
	}
	defer root.Close()
	raw, err := root.ReadFile(legacy)
	if err != nil {
		return false
	}
	return json.Unmarshal(raw, dest) == nil
}

func removeStateFile(serverID, name string) {
	if dir, err := stateDir(serverID); err == nil {
		_ = os.Remove(filepath.Join(dir, name))
	}
	removeLegacyState(serverID, name)
}

func removeLegacyState(serverID, name string) {
	legacy, ok := legacyStateFiles[name]
	if !ok {
		return
	}
	root, err := serverRootFor(serverID)
	if err != nil {
		return
	}
	defer root.Close()
	_ = root.Remove(strings.TrimPrefix(legacy, "/"))
}

func dropServerState(serverID string) {
	if dir, err := stateDir(serverID); err == nil {
		_ = os.RemoveAll(dir)
	}
}
