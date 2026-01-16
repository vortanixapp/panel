#!/bin/sh
set -eu

umask 0002

DATA_DIR="/data"

CFG="${DATA_DIR}/server.cfg"
LOG="${DATA_DIR}/server_log.txt"

is_uint() {
  case "${1:-}" in
    ''|*[!0-9]*) return 1 ;;
    *) return 0 ;;
  esac
}

set_cfg() {
  key="$1"
  value="$2"

  if [ -f "$CFG" ] && grep -q "^${key} " "$CFG"; then
    sed -i "s|^${key} .*|${key} ${value}|" "$CFG"
  else
    echo "${key} ${value}" >> "$CFG"
  fi
}

SAMP_PORT="${SAMP_PORT:-}"
if [ "$SAMP_PORT" = "" ]; then
  echo "SAMP_PORT is required" >&2
  exit 1
fi
if ! is_uint "$SAMP_PORT" || [ "$SAMP_PORT" -lt 1 ] || [ "$SAMP_PORT" -gt 65535 ]; then
  echo "Invalid SAMP_PORT: $SAMP_PORT" >&2
  exit 1
fi

mkdir -p "$DATA_DIR"

if [ ! -x "$DATA_DIR/samp03svr" ]; then
  echo "[samp] ERROR: samp03svr not found in /data. Upload or install a game version archive (Версии) so daemon can unpack server files." >&2
  ls -la "$DATA_DIR" 2>/dev/null || true
  exit 1
fi

cd "$DATA_DIR"

SAMP_RCON_PASSWORD="${SAMP_RCON_PASSWORD:-}"
if [ "$SAMP_RCON_PASSWORD" = "" ]; then
  SAMP_RCON_PASSWORD="vtx$(cat /proc/sys/kernel/random/uuid | tr -d '-' | cut -c1-20)"
fi

if [ ! -f "$CFG" ]; then
  echo "hostname Vortanix SA-MP Server" > "$CFG"
  echo "rcon_password $SAMP_RCON_PASSWORD" >> "$CFG"
  echo "maxplayers 50" >> "$CFG"
  echo "port $SAMP_PORT" >> "$CFG"
  echo "lanmode 0" >> "$CFG"
  echo "announce 0" >> "$CFG"
  echo "query 1" >> "$CFG"
  echo "rcon 0" >> "$CFG"
  echo "logtimeformat [%H:%M:%S]" >> "$CFG"
  echo "weburl vortanix.local" >> "$CFG"
  echo "gamemode0 grandlarc 1" >> "$CFG"
  echo "filterscripts" >> "$CFG"
  echo "plugins" >> "$CFG"
fi

if [ -n "${SAMP_HOSTNAME:-}" ]; then
  set_cfg hostname "$SAMP_HOSTNAME"
fi

current_rcon=""
if [ -f "$CFG" ]; then
  current_rcon="$(grep '^rcon_password ' "$CFG" | head -n 1 | cut -d' ' -f2- || true)"
fi
if [ "$current_rcon" = "" ] || [ "$current_rcon" = "changeme" ]; then
  set_cfg rcon_password "$SAMP_RCON_PASSWORD"
fi

if [ -n "${SAMP_MAXPLAYERS:-}" ]; then
  set_cfg maxplayers "$SAMP_MAXPLAYERS"
fi

set_cfg port "$SAMP_PORT"

if [ -n "${SAMP_WEBURL:-}" ]; then
  set_cfg weburl "$SAMP_WEBURL"
fi

if [ "$#" -eq 0 ]; then
  set -- ./samp03svr
fi

touch "$LOG"
tail -n 200 -F "$LOG" &

exec "$@"
