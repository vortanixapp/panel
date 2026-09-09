#!/bin/sh
set -eu

ROOT="$(CDPATH= cd "$(dirname "$0")/../.." && pwd)"
DIR="$(CDPATH= cd "$(dirname "$0")" && pwd)"
ENV_FILE="${ENV_FILE:-$DIR/.env}"
CONTAINER="${AGENT_CONTAINER:-vortanix-agent}"

if [ ! -f "$ENV_FILE" ]; then
  echo "error: $ENV_FILE not found — cp .env.example .env and set AGENT_TOKEN + NODE_ID"
  exit 1
fi

# shellcheck disable=SC1090
set -a
. "$ENV_FILE"
set +a

if [ -z "${AGENT_TOKEN:-}" ] || [ -z "${NODE_ID:-}" ]; then
  echo "error: AGENT_TOKEN and NODE_ID required in $ENV_FILE"
  exit 1
fi

RELAY_URL="${RELAY_URL:?укажите RELAY_URL — внешний адрес relay панели}"

if ! command -v docker >/dev/null 2>&1; then
  echo "error: docker not found — apt-get install -y docker.io"
  exit 1
fi

if ! docker info >/dev/null 2>&1; then
  echo "error: docker daemon not running — systemctl start docker"
  exit 1
fi

mkdir -p /var/lib/vortanix/servers

echo "=== build agent ==="
docker build -f "$ROOT/apps/agent/Dockerfile" -t vortanix/agent:latest "$ROOT"

echo "=== stop old container ==="
docker stop "$CONTAINER" 2>/dev/null || true
docker rm "$CONTAINER" 2>/dev/null || true

echo "=== start agent ==="
docker run -d \
  --name "$CONTAINER" \
  --restart unless-stopped \
  -e "RELAY_URL=$RELAY_URL" \
  -e "AGENT_TOKEN=$AGENT_TOKEN" \
  -e "NODE_ID=$NODE_ID" \
  -e "VORTANIX_DATA_DIR=/var/lib/vortanix/servers" \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v /var/lib/vortanix/servers:/var/lib/vortanix/servers \
  vortanix/agent:latest

echo "=== logs (last 20 lines) ==="
sleep 2
docker logs "$CONTAINER" --tail 20
echo
echo "Done. Container: $CONTAINER"
docker ps --filter "name=$CONTAINER"
