#!/bin/sh
set -eu

DATA_DIR="/data"

export HOME="$DATA_DIR"
export XDG_CONFIG_HOME="$DATA_DIR/.config"
export LD_LIBRARY_PATH="$DATA_DIR/.steam/sdk64:$DATA_DIR/linux64:$DATA_DIR:${LD_LIBRARY_PATH:-}"

UNTURNED_PORT="${UNTURNED_PORT:-27015}"
UNTURNED_SERVER_NAME="${UNTURNED_SERVER_NAME:-server}"
SERVER_TYPE="${SERVER_TYPE:-}"
mkdir -p "$DATA_DIR"
mkdir -p "$DATA_DIR/.config" "$DATA_DIR/.steam/sdk64" "$DATA_DIR/.steam/steamcmd/linux64" 2>/dev/null || true

if [ -d /home/steam ] && [ ! -e /home/steam/.steam ]; then
  ln -s "$DATA_DIR/.steam" /home/steam/.steam 2>/dev/null || true
fi
if [ "$(id -u)" = "0" ]; then
  mkdir -p /root 2>/dev/null || true
  if [ ! -e /root/.steam ]; then
    ln -s "$DATA_DIR/.steam" /root/.steam 2>/dev/null || true
  fi
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
else
  echo "[unturned] ERROR: steamclient.so not found." >&2
  echo "[unturned] Looked in /data/.steam/steamcmd/linux64, /data/linux64, /data and SteamCMD system paths." >&2
  echo "[unturned] Install files via SteamCMD or include steamclient.so in version archive." >&2
  exit 1
fi

BIN=""
BIN_IS_HEADLESS="0"
if [ -x "$DATA_DIR/Unturned_Headless.x86_64" ]; then
  BIN="$DATA_DIR/Unturned_Headless.x86_64"
  BIN_IS_HEADLESS="1"
elif [ -x "$DATA_DIR/Unturned.x86_64" ]; then
  BIN="$DATA_DIR/Unturned.x86_64"
elif [ -x "$DATA_DIR/Unturned" ]; then
  BIN="$DATA_DIR/Unturned"
fi

echo "[unturned] Selected binary: ${BIN:-<none>}" >&2

UNTURNED_ALLOW_NON_HEADLESS="${UNTURNED_ALLOW_NON_HEADLESS:-0}"
if [ "$BIN" != "" ] && [ "$BIN_IS_HEADLESS" != "1" ] && [ "$UNTURNED_ALLOW_NON_HEADLESS" != "1" ]; then
  echo "[unturned] ERROR: Non-headless Unturned binary detected (${BIN})." >&2
  echo "[unturned] This container image is intended to run headless servers and may crash on shader/libGL initialization." >&2
  echo "[unturned] Fix: install a version archive that contains Unturned_Headless.x86_64 into /data." >&2
  echo "[unturned] Override (not recommended): set UNTURNED_ALLOW_NON_HEADLESS=1" >&2
  exit 1
fi

if [ "$BIN" = "" ] || [ ! -d "$DATA_DIR/Servers" ]; then
  echo "[unturned] ERROR: Unturned server files not found in /data (need Unturned binary + Servers directory). Install a game version archive (Версии) so daemon can unpack server files." >&2
  ls -la "$DATA_DIR" 2>/dev/null || true
  exit 1
fi

cd "$DATA_DIR"

SERVER_COMMANDS_DIR="$DATA_DIR/Servers/$UNTURNED_SERVER_NAME/Server"
SERVER_COMMANDS_FILE="$SERVER_COMMANDS_DIR/Commands.dat"
mkdir -p "$SERVER_COMMANDS_DIR"
touch "$SERVER_COMMANDS_FILE"
chown -R steam:steam "$DATA_DIR/Servers/$UNTURNED_SERVER_NAME" 2>/dev/null || true

if grep -qi '^Name ' "$SERVER_COMMANDS_FILE"; then
  sed -i "s/^Name .*/Name $UNTURNED_SERVER_NAME/I" "$SERVER_COMMANDS_FILE" || true
else
  echo "Name $UNTURNED_SERVER_NAME" >> "$SERVER_COMMANDS_FILE"
fi

if grep -qi '^Port ' "$SERVER_COMMANDS_FILE"; then
  sed -i "s/^Port .*/Port $UNTURNED_PORT/I" "$SERVER_COMMANDS_FILE" || true
else
  echo "Port $UNTURNED_PORT" >> "$SERVER_COMMANDS_FILE"
fi

if ! grep -qi '^InternetServer\b' "$SERVER_COMMANDS_FILE"; then
  echo "InternetServer" >> "$SERVER_COMMANDS_FILE"
fi

if [ "${UNTURNED_LOGIN_TOKEN:-}" != "" ]; then
  if grep -qi '^LoginToken ' "$SERVER_COMMANDS_FILE"; then
    sed -i "s/^LoginToken .*/LoginToken ${UNTURNED_LOGIN_TOKEN}/I" "$SERVER_COMMANDS_FILE" || true
  else
    echo "LoginToken ${UNTURNED_LOGIN_TOKEN}" >> "$SERVER_COMMANDS_FILE"
  fi
fi

EXTRA=""
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
    if echo "$STARTUP_LINE" | grep -Eq '^\./|^/|^Unturned\b|^Unturned_Headless\b'; then
      if [ "$(id -u)" = "0" ]; then
        exec su -s /bin/bash steam -c "$STARTUP_LINE"
      fi
      exec sh -c "$STARTUP_LINE"
    else
      EXTRA="$STARTUP_LINE"
    fi
  fi
fi

FINAL_CMD="\"$BIN\" -nographics -batchmode -Port $UNTURNED_PORT -Name \"$UNTURNED_SERVER_NAME\" $SERVER_TYPE $EXTRA"
if [ "$(id -u)" = "0" ]; then
  exec su -s /bin/bash steam -c "$FINAL_CMD"
fi
exec sh -c "exec $FINAL_CMD"
