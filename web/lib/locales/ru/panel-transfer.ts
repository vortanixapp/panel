export const panelTransfer = {
  "admin.panel_transfer.title": "Перенос панели",
  "admin.panel_transfer.subtitle":
    "Переезд панели на другой сервер целиком: база, секреты и загруженные файлы",
  "admin.panel_transfer.read_only": "только чтение",

  "admin.panel_transfer.tab.ssh": "По SSH",
  "admin.panel_transfer.tab.archive": "Архивом",
  "admin.panel_transfer.mode.ssh_desc":
    "Панель сама поставит себя на новый сервер и перельёт данные",
  "admin.panel_transfer.mode.archive_desc":
    "Скачать шифрованный архив и загрузить его в новую панель вручную",

  "admin.panel_transfer.tile.db_sub": "дамп PostgreSQL",
  "admin.panel_transfer.tile.uploads_sub": "том uploads",
  "admin.panel_transfer.tile.secrets_sub": "строк из deploy/.env",
  "admin.panel_transfer.tile.nodes_sub": "локаций с агентами",

  "admin.panel_transfer.overview.title": "Состав переноса",
  "admin.panel_transfer.overview.version": "Версия панели",
  "admin.panel_transfer.overview.db": "База",
  "admin.panel_transfer.overview.uploads": "Файлы",
  "admin.panel_transfer.overview.secrets": "Секреты",
  "admin.panel_transfer.overview.address": "Текущий адрес",
  "admin.panel_transfer.overview.mode": "Режим установки",
  "admin.panel_transfer.overview.nodes": "Локации",
  "admin.panel_transfer.overview.total": "Всего данных",

  "admin.panel_transfer.moves.db": "База: пользователи, серверы, платежи, настройки",
  "admin.panel_transfer.moves.uploads": "Файлы: аватары, логотип, вложения обращений",
  "admin.panel_transfer.moves.secrets": "Секреты: ключ шифрования, почта, OAuth, реестр",
  "admin.panel_transfer.skipped.redis": "Кэш Redis — соберётся заново",
  "admin.panel_transfer.skipped.certs": "Сертификат Let's Encrypt — выпустится заново",
  "admin.panel_transfer.skipped.servers": "Файлы игровых серверов — остаются на локациях",

  "admin.panel_transfer.requirements.title": "Требования к новому серверу",
  "admin.panel_transfer.requirements.os": "Debian или Ubuntu, x86_64 или aarch64",
  "admin.panel_transfer.requirements.root": "Доступ по SSH под root или sudo без пароля",
  "admin.panel_transfer.requirements.ports": "Свободные порты 80 и 443",
  "admin.panel_transfer.requirements.space": "Свободно не меньше {size} плюс запас 5 ГБ",

  "admin.panel_transfer.ssh.title": "Новый сервер",
  "admin.panel_transfer.ssh.hint":
    "Docker установится сам, панель развернётся той же версии, затем в неё переедут база, секреты и файлы. Эта панель продолжит работать",
  "admin.panel_transfer.ssh.access": "Доступ к серверу",

  "admin.panel_transfer.target.host": "Адрес сервера",
  "admin.panel_transfer.target.port": "Порт SSH",
  "admin.panel_transfer.target.user": "Пользователь",
  "admin.panel_transfer.target.auth_password": "Пароль",
  "admin.panel_transfer.target.auth_key": "Приватный ключ",
  "admin.panel_transfer.target.password": "Пароль SSH",
  "admin.panel_transfer.target.private_key": "Приватный ключ SSH",

  "admin.panel_transfer.address.section": "Адрес новой панели",
  "admin.panel_transfer.address.same": "Тот же домен",
  "admin.panel_transfer.address.same_desc": "Меняется только DNS, узлы не перенастраиваются",
  "admin.panel_transfer.address.new": "Новый адрес",
  "admin.panel_transfer.address.new_desc": "Панель переключит узлы на новый адрес сама",
  "admin.panel_transfer.address.new_label": "Домен или IP новой панели",
  "admin.panel_transfer.address.same_hint":
    "Новая панель будет отвечать на {address}. Переключите DNS после переноса — узлы подключатся сами",

  "admin.panel_transfer.freeze.label": "Остановить запись на время переноса",
  "admin.panel_transfer.freeze.hint":
    "Панель перейдёт в режим только чтение, чтобы данные не разъехались с копией. Режим останется включённым и после переноса",

  "admin.panel_transfer.start": "Начать перенос",
  "admin.panel_transfer.starting": "Запускаю…",
  "admin.panel_transfer.started": "Перенос запущен",
  "admin.panel_transfer.cancel": "Отменить перенос",
  "admin.panel_transfer.cancelled": "Перенос отменён",
  "admin.panel_transfer.busy_hint": "Пока перенос идёт, запустить второй нельзя",
  "admin.panel_transfer.unfreeze": "Снять режим только чтение",
  "admin.panel_transfer.frozen_banner":
    "Панель работает в режиме только чтение: изменения не принимаются, задачи не берутся",

  "admin.panel_transfer.confirm.start":
    "Начать перенос панели на другой сервер? На нём будет установлена панель этой же версии, а затем в неё переедут база, секреты и файлы. Эта панель останется работать",
  "admin.panel_transfer.confirm.start_ok": "Начать",
  "admin.panel_transfer.confirm.unfreeze":
    "Снять режим только чтение? Если панель уже переехала, изменения здесь не попадут на новый сервер",
  "admin.panel_transfer.confirm.unfreeze_ok": "Снять",

  "admin.panel_transfer.progress.title": "Ход переноса",
  "admin.panel_transfer.progress.log": "Журнал",
  "admin.panel_transfer.progress.log_show": "Показать",
  "admin.panel_transfer.progress.log_hide": "Скрыть",
  "admin.panel_transfer.progress.running_hint":
    "Страницу можно закрыть — перенос идёт на сервере и продолжится без неё",

  "admin.panel_transfer.status.pending": "В очереди",
  "admin.panel_transfer.status.running": "Идёт",
  "admin.panel_transfer.status.completed": "Завершён",
  "admin.panel_transfer.status.failed": "Ошибка",
  "admin.panel_transfer.status.cancelled": "Отменён",

  "admin.panel_transfer.stage.prepare": "Подготовка",
  "admin.panel_transfer.stage.freeze": "Только чтение",
  "admin.panel_transfer.stage.probe_target": "Проверка сервера",
  "admin.panel_transfer.stage.install_docker": "Установка Docker",
  "admin.panel_transfer.stage.clone": "Файлы панели",
  "admin.panel_transfer.stage.init_env": "Настройки и адрес",
  "admin.panel_transfer.stage.compose_up": "Запуск панели",
  "admin.panel_transfer.stage.transfer": "Передача данных",
  "admin.panel_transfer.stage.health": "Проверка панели",
  "admin.panel_transfer.stage.relay_cert": "Сертификат relay",
  "admin.panel_transfer.stage.agents": "Переключение узлов",
  "admin.panel_transfer.stage.done": "Готово",

  "admin.panel_transfer.export.title": "Выгрузить архив",
  "admin.panel_transfer.export.hint":
    "Архив содержит базу, секреты и загруженные файлы. Его можно загрузить в чистую панель такой же или более новой версии",
  "admin.panel_transfer.export.password": "Пароль архива",
  "admin.panel_transfer.export.too_short": "Не короче 12 символов",
  "admin.panel_transfer.export.warning":
    "В архиве лежит ключ шифрования SECRETS_KEY: им зашифрованы ключи касс и пароли локаций. Храните файл как пароль",
  "admin.panel_transfer.export.action": "Скачать архив",
  "admin.panel_transfer.export.busy": "Готовлю архив…",
  "admin.panel_transfer.export.done": "Архив готов",
  "admin.panel_transfer.export.failed": "Архив не выгружен",

  "admin.panel_transfer.import.title": "Загрузить архив",
  "admin.panel_transfer.import.hint":
    "Данные этой панели будут заменены данными из архива. Делайте это только на новой панели",
  "admin.panel_transfer.import.pick": "Выберите файл архива",
  "admin.panel_transfer.import.pick_hint": "Файл с расширением .vxt",
  "admin.panel_transfer.import.password": "Пароль архива",
  "admin.panel_transfer.import.inspect": "Проверить архив",
  "admin.panel_transfer.import.inspect_failed": "Архив не прочитан",
  "admin.panel_transfer.import.action": "Заменить данные панели",
  "admin.panel_transfer.import.busy": "Восстанавливаю…",
  "admin.panel_transfer.import.done": "Данные восстановлены",
  "admin.panel_transfer.import.failed": "Данные не восстановлены",
  "admin.panel_transfer.import.warning":
    "Текущая база, секреты и файлы этой панели будут перезаписаны. Ключ шифрования тоже придёт из архива",
  "admin.panel_transfer.import.archive_version": "Версия панели",
  "admin.panel_transfer.import.archive_created": "Снят",
  "admin.panel_transfer.import.archive_db": "База",
  "admin.panel_transfer.import.archive_uploads": "Файлы",
  "admin.panel_transfer.import.archive_source": "Адрес источника",

  "admin.panel_transfer.agents.title": "Узлы не переключились",
  "admin.panel_transfer.agents.hint":
    "Не переключено узлов: {count}. Укажите доступ к новому серверу выше и повторите — панель пересоздаст агентов с новым адресом",
  "admin.panel_transfer.agents.retry": "Повторить переключение",
  "admin.panel_transfer.agents.started": "Переключение узлов запущено",

  "admin.panel_transfer.nodes.title": "Локации",
  "admin.panel_transfer.nodes.empty": "Локаций нет",
};
