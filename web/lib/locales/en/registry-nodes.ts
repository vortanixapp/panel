export const registryNodes = {
  "admin.settings.reg.group.nodes":
    "When a node counts as unreachable, disk thresholds, agent updates and the agents' own parameters. Agent settings are pushed automatically (within 15 seconds) to agents of newer versions; older agents keep working with their own values.",

  "admin.settings.reg.section.nodes.presence": "Node availability",
  "admin.settings.reg.section.nodes.updates": "Agent updates",
  "admin.settings.reg.section.nodes.agent": "Agent behaviour",
  "admin.settings.reg.section.nodes.diagnostics": "Disk diagnostics",
  "admin.settings.reg.section.nodes.diagnostics.description":
    "Thresholds at which the node disk check reports a warning or an error.",

  "admin.settings.reg.nodes.offline_after_sec": "Node is unreachable after silence of",
  "admin.settings.reg.nodes.offline_after_sec.hint":
    "At least three agent heartbeat intervals",
  "admin.settings.reg.nodes.offline_notify_min": "Notify about unavailability after",
  "admin.settings.reg.nodes.disk_low_percent": "Low disk when free space is below",
  "admin.settings.reg.nodes.bulk_delay_sec": "Pause between bulk node commands",
  "admin.settings.reg.nodes.agent_update_timeout_min": "Time for an agent update",
  "admin.settings.reg.nodes.agent_outdated_grace_min": "Do not warn about outdated agents after a release for",

  "admin.settings.reg.agent.heartbeat_sec": "Agent heartbeat every",
  "admin.settings.reg.agent.heartbeat_sec.hint": "At least every 30 seconds, otherwise the node will flicker offline",
  "admin.settings.reg.agent.metrics_sec": "Server metrics every",
  "admin.settings.reg.agent.metrics_sec.hint": "The “running” status in the panel lasts 45 seconds",
  "admin.settings.reg.agent.timeout_scale_percent": "Agent operation time allowance",
  "admin.settings.reg.agent.timeout_scale_percent.hint": "100 keeps the standard timeouts, 200 doubles them",
  "admin.settings.reg.agent.install_timeout_min": "Time to install a server",
  "admin.settings.reg.agent.stop_timeout_sec": "Time for a graceful server stop",
  "admin.settings.reg.agent.stop_timeout_sec.hint": "0 keeps the Docker default (10 seconds)",
  "admin.settings.reg.agent.pids_limit": "Process limit per container",
  "admin.settings.reg.agent.pids_limit.hint":
    "Protects against fork bombs. The VORTANIX_PIDS_LIMIT variable on the node takes precedence",
  "admin.settings.reg.agent.cron_job_timeout_min": "Time for a server cron job",
  "admin.settings.reg.agent.log_max_size_mb": "Server log file size",
  "admin.settings.reg.agent.log_max_size_mb.hint":
    "Docker keeps game server logs in files and rotates them. Applies to servers created or reinstalled after the change",
  "admin.settings.reg.agent.log_max_files": "Log files per server",
  "admin.settings.reg.agent.temp_file_age_min": "Temporary files are removed when older than",

  "admin.settings.reg.agent.disk_warn_free_mb": "Warning: free space below",
  "admin.settings.reg.agent.disk_fail_free_mb": "Error: free space below",
  "admin.settings.reg.agent.disk_warn_used_percent": "Warning: used above",
  "admin.settings.reg.agent.disk_fail_used_percent": "Error: used above",
};
