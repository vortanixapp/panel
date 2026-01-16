#!/bin/sh
set -eu

umask 0002

DATA_DIR="/data"

CFG="${DATA_DIR}/cstrike/server.cfg"
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

CS16_PORT="${CS16_PORT:-}"
if [ "$CS16_PORT" = "" ]; then
  echo "CS16_PORT is required" >&2
  exit 1
fi
if ! is_uint "$CS16_PORT" || [ "$CS16_PORT" -lt 1 ] || [ "$CS16_PORT" -gt 65535 ]; then
  echo "Invalid CS16_PORT: $CS16_PORT" >&2
  exit 1
fi

MAXPLAYERS_RAW="${CS16_MAXPLAYERS:-32}"
if ! is_uint "$MAXPLAYERS_RAW" || [ "$MAXPLAYERS_RAW" -lt 1 ] || [ "$MAXPLAYERS_RAW" -gt 64 ]; then
  MAXPLAYERS_RAW="32"
fi
MAXPLAYERS="$MAXPLAYERS_RAW"

mkdir -p "$DATA_DIR"
mkdir -p "$DATA_DIR/cstrike"

if [ ! -x "$DATA_DIR/hlds_linux" ] || [ ! -d "$DATA_DIR/cstrike" ]; then
  echo "[cs16] ERROR: CS 1.6 server files not found in /data (need hlds_linux + cstrike). Install a game version archive (Версии) so daemon can unpack server files." >&2
  ls -la "$DATA_DIR" 2>/dev/null || true
  exit 1
fi

mkdir -p /root/.steam/sdk32
if [ ! -f /root/.steam/sdk32/steamclient.so ]; then
  for p in \
    /root/.steam/steamcmd/linux32/steamclient.so \
    /root/.steam/steam/steamcmd/linux32/steamclient.so \
    /root/.steam/steam/steamapps/common/Steamworks\ SDK\ Redist/linux32/steamclient.so \
    /root/.local/share/Steam/steamcmd/linux32/steamclient.so \
    /opt/cs16-dist/linux32/steamclient.so \
    /opt/cs16-dist/steamclient.so \
    /data/linux32/steamclient.so \
    /data/steamclient.so
  do
    if [ -f "$p" ]; then
      ln -sf "$p" /root/.steam/sdk32/steamclient.so
      break
    fi
  done
fi

cd "$DATA_DIR"

export LD_LIBRARY_PATH="$DATA_DIR:${LD_LIBRARY_PATH:-}"

CS16_RCON_PASSWORD="${CS16_RCON_PASSWORD:-}"
if [ "$CS16_RCON_PASSWORD" = "" ]; then
  CS16_RCON_PASSWORD="vtx$(cat /proc/sys/kernel/random/uuid | tr -d '-' | cut -c1-20)"
fi

if [ ! -f "$CFG" ]; then
  echo "hostname Vortanix CS 1.6 Server" > "$CFG"
  echo "rcon_password $CS16_RCON_PASSWORD" >> "$CFG"
  echo "maxplayers $MAXPLAYERS" >> "$CFG"
  echo "port $CS16_PORT" >> "$CFG"
  echo "sv_lan 0" >> "$CFG"
  echo "sv_region 3" >> "$CFG"
  echo "sv_contact vortanix.local" >> "$CFG"
  echo "sv_password \"\"" >> "$CFG"
  echo "sv_allowdownload 1" >> "$CFG"
  echo "sv_allowupload 1" >> "$CFG"
  echo "sv_downloadurl \"\"" >> "$CFG"
  echo "sv_enableoldqueries 1" >> "$CFG"
  echo "sv_cheats 0" >> "$CFG"
  echo "sv_consistency 1" >> "$CFG"
  echo "sv_unlag 1" >> "$CFG"
  echo "sv_maxunlag 0.5" >> "$CFG"
  echo "sv_timeout 65" >> "$CFG"
  echo "sv_voiceenable 1" >> "$CFG"
  echo "sv_voicecodec voice_speex" >> "$CFG"
  echo "sv_voicequality 3" >> "$CFG"
  echo "sv_maxrate 25000" >> "$CFG"
  echo "sv_minrate 5000" >> "$CFG"
  echo "sv_maxupdaterate 101" >> "$CFG"
  echo "sv_minupdaterate 20" >> "$CFG"
  echo "mp_autokick 1" >> "$CFG"
  echo "mp_autoteambalance 1" >> "$CFG"
  echo "mp_limitteams 2" >> "$CFG"
  echo "mp_buytime 1.5" >> "$CFG"
  echo "mp_c4timer 45" >> "$CFG"
  echo "mp_freezetime 5" >> "$CFG"
  echo "mp_friendlyfire 0" >> "$CFG"
  echo "mp_roundtime 3" >> "$CFG"
  echo "mp_timelimit 0" >> "$CFG"
  echo "mp_maxrounds 0" >> "$CFG"
  echo "mp_winlimit 0" >> "$CFG"
  echo "mp_startmoney 800" >> "$CFG"
  echo "mp_chattime 10" >> "$CFG"
  echo "map de_dust2" >> "$CFG"
fi

if [ -n "${CS16_HOSTNAME:-}" ]; then
  set_cfg hostname "$CS16_HOSTNAME"
fi

current_rcon=""
if [ -f "$CFG" ]; then
  current_rcon="$(grep '^rcon_password ' "$CFG" | head -n 1 | cut -d' ' -f2- || true)"
fi
if [ "$current_rcon" = "" ] || [ "$current_rcon" = "changeme" ]; then
  set_cfg rcon_password "$CS16_RCON_PASSWORD"
fi

if [ -n "${CS16_MAXPLAYERS:-}" ]; then
  set_cfg maxplayers "$CS16_MAXPLAYERS"
fi

set_cfg port "$CS16_PORT"

if [ -n "${CS16_WEBURL:-}" ]; then
  set_cfg sv_contact "$CS16_WEBURL"
fi

# Create maps directory if it doesn't exist
mkdir -p "$DATA_DIR/cstrike/maps"
mkdir -p "$DATA_DIR/cstrike/sound"
mkdir -p "$DATA_DIR/cstrike/sprites"
mkdir -p "$DATA_DIR/cstrike/models"

touch "$DATA_DIR/listip.cfg" || true
touch "$DATA_DIR/banned.cfg" || true
touch "$DATA_DIR/cstrike/listip.cfg" || true
touch "$DATA_DIR/cstrike/banned.cfg" || true

# Install default maps if they don't exist
if [ ! -f "$DATA_DIR/cstrike/maps/de_dust2.bsp" ]; then
  echo "Installing default CS 1.6 maps..."
  # This would typically download maps from Steam or other source
  # For now, we'll create placeholder files
  touch "$DATA_DIR/cstrike/maps/de_dust2.bsp"
  touch "$DATA_DIR/cstrike/maps/de_nuke.bsp"
  touch "$DATA_DIR/cstrike/maps/de_train.bsp"
  touch "$DATA_DIR/cstrike/maps/de_inferno.bsp"
  touch "$DATA_DIR/cstrike/maps/cs_office.bsp"
fi

if [ "$#" -eq 0 ] || [ "$#" -eq 1 ]; then
  first="${1:-}"
  base="${first##*/}"
  if [ "$#" -eq 0 ] || [ "$base" = "hlds_linux" ]; then
    set -- ./hlds_linux -game cstrike -port "$CS16_PORT" +maxplayers "$MAXPLAYERS" +map de_dust2
  fi
fi

touch "$LOG"
tail -n 200 -F "$LOG" &

exec "$@"
