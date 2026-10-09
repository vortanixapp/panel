#!/bin/sh
# Установка gVisor (runsc) на игровой узел и регистрация его в Docker.
# Запускать на узле от root: sh install-gvisor.sh [--no-reload]
set -eu

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

for tool in curl sha512sum docker; do
  command -v "$tool" >/dev/null 2>&1 || { echo "не найдено: $tool" >&2; exit 1; }
done

case "$(uname -m)" in
  x86_64) ARCH=x86_64 ;;
  aarch64|arm64) ARCH=aarch64 ;;
  *) echo "архитектура $(uname -m) не поддерживается gVisor" >&2; exit 1 ;;
esac

BASE="https://storage.googleapis.com/gvisor/releases/release/latest/${ARCH}"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

cd "$WORK"
for file in runsc containerd-shim-runsc-v1; do
  curl -fsSLO "${BASE}/${file}"
  curl -fsSLO "${BASE}/${file}.sha512"
  sha512sum -c "${file}.sha512"
done

install -m 0755 runsc containerd-shim-runsc-v1 /usr/local/bin/
/usr/local/bin/runsc install

if [ "$RELOAD" -eq 1 ]; then
  systemctl reload docker 2>/dev/null || kill -HUP "$(pidof dockerd)"
fi

if docker info --format '{{json .Runtimes}}' | grep -q '"runsc"'; then
  echo "runsc зарегистрирован в Docker."
else
  echo "runsc не появился в docker info. Если запускали с --no-reload, перезагрузите Docker командой: systemctl reload docker" >&2
  exit 1
fi

if docker run --rm --runtime=runsc busybox:stable true >/dev/null 2>&1; then
  echo "Проверочный контейнер под gVisor запустился."
else
  echo "Проверочный контейнер под gVisor не запустился. Смотрите: journalctl -u docker -n 50" >&2
  exit 1
fi

echo "Готово. Включите песочницу в панели: Админка, Настройки, Узлы, раздел «Песочница gVisor»."
