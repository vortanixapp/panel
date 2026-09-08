package jobs

import (
	"fmt"
	"strings"
)

// Шаг установки «Дисковые квоты».
//
// Без него disk_mb из тарифа остаётся цифрой на экране: каталог серверов лежит
// на корневой файловой системе, где квот нет, и клиент по SFTP может занять
// весь диск локации — то есть остановить чужие серверы.
//
// Прямо на корне квоты не включить без перезагрузки ноды: ext4 требует признак
// project в суперблоке и монтирование с prjquota, а для корня и то и другое
// означает остановку всех игровых серверов. Поэтому поднимаем отдельную
// файловую систему в файле-образе и монтируем её в каталог серверов:
// перезагрузка не нужна, предел держит то же ядро, а на ноду добавляется одно
// loop-устройство вместо одного на сервер.

const (
	quotaDataDir = "/var/lib/vortanix/servers"
	quotaImage   = "/var/lib/vortanix/servers.img"
	quotaScript  = "/opt/vortanix/node-disk-quota.sh"
)

func diskQuotaCommands(meta map[string]any) []string {
	// Размер образа. Пусто означает «80% свободного места»: считает сам скрипт
	// на ноде, потому что здесь свободного места мы не знаем.
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

// diskQuotaScript — то, что выполняется на ноде.
//
// Скрипт, а не набор команд по отдельности: между созданием файловой системы и
// переносом данных нельзя прерваться, а список команд шага выполняется по
// одной, и падение в середине оставило бы каталог серверов пустым при живых
// данных внутри образа.
func diskQuotaScript() string {
	return strings.Join([]string{
		"#!/bin/sh",
		"set -eu",
		"",
		fmt.Sprintf("DIR=%s", quotaDataDir),
		fmt.Sprintf("IMG=%s", quotaImage),
		"SIZE=\"${VORTANIX_QUOTA_SIZE:-}\"",
		"",
		"# Повторный запуск шага не должен ничего ломать: если квоты уже",
		"# работают, выходим не тронув ни данные, ни запущенные серверы.",
		"if findmnt -no OPTIONS -T \"$DIR\" 2>/dev/null | tr ',' '\\n' | grep -qx prjquota; then",
		"  echo \"квоты уже включены для $DIR\"",
		"  exit 0",
		"fi",
		"",
		"mkdir -p \"$(dirname \"$IMG\")\" \"$DIR\"",
		"",
		"# По умолчанию 80% свободного места. Не всё: на разделе живут ещё",
		"# образы docker и система, и отдать им ноль значит уронить ноду вместо",
		"# превышения квоты у одного клиента.",
		"if [ -z \"$SIZE\" ]; then",
		"  avail=$(df -Pm \"$(dirname \"$IMG\")\" | awk 'NR==2{print $4}')",
		"  SIZE=\"$(( avail * 80 / 100 ))M\"",
		"fi",
		"echo \"размер образа: $SIZE\"",
		"",
		"# Останавливаем до переноса: копировать каталог, в который сервер",
		"# пишет, значит получить на новой ФС испорченные файлы.",
		"running=$(docker ps -q --filter 'label=vortanix.managed=true' 2>/dev/null || true)",
		"if [ -n \"$running\" ]; then",
		"  echo \"останавливаю серверов: $(echo \"$running\" | wc -l)\"",
		"  echo \"$running\" | xargs -r docker stop >/dev/null",
		"fi",
		"",
		"[ -f \"$IMG\" ] || truncate -s \"$SIZE\" \"$IMG\"",
		"# -O project,quota — признак в суперблоке; без него монтирование с",
		"# prjquota молча не даст квот.",
		"mkfs.ext4 -q -F -O project,quota -E root_owner=0:0 \"$IMG\"",
		"",
		"TMP=$(mktemp -d)",
		"mount -o loop,prjquota \"$IMG\" \"$TMP\"",
		"# Копируем с сохранением владельцев и прав: каталоги серверов",
		"# принадлежат пользователям SFTP-доступа, и потерять владельца значит",
		"# отобрать у клиента его же файлы.",
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
		"# Без записи в fstab после перезагрузки ноды каталог серверов окажется",
		"# пустым: данные останутся в образе, а игры запустятся на пустом месте.",
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
