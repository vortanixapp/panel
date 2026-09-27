package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/vortanixapp/panel/pkg/portalloc"
)

// Дополнительные порты выдаёт только персонал: клиент видит свои порты, но
// занять порт хоста сам не может. Docker работает от root и займёт любой
// свободный порт ноды, поэтому выбор порта — решение администратора.

func (h *Handler) AdminServerPortsList(w http.ResponseWriter, r *http.Request) {
	if _, ok := tenantClaims(r.Context()); !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	serverID := chi.URLParam(r, "id")
	ports, primaryPort, err := h.loadServerPortRows(r, serverID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "ports load failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ports":        ports,
		"primary_port": primaryPort,
		"min_port":     portalloc.MinClientPort,
	})
}

func (h *Handler) AdminServerPortCreate(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	serverID := chi.URLParam(r, "id")
	var body struct {
		Port     int    `json:"port"`
		Protocol string `json:"protocol"`
		Purpose  string `json:"purpose"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if body.Port < 1 || body.Port > 65535 {
		writeError(w, http.StatusBadRequest, "порт должен быть в диапазоне 1..65535")
		return
	}
	if portalloc.Reserved(body.Port) {
		writeError(w, http.StatusUnprocessableEntity, fmt.Sprintf(
			"порт %d занят службами ноды: выберите порт от %d до 65535 из незарезервированных",
			body.Port, portalloc.MinClientPort))
		return
	}
	body.Protocol = strings.ToLower(strings.TrimSpace(body.Protocol))
	if body.Protocol == "" {
		body.Protocol = "udp"
	}
	if body.Protocol != "tcp" && body.Protocol != "udp" && body.Protocol != "both" {
		writeError(w, http.StatusBadRequest, "протокол должен быть tcp, udp или both")
		return
	}
	body.Purpose = strings.TrimSpace(body.Purpose)
	if body.Purpose == "" {
		body.Purpose = "game"
	}

	var primaryPort int
	var nodeID string
	if err := h.dbOf(r.Context()).QueryRow(r.Context(), `
		SELECT COALESCE(primary_port, 0), COALESCE(node_id::text, '')
		FROM core.servers WHERE id = $1
	`, serverID).Scan(&primaryPort, &nodeID); err != nil {
		writeError(w, http.StatusNotFound, "сервер не найден")
		return
	}
	if primaryPort > 0 && body.Port == primaryPort {
		writeError(w, http.StatusConflict, "порт совпадает с основным портом сервера")
		return
	}
	if nodeID != "" {
		var occupied bool
		_ = h.dbOf(r.Context()).QueryRow(r.Context(), `
			SELECT EXISTS(
				SELECT 1 FROM core.servers
				WHERE node_id::text = $1 AND id <> $2
				  AND COALESCE(primary_port, 0) = $3
			) OR EXISTS(
				SELECT 1 FROM core.server_ports sp
				JOIN core.servers s ON s.id = sp.server_id
				WHERE s.node_id::text = $1 AND sp.server_id <> $2::uuid
				  AND sp.port = $3
			)
		`, nodeID, serverID, body.Port).Scan(&occupied)
		if occupied {
			writeError(w, http.StatusConflict, "порт уже занят другим сервером на этой ноде")
			return
		}
	}
	var duplicate bool
	_ = h.dbOf(r.Context()).QueryRow(r.Context(), `
		SELECT EXISTS(
			SELECT 1 FROM core.server_ports
			WHERE server_id = $1 AND port = $2
			  AND (protocol = 'both' OR $3 = 'both' OR protocol = $3)
		)
	`, serverID, body.Port, body.Protocol).Scan(&duplicate)
	if duplicate {
		writeError(w, http.StatusConflict, "такой порт у сервера уже есть")
		return
	}

	var portID string
	if err := h.dbOf(r.Context()).QueryRow(r.Context(), `
		INSERT INTO core.server_ports ( server_id, port, protocol, purpose, meta)
		VALUES ( $1, $2, $3, $4, '{}'::jsonb)
		RETURNING id::text
	`, serverID, body.Port, body.Protocol, body.Purpose).Scan(&portID); err != nil {
		writeError(w, http.StatusInternalServerError, "не удалось добавить порт")
		return
	}
	if !h.syncPortsForServerHTTP(w, r, serverID) {
		_, _ = h.dbOf(r.Context()).Exec(r.Context(), `
			DELETE FROM core.server_ports WHERE id = $1 AND server_id = $2
		`, portID, serverID)
		return
	}
	audit(r.Context(), h.dbOf(r.Context()), claims.UserID, "server.port_add", "server:"+serverID,
		map[string]any{"port": body.Port, "protocol": body.Protocol, "purpose": body.Purpose})
	writeJSON(w, http.StatusCreated, map[string]any{
		"id":       portID,
		"port":     body.Port,
		"protocol": body.Protocol,
		"purpose":  body.Purpose,
	})
}

func (h *Handler) AdminServerPortDelete(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	serverID := chi.URLParam(r, "id")
	portID := strings.TrimSpace(chi.URLParam(r, "portId"))
	if portID == "" {
		writeError(w, http.StatusBadRequest, "не указан порт")
		return
	}
	var port int
	var proto, purpose string
	if err := h.dbOf(r.Context()).QueryRow(r.Context(), `
		DELETE FROM core.server_ports
		WHERE id = $1 AND server_id = $2
		RETURNING port, protocol, COALESCE(purpose, '')
	`, portID, serverID).Scan(&port, &proto, &purpose); err != nil {
		writeError(w, http.StatusNotFound, "порт не найден")
		return
	}
	if !h.syncPortsForServerHTTP(w, r, serverID) {
		_, _ = h.dbOf(r.Context()).Exec(r.Context(), `
			INSERT INTO core.server_ports (id, server_id, port, protocol, purpose, meta)
			SELECT $1::uuid, $2::uuid, $3, $4, $5, '{}'::jsonb
			WHERE NOT EXISTS (SELECT 1 FROM core.server_ports WHERE id = $1::uuid)
		`, portID, serverID, port, proto, purpose)
		return
	}
	audit(r.Context(), h.dbOf(r.Context()), claims.UserID, "server.port_remove", "server:"+serverID,
		map[string]any{"port": port, "protocol": proto})
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
