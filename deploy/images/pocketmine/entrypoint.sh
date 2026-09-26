#!/bin/sh
set -eu

DATA_DIR="/data"
STARTUP_FILE="$DATA_DIR/.vtx/startup_params"
STARTUP_ARGV="$DATA_DIR/.vtx/startup_argv"
STARTUP_FILE_OLD="$DATA_DIR/.vtx_startup_params"
CORE_MARKER="$DATA_DIR/.vtx/pocketmine_version"
PHP_MARKER="$DATA_DIR/.vtx/pocketmine_php"
PHP_OVERRIDE_FILE="$DATA_DIR/.vtx/php"
PHP_URL_FILE="$DATA_DIR/.vtx/php_url"
CORE_FILE="$DATA_DIR/PocketMine-MP.phar"
MANAGED_PHP="$DATA_DIR/bin/php7/bin/php"
DEFAULT_MANAGED_PHP="$MANAGED_PHP"
PROPERTIES_FILE="$DATA_DIR/server.properties"
BUILD_INFO_URL="${POCKETMINE_BUILD_INFO_URL:-https://github.com/pmmp/PocketMine-MP/releases/latest/download/build_info.json}"
PHP_DOWNLOAD_URL="${PHP_DOWNLOAD_URL:-}"
USER_AGENT="Mozilla/5.0 (compatible; Vortanix)"

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
    set -- "$@" $STARTUP_LINE
  fi
fi

fetch() {
  curl -fsSL --retry 3 --retry-delay 5 --connect-timeout 20 -A "$USER_AGENT" "$@"
}

json_field() {
  printf '%s' "$1" | tr -d '\n\r' | sed -n "s/.*\"$2\"[[:space:]]*:[[:space:]]*\"\([^\"]*\)\".*/\1/p"
}

file_sum() {
  sha256sum "$1" 2>/dev/null | cut -d' ' -f1
}

marker_line() {
  if [ -f "$1" ]; then
    sed -n "${2}p" "$1" | tr -d '\r'
  fi
}

write_marker() {
  printf '%s\n%s\n' "$2" "$(file_sum "$3")" > "$1"
}

