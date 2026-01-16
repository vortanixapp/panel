#!/bin/sh
set -eu

umask 0002

DATA_DIR="/data"

# ------------------------
# utils
# ------------------------
is_uint() {
  case "${1:-}" in
    ''|*[!0-9]*) return 1 ;;
    *) return 0 ;;
  esac
}

# ------------------------
# validate env
# ------------------------
MTA_PORT="${MTA_PORT:-}"
if [ -z "$MTA_PORT" ]; then
  echo "MTA_PORT is required" >&2
  exit 1
fi

if ! is_uint "$MTA_PORT" || [ "$MTA_PORT" -lt 1 ] || [ "$MTA_PORT" -gt 65535 ]; then
  echo "Invalid MTA_PORT: $MTA_PORT" >&2
  exit 1
fi

# ------------------------
# prepare data dir
# ------------------------
mkdir -p "$DATA_DIR"

if [ ! -x "$DATA_DIR/mta-server64" ] && [ ! -x "$DATA_DIR/mta-server" ] && [ ! -x "$DATA_DIR/x64/mta-server64" ] && [ ! -x "$DATA_DIR/server/mta-server64" ]; then
  echo "[mta] ERROR: MTA server binary not found in /data. Upload or install a game version archive (Версии) so daemon can unpack server files." >&2
  ls -la "$DATA_DIR" 2>/dev/null || true
  exit 1
fi

# ------------------------
# detect root + deathmatch
# ------------------------
ROOT_DIR="$DATA_DIR"
DM_DIR="$DATA_DIR/mods/deathmatch"

if [ -d "$DATA_DIR/server/mods/deathmatch" ]; then
  ROOT_DIR="$DATA_DIR/server"
  DM_DIR="$ROOT_DIR/mods/deathmatch"
fi

mkdir -p "$DM_DIR"

CONF="$DM_DIR/mtaserver.conf"

# ------------------------
# create config if missing
# ------------------------
if [ ! -f "$CONF" ]; then
  cat > "$CONF" <<EOF
<config>
    <servername>${MTA_HOSTNAME:-MTA Docker Server}</servername>
    <serverport>${MTA_PORT}</serverport>
    <httpport>$((MTA_PORT + 2))</httpport>
    <aseport>$((MTA_PORT + 123))</aseport>
    <acl>acl.xml</acl>
    <rights>rights.xml</rights>
    <maxplayers>32</maxplayers>
    <password></password>
    <fpslimit>36</fpslimit>
</config>
EOF
fi

ACL_FILE="$DM_DIR/acl.xml"
RIGHTS_FILE="$DM_DIR/rights.xml"

if [ ! -f "$ACL_FILE" ]; then
  cat > "$ACL_FILE" <<'EOF'
<acl>
    <group name="Admin">
        <acl name="Admin"/>
        <object name="user.console"/>
        <object name="user.kick"/>
        <object name="user.ban"/>
        <object name="user.modchat"/>
        <object name="resource.*"/>
        <object name="function.*"/>
    </group>
    <acl name="Admin">
        <right name="general.ModifyOtherObjects" access="true"/>
        <right name="general.http" access="true"/>
        <right name="command.*" access="true"/>
        <right name="function.*" access="true"/>
        <right name="resource.*" access="true"/>
    </acl>
    <group name="Everyone">
        <acl name="Default"/>
    </group>
    <acl name="Default">
        <right name="general.http" access="true"/>
    </acl>
    <user name="Console" password="" />
</acl>
EOF
fi

if [ ! -f "$RIGHTS_FILE" ]; then
  cat > "$RIGHTS_FILE" <<'EOF'
<rights>
    <right name="general.ModifyOtherObjects" />
    <right name="general.http" />
    <right name="command.*" />
    <right name="function.*" />
    <right name="resource.*" />
</rights>
EOF
fi

if grep -q '<acl>' "$CONF"; then
  sed -i "s|<acl>.*</acl>|<acl>acl.xml</acl>|" "$CONF" || true
else
  sed -i "s|</config>|    <acl>acl.xml</acl>\n</config>|" "$CONF" || true
fi

if grep -q '<rights>' "$CONF"; then
  sed -i "s|<rights>.*</rights>|<rights>rights.xml</rights>|" "$CONF" || true
else
  sed -i "s|</config>|    <rights>rights.xml</rights>\n</config>|" "$CONF" || true
fi

# ------------------------
# update ports / hostname
# ------------------------
HTTP_PORT=$((MTA_PORT + 2))
ASE_PORT=$((MTA_PORT + 123))

sed -i "s|<serverport>.*</serverport>|<serverport>${MTA_PORT}</serverport>|" "$CONF"
sed -i "s|<httpport>.*</httpport>|<httpport>${HTTP_PORT}</httpport>|" "$CONF"
sed -i "s|<aseport>.*</aseport>|<aseport>${ASE_PORT}</aseport>|" "$CONF"

if [ -n "${MTA_HOSTNAME:-}" ]; then
  esc=$(printf '%s' "$MTA_HOSTNAME" | sed 's/[&|\\]/\\&/g')
  sed -i "s|<servername>.*</servername>|<servername>${esc}</servername>|" "$CONF"
fi

# ------------------------
# find server binary
# ------------------------
BIN=""

for candidate in \
  "$ROOT_DIR/mta-server64" \
  "$ROOT_DIR/mta-server" \
  "$ROOT_DIR/x64/mta-server64"
do
  if [ -x "$candidate" ]; then
    BIN="$candidate"
    break
  fi
done

if [ -z "$BIN" ]; then
  echo "MTA server binary not found in $ROOT_DIR" >&2
  find "$ROOT_DIR" -maxdepth 3 -name 'mta-server*' 2>/dev/null || true
  exit 1
fi

# ------------------------
# Start server MTA
# ------------------------
cd "$ROOT_DIR"

echo "Starting MTA server:"
echo "  Root:   $ROOT_DIR"
echo "  Binary: $BIN"
echo "  Config: $CONF"

exec "$BIN" -c "$CONF"
