package handlers

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/vortanixapp/panel/internal/api/jobwake"
	"github.com/vortanixapp/panel/internal/api/paneljwt"
	"github.com/vortanixapp/panel/internal/api/payments"
	"github.com/vortanixapp/panel/internal/api/pricing"
	"github.com/vortanixapp/panel/internal/api/relay"

	"github.com/vortanixapp/panel/pkg/notify"
)

func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	name := h.tenantSettingString(ctx, "panel.name")
	if name == "" {
		name = "Vortanix"
	}
	branding := map[string]any{"tenant_name": name}
	var gameCount, serverCount int
	_ = h.dbOf(ctx).QueryRow(ctx, `SELECT COUNT(*) FROM core.games WHERE active = true`).Scan(&gameCount)
	_ = h.dbOf(ctx).QueryRow(ctx, `SELECT COUNT(*) FROM core.servers`).Scan(&serverCount)
	writeJSON(w, http.StatusOK, map[string]any{
		"tenant":   map[string]any{"slug": singleTenantSlug, "name": name},
		"branding": branding,
		"stats":    map[string]int{"games": gameCount, "servers": serverCount},
		"features_enabled": map[string]bool{
			"billing": true, "hosting": true, "monitoring": true, "support": true,
		},
	})
}

func (h *Handler) ServerRenewPreview(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	serverID := chi.URLParam(r, "id")
	var body struct {
		Period    int    `json:"period"`
		PromoCode string `json:"promo_code"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if body.Period <= 0 {
		body.Period = 30
	}
	baseCost, currency, gameID, nodeID, tariffID, err := h.serverRenewBaseCost(r, serverID, body.Period)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	promo, promoErr := payments.PickPromotion(
		r.Context(), h.dbOf(r.Context()), payments.ApplyRenew, claims.UserID,
		body.PromoCode, tariffID, gameID, nodeID, baseCost,
	)
	preview := payments.ApplyRentDiscount(promo, baseCost)
	if promoErr != "" {
		preview.Valid = false
		errMsg := promoErr
		preview.Error = &errMsg
		preview.FinalCost = baseCost
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"period_days":   body.Period,
		"price":         preview.FinalCost,
		"currency":      currency,
		"base_cost":     preview.BaseCost,
		"promo_preview": preview,
	})
}

func (h *Handler) ServerRenew(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	serverID := chi.URLParam(r, "id")
	if !h.authorizeServerAction(w, r, claims, serverID, "renew") {
		return
	}
	var body struct {
		Period    int    `json:"period"`
		WalletID  string `json:"wallet_id"`
		PromoCode string `json:"promo_code"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if body.Period <= 0 {
		body.Period = 30
	}
	baseCost, currency, gameID, nodeID, tariffID, err := h.serverRenewBaseCost(r, serverID, body.Period)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	promo, promoErr := payments.PickPromotion(
		r.Context(), h.dbOf(r.Context()), payments.ApplyRenew, claims.UserID,
		body.PromoCode, tariffID, gameID, nodeID, baseCost,
	)
	if promoErr != "" {
		writeError(w, http.StatusBadRequest, promoErr)
		return
	}
	preview := payments.ApplyRentDiscount(promo, baseCost)
	price := preview.FinalCost
	if price > 0 {
		if _, err := h.debitWalletForRentSource(r, claims, body.WalletID, currency, price,
			"Server renew", "server_renew", serverID); err != nil {
			writeError(w, http.StatusPaymentRequired, err.Error())
			return
		}
		if preview.PromoID != "" {
			_ = payments.IncrementPromoUsage(r.Context(), h.dbOf(r.Context()), preview.PromoID)
		}
	}
	_, err = h.dbOf(r.Context()).Exec(r.Context(), `
		UPDATE core.servers
		SET expires_at = GREATEST(COALESCE(expires_at, now()), now()) + make_interval(days => $2),
		    suspended_at = NULL,
		    rental_period_days = LEAST(GREATEST($2::int, 1), 365),
		    dunning_stage = 0,
		    dunning_for = NULL
		WHERE id = $1
	`, serverID, body.Period)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "renew failed")
		return
	}
	audit(r.Context(), h.dbOf(r.Context()), claims.UserID, "server.renew", serverID, map[string]any{"period": body.Period})
	writeJSON(w, http.StatusOK, map[string]string{"status": "renewed"})
}

