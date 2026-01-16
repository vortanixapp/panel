#!/bin/sh
set -eu

DATA_DIR="/data"

# Default values
UNTURNED_PORT="${UNTURNED_PORT:-27015}"
UNTURNED_SERVER_NAME="${UNTURNED_SERVER_NAME:-server}"
SERVER_TYPE="${SERVER_TYPE:-}"
mkdir -p "$DATA_DIR"

BIN=""
if [ -x "$DATA_DIR/Unturned_Headless.x86_64" ]; then
  BIN="$DATA_DIR/Unturned_Headless.x86_64"
elif [ -x "$DATA_DIR/Unturned.x86_64" ]; then
  BIN="$DATA_DIR/Unturned.x86_64"
elif [ -x "$DATA_DIR/Unturned" ]; then
  BIN="$DATA_DIR/Unturned"
fi

if [ "$BIN" = "" ] || [ ! -d "$DATA_DIR/Servers" ]; then
  echo "[unturned] ERROR: Unturned server files not found in /data (need Unturned binary + Servers directory). Install a game version archive (Версии) so daemon can unpack server files." >&2
  ls -la "$DATA_DIR" 2>/dev/null || true
  exit 1
fi

# Run Unturned server headless
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

exec su -s /bin/bash steam -c "\"$BIN\" -nographics -batchmode -Port $UNTURNED_PORT -Name \"$UNTURNED_SERVER_NAME\" $SERVER_TYPE"
