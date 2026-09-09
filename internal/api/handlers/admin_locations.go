package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/vortanixapp/panel/internal/api/jobwake"
	"github.com/vortanixapp/panel/internal/api/paneljwt"
	"github.com/vortanixapp/panel/internal/api/sshclient"
	"github.com/vortanixapp/panel/pkg/secretbox"
)

var setupOrder = []string{"packages", "docker", "mysql", "phpmyadmin", "ftp", "quota", "daemon", "images"}

var dbSetupComponents = map[string]bool{
	"packages": true, "docker": true, "mysql": true, "ftp": true, "quota": true, "daemon": true,
}

type locationRow struct {
	ID                string
	Name              string
	FQDN              string
	AgentToken        string
	Country           *string
	City              *string
	Region            *string
	Description       *string
	IPAddress         *string
	SSHHost           *string
	SSHUser           *string
	SSHPort           int
	SSHPassword       *string
	SortOrder         int
	IsActive          bool
	Active            bool
	DockerImages      []byte
	Meta              []byte
	MaintenanceMode   bool
	MaintenanceReason string
	MaintenanceUntil  *time.Time
}

func (h *Handler) adminLocationContext(w http.ResponseWriter, r *http.Request) (*paneljwt.Claims, string, *locationRow, bool) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return nil, "", nil, false
	}
	id := chi.URLParam(r, "id")
	loc, err := h.loadLocationRow(r, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "location not found")
			return nil, "", nil, false
		}
		writeError(w, http.StatusInternalServerError, "database error")
		return nil, "", nil, false
	}
	return claims, id, loc, true
}

func (h *Handler) loadLocationRow(r *http.Request, id string) (*locationRow, error) {
	var loc locationRow
	var sshPassEnc *string
	err := h.readerOf(r.Context()).QueryRow(r.Context(), `
		SELECT id::text, name, fqdn, COALESCE(agent_token, ''),
			country, city, region, description, ip_address,
			ssh_host, ssh_user, COALESCE(ssh_port, 22),
			ssh_password_enc,
			COALESCE(sort_order, 0),
			COALESCE(is_active, true), COALESCE(active, true),
			COALESCE(docker_images, '[]'::jsonb), COALESCE(meta, '{}'::jsonb),
			COALESCE(maintenance_mode, false), COALESCE(maintenance_reason, ''),
			maintenance_until
		FROM core.nodes WHERE id = $1
	`, id).Scan(
		&loc.ID, &loc.Name, &loc.FQDN, &loc.AgentToken,
		&loc.Country, &loc.City, &loc.Region, &loc.Description, &loc.IPAddress,
		&loc.SSHHost, &loc.SSHUser, &loc.SSHPort,
		&sshPassEnc,
		&loc.SortOrder, &loc.IsActive, &loc.Active,
		&loc.DockerImages, &loc.Meta,
		&loc.MaintenanceMode, &loc.MaintenanceReason, &loc.MaintenanceUntil,
	)
	if err != nil {
		return nil, err
	}
	if sshPassEnc != nil {
		plain := h.secrets.MustDecrypt(*sshPassEnc)
		loc.SSHPassword = &plain
	}
	loc.AgentToken = h.openAgentToken(loc.AgentToken)
	return &loc, nil
}

func (h *Handler) sshPasswordForStorage(newPassword any, current *string, meta map[string]any) (*string, error) {
	pass := ""
	if p := locStrPtr(newPassword); p != nil {
		pass = *p
	}
	if pass == "" && current != nil {
		pass = strings.TrimSpace(*current)
	}
	if pass == "" {
		pass = metaString(meta, "ssh_password")
	}
	delete(meta, "ssh_password")
	if pass == "" {
		return nil, nil
	}
	sealed, err := h.secrets.Encrypt(pass)
	if err != nil {
		return nil, err
	}
	return &sealed, nil
}

func parseMetaMap(raw []byte) map[string]any {
	out := map[string]any{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &out)
	}
	if out == nil {
		out = map[string]any{}
	}
	return out
}

func metaString(meta map[string]any, key string) string {
	if v, ok := meta[key]; ok && v != nil {
		return fmt.Sprint(v)
	}
	return ""
}

func metaInt(meta map[string]any, key string, def int) int {
	if v, ok := meta[key]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		}
	}
	return def
}

func metaStringSlice(meta map[string]any, key string) []string {
	v, ok := meta[key]
	if !ok || v == nil {
		return nil
	}
	switch arr := v.(type) {
	case []any:
		out := make([]string, 0, len(arr))
		for _, item := range arr {
			s := strings.TrimSpace(fmt.Sprint(item))
			if s != "" {
				out = append(out, s)
			}
		}
		return out
	case []string:
		return arr
	default:
		return nil
	}
}

func locationCode(loc *locationRow, meta map[string]any) string {
	if c := metaString(meta, "code"); c != "" {
		return c
	}
	return loc.FQDN
}

