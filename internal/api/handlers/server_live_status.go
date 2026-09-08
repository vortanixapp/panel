package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/vortanix/vortanix/pkg/gamesettings"
)

// configuredSlots — число мест из настроек самого сервера.
//
// Источник для тарифа за ресурсы: там клиент выставляет вместимость сам, и
// показывать нужно именно её. Какое поле отвечает за места, знает профиль
// игры — оно помечено признаком Slots, поэтому перечислять ключи вроде
// maxplayers, sv_maxclients и max_players по играм не требуется.
func configuredSlots(gameID string, config map[string]any) int {
	profile, ok := gamesettings.For(gameID)
	if !ok {
		return 0
	}
	stored := extractStoredGameSettings(config, profile.Key)
	for _, f := range profile.Fields {
		if !f.Slots {
			continue
		}
		// Сохранённое значение клиента приоритетнее умолчания профиля: второе
		// показывает лишь то, с чем сервер создавался.
		if n := intFromAny(stored[f.Key]); n > 0 {
			return n
		}
		if n, err := strconv.Atoi(strings.TrimSpace(f.Default)); err == nil && n > 0 {
			return n
		}
	}
	return 0
}

func (h *Handler) GetServerStatus(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	serverID := chi.URLParam(r, "id")
	access, err := h.resolveServerAccess(r.Context(), claims.TenantID, claims.UserID, claims.Role, serverID)
	if err != nil {
		writeError(w, http.StatusNotFound, "server not found")
		return
	}
	_ = access

	var status, runtime, prov, provError, gameID string
	var port int
	var limitsRaw []byte
	// Тип оплаты решает, откуда берётся число мест: за слоты — сколько куплено,
	// за ресурсы — сколько выставлено в настройках самого сервера.
	var billingType string
	var configRaw []byte
	err = h.dbOf(r.Context()).QueryRow(r.Context(), `
		SELECT COALESCE(s.status, ''), COALESCE(s.runtime_status, ''), COALESCE(s.provisioning_status, 'pending'),
		       COALESCE(s.provisioning_error, ''), COALESCE(s.game_id, ''), COALESCE(s.primary_port, 0),
		       COALESCE(s.limits, '{}'::jsonb), COALESCE(s.config, '{}'::jsonb), COALESCE(t.billing_type, '')
		FROM core.servers s
		LEFT JOIN core.tariffs t ON t.id = s.tariff_id
		WHERE s.id = $1 AND s.tenant_id = $2
	`, serverID, claims.TenantID).Scan(&status, &runtime, &prov, &provError, &gameID, &port, &limitsRaw, &configRaw, &billingType)
	if err != nil {
		writeError(w, http.StatusNotFound, "server not found")
		return
	}

	limits := map[string]any{}
	_ = json.Unmarshal(limitsRaw, &limits)
	config := map[string]any{}
	_ = json.Unmarshal(configRaw, &config)

	// Число мест: у тарифа за слоты это то, что куплено, у тарифа за ресурсы —
	// то, что клиент сам выставил в настройках игры. Раньше знаменатель был
	// пуст у всех серверов: ключ slots в limits кладут лишь некоторые пути
	// создания, а настройки игры на этот вопрос не спрашивали вовсе.
	paidSlots := 0
	if billingType == "slots" {
		paidSlots = limitsInt(limits, "slots")
	}
	baseSlots := paidSlots
	if baseSlots <= 0 {
		baseSlots = configuredSlots(gameID, config)
	}
	// Агент берёт максимум игроков из тех же limits, когда игра его не
	// сообщила — пусть у него будет то же значение, что показываем мы.
	if baseSlots > 0 && limitsInt(limits, "slots") <= 0 {
		limits["slots"] = baseSlots
	}

	resp := map[string]any{
		"ok":                  true,
		"status":              status,
		"runtime_status":      runtime,
		"provisioning_status": prov,
		"provisioning_error":  nilIfEmpty(provError),
		"online":              runtime == "running",
		"max_players":         baseSlots,
		"online_players":      0,
		"players_online":      []any{},
		"current_map":         "",
	}
	if live, ok := h.cache.GetServerStatus(r.Context(), serverID); ok {
		effective := resolveEffectiveStatus(status, runtime, &live)
		resp["status"] = effective
		resp["online"] = effective == "running" || runtime == "running"
	} else {
		resp["status"] = resolveEffectiveStatus(status, runtime, nil)
		resp["online"] = runtime == "running" || status == "running"
	}
	if progress := h.deriveProvisioningProgress(r.Context(), serverID, prov, status); progress != nil {
		resp["provisioning_progress"] = progress
	}

	nodeID, nodeErr := h.serverNodeID(r.Context(), claims.TenantID, serverID)
	if nodeErr == nil && nodeID != "" {
		result, agentErr := h.agentCommand(r.Context(), nodeID, serverID, "game_query", map[string]any{
			"game_id": gameID,
			"limits":  limits,
			"port":    port,
		})
		if agentErr == nil && result != nil {
			for _, k := range []string{"online", "max_players", "online_players", "players_online", "current_map", "runtime_status"} {
				if v, ok := result[k]; ok {
					resp[k] = v
				}
			}
			if rs, ok := result["runtime_status"].(string); ok && rs != "" {
				resp["runtime_status"] = rs
				resp["online"] = rs == "running"
			}
			// Опрос читает конфиг работающей игры — для тарифа за ресурсы это
			// и есть верный источник. Для тарифа за слоты последнее слово за
			// покупкой: в конфиге может стоять что угодно, а показать нужно
			// оплаченное.
			if paidSlots > 0 {
				resp["max_players"] = paidSlots
			} else if intFromAny(resp["max_players"]) <= 0 && baseSlots > 0 {
				resp["max_players"] = baseSlots
			}
		}
		if runtime == "running" || status == "running" {
			stats, statsErr := h.agentCommand(r.Context(), nodeID, serverID, "stats", map[string]any{"limits": limits})
			if statsErr == nil && stats != nil {
				if v, ok := stats["uptime"].(string); ok && v != "" {
					resp["uptime"] = v
				}
				if v, ok := stats["started_at"].(string); ok && v != "" {
					resp["started_at"] = v
				}
				for _, k := range []string{"disk_used_mb", "disk_total_mb", "cpu_percent", "mem_percent"} {
					if val, ok := stats[k]; ok {
						resp[k] = val
					}
				}
			}
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

func limitsInt(limits map[string]any, key string) int {
	if limits == nil {
		return 0
	}
	switch v := limits[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	default:
		return 0
	}
}
