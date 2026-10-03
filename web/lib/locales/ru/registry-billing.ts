export const registryBilling = {
  "admin.settings.reg.group.billing":
    "Напоминания об оплате, поведение просроченных серверов, лимиты аренды и пополнения. Значения применяются сразу, без перезапуска.",

  "admin.settings.reg.section.billing.reminders": "Напоминания и продление",
  "admin.settings.reg.section.billing.expiry": "Просроченные серверы",
  "admin.settings.reg.section.billing.expiry.description":
    "По умолчанию сервер останавливается сразу после окончания оплаты, а платные серверы не удаляются никогда.",
  "admin.settings.reg.section.billing.rental": "Аренда",
  "admin.settings.reg.section.billing.payments": "Пополнение и платёжные системы",
  "admin.settings.reg.section.billing.bonuses": "Бонусы и хранение данных",

  "admin.settings.reg.billing.dunning_stages_days": "Напоминать об окончании за",
  "admin.settings.reg.billing.dunning_stages_days.hint":
    "Дни до окончания через запятую, например 1, 3, 7",
  "admin.settings.reg.billing.autorenew_horizon_hours": "Автопродление за",
  "admin.settings.reg.billing.autorenew_horizon_hours.hint":
    "Сервер с включённым автопродлением продлевается за это время до окончания",
  "admin.settings.reg.billing.expiring_soon_days": "«Скоро истекает» — за",
  "admin.settings.reg.billing.expiring_soon_days.hint":
    "Порог для счётчиков и подсветки на панели и в списке серверов",
  "admin.settings.reg.billing.balance_low_days": "Баланс на исходе — за",
  "admin.settings.reg.billing.balance_low_days.hint":
    "Предупреждать, если баланса не хватит на продление раньше этого срока",
  "admin.settings.reg.billing.hourly_low_balance_hours": "Почасовой сервер: предупреждать за",

  "admin.settings.reg.billing.server_grace_hours": "Отсрочка перед остановкой",
  "admin.settings.reg.billing.server_grace_hours.hint": "0 — останавливать сразу после окончания",
  "admin.settings.reg.billing.hosting_grace_hours": "Отсрочка перед блокировкой хостинга",
  "admin.settings.reg.billing.hosting_grace_hours.hint": "0 — блокировать сразу после окончания",
  "admin.settings.reg.billing.server_delete_days": "Удалять платный сервер через",
  "admin.settings.reg.billing.server_delete_days.hint":
    "0 — не удалять. Срок идёт с момента остановки, сервер удаляется вместе с файлами без возможности восстановления",
  "admin.settings.reg.billing.trial_keep_hours": "Пробный сервер хранится после остановки",

  "admin.settings.reg.billing.rent_max_batch": "Серверов за один заказ",
  "admin.settings.reg.billing.rent_max_days": "Максимальный срок аренды и продления",
  "admin.settings.reg.billing.hourly_prepaid_hours": "Почасовая аренда: баланс минимум на",
  "admin.settings.reg.billing.hourly_catchup_hours": "Почасовое списание: догонять не больше",
  "admin.settings.reg.billing.hourly_catchup_hours.hint":
    "Сколько часов списывается за один раз после простоя панели",

  "admin.settings.reg.billing.topup_min": "Минимальное пополнение",
  "admin.settings.reg.billing.topup_min.hint": "В валюте платежа, 0 — без ограничения",
  "admin.settings.reg.billing.topup_max": "Максимальное пополнение",
  "admin.settings.reg.billing.topup_max.hint": "В валюте платежа, 0 — без ограничения",
  "admin.settings.reg.billing.topup_default": "Сумма пополнения по умолчанию",
  "admin.settings.reg.billing.topup_presets": "Быстрые суммы пополнения",
  "admin.settings.reg.billing.topup_presets.hint": "Кнопки на странице баланса, через запятую",
  "admin.settings.reg.billing.invoice_expire_min": "Срок жизни счёта",
  "admin.settings.reg.billing.invoice_expire_min.hint": "Для платёжных систем, которые принимают срок (Lava, Enot)",
  "admin.settings.reg.billing.fx_fresh_hours": "Курс валют считается свежим",
  "admin.settings.reg.billing.fx_stale_hours": "Курс валют пригоден, если источник недоступен",

  "admin.settings.reg.billing.bonus_cooldown_hours": "Колесо бонусов: интервал",
  "admin.settings.reg.billing.retain_years": "Хранить обезличенные данные удалённого аккаунта",
  "admin.settings.reg.billing.retain_years.hint":
    "Проверьте требования закона: для бухгалтерских данных действуют минимальные сроки",
};
