export const registryServers = {
  "admin.settings.reg.group.servers":
    "Server defaults, per-account and per-server limits, wipes and backups. Values apply immediately, no restart needed.",

  "admin.settings.reg.section.servers.defaults": "Defaults",
  "admin.settings.reg.section.servers.limits": "Limits",
  "admin.settings.reg.section.servers.limits.description": "0 means no limit.",
  "admin.settings.reg.section.servers.wipes": "Wipes",
  "admin.settings.reg.section.servers.backups": "Backups",

  "admin.settings.reg.servers.port_min": "Ports: range start",
  "admin.settings.reg.servers.port_min.hint":
    "For games without their own range. Change together with the firewall rules on the nodes",
  "admin.settings.reg.servers.port_max": "Ports: range end",
  "admin.settings.reg.servers.default_memory_mb": "Default memory",
  "admin.settings.reg.servers.default_memory_mb.hint":
    "Used when the game has no recommended amount",
  "admin.settings.reg.servers.default_cpu_millis": "Default CPU",
  "admin.settings.reg.servers.default_cpu_millis.hint":
    "In thousandths of a core: 500 is 0.5 core",
  "admin.settings.reg.servers.ftp_account_limit": "FTP accounts per server",
  "admin.settings.reg.servers.ftp_account_limit.hint": "Used when the plan has no value of its own",
  "admin.settings.reg.servers.password_length": "FTP and MySQL password length",
  "admin.settings.reg.servers.logs_tail_default": "Default log lines",
  "admin.settings.reg.servers.logs_tail_max": "Maximum log lines",
  "admin.settings.reg.servers.logs_tail_max.hint": "The agent returns at most 1000 lines",

  "admin.settings.reg.servers.max_per_user": "Servers per account",
  "admin.settings.reg.servers.max_projects": "Projects per account",
  "admin.settings.reg.servers.max_cron_jobs": "Cron jobs per server",
  "admin.settings.reg.servers.cron_command_max": "Cron command length",
  "admin.settings.reg.servers.max_firewall_rules": "Firewall rules per server",
  "admin.settings.reg.servers.max_friends": "Friends per server",
  "admin.settings.reg.servers.max_manual_backups": "Manual backups per server",

  "admin.settings.reg.servers.wipe_max_plans": "Wipe plans per server",
  "admin.settings.reg.servers.wipe_run_timeout_min": "Time for one wipe",
  "admin.settings.reg.servers.wipe_run_timeout_min.hint": "After it a wipe counts as failed",

  "admin.settings.reg.servers.backup_keep_default": "Scheduled copies to keep",
  "admin.settings.reg.servers.backup_keep_max": "Maximum copies in a schedule",
  "admin.settings.reg.servers.backup_wait_min": "Wait for a backup to complete",
  "admin.settings.reg.servers.restore_wait_min": "Wait for a restore to complete",
  "admin.settings.reg.servers.offsite_timeout_hours": "Upload to external storage",
  "admin.settings.reg.servers.offsite_keep_default": "Copies in external storage",
  "admin.settings.reg.servers.offsite_keep_default.hint":
    "Used when the server schedule has no count",
};
