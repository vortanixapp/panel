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
		"mkdir -p \"$(dirname \"$IMG\")\" \"$DIR\"",
		"",
		"if [ -z \"$SIZE\" ]; then",
		"  avail=$(df -Pm \"$(dirname \"$IMG\")\" | awk 'NR==2{print $4}')",
		"  SIZE=\"$(( avail * 80 / 100 ))M\"",
		"fi",
		"echo \"размер образа: $SIZE\"",
		"",
		"running=$(docker ps -q --filter 'label=vortanix.managed=true' 2>/dev/null || true)",
		"if [ -n \"$running\" ]; then",
		"  echo \"останавливаю серверов: $(echo \"$running\" | wc -l)\"",
		"  echo \"$running\" | xargs -r docker stop >/dev/null",
		"fi",
		"",
		"[ -f \"$IMG\" ] || truncate -s \"$SIZE\" \"$IMG\"",
		"mkfs.ext4 -q -F -O project,quota -E root_owner=0:0 \"$IMG\"",
		"",
		"TMP=$(mktemp -d)",
		"mount -o loop,prjquota \"$IMG\" \"$TMP\"",
		"if [ -n \"$(ls -A \"$DIR\" 2>/dev/null || true)\" ]; then",
		"  cp -a \"$DIR/.\" \"$TMP/\"",
		"  echo \"перенесено: $(du -sh \"$TMP\" | cut -f1)\"",
		"fi",
		"umount \"$TMP\"",
		"rmdir \"$TMP\"",
		"",
		"mount -o loop,prjquota \"$IMG\" \"$DIR\"",
		"quotaon -P \"$DIR\" 2>/dev/null || true",
		"",
		"if ! grep -q \"^$IMG \" /etc/fstab 2>/dev/null; then",
		"  printf '%s %s ext4 loop,prjquota,noatime 0 2\\n' \"$IMG\" \"$DIR\" >> /etc/fstab",
		"fi",
		"",
		"if [ -n \"${running:-}\" ]; then",
		"  echo \"$running\" | xargs -r docker start >/dev/null",
		"  echo \"серверы запущены обратно\"",
		"fi",
		"",
		"findmnt -no SOURCE,TARGET,OPTIONS \"$DIR\"",
		"echo 'готово: лимиты проставит агент при следующем запуске каждого сервера'",
	}, "\n")
}
