export const registryRetention = {
  "admin.settings.reg.group.retention":
    "How long metrics, notifications and service history are kept. Old records are removed automatically every few hours.",

  "admin.settings.reg.section.retention.metrics": "Node metrics and events",
  "admin.settings.reg.section.retention.messages": "Notifications and email",
  "admin.settings.reg.section.retention.history": "Service history",
  "admin.settings.reg.section.retention.history.description":
    "0 keeps records forever. Deleted records cannot be restored; check legal requirements for the audit log and charges.",

  "admin.settings.reg.retention.server_metrics_days": "Server metrics: detailed points",
  "admin.settings.reg.retention.server_metrics_days.hint":
    "Points taken every few seconds, used for 24-hour charts. Older data is kept only as 5-minute averages. With TimescaleDB enabled the retention policy is updated automatically",
  "admin.settings.reg.retention.server_metrics_rollup_days": "Server metrics: 5-minute averages",
  "admin.settings.reg.retention.server_metrics_rollup_days.hint":
    "Used for week, month and quarter charts. Not used with TimescaleDB, which has its own retention",
  "admin.settings.reg.retention.node_metrics_days": "Node metrics",
  "admin.settings.reg.retention.node_events_days": "Node events",
  "admin.settings.reg.retention.node_tasks_days": "Finished node tasks",
  "admin.settings.reg.retention.online_points_days": "Monitoring online history",
  "admin.settings.reg.retention.daemon_pull_days": "Node sweep tasks",
  "admin.settings.reg.retention.daemon_action_days": "Daemon action tasks",

  "admin.settings.reg.retention.notifications_days": "Notifications",
  "admin.settings.reg.retention.notifications_read_days": "Read notifications",
  "admin.settings.reg.retention.deliveries_days": "Notification delivery log",
  "admin.settings.reg.retention.mail_log_days": "Email log",

  "admin.settings.reg.retention.login_attempts_days": "Sign-in attempts",
  "admin.settings.reg.retention.audit_logs_days": "Audit log",
  "admin.settings.reg.retention.sessions_days": "Inactive sessions",
  "admin.settings.reg.retention.sessions_days.hint":
    "Do not set it below the “remember me” session lifetime",
  "admin.settings.reg.retention.reset_tokens_days": "Password reset links after expiry",
  "admin.settings.reg.retention.sso_tokens_days": "WHMCS sign-in tokens after expiry",
  "admin.settings.reg.retention.webhook_deliveries_days": "Webhook deliveries",
  "admin.settings.reg.retention.hourly_charges_days": "Hourly charges",
};
