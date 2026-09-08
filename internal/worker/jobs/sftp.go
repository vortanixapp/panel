package jobs

import (
	"fmt"
	"strings"
)

// Доступ к файлам серверов идёт по SFTP.
//
// Раньше стоял vsftpd с включённым, но необязательным TLS и самоподписанным
// сертификатом Debian: клиент спокойно логинился открытым текстом, а проверить
// сертификат всё равно не мог — у нод нет доменов, только адреса, и выпустить
// нормальный сертификат не на что. Плюс пассивных портов было десять на всю
// ноду, то есть десять одновременных передач независимо от числа серверов.
//
// У SFTP обеих бед нет: шифрование обязательное, удостоверяющий центр не нужен
// вовсе (проверяется ключ хоста), и порт всего один.

const (
	sftpPort      = 2222
	sftpGroup     = "vtxsftp"
	sftpChrootDir = "/var/lib/vortanix/sftp"
	sftpConfig    = "/etc/ssh/sshd_config.vortanix-sftp"
	sftpUnit      = "/etc/systemd/system/vortanix-sftp.service"
	sftpDenyConf  = "/etc/ssh/sshd_config.d/00-vortanix-sftp-deny.conf"
	sftpDenyMark  = "vortanix-sftp-deny"
	sftpHostKey   = "/etc/ssh/vortanix_sftp_host_ed25519_key"
	sftpJail      = "/etc/fail2ban/jail.d/vortanix-sftp.conf"
)

// sftpJailBlock — отдельный jail fail2ban для демона файлов.
//
// Стандартный jail sshd банит по `port = ssh`, то есть попытки на 2222 он не
// видит вовсе. А демон смотрит в интернет и пускает по паролю.
func sftpJailBlock() string {
	return strings.Join([]string{
		"[vortanix-sftp]",
		"enabled = true",
		fmt.Sprintf("port = %d", sftpPort),
		"filter = sshd",
		"backend = systemd",
		"journalmatch = _SYSTEMD_UNIT=vortanix-sftp.service",
		"maxretry = 5",
		"findtime = 600",
		"bantime = 3600",
	}, "\n")
}

// sftpDenyBlock — врезка в конфигурацию ОСНОВНОГО sshd ноды.
func sftpDenyBlock() string {
	return strings.Join([]string{
		"# " + sftpDenyMark,
		fmt.Sprintf("# Файлы игровых серверов отдаёт отдельный sshd на порту %d (юнит", sftpPort),
		"# vortanix-sftp), и все ограничения заданы в его конфиге. На админском",
		"# демоне эти учётные записи пускать нельзя: nologin закрывает шелл и exec,",
		"# но не проброс портов — для него нужна отдельная директива.",
		fmt.Sprintf("DenyGroups %s", sftpGroup),
	}, "\n")
}

