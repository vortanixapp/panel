package jobs

import (
	"fmt"
	"strings"
)

const (
	quotaDataDir = "/var/lib/vortanix/servers"
	quotaImage   = "/var/lib/vortanix/servers.img"
	quotaScript  = "/opt/vortanix/node-disk-quota.sh"
)

func diskQuotaCommands(meta map[string]any) []string {
	size := ""
	if raw, ok := meta["quota_size"]; ok {
		if s, ok := raw.(string); ok {
			size = strings.TrimSpace(s)
		}
	}

	return []string{
		"sudo apt-get install -y quota",
		"sudo mkdir -p /opt/vortanix",
		fmt.Sprintf("printf '%%s\\n' %s | sudo tee %s >/dev/null",
			shellQuoteScript(diskQuotaScript()), quotaScript),
		fmt.Sprintf("sudo chmod 700 %s", quotaScript),
		fmt.Sprintf("sudo VORTANIX_QUOTA_SIZE=%s sh %s", shellQuoteScript(size), quotaScript),
	}
}

func diskQuotaScript() string {
	return strings.Join([]string{
		"#!/bin/sh",
		"set -eu",
		"",
		fmt.Sprintf("DIR=%s", quotaDataDir),
		fmt.Sprintf("IMG=%s", quotaImage),
		"SIZE=\"${VORTANIX_QUOTA_SIZE:-}\"",
		"",
		"if findmnt -no OPTIONS -T \"$DIR\" 2>/dev/null | tr ',' '\\n' | grep -qx prjquota; then",
		"  echo \"квоты уже включены для $DIR\"",
		"  exit 0",
		"fi",
		"",
		"quota_module() {",
		"  modprobe quota_v2 2>/dev/null && return 0",
		"  grep -q 'quota_v2' \"/lib/modules/$(uname -r)/modules.builtin\" 2>/dev/null",
		"}",
		"",
		"if ! quota_module; then",
		"  echo \"в ядре $(uname -r) не загружается модуль квот quota_v2, ставлю linux-modules-extra-$(uname -r)\"",
		"  DEBIAN_FRONTEND=noninteractive apt-get install -y \"linux-modules-extra-$(uname -r)\" >/dev/null 2>&1 || true",
		"  if ! quota_module; then",
		"    echo \"ядро $(uname -r) не поддерживает дисковые квоты (нет модуля quota_v2)\" >&2",
		"    echo \"поставьте обычное ядро linux-image-generic, перезагрузите сервер и повторите шаг; игровые серверы не останавливались\" >&2",
		"    exit 1",
		"  fi",
		"fi",
		"",
		"mkdir -p \"$(dirname \"$IMG\")\" \"$DIR\"",
		"",
		"TMP=\"\"",
		"running=\"\"",
		"cleanup() {",
		"  code=$?",
		"  if [ -n \"$TMP\" ]; then",
		"    umount \"$TMP\" 2>/dev/null || true",
		"    rmdir \"$TMP\" 2>/dev/null || true",
		"  fi",
		"  if [ -n \"$running\" ]; then",
		"    echo \"$running\" | xargs -r docker start >/dev/null 2>&1 || true",
		"    echo \"серверы запущены обратно\"",
		"  fi",
		"  exit \"$code\"",
		"}",
		"trap cleanup EXIT",
		"",
		"running=$(docker ps -q --filter 'label=vortanix.managed=true' 2>/dev/null || true)",
		"if [ -n \"$running\" ]; then",
		"  echo \"останавливаю серверов: $(echo \"$running\" | wc -l)\"",
		"  echo \"$running\" | xargs -r docker stop >/dev/null",
		"fi",
		"",
		"formatted() {",
		"  [ -f \"$IMG\" ] || return 1",
		"  [ \"$(blkid -p -s TYPE -o value \"$IMG\" 2>/dev/null || true)\" = ext4 ] && return 0",
		"  dumpe2fs -h \"$IMG\" >/dev/null 2>&1",
		"}",
		"",
		"if formatted; then",
		"  echo \"образ $IMG уже размечен, данные в нём сохраняются\"",
		"  e2fsck -fp \"$IMG\" >/dev/null 2>&1 || true",
		"  tune2fs -O project,quota \"$IMG\" >/dev/null",
		"else",
		"  if [ ! -f \"$IMG\" ]; then",
		"    if [ -z \"$SIZE\" ]; then",
		"      avail=$(df -Pm \"$(dirname \"$IMG\")\" | awk 'NR==2{print $4}')",
		"      SIZE=\"$(( avail * 80 / 100 ))M\"",
		"    fi",
		"    echo \"размер образа: $SIZE\"",
		"    truncate -s \"$SIZE\" \"$IMG\"",
		"  fi",
		"  mkfs.ext4 -q -F -O project,quota -E root_owner=0:0 \"$IMG\"",
		"fi",
		"",
		"TMP=$(mktemp -d)",
		"mount -o loop,prjquota \"$IMG\" \"$TMP\"",
		"if [ -n \"$(ls -A \"$DIR\" 2>/dev/null || true)\" ]; then",
		"  cp -a \"$DIR/.\" \"$TMP/\"",
		"  echo \"перенесено: $(du -sh \"$TMP\" | cut -f1)\"",
		"fi",
		"umount \"$TMP\"",
		"rmdir \"$TMP\"",
		"TMP=\"\"",
		"",
		"mount -o loop,prjquota \"$IMG\" \"$DIR\"",
		"quotaon -P \"$DIR\" 2>/dev/null || true",
		"",
		"if ! grep -q \"^$IMG \" /etc/fstab 2>/dev/null; then",
		"  printf '%s %s ext4 loop,prjquota,noatime 0 2\\n' \"$IMG\" \"$DIR\" >> /etc/fstab",
		"fi",
		"",
		"findmnt -no SOURCE,TARGET,OPTIONS \"$DIR\"",
		"echo 'готово: лимиты проставит агент при следующем запуске каждого сервера'",
	}, "\n")
}
