package jobs

import (
	"fmt"
	"strings"
)

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

func sftpDenyBlock() string {
	return strings.Join([]string{
		"# " + sftpDenyMark,
		fmt.Sprintf("DenyGroups %s", sftpGroup),
	}, "\n")
}

func sftpCommands() []string {
	conf := strings.Join([]string{
		fmt.Sprintf("Port %d", sftpPort),
		"AddressFamily inet",
		"PidFile /run/vortanix-sftp.pid",
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
		"MaxAuthTries 3",
		"LoginGraceTime 30",
		"MaxStartups 10:30:60",
		"PermitOpen none",
		"ClientAliveInterval 30",
		"ClientAliveCountMax 4",
		"Subsystem sftp internal-sftp",
		"",
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
		fmt.Sprintf("sudo test -f %s || sudo ssh-keygen -q -t ed25519 -f %s -N '' -C vortanix-sftp",
			sftpHostKey, sftpHostKey),
		fmt.Sprintf("sudo chmod 600 %s", sftpHostKey),
		fmt.Sprintf("printf '%%s\\n' %s | sudo tee %s >/dev/null", shellQuoteScript(conf), sftpConfig),
		fmt.Sprintf("sudo chmod 600 %s", sftpConfig),
		fmt.Sprintf("sudo /usr/sbin/sshd -t -f %s", sftpConfig),
		fmt.Sprintf("printf '%%s\\n' %s | sudo tee %s >/dev/null", shellQuoteScript(unit), sftpUnit),
		"sudo systemctl daemon-reload",
		"sudo systemctl enable vortanix-sftp",
		"sudo systemctl restart vortanix-sftp",
		fmt.Sprintf("sudo ufw allow %d/tcp || true", sftpPort),

		"sudo mkdir -p /etc/fail2ban/jail.d",
		fmt.Sprintf("printf '%%s\\n' %s | sudo tee %s >/dev/null", shellQuoteScript(sftpJailBlock()), sftpJail),
		"sudo systemctl reload fail2ban 2>/dev/null || sudo systemctl restart fail2ban 2>/dev/null || true",

		"sudo mkdir -p /etc/ssh/sshd_config.d",
		fmt.Sprintf("printf '%%s\\n' %s | sudo tee %s >/dev/null",
			shellQuoteScript(sftpDenyBlock()), sftpDenyConf),
		fmt.Sprintf("sudo chmod 644 %s", sftpDenyConf),
		fmt.Sprintf(`if ! sudo grep -qE '^[[:space:]]*Include[[:space:]]+/etc/ssh/sshd_config\.d/' /etc/ssh/sshd_config; then
  if ! sudo grep -q %s /etc/ssh/sshd_config; then
    sudo cp -f /etc/ssh/sshd_config /etc/ssh/sshd_config.vortanix-backup
    sudo sed -i '1i # %s\nDenyGroups %s' /etc/ssh/sshd_config
  fi
fi`, shellQuote(sftpDenyMark), sftpDenyMark, sftpGroup),
		fmt.Sprintf(`if ! sudo sshd -t; then
  sudo rm -f %s
  if [ -f /etc/ssh/sshd_config.vortanix-backup ]; then
    sudo cp -f /etc/ssh/sshd_config.vortanix-backup /etc/ssh/sshd_config
  fi
  echo 'sshd отверг конфигурацию — запрет на админский вход не применён' >&2
  exit 1
fi`, sftpDenyConf),
		"sudo systemctl reload ssh 2>/dev/null || sudo systemctl reload sshd 2>/dev/null || true",
		fmt.Sprintf("sudo sshd -T 2>/dev/null | grep -qi '^denygroups .*%s' "+
			"&& echo 'админский sshd: вход группе %s запрещён' "+
			"|| echo 'ВНИМАНИЕ: запрет группе %s на админском sshd не подтверждён' >&2",
			sftpGroup, sftpGroup, sftpGroup),
		"sudo systemctl disable --now vsftpd >/dev/null 2>&1 || true",
		"sudo apt-get purge -y vsftpd >/dev/null 2>&1 || true",
		"sudo rm -f /etc/vsftpd.userlist /etc/vsftpd.conf /etc/vsftpd.conf.dpkg-dist || true",
		"sudo ufw delete allow 21/tcp >/dev/null 2>&1 || true",
		"sudo ufw delete allow 30000:30100/tcp >/dev/null 2>&1 || true",
		fmt.Sprintf("echo 'SFTP настроен: порт %d, каждый доступ заперт в своём каталоге'", sftpPort),
	}
}

func sftpChrootFor(username string) string {
	return sftpChrootDir + "/" + username
}

func sftpMountCommands(username, dataDir string) []string {
	root := sftpChrootFor(username)
	mount := root + "/data"
	qRoot := shQuote(root)
	qMount := shQuote(mount)
	qData := shQuote(dataDir)
	fstab := fmt.Sprintf("%s %s none bind,nosuid,nodev,nofail 0 0", dataDir, mount)

	return []string{
		fmt.Sprintf("mkdir -p %s", qMount),
		fmt.Sprintf("chown root:root %s", qRoot),
		fmt.Sprintf("chmod 755 %s", qRoot),
		fmt.Sprintf("mountpoint -q %s || mount --bind %s %s", qMount, qData, qMount),
		fmt.Sprintf("mount -o remount,bind,nosuid,nodev %s || true", qMount),
		fmt.Sprintf("grep -vF %s /etc/fstab > /tmp/vtx.fstab && mv /tmp/vtx.fstab /etc/fstab || true", qMount),
		fmt.Sprintf("echo %s >> /etc/fstab", shQuote(fstab)),
	}
}

func sftpOwnCommands(username, group, dataDir string) []string {
	qUser := shQuote(username)
	qGroup := shQuote(group)
	qDir := shQuote(dataDir)
	return []string{
		fmt.Sprintf("chown -h -R %s:%s %s", qUser, qGroup, qDir),
		fmt.Sprintf("chmod -R g+rwX %s", qDir),
		fmt.Sprintf("chmod 2775 %s", qDir),
	}
}

func sftpUnmountCommands(username string) []string {
	root := sftpChrootFor(username)
	mount := root + "/data"
	qRoot := shQuote(root)
	qMount := shQuote(mount)
	return []string{
		fmt.Sprintf("if mountpoint -q %s; then umount %s || umount -l %s; fi", qMount, qMount, qMount),
		fmt.Sprintf("if mountpoint -q %s; then echo 'bind не снят, удаление каталога отменено' >&2; exit 1; fi", qMount),
		fmt.Sprintf("if [ -f /etc/fstab ]; then grep -vF %s /etc/fstab > /tmp/vtx.fstab || true; mv /tmp/vtx.fstab /etc/fstab; fi", qMount),
		fmt.Sprintf("rm -rf --one-file-system %s", qRoot),
	}
}
