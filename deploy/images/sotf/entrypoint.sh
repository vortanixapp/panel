#!/bin/sh
set -eu

umask 0002

DATA_DIR="/data"
GAME_KEY="sotf"
GAME_NAME="Sons of the Forest"

is_uint() {
  case "${1:-}" in
    ''|*[!0-9]*) return 1 ;;
    *) return 0 ;;
  esac
}

GAME_PORT="${GAME_PORT:-8766}"
if [ "$GAME_PORT" = "" ]; then
  echo "[$GAME_KEY] GAME_PORT is required" >&2
  exit 1
fi
if ! is_uint "$GAME_PORT" || [ "$GAME_PORT" -lt 1 ] || [ "$GAME_PORT" -gt 65535 ]; then
  echo "[$GAME_KEY] Invalid GAME_PORT: $GAME_PORT" >&2
  exit 1
fi

export HOME="$DATA_DIR"
export LD_LIBRARY_PATH="$DATA_DIR/.steam/sdk64:$DATA_DIR/linux64:$DATA_DIR:${LD_LIBRARY_PATH:-}"

mkdir -p "$DATA_DIR" "$DATA_DIR/.steam" "$DATA_DIR/.steam/sdk64" "$DATA_DIR/.steam/steamcmd/linux64" 2>/dev/null || true
if [ -d /root ] && [ ! -e /root/.steam ]; then
  ln -s "$DATA_DIR/.steam" /root/.steam 2>/dev/null || true
fi

tgt_steamclient="$DATA_DIR/.steam/sdk64/steamclient.so"
if [ ! -f "$tgt_steamclient" ]; then
  for src_steamclient in \
    "$DATA_DIR/.steam/steamcmd/linux64/steamclient.so" \
    "$DATA_DIR/linux64/steamclient.so" \
    "$DATA_DIR/steamclient.so" \
    "/opt/vortanix/steamcmd/linux64/steamclient.so" \
    "/usr/lib/steamcmd/linux64/steamclient.so" \
    "/usr/share/steamcmd/linux64/steamclient.so"; do
    if [ -f "$src_steamclient" ]; then
      cp -fL "$src_steamclient" "$tgt_steamclient" 2>/dev/null || true
      break
    fi
  done
fi
if [ -f "$tgt_steamclient" ]; then
  cp -fL "$tgt_steamclient" "$DATA_DIR/.steam/steamcmd/linux64/steamclient.so" 2>/dev/null || true
  chmod 644 "$tgt_steamclient" "$DATA_DIR/.steam/steamcmd/linux64/steamclient.so" 2>/dev/null || true
fi

cd "$DATA_DIR"

BIN=""
for c in \
  "./SonsOfTheForestDS" \
  "./SonsOfTheForest/Binaries/Linux/SonsOfTheForestDS.x86_64" \
  "./start-server.sh" \
  "./start.sh" \
  "./server.sh" \
  "./run.sh"
do
  if [ -x "$c" ]; then
    BIN="$c"
    break
  fi
done

if [ "$BIN" = "" ]; then
  echo "[$GAME_KEY] ERROR: server binary not found in /data for $GAME_NAME" >&2
  ls -la "$DATA_DIR" 2>/dev/null || true
  exit 1
fi

if [ "$#" -eq 0 ]; then
  set -- "$BIN"
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
    if echo "$STARTUP_LINE" | grep -Eq '^\./|^/'; then
      set -- $STARTUP_LINE
    else
      set -- "$@" $STARTUP_LINE
    fi
  fi
fi

exec "$@"
