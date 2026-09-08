#!/bin/sh
set -eu

umask 0002

DATA_DIR="/data"

is_uint() {
  case "${1:-}" in
    ''|*[!0-9]*) return 1 ;;
    *) return 0 ;;
  esac
}

GMOD_PORT="${GMOD_PORT:-}"
if [ "$GMOD_PORT" = "" ]; then
  echo "GMOD_PORT is required" >&2
  exit 1
fi
if ! is_uint "$GMOD_PORT" || [ "$GMOD_PORT" -lt 1 ] || [ "$GMOD_PORT" -gt 65535 ]; then
  echo "Invalid GMOD_PORT: $GMOD_PORT" >&2
  exit 1
fi

MAXPLAYERS="${GMOD_MAXPLAYERS:-24}"
HOSTNAME="${GMOD_HOSTNAME:-Vortanix Garrys Mod Server}"
RCON_PASSWORD="${GMOD_RCON_PASSWORD:-}"
if [ "$RCON_PASSWORD" = "" ]; then
  RCON_PASSWORD="vtx$(cat /proc/sys/kernel/random/uuid | tr -d '-' | cut -c1-20)"
fi

mkdir -p "$DATA_DIR"

if [ ! -x "$DATA_DIR/srcds_run" ] || [ ! -d "$DATA_DIR/garrysmod" ]; then
  echo "[gmod] ERROR: Garry's Mod server files not found in /data (need srcds_run + garrysmod). Install a game version archive (Версии) so daemon can unpack server files." >&2
  ls -la "$DATA_DIR" 2>/dev/null || true
  exit 1
fi

CFG_DIR="$DATA_DIR/garrysmod/cfg"
CFG="$CFG_DIR/server.cfg"
mkdir -p "$CFG_DIR"

if [ ! -f "$CFG" ]; then
  echo "hostname \"$HOSTNAME\"" > "$CFG"
  echo "rcon_password \"$RCON_PASSWORD\"" >> "$CFG"
  echo "sv_lan 0" >> "$CFG"
  echo "sv_region 3" >> "$CFG"
  echo "sv_password \"\"" >> "$CFG"
fi

if [ "$#" -eq 0 ]; then
  set -- ./srcds_run -game garrysmod -port "$GMOD_PORT" +maxplayers "$MAXPLAYERS" +map gm_construct
fi

STARTUP_FILE="$DATA_DIR/.vtx/startup_params"
STARTUP_ARGV="$DATA_DIR/.vtx/startup_argv"
STARTUP_FILE_OLD="$DATA_DIR/.vtx_startup_params"
if [ ! -f "$STARTUP_FILE" ] && [ -f "$STARTUP_FILE_OLD" ]; then
  STARTUP_FILE="$STARTUP_FILE_OLD"
fi
if [ -f "$STARTUP_ARGV" ]; then
  # Аргументы по одному на строку.
  #
  # Однострочный файл подставлялся ниже как $STARTUP_LINE без кавычек. Оболочка
  # в этом случае делит строку на слова, но кавычки НЕ снимает, поэтому
  # -name "Мой мир" приходило игре тремя аргументами: -name, "Мой и мир".
  # Значение с пробелом — в первую очередь имя сервера — доехать не могло.
  # Разбор кавычек теперь делает панель, сюда аргументы приходят разделёнными.
  while IFS= read -r vtx_arg || [ -n "$vtx_arg" ]; do
    if [ -n "$vtx_arg" ]; then
      set -- "$@" "$vtx_arg"
    fi
  done < "$STARTUP_ARGV"
elif [ -f "$STARTUP_FILE" ]; then
  STARTUP_LINE="$(head -n 1 "$STARTUP_FILE" 2>/dev/null | tr -d '\r' | sed -e 's/^ *//; s/ *$//')"
  if [ "$STARTUP_LINE" != "" ]; then
    if echo "$STARTUP_LINE" | grep -Eq '^\./|^/|^srcds_run\b'; then
      set -- $STARTUP_LINE
    else
      set -- "$@" $STARTUP_LINE
    fi
  fi
fi

exec "$@"
