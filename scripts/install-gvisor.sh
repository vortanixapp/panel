#!/bin/sh
# Установка gVisor (runsc) на игровой узел из официального apt-репозитория
# и регистрация его в Docker. Запускать на узле от root:
#   sh install-gvisor.sh [--no-reload]
set -eu

MIN_RELEASE=20240101

RELOAD=1
for arg in "$@"; do
  case "$arg" in
    --no-reload) RELOAD=0 ;;
    *) echo "неизвестный параметр: $arg" >&2; exit 2 ;;
  esac
done

if [ "$(id -u)" -ne 0 ]; then
  echo "запустите от root: sudo sh $0" >&2
  exit 1
fi

for tool in curl docker; do
  command -v "$tool" >/dev/null 2>&1 || { echo "не найдено: $tool" >&2; exit 1; }
done

if ! command -v apt-get >/dev/null 2>&1; then
  echo "Скрипт рассчитан на Debian и Ubuntu (apt). Для других систем установите gVisor по" >&2
  echo "инструкции https://gvisor.dev/docs/user_guide/install/ и выполните: runsc install" >&2
  exit 1
fi

release_of() {
  "$1" --version 2>/dev/null | sed -n 's/.*release-\([0-9]\{8\}\).*/\1/p' | head -n 1
}

for old in /usr/local/bin/runsc; do
  if [ -x "$old" ]; then
    version="$(release_of "$old")"
    if [ -z "$version" ] || [ "$version" -lt "$MIN_RELEASE" ]; then
      echo "Найдена устаревшая сборка ${old} (${version:-неизвестная версия}), удаляю: её ставили прежние версии скрипта."
      rm -f /usr/local/bin/runsc /usr/local/bin/containerd-shim-runsc-v1
    fi
  fi
done

export DEBIAN_FRONTEND=noninteractive
apt-get update -qq
apt-get install -y -qq apt-transport-https ca-certificates curl gnupg

KEYRING=/usr/share/keyrings/gvisor-archive-keyring.gpg
curl -fsSL https://gvisor.dev/archive.key | gpg --dearmor --yes -o "$KEYRING"
echo "deb [arch=$(dpkg --print-architecture) signed-by=${KEYRING}] https://storage.googleapis.com/gvisor/releases release main" \
  > /etc/apt/sources.list.d/gvisor.list

apt-get update -qq
apt-get install -y -qq runsc

RUNSC="$(command -v runsc || true)"
if [ -z "$RUNSC" ]; then
  echo "runsc не найден после установки пакета" >&2
  exit 1
fi

VERSION="$(release_of "$RUNSC")"
if [ -z "$VERSION" ] || [ "$VERSION" -lt "$MIN_RELEASE" ]; then
  echo "Установлена слишком старая версия runsc (${VERSION:-неизвестная}) по пути ${RUNSC}." >&2
  echo "Проверьте PATH и удалите старые копии: which -a runsc" >&2
  exit 1
fi
echo "Установлен runsc ${VERSION} (${RUNSC})."

"$RUNSC" install

if [ "$RELOAD" -eq 1 ]; then
  systemctl reload docker 2>/dev/null || kill -HUP "$(pidof dockerd)"
fi

RUNTIMES="$(docker info --format '{{json .Runtimes}}')"
if ! printf '%s' "$RUNTIMES" | grep -q '"runsc"'; then
  echo "runsc не появился в docker info. Если запускали с --no-reload, перезагрузите Docker: systemctl reload docker" >&2
  exit 1
fi
if ! printf '%s' "$RUNTIMES" | grep -q "\"path\":\"${RUNSC}\""; then
  echo "Внимание: в /etc/docker/daemon.json окружение runsc указывает не на ${RUNSC}." >&2
  echo "Откройте файл, поправьте путь у runsc и выполните: systemctl reload docker" >&2
  exit 1
fi
echo "runsc зарегистрирован в Docker."

TEST_IMAGE="busybox:stable"
if ! docker image inspect "$TEST_IMAGE" >/dev/null 2>&1; then
  if ! PULL_OUT="$(docker pull "$TEST_IMAGE" 2>&1)"; then
    echo "Не удалось скачать $TEST_IMAGE для проверки. gVisor установлен, но проверка пропущена:" >&2
    echo "$PULL_OUT" >&2
    exit 1
  fi
fi

if RUN_OUT="$(docker run --rm --runtime=runsc "$TEST_IMAGE" true 2>&1)"; then
  echo "Проверочный контейнер под gVisor запустился."
else
  echo "Проверочный контейнер под gVisor не запустился. Ответ Docker:" >&2
  echo "$RUN_OUT" >&2
  echo "Подробности: journalctl -u docker -n 50 --no-pager" >&2
  exit 1
fi

echo "Готово. Включите песочницу в панели: Админка, Настройки, Узлы, раздел «Песочница gVisor»."
