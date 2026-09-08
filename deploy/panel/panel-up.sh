#!/bin/sh
set -eu

DIR="$(CDPATH= cd "$(dirname "$0")" && pwd)"
ENV_FILE="${ENV_FILE:-$DIR/.env}"
PANEL_CONTAINER="${PANEL_CONTAINER:-vortanix-panel-ui}"
UPDATER_CONTAINER="${UPDATER_CONTAINER:-vortanix-updater}"

if [ ! -f "$ENV_FILE" ]; then
  echo "error: $ENV_FILE не найден — cp .env.example .env и заполните LICENSE_KEY"
  exit 1
fi

# Не заменять на `. "$ENV_FILE"`: этот же файл читает docker через
# --env-file и берёт значения буквально, поэтому кавычек в нём нет —
# а без кавычек оболочка спотыкается о пробел в названии бренда.
while IFS= read -r line || [ -n "$line" ]; do
  case "$line" in
    ''|\#*) continue ;;
    *=*) ;;
    *) continue ;;
  esac
  key=${line%%=*}
  value=${line#*=}
  value=${value%$(printf '\r')}
  case "$key" in
    *[!A-Za-z0-9_]*|'') continue ;;
  esac
  export "$key=$value"
done < "$ENV_FILE"

if [ -z "${LICENSE_KEY:-}" ]; then
  echo "error: в $ENV_FILE нужен LICENSE_KEY — ключ из кабинета Vortanix"
  exit 1
fi

PANEL_IMAGE="${PANEL_IMAGE:-registry.vortanix.app/vortanix/panel-ui}"
PANEL_VERSION="${PANEL_VERSION:-latest}"
UPDATER_IMAGE="${UPDATER_IMAGE:-registry.vortanix.app/vortanix/updater}"
UPDATER_VERSION="${UPDATER_VERSION:-latest}"
PANEL_PORT="${PANEL_PORT:-3000}"
LICENSE_URL="${LICENSE_URL:-https://license.vortanix.app}"

if ! command -v docker >/dev/null 2>&1; then
  echo "error: docker не найден — apt-get install -y docker.io"
  exit 1
fi

if ! docker info >/dev/null 2>&1; then
  echo "error: демон docker не запущен — systemctl start docker"
  exit 1
fi

if [ -n "${REGISTRY_HOST:-}" ]; then
  echo "=== вход в реестр $REGISTRY_HOST ==="
  if ! printf '%s' "$LICENSE_KEY" | docker login "$REGISTRY_HOST" \
      --username license --password-stdin >/dev/null 2>&1; then
    echo "error: реестр $REGISTRY_HOST не принял ключ лицензии"
    echo "       проверьте LICENSE_KEY в $ENV_FILE и что лицензия не отозвана"
    exit 1
  fi
fi

echo "=== образы ==="
docker pull "$PANEL_IMAGE:$PANEL_VERSION"
docker pull "$UPDATER_IMAGE:$UPDATER_VERSION"

echo "=== панель ==="
docker rm -f "$PANEL_CONTAINER" 2>/dev/null || true
docker run -d \
  --name "$PANEL_CONTAINER" \
  --restart unless-stopped \
  --env-file "$ENV_FILE" \
  -p "$PANEL_PORT:3000" \
  "$PANEL_IMAGE:$PANEL_VERSION"

# Файл окружения монтируется на запись (было :ro): служба правит в нём
# PANEL_VERSION и UPDATER_VERSION после обновления, иначе запуск этого скрипта
# руками откатывает панель на версию времён установки. Доступа это не
# расширяет — у службы и так проброшен docker.sock, то есть полномочия root на
# хосте; запрет на запись в один файл не защищал ни от чего.
echo "=== апдейтер ==="
docker rm -f "$UPDATER_CONTAINER" 2>/dev/null || true
docker run -d \
  --name "$UPDATER_CONTAINER" \
  --restart unless-stopped \
  -e "LICENSE_URL=$LICENSE_URL" \
  -e "LICENSE_KEY=$LICENSE_KEY" \
  -e "PANEL_CONTAINER=$PANEL_CONTAINER" \
  -e "PANEL_PORT=$PANEL_PORT" \
  -e "PANEL_ENV_FILE=/etc/vortanix/panel.env" \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v "$ENV_FILE:/etc/vortanix/panel.env" \
  "$UPDATER_IMAGE:$UPDATER_VERSION"

echo "=== проверка ==="
sleep 3
docker ps --filter "name=$PANEL_CONTAINER" --filter "name=$UPDATER_CONTAINER"
echo
echo "Панель: http://localhost:$PANEL_PORT"
echo "Логи апдейтера: docker logs -f $UPDATER_CONTAINER"
