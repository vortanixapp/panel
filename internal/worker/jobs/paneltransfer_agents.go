package jobs

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/vortanixapp/panel/pkg/buildinfo"
	"github.com/vortanixapp/panel/pkg/sshclient"
)

const panelTransferPinWait = 2 * time.Minute

func (r *Runner) targetRelayPin(cfg sshclient.Config, mode string) (string, error) {
	query := "select value->>'pin' from core.tenant_settings where key = 'relay.tls'"
	cmd := fmt.Sprintf("cd %s && %s exec -T postgres psql -U vortanix -d vortanix -tAc %s",
		panelTransferDir, panelComposeCommand(mode), shellArg(query))
	deadline := time.Now().Add(panelTransferPinWait)
	for {
		out, err := sshclient.RunCapture(cfg, cmd)
		pin := strings.TrimSpace(out)
		if err == nil && strings.HasPrefix(pin, "sha256:") {
			return pin, nil
		}
		if time.Now().After(deadline) {
			return "", errors.New("новая панель не выпустила сертификат relay")
		}
		time.Sleep(5 * time.Second)
	}
}

func panelTransferRelayTarget(address, pin string) (string, string) {
	raw := address
	switch {
	case strings.HasPrefix(address, "https://"):
		raw = "wss://" + strings.TrimPrefix(address, "https://")
	case strings.HasPrefix(address, "http://"):
		raw = "ws://" + strings.TrimPrefix(address, "http://")
	case !strings.Contains(address, "://"):
		raw = "wss://" + address
	}
	return secureRelayURL(raw, pin)
}

func (r *Runner) switchAgents(ctx context.Context, rec *panelTransferRecord, cfg sshclient.Config,
	lg *panelTransferLog, address, mode string) error {
	nodes, err := r.panelTransferNodeIDs(ctx)
	if err != nil {
		return err
	}
	if len(nodes) == 0 {
		lg.say("Узлов для переключения нет")
		return nil
	}

	pin := ""
	if strings.HasPrefix(address, "http://") {
		r.setPanelTransferStage(ctx, rec.ID, "relay_cert")
		lg.say("Жду сертификат relay на новом сервере")
		pin, err = r.targetRelayPin(cfg, mode)
		if err != nil {
			return err
		}
	}

	relayURL, relayPin := panelTransferRelayTarget(address, pin)
	r.setPanelTransferStage(ctx, rec.ID, "agents")
	_, _ = r.db.Exec(ctx, `UPDATE core.panel_transfers SET agents_total = $2, updated_at = now() WHERE id = $1`,
		rec.ID, len(nodes))
	lg.say("Переключаю узлы на %s", relayURL)

	done, failed := 0, 0
	for _, nodeID := range nodes {
		node, err := r.loadNodeSSH(ctx, r.db, nodeID)
		if err != nil {
			failed++
			lg.say("Узел %s пропущен: %s", nodeID, err)
			r.setPanelTransferAgents(ctx, rec.ID, done, failed)
			continue
		}
		commands := daemonAgentCommands(node.ID, relayURL, relayPin, buildinfo.Current())
		nodeCfg := sshclient.Config{
			Host:         node.SSHHost,
			Port:         node.SSHPort,
			User:         node.SSHUser,
			Password:     node.SSHPassword,
			KnownHostKey: node.SSHHostKey,
			Timeout:      30 * time.Second,
			ExecTimeout:  10 * time.Minute,
		}
		if err := sshclient.RunInput(nodeCfg, commands, agentSecrets(node.AgentToken), lg); err != nil {
			failed++
			lg.say("Узел %s не переключён: %s", node.SSHHost, err)
		} else {
			done++
			lg.say("Узел %s переключён", node.SSHHost)
		}
		r.setPanelTransferAgents(ctx, rec.ID, done, failed)
	}
	if failed > 0 {
		lg.say("Не переключено узлов: %d. Повторите переключение из раздела переноса", failed)
	}
	return nil
}

func (r *Runner) panelTransferNodeIDs(ctx context.Context) ([]string, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id::text FROM core.nodes
		WHERE ssh_host IS NOT NULL AND ssh_host <> '' ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("список узлов не получен: %w", err)
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			out = append(out, id)
		}
	}
	return out, rows.Err()
}

func (r *Runner) setPanelTransferAgents(ctx context.Context, id string, done, failed int) {
	_, _ = r.db.Exec(ctx, `
		UPDATE core.panel_transfers SET agents_done = $2, agents_failed = $3, updated_at = now() WHERE id = $1
	`, id, done, failed)
}
