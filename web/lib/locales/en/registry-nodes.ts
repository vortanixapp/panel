export const registryNodes = {
  "admin.settings.reg.group.nodes":
    "When a node counts as unreachable, disk thresholds, agent updates and the agents' own parameters. Agent settings are pushed automatically (within 15 seconds) to agents of newer versions; older agents keep working with their own values.",

  "admin.settings.reg.section.nodes.presence": "Node availability",
  "admin.settings.reg.section.nodes.updates": "Agent updates",
  "admin.settings.reg.section.nodes.agent": "Agent behaviour",
  "admin.settings.reg.section.nodes.performance": "Server speed and isolation",
  "admin.settings.reg.section.nodes.performance.description":
    "The install cache on a node speeds up creating servers, and CPU weights with a soft memory limit keep one server from hurting its neighbours. Applies to servers started after the change.",
  "admin.settings.reg.section.nodes.sandbox": "gVisor sandbox",
  "admin.settings.reg.section.nodes.sandbox.description":
    "gVisor runs a game server behind its own user-space kernel, so a vulnerability in the game cannot reach the host. The node needs runsc installed; network and disk are slower, so check the games one by one. Applies to servers started after the change. The VORTANIX_SANDBOX variable on the node takes priority.",
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

  "admin.settings.reg.agent.install_cache": "Game install cache on the node",
  "admin.settings.reg.agent.install_cache.hint":
    "Steam game files and archives are downloaded to the node once and new servers are copied from the cache (almost no extra space on filesystems with reflink)",
  "admin.settings.reg.agent.install_cache_ttl_hours": "Refresh a game cache no more often than every",
  "admin.settings.reg.agent.install_cache_max_mb": "Install cache size on the node",
  "admin.settings.reg.agent.install_cache_max_mb.hint":
    "When exceeded, the least recently used games are removed",
  "admin.settings.reg.agent.sandbox_mode": "Sandbox mode",
  "admin.settings.reg.agent.sandbox_mode.hint": "Off, only the games in the list, or every game except the list",
  "admin.settings.reg.agent.sandbox_mode.option.off": "Off",
  "admin.settings.reg.agent.sandbox_mode.option.selected": "Only games in the list",
  "admin.settings.reg.agent.sandbox_mode.option.all": "All games except the list",
  "admin.settings.reg.agent.sandbox_games": "Game list",
  "admin.settings.reg.agent.sandbox_games.hint":
    "Game codes separated by commas, for example mc-java, samp. In the \"only the list\" mode these games run under gVisor, in the \"all\" mode they are the exceptions",
  "admin.settings.reg.agent.sandbox_runtime": "Docker runtime name",
  "admin.settings.reg.agent.sandbox_runtime.hint": "How the runtime is named in daemon.json on the node, usually runsc",
  "admin.settings.reg.agent.sandbox_strict": "Refuse to start without the sandbox",
  "admin.settings.reg.agent.sandbox_strict.hint":
    "If the node has no gVisor, the server will not start. Without this option it starts without the sandbox and the agent log shows a warning",
  "admin.settings.reg.agent.cpu_fair_share": "Share CPU by memory size",
  "admin.settings.reg.agent.cpu_fair_share.hint":
    "When the processor is busy, a server with more memory gets more time. Nothing changes without load",
  "admin.settings.reg.agent.memory_reserve_percent": "Soft memory limit",
  "admin.settings.reg.agent.memory_reserve_percent.hint":
    "Share of the server limit. When the node runs short of memory, the system reclaims it from servers above this share first. 0 turns it off",

  "admin.settings.reg.agent.disk_warn_free_mb": "Warning: free space below",
  "admin.settings.reg.agent.disk_fail_free_mb": "Error: free space below",
  "admin.settings.reg.agent.disk_warn_used_percent": "Warning: used above",
  "admin.settings.reg.agent.disk_fail_used_percent": "Error: used above",
};
