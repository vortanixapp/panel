package nodeevents

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgconn"
)

type Kind string

const (
	Connect           Kind = "connect"
	Disconnect        Kind = "disconnect"
	AgentStarted      Kind = "agent_started"
	VersionChanged    Kind = "version_changed"
	RestartRequested  Kind = "restart_requested"
	UpdateStarted     Kind = "update_started"
	UpdateStage       Kind = "update_stage"
	UpdateDone        Kind = "update_done"
	UpdateFailed      Kind = "update_failed"
	ReinstallStarted  Kind = "reinstall_started"
	ReinstallDone     Kind = "reinstall_done"
	ReinstallFailed   Kind = "reinstall_failed"
	Diagnostics       Kind = "diagnostics"
	Cleanup           Kind = "cleanup"
	TaskFailed        Kind = "task_failed"
	Alert             Kind = "alert"
	SSHExec           Kind = "ssh_exec"
	FirewallFailed    Kind = "firewall_failed"
	CronFailed        Kind = "cron_failed"
	DockerUnreachable Kind = "docker_unreachable"
	DockerRecovered   Kind = "docker_recovered"
)

type Level string

const (
	Info    Level = "info"
	Success Level = "success"
	Warn    Level = "warn"
	Error   Level = "error"
)

type Execer interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

func Record(ctx context.Context, db Execer, nodeID string, kind Kind, level Level, data map[string]any, actorID string) error {
	if nodeID == "" || kind == "" {
		return nil
	}
	if level == "" {
		level = Info
	}
	if data == nil {
		data = map[string]any{}
	}
	raw, err := json.Marshal(data)
	if err != nil {
		raw = []byte("{}")
	}
	_, err = db.Exec(ctx, `
		INSERT INTO core.node_events (node_id, kind, level, data, actor_id)
		VALUES ($1, $2, $3, $4::jsonb, NULLIF($5, '')::uuid)
	`, nodeID, string(kind), string(level), raw, actorID)
	return err
}
