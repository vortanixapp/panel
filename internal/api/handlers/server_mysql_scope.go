package handlers

import (
	"fmt"
	"regexp"
	"strings"
)

func mysqlNamespace(serverID string) string {
	id := strings.TrimSpace(serverID)
	if len(id) < 8 {
		return ""
	}
	return "srv_" + strings.ToLower(id[:8])
}

var mysqlIdentifier = regexp.MustCompile(`^[A-Za-z0-9_]{1,64}$`)

func ownsMysqlName(serverID, name string) bool {
	ns := mysqlNamespace(serverID)
	if ns == "" {
		return false
	}
	name = strings.ToLower(strings.TrimSpace(name))
	return name == ns || strings.HasPrefix(name, ns+"_")
}

func validateMysqlName(serverID, kind, name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return kind + ": имя не указано"
	}
	if !mysqlIdentifier.MatchString(name) {
		return fmt.Sprintf("%s: допустимы только латинские буквы, цифры и подчёркивание", kind)
	}
	if !ownsMysqlName(serverID, name) {
		return fmt.Sprintf(
			"%s должно начинаться с %s — это пространство имён вашего сервера. "+
				"Базы и пользователи за его пределами принадлежат другим клиентам.",
			kind, mysqlNamespace(serverID))
	}
	return ""
}

func filterMysqlCatalog(serverID string, catalog map[string]any) map[string]any {
	if catalog == nil {
		return nil
	}
	out := map[string]any{}
	for k, v := range catalog {
		out[k] = v
	}

	if raw, ok := catalog["databases"]; ok {
		out["databases"] = filterOwnedStrings(serverID, raw)
	}
	if raw, ok := catalog["users"]; ok {
		out["users"] = filterOwnedUsers(serverID, raw)
	}
	delete(out, "container")
	return out
}

func filterOwnedStrings(serverID string, raw any) []string {
	out := []string{}
	switch items := raw.(type) {
	case []any:
		for _, item := range items {
			if s, ok := item.(string); ok && ownsMysqlName(serverID, s) {
				out = append(out, s)
			}
		}
	case []string:
		for _, s := range items {
			if ownsMysqlName(serverID, s) {
				out = append(out, s)
			}
		}
	}
	return out
}

func filterOwnedUsers(serverID string, raw any) []map[string]string {
	out := []map[string]string{}
	items, ok := raw.([]any)
	if !ok {
		return out
	}
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		username, _ := m["username"].(string)
		if !ownsMysqlName(serverID, username) {
			continue
		}
		host, _ := m["host"].(string)
		if host == "" {
			host = "%"
		}
		out = append(out, map[string]string{"username": username, "host": host})
	}
	return out
}