func locationMapFromRow(loc *locationRow, includeSecrets bool) map[string]any {
	meta := parseMetaMap(loc.Meta)
	code := locationCode(loc, meta)
	ipPool := metaStringSlice(meta, "ip_pool")
	mysqlInstances := meta["mysql_instances"]
	if mysqlInstances == nil {
		mysqlInstances = []any{}
	}
	var dockerImages any
	if len(loc.DockerImages) > 0 {
		_ = json.Unmarshal(loc.DockerImages, &dockerImages)
	}
	if dockerImages == nil {
		dockerImages = []any{}
	}

	out := map[string]any{
		"id":                  loc.ID,
		"name":                loc.Name,
		"code":                code,
		"region":              loc.Region,
		"city":                loc.City,
		"country":             loc.Country,
		"description":         loc.Description,
		"ip_address":          loc.IPAddress,
		"ip_pool":             ipPool,
		"ssh_host":            loc.SSHHost,
		"ssh_user":            loc.SSHUser,
		"ssh_port":            loc.SSHPort,
		"sort_order":          loc.SortOrder,
		"is_active":           loc.IsActive,
		"mysql_host":          metaString(meta, "mysql_host"),
		"mysql_port":          metaInt(meta, "mysql_port", 3306),
		"mysql_root_username": metaString(meta, "mysql_root_username"),
		"phpmyadmin_port":     metaInt(meta, "phpmyadmin_port", 0),
		"mysql_instances":     mysqlInstances,
		"docker_images":       dockerImages,
		"maintenance_mode":    loc.MaintenanceMode,
		"maintenance_reason":  loc.MaintenanceReason,
		"maintenance_until":   timeOrNil(loc.MaintenanceUntil),
	}
	if includeSecrets {
		sshPass := metaString(meta, "ssh_password")
		if sshPass == "" && loc.SSHPassword != nil {
			sshPass = *loc.SSHPassword
		}
		out["ssh_password"] = sshPass
		out["mysql_root_password_decrypted"] = metaString(meta, "mysql_root_password")
		out["node_id"] = loc.ID
		out["agent_token"] = loc.AgentToken
	}
	return out
}

