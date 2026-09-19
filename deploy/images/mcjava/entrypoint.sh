#!/bin/sh
set -eu

DATA_DIR="/data"
STARTUP_FILE="$DATA_DIR/.vtx/startup_params"
STARTUP_ARGV="$DATA_DIR/.vtx/startup_argv"
STARTUP_FILE_OLD="$DATA_DIR/.vtx_startup_params"
JAVA_FILE="$DATA_DIR/.vtx/java"
JAVA_ROOT="/opt/java"
JAVA_VERSIONS="8 17 21 25"
SERVER_JAR="${SERVER_JAR:-}"
EULA_FILE="$DATA_DIR/eula.txt"
SERVER_PROPERTIES_FILE="$DATA_DIR/server.properties"

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
      log "Extracted MEMORY=$XMX_VAL from startup_params"
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

is_uint() {
  case "${1:-}" in
    ''|*[!0-9]*) return 1 ;;
    *) return 0 ;;
  esac
}

jar_entry() {
  unzip -p "$1" "$2" 2>/dev/null || true
}

jar_has() {
  unzip -l "$1" "$2" >/dev/null 2>&1
}

main_class() {
  jar_entry "$1" META-INF/MANIFEST.MF | tr -d '\r' | sed -n 's/^Main-Class: *//p' | head -n 1
}

class_java() {
  class_path="$(printf '%s' "$2" | tr '.' '/').class"
  major="$(jar_entry "$1" "$class_path" | head -c 8 | od -An -tu1 | awk 'NF >= 8 { print $7 * 256 + $8 }')"
  if is_uint "$major" && [ "$major" -ge 45 ]; then
    echo $((major - 44))
  fi
}

mc_java() {
  mc_major="$(printf '%s' "$1" | cut -d. -f1)"
  mc_minor="$(printf '%s' "$1" | cut -s -d. -f2 | sed 's/[^0-9].*//')"
  mc_patch="$(printf '%s' "$1" | cut -s -d. -f3 | sed 's/[^0-9].*//')"
  is_uint "$mc_major" || return 0
  is_uint "$mc_minor" || mc_minor=0
  is_uint "$mc_patch" || mc_patch=0
  if [ "$mc_major" -gt 1 ]; then
    echo 25
  elif [ "$mc_minor" -le 16 ]; then
    echo 8
  elif [ "$mc_minor" -lt 20 ] || { [ "$mc_minor" -eq 20 ] && [ "$mc_patch" -le 4 ]; }; then
    echo 17
  else
    echo 21
  fi
}

json_string() {
  grep -o "\"$1\" *: *\"[^\"]*\"" | head -n 1 | sed 's/.*"\([^"]*\)"$/\1/'
}

jar_java() {
  jar="$1"
  if jar_has "$jar" version.json; then
    required="$(jar_entry "$jar" version.json | grep -o '"java_version" *: *[0-9]*' | head -n 1 | grep -o '[0-9]*$' || true)"
    if is_uint "$required"; then
      echo "$required version.json"
      return
    fi
  fi
  if jar_has "$jar" install.properties; then
    game="$(jar_entry "$jar" install.properties | tr -d '\r' | sed -n 's/^game-version=//p' | head -n 1)"
    required="$(mc_java "$game")"
    if is_uint "$required"; then
      echo "$required Minecraft $game"
      return
    fi
  fi
  main="$(main_class "$jar")"
  if [ "$main" != "" ]; then
    required="$(class_java "$jar" "$main")"
    if is_uint "$required"; then
      echo "$required $main"
      return
    fi
  fi
}

java_home_for() {
  for v in $JAVA_VERSIONS; do
    if [ "$v" -ge "$1" ] && [ -x "$JAVA_ROOT/$v/bin/java" ]; then
      echo "$JAVA_ROOT/$v"
      return
    fi
  done
}

select_java() {
  wanted=""
  reason=""
  if [ -f "$JAVA_FILE" ]; then
    wanted="$(head -n 1 "$JAVA_FILE" | tr -dc '0-9')"
    reason="$JAVA_FILE"
  fi
  if [ "$wanted" = "" ] && is_uint "${JAVA_VERSION:-}"; then
    wanted="$JAVA_VERSION"
    reason="JAVA_VERSION"
  fi
  if [ "$wanted" = "" ] && [ "${1:-}" != "" ]; then
    wanted="${1%% *}"
    reason="${1#* }"
  fi
  JAVA_HOME_SELECTED=""
  if is_uint "$wanted"; then
    JAVA_HOME_SELECTED="$(java_home_for "$wanted")"
  fi
  if [ "$JAVA_HOME_SELECTED" = "" ]; then
    JAVA_HOME_SELECTED="$(java_home_for 25)"
    [ "$JAVA_HOME_SELECTED" = "" ] && JAVA_HOME_SELECTED="$(dirname "$(dirname "$(readlink -f "$(command -v java)")")")"
    reason="default"
  fi
  JAVA_BIN="$JAVA_HOME_SELECTED/bin/java"
  log "Java $(basename "$JAVA_HOME_SELECTED") ($reason), to override: echo 17 > /data/.vtx/java"
}