// sftpCommands поднимает на ноде отдельный экземпляр sshd только для файлов.
//
// Отдельный, а не Match-блок в основном: порт 22 — это наш админский канал, там
// fail2ban, и клиентские подборы пароля начали бы закрывать доступ нам самим.
func sftpCommands() []string {
	conf := strings.Join([]string{
		fmt.Sprintf("Port %d", sftpPort),
		"AddressFamily inet",
		// Свой pid-файл: с общим по умолчанию два sshd затирают его друг другу,
		// и перезапуск основного, админского, начинает промахиваться мимо
		// своего процесса.
		"PidFile /run/vortanix-sftp.pid",
		// Свой ключ хоста. Иначе файловый сервис для клиентов и админский канал
		// предъявляют один и тот же отпечаток, и клиент, однажды его
		// запомнивший, не отличит один от другого.
		fmt.Sprintf("HostKey %s", sftpHostKey),
		"PermitRootLogin no",
		"PasswordAuthentication yes",
		"PubkeyAuthentication yes",
		"KbdInteractiveAuthentication no",
		"UsePAM yes",
		"X11Forwarding no",
		"AllowAgentForwarding no",
		"AllowTcpForwarding no",
		"PermitTunnel no",
		"PrintMotd no",
		// Демон смотрит в интернет и пускает по паролю, поэтому подбор надо
		// делать дорогим: попыток на соединение мало, окно ввода короткое,
		// одновременных полуоткрытых сессий немного.
		"MaxAuthTries 3",
		"LoginGraceTime 30",
		"MaxStartups 10:30:60",
		"PermitOpen none",
		"ClientAliveInterval 30",
		"ClientAliveCountMax 4",
		"Subsystem sftp internal-sftp",
		"",
		"# Пускаем только владельцев файлов серверов и запираем каждого в свой",
		"# каталог. ForceCommand отрезает шелл: даже с верным паролем это доступ",
		"# к файлам, а не к машине.",
		fmt.Sprintf("AllowGroups %s", sftpGroup),
		fmt.Sprintf("Match Group %s", sftpGroup),
		fmt.Sprintf("    ChrootDirectory %s/%%u", sftpChrootDir),
		"    ForceCommand internal-sftp -u 0002",
		"    AllowTcpForwarding no",
		"    X11Forwarding no",
	}, "\n")

	unit := strings.Join([]string{
		"[Unit]",
		"Description=Vortanix SFTP (файлы игровых серверов)",
		"After=network.target",
		"",
		"[Service]",
		fmt.Sprintf("ExecStart=/usr/sbin/sshd -D -f %s", sftpConfig),
		"ExecReload=/bin/kill -HUP $MAINPID",
		"KillMode=process",
		"Restart=on-failure",
		"RestartSec=5",
		"",
		"[Install]",
		"WantedBy=multi-user.target",
	}, "\n")

	return []string{
		"sudo apt-get install -y openssh-server",
		fmt.Sprintf("sudo mkdir -p %s && sudo chown root:root %s && sudo chmod 755 %s",
			sftpChrootDir, sftpChrootDir, sftpChrootDir),
		fmt.Sprintf("getent group %s >/dev/null 2>&1 || sudo groupadd %s", sftpGroup, sftpGroup),
		// Ключ хоста создаём до проверки конфига: без файла sshd -t упадёт.
		fmt.Sprintf("sudo test -f %s || sudo ssh-keygen -q -t ed25519 -f %s -N '' -C vortanix-sftp",
			sftpHostKey, sftpHostKey),
		fmt.Sprintf("sudo chmod 600 %s", sftpHostKey),
		fmt.Sprintf("printf '%%s\\n' %s | sudo tee %s >/dev/null", shellQuoteScript(conf), sftpConfig),
		fmt.Sprintf("sudo chmod 600 %s", sftpConfig),
		// Конфигурацию проверяем до запуска: sshd с ошибкой в файле не
		// стартует, и без проверки мы узнали бы об этом от клиента.
		fmt.Sprintf("sudo /usr/sbin/sshd -t -f %s", sftpConfig),
		fmt.Sprintf("printf '%%s\\n' %s | sudo tee %s >/dev/null", shellQuoteScript(unit), sftpUnit),
		"sudo systemctl daemon-reload",
		"sudo systemctl enable vortanix-sftp",
		"sudo systemctl restart vortanix-sftp",
		fmt.Sprintf("sudo ufw allow %d/tcp || true", sftpPort),

		// Свой jail: стандартный sshd-jail слушает порт ssh и попыток на 2222
		// не считает.
		"sudo mkdir -p /etc/fail2ban/jail.d",
		fmt.Sprintf("printf '%%s\\n' %s | sudo tee %s >/dev/null", shellQuoteScript(sftpJailBlock()), sftpJail),
		"sudo systemctl reload fail2ban 2>/dev/null || sudo systemctl restart fail2ban 2>/dev/null || true",

		// Учётка файлового доступа — обычный пользователь Linux с паролем, а
		// все ограничения выше действуют ТОЛЬКО на нашем демоне 2222: основной
		// sshd ноды панель не настраивает нигде. С теми же логином и паролем,
		// которые панель показывает клиенту открытым текстом, он логинился на
		// админский порт. Шелла бы не получил — nologin, — но nologin не
		// закрывает канал direct-tcpip, для него есть отдельная директива
		// AllowTcpForwarding. То есть из файлового доступа поднимался
		// SOCKS-прокси внутрь ноды: `ssh -N -D 1080 vtx…@нода`, и дальше
		// видны MySQL, phpMyAdmin, приватная сеть и адрес метаданных облака.
		"sudo mkdir -p /etc/ssh/sshd_config.d",
		fmt.Sprintf("printf '%%s\\n' %s | sudo tee %s >/dev/null",
			shellQuoteScript(sftpDenyBlock()), sftpDenyConf),
		fmt.Sprintf("sudo chmod 644 %s", sftpDenyConf),
		// Каталог sshd_config.d читает не всякая сборка: без строки Include в
		// основном конфиге файл молча игнорируется. Тогда дописываем запрет в
		// сам sshd_config первой строкой — до любого блока Match, иначе
		// директива оказалась бы внутри чужого блока и не сработала бы.
		fmt.Sprintf(`if ! sudo grep -qE '^[[:space:]]*Include[[:space:]]+/etc/ssh/sshd_config\.d/' /etc/ssh/sshd_config; then
  if ! sudo grep -q %s /etc/ssh/sshd_config; then
    sudo cp -f /etc/ssh/sshd_config /etc/ssh/sshd_config.vortanix-backup
    sudo sed -i '1i # %s\nDenyGroups %s' /etc/ssh/sshd_config
  fi
fi`, shellQuote(sftpDenyMark), sftpDenyMark, sftpGroup),
		// Проверяем ДО перезагрузки: сломанный конфиг основного sshd отрезал бы
		// от ноды и нас самих. Не прошло — откатываем и валим шаг установки.
		fmt.Sprintf(`if ! sudo sshd -t; then
  sudo rm -f %s
  if [ -f /etc/ssh/sshd_config.vortanix-backup ]; then
    sudo cp -f /etc/ssh/sshd_config.vortanix-backup /etc/ssh/sshd_config
  fi
  echo 'sshd отверг конфигурацию — запрет на админский вход не применён' >&2
  exit 1
fi`, sftpDenyConf),
		// reload, а не restart: перезапуск оборвал бы текущую сессию установки.
		"sudo systemctl reload ssh 2>/dev/null || sudo systemctl reload sshd 2>/dev/null || true",
		fmt.Sprintf("sudo sshd -T 2>/dev/null | grep -qi '^denygroups .*%s' "+
			"&& echo 'админский sshd: вход группе %s запрещён' "+
			"|| echo 'ВНИМАНИЕ: запрет группе %s на админском sshd не подтверждён' >&2",
			sftpGroup, sftpGroup, sftpGroup),
		// Старый vsftpd сносим сразу, как только новый доступ поднялся:
		// оставить открытым порт 21 с паролями открытым текстом — ровно та
		// дыра, ради которой всё это и переделывалось. Учётные записи при
		// этом не трогаем: те же пользователи Linux работают и по SFTP,
		// доступ им чинит смена пароля в панели.
		"sudo systemctl disable --now vsftpd >/dev/null 2>&1 || true",
		"sudo apt-get purge -y vsftpd >/dev/null 2>&1 || true",
		"sudo rm -f /etc/vsftpd.userlist /etc/vsftpd.conf /etc/vsftpd.conf.dpkg-dist || true",
		"sudo ufw delete allow 21/tcp >/dev/null 2>&1 || true",
		"sudo ufw delete allow 30000:30100/tcp >/dev/null 2>&1 || true",
		fmt.Sprintf("echo 'SFTP настроен: порт %d, каждый доступ заперт в своём каталоге'", sftpPort),
	}
}