func (h *Handler) serverRenewBaseCost(r *http.Request, serverID string, periodDays int) (baseCost float64, currency, gameID, nodeID, tariffID string, err error) {
	return h.serverRenewBaseCostCtx(r.Context(), serverID, periodDays)
}

func (h *Handler) serverRenewBaseCostCtx(ctx context.Context, serverID string, periodDays int) (baseCost float64, currency, gameID, nodeID, tariffID string, err error) {
	currency = "RUB"
	var limitsRaw []byte
	err = h.dbOf(ctx).QueryRow(ctx, `
		SELECT COALESCE(s.game_id, ''), COALESCE(s.node_id::text, ''), COALESCE(s.tariff_id::text, ''),
		       COALESCE(s.limits, '{}'::jsonb), COALESCE(t.currency, 'RUB')
		FROM core.servers s
		LEFT JOIN core.tariffs t ON t.id = s.tariff_id
		WHERE s.id = $1
	`, serverID).Scan(&gameID, &nodeID, &tariffID, &limitsRaw, &currency)
	if err != nil {
		return 0, "", "", "", "", err
	}
	if currency == "" {
		currency = "RUB"
	}
	limits := map[string]any{}
	_ = json.Unmarshal(limitsRaw, &limits)
	order := pricing.RentOrder{
		Slots:    intFromAny(limits["slots"]),
		CPUCores: intFromAny(limits["cpu_cores"]),
		RAMGb:    intFromAny(limits["ram_gb"]),
		DiskGb:   intFromAny(limits["disk_gb"]),
	}
	if order.RAMGb <= 0 {
		if mb := intFromAny(limits["memory_mb"]); mb > 0 {
			order.RAMGb = mb / 1024
		}
	}
	if order.DiskGb <= 0 {
		if mb := intFromAny(limits["disk_mb"]); mb > 0 {
			order.DiskGb = mb / 1024
		}
	}
	if order.CPUCores <= 0 {
		order.CPUCores = intFromAny(limits["cpu"])
	}
	if tariffID != "" {
		tariffJSON, loadErr := h.loadTariffLegacyJSON(ctx, tariffID)
		if loadErr == nil {
			return pricing.CalculateRentCost(tariffJSON, order, periodDays), currency, gameID, nodeID, tariffID, nil
		}
	}
	price, cur, err := h.serverRenewPriceCtx(ctx, serverID, periodDays)
	return price, cur, gameID, nodeID, tariffID, err
}

func (h *Handler) serverRenewPriceCtx(ctx context.Context, serverID string, periodDays int) (price float64, currency string, err error) {
	currency = "RUB"
	err = h.dbOf(ctx).QueryRow(ctx, `
		SELECT COALESCE(t.price_monthly, 0), COALESCE(t.currency, 'RUB')
		FROM core.servers s
		LEFT JOIN core.tariffs t ON t.id = s.tariff_id
		WHERE s.id = $1
	`, serverID).Scan(&price, &currency)
	if err != nil {
		return 0, "", err
	}
	price = price * float64(periodDays) / 30.0
	return price, currency, nil
}

