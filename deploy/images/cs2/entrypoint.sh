#!/bin/sh
set -eu

umask 0002

DATA_DIR="/data"

export HOME="$DATA_DIR"
export LD_LIBRARY_PATH="$DATA_DIR/.steam/sdk64:$DATA_DIR/game/bin/linuxsteamrt64:$DATA_DIR/game/csgo/bin/linuxsteamrt64:${LD_LIBRARY_PATH:-}"

mkdir -p "$DATA_DIR/.steam" || true
mkdir -p /root || true
if [ ! -e /root/.steam ]; then
  ln -s "$DATA_DIR/.steam" /root/.steam 2>/dev/null || true
fi

mkdir -p "$DATA_DIR/.steam/sdk64" 2>/dev/null || true
tgt="$DATA_DIR/.steam/sdk64/steamclient.so"
is_elf64() {
  f="$1"
  if [ ! -f "$f" ]; then
    return 1
  fi
  if command -v readelf >/dev/null 2>&1; then
    readelf -h "$f" 2>/dev/null | grep -q "Class:[[:space:]]*ELF64"
    return $?
  fi
  if command -v file >/dev/null 2>&1; then
    file -b "$f" 2>/dev/null | grep -q "ELF 64-bit"
    return $?
  fi
  return 0
}

if [ -f "$tgt" ]; then
  if ! is_elf64 "$tgt"; then
    rm -f "$tgt" 2>/dev/null || true
  fi
fi

if [ ! -f "$tgt" ]; then
  src="/opt/vortanix/steamcmd/linux64/steamclient.so"
  if [ -f "$src" ] && is_elf64 "$src"; then
    cp -fL "$src" "$tgt" 2>/dev/null || true
    chmod 644 "$tgt" 2>/dev/null || true
  else
    echo "[cs2] WARN: valid steamclient.so not found at $src" >&2
  fi
fi

mkdir -p "$DATA_DIR/.steam/steamcmd/linux64" 2>/dev/null || true
if [ -f "$tgt" ] && [ ! -f "$DATA_DIR/.steam/steamcmd/linux64/steamclient.so" ]; then
  cp -fL "$tgt" "$DATA_DIR/.steam/steamcmd/linux64/steamclient.so" 2>/dev/null || true
  chmod 644 "$DATA_DIR/.steam/steamcmd/linux64/steamclient.so" 2>/dev/null || true
fi

if [ -f "$tgt" ]; then
  chmod 644 "$tgt" 2>/dev/null || true
fi

is_uint() {
  case "${1:-}" in
    ''|*[!0-9]*) return 1 ;;
    *) return 0 ;;
  esac
}

CS2_PORT="${CS2_PORT:-}"
if [ "$CS2_PORT" = "" ]; then
  echo "CS2_PORT is required" >&2
  exit 1
fi
if ! is_uint "$CS2_PORT" || [ "$CS2_PORT" -lt 1 ] || [ "$CS2_PORT" -gt 65535 ]; then
  echo "Invalid CS2_PORT: $CS2_PORT" >&2
  exit 1
fi

MAXPLAYERS_RAW="${CS2_MAXPLAYERS:-20}"
if ! is_uint "$MAXPLAYERS_RAW" || [ "$MAXPLAYERS_RAW" -lt 1 ] || [ "$MAXPLAYERS_RAW" -gt 64 ]; then
  MAXPLAYERS_RAW="20"
fi
MAXPLAYERS="$MAXPLAYERS_RAW"

HOSTNAME="${CS2_HOSTNAME:-Vortanix Counter-Strike 2 Server}"
MAP="${CS2_MAP:-de_dust2}"

RCON_PASSWORD="${CS2_RCON_PASSWORD:-}"
if [ "$RCON_PASSWORD" = "" ]; then
  RCON_PASSWORD="vtx$(cat /proc/sys/kernel/random/uuid | tr -d '-' | cut -c1-20)"
fi

mkdir -p "$DATA_DIR"

BIN=""
if [ -x "$DATA_DIR/game/bin/linuxsteamrt64/cs2" ]; then
  BIN="$DATA_DIR/game/bin/linuxsteamrt64/cs2"
elif [ -x "$DATA_DIR/cs2" ]; then
  BIN="$DATA_DIR/cs2"
fi

if [ "$BIN" = "" ] || [ ! -d "$DATA_DIR/game" ]; then
  echo "[cs2] ERROR: CS2 server files not found in /data (need game/bin/linuxsteamrt64/cs2 + game dir). Install a game version archive (Версии) so daemon can unpack server files." >&2
  ls -la "$DATA_DIR" 2>/dev/null || true
  exit 1
fi

CFG_DIR="$DATA_DIR/game/csgo/cfg"
CFG="$CFG_DIR/server.cfg"
mkdir -p "$CFG_DIR"

if [ ! -f "$CFG" ]; then
  echo "hostname \"$HOSTNAME\"" > "$CFG"
  echo "rcon_password \"$RCON_PASSWORD\"" >> "$CFG"
fi

STEAM_ACCOUNT_TOKEN="${CS2_STEAM_ACCOUNT:-}"
if [ "$STEAM_ACCOUNT_TOKEN" = "" ] && [ -f "$CFG" ]; then
  STEAM_ACCOUNT_TOKEN=$(grep -E '^[[:space:]]*sv_setsteamaccount[[:space:]]+' "$CFG" 2>/dev/null | head -n1 | sed -E 's/^[[:space:]]*sv_setsteamaccount[[:space:]]+//; s/[[:space:]]+$//; s/^"(.*)"$/\1/; s/^\x27(.*)\x27$/\1/')
fi

if [ "$#" -eq 0 ]; then
  if [ "$STEAM_ACCOUNT_TOKEN" != "" ]; then
    set -- "$BIN" -dedicated -console -port "$CS2_PORT" +sv_setsteamaccount "$STEAM_ACCOUNT_TOKEN" +map "$MAP"
  else
    set -- "$BIN" -dedicated -console -port "$CS2_PORT" +map "$MAP"
  fi
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
    STARTUP_LINE_CLEAN="$(printf '%s' "$STARTUP_LINE" | sed -E 's/(^|[[:space:]])\+maxplayers[[:space:]]+[^[:space:]]+/\1/g; s/[[:space:]]+/ /g; s/^ //; s/ $//')"
    if [ "$STARTUP_LINE_CLEAN" != "" ] && echo "$STARTUP_LINE_CLEAN" | grep -Eq '^\./|^/|^cs2\b'; then
      set -- $STARTUP_LINE_CLEAN
    else
      set -- "$@" $STARTUP_LINE_CLEAN
    fi
  fi
fi

if ! printf ' %s ' "$*" | grep -Eq '[[:space:]]\+sv_visiblemaxplayers[[:space:]]+[0-9]+'; then
  set -- "$@" +sv_visiblemaxplayers "$MAXPLAYERS"
fi

exec "$@"