// sftpChrootFor — корень chroot пользователя. Он обязан принадлежать root и не
// быть доступным на запись, поэтому данные лежат не в нём самом, а в
// подкаталоге data, который приезжает туда bind-монтированием.
func sftpChrootFor(username string) string {
	return sftpChrootDir + "/" + username
}

// sftpMountCommands готовит каталог доступа: корень chroot и данные сервера
// внутри него.
//
// Bind, а не перенос файлов: каталог сервера уже смонтирован в работающий
// контейнер игры, и переезд означал бы пересоздание всех серверов.
func sftpMountCommands(username, dataDir string) []string {
	root := sftpChrootFor(username)
	mount := root + "/data"
	qRoot := shQuote(root)
	qMount := shQuote(mount)
	qData := shQuote(dataDir)
	// nofail обязателен: если каталог сервера исчез, без него нода уходит в
	// аварийный режим при загрузке из-за одной строки в fstab.
	// nosuid и nodev: в этот каталог клиент пишет сам, и ни setuid-бинарю, ни
	// узлу устройства взяться там неоткуда по делу. Само по себе это не спасло
	// бы от многого — исполнять с хоста эти файлы всё равно некому, — но и
	// стоит ровно два слова.
	fstab := fmt.Sprintf("%s %s none bind,nosuid,nodev,nofail 0 0", dataDir, mount)

	return []string{
		fmt.Sprintf("mkdir -p %s", qMount),
		// Корень chroot принадлежит root и закрыт на запись — иначе sshd
		// отказывается пускать и пишет об этом только в свой журнал.
		fmt.Sprintf("chown root:root %s", qRoot),
		fmt.Sprintf("chmod 755 %s", qRoot),
		fmt.Sprintf("mountpoint -q %s || mount --bind %s %s", qMount, qData, qMount),
		// mount --bind опции игнорирует, их ставит только повторный remount.
		fmt.Sprintf("mount -o remount,bind,nosuid,nodev %s || true", qMount),
		// Запись в fstab, чтобы доступ пережил перезагрузку ноды. Старую
		// строку этой точки монтирования сначала убираем: иначе повторный
		// запуск накапливает дубли и монтирует каталог поверх самого себя.
		fmt.Sprintf("grep -vF %s /etc/fstab > /tmp/vtx.fstab && mv /tmp/vtx.fstab /etc/fstab || true", qMount),
		fmt.Sprintf("echo %s >> /etc/fstab", shQuote(fstab)),
	}
}

