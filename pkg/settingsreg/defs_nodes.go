package settingsreg

import (
	"fmt"
	"strconv"
)

const (
	GroupNodes = "nodes"
)

func init() {
	addCheck(func(v map[string]string) error {
		beat, _ := strconv.ParseInt(v[AgentHeartbeat.Key], 10, 64)
		offline, _ := strconv.ParseInt(v[NodesOfflineAfter.Key], 10, 64)
		if offline < beat*3 {
			return fmt.Errorf("узел считается недоступным не раньше трёх пропущенных сигналов: не меньше %d сек", beat*3)
		}
		return nil
	})
}

var (
	NodesOfflineAfter = def(Setting{
		Key: "nodes.offline_after_sec", Group: GroupNodes, Section: "presence",
		Kind: KindInt, Default: "90", Min: 30, Max: 600, Unit: UnitSec,
	})
	NodesOfflineNotify = def(Setting{
		Key: "nodes.offline_notify_min", Group: GroupNodes, Section: "presence",
		Kind: KindInt, Default: "2", Min: 1, Max: 60, Unit: UnitMin,
	})
	NodesDiskLowPercent = def(Setting{
		Key: "nodes.disk_low_percent", Group: GroupNodes, Section: "presence",
		Kind: KindInt, Default: "10", Min: 1, Max: 50, Unit: UnitPercent,
	})
	NodesBulkDelay = def(Setting{
		Key: "nodes.bulk_delay_sec", Group: GroupNodes, Section: "presence",
		Kind: KindInt, Default: "2", Min: 0, Max: 60, Unit: UnitSec,
	})
	NodesAgentUpdateTimeout = def(Setting{
		Key: "nodes.agent_update_timeout_min", Group: GroupNodes, Section: "updates",
		Kind: KindInt, Default: "15", Min: 3, Max: 120, Unit: UnitMin,
	})
	NodesAgentOutdatedGrace = def(Setting{
		Key: "nodes.agent_outdated_grace_min", Group: GroupNodes, Section: "updates",
		Kind: KindInt, Default: "60", Min: 5, Max: 10080, Unit: UnitMin,
	})

	AgentHeartbeat = def(Setting{
		Key: "agent.heartbeat_sec", Group: GroupNodes, Section: "agent",
		Kind: KindInt, Default: "30", Min: 10, Max: 30, Unit: UnitSec, Agent: true,
	})
	AgentMetrics = def(Setting{
		Key: "agent.metrics_sec", Group: GroupNodes, Section: "agent",
		Kind: KindInt, Default: "15", Min: 5, Max: 15, Unit: UnitSec, Agent: true,
	})
	AgentTimeoutScale = def(Setting{
		Key: "agent.timeout_scale_percent", Group: GroupNodes, Section: "agent",
		Kind: KindInt, Default: "100", Min: 50, Max: 400, Unit: UnitPercent, Agent: true,
	})
	AgentInstallTimeout = def(Setting{
		Key: "agent.install_timeout_min", Group: GroupNodes, Section: "agent",
		Kind: KindInt, Default: "120", Min: 10, Max: 720, Unit: UnitMin, Agent: true,
	})
	AgentStopTimeout = def(Setting{
		Key: "agent.stop_timeout_sec", Group: GroupNodes, Section: "agent",
		Kind: KindInt, Default: "0", Min: 0, Max: 300, Unit: UnitSec, Agent: true,
	})
	AgentPidsLimit = def(Setting{
		Key: "agent.pids_limit", Group: GroupNodes, Section: "agent",
		Kind: KindInt, Default: "2048", Min: 256, Max: 32768, Unit: UnitCount, Agent: true,
	})
	AgentCronTimeout = def(Setting{
		Key: "agent.cron_job_timeout_min", Group: GroupNodes, Section: "agent",
		Kind: KindInt, Default: "10", Min: 1, Max: 120, Unit: UnitMin, Agent: true,
	})
	AgentLogMaxSizeMB = def(Setting{
		Key: "agent.log_max_size_mb", Group: GroupNodes, Section: "agent",
		Kind: KindInt, Default: "10", Min: 1, Max: 1024, Unit: UnitMB, Agent: true,
	})
	AgentLogMaxFiles = def(Setting{
		Key: "agent.log_max_files", Group: GroupNodes, Section: "agent",
		Kind: KindInt, Default: "3", Min: 1, Max: 20, Unit: UnitCount, Agent: true,
	})
	AgentTempFileAge = def(Setting{
		Key: "agent.temp_file_age_min", Group: GroupNodes, Section: "agent",
		Kind: KindInt, Default: "60", Min: 10, Max: 1440, Unit: UnitMin, Agent: true,
	})

	AgentInstallCache = def(Setting{
		Key: "agent.install_cache", Group: GroupNodes, Section: "performance",
		Kind: KindBool, Default: "0", Agent: true,
	})
	AgentInstallCacheTTL = def(Setting{
		Key: "agent.install_cache_ttl_hours", Group: GroupNodes, Section: "performance",
		Kind: KindInt, Default: "12", Min: 1, Max: 720, Unit: UnitHour, Agent: true,
	})
	AgentInstallCacheMaxMB = def(Setting{
		Key: "agent.install_cache_max_mb", Group: GroupNodes, Section: "performance",
		Kind: KindInt, Default: "204800", Min: 10240, Max: 10485760, Unit: UnitMB, Agent: true,
	})
	AgentCPUFairShare = def(Setting{
		Key: "agent.cpu_fair_share", Group: GroupNodes, Section: "performance",
		Kind: KindBool, Default: "0", Agent: true,
	})
	AgentMemoryReservePercent = def(Setting{
		Key: "agent.memory_reserve_percent", Group: GroupNodes, Section: "performance",
		Kind: KindInt, Default: "0", Min: 0, Max: 95, Unit: UnitPercent, Agent: true,
	})

	AgentDiskWarnMB = def(Setting{
		Key: "agent.disk_warn_free_mb", Group: GroupNodes, Section: "diagnostics",
		Kind: KindInt, Default: "10240", Min: 512, Max: 1048576, Unit: UnitMB, Agent: true,
	})
	AgentDiskFailMB = def(Setting{
		Key: "agent.disk_fail_free_mb", Group: GroupNodes, Section: "diagnostics",
		Kind: KindInt, Default: "2048", Min: 128, Max: 1048576, Unit: UnitMB, Agent: true,
	})
	AgentDiskWarnPercent = def(Setting{
		Key: "agent.disk_warn_used_percent", Group: GroupNodes, Section: "diagnostics",
		Kind: KindInt, Default: "90", Min: 50, Max: 99, Unit: UnitPercent, Agent: true,
	})
	AgentDiskFailPercent = def(Setting{
		Key: "agent.disk_fail_used_percent", Group: GroupNodes, Section: "diagnostics",
		Kind: KindInt, Default: "95", Min: 50, Max: 100, Unit: UnitPercent, Agent: true,
	})
)
