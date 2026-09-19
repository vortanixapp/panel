#!/bin/sh
set -eu

DATA_DIR="/data"
STARTUP_FILE="$DATA_DIR/.vtx/startup_params"
STARTUP_ARGV="$DATA_DIR/.vtx/startup_argv"
STARTUP_FILE_OLD="$DATA_DIR/.vtx_startup_params"
VERSION_FILE="$DATA_DIR/.vtx/bedrock_version"
SERVER_BIN="$DATA_DIR/bedrock_server"
PROPERTIES_FILE="$DATA_DIR/server.properties"
BEDROCK_VERSION="${BEDROCK_VERSION:-latest}"
BEDROCK_DOWNLOAD_URL="${BEDROCK_DOWNLOAD_URL:-}"
LINKS_API="https://net-secondary.web.minecraft-services.net/api/v1.0/download/links"
USER_AGENT="Mozilla/5.0 (compatible; Vortanix)"
KEEP_FILES="server.properties permissions.json allowlist.json whitelist.json"

mkdir -p "$DATA_DIR/.vtx"
cd "$DATA_DIR"

log() {
  echo "[vtx-entrypoint] $*"
}

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
    if [ "${EXTRA_ARGS:-}" != "" ]; then
      export EXTRA_ARGS="${EXTRA_ARGS} ${STARTUP_LINE}"
    else
      export EXTRA_ARGS="${STARTUP_LINE}"
    fi
  fi
fi

EXTRA_ARGS_FINAL="$(printf '%s' "${EXTRA_ARGS:-}" | sed -e 's/^ *//; s/ *$//')"

fetch() {
  curl -fsSL --retry 3 --retry-delay 5 --connect-timeout 20 -A "$USER_AGENT" "$@"
}

channel_link_type() {
  case "$BEDROCK_VERSION" in
    preview|PREVIEW) echo "serverBedrockPreviewLinux" ;;
    *) echo "serverBedrockLinux" ;;
  esac
}

resolve_download_url() {
  if [ "$BEDROCK_DOWNLOAD_URL" != "" ]; then
    printf '%s' "$BEDROCK_DOWNLOAD_URL"
    return 0
  fi
  case "$BEDROCK_VERSION" in
    latest|LATEST|preview|PREVIEW)
      link_type="$(channel_link_type)"
      links="$(fetch "$LINKS_API" 2>/dev/null)" || return 1
      printf '%s' "$links" \
        | tr '{}' '\n\n' \
        | grep "\"downloadType\":\"$link_type\"" \
        | sed -n 's/.*"downloadUrl":"\([^"]*\)".*/\1/p' \
        | head -n 1
      ;;
    *)
      printf '%s' "https://www.minecraft.net/bedrockdedicatedserver/bin-linux/bedrock-server-${BEDROCK_VERSION}.zip"
      ;;
  esac
}

version_of_url() {
  basename "$1" | sed -n 's/^bedrock-server-\(.*\)\.zip$/\1/p'
}

install_server() {
  url="$1"
  tmp_zip="$DATA_DIR/.vtx/bedrock-server.zip"
  extract_dir="$DATA_DIR/.vtx/extract"

  log "Downloading Bedrock server: $url"
  rm -f "$tmp_zip"
  if ! fetch "$url" -o "$tmp_zip"; then
    rm -f "$tmp_zip"
    log "Download failed: $url" >&2
    return 1
  fi

  rm -rf "$extract_dir"
  mkdir -p "$extract_dir"
  if ! unzip -q -o "$tmp_zip" -d "$extract_dir" || [ ! -f "$extract_dir/bedrock_server" ]; then
    rm -rf "$extract_dir" "$tmp_zip"
    log "Archive has no bedrock_server: $url" >&2
    return 1
  fi
  rm -f "$tmp_zip"

  for file in $KEEP_FILES; do
    if [ -f "$DATA_DIR/$file" ]; then
      rm -f "$extract_dir/$file"
    fi
  done

  cp -a "$extract_dir/." "$DATA_DIR/"
  rm -rf "$extract_dir"
  chmod +x "$SERVER_BIN"
  return 0
}

