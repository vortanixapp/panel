package handlers

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/vortanixapp/panel/pkg/portalloc"
)

type ipPoolAddBody struct {
	Addresses any    `json:"addresses"`
	Label     string `json:"label"`
}

func (h *Handler) AdminNodeIPList(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	nodeID := chi.URLParam(r, "id")

	rows, err := h.readerOf(r.Context()).Query(r.Context(), `
		SELECT p.id::text, host(p.address), p.status, COALESCE(p.label, ''),
		       COALESCE(p.server_id::text, ''), COALESCE(s.name, ''), p.assigned_at
		FROM core.ip_pools p
		LEFT JOIN core.servers s ON s.id = p.server_id
		WHERE p.tenant_id = $1 AND p.node_id = $2
		ORDER BY p.address
	`, claims.TenantID, nodeID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()

	list := []map[string]any{}
	free := 0
	for rows.Next() {
		var id, address, status, label, serverID, serverName string
		var assignedAt *time.Time
		if rows.Scan(&id, &address, &status, &label, &serverID, &serverName, &assignedAt) != nil {
			continue
		}
		if status == "free" {
			free++
		}
		list = append(list, map[string]any{
			"id": id, "address": address, "status": status, "label": label,
			"server_id": nilIfEmpty(serverID), "server_name": serverName,
			"assigned_at": timeOrNil(assignedAt),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"addresses": list, "free": free})
}

func (h *Handler) AdminNodeIPAdd(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	nodeID := chi.URLParam(r, "id")
	var body ipPoolAddBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	ctx := r.Context()

	var exists bool
	if h.dbOf(ctx).QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM core.nodes WHERE id = $1 AND tenant_id = $2)`,
		nodeID, claims.TenantID).Scan(&exists) != nil || !exists {
		writeError(w, http.StatusNotFound, "location not found")
		return
	}

	addresses, invalid := parseAddressList(body.Addresses)
	if len(invalid) > 0 {
		writeError(w, http.StatusBadRequest, "не адреса: "+strings.Join(invalid, ", "))
		return
	}
	if len(addresses) == 0 {
		writeError(w, http.StatusBadRequest, "не указано ни одного адреса")
		return
	}

	added := 0
	for _, addr := range addresses {
		tag, err := h.dbOf(ctx).Exec(ctx, `
			INSERT INTO core.ip_pools (tenant_id, node_id, address, status, label)
			VALUES ($1, $2, $3::inet, 'free', $4)
			ON CONFLICT (node_id, address) DO NOTHING
		`, claims.TenantID, nodeID, addr, strings.TrimSpace(body.Label))
		if err == nil {
			added += int(tag.RowsAffected())
		}
	}
	audit(ctx, h.dbOf(ctx), claims.TenantID, claims.UserID, "node.ip.add", "node:"+nodeID,
		map[string]any{"added": added, "total": len(addresses)})
	writeJSON(w, http.StatusOK, map[string]any{
		"added": added, "skipped": len(addresses) - added,
	})
}

func parseAddressList(raw any) (valid []string, invalid []string) {
	parts := []string{}
	switch v := raw.(type) {
	case string:
		parts = strings.FieldsFunc(v, func(r rune) bool {
			return r == '\n' || r == '\r' || r == ',' || r == ';' || r == ' ' || r == '\t'
		})
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok {
				parts = append(parts, s)
			}
		}
	}
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if net.ParseIP(p) == nil {
			invalid = append(invalid, p)
			continue
		}
		valid = append(valid, p)
	}
	return valid, invalid
}

func (h *Handler) AdminNodeIPDelete(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ipID := chi.URLParam(r, "ipId")
	tag, err := h.dbOf(r.Context()).Exec(r.Context(), `
		DELETE FROM core.ip_pools
		WHERE id = $1 AND tenant_id = $2 AND server_id IS NULL
	`, ipID, claims.TenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusConflict, "адрес выдан серверу — сначала освободите его")
		return
	}
	audit(r.Context(), h.dbOf(r.Context()), claims.TenantID, claims.UserID, "node.ip.delete", "ip:"+ipID, nil)
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) AdminNodeIPImport(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	nodeID := chi.URLParam(r, "id")
	ctx := r.Context()

	var meta []byte
	if h.dbOf(ctx).QueryRow(ctx, `SELECT COALESCE(meta, '{}'::jsonb) FROM core.nodes WHERE id = $1 AND tenant_id = $2`,
		nodeID, claims.TenantID).Scan(&meta) != nil {
		writeError(w, http.StatusNotFound, "location not found")
		return
	}
	metaMap := parseMetaMap(meta)
	pool := metaStringSlice(metaMap, "ip_pool")
	added := 0
	for _, addr := range pool {
		addr = strings.TrimSpace(addr)
		if net.ParseIP(addr) == nil {
			continue
		}
		tag, err := h.dbOf(ctx).Exec(ctx, `
			INSERT INTO core.ip_pools (tenant_id, node_id, address, status, label)
			VALUES ($1, $2, $3::inet, 'free', 'из настроек локации')
			ON CONFLICT (node_id, address) DO NOTHING
		`, claims.TenantID, nodeID, addr)
		if err == nil {
			added += int(tag.RowsAffected())
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"added": added, "found": len(pool)})
}

type assignIPBody struct {
	IPID string `json:"ip_id"`
}

func (h *Handler) AdminServerAssignIP(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	serverID := chi.URLParam(r, "id")
	var body assignIPBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.IPID) == "" {
		writeError(w, http.StatusBadRequest, "ip_id required")
		return
	}
	ctx := r.Context()

	var nodeID, gameID string
	if h.dbOf(ctx).QueryRow(ctx, `SELECT node_id::text, game_id FROM core.servers WHERE id = $1 AND tenant_id = $2`,
		serverID, claims.TenantID).Scan(&nodeID, &gameID) != nil {
		writeError(w, http.StatusNotFound, "server not found")
		return
	}

	tx, err := h.dbOf(ctx).Begin(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		UPDATE core.ip_pools SET status = 'free', server_id = NULL, assigned_at = NULL
		WHERE tenant_id = $1 AND server_id = $2::uuid
	`, claims.TenantID, serverID); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}

	var address string
	err = tx.QueryRow(ctx, `
		UPDATE core.ip_pools SET status = 'assigned', server_id = $3::uuid, assigned_at = now()
		WHERE id = $1 AND tenant_id = $2 AND node_id = $4 AND server_id IS NULL AND status = 'free'
		RETURNING host(address)
	`, body.IPID, claims.TenantID, serverID, nodeID).Scan(&address)
	if err != nil {
		writeError(w, http.StatusConflict, "адрес занят, не найден или принадлежит другой локации")
		return
	}

	if _, err := tx.Exec(ctx, `
		UPDATE core.servers SET ip_address = $2, primary_port = NULL WHERE id = $1
	`, serverID, address); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	if _, err := tx.Exec(ctx, `DELETE FROM core.server_ports WHERE server_id = $1`, serverID); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	port := 0
	if gameID != "" && gameID != "test" {
		port, err = portalloc.AssignOn(ctx, tx, claims.TenantID, nodeID, serverID, gameID, address)
		if err != nil {
			writeError(w, http.StatusConflict, "не удалось выдать порт на этом адресе: "+err.Error())
			return
		}
	}
	if err := tx.Commit(ctx); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}

	audit(ctx, h.dbOf(ctx), claims.TenantID, claims.UserID, "server.ip.assign", "server:"+serverID,
		map[string]any{"address": address, "port": port})
	writeJSON(w, http.StatusOK, map[string]any{
		"address": address, "port": port,
		"restart_required": true,
	})
}

