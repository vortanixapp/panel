export const registry = {
  "admin.settings.tab.security": "Безопасность",
  "admin.settings.tab.billing": "Биллинг",
  "admin.settings.tab.servers": "Серверы",
  "admin.settings.tab.notifications": "Уведомления",
  "admin.settings.tab.retention": "Хранение",
  "admin.settings.tab.nodes": "Узлы",
  "admin.settings.tab.uploads": "Загрузки",
  "admin.settings.tab.interface": "Интерфейс",

  "admin.settings.reg.group.security":
    "Время жизни сессий и ссылок, лимиты попыток входа, блокировка IP и политика паролей. Значения применяются сразу, без перезапуска.",
  "admin.settings.reg.group.retention": "",
  "admin.settings.reg.group.nodes": "",
  "admin.settings.reg.group.uploads": "",
  "admin.settings.reg.group.interface": "",

  "admin.settings.reg.empty": "В этом разделе пока нет настроек",
  "admin.settings.reg.reset_field": "По умолчанию",
  "admin.settings.reg.reset_all": "Сбросить раздел",
  "admin.settings.reg.reset_all_confirm":
    "Вернуть все настройки этого раздела к значениям по умолчанию? Изменения вступят в силу после сохранения.",
  "admin.settings.reg.range": "от {min} до {max} {unit} · по умолчанию {default}",
  "admin.settings.reg.default": "по умолчанию {default}",
  "admin.settings.reg.error.number": "Введите целое число",
  "admin.settings.reg.error.list": "Введите числа через запятую",
  "admin.settings.reg.error.range": "Допустимо от {min} до {max}",
  "admin.settings.reg.error.length": "Не длиннее {max} символов",

  "admin.settings.reg.unit.ms": "мс",
  "admin.settings.reg.unit.sec": "сек",
  "admin.settings.reg.unit.min": "мин",
  "admin.settings.reg.unit.hour": "ч",
  "admin.settings.reg.unit.day": "дн",
  "admin.settings.reg.unit.mb": "МБ",
  "admin.settings.reg.unit.percent": "%",
  "admin.settings.reg.unit.count": "шт",

  "admin.settings.reg.section.security.tokens": "Сессии и токены",
  "admin.settings.reg.section.security.tokens.description":
    "Срок жизни токенов доступа и обновления. Уже выданные токены действуют до своего прежнего срока.",
  "admin.settings.reg.section.security.links": "Ссылки из писем",
  "admin.settings.reg.section.security.access": "Регистрация и пароли",
  "admin.settings.reg.section.security.attempts": "Лимиты попыток",
  "admin.settings.reg.section.security.attempts.description":
    "Сколько раз за окно можно попробовать действие с одного IP или для одной учётной записи.",
  "admin.settings.reg.section.security.ipblock": "Автоблокировка IP",
  "admin.settings.reg.section.security.ipblock.description":
    "IP блокируется, если с него набралось слишком много неудачных входов за окно.",
  "admin.settings.reg.section.security.requests": "Частота запросов",

  "admin.settings.reg.auth.access_ttl_min": "Токен доступа живёт",
  "admin.settings.reg.auth.access_ttl_min.hint": "После истечения обновляется автоматически",
  "admin.settings.reg.auth.refresh_ttl_default_days": "Токен обновления по умолчанию",
  "admin.settings.reg.auth.refresh_ttl_session_hours": "Сессия без «запомнить меня»",
  "admin.settings.reg.auth.refresh_ttl_remember_days": "Сессия с «запомнить меня»",
  "admin.settings.reg.auth.refresh_grace_sec": "Окно повторного использования токена",
  "admin.settings.reg.auth.refresh_grace_sec.hint":
    "Защита от гонки двух вкладок; после окна повтор старого токена закрывает сессию",
  "admin.settings.reg.auth.saved_logins_max": "Запомненных аккаунтов в переключателе",
  "admin.settings.reg.auth.twofa_challenge_ttl_min": "Время на ввод кода 2FA",

  "admin.settings.reg.auth.reset_link_ttl_min": "Ссылка сброса пароля",
  "admin.settings.reg.auth.register_confirm_ttl_hours": "Подтверждение регистрации",
  "admin.settings.reg.auth.email_change_ttl_hours": "Подтверждение смены e-mail",
  "admin.settings.reg.auth.email_verify_ttl_min": "Подтверждение адреса",
  "admin.settings.reg.auth.exists_mail_interval_min": "Письмо «адрес уже занят» не чаще",

  "admin.settings.reg.auth.registration_enabled": "Разрешить регистрацию",
  "admin.settings.reg.auth.registration_enabled.hint":
    "Выключите, чтобы закрыть создание новых аккаунтов, включая вход через соцсети",
  "admin.settings.reg.auth.register_confirm_email": "Подтверждать регистрацию по почте",
  "admin.settings.reg.auth.register_confirm_email.hint":
    "Работает, только когда настроена отправка почты",
  "admin.settings.reg.auth.password_min_length": "Минимальная длина пароля",
  "admin.settings.reg.auth.password_min_length.hint": "Максимум 72 байта — предел bcrypt",
  "admin.settings.reg.auth.personal_token_limit": "Личных API-токенов на пользователя",

  "admin.settings.reg.auth.login_attempts": "Попыток входа",
  "admin.settings.reg.auth.login_window_min": "Окно попыток входа",
  "admin.settings.reg.auth.register_per_hour": "Регистраций в час",
  "admin.settings.reg.auth.register_dup_per_hour": "Повторов занятого e-mail в час",
  "admin.settings.reg.auth.register_mail_per_hour": "Писем подтверждения на адрес в час",
  "admin.settings.reg.auth.forgot_attempts": "Запросов сброса пароля",
  "admin.settings.reg.auth.reset_attempts": "Попыток задать новый пароль",
  "admin.settings.reg.auth.twofa_ip_attempts": "Попыток 2FA с одного IP",
  "admin.settings.reg.auth.twofa_challenge_tries": "Попыток кода на один вход",
  "admin.settings.reg.auth.twofa_user_attempts": "Попыток 2FA на учётную запись",
  "admin.settings.reg.auth.account_attempts": "Смена пароля и кодов 2FA",
  "admin.settings.reg.auth.social_attempts": "Вход через соцсети и смена e-mail",
  "admin.settings.reg.auth.confirm_attempts": "Подтверждение регистрации",
  "admin.settings.reg.auth.attempts_window_min": "Окно остальных лимитов",

  "admin.settings.reg.auth.ip_block_window_min": "Окно подсчёта неудач",
  "admin.settings.reg.auth.ip_block_threshold": "Порог неудачных входов",
  "admin.settings.reg.auth.ip_block_duration_min": "Длительность блокировки",

  "admin.settings.reg.security.write_rpm": "Записывающих запросов в минуту",
  "admin.settings.reg.security.write_rpm.hint":
    "На пользователя или API-ключ: POST, PUT, PATCH и DELETE",
};
