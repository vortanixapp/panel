package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/vortanix/vortanix/internal/api/jobwake"
	"github.com/vortanix/vortanix/internal/api/licenseclient"
	"github.com/vortanix/vortanix/internal/api/licensestate"
)

func (h *Handler) AdminUpdatesCheck(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	if !requireAdmin(w, claims) {
		return
	}

	// «Проверить сейчас» должна действительно спросить сервис лицензий, а не
	// перечитать сохранённое: иначе кнопка показывает то же, что и до нажатия.
	if r.URL.Query().Get("refresh") == "1" && h.licenseOps != nil {
		h.licenseOps.RefreshTenant(r.Context(), h.dbOf(r.Context()))
		h.forgetTenantLicense(r.Context())
	}

	st := h.licenseFor(r.Context())
	upd := st.Update
	now := time.Now()

	// Сравниваем цель с текущей версией, а не просто проверяем, что цель
	// задана. Иначе после обновления target_version остаётся прежним, флаг
	// навсегда true, и панель бесконечно предлагает обновиться до версии, на
	// которой уже работает.
	panelVersion := envOr("VORTANIX_VERSION", "dev")

	// Страница разложена на три вкладки — панель, агенты на нодах, служба
	// обновления, — и каждой нужны свои релизы и свой журнал. Компонент
	// приходит параметром: тянуть данные всех трёх на каждый опрос незачем,
	// а опрашивается страница часто.
	component := strings.TrimSpace(r.URL.Query().Get("component"))
	switch component {
	case "", componentPanel:
		component = componentPanel
	case componentAgent, componentUpdater:
	default:
		writeError(w, http.StatusBadRequest, "неизвестный компонент")
		return
	}

	// Версия, с которой сравнивается цель, у каждого компонента своя: у панели
	// это её собственная сборка, у службы — то, что она о себе доложила, а
	// знает об этом только сервис лицензий.
	rel := h.updateReleases(r, component)
	currentVersion := panelVersion
	if component != componentPanel && rel.CurrentVersion != "" {
		currentVersion = rel.CurrentVersion
	}

	out := map[string]any{
		"component":        component,
		"core_version":     panelVersion,
		"current_version":  currentVersion,
		"target_version":   upd.TargetVersion,
		"update_available": upd.Available(currentVersion),
		"mandatory_after":  nullableTime(timeValue(upd.MandatoryAfter)),
		"overdue":          upd.Overdue(now),
		"notes":            upd.Notes,
		"deferred_until":   nullableTime(timeValue(upd.DeferUntil)),
		// Из базы, а не из состояния лицензии в памяти: то обновляется лишь
		// после сверки с сервисом, и сразу после нажатия переключатель
		// показывал бы прежнее положение.
		"auto_update":      licensestate.LoadUpdateAuto(r.Context(), h.dbOf(r.Context())),
		"last_verified_at": nullableTime(st.LastVerifiedAt),
		"releases":         rel.Releases,
	}

	switch component {
	case componentAgent:
		// У агентов обновление идёт не через службу обновления, а заданиями
		// воркера, поэтому и состояние, и журнал у них свои.
		out["daemons"] = h.daemonVersions(r, claims.TenantID)
		out["agent_events"] = h.agentUpdateEvents(r, claims.TenantID)
		out["events"] = []any{}
	default:
		out["events"] = h.updateEvents(r, component)
	}

	writeJSON(w, http.StatusOK, out)
}

const (
	componentPanel   = "panel-ui"
	componentAgent   = "agent"
	componentUpdater = "updater"
)

