package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type mysqlInstance struct {
	Key          string
	Container    string
	Port         int
	RootPassword string
}

func resolveMySQLInstance(payload map[string]any) (mysqlInstance, error) {
	key := StringFromPayload(payload["instance_key"])
	if key == "" {
		key = StringFromPayload(payload["mysql_instance_key"])
	}
	engine := strings.ToLower(StringFromPayload(payload["mysql_engine"]))

	if instances := payloadMysqlInstances(payload); len(instances) > 0 {
		for _, inst := range instances {
			ik := StringFromPayload(inst["key"])
			if key != "" && ik != key {
				continue
			}
			container := StringFromPayload(inst["container"])
			port := IntFromPayload(inst["port"])
			if container == "" || port <= 0 {
				continue
			}
			rootPW := StringFromPayload(inst["root_password"])
			if rootPW == "" {
				rootPW = defaultMySQLRootPassword(container, port)
			}
			return mysqlInstance{Key: ik, Container: container, Port: port, RootPassword: rootPW}, nil
		}
		if key != "" {
			return mysqlInstance{}, fmt.Errorf("unknown mysql_instance_key: %s", key)
		}
	}

	switch engine {
	case "", "default", "mysql80":
		return mysqlInstance{Container: envOr("VORTANIX_MYSQL80_CONTAINER", "vortanix-mysql80-3306"), Port: 3306, RootPassword: envOr("VORTANIX_MYSQL80_ROOT_PASSWORD", defaultMySQLRootPassword("vortanix-mysql80-3306", 3306))}, nil
	case "mysql57":
		return mysqlInstance{Container: envOr("VORTANIX_MYSQL57_CONTAINER", "vortanix-mysql57-3307"), Port: 3307, RootPassword: envOr("VORTANIX_MYSQL57_ROOT_PASSWORD", defaultMySQLRootPassword("vortanix-mysql57-3307", 3307))}, nil
	case "mariadb":
		return mysqlInstance{Container: envOr("VORTANIX_MARIADB_CONTAINER", "vortanix-mariadb"), Port: 3308, RootPassword: envOr("VORTANIX_MARIADB_ROOT_PASSWORD", envOr("MARIADB_ROOT_PASSWORD", ""))}, nil
	}

	if key == "" {
		key = "mysql80-3306"
	}
	return mysqlInstance{}, fmt.Errorf("mysql instance not configured")
}

func payloadMysqlInstances(payload map[string]any) []map[string]any {
	if raw, ok := payload["mysql_instances"]; ok {
		if arr, ok := raw.([]any); ok {
			out := make([]map[string]any, 0, len(arr))
			for _, item := range arr {
				if m, ok := item.(map[string]any); ok {
					out = append(out, m)
				}
			}
			return out
		}
	}
	path := envOr("VORTANIX_MYSQL_INSTANCES_FILE", "/opt/vortanix/mysql-instances.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var doc struct {
		Instances []map[string]any `json:"instances"`
	}
	if json.Unmarshal(data, &doc) != nil {
		return nil
	}
	return doc.Instances
}

func defaultMySQLRootPassword(container string, port int) string {
	return fmt.Sprintf("vtx_%s_%d", strings.ReplaceAll(container, "-", "_"), port)
}

