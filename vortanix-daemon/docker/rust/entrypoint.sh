#!/bin/sh
set -eu

umask 0002

DATA_DIR="/data"

if [ -f "$DATA_DIR/rust.env" ]; then
  set -a
  . "$DATA_DIR/rust.env"
  set +a
fi

is_uint() {
  case "${1:-}" in
    ''|*[!0-9]*) return 1 ;;
    *) return 0 ;;
  esac
}

RUST_PORT="${RUST_PORT:-}"
if [ "$RUST_PORT" = "" ]; then
  echo "RUST_PORT is required" >&2
  exit 1
fi
if ! is_uint "$RUST_PORT" || [ "$RUST_PORT" -lt 1 ] || [ "$RUST_PORT" -gt 65535 ]; then
  echo "Invalid RUST_PORT: $RUST_PORT" >&2
  exit 1
fi

QUERY_PORT=$((RUST_PORT + 1))
if [ "$QUERY_PORT" -gt 65535 ]; then
  echo "Invalid Rust query port: $QUERY_PORT" >&2
  exit 1
fi

MAXPLAYERS_RAW="${RUST_MAXPLAYERS:-50}"
if ! is_uint "$MAXPLAYERS_RAW" || [ "$MAXPLAYERS_RAW" -lt 1 ] || [ "$MAXPLAYERS_RAW" -gt 500 ]; then
  MAXPLAYERS_RAW="50"
fi
MAXPLAYERS="$MAXPLAYERS_RAW"

HOSTNAME="${RUST_HOSTNAME:-Vortanix Rust Server}"
WORLD_SIZE_RAW="${RUST_WORLD_SIZE:-3500}"
if ! is_uint "$WORLD_SIZE_RAW" || [ "$WORLD_SIZE_RAW" -lt 1000 ] || [ "$WORLD_SIZE_RAW" -gt 6000 ]; then
  WORLD_SIZE_RAW="3500"
fi
WORLD_SIZE="$WORLD_SIZE_RAW"

SEED_RAW="${RUST_SEED:-}"
if [ "$SEED_RAW" = "" ]; then
  SEED_RAW="$(date +%s)"
fi
if ! is_uint "$SEED_RAW"; then
  SEED_RAW="$(date +%s)"
fi
SEED="$SEED_RAW"

IDENTITY="${RUST_IDENTITY:-server}"
LEVEL="${RUST_LEVEL:-Procedural Map}"

RCON_PASSWORD="${RUST_RCON_PASSWORD:-}"
if [ "$RCON_PASSWORD" = "" ]; then
  RCON_PASSWORD="vtx$(cat /proc/sys/kernel/random/uuid | tr -d '-' | cut -c1-20)"
fi

mkdir -p "$DATA_DIR"

BIN=""
if [ -x "$DATA_DIR/RustDedicated" ]; then
  BIN="$DATA_DIR/RustDedicated"
fi

if [ "$BIN" = "" ]; then
  echo "[rust] ERROR: RustDedicated not found in /data. Install a game version archive (Версии) so daemon can unpack server files." >&2
  find "$DATA_DIR" -maxdepth 4 -name RustDedicated -type f 2>/dev/null | head -n 50 >&2 || true
  exit 1
fi

if [ "$#" -eq 0 ]; then
  set -- "$BIN" \
    -batchmode -nographics \
    +server.hostname "$HOSTNAME" \
    +server.port "$RUST_PORT" \
    +server.queryport "$QUERY_PORT" \
    +server.maxplayers "$MAXPLAYERS" \
    +server.identity "$IDENTITY" \
    +server.level "$LEVEL" \
    +server.seed "$SEED" \
    +server.worldsize "$WORLD_SIZE" \
    +rcon.password "$RCON_PASSWORD" \
    +rcon.port "$QUERY_PORT" \
    +rcon.web 1
fi

exec "$@"