find_php() {
  for candidate in "$MANAGED_PHP" "$DATA_DIR"/bin/*/bin/php "$DATA_DIR/bin/php"; do
    if [ -f "$candidate" ]; then
      printf '%s' "$candidate"
      return 0
    fi
  done
}

find_core() {
  if [ "${POCKETMINE_FILE:-}" != "" ]; then
    printf '%s' "$POCKETMINE_FILE"
    return 0
  fi
  for candidate in "$CORE_FILE" "$DATA_DIR/src/PocketMine.php" "$DATA_DIR/src/pocketmine/PocketMine.php" "$DATA_DIR"/*.phar; do
    if [ -f "$candidate" ]; then
      printf '%s' "$candidate"
      return 0
    fi
  done
}

php_custom_url() {
  if [ "$PHP_DOWNLOAD_URL" != "" ]; then
    printf '%s' "$PHP_DOWNLOAD_URL"
    return 0
  fi
  if [ -f "$PHP_URL_FILE" ]; then
    head -n 1 "$PHP_URL_FILE" | tr -d '\r ' | tr -d '\n'
  fi
}

php_url() {
  custom="$(php_custom_url)"
  if [ "$custom" != "" ]; then
    printf '%s' "$custom"
    return 0
  fi
  printf 'https://github.com/pmmp/PHP-Binaries/releases/download/pm5-php-%s-latest/PHP-%s-Linux-x86_64-PM5.tar.gz' "$1" "$1"
}

write_php_marker() {
  printf '%s\n%s\n%s\n%s\n' "$1" "$(file_sum "$2")" "$2" "$3" > "$PHP_MARKER"
}

php_source_changed() {
  if [ "$INSTALLED_PHP_URL" = "" ]; then
    return 1
  fi
  [ "$INSTALLED_PHP_URL" != "$(php_url "$1")" ]
}

install_php() {
  url="$(php_url "$1")"
  tmp_archive="$DATA_DIR/.vtx/php.tar.gz"
  extract_dir="$DATA_DIR/.vtx/php-extract"

  if [ "$(php_custom_url)" = "" ] && [ "$(uname -m)" != "x86_64" ]; then
    log "pmmp publishes PHP builds only for x86_64, put your PHP build into bin/ of the server folder" >&2
    return 1
  fi
  log "Downloading PHP $1: $url"
  rm -rf "$tmp_archive" "$extract_dir"
  if ! fetch "$url" -o "$tmp_archive"; then
    rm -f "$tmp_archive"
    log "PHP download failed: $url" >&2
    return 1
  fi
  mkdir -p "$extract_dir"
  if ! tar -xzf "$tmp_archive" -C "$extract_dir"; then
    rm -rf "$tmp_archive" "$extract_dir"
    log "Archive did not unpack: $url" >&2
    return 1
  fi
  extracted=""
  for candidate in "$extract_dir"/bin/*/bin/php "$extract_dir/bin/php" \
    "$extract_dir"/*/bin/*/bin/php "$extract_dir"/*/bin/php; do
    if [ -f "$candidate" ]; then
      extracted="$candidate"
      break
    fi
  done
  if [ "$extracted" = "" ]; then
    rm -rf "$tmp_archive" "$extract_dir"
    log "Archive has no bin/php: $url" >&2
    return 1
  fi
  bin_root=""
  dir="$(dirname "$extracted")"
  while [ "$dir" != "$extract_dir" ] && [ "$dir" != "/" ]; do
    if [ "$(basename "$dir")" = "bin" ]; then
      bin_root="$dir"
    fi
    dir="$(dirname "$dir")"
  done
  if [ "$bin_root" = "" ]; then
    rm -rf "$tmp_archive" "$extract_dir"
    log "Archive has no bin folder: $url" >&2
    return 1
  fi
  rm -rf "$tmp_archive" "$DATA_DIR/bin"
  mv "$bin_root" "$DATA_DIR/bin"
  rm -rf "$extract_dir"
  return 0
}

install_core() {
  tmp_phar="$DATA_DIR/.vtx/PocketMine-MP.phar"
  log "Downloading PocketMine-MP $2: $1"
  rm -f "$tmp_phar"
  if ! fetch "$1" -o "$tmp_phar"; then
    rm -f "$tmp_phar"
    log "PocketMine-MP download failed: $1" >&2
    return 1
  fi
  mv -f "$tmp_phar" "$CORE_FILE"
  return 0
}

fix_extension_dir() {
  ini="$(dirname "$1")/php.ini"
  php_root="$(dirname "$(dirname "$1")")"
  if [ ! -f "$ini" ]; then
    return 0
  fi
  for candidate in "$php_root"/lib/php/extensions/*debug-zts*; do
    if [ -d "$candidate" ]; then
      if ! grep -qxF "extension_dir=\"$candidate\"" "$ini"; then
        sed -i '/^extension_dir=/d' "$ini"
        printf 'extension_dir="%s"\n' "$candidate" >> "$ini"
      fi
      return 0
    fi
  done
}

set_property() {
  if grep -q "^$1=" "$PROPERTIES_FILE" 2>/dev/null; then
    sed -i "s/^$1=.*/$1=$2/" "$PROPERTIES_FILE" || true
  else
    printf '%s=%s\n' "$1" "$2" >> "$PROPERTIES_FILE"
  fi
}

PHP_BIN="$(find_php)"
CORE="$(find_core)"
INSTALLED_CORE="$(marker_line "$CORE_MARKER" 1)"
INSTALLED_PHP="$(marker_line "$PHP_MARKER" 1)"
INSTALLED_PHP_PATH="$(marker_line "$PHP_MARKER" 3)"
INSTALLED_PHP_URL="$(marker_line "$PHP_MARKER" 4)"
if [ "$INSTALLED_PHP_PATH" = "" ]; then
  INSTALLED_PHP_PATH="$DEFAULT_MANAGED_PHP"
fi

CORE_MANAGED=0
if [ "$CORE" = "" ]; then
  CORE_MANAGED=1
elif [ "$INSTALLED_CORE" != "" ]; then
  if [ "$CORE" = "$CORE_FILE" ] && [ "$(file_sum "$CORE_FILE")" = "$(marker_line "$CORE_MARKER" 2)" ]; then
    CORE_MANAGED=1
  else
    log "PocketMine-MP was replaced manually, automatic updates are off"
    rm -f "$CORE_MARKER"
    INSTALLED_CORE=""
  fi
fi

PHP_MANAGED=0
if [ "$PHP_BIN" = "" ]; then
  PHP_MANAGED=1
elif [ "$INSTALLED_PHP" != "" ]; then
  if [ "$PHP_BIN" = "$INSTALLED_PHP_PATH" ] && [ "$(file_sum "$INSTALLED_PHP_PATH")" = "$(marker_line "$PHP_MARKER" 2)" ]; then
    PHP_MANAGED=1
    MANAGED_PHP="$INSTALLED_PHP_PATH"
  else
    log "PHP was replaced manually, automatic updates are off"
    rm -f "$PHP_MARKER"
    INSTALLED_PHP=""
  fi