func (h *Handler) ListAdminLocations(w http.ResponseWriter, r *http.Request) {
	_, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	rows, err := h.readerOf(r.Context()).Query(r.Context(), `
		SELECT n.id::text, n.name, COALESCE(n.country, ''),
			COALESCE(n.is_active, n.active, true),
			COALESCE(n.meta->>'code', n.fqdn, ''),
			n.fqdn, n.status, n.meta, n.last_seen_at, n.created_at,
			COALESCE(n.ip_address, ''),
			COALESCE(d.status, 'unknown'), d.last_seen_at,
			COALESCE(n.ssh_host, ''), COALESCE(n.ssh_user, ''),
			n.ssh_password_enc,
			(SELECT COUNT(*)::int FROM core.servers s WHERE s.node_id = n.id),
			(SELECT COUNT(*)::int FROM core.tariffs t WHERE t.node_id = n.id),
			COALESCE(n.maintenance_mode, false), COALESCE(n.maintenance_reason, ''),
			n.maintenance_until
		FROM core.nodes n
		LEFT JOIN core.node_daemons d ON d.node_id = n.id
		ORDER BY COALESCE(n.sort_order, 0), n.name
	`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()

	list := []map[string]any{}
	for rows.Next() {
		var id, name, country, code, fqdn, nodeStatus string
		var isActive bool
		var serversCount, tariffsCount int
		var meta []byte
		var lastSeen *time.Time
		var createdAt time.Time
		var daemonStatus string
		var daemonLastSeen *time.Time
		var sshHost, sshUser, ipAddress string
		var sshPassEnc *string
		var maintenance bool
		var maintenanceReason string
		var maintenanceUntil *time.Time
		if rows.Scan(&id, &name, &country, &isActive, &code, &fqdn, &nodeStatus, &meta, &lastSeen, &createdAt,
			&ipAddress, &daemonStatus, &daemonLastSeen, &sshHost, &sshUser, &sshPassEnc, &serversCount, &tariffsCount,
			&maintenance, &maintenanceReason, &maintenanceUntil) != nil {
			continue
		}
		metaMap := parseMetaMap(meta)
		sshConfigured := sshHost != "" && sshUser != "" &&
			(sshPassEnc != nil && *sshPassEnc != "" || metaString(metaMap, "ssh_password") != "")
		containerRunning, containerKnown := agentContainerRunning(metaMap)
		isOnline := agentDaemonOnline(daemonStatus, daemonLastSeen)
		agentStatus := resolveAgentStatus(isOnline, containerKnown, containerRunning, daemonStatus)
		item := map[string]any{
			"id": id, "node_id": id, "name": name, "country": country, "is_active": isActive,
			"code": code, "fqdn": fqdn,
			"status": agentStatus, "node_status": nodeStatus,
			"daemon_status": daemonStatus, "agent_status": agentStatus, "is_online": isOnline,
			"ssh_host": sshHost, "ssh_user": sshUser, "ssh_configured": sshConfigured,
			"ip_address":    ipAddress,
			"meta":          metaMap,
			"servers_count": serversCount, "tariffs_count": tariffsCount,
			"created_at":         createdAt,
			"maintenance_mode":   maintenance,
			"maintenance_reason": maintenanceReason,
			"maintenance_until":  timeOrNil(maintenanceUntil),
		}
		if lastSeen != nil {
			item["last_seen_at"] = lastSeen
		}
		if daemonLastSeen != nil {
			item["daemon_last_seen_at"] = daemonLastSeen
		}
		list = append(list, item)
	}
	writeJSON(w, http.StatusOK, map[string]any{"locations": list, "nodes": list})
}

func (h *Handler) CreateAdminLocation(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	code := strings.TrimSpace(fmt.Sprint(body["code"]))
	if code == "" {
		code = strings.TrimSpace(fmt.Sprint(body["fqdn"]))
	}
	name := strings.TrimSpace(fmt.Sprint(body["name"]))
	if code == "" || name == "" {
		writeError(w, http.StatusBadRequest, "code and name are required")
		return
	}

	fqdn := code
	if ip := strings.TrimSpace(fmt.Sprint(body["ip_address"])); ip != "" {
		fqdn = ip
	} else if ssh := strings.TrimSpace(fmt.Sprint(body["ssh_host"])); ssh != "" {
		fqdn = ssh
	}

	token, err := generateAgentToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "token generation failed")
		return
	}

	meta := map[string]any{
		"code": code,
	}
	if pool := parseIpPoolInput(body["ip_pool"]); len(pool) > 0 {
		meta["ip_pool"] = pool
	}
	if mi := parseMysqlInstancesInput(body["mysql_instances"]); mi != nil {
		meta["mysql_instances"] = mi
	}
	if di := parseDockerImagesInput(body["docker_images"]); di != nil {
		meta["docker_images"] = di
	}
	metaJSON, _ := json.Marshal(meta)

	country := locStrPtr(body["country"])
	city := locStrPtr(body["city"])
	region := locStrPtr(body["region"])
	description := locStrPtr(body["description"])
	ipAddress := locStrPtr(body["ip_address"])
	sshHost := locStrPtr(body["ssh_host"])
	sshUser := locStrPtr(body["ssh_user"])
	sshPassEnc, err := h.sshPasswordForStorage(body["ssh_password"], nil, meta)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to encrypt ssh password")
		return
	}
	metaJSON, _ = json.Marshal(meta)
	sshPort := locIntFromAny(body["ssh_port"], 22)
	sortOrder := locIntFromAny(body["sort_order"], 0)
	isActive := locBoolFromAny(body["is_active"], true)

	dockerImagesJSON := []byte("[]")
	if di := parseDockerImagesInput(body["docker_images"]); di != nil {
		dockerImagesJSON, _ = json.Marshal(di)
	}

	tokenStored, tokenHash, err := h.sealAgentToken(token)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "token encryption failed")
		return
	}

	var id string
	err = h.dbOf(r.Context()).QueryRow(r.Context(), `
		INSERT INTO core.nodes ( name, fqdn, agent_token, agent_token_hash, country, city, region, description,
			ip_address, ssh_host, ssh_user, ssh_port, ssh_password_enc,
			sort_order, is_active, active, docker_images, meta
		) VALUES ($1,$2,$3,$17,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$14,$15,$16::jsonb)
		RETURNING id::text
	`, name, fqdn, tokenStored, country, city, region, description,
		ipAddress, sshHost, sshUser, sshPort, sshPassEnc,
		sortOrder, isActive, dockerImagesJSON, metaJSON, tokenHash,
	).Scan(&id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create location")
		return
	}
	_ = h.cache.InvalidateTenantNodes(r.Context())
	audit(r.Context(), h.dbOf(r.Context()), claims.UserID, "location.create", "node:"+id, map[string]any{"name": name, "code": code})
	writeJSON(w, http.StatusCreated, map[string]any{
		"id": id, "ok": true, "agent_token": token, "name": name, "fqdn": fqdn, "status": "offline",
	})
}

func (h *Handler) GetAdminLocation(w http.ResponseWriter, r *http.Request) {
	_, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	loc, err := h.loadLocationRow(r, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "location not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}

	serverMetrics := h.loadServerMetrics(r, id)
	cpuMetrics, ramMetrics := h.loadChartMetrics(r, id)
	serviceStatuses := buildServiceStatuses(loc)
	daemon := h.loadDaemonInfo(r, id)
	metricsStale := !h.nodeMetricsFresh(r, id)
	if metricsStale && hasSSHConfigured(loc, parseMetaMap(loc.Meta)) {
		h.enqueueLocationPull(r, id)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"location":        locationMapFromRow(loc, true),
		"serverMetrics":   serverMetrics,
		"serviceStatuses": serviceStatuses,
		"metrics_stale":   metricsStale,
		"sync_pending":    metricsStale,
		"metrics": map[string]any{
			"cpu_usage": cpuMetrics,
			"ram_usage": ramMetrics,
		},
		"daemon": daemon,
	})
}

