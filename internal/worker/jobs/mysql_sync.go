package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

const mysqlInstancesReadCmd = `cat /opt/vortanix/mysql-instances.json 2>/dev/null || sudo -n cat /opt/vortanix/mysql-instances.json 2>/dev/null`

func mysqlInstanceText(v any) string {
	if v == nil {
		return ""
	}
	s := strings.TrimSpace(fmt.Sprint(v))
	if s == "<nil>" {
		return ""
	}
	return s
}

func mysqlInstanceName(engine, version string, port int) string {
	label := "MySQL"
	if engine == "mariadb" {
		label = "MariaDB"
	}
	if version != "" {
		label += " " + version
	}
	if port > 0 {
		label += fmt.Sprintf(" · %d", port)
	}
	return label
}

func (r *Runner) syncMySQLInstances(ctx context.Context, q dbExec, nodeID, raw string) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}
	var doc struct {
		Instances []map[string]any `json:"instances"`
	}
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		return 0, fmt.Errorf("файл инстансов MySQL не разобран: %w", err)
	}

	var currentRaw []byte
	if err := q.QueryRow(ctx, `
		SELECT COALESCE(meta->'mysql_instances', '[]'::jsonb) FROM core.nodes WHERE id = $1
	`, nodeID).Scan(&currentRaw); err != nil {
		return 0, err
	}
	var current []map[string]any
	if json.Unmarshal(currentRaw, &current) != nil {
		current = nil
	}

	defaults := map[string]map[string]any{}
	for _, d := range defaultMySQLInstances() {
		if m, ok := d.(map[string]any); ok {
			defaults[mysqlInstanceText(m["key"])] = m
		}
	}

	changed := false
	for _, inst := range doc.Instances {
		key := mysqlInstanceText(inst["key"])
		container := mysqlInstanceText(inst["container"])
		port := intFromJSONNumber(inst["port"])
		password := mysqlInstanceText(inst["root_password"])
		if key == "" {
			key = container
		}
		if key == "" {
			continue
		}

		var target map[string]any
		for _, c := range current {
			if mysqlInstanceText(c["key"]) == key || (container != "" && mysqlInstanceText(c["container"]) == container) {
				target = c
				break
			}
		}
		if target == nil {
			target = map[string]any{"key": key, "enabled": true}
			if d, ok := defaults[key]; ok {
				target["engine"] = d["engine"]
				target["version"] = d["version"]
			}
			current = append(current, target)
			changed = true
		}
		set := func(field string, value any) {
			if fmt.Sprint(target[field]) != fmt.Sprint(value) {
				target[field] = value
				changed = true
			}
		}
		if container != "" {
			set("container", container)
		}
		if port > 0 {
			set("port", port)
		}
		if password != "" {
			set("root_password", password)
		}
		if mysqlInstanceText(target["name"]) == "" {
			set("name", mysqlInstanceName(
				strings.ToLower(mysqlInstanceText(target["engine"])),
				mysqlInstanceText(target["version"]),
				intFromJSONNumber(target["port"]),
			))
		}
	}
	if !changed {
		return len(doc.Instances), nil
	}
	b, _ := json.Marshal(current)
	_, err := q.Exec(ctx, `
		UPDATE core.nodes
		SET meta = jsonb_set(COALESCE(meta, '{}'::jsonb), '{mysql_instances}', $2::jsonb, true)
		WHERE id = $1
	`, nodeID, b)
	return len(doc.Instances), err
}
