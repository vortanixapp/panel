package handlers

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/vortanixapp/panel/pkg/gamecatalog"
	"github.com/vortanixapp/panel/pkg/settingsreg"
)

const (
	firewallRulesMinAgent = "0.1.95"
	maxFirewallConnLimit  = 1000
)

type firewallPort struct {
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
	Purpose  string `json:"purpose"`
	Primary  bool   `json:"primary"`
}

func (h *Handler) firewallServerPorts(ctx context.Context, serverID string) []firewallPort {
	var primary int
	_ = h.dbOf(ctx).QueryRow(ctx, `SELECT COALESCE(primary_port, 0) FROM core.servers WHERE id = $1`, serverID).Scan(&primary)
	list := []firewallPort{}
	if primary > 0 {
		list = append(list,
			firewallPort{Port: primary, Protocol: "tcp", Purpose: "основной порт сервера", Primary: true},
			firewallPort{Port: primary, Protocol: "udp", Purpose: "основной порт сервера", Primary: true},
		)
	}
	rows, err := h.dbOf(ctx).Query(ctx, `
		SELECT port, protocol, COALESCE(purpose, '') FROM core.server_ports WHERE server_id = $1 ORDER BY port ASC
	`, serverID)
	if err != nil {
		return list
	}
	defer rows.Close()
	for rows.Next() {
		var p firewallPort
		if rows.Scan(&p.Port, &p.Protocol, &p.Purpose) == nil {
			p.Protocol = strings.ToLower(p.Protocol)
			list = append(list, p)
		}
	}
	return list
}

