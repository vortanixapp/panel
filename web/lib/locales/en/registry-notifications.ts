export const registryNotifications = {
  "admin.settings.reg.group.notifications":
    "Delivery retries, default quiet hours, the Telegram bot, mailings and webhooks. Values apply immediately, no restart needed.",

  "admin.settings.reg.section.notifications.delivery": "Notification delivery",
  "admin.settings.reg.section.notifications.telegram": "Telegram",
  "admin.settings.reg.section.notifications.mail": "Mailings and email",
  "admin.settings.reg.section.notifications.webhooks": "Webhooks",

  "admin.settings.reg.notify.max_attempts": "Delivery attempts",
  "admin.settings.reg.notify.backoff_base_min": "Pause before the first retry",
  "admin.settings.reg.notify.backoff_base_min.hint": "The pause then doubles up to the maximum",
  "admin.settings.reg.notify.backoff_max_min": "Maximum pause between retries",
  "admin.settings.reg.notify.quiet_from_hour": "Default quiet hours: from",
  "admin.settings.reg.notify.quiet_from_hour.hint":
    "For users who have not set their own time. They turn quiet hours on in their profile",
  "admin.settings.reg.notify.quiet_to_hour": "Default quiet hours: until",

  "admin.settings.reg.notify.telegram_link_ttl_min": "Telegram link code is valid for",
  "admin.settings.reg.notify.test_gap_sec": "Test notification no more often than every",
  "admin.settings.reg.notify.bot_actions_per_min": "Bot actions per minute per user",

  "admin.settings.reg.notify.mailing_per_minute": "Mailing emails per minute",
  "admin.settings.reg.notify.mailing_per_minute.hint":
    "0 means no limit. Set one if your mail server blocks bulk sending",
  "admin.settings.reg.notify.mailing_attempts": "Mailing start attempts",
  "admin.settings.reg.notify.mailing_send_timeout_sec": "Time to send one email",

  "admin.settings.reg.notify.webhook_timeout_sec": "Webhook response timeout",
  "admin.settings.reg.notify.webhook_attempts": "Webhook delivery attempts",
};