// sftpOwnCommands отдаёт каталог сервера доступу целиком.
//
// chown одного верхнего каталога мало. Файлы игры кладёт SteamCMD в своём
// контейнере от root, и обычно раньше, чем клиент вообще заведёт доступ:
// установленное до этого остаётся root:root с правами 755. Клиент такие файлы
// видит и читает, но не может ни отредактировать server.cfg, ни положить плагин
// в подкаталог, ни что-либо удалить — то есть файловый доступ есть, а
// пользоваться им нельзя. Бит setgid здесь не спасает: он раздаёт группу только
// новым файлам, уже созданные не трогает.
//
// Запуск игры от смены владельца не страдает: контейнер работает от root и
// права файловой системы игнорирует.
func sftpOwnCommands(username, group, dataDir string) []string {
	qUser := shQuote(username)
	qGroup := shQuote(group)
	qDir := shQuote(dataDir)
	return []string{
		// -h обязателен. Без него chown идёт по символической ссылке и меняет
		// владельца её цели: клиенту достаточно положить в свой каталог ссылку
		// на /etc/shadow и нажать «сменить пароль», чтобы файл стал его.
		fmt.Sprintf("chown -h -R %s:%s %s", qUser, qGroup, qDir),
		// g+rwX, а не g+rwx: заглавная X добавляет право входа каталогам и уже
		// исполняемым файлам, но не делает исполняемым каждый конфиг. Символические
		// ссылки chmod при обходе не трогает.
		fmt.Sprintf("chmod -R g+rwX %s", qDir),
		// setgid без sticky. setgid нужен: новые файлы контейнера получают
		// группу сервера, и после рекурсивного chown выше биту наследования
		// наконец есть к чему применяться. А вот sticky здесь только вредил:
		// каталог не общий для чужих друг другу людей, зато при втором
		// FTP-доступе на тот же сервер владельцем становится новый
		// пользователь, и первый переставал удалять то, что удалял раньше.
		fmt.Sprintf("chmod 2775 %s", qDir),
	}
}

func sftpUnmountCommands(username string) []string {
	root := sftpChrootFor(username)
	mount := root + "/data"
	qRoot := shQuote(root)
	qMount := shQuote(mount)
	return []string{
		// umount падает с EBUSY, если у клиента открыт FileZilla. Раньше ошибку
		// глушило `|| true`, а следующая строка делала rm -rf по корню клетки —
		// сквозь ЖИВОЕ монтирование, то есть сносила файлы работающего сервера
		// на хосте. Достаточно было удалить доступ, не закрыв окно.
		// Ленивое размонтирование убирает точку из дерева сразу, даже когда
		// дескрипторы ещё открыты.
		fmt.Sprintf("if mountpoint -q %s; then umount %s || umount -l %s; fi", qMount, qMount, qMount),
		// Если точка всё-таки на месте — лучше оставить мусор на ноде, чем
		// удалить данные клиента.
		fmt.Sprintf("if mountpoint -q %s; then echo 'bind не снят, удаление каталога отменено' >&2; exit 1; fi", qMount),
		fmt.Sprintf("if [ -f /etc/fstab ]; then grep -vF %s /etc/fstab > /tmp/vtx.fstab || true; mv /tmp/vtx.fstab /etc/fstab; fi", qMount),
		// --one-file-system: если монтирование каким-то образом уцелело, rm
		// остановится на границе файловой системы и не уйдёт в каталог сервера.
		// И без `|| true`: set -e должен останавливать скрипт, а не прятать сбой.
		fmt.Sprintf("rm -rf --one-file-system %s", qRoot),
	}
}