func firewallPortCovered(ports []firewallPort, protocol string, from, to int) bool {
	for port := from; port <= to; port++ {
		found := false
		for _, p := range ports {
			if p.Port == port && p.Protocol == protocol {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func normalizeFirewallSource(raw string) (string, bool) {
	src := strings.TrimSpace(raw)
	if src == "" {
		return "", true
	}
	if ip := net.ParseIP(src); ip != nil {
		if ip.To4() == nil {
			return "", false
		}
		return ip.To4().String(), true
	}
	ip, network, err := net.ParseCIDR(src)
	if err != nil || ip.To4() == nil {
		return "", false
	}
	return network.String(), true
}

func (h *Handler) firewallConnLimit(ctx context.Context, serverID string) (int, []int) {
	var limit int
	_ = h.dbOf(ctx).QueryRow(ctx, `
		SELECT COALESCE((config->>'firewall_conn_limit')::int, 0) FROM core.servers WHERE id = $1
	`, serverID).Scan(&limit)
	ports := []int{}
	if limit <= 0 {
		return 0, ports
	}
	seen := map[int]bool{}
	for _, p := range h.firewallServerPorts(ctx, serverID) {
		if p.Protocol == "tcp" && !seen[p.Port] {
			seen[p.Port] = true
			ports = append(ports, p.Port)
		}
	}
	return limit, ports
}

func (h *Handler) firewallAgentTooOld(ctx context.Context, nodeID string, rules []map[string]any, connLimit int) string {
	needed := connLimit > 0
	for _, r := range rules {
		action, _ := r["action"].(string)
		source, _ := r["source"].(string)
		if action == "allow" || source != "" {
			needed = true
		}
	}
	if !needed {
		return ""
	}
	var version string
	_ = h.dbOf(ctx).QueryRow(ctx, `
		SELECT COALESCE(version, '') FROM core.node_daemons WHERE node_id = $1::uuid
	`, nodeID).Scan(&version)
	if gamecatalog.AgentAtLeast(version, firewallRulesMinAgent) {
		return ""
	}
	return "Правила по адресам и лимит подключений работают с агентом " + firewallRulesMinAgent +
		" и новее: обновите агента на ноде (сейчас " + version + ")"
}

func (h *Handler) ServerFirewallList(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if _, ok := h.authorizeServerTab(w, r, id, "firewall_list"); !ok {
		return
	}
	ctx := r.Context()
	list, err := h.loadFirewallRules(ctx, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	limit, _ := h.firewallConnLimit(ctx, id)
	writeJSON(w, http.StatusOK, map[string]any{
		"rules":      list,
		"ports":      h.firewallServerPorts(ctx, id),
		"conn_limit": limit,
	})
}

func (h *Handler) ServerFirewallCreate(w http.ResponseWriter, r *http.Request) {
	serverID := chi.URLParam(r, "id")
	if _, ok := h.authorizeServerTab(w, r, serverID, "firewall_create"); !ok {
		return
	}
	var body struct {
		Protocol string `json:"protocol"`
		PortFrom int    `json:"port_from"`
		PortTo   *int   `json:"port_to"`
		Action   string `json:"action"`
		Source   string `json:"source"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.PortFrom < 1 || body.PortFrom > 65535 {
		writeError(w, http.StatusBadRequest, "Неверное правило: укажите порт от 1 до 65535")
		return
	}
	body.Protocol = strings.ToLower(strings.TrimSpace(body.Protocol))
	if body.Protocol == "" {
		body.Protocol = "tcp"
	}
	if body.Protocol != "tcp" && body.Protocol != "udp" {
		writeError(w, http.StatusBadRequest, "Протокол должен быть TCP или UDP")
		return
	}
	body.Action = strings.ToLower(strings.TrimSpace(body.Action))
	if body.Action == "" {
		body.Action = "deny"
	}
	if body.Action != "allow" && body.Action != "deny" {
		writeError(w, http.StatusBadRequest, "Действие должно быть «разрешить» или «закрыть»")
		return
	}
	source, ok := normalizeFirewallSource(body.Source)
	if !ok {
		writeError(w, http.StatusBadRequest, "Адрес источника указан неверно: нужен IPv4-адрес или подсеть вида 203.0.113.0/24")
		return
	}
	if body.Action == "allow" && source == "" {
		writeError(w, http.StatusBadRequest, "Для правила «разрешить только» укажите адрес или подсеть")
		return
	}
	portTo := body.PortFrom
	if body.PortTo != nil && *body.PortTo > body.PortFrom {
		portTo = *body.PortTo
	}
	if portTo > 65535 {
		writeError(w, http.StatusBadRequest, "Порт вне диапазона 1–65535")
		return
	}
	ctx := r.Context()
	ports := h.firewallServerPorts(ctx, serverID)
	if !firewallPortCovered(ports, body.Protocol, body.PortFrom, portTo) {
		writeError(w, http.StatusBadRequest, "Порт не относится к этому серверу: выберите основной или один из дополнительных портов")
		return
	}
	if body.Action == "deny" && source == "" {
		for _, p := range ports {
			if p.Primary && p.Port >= body.PortFrom && p.Port <= portTo && p.Protocol == body.Protocol {
				writeError(w, http.StatusBadRequest, "Нельзя закрыть основной порт сервера для всех адресов: игроки не смогут подключиться. Укажите адрес или подсеть, которые нужно закрыть")
				return
			}
		}
	}
	if h.overServerLimit(ctx, w, "core.server_firewall_rules", "", serverID, settingsreg.ServersMaxFirewallRules,
		"Достигнут предел числа правил файрвола для сервера") {
		return
	}
	var portToArg *int
	if portTo > body.PortFrom {
		portToArg = &portTo
	}
	var sourceArg *string
	if source != "" {
		sourceArg = &source
	}
	var rid string
	err := h.dbOf(ctx).QueryRow(ctx, `
		INSERT INTO core.server_firewall_rules (server_id, protocol, port_from, port_to, action, source)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id::text
	`, serverID, body.Protocol, body.PortFrom, portToArg, body.Action, sourceArg).Scan(&rid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "create failed")
		return
	}
	if !h.syncFirewallForServerHTTP(w, r, serverID) {
		_, _ = h.dbOf(ctx).Exec(ctx, `DELETE FROM core.server_firewall_rules WHERE id = $1 AND server_id = $2`, rid, serverID)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"id": rid})
}

func (h *Handler) ServerFirewallLimit(w http.ResponseWriter, r *http.Request) {
	serverID := chi.URLParam(r, "id")
	if _, ok := h.authorizeServerTab(w, r, serverID, "firewall_create"); !ok {
		return
	}
	var body struct {
		ConnLimit int `json:"conn_limit"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.ConnLimit < 0 || body.ConnLimit > maxFirewallConnLimit {
		writeError(w, http.StatusBadRequest, "Лимит подключений — от 0 (выключен) до 1000")
		return
	}
	ctx := r.Context()
	prev, _ := h.firewallConnLimit(ctx, serverID)
	set := func(value int) error {
		_, err := h.dbOf(ctx).Exec(ctx, `
			UPDATE core.servers
			SET config = COALESCE(config, '{}'::jsonb) || jsonb_build_object('firewall_conn_limit', $2::int)
			WHERE id = $1
		`, serverID, value)
		return err
	}
	if err := set(body.ConnLimit); err != nil {
		writeError(w, http.StatusInternalServerError, "update failed")
		return
	}
	if !h.syncFirewallForServerHTTP(w, r, serverID) {
		_ = set(prev)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"conn_limit": body.ConnLimit})
}
