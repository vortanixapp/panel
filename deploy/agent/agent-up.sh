#!/bin/sh
set -eu

ROOT="$(CDPATH= cd "$(dirname "$0")/../.." && pwd)"
DIR="$(CDPATH= cd "$(dirname "$0")" && pwd)"
ENV_FILE="${ENV_FILE:-$DIR/.env}"
CONTAINER="${AGENT_CONTAINER:-vortanix-agent}"
BUILD=0

usage() {
  cat <<'TXT'
Поднимает агент Vortanix на игровой ноде.

  sh deploy/agent/agent-up.sh            взять готовый образ
  sh deploy/agent/agent-up.sh --build    собрать образ из исходников

Настройки берутся из deploy/agent/.env — скопируйте .env.example и впишите
RELAY_URL, AGENT_TOKEN и NODE_ID из раздела «Локации» в панели.

Панель выдаёт готовую команду установки там же. Этот скрипт нужен, когда
ноду поднимают заранее или разворачивают из системы управления конфигурацией.
TXT
}

for arg in "$@"; do
  case "$arg" in
    --build) BUILD=1 ;;
    -h|--help) usage; exit 0 ;;
    *) echo "неизвестный флаг: $arg" >&2; exit 2 ;;
  esac
done

if [ ! -f "$ENV_FILE" ]; then
  echo "Не найден $ENV_FILE — скопируйте .env.example в .env и заполните" >&2
  exit 1
fi

# shellcheck disable=SC1090
set -a
. "$ENV_FILE"
set +a

if [ -z "${AGENT_TOKEN:-}" ] || [ -z "${NODE_ID:-}" ]; then
  echo "В $ENV_FILE нужны AGENT_TOKEN и NODE_ID — их выдаёт панель при создании локации" >&2
  exit 1
fi

RELAY_URL="${RELAY_URL:?укажите RELAY_URL — внешний адрес relay панели}"
AGENT_IMAGE="${AGENT_IMAGE:-ghcr.io/vortanixapp/vortanix-agent:latest}"

if ! command -v docker >/dev/null 2>&1; then
  echo "Docker не установлен: curl -fsSL https://get.docker.com | sh" >&2
  exit 1
fi

if ! docker info >/dev/null 2>&1; then
  echo "Демон Docker не запущен: systemctl start docker" >&2
  exit 1
fi

mkdir -p /var/lib/vortanix/servers

if [ "$BUILD" -eq 1 ]; then
  AGENT_IMAGE="vortanix/agent:local"
  echo "=== сборка образа ==="
  docker build -f "$ROOT/Dockerfile" --build-arg COMPONENT=vortanix-agent \
    -t "$AGENT_IMAGE" "$ROOT"
else
  echo "=== загрузка образа $AGENT_IMAGE ==="
  docker pull "$AGENT_IMAGE"
fi

echo "=== остановка прежнего контейнера ==="
docker stop "$CONTAINER" 2>/dev/null || true
docker rm "$CONTAINER" 2>/dev/null || true

echo "=== запуск агента ==="
docker run -d \
  --name "$CONTAINER" \
  --restart unless-stopped \
  -e "RELAY_URL=$RELAY_URL" \
  -e "AGENT_TOKEN=$AGENT_TOKEN" \
  -e "NODE_ID=$NODE_ID" \
  -e "VORTANIX_DATA_DIR=/var/lib/vortanix/servers" \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v /var/lib/vortanix/servers:/var/lib/vortanix/servers \
  "$AGENT_IMAGE"

echo "=== журнал, последние 20 строк ==="
sleep 2
docker logs "$CONTAINER" --tail 20
echo
docker ps --filter "name=$CONTAINER"
