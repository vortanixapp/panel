// Главная пользователя, мониторинг, активность, бонус за вход.
export const dashboard = {
  // Формы с числом: plural() выбирает ключ, а не готовое слово, иначе
  // подпись застыла бы на языке, который стоял при расчёте списка.
  "dashboard.balance.burn_days_one": "Списание {amount} ₽ в сутки, хватит на {days} день",
  "dashboard.balance.burn_days_few": "Списание {amount} ₽ в сутки, хватит на {days} дня",
  "dashboard.balance.burn_days_many": "Списание {amount} ₽ в сутки, хватит на {days} дней",
  "dashboard.servers.expires_in_one": "истекает через {days} день",
  "dashboard.servers.expires_in_few": "истекает через {days} дня",
  "dashboard.servers.expires_in_many": "истекает через {days} дней",
  "dashboard.next_charge.in_one": "через {days} день",
  "dashboard.next_charge.in_few": "через {days} дня",
  "dashboard.next_charge.in_many": "через {days} дней",
  "dashboard.support.open_line_one": "{count} открыт",
  "dashboard.support.open_line_few": "{count} открыто",
  "dashboard.support.open_line_many": "{count} открыто",
  "dashboard.support.tickets_line_one": "{count} открытое обращение",
  "dashboard.support.tickets_line_few": "{count} открытых обращения",
  "dashboard.support.tickets_line_many": "{count} открытых обращений",
  "dashboard.step.topup_days_one": "хватит на {days} день",
  "dashboard.step.topup_days_few": "хватит на {days} дня",
  "dashboard.step.topup_days_many": "хватит на {days} дней",
  "dashboard.days_one": "{count} день",
  "dashboard.days_few": "{count} дня",
  "dashboard.days_many": "{count} дней",
  "dashboard.time.days_ago_one": "{count} день назад",
  "dashboard.time.days_ago_few": "{count} дня назад",
  "dashboard.time.days_ago_many": "{count} дней назад",

  // Относительное время в журнале и уведомлениях
  "dashboard.time.just_now": "только что",
  "dashboard.time.minutes_ago": "{count} мин назад",
  "dashboard.time.hours_ago": "{count} ч назад",
  "dashboard.time.yesterday": "вчера",
  "dashboard.group.today": "Сегодня · {date}",
  "dashboard.group.yesterday": "Вчера · {date}",

  // Относительное время в ленте операций
  "dashboard.rel.today": "сегодня, {time}",
  "dashboard.rel.yesterday": "вчера, {time}",
  "dashboard.rel.date": "{date}, {time}",

  // Карточка баланса
  "dashboard.balance": "Баланс",
  "dashboard.balance.burn": "Списание {amount} ₽ в сутки",
  "dashboard.topup": "Пополнить",
  "dashboard.invoices": "Счета",

  // Карточка серверов
  "dashboard.servers.title": "Серверы",
  "dashboard.servers.zero_active": "0 активных",
  "dashboard.servers.count": "{active} активных · всего {total}",
  "dashboard.servers.empty_title": "Ни одного сервера",
  "dashboard.servers.empty_text":
    "Выберите игру, локацию и объём памяти — сервер поднимется примерно за 40 секунд.",
  "dashboard.servers.rent": "Арендовать",
  "dashboard.servers.of_players": "из {max} игроков",
  "dashboard.servers.expired": "срок истёк",
  "dashboard.servers.until": "до {date}",
  "dashboard.servers.renew": "Продлить",
  "dashboard.servers.console": "Консоль",

  // Ближайшее списание
  "dashboard.next_charge.expired": "истёк",

  // Карточка поддержки
  "dashboard.support.title": "Поддержка",
  "dashboard.support.none": "нет обращений",
  "dashboard.support.waiting": "Обращения ждут вашего ответа",
  "dashboard.support.first_reply": "Средний первый ответ — 4 минуты",
  "dashboard.support.open_tickets": "Открыть обращения",
  "dashboard.support.write": "Написать в поддержку",

  // График расходов
  "dashboard.spend.title": "Расходы за 30 дней",
  "dashboard.spend.day_tooltip": "{date} · −{debit} ₽ / +{credit} ₽",
  "dashboard.spend.empty": "По последним операциям расходов за период не найдено.",

  // Бонус за вход на главной
  "dashboard.bonus.prize_won": "Приз получен! Проверьте баланс.",
  "dashboard.bonus.no_prize": "Крутите завтра — сегодня приз не выпал",
  "dashboard.bonus.available": "Прокрутка доступна — бонус зачислится на баланс.",
  "dashboard.bonus.done": "Уже получено. Следующая прокрутка через 24 часа.",
  "dashboard.bonus.already": "Уже получено",

  // Последние операции
  "dashboard.recent.title": "Последние операции",
  "dashboard.recent.journal": "Журнал",
  "dashboard.recent.empty_text": "Пополнения и списания появятся здесь",

  // Что сделать дальше
  "dashboard.next_steps.title": "Что сделать дальше",
  "dashboard.step.choose_game": "Выбрать игру",
  "dashboard.step.choose_game_sub": "каталог игр и модпаков",
  "dashboard.step.topup_sub": "СБП, карта, крипта",
  "dashboard.step.bonus": "Забрать бонус",
  "dashboard.step.bonus_sub": "ежедневная прокрутка",
  "dashboard.step.renew": "Продлить {name}",
  "dashboard.step.renew_expired": "срок аренды истёк",
  "dashboard.step.support": "Ответить в поддержке",
  "dashboard.step.topup_keep": "чтобы серверы продолжали работать",
  "dashboard.step.rent_more": "Арендовать ещё сервер",
  "dashboard.step.rent_more_sub": "запуск примерно за 40 секунд",


  // Журнал действий
  "dashboard.activity.title": "Журнал",
  "dashboard.activity.subtitle": "Все действия в аккаунте: серверы, биллинг, доступ",
  "dashboard.activity.export_csv": "Экспорт CSV",
  "dashboard.activity.exporting": "Выгрузка…",
  "dashboard.activity.exported": "Журнал выгружен",
  "dashboard.activity.export_failed": "Не удалось выгрузить журнал",
  "dashboard.activity.cat_billing": "Биллинг",
  "dashboard.activity.cat_auth": "Доступ",
  "dashboard.activity.range_24h": "24 часа",
  "dashboard.activity.range_7d": "7 дней",
  "dashboard.activity.range_30d": "30 дней",
  "dashboard.activity.stat_events": "Событий за 24 ч",
  "dashboard.activity.stat_sub_total": "всего",
  "dashboard.activity.stat_server_actions": "Действий с серверами",
  "dashboard.activity.for_24h": "за 24 ч",
  "dashboard.activity.stat_logins": "Входов в аккаунт",
  "dashboard.activity.stat_errors": "Ошибок",
  "dashboard.activity.search_placeholder": "Действие, ресурс или e-mail",
  "dashboard.activity.clear": "Очистить",
  "dashboard.activity.load_failed_title": "Не удалось загрузить журнал",
  "dashboard.activity.load_failed_text":
    "Проверьте соединение и попробуйте обновить страницу.",
  "dashboard.activity.empty_title": "Записей нет",
  "dashboard.activity.empty_text": "Измените запрос, тип действия или период.",
  "dashboard.activity.records_one": "запись",
  "dashboard.activity.records_few": "записи",
  "dashboard.activity.records_many": "записей",
  "dashboard.activity.shown": "Показано {shown} из {total} · {range}",
  "dashboard.activity.show_more": "Показать ещё {count}",

  // Оповещения
  "dashboard.notifications.title": "Оповещения",
  "dashboard.notifications.subtitle": "Серверы, деньги, поддержка и безопасность",
  "dashboard.notifications.mark_all": "Прочитать все",
  "dashboard.notifications.marked_all": "Все оповещения прочитаны",
  "dashboard.notifications.cleared": "Убрано прочитанных: {count}",
  "dashboard.notifications.clear_empty": "Прочитанных оповещений нет",
  "dashboard.notifications.clear_failed": "Не удалось убрать прочитанные",
  "dashboard.notifications.clear_read": "Убрать прочитанные",
  "dashboard.notifications.channels_updated": "Каналы доставки обновлены",
  "dashboard.notifications.channels_failed": "Не удалось обновить каналы",
  "dashboard.notifications.unread_label": "Непрочитанных:",
  "dashboard.notifications.filter_unread": "Непрочитанные",
  "dashboard.notifications.load_failed_title": "Не удалось загрузить оповещения",
  "dashboard.notifications.load_failed_text":
    "Проверьте соединение и попробуйте обновить.",
  "dashboard.notifications.all_read_title": "Всё прочитано",
  "dashboard.notifications.all_read_text":
    "Новые события появятся здесь автоматически.",
  "dashboard.notifications.empty_title": "Оповещений пока нет",
  "dashboard.notifications.empty_text":
    "Здесь появятся события по серверам, платежам и обращениям.",
  "dashboard.notifications.more": "Ещё",
  "dashboard.notifications.channels_title": "Каналы доставки",
  "dashboard.notifications.channels_hint": "Куда дублировать важные события",
  "dashboard.notifications.configure": "Настроить →",
  "dashboard.notifications.read": "Прочитано",
  "dashboard.notifications.remove": "Убрать",
  "dashboard.notifications.remove_aria": "Убрать оповещение",
  "dashboard.notifications.channel_not_configured":
    "Канал включён, но адрес доставки не указан — оповещения не придут",

  // Мониторинг: общие подписи и форматирование
  "monitoring.title": "Мониторинг",
  "monitoring.subtitle": "Статус игровых серверов · обновление каждые 15 с",
  "monitoring.subtitle_short": "Статус игровых серверов",
  "monitoring.back": "← Мониторинг",
  "monitoring.server.untitled": "Без названия",
  "monitoring.address_copied": "Адрес скопирован",
  "monitoring.address_hidden": "адрес скрыт",
  "monitoring.unit.ms": "{value} мс",
  "monitoring.duration.hours_minutes": "{hours} ч {minutes} мин",
  "monitoring.duration.hours": "{hours} ч",
  "monitoring.duration.minutes": "{minutes} мин",
  "monitoring.duration.seconds": "{seconds} с",
  "monitoring.updated.seconds_ago": "обновлено {value} с назад",
  "monitoring.updated.minutes_ago": "обновлено {value} мин назад",
  "monitoring.updated.hours_ago": "обновлено {value} ч назад",

  // Статусы сервера в мониторинге
  "monitoring.status.online": "Онлайн",
  "monitoring.status.starting": "Запуск",
  "monitoring.status.stopping": "Остановка",
  "monitoring.status.error": "Ошибка",
  "monitoring.status.offline": "Офлайн",

  // Список серверов
  "monitoring.error.title": "Мониторинг недоступен",
  "monitoring.error.text":
    "Не удалось загрузить данные мониторинга. Проверьте соединение и попробуйте ещё раз.",
  "monitoring.empty.title": "Данных пока нет",
  "monitoring.empty.text":
    "Мониторинг появится, как только вы запустите первый игровой сервер. Данные собираются автоматически, настройка не нужна.",
  "monitoring.empty.rent_cta": "Арендовать сервер",
  "monitoring.empty.feat_online": "Онлайн и слоты",
  "monitoring.empty.feat_uptime": "Аптайм 30 дней",
  "monitoring.empty.feat_banners": "Баннеры для форумов",
  "monitoring.add_server": "Добавить сервер",
  "monitoring.my_servers": "Мои серверы",
  "monitoring.count_of": "{shown} из {total}",
  "monitoring.search_placeholder": "Поиск по названию или IP",
  "monitoring.view.table": "Таблица",
  "monitoring.view.cards": "Карточки",
  "monitoring.filter_empty_hint": "Измените запрос или сбросьте фильтр по игре.",
  "monitoring.details": "Детали",
  "monitoring.public_link": "Публично",

  // Показатели вверху списка
  "monitoring.kpi.total": "Всего серверов",
  "monitoring.kpi.online_sub": "{count} онлайн",
  "monitoring.kpi.players": "Игроков сейчас",
  "monitoring.kpi.slots_sub": "из {count} слотов",
  "monitoring.kpi.avg_uptime": "Средний аптайм",
  "monitoring.kpi.for_30d": "за 30 дней",
  "monitoring.kpi.incidents": "Инцидентов",
  "monitoring.kpi.for_7d": "за 7 дней",

  // Колонки таблиц мониторинга
  "monitoring.col.server": "Сервер",
  "monitoring.col.online": "Онлайн",
  "monitoring.col.day": "24 часа",
  "monitoring.col.cpu_ram": "CPU / RAM",
  "monitoring.col.ping": "Пинг",
  "monitoring.col.tps": "TPS",
  "monitoring.col.uptime": "Аптайм",
  "monitoring.col.address": "Адрес",
  "monitoring.col.game": "Игра",
  "monitoring.col.votes": "Голоса",

  // Публичный рейтинг
  "monitoring.public.title": "Публичный мониторинг",
  "monitoring.public.subtitle": "Топ серверов сети за сутки",
  "monitoring.public.all_ranking": "Весь рейтинг",
  "monitoring.public.rating_empty":
    "В рейтинге пока нет серверов. В него попадают серверы с включённой публичной страницей.",
  "monitoring.top.all_games": "Все игры",
  "monitoring.top.title": "Топ серверов",
  "monitoring.top.subtitle": "Рейтинг по голосам игроков и текущему онлайну",
  "monitoring.top.load_failed": "Не удалось загрузить рейтинг.",
  "monitoring.top.empty_title": "Рейтинг пуст",
  "monitoring.top.empty_text":
    "Сюда попадают серверы с включённой публичной страницей. Включите её на вкладке «Публичная страница» в деталях сервера.",

  // Публичная страница сервера
  "monitoring.public.unavailable_title": "Страница недоступна",
  "monitoring.public.unavailable_text":
    "Владелец не публиковал этот сервер, либо ссылка устарела.",
  "monitoring.public.online_now": "Сейчас онлайн",
  "monitoring.public.vote": "Голосовать · {count}",
  "monitoring.vote.thanks": "Спасибо за голос!",
  "monitoring.vote.already": "Сегодня вы уже голосовали за этот сервер",
  "monitoring.public.website": "Сайт",

  // Плитки показателей сервера
  "monitoring.stat.version": "Версия",
  "monitoring.stat.map": "Карта",
  "monitoring.stat.map_note": "мир",
  "monitoring.stat.region": "Регион",
  "monitoring.stat.peak_24h": "Пик за сутки",
  "monitoring.stat.uptime_30d_note": "30 дней",
  "monitoring.stat.tps_goal": "цель 20.0",

  // Графики онлайна
  "monitoring.chart.online_24h": "Онлайн за 24 часа",
  "monitoring.chart.online_7d": "Онлайн за 7 дней",
  "monitoring.chart.step_10m": "шаг 10 минут",
  "monitoring.chart.step_1h": "шаг 1 час",
  "monitoring.chart.peak": "пик {value}",
  "monitoring.chart.collecting": "История ещё собирается.",
  "monitoring.chart.legend_online": "Онлайн",
  "monitoring.chart.legend_slots": "Слоты",
  "monitoring.chart.empty":
    "Данных пока нет. История собирается с момента запуска сервера.",
  "monitoring.chart.avg_online": "Средний онлайн",
  "monitoring.chart.peak_label": "Пик",
  "monitoring.chart.min": "Минимум",
  "monitoring.chart.fill": "Заполненность",

  // Игроки
  "monitoring.players.title": "Игроки на сервере",
  "monitoring.players.empty_public": "Сейчас на сервере никого нет.",
  "monitoring.players.search_placeholder": "Поиск по нику",
  "monitoring.players.count_summary": "{count} игроков · средний онлайн {avg}",
  "monitoring.players.peak_24h": "Пик за сутки: {count} игроков",
  "monitoring.players.col_nick": "Ник",
  "monitoring.players.col_frags": "Фраги",
  "monitoring.players.col_session": "Время в сессии",
  "monitoring.players.empty_running":
    "Список игроков недоступен — сервер не отвечает на запрос игроков.",
  "monitoring.players.empty_offline": "Сервер офлайн, игроков нет.",

  // Баннеры
  "monitoring.banner.title": "Баннер сервера",
  "monitoring.banner.hint":
    "Поставьте на форум или сайт — онлайн обновляется автоматически.",
  "monitoring.banner.alt": "Баннер {name}",
  "monitoring.banner.copied": "Код баннера скопирован",
  "monitoring.banner.get_code": "Получить код",
  "monitoring.banners.public_off":
    "Публичная страница выключена — баннеры отдаются, но ссылка из них ведёт на закрытую страницу. Включите публичную страницу на соседней вкладке.",
  "monitoring.banners.size_title": "Баннер {size}",
  "monitoring.banners.png_note": "PNG · авто-обновление 5 мин",
  "monitoring.banners.refresh_preview": "Обновить превью",
  "monitoring.banners.unavailable": "Баннер временно недоступен",
  "monitoring.banners.code_site": "Код для сайта",
  "monitoring.banners.code_forum": "Код для форума (BBCode)",
  "monitoring.code.copied": "Код скопирован",
  "monitoring.code.copy": "Скопировать",

  // Вкладки страницы сервера
  "monitoring.tab.players": "Игроки",
  "monitoring.tab.stats": "Статистика",
  "monitoring.tab.banners": "Баннеры",
  "monitoring.tab.console": "Консоль",
  "monitoring.tab.public": "Публичная страница",
  "monitoring.tab.incidents": "Аптайм и инциденты",
  "monitoring.detail.not_found_title": "Сервер не найден",
  "monitoring.detail.not_found_text":
    "Мониторинг недоступен. Проверьте ID сервера или права доступа.",
  "monitoring.detail.back_to_list": "К списку мониторинга",
  "monitoring.detail.back_link": "← К списку мониторинга",

  // Консоль
  "monitoring.console.stream_down": "поток недоступен",
  "monitoring.console.stream_live": "поток активен",
  "monitoring.console.logs_failed": "Не удалось получить логи — агент недоступен.",
  "monitoring.console.no_records": "Записей нет.",

  // Настройки публичной страницы
  "monitoring.public_tab.enabled": "включена",
  "monitoring.public_tab.disabled": "выключена",
  "monitoring.public_tab.state": "Публичная страница {state}",
  "monitoring.public_tab.state_hint":
    "Доступна всем по прямой ссылке и в общем рейтинге",
  "monitoring.public_tab.link": "Ссылка",
  "monitoring.public_tab.link_copied": "Ссылка скопирована",
  "monitoring.public_tab.description": "Описание для игроков",
  "monitoring.public_tab.website": "Сайт",
  "monitoring.public_tab.tags": "Теги",
  "monitoring.public_tab.tag_remove": "Удалить тег {tag}",
  "monitoring.public_tab.tag_placeholder": "+ тег",
  "monitoring.public_tab.preview": "Предпросмотр",
  "monitoring.public_tab.visible_title": "Что видно публично",
  "monitoring.public_tab.show_players": "Список игроков",
  "monitoring.public_tab.show_chart": "График онлайна",
  "monitoring.public_tab.show_incidents": "Аптайм и инциденты",
  "monitoring.public_tab.show_address": "IP и порт",
  "monitoring.public_tab.show_version": "Версия и карта",
  "monitoring.public_tab.visits": "Переходы за 7 дней",
  "monitoring.public_tab.visits_bar": "{count} переходов",
  "monitoring.public_tab.votes": "Голоса",
  "monitoring.public_tab.votes_hint":
    "Игроки голосуют с публичной страницы — один голос с адреса в сутки.",
  "monitoring.settings.saved": "Настройки сохранены",
  "monitoring.settings.save_failed": "Не удалось сохранить настройки",

  // Аптайм и инциденты
  "monitoring.incidents.uptime_30d": "Аптайм за 30 дней",
  "monitoring.incidents.day_tooltip": "{day}: {uptime}%",
  "monitoring.incidents.day_no_data": "{day}: нет данных",
  "monitoring.incidents.range_start": "30 дней назад",
  "monitoring.incidents.range_end": "сегодня",
  "monitoring.incidents.title": "Инциденты",
  "monitoring.incidents.fallback_title": "Инцидент",
  "monitoring.incidents.empty": "Инцидентов не было — сервер отвечал на все запросы.",
  "monitoring.incidents.resolved": "Устранён",
  "monitoring.incidents.active": "Активен",
};
