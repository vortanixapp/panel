#!/bin/sh
set -eu

umask 0002

DATA_DIR="/data"
GAME_KEY="${VTX_GAME_KEY:-native}"
GAME_NAME="${VTX_GAME_NAME:-$GAME_KEY}"

is_uint() {
  case "${1:-}" in
    ''|*[!0-9]*) return 1 ;;
    *) return 0 ;;
  esac
}

PORT_ENV="${VTX_PORT_ENV:-GAME_PORT}"
PORT="$(eval "printf '%s' \"\${$PORT_ENV:-}\"")"
if [ "$PORT" != "" ] && ! is_uint "$PORT"; then
  echo "[$GAME_KEY] Invalid $PORT_ENV: $PORT" >&2
  exit 1
fi
export GAME_PORT="${PORT:-}"

export HOME="$DATA_DIR"
export LD_LIBRARY_PATH="$DATA_DIR:$DATA_DIR/lib:${LD_LIBRARY_PATH:-}"
mkdir -p "$DATA_DIR" 2>/dev/null || true
cd "$DATA_DIR"

BINARIES="${VTX_SERVER_BINARIES:-}"
BINARIES="$BINARIES:./start-server.sh:./start.sh:./server.sh:./run.sh"

BIN=""
OLD_IFS="$IFS"
IFS=':'
for c in $BINARIES; do
  [ "$c" = "" ] && continue
  if [ -f "$c" ] && [ ! -x "$c" ]; then
    chmod +x "$c" 2>/dev/null || true
  fi
  if [ -x "$c" ]; then
    BIN="$c"
    break
  fi
done
IFS="$OLD_IFS"

if [ "$BIN" = "" ]; then
  echo "[$GAME_KEY] ERROR: server binary not found in /data for $GAME_NAME" >&2
  echo "[$GAME_KEY] looked for: $BINARIES" >&2
  ls -la "$DATA_DIR" 2>/dev/null || true
  exit 1
fi

if [ "$#" -eq 0 ]; then
  set -- "$BIN"
  if [ "${VTX_PARAM_ARGS:-}" != "" ]; then
    set -- "$@" ${VTX_PARAM_ARGS}
  fi
fi

STARTUP_FILE="$DATA_DIR/.vtx/startup_params"
STARTUP_ARGV="$DATA_DIR/.vtx/startup_argv"
STARTUP_FILE_OLD="$DATA_DIR/.vtx_startup_params"
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
    if echo "$STARTUP_LINE" | grep -Eq '^\./|^/'; then
      set -- $STARTUP_LINE
    else
      set -- "$@" $STARTUP_LINE
    fi
  fi
fi

exec "$@"