INSTALLED_VERSION=""
if [ -f "$VERSION_FILE" ]; then
  INSTALLED_VERSION="$(head -n 1 "$VERSION_FILE" | tr -d '\r')"
fi

if [ -f "$SERVER_BIN" ] && [ ! -x "$SERVER_BIN" ]; then
  chmod +x "$SERVER_BIN" 2>/dev/null || true
fi

MANAGED=0
if [ ! -f "$SERVER_BIN" ] || [ "$INSTALLED_VERSION" != "" ]; then
  MANAGED=1
fi

if [ "$MANAGED" = "1" ]; then
  URL="$(resolve_download_url || true)"
  TARGET_VERSION=""
  if [ "$URL" != "" ]; then
    TARGET_VERSION="$(version_of_url "$URL")"
    if [ "$TARGET_VERSION" = "" ]; then
      TARGET_VERSION="$URL"
    fi
  fi

  if [ "$URL" = "" ]; then
    if [ -f "$SERVER_BIN" ]; then
      log "Could not check for Bedrock updates, starting installed version ${INSTALLED_VERSION:-unknown}"
    else
      log "Could not get the Bedrock download link from $LINKS_API" >&2
      log "Set BEDROCK_VERSION to an exact version or BEDROCK_DOWNLOAD_URL to a zip link" >&2
      exit 1
    fi
  elif [ "$TARGET_VERSION" != "$INSTALLED_VERSION" ] || [ ! -f "$SERVER_BIN" ]; then
    if [ "$INSTALLED_VERSION" != "" ]; then
      log "Updating Bedrock server $INSTALLED_VERSION -> $TARGET_VERSION"
    fi
    if install_server "$URL"; then
      printf '%s\n' "$TARGET_VERSION" > "$VERSION_FILE"
      INSTALLED_VERSION="$TARGET_VERSION"
    elif [ -f "$SERVER_BIN" ]; then
      log "Update failed, starting installed version ${INSTALLED_VERSION:-unknown}"
    else
      exit 1
    fi
  fi
fi

if [ ! -f "$PROPERTIES_FILE" ]; then
  cat > "$PROPERTIES_FILE" <<'EOF'
server-name=Dedicated Server
gamemode=survival
difficulty=easy
allow-cheats=false
max-players=20
online-mode=true
allow-list=false
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

set_property() {
  if grep -q "^$1=" "$PROPERTIES_FILE" 2>/dev/null; then
    sed -i "s/^$1=.*/$1=$2/" "$PROPERTIES_FILE" || true
  else
    printf '%s=%s\n' "$1" "$2" >> "$PROPERTIES_FILE"
  fi
}

if [ ! -d "$DATA_DIR/worlds" ]; then
  set_property allow-list false
  if grep -q '^white-list=' "$PROPERTIES_FILE" 2>/dev/null; then
    set_property white-list false
  fi
fi

if [ "${SERVER_PORT:-}" != "" ]; then
  case "$SERVER_PORT" in
    ''|*[!0-9]*)
      log "Invalid SERVER_PORT: $SERVER_PORT" >&2
      exit 1
      ;;
  esac

  SERVER_PORT_V6=$((SERVER_PORT + 1))
  set_property server-port "$SERVER_PORT"
  set_property server-portv6 "$SERVER_PORT_V6"
  log "server-port=$SERVER_PORT server-portv6=$SERVER_PORT_V6"
fi

if [ ! -x "$SERVER_BIN" ]; then
  log "bedrock_server not found in $DATA_DIR" >&2
  exit 1
fi

if [ "$EXTRA_ARGS_FINAL" != "" ]; then
  log "Extra args: $EXTRA_ARGS_FINAL"
fi

log "Starting Bedrock server ${INSTALLED_VERSION:-}"

export LD_LIBRARY_PATH="$DATA_DIR${LD_LIBRARY_PATH:+:$LD_LIBRARY_PATH}"
exec sh -c "exec \"$SERVER_BIN\" $EXTRA_ARGS_FINAL"