func (h *Handler) GetAdminLocationEdit(w http.ResponseWriter, r *http.Request) {
	_, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	loc, err := h.loadLocationRow(r, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "location not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"location": locationMapFromRow(loc, false)})
}

func (h *Handler) PutAdminLocation(w http.ResponseWriter, r *http.Request) {
	h.saveAdminLocation(w, r, true)
}

func (h *Handler) PatchAdminLocation(w http.ResponseWriter, r *http.Request) {
	h.saveAdminLocation(w, r, false)
}

func (h *Handler) saveAdminLocation(w http.ResponseWriter, r *http.Request, full bool) {
	_, id, loc, ok := h.adminLocationContext(w, r)
	if !ok {
		return
	}
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	meta := parseMetaMap(loc.Meta)
	if full {
		if v, ok := body["code"]; ok {
			meta["code"] = strings.TrimSpace(fmt.Sprint(v))
		}
		if pool := parseIpPoolInput(body["ip_pool"]); body["ip_pool"] != nil {
			meta["ip_pool"] = pool
		}
		if mi := parseMysqlInstancesInput(body["mysql_instances"]); body["mysql_instances"] != nil {
			meta["mysql_instances"] = mi
		}
	} else {
		if v, ok := body["code"]; ok {
			meta["code"] = strings.TrimSpace(fmt.Sprint(v))
		}
		if _, ok := body["ip_pool"]; ok {
			meta["ip_pool"] = parseIpPoolInput(body["ip_pool"])
		}
		if _, ok := body["mysql_instances"]; ok {
			meta["mysql_instances"] = parseMysqlInstancesInput(body["mysql_instances"])
		}
	}
	if v, ok := body["phpmyadmin_port"]; ok {
		meta["phpmyadmin_port"] = locIntFromAny(v, 8081)
	}
	if di := parseDockerImagesInput(body["docker_images"]); body["docker_images"] != nil {
		loc.DockerImages, _ = json.Marshal(di)
	} else if full && body["docker_images"] == nil {
	}

	name := loc.Name
	if v, ok := body["name"]; ok && strings.TrimSpace(fmt.Sprint(v)) != "" {
		name = strings.TrimSpace(fmt.Sprint(v))
	} else if full {
		if v := strings.TrimSpace(fmt.Sprint(body["name"])); v != "" {
			name = v
		}
	}

	country := loc.Country
	city := loc.City
	region := loc.Region
	description := loc.Description
	ipAddress := loc.IPAddress
	sshHost := loc.SSHHost
	sshUser := loc.SSHUser
	sshPort := loc.SSHPort
	sortOrder := loc.SortOrder
	isActive := loc.IsActive

	if _, ok := body["country"]; ok || full {
		country = locStrPtr(body["country"])
	}
	if _, ok := body["city"]; ok || full {
		city = locStrPtr(body["city"])
	}
	if _, ok := body["region"]; ok || full {
		region = locStrPtr(body["region"])
	}
	if _, ok := body["description"]; ok || full {
		description = locStrPtr(body["description"])
	}
	if _, ok := body["ip_address"]; ok || full {
		ipAddress = locStrPtr(body["ip_address"])
	}
	if _, ok := body["ssh_host"]; ok || full {
		sshHost = locStrPtr(body["ssh_host"])
	}
	if _, ok := body["ssh_user"]; ok || full {
		sshUser = locStrPtr(body["ssh_user"])
	}
	if _, ok := body["ssh_port"]; ok || full {
		sshPort = locIntFromAny(body["ssh_port"], 22)
	}
	if _, ok := body["sort_order"]; ok || full {
		sortOrder = locIntFromAny(body["sort_order"], 0)
	}
	if _, ok := body["is_active"]; ok || full {
		isActive = locBoolFromAny(body["is_active"], true)
	}

	sshPassEnc, err := h.sshPasswordForStorage(body["ssh_password"], loc.SSHPassword, meta)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to encrypt ssh password")
		return
	}

	metaJSON, _ := json.Marshal(meta)
	_, err = h.dbOf(r.Context()).Exec(r.Context(), `
		UPDATE core.nodes SET
			name = $2, country = $3, city = $4, region = $5, description = $6,
			ip_address = $7, ssh_host = $8, ssh_user = $9, ssh_port = $10,
			ssh_password_enc = COALESCE($11, ssh_password_enc),
			sort_order = $12, is_active = $13, active = $13,
			docker_images = COALESCE($14, docker_images), meta = $15::jsonb
		WHERE id = $1
	`, id, name, country, city, region, description,
		ipAddress, sshHost, sshUser, sshPort, sshPassEnc,
		sortOrder, isActive, nullableJSONBytes(loc.DockerImages), metaJSON)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "update failed")
		return
	}
	_ = h.cache.InvalidateTenantNodes(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "status": "updated"})
}