func (h *Handler) ServerAutoStart(w http.ResponseWriter, r *http.Request) {
	_, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var body struct {
		Enabled bool `json:"enabled"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	_, err := h.dbOf(r.Context()).Exec(r.Context(), `
		UPDATE core.servers SET auto_start = $2, config = config || jsonb_build_object('auto_start', $2)
		WHERE id = $1
	`, chi.URLParam(r, "id"), body.Enabled)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "update failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"auto_start": body.Enabled})
}

func (h *Handler) ServerStartupParams(w http.ResponseWriter, r *http.Request) {
	_, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var body map[string]any
	_ = json.NewDecoder(r.Body).Decode(&body)

	params := firstLine(bodyString(body, "startup_params"))

	if strings.HasPrefix(params, "/") || strings.HasPrefix(params, "./") {
		writeError(w, http.StatusUnprocessableEntity,
			"параметры запуска не могут начинаться с пути — укажите только аргументы, команду запуска подставит образ игры")
		return
	}

	_, err := h.dbOf(r.Context()).Exec(r.Context(), `
		UPDATE core.servers SET config = config || jsonb_build_object('startup_params', to_jsonb($2::text))
		WHERE id = $1
	`, chi.URLParam(r, "id"), params)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "update failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func firstLine(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}

func (h *Handler) defaultStartupParams(ctx context.Context, gameID string) string {
	var params string
	_ = h.dbOf(ctx).QueryRow(ctx, `
		SELECT COALESCE(meta->>'default_startup_params', '')
		FROM core.games WHERE slug = $1
	`, gameID).Scan(&params)
	return firstLine(params)
}

func (h *Handler) ServerSwitchVersion(w http.ResponseWriter, r *http.Request) {
	_, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	serverID := chi.URLParam(r, "id")
	var body struct {
		VersionID string `json:"version_id"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if body.VersionID == "" {
		writeError(w, http.StatusBadRequest, "version_id required")
		return
	}
	_, err := h.dbOf(r.Context()).Exec(r.Context(), `
		UPDATE core.servers SET game_version_id = $2::uuid
		WHERE id = $1
	`, serverID, body.VersionID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid version")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "version_switched", "version_id": body.VersionID})
}

func (h *Handler) ServerFilesMkdir(w http.ResponseWriter, r *http.Request) {
	serverID := chi.URLParam(r, "id")
	var body struct {
		Path string `json:"path"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	_, ok := h.agentCommandForServer(w, r, serverID, "files_mkdir", map[string]any{"path": body.Path})
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "created"})
}

func (h *Handler) ServerFilesDelete(w http.ResponseWriter, r *http.Request) {
	serverID := chi.URLParam(r, "id")
	var body struct {
		Path string `json:"path"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	_, ok := h.agentCommandForServer(w, r, serverID, "files_delete", map[string]any{"path": body.Path})
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) ServerFilesUpload(w http.ResponseWriter, r *http.Request) {
	serverID := chi.URLParam(r, "id")
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if err := r.ParseMultipartForm(64 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "invalid multipart")
		return
	}
	dirPath := strings.TrimSpace(r.FormValue("path"))
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "file required")
		return
	}
	defer file.Close()
	content, err := io.ReadAll(file)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read uploaded file")
		return
	}
	if len(content) == 0 {
		writeError(w, http.StatusBadRequest, "empty file")
		return
	}
	filename := strings.TrimSpace(header.Filename)
	if filename == "" {
		filename = "uploaded.bin"
	}
	fullPath := path.Join(path.Clean("/"+dirPath), filename)
	if !h.authorizeServerAction(w, r, claims, serverID, "files_upload") {
		return
	}
	nodeID, err := h.serverNodeID(r.Context(), serverID)
	if err != nil {
		writeError(w, http.StatusNotFound, "server not found")
		return
	}
	_, relayErr := h.relay.CommandSyncBinary(r.Context(), nodeID, relay.CommandRequest{
		CommandID: uuid.NewString(),
		Action:    "files_write_binary",
		ServerID:  serverID,
		Payload: map[string]any{
			"path": fullPath,
			"size": len(content),
		},
	}, content)
	if relayErr != nil {
		_, ok = h.agentCommandForServer(w, r, serverID, "files_write", map[string]any{
			"path":           fullPath,
			"encoding":       "base64",
			"content_base64": base64.StdEncoding.EncodeToString(content),
		})
		if !ok {
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":       "uploaded",
		"path":         fullPath,
		"size_bytes":   len(content),
		"content_type": header.Header.Get("Content-Type"),
		"transport": map[string]any{
			"binary_relay": relayErr == nil,
			"fallback":     relayErr != nil,
		},
	})
}

