#!/bin/sh
set -eu

DATA_DIR="/data"
STARTUP_FILE="$DATA_DIR/.vtx/startup_params"
STARTUP_ARGV="$DATA_DIR/.vtx/startup_argv"
STARTUP_FILE_OLD="$DATA_DIR/.vtx_startup_params"
SERVER_BIN="$DATA_DIR/bedrock_server"
PROPERTIES_FILE="$DATA_DIR/server.properties"
BEDROCK_VERSION="${BEDROCK_VERSION:-latest}"
BEDROCK_DOWNLOAD_URL="${BEDROCK_DOWNLOAD_URL:-}"

mkdir -p "$DATA_DIR"
cd "$DATA_DIR"

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
    if [ "${EXTRA_ARGS:-}" != "" ]; then
      export EXTRA_ARGS="${EXTRA_ARGS} ${STARTUP_LINE}"
    else
      export EXTRA_ARGS="${STARTUP_LINE}"
    fi
  fi
fi

EXTRA_ARGS_FINAL="$(printf '%s' "${EXTRA_ARGS:-}" | sed -e 's/^ *//; s/ *$//')"

resolve_download_url() {
  if [ "$BEDROCK_DOWNLOAD_URL" != "" ]; then
    printf '%s' "$BEDROCK_DOWNLOAD_URL"
    return
  fi

  if [ "$BEDROCK_VERSION" = "latest" ]; then
    printf '%s' "https://www.minecraft.net/bedrockdedicatedserver/bin-linux/bedrock-server.zip"
  else
    printf '%s' "https://www.minecraft.net/bedrockdedicatedserver/bin-linux/bedrock-server-${BEDROCK_VERSION}.zip"
  fi
}

download_server() {
  URL="$(resolve_download_url)"
  TMP_ZIP="$DATA_DIR/.bedrock-server.zip"

  echo "[vtx-entrypoint] Downloading Bedrock server: $URL"
  curl -fsSL "$URL" -o "$TMP_ZIP"

  rm -rf "$DATA_DIR/.extract"
  mkdir -p "$DATA_DIR/.extract"
  unzip -q "$TMP_ZIP" -d "$DATA_DIR/.extract"
  rm -f "$TMP_ZIP"

  cp -f "$DATA_DIR/.extract/bedrock_server" "$DATA_DIR/bedrock_server"
  chmod +x "$DATA_DIR/bedrock_server"

  for file in \
    libCrypto.so \
    libssl.so \
    libcurl.so.4 \
    libz.so.1 \
    libstdc++.so.6 \
    permissions.json \
    allowlist.json \
    whitelist.json \
    valid_known_packs.json \
    server.properties \
    bedrock_server_how_to.html
  do
    if [ -f "$DATA_DIR/.extract/$file" ] && [ ! -f "$DATA_DIR/$file" ]; then
      cp -f "$DATA_DIR/.extract/$file" "$DATA_DIR/$file"
    fi
  done

  rm -rf "$DATA_DIR/.extract"
}

if [ ! -x "$SERVER_BIN" ]; then
  download_server
fi

if [ ! -f "$PROPERTIES_FILE" ]; then
  cat > "$PROPERTIES_FILE" <<'EOF'
server-name=Dedicated Server
gamemode=survival
difficulty=easy
allow-cheats=false
max-players=20
online-mode=true
white-list=false
server-port=19132
server-portv6=19133
view-distance=32
tick-distance=4
player-idle-timeout=30
max-threads=8
level-name=Bedrock level
level-seed=
default-player-permission-level=member
texturepack-required=false
content-log-file-enabled=false
compression-threshold=1
server-authoritative-movement=server-auth
player-position-acceptance-threshold=0.5
player-movement-score-threshold=20
player-movement-distance-threshold=0.3
player-movement-duration-threshold-in-ms=500
correct-player-movement=false
server-authoritative-block-breaking=false
EOF
fi

if [ "${SERVER_PORT:-}" != "" ]; then
  case "$SERVER_PORT" in
    ''|*[!0-9]*)
      echo "[vtx-entrypoint] Invalid SERVER_PORT: $SERVER_PORT" >&2
      exit 1
      ;;
  esac

  SERVER_PORT_V6=$((SERVER_PORT + 1))

  if grep -q '^server-port=' "$PROPERTIES_FILE" 2>/dev/null; then
    sed -i "s/^server-port=.*/server-port=${SERVER_PORT}/" "$PROPERTIES_FILE" || true
  else
    printf 'server-port=%s\n' "$SERVER_PORT" >> "$PROPERTIES_FILE"
  fi

  if grep -q '^server-portv6=' "$PROPERTIES_FILE" 2>/dev/null; then
    sed -i "s/^server-portv6=.*/server-portv6=${SERVER_PORT_V6}/" "$PROPERTIES_FILE" || true
  else
    printf 'server-portv6=%s\n' "$SERVER_PORT_V6" >> "$PROPERTIES_FILE"
  fi

  echo "[vtx-entrypoint] server-port=$SERVER_PORT server-portv6=$SERVER_PORT_V6"
fi

if [ "$EXTRA_ARGS_FINAL" != "" ]; then
  echo "[vtx-entrypoint] Extra args: $EXTRA_ARGS_FINAL"
fi

echo "[vtx-entrypoint] Starting Bedrock server"

exec sh -c "exec \"$SERVER_BIN\" $EXTRA_ARGS_FINAL"
