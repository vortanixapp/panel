package settingsreg

import (
	"fmt"
	"strconv"
)

const (
	GroupServers = "servers"
)

func init() {
	addCheck(func(v map[string]string) error {
		lower, _ := strconv.ParseInt(v[ServersPortMin.Key], 10, 64)
		upper, _ := strconv.ParseInt(v[ServersPortMax.Key], 10, 64)
		if lower >= upper {
			return fmt.Errorf("начало диапазона портов должно быть меньше конца")
		}
		return nil
	})
}

var (
	ServersPortMin = def(Setting{
		Key: "servers.port_min", Group: GroupServers, Section: "defaults",
		Kind: KindInt, Default: "27000", Min: 1024, Max: 65535,
	})
	ServersPortMax = def(Setting{
		Key: "servers.port_max", Group: GroupServers, Section: "defaults",
		Kind: KindInt, Default: "28999", Min: 1024, Max: 65535,
	})
	ServersDefaultMemory = def(Setting{
		Key: "servers.default_memory_mb", Group: GroupServers, Section: "defaults",
		Kind: KindInt, Default: "512", Min: 128, Max: 262144, Unit: UnitMB,
	})
	ServersDefaultCPUMillis = def(Setting{
		Key: "servers.default_cpu_millis", Group: GroupServers, Section: "defaults",
		Kind: KindInt, Default: "500", Min: 100, Max: 64000,
	})
	ServersFTPLimit = def(Setting{
		Key: "servers.ftp_account_limit", Group: GroupServers, Section: "defaults",
		Kind: KindInt, Default: "1", Min: 1, Max: 50, Unit: UnitCount,
	})
	ServersPasswordLength = def(Setting{
		Key: "servers.password_length", Group: GroupServers, Section: "defaults",
		Kind: KindInt, Default: "16", Min: 8, Max: 64, Unit: UnitCount,
	})
	ServersLogsTail = def(Setting{
		Key: "servers.logs_tail_default", Group: GroupServers, Section: "defaults",
		Kind: KindInt, Default: "200", Min: 10, Max: 1000, Unit: UnitCount,
	})
	ServersLogsTailMax = def(Setting{
		Key: "servers.logs_tail_max", Group: GroupServers, Section: "defaults",
		Kind: KindInt, Default: "1000", Min: 100, Max: 1000, Unit: UnitCount,
	})

	ServersMaxPerUser = def(Setting{
		Key: "servers.max_per_user", Group: GroupServers, Section: "limits",
		Kind: KindInt, Default: "0", Min: 0, Max: 10000, Unit: UnitCount,
	})
	ServersMaxProjects = def(Setting{
		Key: "servers.max_projects", Group: GroupServers, Section: "limits",
		Kind: KindInt, Default: "30", Min: 1, Max: 1000, Unit: UnitCount,
	})
	ServersMaxCronJobs = def(Setting{
		Key: "servers.max_cron_jobs", Group: GroupServers, Section: "limits",
		Kind: KindInt, Default: "0", Min: 0, Max: 1000, Unit: UnitCount,
	})
	ServersCronCommandMax = def(Setting{
		Key: "servers.cron_command_max", Group: GroupServers, Section: "limits",
		Kind: KindInt, Default: "2000", Min: 100, Max: 2000, Unit: UnitCount,
	})
	ServersMaxFirewallRules = def(Setting{
		Key: "servers.max_firewall_rules", Group: GroupServers, Section: "limits",
		Kind: KindInt, Default: "0", Min: 0, Max: 1000, Unit: UnitCount,
	})
	ServersMaxFriends = def(Setting{
		Key: "servers.max_friends", Group: GroupServers, Section: "limits",
		Kind: KindInt, Default: "0", Min: 0, Max: 1000, Unit: UnitCount,
	})
	ServersMaxManualBackups = def(Setting{
		Key: "servers.max_manual_backups", Group: GroupServers, Section: "limits",
		Kind: KindInt, Default: "0", Min: 0, Max: 1000, Unit: UnitCount,
	})

	ServersWipeMaxPlans = def(Setting{
		Key: "servers.wipe_max_plans", Group: GroupServers, Section: "wipes",
		Kind: KindInt, Default: "10", Min: 1, Max: 50, Unit: UnitCount,
	})
	ServersWipeRunTimeout = def(Setting{
		Key: "servers.wipe_run_timeout_min", Group: GroupServers, Section: "wipes",
		Kind: KindInt, Default: "90", Min: 10, Max: 720, Unit: UnitMin,
	})

	ServersBackupKeepDefault = def(Setting{
		Key: "servers.backup_keep_default", Group: GroupServers, Section: "backups",
		Kind: KindInt, Default: "7", Min: 1, Max: 365, Unit: UnitCount,
	})
	ServersBackupKeepMax = def(Setting{
		Key: "servers.backup_keep_max", Group: GroupServers, Section: "backups",
		Kind: KindInt, Default: "30", Min: 1, Max: 365, Unit: UnitCount,
	})
	ServersBackupWait = def(Setting{
		Key: "servers.backup_wait_min", Group: GroupServers, Section: "backups",
		Kind: KindInt, Default: "20", Min: 1, Max: 360, Unit: UnitMin,
	})
	ServersRestoreWait = def(Setting{
		Key: "servers.restore_wait_min", Group: GroupServers, Section: "backups",
		Kind: KindInt, Default: "30", Min: 1, Max: 360, Unit: UnitMin,
	})
	ServersOffsiteTimeout = def(Setting{
		Key: "servers.offsite_timeout_hours", Group: GroupServers, Section: "backups",
		Kind: KindInt, Default: "3", Min: 1, Max: 24, Unit: UnitHour,
	})
	ServersOffsiteKeep = def(Setting{
		Key: "servers.offsite_keep_default", Group: GroupServers, Section: "backups",
		Kind: KindInt, Default: "7", Min: 1, Max: 365, Unit: UnitCount,
	})
)