func (h *Handler) ServerFilesDownload(w http.ResponseWriter, r *http.Request) {
	serverID := chi.URLParam(r, "id")
	filePath := strings.TrimSpace(r.URL.Query().Get("path"))
	if filePath == "" {
		writeError(w, http.StatusBadRequest, "path required")
		return
	}
	result, ok := h.agentCommandForServer(w, r, serverID, "files_read", map[string]any{
		"path":     filePath,
		"encoding": "base64",
		"compress": true,
	})
	if !ok {
		return
	}
	b64, _ := result["content_base64"].(string)
	if b64 == "" {
		writeError(w, http.StatusBadGateway, "empty file response")
		return
	}
	content, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		writeError(w, http.StatusBadGateway, "invalid file payload")
		return
	}
	if strings.EqualFold(toString(result["compression"]), "gzip") {
		gz, gzErr := gzip.NewReader(bytes.NewReader(content))
		if gzErr != nil {
			writeError(w, http.StatusBadGateway, "invalid compressed payload")
			return
		}
		defer gz.Close()
		plain, readErr := io.ReadAll(gz)
		if readErr != nil {
			writeError(w, http.StatusBadGateway, "failed to decode compressed payload")
			return
		}
		content = plain
	}
	baseName := path.Base(filePath)
	if baseName == "." || baseName == "/" {
		baseName = "download.bin"
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", `attachment; filename="`+baseName+`"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(content)
}

func (h *Handler) AdminUsersCreateForm(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"roles":      []string{"user", "admin", "support"},
		"currencies": []string{"RUB", "USD", "EUR"},
	})
}

func (h *Handler) AdminUserCreateWallet(w http.ResponseWriter, r *http.Request) {
	_, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	userID := chi.URLParam(r, "id")
	var body struct {
		Currency string `json:"currency"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if body.Currency == "" {
		body.Currency = "RUB"
	}
	var wid string
	err := h.dbOf(r.Context()).QueryRow(r.Context(), `
		INSERT INTO core.wallets ( user_id, currency)
		VALUES ( $1, $2)
		ON CONFLICT (user_id, currency) DO UPDATE SET updated_at = now()
		RETURNING id::text
	`, userID, body.Currency).Scan(&wid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":        true,
		"wallet_id": wid,
		"message":   "Кошелёк создан",
	})
}

func (h *Handler) AdminListServers(w http.ResponseWriter, r *http.Request) {
	_, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	list := h.listEnrichedServers(r.Context(), "", true, 0)
	writeJSON(w, http.StatusOK, map[string]any{"servers": list})
}

func (h *Handler) AdminToggleServerBlock(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	var body struct {
		Blocked bool   `json:"blocked"`
		Reason  string `json:"reason"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	ctx := r.Context()

	var nodeID, name, gameID, status, ownerID string
	var limitsRaw []byte
	err := h.dbOf(ctx).QueryRow(ctx, `
		UPDATE core.servers SET is_blocked = $2, blocked_reason = NULLIF($3, ''),
		    blocked_at = CASE WHEN $2 THEN now() ELSE NULL END
		WHERE id = $1
		RETURNING COALESCE(node_id::text, ''), name, game_id, COALESCE(status, ''),
		          COALESCE(user_id::text, ''), COALESCE(limits, '{}'::jsonb)
	`, id, body.Blocked, body.Reason).
		Scan(&nodeID, &name, &gameID, &status, &ownerID, &limitsRaw)
	if err != nil {
		writeError(w, http.StatusNotFound, "server not found")
		return
	}

	if body.Blocked && nodeID != "" && status != "stopped" && status != "error" {
		var lim map[string]any
		_ = json.Unmarshal(limitsRaw, &lim)
		cmdErr := h.relay.SendCommand(ctx, nodeID, relay.CommandRequest{
			CommandID: uuid.NewString(),
			Action:    "power",
			ServerID:  id,
			Payload: map[string]any{
				"power_action": "stop",
				"name":         name,
				"game_id":      gameID,
				"limits":       lim,
			},
		})
		if cmdErr != nil {
			log.Printf("блокировка сервера %s: остановка не ушла на ноду: %v", id, cmdErr)
		} else {
			_, _ = h.dbOf(ctx).Exec(ctx, `UPDATE core.servers SET status = 'stopping' WHERE id = $1`, id)
			_ = h.cache.SetServerStatus(ctx, id, "stopping")
		}
	}
	_ = h.cache.InvalidateTenantServers(ctx)

	if ownerID != "" {
		reason := body.Reason
		if reason == "" {
			reason = "обратитесь в поддержку"
		}
		event := notify.Event{
			Kind:      notify.KindServerBlocked,
			Title:     "Сервер заблокирован",
			Body:      "Сервер «" + name + "» заблокирован администратором и остановлен. Причина: " + reason + ".",
			Meta:      map[string]any{"server_id": id},
			DedupeKey: "server.blocked:" + id,
		}
		if !body.Blocked {
			event = notify.Event{
				Kind:      notify.KindServerUnblocked,
				Title:     "Блокировка снята",
				Body:      "Сервер «" + name + "» разблокирован. Его можно запускать.",
				Meta:      map[string]any{"server_id": id},
				DedupeKey: "server.unblocked:" + id,
			}
		}
		h.notifyUser(ctx, ownerID, event)
	}

	auditAction := "server.block"
	if !body.Blocked {
		auditAction = "server.unblock"
	}
	audit(ctx, h.dbOf(ctx), claims.UserID, auditAction, "server:"+id,
		map[string]any{"reason": body.Reason})

	writeJSON(w, http.StatusOK, map[string]bool{"blocked": body.Blocked})
}

func (h *Handler) AdminTariffDuplicate(w http.ResponseWriter, r *http.Request) {
	_, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	var newID string
	err := h.dbOf(r.Context()).QueryRow(r.Context(), `
		INSERT INTO core.tariffs (
			node_id, game_id, name, slug, billing_type, price_monthly, currency,
			slots_min, slots_max, cpu_cores, cpu_shares, ram_mb, disk_mb,
			rental_periods, renewal_periods, discounts, position, active, meta
		)
		SELECT node_id, game_id, name || ' (копия)', slug || '-copy-' || substr(gen_random_uuid()::text, 1, 8),
		       billing_type, price_monthly, currency, slots_min, slots_max, cpu_cores, cpu_shares, ram_mb, disk_mb,
		       rental_periods, renewal_periods, discounts, position, false, meta
		FROM core.tariffs WHERE id = $1
		RETURNING id::text
	`, id).Scan(&newID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "duplicate failed")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"id": newID})
}