func envOr(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func mysqlEscapeIdent(s string) string {
	return strings.ReplaceAll(s, "`", "``")
}

func mysqlEscapeString(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

func MySQLCreateDB(ctx context.Context, payload map[string]any) error {
	inst, err := resolveMySQLInstance(payload)
	if err != nil {
		return err
	}
	database := StringFromPayload(payload["database"])
	username := StringFromPayload(payload["username"])
	password := StringFromPayload(payload["password"])
	if database == "" || username == "" || password == "" {
		return fmt.Errorf("database, username and password required")
	}

	dbIdent := mysqlEscapeIdent(database)
	userEsc := mysqlEscapeString(username)
	passEsc := mysqlEscapeString(password)
	sql := fmt.Sprintf(
		"CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"+
			"CREATE USER IF NOT EXISTS '%s'@'%%' IDENTIFIED BY '%s';"+
			"ALTER USER '%s'@'%%' IDENTIFIED BY '%s';"+
			"GRANT ALL PRIVILEGES ON `%s`.* TO '%s'@'%%';"+
			"FLUSH PRIVILEGES;",
		dbIdent, userEsc, passEsc, userEsc, passEsc, dbIdent, userEsc,
	)
	return mysqlExec(ctx, inst, sql)
}

func MySQLDeleteDB(ctx context.Context, payload map[string]any) error {
	inst, err := resolveMySQLInstance(payload)
	if err != nil {
		return err
	}
	database := StringFromPayload(payload["database"])
	username := StringFromPayload(payload["username"])
	if database == "" {
		return fmt.Errorf("database required")
	}
	dbIdent := mysqlEscapeIdent(database)
	userEsc := mysqlEscapeString(username)
	sql := fmt.Sprintf("DROP DATABASE IF EXISTS `%s`;", dbIdent)
	if username != "" {
		sql += fmt.Sprintf("DROP USER IF EXISTS '%s'@'%%';", userEsc)
	}
	sql += "FLUSH PRIVILEGES;"
	return mysqlExec(ctx, inst, sql)
}

func MySQLCreateDatabase(ctx context.Context, payload map[string]any) error {
	inst, err := resolveMySQLInstance(payload)
	if err != nil {
		return err
	}
	database := StringFromPayload(payload["database"])
	if database == "" {
		return fmt.Errorf("database required")
	}
	if err := guardMySQLDatabase(database); err != nil {
		return err
	}
	sql := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;", mysqlEscapeIdent(database))
	return mysqlExec(ctx, inst, sql)
}

func MySQLDeleteDatabase(ctx context.Context, payload map[string]any) error {
	inst, err := resolveMySQLInstance(payload)
	if err != nil {
		return err
	}
	database := StringFromPayload(payload["database"])
	if database == "" {
		return fmt.Errorf("database required")
	}
	if err := guardMySQLDatabase(database); err != nil {
		return err
	}
	sql := fmt.Sprintf("DROP DATABASE IF EXISTS `%s`;", mysqlEscapeIdent(database))
	return mysqlExec(ctx, inst, sql)
}

func MySQLCreateUser(ctx context.Context, payload map[string]any) error {
	inst, err := resolveMySQLInstance(payload)
	if err != nil {
		return err
	}
	username := StringFromPayload(payload["username"])
	password := StringFromPayload(payload["password"])
	database := StringFromPayload(payload["database"])
	if username == "" || password == "" {
		return fmt.Errorf("username and password required")
	}
	if err := guardMySQLUser(username); err != nil {
		return err
	}
	if database != "" {
		if err := guardMySQLDatabase(database); err != nil {
			return err
		}
	}
	userEsc := mysqlEscapeString(username)
	passEsc := mysqlEscapeString(password)
	sql := fmt.Sprintf("CREATE USER IF NOT EXISTS '%s'@'%%' IDENTIFIED BY '%s';ALTER USER '%s'@'%%' IDENTIFIED BY '%s';", userEsc, passEsc, userEsc, passEsc)
	if database != "" {
		sql += fmt.Sprintf("GRANT ALL PRIVILEGES ON `%s`.* TO '%s'@'%%';", mysqlEscapeIdent(database), userEsc)
	}
	sql += "FLUSH PRIVILEGES;"
	return mysqlExec(ctx, inst, sql)
}

func MySQLDeleteUser(ctx context.Context, payload map[string]any) error {
	inst, err := resolveMySQLInstance(payload)
	if err != nil {
		return err
	}
	username := StringFromPayload(payload["username"])
	if username == "" {
		return fmt.Errorf("username required")
	}
	if err := guardMySQLUser(username); err != nil {
		return err
	}
	sql := fmt.Sprintf("DROP USER IF EXISTS '%s'@'%%';FLUSH PRIVILEGES;", mysqlEscapeString(username))
	return mysqlExec(ctx, inst, sql)
}

func MySQLResetUserPassword(ctx context.Context, payload map[string]any) error {
	inst, err := resolveMySQLInstance(payload)
	if err != nil {
		return err
	}
	username := StringFromPayload(payload["username"])
	password := StringFromPayload(payload["password"])
	if username == "" || password == "" {
		return fmt.Errorf("username and password required")
	}
	if err := guardMySQLUser(username); err != nil {
		return err
	}
	sql := fmt.Sprintf("ALTER USER '%s'@'%%' IDENTIFIED BY '%s';FLUSH PRIVILEGES;", mysqlEscapeString(username), mysqlEscapeString(password))
	return mysqlExec(ctx, inst, sql)
}

func MySQLListCatalog(ctx context.Context, payload map[string]any) (map[string]any, error) {
	inst, err := resolveMySQLInstance(payload)
	if err != nil {
		return nil, err
	}
	dbRows, err := mysqlQueryLines(ctx, inst, "SHOW DATABASES;")
	if err != nil {
		return nil, err
	}
	databases := make([]string, 0, len(dbRows))
	for _, db := range dbRows {
		db = strings.TrimSpace(db)
		if db == "" {
			continue
		}
		databases = append(databases, db)
	}
	userRows, err := mysqlQueryLines(ctx, inst, "SELECT User,Host FROM mysql.user ORDER BY User,Host;")
	if err != nil {
		return nil, err
	}
	users := make([]map[string]string, 0, len(userRows))
	for _, line := range userRows {
		parts := strings.SplitN(line, "\t", 2)
		user := strings.TrimSpace(parts[0])
		host := "%"
		if len(parts) > 1 {
			host = strings.TrimSpace(parts[1])
		}
		if user == "" {
			continue
		}
		users = append(users, map[string]string{"username": user, "host": host})
	}
	return map[string]any{
		"instance_key": inst.Key,
		"container":    inst.Container,
		"port":         inst.Port,
		"databases":    databases,
		"users":        users,
	}, nil
}

func MySQLMigrateDB(ctx context.Context, serverID string, payload map[string]any) error {
	sourceKey := StringFromPayload(payload["source_instance_key"])
	if sourceKey == "" {
		sourceKey = StringFromPayload(payload["instance_key"])
	}
	targetKey := StringFromPayload(payload["target_instance_key"])
	if sourceKey == "" || targetKey == "" {
		return fmt.Errorf("source_instance_key and target_instance_key required")
	}
	if sourceKey == targetKey {
		return fmt.Errorf("target must differ from source")
	}

	srcPayload := copyPayload(payload)
	srcPayload["instance_key"] = sourceKey
	dstPayload := copyPayload(payload)
	dstPayload["instance_key"] = targetKey

	src, err := resolveMySQLInstance(srcPayload)
	if err != nil {
		return err
	}
	dst, err := resolveMySQLInstance(dstPayload)
	if err != nil {
		return err
	}

	database := StringFromPayload(payload["database"])
	username := StringFromPayload(payload["username"])
	password := StringFromPayload(payload["password"])
	if database == "" || username == "" || password == "" {
		return fmt.Errorf("database, username and password required")
	}

	dbIdent := mysqlEscapeIdent(database)
	userEsc := mysqlEscapeString(username)
	passEsc := mysqlEscapeString(password)
	createSQL := fmt.Sprintf(
		"CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"+
			"CREATE USER IF NOT EXISTS '%s'@'%%' IDENTIFIED BY '%s';"+
			"ALTER USER '%s'@'%%' IDENTIFIED BY '%s';"+
			"GRANT ALL PRIVILEGES ON `%s`.* TO '%s'@'%%';"+
			"FLUSH PRIVILEGES;",
		dbIdent, userEsc, passEsc, userEsc, passEsc, dbIdent, userEsc,
	)
	if err := mysqlExec(ctx, dst, createSQL); err != nil {
		return err
	}

	tmpPath := fmt.Sprintf("/tmp/vortanix-mysql-migrate-%s.sql", strings.ReplaceAll(serverID, "-", ""))
	dumpScript := fmt.Sprintf(
		"rm -f %s && docker exec -e MYSQL_PWD=%s %s mysqldump -uroot --single-transaction --quick --skip-lock-tables --databases %s > %s",
		shellQuote(tmpPath), shellQuote(src.RootPassword), shellQuote(src.Container), shellQuote(database), shellQuote(tmpPath),
	)
	if out, err := exec.CommandContext(ctx, "sh", "-c", dumpScript).CombinedOutput(); err != nil {
		return fmt.Errorf("mysqldump: %w: %s", err, strings.TrimSpace(string(out)))
	}

	importScript := fmt.Sprintf(
		"docker exec -i -e MYSQL_PWD=%s %s mysql -uroot < %s",
		shellQuote(dst.RootPassword), shellQuote(dst.Container), shellQuote(tmpPath),
	)
	if out, err := exec.CommandContext(ctx, "sh", "-c", importScript).CombinedOutput(); err != nil {
		return fmt.Errorf("mysql import: %w: %s", err, strings.TrimSpace(string(out)))
	}
	_ = exec.CommandContext(ctx, "rm", "-f", tmpPath).Run()

	deleteSource := true
	if v, ok := payload["delete_source"]; ok {
		switch b := v.(type) {
		case bool:
			deleteSource = b
		case string:
			deleteSource = b != "0" && b != "false"
		}
	}
	if deleteSource {
		dropSQL := fmt.Sprintf("DROP DATABASE IF EXISTS `%s`;DROP USER IF EXISTS '%s'@'%%';FLUSH PRIVILEGES;", dbIdent, userEsc)
		_ = mysqlExec(ctx, src, dropSQL)
	}
	return nil
}

func (m mysqlInstance) passwords() []string {
	out := []string{m.RootPassword}
	if legacy := defaultMySQLRootPassword(m.Container, m.Port); legacy != m.RootPassword {
		out = append(out, legacy)
	}
	return out
}

func mysqlRun(ctx context.Context, inst mysqlInstance, args ...string) ([]byte, error) {
	var out []byte
	var err error
	for _, pw := range inst.passwords() {
		full := append([]string{"exec", "-e", "MYSQL_PWD=" + pw, inst.Container}, args...)
		out, err = exec.CommandContext(ctx, "docker", full...).CombinedOutput()
		if err == nil {
			return out, nil
		}
		if !strings.Contains(strings.ToLower(string(out)), "access denied") {
			break
		}
	}
	return out, err
}

func mysqlExec(ctx context.Context, inst mysqlInstance, sql string) error {
	out, err := mysqlRun(ctx, inst, "mysql", "-uroot", "-e", sql)
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func mysqlQueryLines(ctx context.Context, inst mysqlInstance, sql string) ([]string, error) {
	out, err := mysqlRun(ctx, inst, "mysql", "-N", "-B", "-uroot", "-e", sql)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	lines := strings.Split(strings.ReplaceAll(string(out), "\r\n", "\n"), "\n")
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		result = append(result, line)
	}
	return result, nil
}

func copyPayload(src map[string]any) map[string]any {
	dst := make(map[string]any, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

var protectedMySQLDatabases = map[string]bool{
	"mysql":              true,
	"information_schema": true,
	"performance_schema": true,
	"sys":                true,
	"system":             true,
}

var protectedMySQLUsers = map[string]bool{
	"root":             true,
	"mysql.sys":        true,
	"mysql.session":    true,
	"mysql.infoschema": true,
	"debian-sys-maint": true,
	"healthchecker":    true,
}

func guardMySQLDatabase(name string) error {
	if protectedMySQLDatabases[strings.ToLower(strings.TrimSpace(name))] {
		return fmt.Errorf("служебная база %q защищена от изменений", name)
	}
	return nil
}

func guardMySQLUser(name string) error {
	if protectedMySQLUsers[strings.ToLower(strings.TrimSpace(name))] {
		return fmt.Errorf("служебный пользователь %q защищён от изменений", name)
	}
	return nil
}
