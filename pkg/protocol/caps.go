package protocol

const ProtoVersion = 2

const (
	MsgWelcome       = "welcome"
	MsgStateSnapshot = "state_snapshot"
	MsgTaskProgress  = "task_progress"
	MsgNodeEvent     = "node_event"
)

const (
	ActionNodeInfo       = "node_info"
	ActionAgentLogs      = "agent_logs"
	ActionAgentRestart   = "agent_restart"
	ActionContainersList = "containers_list"
	ActionDiskUsage      = "disk_usage"
	ActionCleanupPreview = "cleanup_preview"
	ActionCleanupApply   = "cleanup_apply"
	ActionDiagnostics    = "diagnostics"
)

const (
	CapLanes         = "lanes"
	CapOutbox        = "outbox"
	CapStateSnapshot = "state_snapshot"
	CapTasks         = "tasks"
	CapNodeEvents    = "node_events"
	CapNodeInfo      = "node_info"
	CapAgentLogs     = "agent_logs"
	CapAgentRestart  = "agent_restart"
	CapContainers    = "containers"
	CapDiskUsage     = "disk_usage"
	CapCleanup       = "cleanup"
	CapDiagnostics   = "diagnostics"
	CapFirewallHost  = "firewall_host"
	CapCronScheduler = "cron_scheduler"
)

const (
	CodeAgentTooOld      = "agent_too_old"
	CodeNodeOffline      = "node_offline"
	CodeAgentBusy        = "agent_busy"
	CodeUnsupported      = "unsupported_action"
	CodeRestartPolicy    = "restart_policy_missing"
	CodeAgentPanic       = "agent_panic"
	CodeTaskTimeout      = "timeout"
	CodeAgentRestarted   = "agent_restarted"
	ResultVersionKey     = "v"
	ResultVersionCurrent = ProtoVersion
)

var requiredCaps = map[string]string{
	ActionNodeInfo:       CapNodeInfo,
	ActionAgentLogs:      CapAgentLogs,
	ActionAgentRestart:   CapAgentRestart,
	ActionContainersList: CapContainers,
	ActionDiskUsage:      CapDiskUsage,
	ActionCleanupPreview: CapCleanup,
	ActionCleanupApply:   CapCleanup,
	ActionDiagnostics:    CapDiagnostics,
}

func RequiredCap(action string) string {
	return requiredCaps[action]
}

var RelayCaps = []string{CapStateSnapshot, CapTasks, CapNodeEvents}

type WelcomeMessage struct {
	Type       string   `json:"type"`
	Proto      int      `json:"proto"`
	Caps       []string `json:"caps"`
	ServerTime string   `json:"server_time"`
}

type SnapshotServer struct {
	ServerID string `json:"server_id"`
	Status   string `json:"status"`
	Error    string `json:"error,omitempty"`
}

type StateSnapshotMessage struct {
	Type    string           `json:"type"`
	Servers []SnapshotServer `json:"servers"`
}

type TaskProgressMessage struct {
	Type      string         `json:"type"`
	CommandID string         `json:"command_id"`
	Progress  map[string]any `json:"progress"`
}

type NodeEventMessage struct {
	Type  string         `json:"type"`
	Kind  string         `json:"kind"`
	Level string         `json:"level,omitempty"`
	Data  map[string]any `json:"data,omitempty"`
}
