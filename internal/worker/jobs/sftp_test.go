package jobs

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// checkShellSyntax прогоняет сгенерированный скрипт через `bash -n`. Команды
// собираются форматированием строк, и опечатка в кавычке или незакрытый if
// проявились бы только на ноде, посреди установки.
func checkShellSyntax(t *testing.T, name, script string) {
	t.Helper()
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash недоступен — проверка синтаксиса пропущена")
	}
	path := filepath.Join(t.TempDir(), name+".sh")
	if err := os.WriteFile(path, []byte("set -e\n"+script+"\n"), 0o600); err != nil {
		t.Fatalf("не записать скрипт: %v", err)
	}
	out, err := exec.Command(bash, "-n", path).CombinedOutput()
	if err != nil {
		t.Errorf("%s: bash отверг скрипт: %v\n%s", name, err, out)
	}
}

func TestSetupScriptsAreValidShell(t *testing.T) {
	for _, component := range []string{"packages", "docker", "mysql", "phpmyadmin", "ftp", "daemon"} {
		cmds, err := setupCommands(component, nil, "token", "node-1", "wss://relay.example/v1/agent/connect", nil)
		if err != nil {
			t.Fatalf("%s: %v", component, err)
		}
		checkShellSyntax(t, component, strings.Join(cmds, "\n"))
	}
}

// Пароль root у MySQL вычислялся из имени контейнера и порта, а порт открыт
// наружу — то есть удалённый root был у любого, кто умеет считать.
func TestMySQLRootPasswordIsNotDerivable(t *testing.T) {
	script := strings.Join(mysqlDockerCommands(nil), "\n")

	if strings.Contains(script, "MYSQL_ROOT_PASSWORD=\"vtx_") {
		t.Error("контейнер снова поднимается с вычисляемым паролем root")
	}
	if !strings.Contains(script, "openssl rand") {
		t.Error("пароль root не генерируется случайно")
	}
	if !strings.Contains(script, "MYSQL_ROOT_HOST=localhost") {
		t.Error("образ заведёт удалённую учётку root")
	}
	if !strings.Contains(script, "DROP USER IF EXISTS 'root'@'%'") {
		t.Error("на уже поднятых нодах удалённый root не убирается")
	}
	if !strings.Contains(script, "ALTER USER IF EXISTS 'root'@'localhost'") {
		t.Error("старый вычисляемый пароль не ротируется на существующих нодах")
	}
	if strings.Contains(script, `{"instances":[]}`) {
		t.Error("в файл инстансов снова пишется пустой список — агент откатится на вычисляемый пароль")
	}
	// Агент читает пароли из /opt/vortanix, значит каталог обязан быть ему виден.
	daemon := strings.Join(daemonAgentCommands("t", "n", "wss://r/v1/agent/connect", ""), "\n")
	if !strings.Contains(daemon, "/opt/vortanix:/opt/vortanix") {
		t.Error("агенту не примонтирован каталог с паролями MySQL")
	}
}

func TestSftpScriptsAreValidShell(t *testing.T) {
	checkShellSyntax(t, "setup", strings.Join(sftpCommands(), "\n"))
	checkShellSyntax(t, "create", ftpCreateScript("vtxaabbccddeeff", "pw", "aabbccdd-eeff-0011-2233-445566778899"))
	checkShellSyntax(t, "password", ftpPasswordScript("vtxaabbccddeeff", "pw", "aabbccdd-eeff-0011-2233-445566778899"))
	checkShellSyntax(t, "delete", ftpDeleteScript("vtxaabbccddeeff", "aabbccdd-eeff-0011-2233-445566778899"))
}

// Вход по учётке файлового доступа обязан быть закрыт на ОСНОВНОМ sshd ноды.
// Иначе тех же логина и пароля хватает, чтобы поднять проброс портов внутрь
// ноды: nologin запрещает шелл и exec, но не канал direct-tcpip.
func TestSftpSetupClosesAdminSSHDForFileAccounts(t *testing.T) {
	script := strings.Join(sftpCommands(), "\n")

	if !strings.Contains(script, "DenyGroups "+sftpGroup) {
		t.Error("основному sshd не запрещают вход группе файловых доступов")
	}
	if !strings.Contains(script, sftpDenyConf) {
		t.Error("врезка для основного sshd не кладётся в sshd_config.d")
	}
	if !strings.Contains(script, "Include") {
		t.Error("нет запасного пути для сборок, где sshd_config.d не подключён")
	}
	// Проверка конфига обязана идти до перезагрузки демона.
	check := strings.Index(script, "sshd -t")
	reload := strings.Index(script, "systemctl reload ssh")
	if check < 0 || reload < 0 || check > reload {
		t.Error("конфиг основного sshd перезагружают, не проверив его перед этим")
	}
	if !strings.Contains(script, "sshd_config.vortanix-backup") {
		t.Error("нет отката основного конфига, если sshd его отверг")
	}
}

// Удаление доступа не должно сносить файлы работающего сервера: rm шёл по
// корню клетки, а внутри неё оставался живой bind каталога сервера.
func TestSftpUnmountRefusesToDeleteThroughLiveBind(t *testing.T) {
	script := strings.Join(sftpUnmountCommands("vtxaabbccddeeff"), "\n")

	rmLine := ""
	for _, line := range strings.Split(script, "\n") {
		if strings.Contains(line, "rm -rf") {
			rmLine = line
			break
		}
	}
	if rmLine == "" {
		t.Fatal("не найдена строка удаления корня клетки")
	}
	if !strings.Contains(rmLine, "--one-file-system") {
		t.Error("rm по корню клетки без --one-file-system: уйдёт в каталог сервера")
	}
	if strings.Contains(rmLine, "|| true") {
		t.Error("сбой rm снова заглушён — set -e не остановит скрипт")
	}
	if !strings.Contains(script, "umount -l") {
		t.Error("нет ленивого размонтирования: umount падает с EBUSY при открытой сессии клиента")
	}
	guard := strings.Index(script, "удаление каталога отменено")
	rm := strings.Index(script, "rm -rf")
	if guard < 0 || rm < 0 || guard > rm {
		t.Error("перед удалением не проверяют, что монтирование действительно снято")
	}
}

// Права должны переставляться на всё дерево, иначе клиент видит свои файлы,
// но не может отредактировать ни один конфиг в подкаталоге.
func TestSftpOwnershipIsRecursiveAndSymlinkSafe(t *testing.T) {
	script := strings.Join(sftpOwnCommands("vtxaabbccddeeff", "vtxaabbccddee", "/var/lib/vortanix/servers/x"), "\n")

	if !strings.Contains(script, "chown -h -R") {
		t.Error("chown не рекурсивный либо без -h: без -h он идёт по симлинку и меняет владельца чужой цели")
	}
	if !strings.Contains(script, "chmod -R g+rwX") {
		t.Error("группе не возвращают право записи на уже установленные файлы")
	}
	if strings.Contains(script, "chmod 3775") {
		t.Error("sticky на каталоге сервера отбирает право удаления у первого доступа при появлении второго")
	}
	if !strings.Contains(script, "chmod 2775") {
		t.Error("без setgid новые файлы контейнера не получат группу сервера")
	}
	for _, s := range []string{ftpCreateScript("u", "p", "id"), ftpPasswordScript("u", "p", "id")} {
		if !strings.Contains(s, "chown -h -R") {
			t.Error("рекурсивная выдача прав применяется не во всех путях (создание и починка доступа)")
		}
	}
}