fi

WANT_PHP="${PHP_VERSION:-}"
if [ "$WANT_PHP" = "" ] && [ -f "$PHP_OVERRIDE_FILE" ]; then
  WANT_PHP="$(head -n 1 "$PHP_OVERRIDE_FILE" | tr -d '\r ')"
fi

INFO=""
if [ "$CORE_MANAGED" = "1" ] || { [ "$PHP_MANAGED" = "1" ] && [ "$WANT_PHP" = "" ]; }; then
  INFO="$(fetch "$BUILD_INFO_URL" 2>/dev/null || true)"
fi

if [ "$CORE_MANAGED" = "1" ]; then
  LATEST_CORE="$(json_field "$INFO" base_version)"
  CORE_URL="$(json_field "$INFO" download_url)"
  if [ "$LATEST_CORE" = "" ] || [ "$CORE_URL" = "" ]; then
    if [ "$CORE" = "" ]; then
      log "Could not get PocketMine-MP release info from $BUILD_INFO_URL" >&2
      log "Put PocketMine-MP.phar or src/ into the server folder" >&2
      exit 1
    fi
    log "Could not check for PocketMine-MP updates, starting installed version ${INSTALLED_CORE:-unknown}"
  elif [ "$LATEST_CORE" != "$INSTALLED_CORE" ] || [ "$CORE" = "" ]; then
    if [ "$INSTALLED_CORE" != "" ]; then
      log "Updating PocketMine-MP $INSTALLED_CORE -> $LATEST_CORE"
    fi
    if install_core "$CORE_URL" "$LATEST_CORE"; then
      write_marker "$CORE_MARKER" "$LATEST_CORE" "$CORE_FILE"
      INSTALLED_CORE="$LATEST_CORE"
      CORE="$CORE_FILE"
    elif [ "$CORE" = "" ]; then
      exit 1
    else
      log "Update failed, starting installed version ${INSTALLED_CORE:-unknown}"
    fi
  fi
fi

if [ "$PHP_MANAGED" = "1" ]; then
  if [ "$WANT_PHP" = "" ]; then
    WANT_PHP="$(json_field "$INFO" php_version)"
  fi
  if [ "$WANT_PHP" = "" ]; then
    if [ "$PHP_BIN" = "" ]; then
      log "Could not find out which PHP to download: put PHP into bin/ or write the version to /data/.vtx/php" >&2
      exit 1
    fi
  elif [ "$WANT_PHP" != "$INSTALLED_PHP" ] || [ "$PHP_BIN" = "" ] || php_source_changed "$WANT_PHP"; then
    if [ "$INSTALLED_PHP" != "" ] && [ "$WANT_PHP" != "$INSTALLED_PHP" ]; then
      log "Updating PHP $INSTALLED_PHP -> $WANT_PHP"
    elif [ "$INSTALLED_PHP" != "" ]; then
      log "PHP source changed, downloading $WANT_PHP again"
    fi
    if install_php "$WANT_PHP"; then
      MANAGED_PHP="$(find_php)"
      write_php_marker "$WANT_PHP" "$MANAGED_PHP" "$(php_url "$WANT_PHP")"
      INSTALLED_PHP="$WANT_PHP"
      PHP_BIN="$MANAGED_PHP"
    elif [ "$PHP_BIN" = "" ]; then
      exit 1
    else
      log "PHP update failed, starting installed PHP $INSTALLED_PHP"
    fi
  fi
fi

if [ ! -x "$PHP_BIN" ]; then
  chmod +x "$PHP_BIN" 2>/dev/null || true
fi
fix_extension_dir "$PHP_BIN"
export PHPRC=""

if ! PHP_RUNNING="$("$PHP_BIN" -r 'echo PHP_VERSION;' 2>&1)"; then
  log "PHP does not start: $PHP_BIN" >&2
  log "$PHP_RUNNING" >&2
  log "Use a Linux x86_64 PHP build, for example from github.com/pmmp/PHP-Binaries" >&2
  exit 1
fi

if [ ! -f "$PROPERTIES_FILE" ]; then
  cat > "$PROPERTIES_FILE" <<'EOF'
motd=PocketMine-MP Server
gamemode=survival
difficulty=2
enable-ipv6=off
EOF
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

if [ "$INSTALLED_CORE" != "" ]; then
  log "Starting PocketMine-MP $INSTALLED_CORE on PHP $PHP_RUNNING"
else
  log "Starting $(basename "$CORE") on PHP $PHP_RUNNING"
fi
exec "$PHP_BIN" "$CORE" --no-wizard "$@"
