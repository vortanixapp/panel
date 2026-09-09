#!/bin/sh
set -eu

DATA_DIR="/data"
STARTUP_FILE="$DATA_DIR/.vtx/startup_params"
STARTUP_ARGV="$DATA_DIR/.vtx/startup_argv"
STARTUP_FILE_OLD="$DATA_DIR/.vtx_startup_params"
SERVER_JAR="${SERVER_JAR:-$DATA_DIR/server.jar}"
EULA_FILE="$DATA_DIR/eula.txt"
SERVER_PROPERTIES_FILE="$DATA_DIR/server.properties"

mkdir -p "$DATA_DIR"
cd "$DATA_DIR"

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
    JVM_EXTRA=""
    ARGS_EXTRA=""

    for t in $STARTUP_LINE; do
      case "$t" in
        -X*|-XX:*|-D*|-javaagent:*|-agentlib:*|-agentpath:*)
          JVM_EXTRA="$JVM_EXTRA $t"
          ;;
        *)
          ARGS_EXTRA="$ARGS_EXTRA $t"
          ;;
      esac
    done

    JVM_EXTRA="$(printf '%s' "$JVM_EXTRA" | sed -e 's/^ *//; s/ *$//')"
    ARGS_EXTRA="$(printf '%s' "$ARGS_EXTRA" | sed -e 's/^ *//; s/ *$//')"

    XMX_VAL=$(echo "$JVM_EXTRA" | grep -oE -- '-Xmx[0-9]+[mMkKgG]' | tail -n 1 | sed 's/-Xmx//')
    if [ "$XMX_VAL" != "" ]; then
      export MEMORY="$XMX_VAL"
      export INIT_MEMORY="$XMX_VAL"
      export MAX_MEMORY="$XMX_VAL"
      echo "[vtx-entrypoint] Extracted MEMORY=$XMX_VAL from startup_params"
    fi

    if [ "$JVM_EXTRA" != "" ]; then
      if [ "${JVM_OPTS:-}" != "" ]; then
        export JVM_OPTS="${JVM_OPTS} ${JVM_EXTRA}"
      else
        export JVM_OPTS="${JVM_EXTRA}"
      fi
    fi

    if [ "$ARGS_EXTRA" != "" ]; then
      if [ "${EXTRA_ARGS:-}" != "" ]; then
        export EXTRA_ARGS="${EXTRA_ARGS} ${ARGS_EXTRA}"
      else
        export EXTRA_ARGS="${ARGS_EXTRA}"
      fi
    fi
  fi
fi

if [ ! -f "$EULA_FILE" ] || ! grep -q '^eula=true$' "$EULA_FILE" 2>/dev/null; then
  printf 'eula=true\n' > "$EULA_FILE"
fi

if [ ! -f "$SERVER_JAR" ]; then
  echo "[vtx-entrypoint] server jar not found: $SERVER_JAR"
  echo "[vtx-entrypoint] Put your jar at /data/server.jar or set SERVER_JAR env"
  exit 1
fi

if [ "${SERVER_PORT:-}" != "" ]; then
  if [ -f "$SERVER_PROPERTIES_FILE" ]; then
    if grep -q '^server-port=' "$SERVER_PROPERTIES_FILE" 2>/dev/null; then
      sed -i "s/^server-port=.*/server-port=${SERVER_PORT}/" "$SERVER_PROPERTIES_FILE" || true
    else
      printf '\nserver-port=%s\n' "$SERVER_PORT" >> "$SERVER_PROPERTIES_FILE"
    fi

    if grep -q '^enable-query=' "$SERVER_PROPERTIES_FILE" 2>/dev/null; then
      sed -i 's/^enable-query=.*/enable-query=true/' "$SERVER_PROPERTIES_FILE" || true
    else
      printf '\nenable-query=true\n' >> "$SERVER_PROPERTIES_FILE"
    fi

    if grep -q '^query\.port=' "$SERVER_PROPERTIES_FILE" 2>/dev/null; then
      sed -i "s/^query\\.port=.*/query.port=${SERVER_PORT}/" "$SERVER_PROPERTIES_FILE" || true
    else
      printf 'query.port=%s\n' "$SERVER_PORT" >> "$SERVER_PROPERTIES_FILE"
    fi
  else
    printf 'server-port=%s\n' "$SERVER_PORT" > "$SERVER_PROPERTIES_FILE"
    printf 'enable-query=true\n' >> "$SERVER_PROPERTIES_FILE"
    printf 'query.port=%s\n' "$SERVER_PORT" >> "$SERVER_PROPERTIES_FILE"
  fi
fi

JVM_OPTS_FINAL="${JVM_OPTS:-}"
EXTRA_ARGS_FINAL="${EXTRA_ARGS:-}"

if ! printf '%s' "$JVM_OPTS_FINAL" | grep -Eq -- '(^|[[:space:]])-Xmx[0-9]+[mMkKgG]'; then
  MEM_FROM_ENV="${MEMORY:-${MAX_MEMORY:-}}"
  if [ "$MEM_FROM_ENV" != "" ]; then
    JVM_OPTS_FINAL="${JVM_OPTS_FINAL} -Xmx${MEM_FROM_ENV}"
  fi
fi

if ! printf '%s' "$JVM_OPTS_FINAL" | grep -Eq -- '(^|[[:space:]])-Xms[0-9]+[mMkKgG]'; then
  INIT_FROM_ENV="${INIT_MEMORY:-}"
  if [ "$INIT_FROM_ENV" != "" ]; then
    JVM_OPTS_FINAL="${JVM_OPTS_FINAL} -Xms${INIT_FROM_ENV}"
  fi
fi

if ! printf '%s' "$JVM_OPTS_FINAL" | grep -Eq -- '(^|[[:space:]])-Xmx[0-9]+[mMkKgG]'; then
  JVM_OPTS_FINAL="${JVM_OPTS_FINAL} -Xmx1024M"
fi

if ! printf '%s' "$JVM_OPTS_FINAL" | grep -Eq -- '(^|[[:space:]])-Xms[0-9]+[mMkKgG]'; then
  JVM_OPTS_FINAL="${JVM_OPTS_FINAL} -Xms512M"
fi

JVM_OPTS_FINAL="$(printf '%s' "$JVM_OPTS_FINAL" | sed -e 's/^ *//; s/ *$//')"
EXTRA_ARGS_FINAL="$(printf '%s' "$EXTRA_ARGS_FINAL" | sed -e 's/^ *//; s/ *$//')"

echo "[vtx-entrypoint] Starting MC Java with jar: $SERVER_JAR"
if [ "$JVM_OPTS_FINAL" != "" ]; then
  echo "[vtx-entrypoint] JVM opts: $JVM_OPTS_FINAL"
fi
if [ "$EXTRA_ARGS_FINAL" != "" ]; then
  echo "[vtx-entrypoint] Extra args: $EXTRA_ARGS_FINAL"
fi

exec sh -c "exec java $JVM_OPTS_FINAL -jar \"$SERVER_JAR\" nogui $EXTRA_ARGS_FINAL"
