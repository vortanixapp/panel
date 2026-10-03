package handlers

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/vortanixapp/panel/pkg/backupname"
	"github.com/vortanixapp/panel/pkg/gameconsole"
	"github.com/vortanixapp/panel/pkg/settingsreg"
)

const (
	stopWait    = 6 * time.Minute
	consoleWait = 20 * time.Second
)

func backupWait() time.Duration {
	return settingsreg.ServersBackupWait.Duration()
}

func restoreWait() time.Duration {
	return settingsreg.ServersRestoreWait.Duration()
}

var errServerStopped = errors.New("сервер должен быть запущен")

func (h *Handler) serverRunning(ctx context.Context, serverID string) bool {
	var status, runtime string
	if h.dbOf(ctx).QueryRow(ctx, `
		SELECT COALESCE(status, ''), COALESCE(runtime_status, '') FROM core.servers WHERE id = $1
	`, serverID).Scan(&status, &runtime) != nil {
		return false
	}
	var live *string
	if cached, ok := h.cache.GetServerStatus(ctx, serverID); ok {
		live = &cached
	}
	return resolveEffectiveStatus(status, runtime, live) == "running"
}

func (h *Handler) createServerBackup(ctx context.Context, serverID, name, source string) (string, string, int64, error) {
	if name != "" {
		if err := backupname.Check(name); err != nil {
			return "", "", 0, err
		}
	}
	if !h.serverRunning(ctx, serverID) {
		return "", "", 0, errServerStopped
	}
	nodeID, err := h.serverNodeID(ctx, serverID)
	if err != nil {
		return "", "", 0, err
	}
	var backupID string
	if err := h.dbOf(ctx).QueryRow(ctx, `
		INSERT INTO core.server_backups (server_id, status, source)
		VALUES ($1, 'running', $2) RETURNING id::text
	`, serverID, source).Scan(&backupID); err != nil {
		return "", "", 0, err
	}
	result, err := h.agentCommandWait(ctx, nodeID, serverID, "backup_create", map[string]any{"name": name}, backupWait())
	if err != nil {
		_, _ = h.dbOf(ctx).Exec(ctx, `UPDATE core.server_backups SET status = 'failed', error = $2 WHERE id = $1`, backupID, err.Error())
		return "", "", 0, err
	}
	filename, _ := result["filename"].(string)
	var size int64
	if v, ok := result["size_bytes"].(float64); ok {
		size = int64(v)
	}
	_, _ = h.dbOf(ctx).Exec(ctx, `
		UPDATE core.server_backups
		SET status = 'completed', filename = $2, size_bytes = $3, completed_at = now()
		WHERE id = $1
	`, backupID, filename, size)
	h.enqueueBackupUpload(ctx, serverID, backupID, filename)
	return backupID, filename, size, nil
}

func (h *Handler) restoreServerBackup(ctx context.Context, serverID, name string) error {
	if err := backupname.Check(strings.TrimSpace(name)); err != nil {
		return err
	}
	nodeID, err := h.serverNodeID(ctx, serverID)
	if err != nil {
		return err
	}
	_, err = h.agentCommandWait(ctx, nodeID, serverID, "backup_restore", map[string]any{"name": name}, restoreWait())
	return err
}

func (h *Handler) listServerBackups(ctx context.Context, serverID string) ([]string, error) {
	nodeID, err := h.serverNodeID(ctx, serverID)
	if err != nil {
		return nil, err
	}
	_, _ = h.agentCommand(ctx, nodeID, serverID, "files_mkdir", map[string]any{"path": backupsDir})
	result, err := h.agentCommand(ctx, nodeID, serverID, "files_list", map[string]any{"path": backupsDir})
	if err != nil {
		return nil, err
	}
	var names []string
	if files, ok := result["files"].([]any); ok {
		for _, item := range files {
			entry, _ := item.(map[string]any)
			name, _ := entry["name"].(string)
			isDir, _ := entry["is_dir"].(bool)
			if name != "" && !isDir && backupname.Valid(name) {
				names = append(names, name)
			}
		}
	}
	for i := 1; i < len(names); i++ {
		for j := i; j > 0 && names[j] > names[j-1]; j-- {
			names[j], names[j-1] = names[j-1], names[j]
		}
	}
	return names, nil
}

func (h *Handler) consoleSend(ctx context.Context, serverID, gameID, command string) (string, error) {
	nodeID, err := h.serverNodeID(ctx, serverID)
	if err != nil {
		return "", err
	}
	result, err := h.agentCommandWait(ctx, nodeID, serverID, "console_command", map[string]any{
		"command": command,
		"game_id": gameID,
	}, consoleWait)
	if err != nil {
		return "", err
	}
	out, _ := result["output"].(string)
	return strings.TrimSpace(out), nil
}

func (h *Handler) consoleTool(ctx context.Context, serverID, gameID, toolID string, args map[string]string) (string, error) {
	cmd, ok := gameconsole.For(gameID).Command(toolID)
	if !ok {
		return "", fmt.Errorf("в консоли этой игры нет команды %q", toolID)
	}
	line, err := cmd.Render(args)
	if err != nil {
		return "", err
	}
	return h.consoleSend(ctx, serverID, gameID, line)
}

func (h *Handler) serverLogLines(ctx context.Context, serverID string, tail int) ([]string, error) {
	nodeID, err := h.serverNodeID(ctx, serverID)
	if err != nil {
		return nil, err
	}
	result, err := h.agentCommand(ctx, nodeID, serverID, "logs", map[string]any{"tail": tail})
	if err != nil {
		return nil, err
	}
	var lines []string
	if raw, ok := result["lines"].([]any); ok {
		for _, item := range raw {
			if s, ok := item.(string); ok {
				lines = append(lines, s)
			}
		}
	}
	return lines, nil
}
