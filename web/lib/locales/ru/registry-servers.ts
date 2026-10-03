export const registryServers = {
  "admin.settings.reg.group.servers":
    "Значения по умолчанию для серверов, лимиты на аккаунт и сервер, вайпы и резервные копии. Значения применяются сразу, без перезапуска.",

  "admin.settings.reg.section.servers.defaults": "Значения по умолчанию",
  "admin.settings.reg.section.servers.limits": "Лимиты",
  "admin.settings.reg.section.servers.limits.description":
    "0 означает «без ограничения».",
  "admin.settings.reg.section.servers.wipes": "Вайпы",
  "admin.settings.reg.section.servers.backups": "Резервные копии",

  "admin.settings.reg.servers.port_min": "Порты: начало диапазона",
  "admin.settings.reg.servers.port_min.hint":
    "Для игр без своего диапазона. Меняйте синхронно с правилами файрвола на нодах",
  "admin.settings.reg.servers.port_max": "Порты: конец диапазона",
  "admin.settings.reg.servers.default_memory_mb": "Память по умолчанию",
  "admin.settings.reg.servers.default_memory_mb.hint":
    "Если у игры нет рекомендованного объёма",
  "admin.settings.reg.servers.default_cpu_millis": "Процессор по умолчанию",
  "admin.settings.reg.servers.default_cpu_millis.hint":
    "В тысячных долях ядра: 500 — это 0,5 ядра",
  "admin.settings.reg.servers.ftp_account_limit": "FTP-аккаунтов на сервер",
  "admin.settings.reg.servers.ftp_account_limit.hint": "Если в тарифе не задано своё значение",
  "admin.settings.reg.servers.password_length": "Длина паролей FTP и MySQL",
  "admin.settings.reg.servers.logs_tail_default": "Строк логов по умолчанию",
  "admin.settings.reg.servers.logs_tail_max": "Строк логов максимум",
  "admin.settings.reg.servers.logs_tail_max.hint": "Агент отдаёт не больше 1000 строк",

  "admin.settings.reg.servers.max_per_user": "Серверов на аккаунт",
  "admin.settings.reg.servers.max_projects": "Проектов на аккаунт",
  "admin.settings.reg.servers.max_cron_jobs": "Заданий cron на сервер",
  "admin.settings.reg.servers.cron_command_max": "Длина команды cron",
  "admin.settings.reg.servers.max_firewall_rules": "Правил файрвола на сервер",
  "admin.settings.reg.servers.max_friends": "Друзей на сервер",
  "admin.settings.reg.servers.max_manual_backups": "Ручных копий на сервер",

  "admin.settings.reg.servers.wipe_max_plans": "Планов вайпа на сервер",
  "admin.settings.reg.servers.wipe_run_timeout_min": "Время на один вайп",
  "admin.settings.reg.servers.wipe_run_timeout_min.hint": "По истечении вайп считается неудавшимся",

  "admin.settings.reg.servers.backup_keep_default": "Хранить копий по расписанию",
  "admin.settings.reg.servers.backup_keep_max": "Максимум копий в расписании",
  "admin.settings.reg.servers.backup_wait_min": "Ожидание создания копии",
  "admin.settings.reg.servers.restore_wait_min": "Ожидание восстановления копии",
  "admin.settings.reg.servers.offsite_timeout_hours": "Загрузка копии во внешнее хранилище",
  "admin.settings.reg.servers.offsite_keep_default": "Копий во внешнем хранилище",
  "admin.settings.reg.servers.offsite_keep_default.hint":
    "Если в расписании сервера число копий не задано",
};
