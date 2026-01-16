# Vortanix CS 1.6 Docker Image

Docker образ для Counter-Strike 1.6 серверов, интегрированный с платформой Vortanix.

## Описание

Этот Docker образ содержит полностью настроенный Counter-Strike 1.6 сервер с:
- Поддержкой стандартных карт (de_dust2, de_nuke, de_train, de_inferno, cs_office)
- Автоматической генерацией RCON пароля
- Настройками производительности
- Интеграцией с Vortanix daemon
- Установкой через SteamCMD (официальный метод)

## Установка

CS 1.6 сервер устанавливается через SteamCMD с использованием официальных файлов Valve:
- App ID: 90 (Counter-Strike)
- Мод: cstrike
- Анонимная авторизация Steam

## Структура файлов

```
vortanix-daemon/docker/cs16/
├── Dockerfile          # Основной Dockerfile
├── entrypoint.sh       # Скрипт запуска сервера
└── README.md          # Данная документация
```

## Сборка образа

```bash
# Перейдите в директорию с Dockerfile
cd /home/instor/vortanix/vortanix-daemon/docker/cs16/

# Соберите образ
docker build -t vortanix/cs16:1.0.0-r1 .

# Или с другим тегом
docker build -t myregistry.com/vortanix/cs16:latest .
```

## Переменные окружения

### Обязательные переменные

- `CS16_PORT` - Порт сервера (1-65535)

### Опциональные переменные

- `CS16_HOSTNAME` - Название сервера (по умолчанию: "Vortanix CS 1.6 Server")
- `CS16_RCON_PASSWORD` - RCON пароль (генерируется автоматически если не указан)
- `CS16_MAXPLAYERS` - Максимальное количество игроков (по умолчанию: 32)
- `CS16_WEBURL` - Контактная информация (по умолчанию: "vortanix.local")

## Запуск контейнера

### Базовый запуск

```bash
docker run -d \
  --name cs16-server-1 \
  -p 27015:27015/udp \
  -p 27015:27015/tcp \
  -e CS16_PORT=27015 \
  -e CS16_HOSTNAME="My CS 1.6 Server" \
  -v cs16-data-1:/data \
  vortanix/cs16:1.0.0-r1
```

### С Vortanix интеграцией

```bash
docker run -d \
  --name vtx-cs16-123 \
  -p 27015:27015/udp \
  -p 27015:27015/tcp \
  -e CS16_PORT=27015 \
  -e CS16_HOSTNAME="Vortanix CS 1.6 Server" \
  -e CS16_RCON_PASSWORD="secure_password_here" \
  -e CS16_MAXPLAYERS=32 \
  -v /opt/vortanix/servers/123/data:/data \
  vortanix/cs16:1.0.0-r1
```

## Интеграция с Vortanix Daemon

### Поддерживаемые коды игр

Daemon автоматически выбирает правильный Docker образ для следующих кодов игр:
- `cs16`
- `counter-strike`
- `counter_strike`
- `cs_1_6`

### Конфигурация Daemon

В файле `/home/instor/vortanix/vortanix-daemon/vtxdaemon/config.py`:

```python
CS16_DOCKER_IMAGE = os.getenv('CS16_DOCKER_IMAGE', 'vortanix/cs16:1.0.0-r1')
```

### API Endpoints

Daemon поддерживает следующие операции для CS 1.6 серверов:
- `POST /servers/create` - Создание сервера
- `POST /servers/reinstall` - Переустановка сервера
- `POST /servers/start` - Запуск сервера
- `POST /servers/stop` - Остановка сервера
- `POST /servers/restart` - Перезагрузка сервера
- `GET /servers/{id}/status` - Получение статуса
- `GET /servers/{id}/logs` - Получение логов
- `POST /servers/{id}/rcon` - Отправка RCON команд

## Настройки сервера

### Стандартные настройки

Сервер поставляется с оптимизированными настройками:
- Максимум 32 игрока
- Регион: Европа (sv_region 3)
- LAN режим отключен (sv_lan 0)
- Автоматический баланс команд
- Включен friendly fire
- Стандартное время раунда 5 минут
- Freezetime 5 секунд

### Изменение настроек

Настройки можно изменить через:
1. Переменные окружения при запуске
2. RCON команды во время работы
3. Редактирование файла server.cfg в data директории

## Мониторинг и логи

### Просмотр логов

```bash
# Логи контейнера
docker logs cs16-server-1

# Логи сервера в реальном времени
docker logs -f cs16-server-1

# Логи сервера из data директории
docker exec cs16-server-1 tail -f /data/server_log.txt
```

### Мониторинг состояния

```bash
# Статус контейнера
docker ps | grep cs16

# Статистика ресурсов
docker stats cs16-server-1

# Подключение к консоли сервера
docker exec -it cs16-server-1 /bin/sh
```

## Управление данными

### Data Volume

Данные сервера сохраняются в Docker volume:
- Карты: `/data/cstrike/maps/`
- Конфигурация: `/data/server.cfg`
- Логи: `/data/server_log.txt`
- Моды и плагины: `/data/cstrike/`

### Резервное копирование

```bash
# Создание резервной копии
docker run --rm -v cs16-data-1:/data -v $(pwd):/backup \
  alpine tar czf /backup/cs16-backup.tar.gz -C /data .

# Восстановление из резервной копии
docker run --rm -v cs16-data-1:/data -v $(pwd):/backup \
  alpine tar xzf /backup/cs16-backup.tar.gz -C /data
```

## Безопасность

### Рекомендации

1. **Используйте сильные RCON пароли**
2. **Ограничьте доступ к портам через firewall**
3. **Регулярно обновляйте образ**
4. **Мониторьте логи на предмет подозрительной активности**

### Сетевая безопасность

По умолчанию открыты порты:
- `27015/udp` - Игровой трафик
- `27015/tcp` - RCON и query

Убедитесь, что эти порты правильно настроены в firewall.

## Устранение неполадок

### Частые проблемы

1. **Сервер не запускается**
   - Проверьте переменную CS16_PORT
   - Убедитесь, что порт не занят другим процессом
   - Проверьте логи: `docker logs <container_name>`

2. **Игроки не могут подключиться**
   - Проверьте, что порты открыты в firewall
   - Убедитесь, что контейнер запущен
   - Проверьте настройки sv_lan

3. **Проблемы с производительностью**
   - Уменьшите maxplayers
   - Ограничьте количество карт
   - Проверьте ресурсы хоста

### Отладка

```bash
# Подробные логи
docker run -it --rm \
  -e CS16_PORT=27015 \
  -e CS16_HOSTNAME="Debug Server" \
  vortanix/cs16:1.0.0-r1

# Проверка файлов в контейнере
docker run -it --rm -e CS16_PORT=27015 vortanix/cs16:1.0.0-r1 /bin/sh

# Тест сетевого подключения
docker exec <container_name> netstat -ulnp | grep 27015
```

## Разработка

### Локальная разработка

```bash
# Сборка с тегами для разработки
docker build -t vortanix/cs16:dev .

# Запуск в режиме разработки
docker run -it --rm \
  -p 27015:27015/udp \
  -e CS16_PORT=27015 \
  -e CS16_HOSTNAME="Dev Server" \
  vortanix/cs16:dev
```

### Внесение изменений

1. Измените Dockerfile или entrypoint.sh
2. Пересоберите образ
3. Протестируйте изменения
4. Обновите версию в теге

## Лицензия

Этот проект является частью платформы Vortanix и распространяется согласно лицензионным соглашениям Vortanix.

## Поддержка

Для получения поддержки обратитесь к документации Vortanix или создайте issue в репозитории проекта.
