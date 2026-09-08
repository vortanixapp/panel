package jobs

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

func setupCommands(component string, meta map[string]any, agentToken, nodeID, relayURL string, images []string) ([]string, error) {
	switch component {
	case "packages":
		return []string{
			"sudo apt-get update -y",
			"sudo apt-get install -y curl wget git htop vim nano ufw fail2ban unzip libarchive-tools python3",
			"sudo dpkg --add-architecture i386 || true",
			"sudo apt-get update -y",
			"sudo apt-get install -y libc6-i386 lib32gcc-s1 libstdc++6:i386 || true",
		}, nil
	case "docker":
		return []string{
			"curl -fsSL https://get.docker.com | sudo sh",
			"sudo systemctl enable docker",
			"sudo systemctl start docker",
		}, nil
	case "mysql":
		return mysqlDockerCommands(meta), nil
	case "phpmyadmin":
		return phpMyAdminCommands(), nil
	case "ftp":
		return sftpCommands(), nil
	case "quota":
		return diskQuotaCommands(meta), nil
	case "daemon":
		return daemonAgentCommands(agentToken, nodeID, relayURL, ""), nil
	case "images":
		return gameBuildCommands(images)
	default:
		return nil, fmt.Errorf("unknown setup component: %s", component)
	}
}

func mysqlDockerCommands(meta map[string]any) []string {
	instances := defaultMySQLInstances()
	if raw, ok := meta["mysql_instances"]; ok {
		if arr, ok := raw.([]any); ok && len(arr) > 0 {
			instances = arr
		}
	}
	cmds := []string{
		"sudo systemctl start docker || true",
		"sudo mkdir -p /opt/vortanix",
		`VTX_MYSQL_JSON=""`,
	}
	for _, it := range instances {
		m, ok := it.(map[string]any)
		if !ok {
			continue
		}
		container := fmt.Sprint(m["container"])
		port := intFromJSONNumber(m["port"])
		engine := strings.ToLower(fmt.Sprint(m["engine"]))
		version := fmt.Sprint(m["version"])
		key := fmt.Sprint(m["key"])
		if container == "" || port <= 0 {
			continue
		}
		if key == "" || key == "<nil>" {
			key = container
		}
		image := "mysql:8.0"
		if engine == "mariadb" {
			if version != "" {
				image = "mariadb:" + version
			} else {
				image = "mariadb:10.11"
			}
		} else if version != "" {
			image = "mysql:" + version
		}
		image = mirrored(image)
		// Прежний пароль root вычислялся из имени контейнера и порта, то есть
		// был известен любому, кто видит открытый 3306. А порт открыт наружу
		// строкой ufw ниже. Теперь пароль случайный и лежит на самой ноде;
		// файл создаётся один раз, поэтому повторная установка компонента
		// ничего не ломает.
		oldPW := fmt.Sprintf("vtx_%s_%d", strings.ReplaceAll(container, "-", "_"), port)
		pwFile := "/opt/vortanix/" + container + ".rootpw"
		vol := "/opt/vortanix/" + container
		cmds = append(cmds,
			fmt.Sprintf("sudo mkdir -p %s", vol),
			fmt.Sprintf(`if ! sudo test -s %s; then sudo sh -c 'umask 077; { openssl rand -hex 24 2>/dev/null || tr -dc "a-f0-9" </dev/urandom | head -c 48; } > %s'; fi`,
				pwFile, pwFile),
			fmt.Sprintf("sudo chmod 600 %s", pwFile),
			fmt.Sprintf(`VTX_MYSQL_PW="$(sudo cat %s)"`, pwFile),
			// MYSQL_ROOT_HOST=localhost — удалённой учётки root не заводим
			// вовсе: агент ходит в базу через docker exec, снаружи root не
			// нужен никому.
			fmt.Sprintf(`if ! sudo docker ps -a --format '{{.Names}}' | grep -qx %q; then sudo docker run -d --name %s --restart unless-stopped -e MYSQL_ROOT_PASSWORD="$VTX_MYSQL_PW" -e MYSQL_ROOT_HOST=localhost -e MYSQL_DATABASE=system -p %d:3306 -v %s:/var/lib/mysql %s; fi`,
				container, container, port, vol, image),
			// Ноды, поднятые раньше, уже живут со старым вычисляемым паролем —
			// меняем его на месте, пока старый ещё работает, и заодно сносим
			// удалённую учётку root, если образ её всё-таки завёл.
			fmt.Sprintf(`if sudo docker ps -a --format '{{.Names}}' | grep -qx %q; then
  if sudo docker exec -e MYSQL_PWD=%q %s mysql -uroot -e 'SELECT 1' >/dev/null 2>&1; then
    sudo docker exec -e MYSQL_PWD=%q %s mysql -uroot -e "ALTER USER IF EXISTS 'root'@'localhost' IDENTIFIED BY '$VTX_MYSQL_PW'; ALTER USER IF EXISTS 'root'@'%%' IDENTIFIED BY '$VTX_MYSQL_PW'; FLUSH PRIVILEGES;" || true
  fi
  sudo docker exec -e MYSQL_PWD="$VTX_MYSQL_PW" %s mysql -uroot -e "DROP USER IF EXISTS 'root'@'%%'; FLUSH PRIVILEGES;" >/dev/null 2>&1 || true
fi`, container, oldPW, container, oldPW, container, container),
			// Порт наружу оставляем: клиенты подключаются к своим базам из
			// игровых серверов и внешних инструментов. Опасен был не порт, а
			// предсказуемый root — его больше нет ни снаружи, ни по формуле.
			fmt.Sprintf("sudo ufw allow %d/tcp >/dev/null 2>&1 || true", port),
			fmt.Sprintf(`VTX_MYSQL_JSON="$VTX_MYSQL_JSON${VTX_MYSQL_JSON:+,}{\"key\":\"%s\",\"container\":\"%s\",\"port\":%d,\"root_password\":\"$VTX_MYSQL_PW\"}"`,
				key, container, port),
		)
	}
	// Пароли уезжают в файл, который читает агент. Раньше сюда писался пустой
	// список, и агент всегда откатывался на вычисляемый пароль.
	cmds = append(cmds,
		`printf '{"instances":[%s]}' "$VTX_MYSQL_JSON" | sudo tee /opt/vortanix/mysql-instances.json >/dev/null`,
		"sudo chmod 600 /opt/vortanix/mysql-instances.json",
	)
	return cmds
}

