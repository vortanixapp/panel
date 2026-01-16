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

TF2_PORT="${TF2_PORT:-}"
if [ "$TF2_PORT" = "" ]; then
  echo "TF2_PORT is required" >&2
  exit 1
fi
if ! is_uint "$TF2_PORT" || [ "$TF2_PORT" -lt 1 ] || [ "$TF2_PORT" -gt 65535 ]; then
  echo "Invalid TF2_PORT: $TF2_PORT" >&2
  exit 1
fi

MAXPLAYERS_RAW="${TF2_MAXPLAYERS:-24}"
if ! is_uint "$MAXPLAYERS_RAW" || [ "$MAXPLAYERS_RAW" -lt 1 ] || [ "$MAXPLAYERS_RAW" -gt 32 ]; then
  MAXPLAYERS_RAW="24"
fi
MAXPLAYERS="$MAXPLAYERS_RAW"

HOSTNAME="${TF2_HOSTNAME:-Vortanix Team Fortress 2 Server}"
MAP="${TF2_MAP:-ctf_2fort}"

RCON_PASSWORD="${TF2_RCON_PASSWORD:-}"
if [ "$RCON_PASSWORD" = "" ]; then
  RCON_PASSWORD="vtx$(cat /proc/sys/kernel/random/uuid | tr -d '-' | cut -c1-20)"
fi

mkdir -p "$DATA_DIR"

if [ ! -x "$DATA_DIR/srcds_run" ] || [ ! -d "$DATA_DIR/tf" ]; then
  echo "[tf2] ERROR: TF2 server files not found in /data (need srcds_run + tf). Install a game version archive (Версии) so daemon can unpack server files." >&2
  ls -la "$DATA_DIR" 2>/dev/null || true
  exit 1
fi

CFG_DIR="$DATA_DIR/tf/cfg"
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
  set -- ./srcds_run -game tf -console -port "$TF2_PORT" +maxplayers "$MAXPLAYERS" +map "$MAP"
fi

exec "$@"