func (h *Handler) AdminLanguageList(w http.ResponseWriter, r *http.Request) {
	_, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	locale := r.URL.Query().Get("locale")
	if locale == "" {
		locale = "ru"
	}
	rows, _ := h.dbOf(r.Context()).Query(r.Context(), `
		SELECT key, value FROM core.translation_keys WHERE locale = $1 ORDER BY key
	`, locale)
	defer func() {
		if rows != nil {
			rows.Close()
		}
	}()
	list := []map[string]string{}
	if rows != nil {
		for rows.Next() {
			var k, v string
			if rows.Scan(&k, &v) == nil {
				list = append(list, map[string]string{"key": k, "value": v})
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"locale": locale, "messages": list})
}

func (h *Handler) AdminLanguageUpdate(w http.ResponseWriter, r *http.Request) {
	_, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	var body struct {
		Locale   string            `json:"locale"`
		Messages map[string]string `json:"messages"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if body.Locale == "" {
		body.Locale = "ru"
	}
	for k, v := range body.Messages {
		_, _ = h.dbOf(r.Context()).Exec(r.Context(), `
			INSERT INTO core.translation_keys (locale, key, value)
			VALUES ($1, $2, $3)
			ON CONFLICT (locale, key) DO UPDATE SET value = EXCLUDED.value, updated_at = now()
		`, body.Locale, k, v)
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (h *Handler) LocationPullDaemon(w http.ResponseWriter, r *http.Request) {
	_, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	nodeID := chi.URLParam(r, "id")
	h.enqueueLocationPull(r, nodeID)
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "pulling"})
}

func (h *Handler) enqueueLocationPull(r *http.Request, nodeID string) {
	ctx := r.Context()
	var pending bool
	_ = h.dbOf(ctx).QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM core.jobs
			WHERE type = 'daemon_pull' AND status IN ('pending', 'running')
			  AND payload->>'node_id' = $1
		)
	`, nodeID).Scan(&pending)
	if pending {
		return
	}
	payload, _ := json.Marshal(map[string]string{"node_id": nodeID})
	_, _ = h.dbOf(ctx).Exec(ctx, `
		INSERT INTO core.jobs ( type, status, payload) VALUES ( 'daemon_pull', 'pending', $1::jsonb)
	`, payload)
	jobwake.Notify("daemon_pull")
}

func (h *Handler) nodeMetricsFresh(r *http.Request, nodeID string) bool {
	var fresh bool
	_ = h.readerOf(r.Context()).QueryRow(r.Context(), `
		SELECT EXISTS(
			SELECT 1 FROM core.node_metrics
			WHERE node_id = $1 AND measured_at > NOW() - INTERVAL '5 minutes'
		)
	`, nodeID).Scan(&fresh)
	return fresh
}

func (h *Handler) AdminMysqlIndex(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok || !isStaffRole(claims.Role) {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	rows, err := h.dbOf(r.Context()).Query(r.Context(), `
		SELECT id::text, COALESCE(name, ''), COALESCE(region, ''), COALESCE(meta, '{}'::jsonb)
		FROM core.nodes
		ORDER BY created_at DESC
	`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()
	instances := make([]map[string]any, 0)
	for rows.Next() {
		var nodeID, name, region string
		var metaRaw []byte
		if rows.Scan(&nodeID, &name, &region, &metaRaw) != nil {
			continue
		}
		meta := map[string]any{}
		_ = json.Unmarshal(metaRaw, &meta)
		raw, _ := meta["mysql_instances"].([]any)
		for idx, item := range raw {
			inst, ok := item.(map[string]any)
			if !ok {
				continue
			}
			key := strings.TrimSpace(toString(inst["key"]))
			if key == "" {
				key = "instance-" + toString(idx+1)
			}
			instances = append(instances, map[string]any{
				"node_id":       nodeID,
				"node_name":     name,
				"region":        region,
				"key":           key,
				"name":          strings.TrimSpace(toString(inst["name"])),
				"container":     strings.TrimSpace(toString(inst["container"])),
				"port":          intFromAny(inst["port"]),
				"root_password": strings.TrimSpace(toString(inst["root_password"])),
			})
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"instances": instances,
		"count":     len(instances),
	})
}

func (h *Handler) AdminMysqlInstanceCreate(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok || !isStaffRole(claims.Role) {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	var body struct {
		NodeID       string `json:"node_id"`
		Key          string `json:"key"`
		Name         string `json:"name"`
		Container    string `json:"container"`
		Port         int    `json:"port"`
		RootPassword string `json:"root_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	body.NodeID = strings.TrimSpace(body.NodeID)
	body.Key = strings.TrimSpace(body.Key)
	body.Name = strings.TrimSpace(body.Name)
	body.Container = strings.TrimSpace(body.Container)
	body.RootPassword = strings.TrimSpace(body.RootPassword)
	if body.NodeID == "" || body.Key == "" || body.Container == "" || body.Port <= 0 {
		writeError(w, http.StatusBadRequest, "node_id, key, container, port required")
		return
	}
	if err := h.adminMysqlInstanceUpsert(r.Context(), body.NodeID, body.Key, map[string]any{
		"key":           body.Key,
		"name":          body.Name,
		"container":     body.Container,
		"port":          body.Port,
		"root_password": body.RootPassword,
	}, false); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"ok": true})
}

func (h *Handler) AdminMysqlInstanceUpdate(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok || !isStaffRole(claims.Role) {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	var body struct {
		NodeID       string `json:"node_id"`
		Key          string `json:"key"`
		Name         string `json:"name"`
		Container    string `json:"container"`
		Port         int    `json:"port"`
		RootPassword string `json:"root_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	body.NodeID = strings.TrimSpace(body.NodeID)
	body.Key = strings.TrimSpace(body.Key)
	if body.NodeID == "" || body.Key == "" {
		writeError(w, http.StatusBadRequest, "node_id and key required")
		return
	}
	patch := map[string]any{}
	if v := strings.TrimSpace(body.Name); v != "" {
		patch["name"] = v
	}
	if v := strings.TrimSpace(body.Container); v != "" {
		patch["container"] = v
	}
	if body.Port > 0 {
		patch["port"] = body.Port
	}
	if v := strings.TrimSpace(body.RootPassword); v != "" {
		patch["root_password"] = v
	}
	if len(patch) == 0 {
		writeError(w, http.StatusBadRequest, "nothing to update")
		return
	}
	if err := h.adminMysqlInstanceUpsert(r.Context(), body.NodeID, body.Key, patch, true); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) AdminMysqlInstanceDelete(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok || !isStaffRole(claims.Role) {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	var body struct {
		NodeID string `json:"node_id"`
		Key    string `json:"key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	body.NodeID = strings.TrimSpace(body.NodeID)
	body.Key = strings.TrimSpace(body.Key)
	if body.NodeID == "" || body.Key == "" {
		writeError(w, http.StatusBadRequest, "node_id and key required")
		return
	}
	var metaRaw []byte
	err := h.dbOf(r.Context()).QueryRow(r.Context(), `
		SELECT COALESCE(meta, '{}'::jsonb)
		FROM core.nodes
		WHERE id = $1
	`, body.NodeID).Scan(&metaRaw)
	if err != nil {
		writeError(w, http.StatusNotFound, "node not found")
		return
	}
	meta := map[string]any{}
	_ = json.Unmarshal(metaRaw, &meta)
	src, _ := meta["mysql_instances"].([]any)
	dst := make([]any, 0, len(src))
	removed := false
	for _, item := range src {
		inst, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if strings.TrimSpace(toString(inst["key"])) == body.Key {
			removed = true
			continue
		}
		dst = append(dst, inst)
	}
	if !removed {
		writeError(w, http.StatusNotFound, "instance not found")
		return
	}
	meta["mysql_instances"] = dst
	metaJSON, _ := json.Marshal(meta)
	_, err = h.dbOf(r.Context()).Exec(r.Context(), `
		UPDATE core.nodes SET meta = $2::jsonb WHERE id = $1
	`, body.NodeID, metaJSON)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save node meta")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) adminMysqlInstanceUpsert(ctx context.Context, nodeID, key string, patch map[string]any, allowExisting bool) error {
	var metaRaw []byte
	err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT COALESCE(meta, '{}'::jsonb)
		FROM core.nodes
		WHERE id = $1
	`, nodeID).Scan(&metaRaw)
	if err != nil {
		return errors.New("node not found")
	}
	meta := map[string]any{}
	_ = json.Unmarshal(metaRaw, &meta)
	list, _ := meta["mysql_instances"].([]any)
	if list == nil {
		list = []any{}
	}
	found := false
	for i, item := range list {
		inst, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if strings.TrimSpace(toString(inst["key"])) != key {
			continue
		}
		if !allowExisting {
			return errors.New("instance key already exists")
		}
		for k, v := range patch {
			inst[k] = v
		}
		list[i] = inst
		found = true
		break
	}
	if !found {
		if allowExisting {
			return errors.New("instance not found")
		}
		list = append(list, patch)
	}
	meta["mysql_instances"] = list
	metaJSON, _ := json.Marshal(meta)
	_, err = h.dbOf(ctx).Exec(ctx, `
		UPDATE core.nodes SET meta = $2::jsonb WHERE id = $1
	`, nodeID, metaJSON)
	if err != nil {
		return errors.New("failed to save node meta")
	}
	return nil
}

func (h *Handler) RecordHTTPActivity(r *http.Request, claims *paneljwt.Claims, event string, statusCode int, duration time.Duration) {
	if claims == nil {
		return
	}
	_, _ = h.dbOf(r.Context()).Exec(r.Context(), `
		INSERT INTO core.http_activity_logs ( user_id, event, method, status_code, is_success, request_path, ip_address, user_agent, duration_ms)
		VALUES ( $1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, claims.UserID, event, r.Method, statusCode, statusCode < 400, r.URL.Path, clientIP(r), r.UserAgent(), int(duration.Milliseconds()))
}