func defaultMySQLInstances() []any {
	return []any{
		map[string]any{"key": "mysql80-3306", "engine": "mysql", "version": "8.0", "port": 3306, "container": "vortanix-mysql80-3306", "enabled": true},
		map[string]any{"key": "mysql57-3307", "engine": "mysql", "version": "5.7", "port": 3307, "container": "vortanix-mysql57-3307", "enabled": true},
	}
}

func phpMyAdminCommands() []string {
	image := mirrored("phpmyadmin/phpmyadmin:latest")
	return []string{
		"sudo systemctl start docker || true",
		"sudo docker rm -f vortanix-phpmyadmin 2>/dev/null || true",
		"sudo mkdir -p /opt/vortanix/phpmyadmin",
		fmt.Sprintf("sudo docker pull %s", image),
		fmt.Sprintf("sudo docker run -d --name vortanix-phpmyadmin --restart unless-stopped --add-host=host.docker.internal:host-gateway -e UPLOAD_LIMIT=256M -p 8081:80 %s", image),
		"sudo ufw allow 8081/tcp || true",
	}
}

func daemonAgentCommands(agentToken, nodeID, relayURL, version string) []string {
	if relayURL == "" {
		relayURL = "ws://127.0.0.1:8082/v1/agent/connect"
	}
	image := agentImage(version)
	return []string{
		"sudo systemctl start docker || true",
		// /opt/vortanix монтируется агенту на запись. Оттуда он читает
		// mysql-instances.json со случайными паролями root — без этого файла он
		// всегда откатывался бы на вычисляемый пароль, — и туда же складывает
		// кэш архивов плагинов и карт (VORTANIX_PLUGIN_CACHE_DIR по умолчанию
		// /opt/vortanix/plugin-cache). Только на чтение монтировать нельзя:
		// кэш перестанет создаваться, и установка плагина будет падать.
		"sudo mkdir -p /var/lib/vortanix/servers /opt/vortanix/plugin-cache",
		// Образ тянем до остановки работающего агента и ошибку не глушим.
		// С «|| true» неудачная загрузка молча перезапускала старую версию —
		// обновление выглядело успешным, а версия на ноде не менялась.
		fmt.Sprintf("sudo docker pull %s", image),
		"sudo docker rm -f vortanix-agent 2>/dev/null || true",
		// --cap-add SYS_ADMIN и /dev нужны для дисковых квот. Ими агент
		// выставляет лимит места из тарифа: setquota обращается к блочному
		// устройству файловой системы через quotactl, а тот требует
		// CAP_SYS_ADMIN, которого в наборе docker по умолчанию нет, и самого
		// устройства, которого нет в /dev контейнера. Без них квоты на ноде
		// работают, но ни одному серверу лимит не проставляется — проверено на
		// боевой ноде, где setquota отвечал «Cannot stat() mounted device
		// /dev/loop0».
		//
		// Полномочий это не добавляет: агенту и так проброшен docker.sock, то
		// есть он может поднять привилегированный контейнер и сделать на хосте
		// всё что угодно.
		fmt.Sprintf("sudo docker run -d --name vortanix-agent --restart unless-stopped --cap-add SYS_ADMIN -e RELAY_URL=%q -e AGENT_TOKEN=%q -e NODE_ID=%q -e VORTANIX_VERSION=%q -e VORTANIX_DATA_DIR=/var/lib/vortanix/servers -v /var/run/docker.sock:/var/run/docker.sock -v /dev:/dev -v /var/lib/vortanix/servers:/var/lib/vortanix/servers -v /opt/vortanix:/opt/vortanix %s",
			relayURL, agentToken, nodeID, versionLabelOf(image, version), image),
	}
}