func (h *Handler) AdminServerReleaseIP(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	serverID := chi.URLParam(r, "id")
	ctx := r.Context()

	var nodeID, gameID string
	if h.dbOf(ctx).QueryRow(ctx, `SELECT node_id::text, game_id FROM core.servers WHERE id = $1 AND tenant_id = $2`,
		serverID, claims.TenantID).Scan(&nodeID, &gameID) != nil {
		writeError(w, http.StatusNotFound, "server not found")
		return
	}

	tx, err := h.dbOf(ctx).Begin(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `
		UPDATE core.ip_pools SET status = 'free', server_id = NULL, assigned_at = NULL
		WHERE tenant_id = $1 AND server_id = $2::uuid
	`, claims.TenantID, serverID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusConflict, "у сервера нет выделенного адреса")
		return
	}

	var fqdn string
	_ = tx.QueryRow(ctx, `SELECT COALESCE(fqdn, '') FROM core.nodes WHERE id = $1`, nodeID).Scan(&fqdn)
	if _, err := tx.Exec(ctx, `
		UPDATE core.servers SET ip_address = $2, primary_port = NULL WHERE id = $1
	`, serverID, fqdn); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	if _, err := tx.Exec(ctx, `DELETE FROM core.server_ports WHERE server_id = $1`, serverID); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	port := 0
	if gameID != "" && gameID != "test" {
		port, err = portalloc.AssignOn(ctx, tx, claims.TenantID, nodeID, serverID, gameID, "")
		if err != nil {
			writeError(w, http.StatusConflict, "не удалось выдать порт на общем адресе: "+err.Error())
			return
		}
	}
	if err := tx.Commit(ctx); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}

	audit(ctx, h.dbOf(ctx), claims.TenantID, claims.UserID, "server.ip.release", "server:"+serverID, nil)
	writeJSON(w, http.StatusOK, map[string]any{
		"address": fqdn, "port": port, "restart_required": true,
	})
}

func (h *Handler) serverBindIP(ctx context.Context, tenantID, serverID string) string {
	var address string
	if h.readerOf(ctx).QueryRow(ctx, `
		SELECT host(address) FROM core.ip_pools
		WHERE tenant_id = $1 AND server_id = $2::uuid
		LIMIT 1
	`, tenantID, serverID).Scan(&address) != nil {
		return ""
	}
	return address
}