func (h *Handler) ToggleAdminLocation(w http.ResponseWriter, r *http.Request) {
	_, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	tag, err := h.dbOf(r.Context()).Exec(r.Context(), `
		UPDATE core.nodes SET is_active = NOT COALESCE(is_active, true), active = NOT COALESCE(active, true)
		WHERE id = $1
	`, id)
	if err != nil || tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "location not found")
		return
	}
	_ = h.cache.InvalidateTenantNodes(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) DeleteAdminLocation(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	ctx := r.Context()

	var live int
	if err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT count(*) FROM core.servers WHERE node_id = $1::uuid
	`, id).Scan(&live); err != nil {
		writeError(w, http.StatusInternalServerError, "не удалось проверить серверы локации")
		return
	}
	if live > 0 {
		writeError(w, http.StatusConflict,
			fmt.Sprintf("на локации %d сервер(ов): перенесите их на другую ноду или удалите, прежде чем удалять локацию", live))
		return
	}

	tag, err := h.dbOf(ctx).Exec(ctx, `DELETE FROM core.nodes WHERE id = $1`, id)
	if err != nil || tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "location not found")
		return
	}
	_ = h.cache.InvalidateTenantNodes(r.Context())
	audit(r.Context(), h.dbOf(r.Context()), claims.UserID, "location.delete", "node:"+id, nil)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "message": "Локация удалена успешно.", "status": "deleted"})
}

func (h *Handler) GetAdminLocationInstallScript(w http.ResponseWriter, r *http.Request) {
	_, id, loc, ok := h.adminLocationContext(w, r)
	if !ok {
		return
	}
	token := strings.TrimSpace(loc.AgentToken)
	if token == "" {
		var err error
		token, err = generateAgentToken()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "token generation failed")
			return
		}
		stored, hash, sealErr := h.sealAgentToken(token)
		if sealErr != nil {
			writeError(w, http.StatusInternalServerError, "token encryption failed")
			return
		}
		_, _ = h.dbOf(r.Context()).Exec(r.Context(), `
			UPDATE core.nodes SET agent_token = $2, agent_token_hash = $3
			WHERE id = $1
		`, id, stored, hash)
		loc.AgentToken = token
	}
	relayURL := strings.TrimSuffix(strings.TrimSpace(envOr("RELAY_PUBLIC_URL", "")), "/")
	if relayURL == "" {
		writeError(w, http.StatusPreconditionFailed,
			"не задан RELAY_PUBLIC_URL — внешний адрес relay, по которому нода до него достучится. "+
				"Без него команда установки увела бы ноду в никуда")
		return
	}
	if !strings.HasPrefix(relayURL, "ws") {
		relayURL = "wss://" + strings.TrimPrefix(strings.TrimPrefix(relayURL, "https://"), "http://")
	}
	connectURL := relayURL + "/v1/agent/connect"
	envFile := "RELAY_URL=" + connectURL + "\nAGENT_TOKEN=" + token + "\nNODE_ID=" + id + "\n"
	script := `#!/bin/sh
set -eu
mkdir -p /var/lib/vortanix/servers
cat > /root/vortanix-agent.env <<'EOF'
` + envFile + `EOF
docker run -d --name vortanix-agent --restart unless-stopped \
  --env-file /root/vortanix-agent.env \
  -e VORTANIX_DATA_DIR=/var/lib/vortanix/servers \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v /var/lib/vortanix/servers:/var/lib/vortanix/servers \
  ` + agentImageRef() + `
docker logs vortanix-agent --tail 20`
	writeJSON(w, http.StatusOK, map[string]string{
		"node_id":     id,
		"location_id": id,
		"fqdn":        loc.FQDN,
		"agent_token": token,
		"relay_url":   connectURL,
		"env_file":    envFile,
		"script":      script,
	})
}

func (h *Handler) sealAgentToken(token string) (stored, hash string, err error) {
	hash = secretbox.TokenHash(token)
	stored, err = h.secrets.Encrypt(token)
	if err != nil {
		return "", "", err
	}
	return stored, hash, nil
}

func (h *Handler) openAgentToken(stored string) string {
	return h.secrets.MustDecrypt(stored)
}

func (h *Handler) RegenerateAdminLocationAgentToken(w http.ResponseWriter, r *http.Request) {
	claims, id, _, ok := h.adminLocationContext(w, r)
	if !ok {
		return
	}
	token, err := generateAgentToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "token generation failed")
		return
	}
	stored, hash, err := h.sealAgentToken(token)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "token encryption failed")
		return
	}
	_, err = h.dbOf(r.Context()).Exec(r.Context(), `
		UPDATE core.nodes SET agent_token = $2, agent_token_hash = $3
		WHERE id = $1
	`, id, stored, hash)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update token")
		return
	}
	_ = h.cache.InvalidateTenantNodes(r.Context())
	audit(r.Context(), h.dbOf(r.Context()), claims.UserID, "location.agent_token_regenerate", "node:"+id, nil)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "node_id": id, "agent_token": token})
}

func (h *Handler) TestAdminLocationSSH(w http.ResponseWriter, r *http.Request) {
	_, _, loc, ok := h.adminLocationContext(w, r)
	if !ok {
		return
	}
	meta := parseMetaMap(loc.Meta)
	cfg, err := locationSSHConfig(loc, meta, "")
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	out, runErr := sshclient.RunCapture(cfg, "echo OK && uname -sr")
	if runErr != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"ok": false, "error": runErr.Error(), "output": strings.TrimSpace(out),
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok": true, "output": strings.TrimSpace(out), "host": cfg.Host, "user": cfg.User,
	})
}

func (h *Handler) TestAdminLocationSSHBody(w http.ResponseWriter, r *http.Request) {
	_, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	host := strings.TrimSpace(fmt.Sprint(body["ssh_host"]))
	user := strings.TrimSpace(fmt.Sprint(body["ssh_user"]))
	pass := strings.TrimSpace(fmt.Sprint(body["ssh_password"]))
	port := locIntFromAny(body["ssh_port"], 22)
	if host == "" || user == "" || pass == "" {
		writeError(w, http.StatusBadRequest, "ssh_host, ssh_user и ssh_password обязательны")
		return
	}
	cfg := sshclient.Config{Host: host, Port: port, User: user, Password: pass}
	out, runErr := sshclient.RunCapture(cfg, "echo OK && uname -sr")
	if runErr != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"ok": false, "error": runErr.Error(), "output": strings.TrimSpace(out),
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok": true, "output": strings.TrimSpace(out), "host": host, "user": user,
	})
}

func locationSSHConfig(loc *locationRow, meta map[string]any, passwordOverride string) (sshclient.Config, error) {
	host := ""
	if loc.SSHHost != nil {
		host = strings.TrimSpace(*loc.SSHHost)
	}
	user := ""
	if loc.SSHUser != nil {
		user = strings.TrimSpace(*loc.SSHUser)
	}
	pass := strings.TrimSpace(passwordOverride)
	if pass == "" {
		pass = metaString(meta, "ssh_password")
		if pass == "" && loc.SSHPassword != nil {
			pass = *loc.SSHPassword
		}
	}
	if host == "" || user == "" || pass == "" {
		return sshclient.Config{}, fmt.Errorf("SSH host, user и password обязательны")
	}
	port := loc.SSHPort
	if port <= 0 {
		port = 22
	}
	return sshclient.Config{Host: host, Port: port, User: user, Password: pass}, nil
}

func generateAgentToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "agt_" + hex.EncodeToString(b), nil
}

func (h *Handler) GetAdminLocationSetup(w http.ResponseWriter, r *http.Request) {
	_, id, loc, ok := h.adminLocationContext(w, r)
	if !ok {
		return
	}
	meta := parseMetaMap(loc.Meta)
	statuses := h.loadSetupStatuses(r, id, meta)
	daemon := h.loadDaemonInfo(r, id)
	isOnline := false
	if daemon != nil {
		if v, ok := daemon["is_online"].(bool); ok {
			isOnline = v
		}
	}
	relayURL := envOr("RELAY_PUBLIC_URL", "")
	checks := []map[string]any{
		{"key": "ssh", "label": "SSH (host / user / password)", "ok": hasSSHConfigured(loc, meta), "hint": "Укажите SSH в настройках локации — worker подключается для установки пакетов и agent."},
		{"key": "relay", "label": "Связь с нодами (relay)", "ok": relayURL != "", "hint": "Задайте RELAY_PUBLIC_URL — адрес панели, по которому её видит нода, без пути: wss://panel.example.com или ws://203.0.113.10"},
		{"key": "agent", "label": "Vortanix Agent онлайн", "ok": isOnline, "hint": "Запустите шаг «Agent» в установке или sh agent-up.sh на игровой ноде."},
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"location": locationMapFromRow(loc, false),
		"statuses": statuses,
		"checks":   checks,
	})
}

func (h *Handler) GetAdminLocationSetupStatus(w http.ResponseWriter, r *http.Request) {
	_, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	loc, err := h.loadLocationRow(r, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "location not found")
		return
	}
	meta := parseMetaMap(loc.Meta)
	progress, _ := meta["setup_progress"].(map[string]any)
	logText := ""
	completed := false
	component := ""
	if progress != nil {
		logText, _ = progress["log"].(string)
		completed, _ = progress["completed"].(bool)
		component, _ = progress["component"].(string)
	}
	if component != "" {
		st := h.setupComponentStatus(r, id, component, meta)
		if st == "installed" || st == "failed" {
			completed = true
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"log": logText, "completed": completed, "component": component})
}

func (h *Handler) RunAdminLocationSetupStep(w http.ResponseWriter, r *http.Request) {
	_, id, loc, ok := h.adminLocationContext(w, r)
	if !ok {
		return
	}
	step := chi.URLParam(r, "step")
	valid := false
	for _, s := range setupOrder {
		if s == step {
			valid = true
			break
		}
	}
	if !valid {
		writeError(w, http.StatusBadRequest, "unknown setup step")
		return
	}
	meta := parseMetaMap(loc.Meta)
	if !hasSSHConfigured(loc, meta) {
		writeError(w, http.StatusUnprocessableEntity, "Для локации не настроены SSH host/user/password.")
		return
	}
	if errMsg := h.ensureSetupOrder(r, id, step, meta); errMsg != "" {
		writeError(w, http.StatusUnprocessableEntity, errMsg)
		return
	}

	msg := fmt.Sprintf("Начинаем установку %s...", step)
	meta["setup_progress"] = map[string]any{"log": msg, "completed": false, "component": step}
	metaJSON, _ := json.Marshal(meta)
	_, _ = h.dbOf(r.Context()).Exec(r.Context(), `UPDATE core.nodes SET meta = $2::jsonb WHERE id = $1`, id, metaJSON)

	if dbSetupComponents[step] {
		if _, err := h.dbOf(r.Context()).Exec(r.Context(), `
			INSERT INTO core.node_setups ( node_id, component, status)
			VALUES ( $1, $2, 'installing')
			ON CONFLICT (node_id, component) DO UPDATE SET status = 'installing', updated_at = now()
		`, id, step); err != nil {
			writeError(w, http.StatusInternalServerError, "Статус шага не сохраняется: "+err.Error())
			return
		}
	} else {
		statuses := setupStatusesFromMeta(meta)
		statuses[step] = "installing"
		meta["setup_statuses"] = statuses
		metaJSON, _ = json.Marshal(meta)
		_, _ = h.dbOf(r.Context()).Exec(r.Context(), `UPDATE core.nodes SET meta = $2::jsonb WHERE id = $1`, id, metaJSON)
	}

	payload, _ := json.Marshal(map[string]string{"node_id": id, "component": step})
	_, _ = h.dbOf(r.Context()).Exec(r.Context(), `
		INSERT INTO core.jobs ( type, status, payload) VALUES ( 'node_setup', 'pending', $1::jsonb)
	`, payload)
	jobwake.Notify("node_setup")
	writeJSON(w, http.StatusAccepted, map[string]any{"success": true, "message": msg})
}

func (h *Handler) GetAdminLocationDaemon(w http.ResponseWriter, r *http.Request) {
	_, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	agent := h.loadAgentInfo(r, id)
	writeJSON(w, http.StatusOK, map[string]any{"agent": agent, "daemon": agent})
}

func (h *Handler) loadDaemonInfo(r *http.Request, nodeID string) map[string]any {
	return h.loadAgentInfo(r, nodeID)
}

func (h *Handler) InstallAdminLocationDaemon(w http.ResponseWriter, r *http.Request) {
	h.enqueueDaemonJob(w, r, "install", nil)
}

func (h *Handler) RefreshAdminLocationDaemon(w http.ResponseWriter, r *http.Request) {
	h.enqueueDaemonJob(w, r, "refresh", nil)
}

func (h *Handler) RestartAdminLocationDaemon(w http.ResponseWriter, r *http.Request) {
	h.enqueueDaemonJob(w, r, "restart", nil)
}

func (h *Handler) enqueueDaemonJob(w http.ResponseWriter, r *http.Request, action string, params map[string]any) {
	_, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	nodeID := chi.URLParam(r, "id")
	payload, _ := json.Marshal(map[string]any{"node_id": nodeID, "action": action, "params": params})
	_, _ = h.dbOf(r.Context()).Exec(r.Context(), `
		INSERT INTO core.jobs ( type, status, payload) VALUES ( 'daemon_action', 'pending', $1::jsonb)
	`, payload)
	jobwake.Notify("daemon_action")
	writeJSON(w, http.StatusAccepted, map[string]any{"status": action, "success": true})
}

func (h *Handler) loadServerMetrics(r *http.Request, nodeID string) map[string]any {
	types := []string{"os_info", "cpu_model", "ram_total", "disk_total", "disk_used", "disk_available", "uptime"}
	out := map[string]any{}
	for _, t := range types {
		var value float64
		var textValue *string
		var measuredAt time.Time
		err := h.readerOf(r.Context()).QueryRow(r.Context(), `
			SELECT value, text_value, measured_at FROM core.node_metrics
			WHERE node_id = $1 AND metric_type = $2
			ORDER BY measured_at DESC LIMIT 1
		`, nodeID, t).Scan(&value, &textValue, &measuredAt)
		if err != nil {
			continue
		}
		item := map[string]any{"metric_type": t, "value": value, "measured_at": measuredAt.Format(time.RFC3339)}
		if textValue != nil {
			item["text_value"] = *textValue
		}
		out[t] = item
	}
	return out
}

func (h *Handler) loadChartMetrics(r *http.Request, nodeID string) ([]map[string]any, []map[string]any) {
	load := func(metricType string) []map[string]any {
		rows, err := h.readerOf(r.Context()).Query(r.Context(), `
			SELECT value, measured_at FROM core.node_metrics
			WHERE node_id = $1 AND metric_type = $2
			ORDER BY measured_at DESC LIMIT 50
		`, nodeID, metricType)
		if err != nil {
			return nil
		}
		defer rows.Close()
		points := make([]map[string]any, 0)
		for rows.Next() {
			var value float64
			var measuredAt time.Time
			if rows.Scan(&value, &measuredAt) == nil {
				points = append(points, map[string]any{
					"t": measuredAt.Format(time.RFC3339), "v": value, "value": value, "measured_at": measuredAt.Format(time.RFC3339),
				})
			}
		}
		for i, j := 0, len(points)-1; i < j; i, j = i+1, j-1 {
			points[i], points[j] = points[j], points[i]
		}
		return points
	}
	return load("cpu_usage"), load("ram_usage")
}

func buildServiceStatuses(loc *locationRow) map[string]map[string]any {
	services := map[string]string{
		"docker": "Docker", "mysql": "MySQL", "vortanix-sftp": "SFTP", "vortanix-agent": "Vortanix Agent",
	}
	out := map[string]map[string]any{}
	for unit, label := range services {
		out[unit] = map[string]any{"label": label, "state": "unknown", "error": nil}
	}
	meta := parseMetaMap(loc.Meta)
	if loc.SSHHost == nil || *loc.SSHHost == "" || loc.SSHUser == nil || *loc.SSHUser == "" || !hasSSHPassword(loc, meta) {
		for k := range out {
			out[k]["state"] = "unconfigured"
		}
		return out
	}
	raw, ok := meta["service_statuses"].(map[string]any)
	if !ok {
		return out
	}
	for unit, entry := range raw {
		em, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		if _, exists := out[unit]; !exists {
			label := unit
			if l, ok := em["label"].(string); ok && l != "" {
				label = l
			}
			out[unit] = map[string]any{"label": label, "state": "unknown", "error": nil}
		}
		if state, ok := em["state"].(string); ok && state != "" {
			out[unit]["state"] = state
		}
		if label, ok := em["label"].(string); ok && label != "" {
			out[unit]["label"] = label
		}
		if errVal, ok := em["error"]; ok {
			out[unit]["error"] = errVal
		}
	}
	return out
}

func (h *Handler) loadSetupStatuses(r *http.Request, nodeID string, meta map[string]any) map[string]string {
	statuses := setupStatusesFromMeta(meta)
	rows, _ := h.readerOf(r.Context()).Query(r.Context(), `
		SELECT component, status FROM core.node_setups WHERE node_id = $1
	`, nodeID)
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var comp, st string
			if rows.Scan(&comp, &st) == nil {
				statuses[comp] = st
			}
		}
	}
	for _, c := range setupOrder {
		if _, ok := statuses[c]; !ok {
			statuses[c] = "pending"
		}
	}
	return statuses
}

func setupStatusesFromMeta(meta map[string]any) map[string]string {
	out := map[string]string{}
	if raw, ok := meta["setup_statuses"].(map[string]any); ok {
		for k, v := range raw {
			out[k] = fmt.Sprint(v)
		}
	}
	return out
}

func (h *Handler) setupComponentStatus(r *http.Request, nodeID, component string, meta map[string]any) string {
	if dbSetupComponents[component] {
		var st string
		_ = h.readerOf(r.Context()).QueryRow(r.Context(), `
			SELECT status FROM core.node_setups WHERE node_id = $1 AND component = $2
		`, nodeID, component).Scan(&st)
		return st
	}
	statuses := setupStatusesFromMeta(meta)
	return statuses[component]
}

func (h *Handler) ensureSetupOrder(r *http.Request, nodeID, component string, meta map[string]any) string {
	idx := -1
	for i, s := range setupOrder {
		if s == component {
			idx = i
			break
		}
	}
	if idx < 0 {
		return "Unknown setup component."
	}
	statuses := h.loadSetupStatuses(r, nodeID, meta)
	for _, st := range statuses {
		if st == "installing" {
			return "Установка уже выполняется. Дождитесь завершения текущего шага."
		}
	}
	for i := 0; i < idx; i++ {
		req := setupOrder[i]
		if statuses[req] != "installed" {
			return fmt.Sprintf("Нельзя выполнить шаг %q до завершения шага %q.", component, req)
		}
	}
	return ""
}

func hasSSHPassword(loc *locationRow, meta map[string]any) bool {
	if metaString(meta, "ssh_password") != "" {
		return true
	}
	return loc.SSHPassword != nil && *loc.SSHPassword != ""
}

func hasSSHConfigured(loc *locationRow, meta map[string]any) bool {
	return loc.SSHHost != nil && *loc.SSHHost != "" && loc.SSHUser != nil && *loc.SSHUser != "" && hasSSHPassword(loc, meta)
}

var ipRe = regexp.MustCompile(`^\d{1,3}(?:\.\d{1,3}){3}$`)

func parseIpPoolInput(raw any) []string {
	if raw == nil {
		return nil
	}
	switch v := raw.(type) {
	case []any:
		out := []string{}
		for _, item := range v {
			ip := strings.TrimSpace(fmt.Sprint(item))
			if ipRe.MatchString(ip) {
				out = append(out, ip)
			}
		}
		return out
	case []string:
		out := []string{}
		for _, ip := range v {
			ip = strings.TrimSpace(ip)
			if ipRe.MatchString(ip) {
				out = append(out, ip)
			}
		}
		return out
	}
	s := strings.TrimSpace(fmt.Sprint(raw))
	if s == "" {
		return nil
	}
	var decoded []any
	if json.Unmarshal([]byte(s), &decoded) == nil {
		return parseIpPoolInput(decoded)
	}
	parts := regexp.MustCompile(`[\s,;]+`).Split(s, -1)
	out := []string{}
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if ipRe.MatchString(p) {
			out = append(out, p)
		}
	}
	return out
}

func parseMysqlInstancesInput(raw any) []map[string]any {
	if raw == nil {
		return nil
	}
	s := strings.TrimSpace(fmt.Sprint(raw))
	if s == "" {
		return nil
	}
	var decoded []any
	if err := json.Unmarshal([]byte(s), &decoded); err != nil {
		if err2 := json.Unmarshal([]byte(fmt.Sprint(raw)), &decoded); err2 != nil {
			return nil
		}
	}
	out := []map[string]any{}
	for _, item := range decoded {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, m)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func parseDockerImagesInput(raw any) []map[string]any {
	return parseMysqlInstancesInput(raw)
}

func locStrPtr(v any) *string {
	if v == nil {
		return nil
	}
	s := strings.TrimSpace(fmt.Sprint(v))
	if s == "" {
		return nil
	}
	return &s
}

func locIntFromAny(v any, def int) int {
	if v == nil {
		return def
	}
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case json.Number:
		i, _ := n.Int64()
		return int(i)
	default:
		return def
	}
}

func locBoolFromAny(v any, def bool) bool {
	if v == nil {
		return def
	}
	switch b := v.(type) {
	case bool:
		return b
	default:
		return def
	}
}

func nullableJSONBytes(b []byte) any {
	if len(b) == 0 {
		return nil
	}
	return b
}

func locRandomToken(n int) (string, error) {
	b := make([]byte, (n+1)/2)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b)[:n], nil
}
