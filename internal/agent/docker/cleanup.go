package docker

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

// CleanupServerTraces снимает следы сервера, которые переживают удаление
// контейнера: цепочку iptables и файл расписания в /etc/cron.d.
//
// Раньше не снимался ни один из них. Цепочка опаснее: docker переиспользует
// адреса контейнеров, и правила удалённого клиента начинали резать трафик
// серверу другого клиента, попавшего на тот же адрес. Файл расписания просто
// продолжал запускать команды несуществующего сервера.
//
// Ошибки не поднимаем наверх: сервер уже удалён, и отказ в очистке не должен
// мешать удалению — но и молчать о них нельзя, поэтому возвращаем списком.
func CleanupServerTraces(ctx context.Context, serverID string) []error {
	var errs []error

	// Адрес контейнера известен, только пока контейнер жив, поэтому переход
	// снимаем по имени цепочки, а не по адресу.
	chain := firewallChainName(serverID)
	if err := dropFirewallJumps(ctx, chain); err != nil {
		errs = append(errs, err)
	}
	// Цепочку сначала чистим, потом удаляем: непустую iptables удалить не даст.
	_ = iptables(ctx, "-F", chain)
	_ = iptables(ctx, "-X", chain)

	if err := os.Remove(cronFilePath(serverID)); err != nil && !os.IsNotExist(err) {
		errs = append(errs, fmt.Errorf("файл расписания: %w", err))
	}
	return errs
}

// dropFirewallJumps убирает все переходы в цепочку из DOCKER-USER. Их может
// быть несколько: адрес контейнера меняется при пересоздании, а старое правило
// оставалось висеть.
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
