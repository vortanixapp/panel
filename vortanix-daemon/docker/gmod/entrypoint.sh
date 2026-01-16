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

exec "$@"