func agentImage(version string) string {
	image := envOr("AGENT_IMAGE", mirrored("agent:latest"))
	version = strings.TrimSpace(version)
	if version == "" {
		return image
	}
	if idx := strings.LastIndex(image, ":"); idx > strings.LastIndex(image, "/") {
		image = image[:idx]
	}
	return image + ":" + version
}

func versionLabelOf(image, version string) string {
	if version != "" {
		return version
	}
	if idx := strings.LastIndex(image, ":"); idx > strings.LastIndex(image, "/") {
		return image[idx+1:]
	}
	return "dev"
}

func intFromJSONNumber(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case json.Number:
		i, _ := n.Int64()
		return int(i)
	default:
		return 0
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// relayWebsocketURL приводит адрес relay к схеме вебсокета. В окружении он
// задан как https://relay.vortanix.app — для agent'а это негодный адрес: он
// открывает ws-соединение и падает с "malformed ws or wss URL", а нода
// остаётся в состоянии "не установлен" при живом контейнере.
func relayWebsocketURL(raw string) string {
	u := strings.TrimSpace(raw)
	if u == "" {
		u = "ws://127.0.0.1:8082"
	}
	switch {
	case strings.HasPrefix(u, "ws://"), strings.HasPrefix(u, "wss://"):
	case strings.HasPrefix(u, "https://"):
		u = "wss://" + strings.TrimPrefix(u, "https://")
	case strings.HasPrefix(u, "http://"):
		u = "ws://" + strings.TrimPrefix(u, "http://")
	default:
		// Без схемы: локальные адреса без TLS, всё остальное — снаружи.
		if strings.HasPrefix(u, "127.") || strings.HasPrefix(u, "localhost") {
			u = "ws://" + u
		} else {
			u = "wss://" + u
		}
	}
	if !strings.Contains(u, "/v1/agent/connect") {
		u = strings.TrimSuffix(u, "/") + "/v1/agent/connect"
	}
	return u
}

// relayUnusableForNode ловит настройку, при которой агент гарантированно не
// заработает: relay задан петлевым адресом, а нода — отдельная машина.
// Контейнер в таком случае поднимается и вечно стучится сам в себя, а
// установка выглядит успешной.
func relayUnusableForNode(relayURL, sshHost string) bool {
	return isLoopbackAddr(relayURL) && !isLoopbackAddr(sshHost)
}

func isLoopbackAddr(s string) bool {
	s = strings.TrimSpace(strings.ToLower(s))
	for _, scheme := range []string{"wss://", "ws://", "https://", "http://"} {
		s = strings.TrimPrefix(s, scheme)
	}
	return strings.HasPrefix(s, "127.") || strings.HasPrefix(s, "localhost") ||
		strings.HasPrefix(s, "0.0.0.0") || strings.HasPrefix(s, "[::1]")
}

// registryLoginCommands логинит ноду в наш реестр перед шагами, которые тянут
// наши образы. Зеркала публичных образов открыты и так, а агент и игры — нет:
// это наша сборка, и доступ к ней даёт действующая лицензия.
//
// Ключ уходит через stdin, а не аргументом: команды в журнал установки не
// попадают, но в списке процессов ноды аргумент был бы виден.
func registryLoginCommands(component, licenseKey string) []string {
	if licenseKey == "" {
		return nil
	}
	switch component {
	case "daemon", "images":
	default:
		return nil
	}
	host := strings.Trim(strings.TrimSpace(envOr("REGISTRY_HOST", "registry.vortanix.app")), "/")
	if host == "" {
		return nil
	}
	return []string{
		fmt.Sprintf("printf '%%s' %s | sudo docker login %s --username vortanix --password-stdin",
			shellQuote(licenseKey), host),
	}
}

// dockerHubLoginCommands логинит ноду в Docker Hub перед загрузкой образов игр.
// Они лежат там, а не в нашем реестре: 48 образов у нас просто не помещаются.
// Без входа Docker Hub режет анонимные загрузки по IP и отдаёт 429.
func dockerHubLoginCommands(component, user, token string) []string {
	if component != "images" {
		return nil
	}
	if user == "" || token == "" {
		// Настройки пусты — идём анонимно, как раньше. Загрузка сработает,
		// пока не упрётся в лимит Docker Hub по IP.
		user = strings.TrimSpace(os.Getenv("DOCKERHUB_USER"))
		token = strings.TrimSpace(os.Getenv("DOCKERHUB_TOKEN"))
	}
	if user == "" || token == "" {
		return nil
	}
	return []string{
		fmt.Sprintf("printf '%%s' %s | sudo docker login --username %s --password-stdin",
			shellQuote(token), shellQuote(user)),
	}
}

// mirrored переводит публичный образ на наше зеркало. Docker Hub режет
// анонимные загрузки по IP, и установка ноды падала на mysql с 429; свой
// реестр отдаёт эти образы без ключа лицензии, которого у ноды нет.
func mirrored(image string) string {
	host := strings.Trim(strings.TrimSpace(envOr("REGISTRY_HOST", "registry.vortanix.app")), "/")
	if host == "" {
		return image
	}
	name, tag := image, "latest"
	if idx := strings.LastIndex(image, ":"); idx > strings.LastIndex(image, "/") {
		name, tag = image[:idx], image[idx+1:]
	}
	if idx := strings.LastIndex(name, "/"); idx >= 0 {
		name = name[idx+1:]
	}
	return fmt.Sprintf("%s/vortanix/%s:%s", host, name, tag)
}

func shellQuoteScript(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}
