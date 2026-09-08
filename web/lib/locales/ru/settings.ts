// Настройки пользователя: аккаунт, оформление, отображение, уведомления.
export const settings = {
  // Заголовки разделов настроек
  "settings.account.section_title": "Аккаунт",
  "settings.account.section_desc": "Личные данные, пароль и привязанные аккаунты",
  "settings.appearance.section_title": "Внешний вид",
  "settings.appearance.section_desc": "Тема и шрифт панели",
  "settings.display.section_title": "Отображение",
  "settings.display.section_desc": "Какие разделы видны в боковом меню",
  "settings.notifications.section_title": "Уведомления",
  "settings.notifications.section_desc":
    "Настройте, о каких событиях вы хотите получать уведомления",

  // Внешний вид
  "settings.appearance.theme_title": "Тема",
  "settings.appearance.theme_hint": "Применяется сразу и хранится в этом браузере",
  "settings.appearance.theme_light": "Светлая",
  "settings.appearance.theme_dark": "Тёмная",
  "settings.appearance.theme_system": "Системная",
  "settings.appearance.font_title": "Шрифт",
  "settings.appearance.font_system": "Системный",
  "settings.appearance.saved": "Настройки внешнего вида сохранены",

  // Отображение
  "settings.display.sidebar_title": "Боковое меню",
  "settings.display.sidebar_hint":
    "Выберите разделы, которые будут видны в навигации. Настройка хранится в этом браузере",
  "settings.display.pick_one": "Выберите хотя бы один раздел",
  "settings.display.saved": "Настройки отображения сохранены",

  // Уведомления
  "settings.notifications.channels_title": "Каналы",
  "settings.notifications.channels_hint": "Для Telegram нужен chat id, для Discord — webhook",
  "settings.notifications.email_desc":
    "События по серверам, заказам и поддержке на адрес учётной записи",
  "settings.notifications.telegram_desc": "Доставка через бота панели в указанный чат",
  "settings.notifications.discord_desc": "Доставка сообщением в канал через webhook",
  "settings.notifications.chat_id_max": "Не более 64 символов",
  "settings.notifications.webhook_max": "Не более 300 символов",
  "settings.notifications.chat_id_required": "Укажите chat id — иначе уведомления не дойдут",
  "settings.notifications.webhook_required": "Укажите webhook — иначе уведомления не дойдут",
  "settings.notifications.saved": "Каналы доставки сохранены",
  "settings.notifications.load_failed":
    "Не удалось загрузить каналы доставки. Обновите страницу.",

  // Аккаунт: вкладки
  "settings.account.tab_profile": "Аккаунт",
  "settings.account.tab_contacts": "Контакты",
  "settings.account.tab_security": "Безопасность",
  "settings.account.tab_twofa": "2FA",
  "settings.account.tab_sessions": "Сессии",
  "settings.account.tab_appearance": "Внешний вид",
  "settings.account.tab_notifications": "Уведомления",

  // Аккаунт: шапка
  "settings.account.subtitle": "Управление аккаунтом и параметрами панели",
  "settings.account.save_hint": "Изменения сохраняются по кнопке в разделе",
  "settings.account.loading": "Загрузка аккаунта…",

  // Аккаунт: профиль
  "settings.account.avatar_upload": "Загрузить аватар",
  "settings.account.avatar_hint": "JPG, PNG или WebP, до 4 МБ",
  "settings.account.avatar_too_big": "Файл не больше 4 МБ",
  "settings.account.avatar_updated": "Аватар обновлён",
  "settings.account.avatar_upload_failed": "Ошибка загрузки",
  "settings.account.email_verified": "Email подтверждён",
  "settings.account.email_unverified": "Email не подтверждён —",
  "settings.account.email_verify_link": "подтвердить",
  "settings.account.first_name": "Имя",
  "settings.account.last_name": "Фамилия",
  "settings.account.display_name": "Отображаемое имя",
  "settings.account.phone": "Телефон",
  "settings.account.language": "Язык",
  "settings.account.language_ru": "Русский",
  "settings.account.language_en": "English",
  "settings.account.save_profile": "Сохранить профиль",
  "settings.account.profile_saved": "Профиль сохранён",

  // Аккаунт: язык интерфейса
  "settings.account.language_label": "Язык интерфейса",
  "settings.account.language_select": "Выберите язык",
  "settings.account.language_hint":
    "Сохраняется локально в браузере до появления i18n на сервере.",

  // Аккаунт: контакты
  "settings.account.contacts_hint": "Обновление почты и персональных контактных данных.",
  "settings.account.new_email": "Новый email",
  "settings.account.update_email": "Обновить email",
  "settings.account.email_invalid": "Введите корректный email",
  "settings.account.email_updated": "Email обновлён. Подтвердите новый адрес.",
  "settings.account.email_dev_link": "Dev: ссылка подтверждения в консоли ответа API",
  "settings.account.email_change_failed": "Ошибка смены email",

  // Аккаунт: пароль
  "settings.account.password_section": "Смена пароля",
  "settings.account.password_section_hint":
    "Оставьте поля пустыми, если пароль менять не нужно.",
  "settings.account.password_current": "Текущий пароль",
  "settings.account.password_new": "Новый пароль",
  "settings.account.password_confirm": "Подтверждение",
  "settings.account.password_current_required": "Введите текущий пароль",
  "settings.account.password_min": "Новый пароль — минимум 8 символов",
  "settings.account.password_mismatch": "Пароли не совпадают",
  "settings.account.password_updated": "Пароль обновлён",
  "settings.account.password_change_failed": "Не удалось сменить пароль",
  "settings.account.password_update": "Обновить пароль",
  "settings.account.saved": "Настройки аккаунта сохранены",

  // Аккаунт: соцсети
  "settings.account.social_title": "Вход через соцсети",
  "settings.account.social_hint": "Привяжите аккаунты для быстрого входа на странице login.",
  "settings.account.social_checking": "Проверяем доступные способы входа…",
  "settings.account.social_none":
    "Ни один провайдер не настроен на сервере панели. Задайте ключи OAuth (Google / Discord / VK) или токен Telegram-бота, чтобы включить вход через соцсети.",
  "settings.account.social_link": "Привязать",
  "settings.account.social_unlink": "Отвязать",
  "settings.account.social_linked": "Соцсеть отвязана",
  "settings.account.provider_not_configured": "Провайдер не настроен на сервере",
  "settings.account.link_failed": "Ошибка привязки",
  "settings.account.unlink_failed": "Ошибка отвязки",
  "settings.account.telegram_linked": "Telegram привязан",
  "settings.account.telegram_link_failed": "Ошибка привязки Telegram",

  // Аккаунт: сессии
  "settings.account.sessions_title": "Активные сессии",
  "settings.account.sessions_hint": "Подключённые устройства и недавние входы.",
  "settings.account.sessions_close_others": "Завершить другие сессии",
  "settings.account.sessions_loading": "Загрузка сессий…",
  "settings.account.sessions_empty": "Активных сессий не найдено.",
  "settings.account.session_unknown_device": "Неизвестное устройство",
  "settings.account.session_current": "Текущая",
  "settings.account.session_close": "Завершить",
  "settings.account.session_activity": "Активность:",
  "settings.account.session_closed": "Сессия завершена",
  "settings.account.sessions_others_closed": "Другие сессии завершены",

  // Аккаунт: двухфакторная аутентификация
  "settings.account.twofa_title": "Двухфакторная аутентификация",
  "settings.account.twofa_on_desc": "2FA включена для вашего аккаунта.",
  "settings.account.twofa_off_desc": "Дополнительная защита при входе.",
  "settings.account.twofa_setup": "Настроить 2FA",
  "settings.account.twofa_disable": "Отключить 2FA",
  "settings.account.twofa_secret_label": "Секрет для приложения (Google Authenticator и др.):",
  "settings.account.twofa_confirm": "Подтвердить и включить",
  "settings.account.twofa_scan": "Сканируйте QR в приложении-аутентификаторе",
  "settings.account.twofa_create_failed": "Не удалось создать 2FA",
  "settings.account.twofa_enabled": "2FA включена",
  "settings.account.twofa_enable_failed": "Ошибка включения 2FA",
  "settings.account.twofa_disabled": "2FA отключена",
  "settings.account.twofa_disable_password":
    "Введите пароль, чтобы отключить двухфакторную защиту",
  "settings.account.twofa_disable_password_required": "Без пароля отключить нельзя",
  "settings.account.twofa_disable_failed": "Ошибка отключения 2FA",
};
