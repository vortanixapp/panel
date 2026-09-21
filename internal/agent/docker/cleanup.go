package docker

import (
	"context"
	"fmt"
)

func CleanupServerTraces(ctx context.Context, serverID string) []error {
	var errs []error
	unlock := firewallLock(serverID)
	defer unlock()
	var state struct {
		Rules []FirewallRule `json:"rules"`
	}
	if readStateFile(serverID, stateFirewallFile, &state) {
		if _, err := HostShell(ctx, firewallRemoveScript(firewallChainName(serverID))); err != nil {
			errs = append(errs, fmt.Errorf("правила файрвола: %w", err))
		}
	}
	dropServerState(serverID)
	return errs
}
