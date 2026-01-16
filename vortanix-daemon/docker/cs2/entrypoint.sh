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

# CS2 requires Steamworks client library at /data/.steam/sdk64/steamclient.so.
# Daemon tries to provision it, but we also enforce it here to avoid startup loops.
mkdir -p "$DATA_DIR/.steam/sdk64" 2>/dev/null || true
if [ ! -f "$DATA_DIR/.steam/sdk64/steamclient.so" ]; then
  p=$(find "$DATA_DIR" -maxdepth 25 -type f -name steamclient.so 2>/dev/null | head -n 1 || true)
  if [ "$p" != "" ]; then
    ln -sf "$p" "$DATA_DIR/.steam/sdk64/steamclient.so" 2>/dev/null || cp -fL "$p" "$DATA_DIR/.steam/sdk64/steamclient.so"
    chmod 644 "$DATA_DIR/.steam/sdk64/steamclient.so" 2>/dev/null || true
  fi
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
    set -- "$BIN" -dedicated -console -port "$CS2_PORT" +sv_setsteamaccount "$STEAM_ACCOUNT_TOKEN" +sv_maxplayers "$MAXPLAYERS" +map "$MAP"
  else
    set -- "$BIN" -dedicated -console -port "$CS2_PORT" +sv_maxplayers "$MAXPLAYERS" +map "$MAP"
  fi
fi

exec "$@"
