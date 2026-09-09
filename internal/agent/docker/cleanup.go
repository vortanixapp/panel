package docker

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

func CleanupServerTraces(ctx context.Context, serverID string) []error {
	var errs []error

	chain := firewallChainName(serverID)
	if err := dropFirewallJumps(ctx, chain); err != nil {
		errs = append(errs, err)
	}
	_ = iptables(ctx, "-F", chain)
	_ = iptables(ctx, "-X", chain)

	if err := os.Remove(cronFilePath(serverID)); err != nil && !os.IsNotExist(err) {
		errs = append(errs, fmt.Errorf("файл расписания: %w", err))
	}
	return errs
}

func dropFirewallJumps(ctx context.Context, chain string) error {
	script := fmt.Sprintf(
		"while iptables -L DOCKER-USER -n --line-numbers 2>/dev/null | grep -q ' %s '; do "+
			"n=$(iptables -L DOCKER-USER -n --line-numbers | grep ' %s ' | head -1 | awk '{print $1}'); "+
			"[ -n \"$n\" ] || break; iptables -D DOCKER-USER \"$n\" || break; done",
		chain, chain,
	)
	if err := exec.CommandContext(ctx, "sh", "-c", script).Run(); err != nil {
		return fmt.Errorf("переходы в цепочку %s: %w", chain, err)
	}
	return nil
}
