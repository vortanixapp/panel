#!/bin/sh
set -eu

umask 0002

DATA_DIR="/data"
GAME_KEY="${VTX_GAME_KEY:-srcds}"
GAME_NAME="${VTX_GAME_NAME:-$GAME_KEY}"
MOD="${VTX_PARAM_MOD:-cstrike}"
MAP="${VTX_PARAM_MAP:-de_dust2}"

is_uint() {
  case "${1:-}" in
    ''|*[!0-9]*) return 1 ;;
    *) return 0 ;;
  esac
}

PORT_ENV="${VTX_PORT_ENV:-GAME_PORT}"
PORT="$(eval "printf '%s' \"\${$PORT_ENV:-}\"")"
if [ "$PORT" = "" ]; then
  echo "[$GAME_KEY] $PORT_ENV is required" >&2
  exit 1
fi
if ! is_uint "$PORT" || [ "$PORT" -lt 1 ] || [ "$PORT" -gt 65535 ]; then
  echo "[$GAME_KEY] Invalid $PORT_ENV: $PORT" >&2
  exit 1
fi

MAXPLAYERS="${VTX_PARAM_MAXPLAYERS:-32}"
if ! is_uint "$MAXPLAYERS" || [ "$MAXPLAYERS" -lt 1 ] || [ "$MAXPLAYERS" -gt 64 ]; then
  MAXPLAYERS="32"
fi

HOSTNAME_VALUE="${SERVER_HOSTNAME:-Vortanix $GAME_NAME Server}"
RCON_PASSWORD="${RCON_PASSWORD:-}"
if [ "$RCON_PASSWORD" = "" ]; then
  RCON_PASSWORD="vtx$(cat /proc/sys/kernel/random/uuid | tr -d '-' | cut -c1-20)"
fi

export HOME="$DATA_DIR"
mkdir -p "$DATA_DIR" 2>/dev/null || true
cd "$DATA_DIR"

BINARIES="${VTX_SERVER_BINARIES:-}"
BINARIES="$BINARIES:./srcds_run:./hlds_linux:./start-server.sh:./start.sh"

BIN=""
OLD_IFS="$IFS"
IFS=':'
for c in $BINARIES; do
  [ "$c" = "" ] && continue
  if [ -x "$c" ]; then
    BIN="$c"
    break
  fi
done
IFS="$OLD_IFS"

if [ "$BIN" = "" ] || [ ! -d "$DATA_DIR/$MOD" ]; then
  echo "[$GAME_KEY] ERROR: server files not found in /data for $GAME_NAME" >&2
  echo "[$GAME_KEY] нужен исполняемый сервер и каталог мода $MOD" >&2
  ls -la "$DATA_DIR" 2>/dev/null || true
  exit 1
fi

CFG_DIR="$DATA_DIR/$MOD/cfg"
CFG="$CFG_DIR/server.cfg"
mkdir -p "$CFG_DIR"
if [ ! -f "$CFG" ]; then
  {
    echo "hostname \"$HOSTNAME_VALUE\""
    echo "rcon_password \"$RCON_PASSWORD\""
    echo "sv_lan 0"
    echo "sv_region 3"
    echo "sv_password \"\""
  } > "$CFG"
fi

if [ "$#" -eq 0 ]; then
  set -- "$BIN" -game "$MOD" -console -port "$PORT" +maxplayers "$MAXPLAYERS" +map "$MAP"
fi

STARTUP_FILE="$DATA_DIR/.vtx/startup_params"
STARTUP_ARGV="$DATA_DIR/.vtx/startup_argv"
STARTUP_FILE_OLD="$DATA_DIR/.vtx_startup_params"
if [ ! -f "$STARTUP_FILE" ] && [ -f "$STARTUP_FILE_OLD" ]; then
  STARTUP_FILE="$STARTUP_FILE_OLD"
fi
if [ -f "$STARTUP_ARGV" ]; then
  while IFS= read -r vtx_arg || [ -n "$vtx_arg" ]; do
    if [ -n "$vtx_arg" ]; then
      set -- "$@" "$vtx_arg"
    fi
  done < "$STARTUP_ARGV"
elif [ -f "$STARTUP_FILE" ]; then
  STARTUP_LINE="$(head -n 1 "$STARTUP_FILE" 2>/dev/null | tr -d '\r' | sed -e 's/^ *//; s/ *$//')"
  if [ "$STARTUP_LINE" != "" ]; then
    if echo "$STARTUP_LINE" | grep -Eq '^\./|^/|^srcds_run\b|^hlds_linux\b'; then
      set -- $STARTUP_LINE
    else
      set -- "$@" $STARTUP_LINE
    fi
  fi
fi

exec "$@"