// updateComponentOf — что именно просят обновить.
//
// Страница разложена на вкладки, и «Обновить сейчас» на каждой означает своё.
// Раньше компонент не передавался вовсе: решение уходило без него, а в аудит
// писалась строка panel-ui независимо от нажатой кнопки. Хуже того, флаг
// «применить сейчас» один на установку, и цикл панели, которому применять
// нечего, гасил команду, адресованную службе обновления.
//
// Пустое тело трактуем как панель, а не как ошибку: клиентская панель и это
// API обновляются независимо, и версия панели без этого поля должна
// продолжать работать ровно как прежде.
func updateComponentOf(r *http.Request) (string, bool) {
	var body struct {
		Component string `json:"component"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	// Агента здесь быть не может: его обновляет воркер отдельным заданием, а
	// не признаком «применить сейчас». Приняли бы — флаг взвёлся и никогда не
	// снялся: снимает его тот компонент, которому он адресован, а за агента
	// службу обновления никто не спрашивает.
	switch c := strings.TrimSpace(body.Component); c {
	case "":
		return componentPanel, true
	case componentPanel, componentUpdater:
		return c, true
	default:
		return "", false
	}
}

func (h *Handler) AdminUpdatesApply(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok || claims.Role != "owner" {
		writeError(w, http.StatusForbidden, "owner only")
		return
	}

	component, ok := updateComponentOf(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "неизвестный компонент")
		return
	}

	if err := licensestate.SaveUpdateDecision(r.Context(), h.dbOf(r.Context()), "apply_now", component, nil); err != nil {
		writeError(w, http.StatusInternalServerError, "не удалось сохранить решение")
		return
	}
	h.licenseOps.RefreshTenant(r.Context(), h.dbOf(r.Context()))

	audit(r.Context(), h.dbOf(r.Context()), claims.TenantID, claims.UserID, "admin.update.apply", component, nil)
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "scheduled"})
}

// AdminUpdatesAuto включает и выключает автоматическую установку обновлений.
//
// По умолчанию выключено: раскатанная версия показывается как доступная и
// ставится по кнопке. Раньше выбора не было вовсе — назначенная версия
// приезжала сама, когда служба обновления её замечала.
func (h *Handler) AdminUpdatesAuto(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok || claims.Role != "owner" {
		writeError(w, http.StatusForbidden, "owner only")
		return
	}
	var body struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	action := "auto_off"
	if body.Enabled {
		action = "auto_on"
	}
	// Выбор владельца записываем у себя сразу: страница читает его отсюда, и
	// ждать ответа сервиса лицензий, чтобы показать только что нажатое, незачем.
	if err := licensestate.SaveUpdateAuto(r.Context(), h.dbOf(r.Context()), body.Enabled); err != nil {
		writeError(w, http.StatusInternalServerError, "не удалось сохранить решение")
		return
	}
	if err := licensestate.SaveUpdateDecision(r.Context(), h.dbOf(r.Context()), action, componentPanel, nil); err != nil {
		writeError(w, http.StatusInternalServerError, "не удалось сохранить решение")
		return
	}
	h.licenseOps.RefreshTenant(r.Context(), h.dbOf(r.Context()))
	h.forgetTenantLicense(r.Context())

	audit(r.Context(), h.dbOf(r.Context()), claims.TenantID, claims.UserID,
		"admin.update.auto", componentPanel, map[string]any{"enabled": body.Enabled})
	writeJSON(w, http.StatusOK, map[string]bool{"auto_update": body.Enabled})
}

func (h *Handler) daemonVersions(r *http.Request, tenantID string) []map[string]any {
	rows, err := h.dbOf(r.Context()).Query(r.Context(), `
		SELECT n.id::text, n.name, COALESCE(d.version, ''), COALESCE(d.status, 'unknown'), d.last_seen_at
		FROM core.nodes n
		LEFT JOIN core.node_daemons d ON d.node_id = n.id AND d.tenant_id = n.tenant_id
		WHERE n.tenant_id = $1
		ORDER BY n.name
	`, tenantID)
	if err != nil {
		return nil
	}
	defer rows.Close()

	out := make([]map[string]any, 0, 8)
	for rows.Next() {
		var id, name, version, status string
		var lastSeen *time.Time
		if rows.Scan(&id, &name, &version, &status, &lastSeen) != nil {
			continue
		}
		item := map[string]any{"id": id, "name": name, "version": version, "status": status}
		if lastSeen != nil {
			item["last_seen_at"] = lastSeen.Format(time.RFC3339)
		}
		out = append(out, item)
	}
	return out
}

func (h *Handler) AdminDaemonSelfUpdate(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	if !requireAdmin(w, claims) {
		return
	}

	nodeID := chi.URLParam(r, "id")
	payload, _ := json.Marshal(map[string]any{
		"node_id": nodeID,
		"action":  "update",
		"params":  map[string]any{"version": h.licenseFor(r.Context()).Update.TargetVersion},
	})
	if _, err := h.dbOf(r.Context()).Exec(r.Context(), `
		INSERT INTO core.jobs (tenant_id, type, status, payload)
		VALUES ($1, 'daemon_action', 'pending', $2::jsonb)
	`, claims.TenantID, payload); err != nil {
		writeError(w, http.StatusInternalServerError, "не удалось поставить задачу")
		return
	}
	// Будим воркер: без этого задача лежит в очереди до его тикера, и кнопка
	// обновления агента срабатывает с задержкой на ровном месте.
	jobwake.Notify("daemon_action")

	audit(r.Context(), h.dbOf(r.Context()), claims.TenantID, claims.UserID, "admin.daemon.update", nodeID, nil)
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "scheduled"})
}

func timeValue(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return *t
}

// updateEvents отдаёт журнал обновления этой панели. Ошибку не поднимаем:
// страница обновлений полезна и без журнала, а её падение из-за недоступного
// сервиса лицензий было бы хуже пустого списка.
func (h *Handler) updateEvents(r *http.Request, component string) []licenseclient.UpdateEvent {
	if h.licenseOps == nil {
		return nil
	}
	row, err := licensestate.LoadRow(r.Context(), h.dbOf(r.Context()))
	if err != nil || row.LicenseKey == "" {
		return nil
	}
	events, err := h.licenseOps.UpdateEvents(r.Context(), row.LicenseKey, component)
	if err != nil {
		log.Printf("журнал обновления недоступен: %v", err)
		return nil
	}
	return events
}

// updateReleases отдаёт историю версий компонента с описаниями.
func (h *Handler) updateReleases(r *http.Request, component string) licenseclient.ComponentReleases {
	if h.licenseOps == nil {
		return licenseclient.ComponentReleases{}
	}
	row, err := licensestate.LoadRow(r.Context(), h.dbOf(r.Context()))
	if err != nil || row.LicenseKey == "" {
		return licenseclient.ComponentReleases{}
	}
	releases, err := h.licenseOps.UpdateReleases(r.Context(), row.LicenseKey, component)
	if err != nil {
		log.Printf("список релизов недоступен: %v", err)
		return licenseclient.ComponentReleases{}
	}
	return releases
}

// agentUpdateEvents — журнал обновлений агентов на нодах.
//
// Их обновляет не служба обновления панели, а воркер по SSH, поэтому события
// лежат не там же, где у панели, а в очереди задач. Показываем последние:
// вкладка «Агенты» без журнала отвечала бы только на вопрос «какая версия», но
// не «что происходило».
func (h *Handler) agentUpdateEvents(r *http.Request, tenantID string) []map[string]any {
	rows, err := h.dbOf(r.Context()).Query(r.Context(), `
		SELECT COALESCE(n.name, ''), j.status, COALESCE(j.payload->'params'->>'version', ''),
		       COALESCE(j.result->>'error', ''), j.created_at
		FROM core.jobs j
		LEFT JOIN core.nodes n ON n.id::text = j.payload->>'node_id'
		WHERE j.tenant_id = $1 AND j.type = 'daemon_action'
		  AND j.payload->>'action' = 'update'
		ORDER BY j.created_at DESC
		LIMIT 30
	`, tenantID)
	if err != nil {
		log.Printf("журнал обновления агентов недоступен: %v", err)
		return nil
	}
	defer rows.Close()

	out := []map[string]any{}
	for rows.Next() {
		var node, status, version, errText string
		var created time.Time
		if rows.Scan(&node, &status, &version, &errText, &created) != nil {
			continue
		}
		out = append(out, map[string]any{
			"node": node, "status": status, "version": version,
			"error": errText, "created_at": created.Format(time.RFC3339),
		})
	}
	return out
}
