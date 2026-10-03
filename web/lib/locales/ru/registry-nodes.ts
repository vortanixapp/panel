export const registryNodes = {
  "admin.settings.reg.group.nodes":
    "Когда узел считается недоступным, пороги по диску, обновление агентов и параметры самих агентов. Настройки агентов рассылаются автоматически (до 15 секунд) агентам новых версий; старые агенты продолжают работать со своими значениями.",

  "admin.settings.reg.section.nodes.presence": "Доступность узлов",
  "admin.settings.reg.section.nodes.updates": "Обновление агентов",
  "admin.settings.reg.section.nodes.agent": "Работа агента",
  "admin.settings.reg.section.nodes.diagnostics": "Диагностика диска",
  "admin.settings.reg.section.nodes.diagnostics.description":
    "Пороги, при которых проверка диска на узле показывает предупреждение или ошибку.",

  "admin.settings.reg.nodes.offline_after_sec": "Узел недоступен, если молчит",
  "admin.settings.reg.nodes.offline_after_sec.hint":
    "Не меньше трёх интервалов сигнала агента",
  "admin.settings.reg.nodes.offline_notify_min": "Оповещать о недоступности через",
  "admin.settings.reg.nodes.disk_low_percent": "Мало диска, если свободно меньше",
  "admin.settings.reg.nodes.bulk_delay_sec": "Пауза между массовыми командами узлам",
  "admin.settings.reg.nodes.agent_update_timeout_min": "Время на обновление агента",
  "admin.settings.reg.nodes.agent_outdated_grace_min": "Не напоминать об устаревших агентах после выхода версии",

  "admin.settings.reg.agent.heartbeat_sec": "Сигнал агента раз в",
  "admin.settings.reg.agent.heartbeat_sec.hint": "Не реже раза в 30 секунд — иначе узел будет мигать офлайном",
  "admin.settings.reg.agent.metrics_sec": "Метрики серверов раз в",
  "admin.settings.reg.agent.metrics_sec.hint": "Статус «запущен» в панели держится 45 секунд",
  "admin.settings.reg.agent.timeout_scale_percent": "Запас времени на операции агента",
  "admin.settings.reg.agent.timeout_scale_percent.hint": "100 — стандартные таймауты, 200 — вдвое длиннее",
  "admin.settings.reg.agent.install_timeout_min": "Время на установку сервера",
  "admin.settings.reg.agent.stop_timeout_sec": "Время на корректную остановку сервера",
  "admin.settings.reg.agent.stop_timeout_sec.hint": "0 — как задано в Docker (10 секунд)",
  "admin.settings.reg.agent.pids_limit": "Лимит процессов в контейнере",
  "admin.settings.reg.agent.pids_limit.hint":
    "Защита от fork-бомб. Переменная VORTANIX_PIDS_LIMIT на ноде главнее",
  "admin.settings.reg.agent.cron_job_timeout_min": "Время на задание cron сервера",
  "admin.settings.reg.agent.log_max_size_mb": "Размер файла логов сервера",
  "admin.settings.reg.agent.log_max_size_mb.hint":
    "Docker хранит логи игрового сервера в файлах и ротирует их. Применяется к серверам, созданным или переустановленным после изменения",
  "admin.settings.reg.agent.log_max_files": "Файлов логов на сервер",
  "admin.settings.reg.agent.temp_file_age_min": "Временные файлы удаляются старше",

  "admin.settings.reg.agent.disk_warn_free_mb": "Предупреждение: свободно меньше",
  "admin.settings.reg.agent.disk_fail_free_mb": "Ошибка: свободно меньше",
  "admin.settings.reg.agent.disk_warn_used_percent": "Предупреждение: занято больше",
  "admin.settings.reg.agent.disk_fail_used_percent": "Ошибка: занято больше",
};
