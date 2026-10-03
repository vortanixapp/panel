package settingsreg

const (
	GroupNotifications = "notifications"
)

var (
	NotifyMaxAttempts = def(Setting{
		Key: "notify.max_attempts", Group: GroupNotifications, Section: "delivery",
		Kind: KindInt, Default: "5", Min: 1, Max: 20, Unit: UnitCount,
	})
	NotifyBackoffBase = def(Setting{
		Key: "notify.backoff_base_min", Group: GroupNotifications, Section: "delivery",
		Kind: KindInt, Default: "1", Min: 1, Max: 60, Unit: UnitMin,
	})
	NotifyBackoffMax = def(Setting{
		Key: "notify.backoff_max_min", Group: GroupNotifications, Section: "delivery",
		Kind: KindInt, Default: "30", Min: 1, Max: 1440, Unit: UnitMin,
	})
	NotifyQuietFromHour = def(Setting{
		Key: "notify.quiet_from_hour", Group: GroupNotifications, Section: "delivery",
		Kind: KindInt, Default: "23", Min: 0, Max: 23, Unit: UnitHour,
	})
	NotifyQuietToHour = def(Setting{
		Key: "notify.quiet_to_hour", Group: GroupNotifications, Section: "delivery",
		Kind: KindInt, Default: "8", Min: 0, Max: 23, Unit: UnitHour,
	})

	NotifyTelegramLinkTTL = def(Setting{
		Key: "notify.telegram_link_ttl_min", Group: GroupNotifications, Section: "telegram",
		Kind: KindInt, Default: "10", Min: 1, Max: 60, Unit: UnitMin,
	})
	NotifyTestGap = def(Setting{
		Key: "notify.test_gap_sec", Group: GroupNotifications, Section: "telegram",
		Kind: KindInt, Default: "30", Min: 5, Max: 600, Unit: UnitSec,
	})
	NotifyBotActionsPerMin = def(Setting{
		Key: "notify.bot_actions_per_min", Group: GroupNotifications, Section: "telegram",
		Kind: KindInt, Default: "40", Min: 5, Max: 600, Unit: UnitCount,
	})

	NotifyMailingPerMinute = def(Setting{
		Key: "notify.mailing_per_minute", Group: GroupNotifications, Section: "mail",
		Kind: KindInt, Default: "0", Min: 0, Max: 6000, Unit: UnitCount,
	})
	NotifyMailingAttempts = def(Setting{
		Key: "notify.mailing_attempts", Group: GroupNotifications, Section: "mail",
		Kind: KindInt, Default: "5", Min: 1, Max: 20, Unit: UnitCount,
	})
	NotifyMailingSendTimeout = def(Setting{
		Key: "notify.mailing_send_timeout_sec", Group: GroupNotifications, Section: "mail",
		Kind: KindInt, Default: "30", Min: 5, Max: 300, Unit: UnitSec,
	})

	NotifyWebhookTimeout = def(Setting{
		Key: "notify.webhook_timeout_sec", Group: GroupNotifications, Section: "webhooks",
		Kind: KindInt, Default: "10", Min: 2, Max: 60, Unit: UnitSec,
	})
	NotifyWebhookAttempts = def(Setting{
		Key: "notify.webhook_attempts", Group: GroupNotifications, Section: "webhooks",
		Kind: KindInt, Default: "6", Min: 1, Max: 20, Unit: UnitCount,
	})
)
