export const panelTransfer = {
  "admin.panel_transfer.title": "Перенос панели",
  "admin.panel_transfer.subtitle":
    "Переезд панели на другой сервер целиком: база, секреты и загруженные файлы",

  "admin.panel_transfer.tab.ssh": "По SSH",
  "admin.panel_transfer.tab.archive": "Архивом",

  "admin.panel_transfer.overview.title": "Что переедет",
  "admin.panel_transfer.overview.version": "Версия панели",
  "admin.panel_transfer.overview.db": "База",
  "admin.panel_transfer.overview.uploads": "Файлы",
  "admin.panel_transfer.overview.secrets": "Секретов в .env",
  "admin.panel_transfer.overview.address": "Текущий адрес",
  "admin.panel_transfer.overview.mode": "Режим установки",
  "admin.panel_transfer.overview.nodes": "Локаций",

  "admin.panel_transfer.skipped.title": "Что не переносится",
  "admin.panel_transfer.skipped.redis": "Кэш Redis — соберётся заново",
  "admin.panel_transfer.skipped.certs": "Сертификат Let's Encrypt — выпустится заново",
  "admin.panel_transfer.skipped.servers": "Файлы игровых серверов — остаются на локациях",

  "admin.panel_transfer.ssh.title": "Новый сервер",
  "admin.panel_transfer.ssh.hint":
    "Нужна чистая Debian или Ubuntu с root-доступом и свободными портами 80 и 443. Панель установится сама, затем в неё переедут база, секреты и файлы",

  "admin.panel_transfer.target.host": "Адрес сервера",
  "admin.panel_transfer.target.port": "Порт SSH",
  "admin.panel_transfer.target.user": "Пользователь",
  "admin.panel_transfer.target.auth_password": "Пароль",
  "admin.panel_transfer.target.auth_key": "Приватный ключ",
  "admin.panel_transfer.target.password": "Пароль SSH",
  "admin.panel_transfer.target.private_key": "Приватный ключ SSH",

  "admin.panel_transfer.address.same": "Тот же домен",
  "admin.panel_transfer.address.same_hint":
    "Панель на новом сервере будет отвечать на {address}. Переключите DNS после переноса — узлы подключатся сами",
  "admin.panel_transfer.address.new": "Адрес новой панели",

  "admin.panel_transfer.freeze.label": "Остановить запись на время переноса",
  "admin.panel_transfer.freeze.hint":
    "Старая панель перейдёт в режим только чтение, чтобы данные не разъехались с копией. Режим останется включённым и после переноса",

  "admin.panel_transfer.start": "Начать перенос",
  "admin.panel_transfer.starting": "Запускаю…",
  "admin.panel_transfer.started": "Перенос запущен",
  "admin.panel_transfer.cancel": "Отменить перенос",
  "admin.panel_transfer.cancelled": "Перенос отменён",
  "admin.panel_transfer.unfreeze": "Снять режим только чтение",
  "admin.panel_transfer.frozen_banner": "Панель работает в режиме только чтение",

  "admin.panel_transfer.confirm.start":
    "Начать перенос панели на другой сервер? На нём будет установлена панель этой же версии, а затем в неё переедут база, секреты и файлы. Эта панель останется работать",
  "admin.panel_transfer.confirm.start_ok": "Начать",
  "admin.panel_transfer.confirm.unfreeze":
    "Снять режим только чтение? Если панель уже переехала, изменения здесь не попадут на новый сервер",
  "admin.panel_transfer.confirm.unfreeze_ok": "Снять",

  "admin.panel_transfer.progress.title": "Ход переноса",
  "admin.panel_transfer.progress.bytes": "Передано {done} из {total}",
  "admin.panel_transfer.progress.agents": "Узлы переключены",
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
  "admin.panel_transfer.stage.install_docker": "Docker",
  "admin.panel_transfer.stage.clone": "Файлы панели",
  "admin.panel_transfer.stage.init_env": "Настройки",
  "admin.panel_transfer.stage.compose_up": "Запуск",
  "admin.panel_transfer.stage.transfer": "Передача данных",
  "admin.panel_transfer.stage.health": "Проверка панели",
  "admin.panel_transfer.stage.relay_cert": "Сертификат relay",
  "admin.panel_transfer.stage.agents": "Узлы",
  "admin.panel_transfer.stage.done": "Готово",

  "admin.panel_transfer.export.title": "Выгрузить архив",
  "admin.panel_transfer.export.hint":
    "Архив содержит базу, секреты и загруженные файлы. Его можно загрузить в чистую панель такой же или более новой версии",
  "admin.panel_transfer.export.password": "Пароль архива (не короче 12 символов)",
  "admin.panel_transfer.export.warning":
    "В архиве лежит ключ шифрования SECRETS_KEY: им зашифрованы ключи касс и пароли локаций. Храните файл как пароль",
  "admin.panel_transfer.export.action": "Скачать архив",
  "admin.panel_transfer.export.busy": "Готовлю архив…",
  "admin.panel_transfer.export.done": "Архив готов",
  "admin.panel_transfer.export.failed": "Архив не выгружен",

  "admin.panel_transfer.import.title": "Загрузить архив",
  "admin.panel_transfer.import.hint":
    "Данные этой панели будут заменены данными из архива. Делайте это только на новой панели",
  "admin.panel_transfer.import.file": "Файл архива",
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
    "Не переключено узлов: {count}. Укажите доступ к новому серверу и повторите — панель пересоздаст агентов с новым адресом",
  "admin.panel_transfer.agents.retry": "Повторить переключение",
  "admin.panel_transfer.agents.started": "Переключение узлов запущено",

  "admin.panel_transfer.nodes.title": "Локации",
  "admin.panel_transfer.nodes.empty": "Локаций нет",
};
