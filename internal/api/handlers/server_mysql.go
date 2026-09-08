package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) ServerMysqlInfo(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	serverID := chi.URLParam(r, "id")
	info := h.loadServerMysql(r, claims.TenantID, serverID)
	if result, ok := h.agentCommandForServer(w, r, serverID, "mysql_list_catalog", h.mysqlAgentPayload(r, claims.TenantID, serverID, info)); ok {
		info["catalog"] = filterMysqlCatalog(serverID, result)
	}
	writeJSON(w, http.StatusOK, info)
}

func (h *Handler) ServerMysqlResetPassword(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	serverID := chi.URLParam(r, "id")
	var body struct {
		MysqlInstanceKey string `json:"mysql_instance_key"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	password := randomPassword(16)
	dbName := "srv_" + serverID[:8]
	dbUser := dbName
	instanceKey := strings.TrimSpace(body.MysqlInstanceKey)
	storedKey := instanceKey
	if storedKey == "" {
		storedKey = "default"
	}

	mysqlData := map[string]any{
		"host":               "127.0.0.1",
		"port":               3306,
		"database":           dbName,
		"username":           dbUser,
		"password":           password,
		"mysql_instance_key": storedKey,
	}
	raw, _ := json.Marshal(mysqlData)
	_, err := h.dbOf(r.Context()).Exec(r.Context(), `
		UPDATE core.servers SET config = jsonb_set(COALESCE(config, '{}'::jsonb), '{mysql}', $3::jsonb)
		WHERE id = $1 AND tenant_id = $2
	`, serverID, claims.TenantID, raw)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "mysql create failed")
		return
	}
	if _, ok := h.agentCommandForServer(w, r, serverID, "mysql_create_db", h.mysqlAgentPayload(r, claims.TenantID, serverID, map[string]any{
		"database": dbName, "username": dbUser, "password": password, "instance_key": instanceKey,
	})); !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "mysql": mysqlData})
}

func (h *Handler) ServerMysqlDelete(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	serverID := chi.URLParam(r, "id")
	info := h.loadServerMysql(r, claims.TenantID, serverID)
	if _, ok := h.agentCommandForServer(w, r, serverID, "mysql_delete_db", h.mysqlAgentPayload(r, claims.TenantID, serverID, info)); !ok {
		return
	}
	_, err := h.dbOf(r.Context()).Exec(r.Context(), `
		UPDATE core.servers SET config = config - 'mysql' WHERE id = $1 AND tenant_id = $2
	`, serverID, claims.TenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "mysql delete failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) ServerMysqlMigrate(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	serverID := chi.URLParam(r, "id")
	var body struct {
		TargetMysqlInstanceKey string `json:"target_mysql_instance_key"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	info := h.loadServerMysql(r, claims.TenantID, serverID)
	payload := h.mysqlAgentPayload(r, claims.TenantID, serverID, info)
	payload["target_instance_key"] = body.TargetMysqlInstanceKey
	if srcKey, ok := info["mysql_instance_key"].(string); ok && srcKey != "" {
		payload["source_instance_key"] = srcKey
	}
	if _, ok := h.agentCommandForServer(w, r, serverID, "mysql_migrate_db", payload); !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "status": "migrating"})
}

