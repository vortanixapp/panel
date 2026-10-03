export const registryNotifications = {
  "admin.settings.reg.group.notifications":
    "Повторы доставки, тихие часы по умолчанию, Telegram-бот, рассылки и вебхуки. Значения применяются сразу, без перезапуска.",

  "admin.settings.reg.section.notifications.delivery": "Доставка уведомлений",
  "admin.settings.reg.section.notifications.telegram": "Telegram",
  "admin.settings.reg.section.notifications.mail": "Рассылки и почта",
  "admin.settings.reg.section.notifications.webhooks": "Вебхуки",

  "admin.settings.reg.notify.max_attempts": "Попыток доставки",
  "admin.settings.reg.notify.backoff_base_min": "Пауза перед первым повтором",
  "admin.settings.reg.notify.backoff_base_min.hint": "Дальше пауза удваивается до максимума",
  "admin.settings.reg.notify.backoff_max_min": "Максимальная пауза между повторами",
  "admin.settings.reg.notify.quiet_from_hour": "Тихие часы по умолчанию: с",
  "admin.settings.reg.notify.quiet_from_hour.hint":
    "Для пользователей, которые не задали своё время. Сами тихие часы они включают в профиле",
  "admin.settings.reg.notify.quiet_to_hour": "Тихие часы по умолчанию: до",

  "admin.settings.reg.notify.telegram_link_ttl_min": "Код привязки Telegram действует",
  "admin.settings.reg.notify.test_gap_sec": "Тестовое уведомление не чаще, чем раз в",
  "admin.settings.reg.notify.bot_actions_per_min": "Действий бота в минуту на пользователя",

  "admin.settings.reg.notify.mailing_per_minute": "Писем рассылки в минуту",
  "admin.settings.reg.notify.mailing_per_minute.hint":
    "0 — без ограничения. Ограничьте, если почтовый сервер блокирует массовую отправку",
  "admin.settings.reg.notify.mailing_attempts": "Попыток запуска рассылки",
  "admin.settings.reg.notify.mailing_send_timeout_sec": "Время на отправку одного письма",

  "admin.settings.reg.notify.webhook_timeout_sec": "Время на ответ вебхука",
  "admin.settings.reg.notify.webhook_attempts": "Попыток доставки вебхука",
};
