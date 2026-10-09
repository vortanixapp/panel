package settingsreg

const (
	GroupRetention = "retention"
)

var (
	RetentionServerMetrics = def(Setting{
		Key: "retention.server_metrics_days", Group: GroupRetention, Section: "metrics",
		Kind: KindInt, Default: "2", Min: 1, Max: 365, Unit: UnitDay,
	})
	RetentionServerMetricsRollup = def(Setting{
		Key: "retention.server_metrics_rollup_days", Group: GroupRetention, Section: "metrics",
		Kind: KindInt, Default: "90", Min: 1, Max: 730, Unit: UnitDay,
	})
	RetentionNodeMetrics = def(Setting{
		Key: "retention.node_metrics_days", Group: GroupRetention, Section: "metrics",
		Kind: KindInt, Default: "8", Min: 1, Max: 365, Unit: UnitDay,
	})
	RetentionNodeEvents = def(Setting{
		Key: "retention.node_events_days", Group: GroupRetention, Section: "metrics",
		Kind: KindInt, Default: "90", Min: 1, Max: 3650, Unit: UnitDay,
	})
	RetentionNodeTasks = def(Setting{
		Key: "retention.node_tasks_days", Group: GroupRetention, Section: "metrics",
		Kind: KindInt, Default: "30", Min: 1, Max: 3650, Unit: UnitDay,
	})
	RetentionOnlinePoints = def(Setting{
		Key: "retention.online_points_days", Group: GroupRetention, Section: "metrics",
		Kind: KindInt, Default: "45", Min: 1, Max: 365, Unit: UnitDay,
	})
	RetentionDaemonPull = def(Setting{
		Key: "retention.daemon_pull_days", Group: GroupRetention, Section: "metrics",
		Kind: KindInt, Default: "3", Min: 1, Max: 365, Unit: UnitDay,
	})
	RetentionDaemonAction = def(Setting{
		Key: "retention.daemon_action_days", Group: GroupRetention, Section: "metrics",
		Kind: KindInt, Default: "30", Min: 1, Max: 3650, Unit: UnitDay,
	})

	RetentionNotifications = def(Setting{
		Key: "retention.notifications_days", Group: GroupRetention, Section: "messages",
		Kind: KindInt, Default: "180", Min: 7, Max: 3650, Unit: UnitDay,
	})
	RetentionNotificationsRead = def(Setting{
		Key: "retention.notifications_read_days", Group: GroupRetention, Section: "messages",
		Kind: KindInt, Default: "90", Min: 1, Max: 3650, Unit: UnitDay,
	})
	RetentionDeliveries = def(Setting{
		Key: "retention.deliveries_days", Group: GroupRetention, Section: "messages",
		Kind: KindInt, Default: "30", Min: 1, Max: 3650, Unit: UnitDay,
	})
	RetentionMailLog = def(Setting{
		Key: "retention.mail_log_days", Group: GroupRetention, Section: "messages",
		Kind: KindInt, Default: "90", Min: 1, Max: 3650, Unit: UnitDay,
	})

	RetentionLoginAttempts = def(Setting{
		Key: "retention.login_attempts_days", Group: GroupRetention, Section: "history",
		Kind: KindInt, Default: "0", Min: 0, Max: 3650, Unit: UnitDay,
	})
	RetentionAuditLogs = def(Setting{
		Key: "retention.audit_logs_days", Group: GroupRetention, Section: "history",
		Kind: KindInt, Default: "0", Min: 0, Max: 3650, Unit: UnitDay,
	})
	RetentionSessions = def(Setting{
		Key: "retention.sessions_days", Group: GroupRetention, Section: "history",
		Kind: KindInt, Default: "0", Min: 0, Max: 3650, Unit: UnitDay,
	})
	RetentionResetTokens = def(Setting{
		Key: "retention.reset_tokens_days", Group: GroupRetention, Section: "history",
		Kind: KindInt, Default: "0", Min: 0, Max: 3650, Unit: UnitDay,
	})
	RetentionSSOTokens = def(Setting{
		Key: "retention.sso_tokens_days", Group: GroupRetention, Section: "history",
		Kind: KindInt, Default: "0", Min: 0, Max: 3650, Unit: UnitDay,
	})
	RetentionWebhookDeliveries = def(Setting{
		Key: "retention.webhook_deliveries_days", Group: GroupRetention, Section: "history",
		Kind: KindInt, Default: "0", Min: 0, Max: 3650, Unit: UnitDay,
	})
	RetentionHourlyCharges = def(Setting{
		Key: "retention.hourly_charges_days", Group: GroupRetention, Section: "history",
		Kind: KindInt, Default: "0", Min: 0, Max: 3650, Unit: UnitDay,
	})
)
