export const registryRetention = {
  "admin.settings.reg.group.retention":
    "Сколько хранятся метрики, уведомления и служебная история. Старые записи удаляются автоматически раз в несколько часов.",

  "admin.settings.reg.section.retention.metrics": "Метрики и события узлов",
  "admin.settings.reg.section.retention.messages": "Уведомления и письма",
  "admin.settings.reg.section.retention.history": "Служебная история",
  "admin.settings.reg.section.retention.history.description":
    "0 — хранить без ограничения. Удалённые записи восстановить нельзя; для журнала аудита и списаний проверьте требования закона.",

  "admin.settings.reg.retention.server_metrics_days": "Метрики серверов: подробные точки",
  "admin.settings.reg.retention.server_metrics_days.hint":
    "Точки раз в несколько секунд, нужны для графиков за сутки. Старше этого срока остаются только усреднённые за 5 минут. Если включён TimescaleDB, политика хранения обновляется автоматически",
  "admin.settings.reg.retention.server_metrics_rollup_days": "Метрики серверов: усреднённые за 5 минут",
  "admin.settings.reg.retention.server_metrics_rollup_days.hint":
    "Для графиков за неделю, месяц и квартал. Не работает вместе с TimescaleDB, там действует свой срок хранения",
  "admin.settings.reg.retention.node_metrics_days": "Метрики узлов",
  "admin.settings.reg.retention.node_events_days": "События узлов",
  "admin.settings.reg.retention.node_tasks_days": "Завершённые задачи узлов",
  "admin.settings.reg.retention.online_points_days": "История онлайна в мониторинге",
  "admin.settings.reg.retention.daemon_pull_days": "Задачи обхода нод",
  "admin.settings.reg.retention.daemon_action_days": "Задачи действий демона",

  "admin.settings.reg.retention.notifications_days": "Уведомления",
  "admin.settings.reg.retention.notifications_read_days": "Прочитанные уведомления",
  "admin.settings.reg.retention.deliveries_days": "Журнал доставки уведомлений",
  "admin.settings.reg.retention.mail_log_days": "Журнал писем",

  "admin.settings.reg.retention.login_attempts_days": "Попытки входа",
  "admin.settings.reg.retention.audit_logs_days": "Журнал аудита",
  "admin.settings.reg.retention.sessions_days": "Неактивные сессии",
  "admin.settings.reg.retention.sessions_days.hint":
    "Не ставьте меньше срока жизни сессии «запомнить меня»",
  "admin.settings.reg.retention.reset_tokens_days": "Ссылки сброса пароля после истечения",
  "admin.settings.reg.retention.sso_tokens_days": "Токены входа WHMCS после истечения",
  "admin.settings.reg.retention.webhook_deliveries_days": "Доставки вебхуков",
  "admin.settings.reg.retention.hourly_charges_days": "Почасовые списания",
};
