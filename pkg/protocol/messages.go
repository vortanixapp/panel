package protocol

const (
	MsgHello           = "hello"
	MsgHeartbeat       = "heartbeat"
	MsgServerStatus    = "server_status"
	MsgAck             = "ack"
	MsgMetrics         = "metrics"
	MsgConsoleOutput   = "console_output"
	MsgInstallProgress = "install_progress"
)

const (
	MsgCommand             = "command"
	ActionConsole          = "console_attach"
	ActionConsoleIn        = "console_input"
	ActionCronSync         = "cron_sync"
	ActionFirewallSync     = "firewall_sync"
	ActionPortsSync        = "ports_sync"
	ActionFilesWriteBinary = "files_write_binary"
)

type HelloMessage struct {
	Type   string `json:"type"`
	NodeID string `json:"node_id"`
}

type HeartbeatMessage struct {
	Type   string         `json:"type"`
	NodeID string         `json:"node_id"`
	Meta   map[string]any `json:"meta,omitempty"`
}

type ServerStatusMessage struct {
	Type     string `json:"type"`
	ServerID string `json:"server_id"`
	Status   string `json:"status"`
	Error    string `json:"error,omitempty"`
}

type InstallProgressMessage struct {
	Type     string `json:"type"`
	ServerID string `json:"server_id"`
	Stage    string `json:"stage"`
	Percent  int    `json:"percent"`
	Message  string `json:"message,omitempty"`
	Line     string `json:"line,omitempty"`
}

type MetricsMessage struct {
	Type       string  `json:"type"`
	ServerID   string  `json:"server_id"`
	CPUPct     float64 `json:"cpu_pct"`
	MemUsedMB  int     `json:"mem_used_mb"`
	MemLimitMB int     `json:"mem_limit_mb"`
}

type ConsoleOutputMessage struct {
	Type      string `json:"type"`
	SessionID string `json:"session_id"`
	ServerID  string `json:"server_id"`
	Data      string `json:"data"`
}

type AckMessage struct {
	Type      string         `json:"type"`
	CommandID string         `json:"command_id"`
	OK        bool           `json:"ok"`
	Error     string         `json:"error,omitempty"`
	Result    map[string]any `json:"result,omitempty"`
}

type CommandMessage struct {
	Type     string         `json:"type"`
	ID       string         `json:"id"`
	Action   string         `json:"action"`
	ServerID string         `json:"server_id"`
	Payload  map[string]any `json:"payload"`
}

type InternalCommandRequest struct {
	CommandID string         `json:"command_id"`
	Action    string         `json:"action"`
	ServerID  string         `json:"server_id"`
	Payload   map[string]any `json:"payload"`
}

type TenantEvent struct {
	Type     string         `json:"type"`
	ServerID string         `json:"server_id,omitempty"`
	NodeID   string         `json:"node_id,omitempty"`
	Status   string         `json:"status,omitempty"`
	Metrics  map[string]any `json:"metrics,omitempty"`

	Percent *int   `json:"percent,omitempty"`
	Message string `json:"message,omitempty"`
}

type ConsoleTicket struct {
	ServerID string `json:"server_id"`
	NodeID   string `json:"node_id"`
	UserID   string `json:"user_id"`
}

func TenantEventsChannel() string {
	return "panel:events"
}

func ConsoleChannel(sessionID string) string {
	return "console:" + sessionID
}