func (h *Handler) ServerMysqlCreateDatabase(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	serverID := chi.URLParam(r, "id")
	var body struct {
		Database string `json:"database"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if msg := validateMysqlName(serverID, "имя базы", body.Database); msg != "" {
		writeCodedError(w, http.StatusForbidden, "mysql_name_forbidden", msg)
		return
	}
	info := h.loadServerMysql(r, claims.TenantID, serverID)
	payload := h.mysqlAgentPayload(r, claims.TenantID, serverID, info)
	payload["database"] = body.Database
	if _, ok := h.agentCommandForServer(w, r, serverID, "mysql_create_database", payload); !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) ServerMysqlDeleteDatabase(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	serverID := chi.URLParam(r, "id")
	var body struct {
		Database string `json:"database"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if msg := validateMysqlName(serverID, "имя базы", body.Database); msg != "" {
		writeCodedError(w, http.StatusForbidden, "mysql_name_forbidden", msg)
		return
	}
	info := h.loadServerMysql(r, claims.TenantID, serverID)
	payload := h.mysqlAgentPayload(r, claims.TenantID, serverID, info)
	payload["database"] = body.Database
	if _, ok := h.agentCommandForServer(w, r, serverID, "mysql_delete_database", payload); !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) ServerMysqlCreateUser(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	serverID := chi.URLParam(r, "id")
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Database string `json:"database"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	body.Username = strings.TrimSpace(body.Username)
	body.Password = strings.TrimSpace(body.Password)
	body.Database = strings.TrimSpace(body.Database)
	if body.Password == "" {
		writeError(w, http.StatusBadRequest, "password required")
		return
	}
	if msg := validateMysqlName(serverID, "имя пользователя", body.Username); msg != "" {
		writeCodedError(w, http.StatusForbidden, "mysql_name_forbidden", msg)
		return
	}
	if body.Database != "" {
		if msg := validateMysqlName(serverID, "имя базы", body.Database); msg != "" {
			writeCodedError(w, http.StatusForbidden, "mysql_name_forbidden", msg)
			return
		}
	}
	info := h.loadServerMysql(r, claims.TenantID, serverID)
	payload := h.mysqlAgentPayload(r, claims.TenantID, serverID, info)
	payload["username"] = body.Username
	payload["password"] = body.Password
	if body.Database != "" {
		payload["database"] = body.Database
	}
	if _, ok := h.agentCommandForServer(w, r, serverID, "mysql_create_user", payload); !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) ServerMysqlDeleteUser(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	serverID := chi.URLParam(r, "id")
	var body struct {
		Username string `json:"username"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if msg := validateMysqlName(serverID, "имя пользователя", body.Username); msg != "" {
		writeCodedError(w, http.StatusForbidden, "mysql_name_forbidden", msg)
		return
	}
	info := h.loadServerMysql(r, claims.TenantID, serverID)
	payload := h.mysqlAgentPayload(r, claims.TenantID, serverID, info)
	payload["username"] = body.Username
	if _, ok := h.agentCommandForServer(w, r, serverID, "mysql_delete_user", payload); !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) ServerMysqlResetUserPassword(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	serverID := chi.URLParam(r, "id")
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	body.Username = strings.TrimSpace(body.Username)
	body.Password = strings.TrimSpace(body.Password)
	if body.Password == "" {
		writeError(w, http.StatusBadRequest, "password required")
		return
	}
	if msg := validateMysqlName(serverID, "имя пользователя", body.Username); msg != "" {
		writeCodedError(w, http.StatusForbidden, "mysql_name_forbidden", msg)
		return
	}
	info := h.loadServerMysql(r, claims.TenantID, serverID)
	payload := h.mysqlAgentPayload(r, claims.TenantID, serverID, info)
	payload["username"] = body.Username
	payload["password"] = body.Password
	if _, ok := h.agentCommandForServer(w, r, serverID, "mysql_reset_user_password", payload); !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) mysqlAgentPayload(r *http.Request, tenantID, serverID string, base map[string]any) map[string]any {
	payload := map[string]any{}
	for k, v := range base {
		payload[k] = v
	}
	if instances := h.nodeMysqlInstances(r, tenantID, serverID); len(instances) > 0 {
		payload["mysql_instances"] = instances
	}

	key := ""
	for _, field := range []string{"instance_key", "mysql_instance_key"} {
		if v, ok := payload[field].(string); ok {
			if trimmed := strings.TrimSpace(v); trimmed != "" && trimmed != "default" {
				key = trimmed
			}
		}
	}
	payload["instance_key"] = key
	payload["mysql_instance_key"] = key

	return payload
}

func (h *Handler) loadServerMysql(r *http.Request, tenantID, serverID string) map[string]any {
	var configRaw []byte
	var nodeMeta []byte
	_ = h.dbOf(r.Context()).QueryRow(r.Context(), `
		SELECT COALESCE(s.config, '{}'::jsonb), COALESCE(n.meta, '{}'::jsonb)
		FROM core.servers s
		LEFT JOIN core.nodes n ON n.id = s.node_id
		WHERE s.id = $1 AND s.tenant_id = $2
	`, serverID, tenantID).Scan(&configRaw, &nodeMeta)
	config := map[string]any{}
	_ = json.Unmarshal(configRaw, &config)
	nodeMetaMap := map[string]any{}
	_ = json.Unmarshal(nodeMeta, &nodeMetaMap)

	out := map[string]any{}
	if mysql, ok := config["mysql"].(map[string]any); ok {
		out = mysql
		out["mysql_password_decrypted"] = mysql["password"]
	}
	if pma, ok := nodeMetaMap["phpmyadmin_port"]; ok {
		out["phpmyadmin_port"] = pma
	}
	if raw, ok := nodeMetaMap["mysql_instances"]; ok {
		if arr, ok := raw.([]any); ok {
			out["mysql_instances"] = arr
		}
	}
	return out
}

func randomPassword(n int) string {
	b := make([]byte, n/2+1)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)[:n]
}