forge_args_file() {
  for pattern in \
    "libraries/net/minecraftforge/forge/*/unix_args.txt" \
    "libraries/net/neoforged/neoforge/*/unix_args.txt" \
    "libraries/net/neoforged/forge/*/unix_args.txt"
  do
    found="$(ls -1 $pattern 2>/dev/null | sort -V | tail -n 1 || true)"
    if [ "$found" != "" ]; then
      echo "$found"
      return
    fi
  done
}

forge_args_java() {
  loader_version="$(basename "$(dirname "$1")")"
  case "$1" in
    libraries/net/neoforged/neoforge/*)
      nf_major="$(printf '%s' "$loader_version" | cut -d. -f1)"
      nf_minor="$(printf '%s' "$loader_version" | cut -s -d. -f2)"
      if is_uint "$nf_major" && [ "$nf_major" -le 21 ]; then
        mc_java "1.$nf_major.$nf_minor"
      else
        mc_java "$loader_version"
      fi
      ;;
    *)
      mc_java "${loader_version%%-*}"
      ;;
  esac
}

legacy_forge_jar() {
  ls -1 forge-*.jar 2>/dev/null | grep -v -- '-installer' | sort -V | tail -n 1 || true
}

run_forge_installer() {
  installer="$1"
  game="$(jar_entry "$installer" install_profile.json | json_string minecraft)"
  select_java "$(mc_java "$game") Forge installer for Minecraft ${game:-unknown}"
  log "Installing Forge server from $(basename "$installer")"
  attempt=1
  while ! "$JAVA_BIN" -jar "$installer" --installServer; do
    if [ "$attempt" -ge 3 ]; then
      log "Forge installer failed, last lines of $(basename "$installer").log:" >&2
      tail -n 20 "$installer.log" >&2 2>/dev/null || true
      exit 1
    fi
    attempt=$((attempt + 1))
    log "Forge installer failed, retrying ($attempt/3)"
    sleep 5
  done
  rm -f "$installer.log" "$DATA_DIR/installer.log" 2>/dev/null || true
  mv -f "$installer" "$DATA_DIR/.vtx/forge-installer.jar"
}

if [ ! -f "$EULA_FILE" ] || ! grep -q '^eula=true$' "$EULA_FILE" 2>/dev/null; then
  printf 'eula=true\n' > "$EULA_FILE"
fi

if [ "$SERVER_JAR" = "" ] && [ -f "$DATA_DIR/server.jar" ] && jar_has "$DATA_DIR/server.jar" install_profile.json; then
  run_forge_installer "$DATA_DIR/server.jar"
fi

LAUNCH=""
ARGS_FILE=""
if [ "$SERVER_JAR" != "" ]; then
  LAUNCH="jar"
else
  ARGS_FILE="$(forge_args_file)"
  if [ "$ARGS_FILE" != "" ]; then
    LAUNCH="args"
  elif [ -f "$DATA_DIR/server.jar" ]; then
    SERVER_JAR="$DATA_DIR/server.jar"
    LAUNCH="jar"
  else
    SERVER_JAR="$(legacy_forge_jar)"
    if [ "$SERVER_JAR" != "" ]; then
      SERVER_JAR="$DATA_DIR/$SERVER_JAR"
      LAUNCH="jar"
    fi
  fi
fi

if [ "$LAUNCH" = "jar" ] && [ ! -f "$SERVER_JAR" ]; then
  LAUNCH=""
fi

if [ "$LAUNCH" = "" ]; then
  log "server jar not found: ${SERVER_JAR:-$DATA_DIR/server.jar}"
  log "Put your jar at /data/server.jar or set SERVER_JAR env"
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

if [ "$LAUNCH" = "args" ]; then
  select_java "$(forge_args_java "$ARGS_FILE") $ARGS_FILE"
  USER_ARGS=""
  if [ -f "$DATA_DIR/user_jvm_args.txt" ]; then
    USER_ARGS="@user_jvm_args.txt"
  fi
  log "Starting MC Java with $ARGS_FILE"
  TARGET="$USER_ARGS @$ARGS_FILE"
else
  select_java "$(jar_java "$SERVER_JAR")"
  log "Starting MC Java with jar: $SERVER_JAR"
  TARGET="-jar \"$SERVER_JAR\""
fi

if [ "$JVM_OPTS_FINAL" != "" ]; then
  log "JVM opts: $JVM_OPTS_FINAL"
fi
if [ "$EXTRA_ARGS_FINAL" != "" ]; then
  log "Extra args: $EXTRA_ARGS_FINAL"
fi

export JAVA_HOME="$JAVA_HOME_SELECTED"
export PATH="$JAVA_HOME/bin:$PATH"
exec sh -c "exec \"$JAVA_BIN\" $JVM_OPTS_FINAL $TARGET nogui $EXTRA_ARGS_FINAL"
