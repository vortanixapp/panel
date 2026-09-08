package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/vortanixapp/panel/pkg/sshclient"
)

var ftpUsernameRe = regexp.MustCompile(`^[a-z0-9_-]{1,32}$`)

const ftpGroupPrefix = "vtx"

func (r *Runner) FTPAccountLoop(ctx context.Context, wake <-chan struct{}) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-wake:
			r.heartbeat.Beat(LoopFTPAccount)
			r.drainFTPAccounts(ctx)
		case <-ticker.C:
			r.heartbeat.Beat(LoopFTPAccount)
			r.drainFTPAccounts(ctx)
		}
	}
}

func (r *Runner) drainFTPAccounts(ctx context.Context) {
	for r.processFTPAccount(ctx) {
	}
}

type ftpJobPayload struct {
	AccountID string `json:"account_id"`
	ServerID  string `json:"server_id"`
	// NodeID кладётся в задание при удалении сервера: строка сервера к моменту
	// уборки уже удалена, и найти ноду по ней будет негде.
	NodeID   string `json:"node_id"`
	Action   string `json:"action"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func (r *Runner) processFTPAccount(ctx context.Context) bool {
	jobID, tenantID, payload, ok := r.claimJob(ctx, "ftp_account")
	if !ok {
		return false
	}

	var pl ftpJobPayload
	_ = json.Unmarshal(payload, &pl)
	if pl.ServerID == "" || pl.Username == "" || pl.Action == "" {
		r.failJobGeneric(ctx, r.db, jobID, "missing server_id, username or action")
		return true
	}
	if !ftpUsernameRe.MatchString(pl.Username) {
		r.failFTPAccount(ctx, jobID, pl, "недопустимое имя пользователя")
		return true
	}

	nodeID := pl.NodeID
	if nodeID == "" {
		if err := r.db.QueryRow(ctx, `
			SELECT node_id::text FROM core.servers WHERE id = $1 AND tenant_id = $2
		`, pl.ServerID, tenantID).Scan(&nodeID); err != nil {
			r.failFTPAccount(ctx, jobID, pl, "сервер не найден")
			return true
		}
	}

	node, err := r.loadNodeSSH(ctx, r.db, tenantID, nodeID)
	if err != nil {
		r.failFTPAccount(ctx, jobID, pl, err.Error())
		return true
	}

	cfg := sshclient.Config{
		Host:     node.SSHHost,
		Port:     node.SSHPort,
		User:     node.SSHUser,
		Password: node.SSHPassword,
		Timeout:  45 * time.Second,
	}

	script := ""
	switch pl.Action {
	case "create":
		script = ftpCreateScript(pl.Username, pl.Password, pl.ServerID)
	case "password":
		script = ftpPasswordScript(pl.Username, pl.Password, pl.ServerID)
	case "delete":
		script = ftpDeleteScript(pl.Username, pl.ServerID)
	default:
		r.failFTPAccount(ctx, jobID, pl, "неизвестное действие: "+pl.Action)
		return true
	}

	if out, runErr := sshclient.RunCapture(cfg, script); runErr != nil {
		msg := strings.TrimSpace(out)
		if msg == "" {
			msg = runErr.Error()
		}
		r.failFTPAccount(ctx, jobID, pl, msg)
		return true
	}

	if pl.Action == "delete" {
		_, _ = r.db.Exec(ctx, `DELETE FROM core.server_ftp_accounts WHERE id = $1`, pl.AccountID)
	} else {
		_, _ = r.db.Exec(ctx, `
			UPDATE core.server_ftp_accounts
			SET status = 'active', error_message = NULL, updated_at = now()
			WHERE id = $1
		`, pl.AccountID)
	}
	_, _ = r.db.Exec(ctx, `UPDATE core.jobs SET status = 'completed', result = '{"ok":true}'::jsonb WHERE id = $1`, jobID)
	return true
}

func (r *Runner) failFTPAccount(ctx context.Context, jobID string, pl ftpJobPayload, msg string) {
	if pl.AccountID != "" {
		_, _ = r.db.Exec(ctx, `
			UPDATE core.server_ftp_accounts
			SET status = 'failed', error_message = $2, updated_at = now()
			WHERE id = $1
		`, pl.AccountID, msg)
	}
	r.failJobGeneric(ctx, r.db, jobID, msg)
	log.Printf("ftp_account %s failed: %s", pl.Action, msg)
}

// enqueueFtpCleanup ставит задания на снятие доступов удаляемого сервера.
//
// Аккаунты уедут каскадом вместе со строкой сервера, а пользователь Linux,
// монтирование и запись в fstab остались бы на ноде навсегда. node_id кладём
// в задание: сервера к моменту исполнения уже не будет.
func (r *Runner) enqueueFtpCleanup(ctx context.Context, tenantID, serverID, nodeID string) {
	rows, err := r.db.Query(ctx, `
		SELECT id::text, username FROM core.server_ftp_accounts WHERE server_id = $1
	`, serverID)
	if err != nil {
		log.Printf("ftp cleanup: не удалось выбрать доступы сервера %s: %v", serverID, err)
		return
	}
	type acc struct{ id, username string }
	list := []acc{}
	for rows.Next() {
		var a acc
		if rows.Scan(&a.id, &a.username) == nil {
			list = append(list, a)
		}
	}
	rows.Close()

	for _, a := range list {
		payload, _ := json.Marshal(map[string]any{
			"account_id": a.id,
			"server_id":  serverID,
			"node_id":    nodeID,
			"action":     "delete",
			"username":   a.username,
		})
		if _, err := r.db.Exec(ctx, `
			INSERT INTO core.jobs (tenant_id, type, status, payload)
			VALUES ($1, 'ftp_account', 'pending', $2::jsonb)
		`, tenantID, payload); err != nil {
			log.Printf("ftp cleanup: не удалось поставить задание для %s: %v", a.username, err)
		}
	}
}

func ftpDataDir(serverID string) string {
	base := envOr("VORTANIX_DATA_DIR", "/var/lib/vortanix/servers")
	return strings.TrimSuffix(base, "/") + "/" + serverID
}

func ftpGroup(serverID string) string {
	short := strings.ReplaceAll(serverID, "-", "")
	if len(short) > 12 {
		short = short[:12]
	}
	return ftpGroupPrefix + short
}

func ftpCreateScript(username, password, serverID string) string {
	dir := ftpDataDir(serverID)
	group := ftpGroup(serverID)
	qUser := shQuote(username)
	qDir := shQuote(dir)
	qGroup := shQuote(group)
	// Домашний каталог — /data внутри chroot: снаружи это по-прежнему каталог
	// сервера, просто пользователь видит его от корня своего заточения.
	cmds := []string{
		"set -e",
		fmt.Sprintf("mkdir -p %s", qDir),
		fmt.Sprintf("getent group %s >/dev/null 2>&1 || groupadd %s", qGroup, qGroup),
		fmt.Sprintf("getent group %s >/dev/null 2>&1 || groupadd %s", sftpGroup, sftpGroup),
		// Имя выводится из id сервера, и при совпадении useradd молча
		// пропускается — дальше usermod и перемонтирование увели бы ЧУЖОГО
		// пользователя на этот каталог: прежний клиент потерял бы доступ, а
		// новый получил его файлы. Группа кодирует другой отрезок того же id,
		// поэтому расхождение по ней — верный признак занятого имени. Лучше
		// отказать с понятной ошибкой, чем выдать чужие данные.
		fmt.Sprintf(`if id -u %s >/dev/null 2>&1 && [ "$(id -gn %s)" != %s ]; then`+"\n"+
			`  echo "имя доступа %s уже занято другим сервером" >&2; exit 1`+"\n"+
			`fi`, qUser, qUser, qGroup, qUser),
		fmt.Sprintf("id -u %s >/dev/null 2>&1 || useradd -d /data -s /usr/sbin/nologin -g %s -M %s",
			qUser, qGroup, qUser),
		fmt.Sprintf("usermod -d /data -g %s -aG %s %s || true", qGroup, sftpGroup, qUser),
		fmt.Sprintf("echo %s | chpasswd", shQuote(username+":"+password)),
	}
	cmds = append(cmds, sftpOwnCommands(username, group, dir)...)
	cmds = append(cmds, sftpMountCommands(username, dir)...)
	return strings.Join(cmds, "\n")
}

// ftpPasswordScript меняет пароль и заодно чинит доступ.
//
// Аккаунты, заведённые до перехода на SFTP, не состоят в группе vtxsftp и не
// имеют chroot — войти по новому порту они не могут. Все шаги идемпотентны,
// поэтому смена пароля работает и как кнопка «почини мне доступ».
func ftpPasswordScript(username, password, serverID string) string {
	dir := ftpDataDir(serverID)
	group := ftpGroup(serverID)
	qUser := shQuote(username)
	qGroup := shQuote(group)
	cmds := []string{
		"set -e",
		fmt.Sprintf("id -u %s >/dev/null 2>&1", qUser),
		fmt.Sprintf("getent group %s >/dev/null 2>&1 || groupadd %s", sftpGroup, sftpGroup),
		fmt.Sprintf("getent group %s >/dev/null 2>&1 || groupadd %s", qGroup, qGroup),
		fmt.Sprintf("usermod -d /data -g %s -aG %s %s || true", qGroup, sftpGroup, qUser),
		fmt.Sprintf("echo %s | chpasswd", shQuote(username+":"+password)),
	}
	// Права переставляем и здесь: это же кнопка «почини мне доступ», и после
	// переустановки сервера SteamCMD снова оставляет файлы за root.
	cmds = append(cmds, sftpOwnCommands(username, group, dir)...)
	cmds = append(cmds, sftpMountCommands(username, dir)...)
	return strings.Join(cmds, "\n")
}

func ftpDeleteScript(username, serverID string) string {
	qUser := shQuote(username)
	qGroup := shQuote(ftpGroup(serverID))
	qDir := shQuote(ftpDataDir(serverID))
	// Сначала снимаем монтирование, потом удаляем пользователя: userdel при
	// живом bind внутри домашнего каталога способен утащить за собой данные
	// сервера.
	cmds := []string{"set -e"}
	cmds = append(cmds, sftpUnmountCommands(username)...)
	cmds = append(cmds,
		fmt.Sprintf("id -u %s >/dev/null 2>&1 && userdel %s || true", qUser, qUser),
		// Файлы возвращаем root. Иначе они остаются за освободившимся uid, и
		// следующий заведённый на ноде пользователь получает их вместе с
		// номером — а номера система переиспользует.
		fmt.Sprintf("[ -d %s ] && chown -h -R root:root %s || true", qDir, qDir),
		// Группу сервера убираем следом. Если на сервере остался ещё один
		// доступ, она у него первичная и groupdel откажется — так и нужно.
		fmt.Sprintf("getent group %s >/dev/null 2>&1 && groupdel %s 2>/dev/null || true", qGroup, qGroup),
	)
	return strings.Join(cmds, "\n")
}

func shQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}
