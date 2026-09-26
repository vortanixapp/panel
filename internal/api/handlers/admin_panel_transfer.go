package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/vortanixapp/panel/internal/api/jobwake"
	"github.com/vortanixapp/panel/internal/api/paneljwt"
	"github.com/vortanixapp/panel/pkg/buildinfo"
	"github.com/vortanixapp/panel/pkg/paneltransfer"
	"github.com/vortanixapp/panel/pkg/updates"
)

const (
	settingPanelFreeze      = "panel.freeze"
	settingPanelFreezeSince = "panel.freeze_since"
)

type panelTransferView struct {
	ID           string `json:"id"`
	Mode         string `json:"mode"`
	Status       string `json:"status"`
	Stage        string `json:"stage"`
	TargetHost   string `json:"target_host"`
	TargetUser   string `json:"target_user"`
	NewAddress   string `json:"new_address"`
	SameAddress  bool   `json:"same_address"`
	FreezeWrites bool   `json:"freeze_writes"`
	BytesTotal   int64  `json:"bytes_total"`
	BytesDone    int64  `json:"bytes_done"`
	AgentsTotal  int    `json:"agents_total"`
	AgentsDone   int    `json:"agents_done"`
	AgentsFailed int    `json:"agents_failed"`
	Log          string `json:"log"`
	Error        string `json:"error,omitempty"`
	CreatedAt    string `json:"created_at"`
	StartedAt    string `json:"started_at,omitempty"`
	FinishedAt   string `json:"finished_at,omitempty"`
}

func (h *Handler) panelTransferOwner(w http.ResponseWriter, r *http.Request) (*paneljwt.Claims, bool) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusForbidden, "forbidden")
		return claims, false
	}
	if _, isKey := apiKeyFromContext(r.Context()); isKey || claims.Role != "owner" {
		writeCodedError(w, http.StatusForbidden, "owner_only",
			"Перенос панели доступен только владельцу: операция выдаёт наружу ключ шифрования и всю базу")
		return claims, false
	}
	return claims, true
}

func (h *Handler) loadPanelTransfers(r *http.Request) (active *panelTransferView, last *panelTransferView, err error) {
	rows, err := h.dbOf(r.Context()).Query(r.Context(), `
		SELECT id::text, mode, status, stage, target_host, target_user, new_address, same_address,
		       freeze_writes, bytes_total, bytes_done, agents_total, agents_done, agents_failed,
		       log, COALESCE(error, ''), created_at, started_at, finished_at
		FROM core.panel_transfers
		ORDER BY created_at DESC LIMIT 5
	`)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var v panelTransferView
		var created time.Time
		var started, finished *time.Time
		if err := rows.Scan(&v.ID, &v.Mode, &v.Status, &v.Stage, &v.TargetHost, &v.TargetUser, &v.NewAddress,
			&v.SameAddress, &v.FreezeWrites, &v.BytesTotal, &v.BytesDone, &v.AgentsTotal, &v.AgentsDone,
			&v.AgentsFailed, &v.Log, &v.Error, &created, &started, &finished); err != nil {
			return nil, nil, err
		}
		v.CreatedAt = created.UTC().Format(time.RFC3339)
		if started != nil {
			v.StartedAt = started.UTC().Format(time.RFC3339)
		}
		if finished != nil {
			v.FinishedAt = finished.UTC().Format(time.RFC3339)
		}
		item := v
		if last == nil {
			last = &item
		}
		if active == nil && (v.Status == "pending" || v.Status == "running") {
			active = &item
		}
	}
	return active, last, rows.Err()
}

func (h *Handler) GetAdminPanelTransfer(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.panelTransferOwner(w, r); !ok {
		return
	}
	ctx := r.Context()
	active, last, err := h.loadPanelTransfers(r)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "не удалось прочитать переносы")
		return
	}

	out := map[string]any{
		"version": buildinfo.Current(),
		"frozen":  h.freezeActive(ctx),
		"active":  active,
		"last":    last,
		"nodes":   h.panelTransferNodes(r),
	}
	probe, err := updates.NewUpdater().TransferProbe(ctx)
	if err != nil {
		out["probe_error"] = err.Error()
	} else {
		out["probe"] = probe
	}
	writeJSON(w, http.StatusOK, out)
}

type panelTransferNode struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Host   string `json:"host"`
	Status string `json:"status"`
}

func (h *Handler) panelTransferNodes(r *http.Request) []panelTransferNode {
	out := []panelTransferNode{}
	rows, err := h.dbOf(r.Context()).Query(r.Context(), `
		SELECT id::text, COALESCE(name, ''), COALESCE(ssh_host, ''), COALESCE(status, '')
		FROM core.nodes ORDER BY name
	`)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var n panelTransferNode
		if rows.Scan(&n.ID, &n.Name, &n.Host, &n.Status) == nil {
			out = append(out, n)
		}
	}
	return out
}

func (h *Handler) PostAdminPanelTransferSSH(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.panelTransferOwner(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	var body struct {
		Host         string `json:"host"`
		Port         int    `json:"port"`
		User         string `json:"user"`
		Password     string `json:"password"`
		PrivateKey   string `json:"private_key"`
		HostKey      string `json:"host_key"`
		NewAddress   string `json:"new_address"`
		SameAddress  bool   `json:"same_address"`
		FreezeWrites bool   `json:"freeze_writes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	body.Host = strings.TrimSpace(body.Host)
	body.User = strings.TrimSpace(body.User)
	body.NewAddress = strings.TrimSpace(body.NewAddress)
	if body.Port == 0 {
		body.Port = 22
	}
	if body.Host == "" || body.User == "" {
		writeError(w, http.StatusUnprocessableEntity, "Укажите адрес нового сервера и пользователя SSH")
		return
	}
	if body.Password == "" && body.PrivateKey == "" {
		writeError(w, http.StatusUnprocessableEntity, "Укажите пароль или приватный ключ SSH")
		return
	}
	if !body.SameAddress && body.NewAddress == "" {
		writeError(w, http.StatusUnprocessableEntity, "Укажите адрес, на котором будет отвечать новая панель")
		return
	}

	active, _, err := h.loadPanelTransfers(r)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "не удалось прочитать переносы")
		return
	}
	if active != nil {
		writeError(w, http.StatusConflict, "Перенос уже идёт")
		return
	}

	kind, secret := "password", body.Password
	if body.PrivateKey != "" {
		kind, secret = "key", body.PrivateKey
	}
	sealed, err := h.secrets.Encrypt(secret)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "доступ к серверу не сохранён")
		return
	}

	var transferID string
	err = h.dbOf(ctx).QueryRow(ctx, `
		INSERT INTO core.panel_transfers
			(mode, status, target_host, target_port, target_user, target_host_key,
			 target_secret_kind, target_secret_enc, new_address, same_address, freeze_writes, created_by)
		VALUES ('ssh', 'pending', $1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id::text
	`, body.Host, body.Port, body.User, strings.TrimSpace(body.HostKey), kind, sealed,
		body.NewAddress, body.SameAddress, body.FreezeWrites, claims.UserID).Scan(&transferID)
	if err != nil {
		writeError(w, http.StatusConflict, "Перенос уже идёт")
		return
	}

	payload, _ := json.Marshal(map[string]string{"transfer_id": transferID})
	if _, err := h.dbOf(ctx).Exec(ctx, `
		INSERT INTO core.jobs (type, status, payload) VALUES ('panel_transfer', 'pending', $1::jsonb)
	`, payload); err != nil {
		writeError(w, http.StatusInternalServerError, "задача переноса не создана")
		return
	}
	jobwake.Notify("panel_transfer")
	audit(ctx, h.dbOf(ctx), claims.UserID, "panel_transfer.start", transferID, map[string]any{
		"host": body.Host, "mode": "ssh", "same_address": body.SameAddress, "new_address": body.NewAddress,
	})
	writeJSON(w, http.StatusAccepted, map[string]any{"id": transferID})
}

func (h *Handler) PostAdminPanelTransferAgents(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.panelTransferOwner(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	var body struct {
		Host       string `json:"host"`
		Port       int    `json:"port"`
		User       string `json:"user"`
		Password   string `json:"password"`
		PrivateKey string `json:"private_key"`
		HostKey    string `json:"host_key"`
		NewAddress string `json:"new_address"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	body.Host = strings.TrimSpace(body.Host)
	body.User = strings.TrimSpace(body.User)
	body.NewAddress = strings.TrimSpace(body.NewAddress)
	if body.Port == 0 {
		body.Port = 22
	}
	if body.Host == "" || body.User == "" || body.NewAddress == "" {
		writeError(w, http.StatusUnprocessableEntity, "Укажите доступ к новому серверу и его адрес")
		return
	}
	if body.Password == "" && body.PrivateKey == "" {
		writeError(w, http.StatusUnprocessableEntity, "Укажите пароль или приватный ключ SSH")
		return
	}
	active, _, err := h.loadPanelTransfers(r)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "не удалось прочитать переносы")
		return
	}
	if active != nil {
		writeError(w, http.StatusConflict, "Перенос уже идёт")
		return
	}

	kind, secret := "password", body.Password
	if body.PrivateKey != "" {
		kind, secret = "key", body.PrivateKey
	}
	sealed, err := h.secrets.Encrypt(secret)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "доступ к серверу не сохранён")
		return
	}
	var transferID string
	err = h.dbOf(ctx).QueryRow(ctx, `
		INSERT INTO core.panel_transfers
			(mode, status, target_host, target_port, target_user, target_host_key,
			 target_secret_kind, target_secret_enc, new_address, freeze_writes, created_by)
		VALUES ('agents', 'pending', $1, $2, $3, $4, $5, $6, $7, false, $8)
		RETURNING id::text
	`, body.Host, body.Port, body.User, strings.TrimSpace(body.HostKey), kind, sealed,
		body.NewAddress, claims.UserID).Scan(&transferID)
	if err != nil {
		writeError(w, http.StatusConflict, "Перенос уже идёт")
		return
	}
	payload, _ := json.Marshal(map[string]string{"transfer_id": transferID})
	if _, err := h.dbOf(ctx).Exec(ctx, `
		INSERT INTO core.jobs (type, status, payload) VALUES ('panel_transfer', 'pending', $1::jsonb)
	`, payload); err != nil {
		writeError(w, http.StatusInternalServerError, "задача переключения узлов не создана")
		return
	}
	jobwake.Notify("panel_transfer")
	audit(ctx, h.dbOf(ctx), claims.UserID, "panel_transfer.agents", transferID,
		map[string]any{"host": body.Host, "new_address": body.NewAddress})
	writeJSON(w, http.StatusAccepted, map[string]any{"id": transferID})
}

func (h *Handler) PostAdminPanelTransferCancel(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.panelTransferOwner(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	tag, err := h.dbOf(ctx).Exec(ctx, `
		UPDATE core.panel_transfers SET status = 'cancelled', target_secret_enc = NULL,
		       finished_at = now(), updated_at = now()
		WHERE status IN ('pending', 'running')
	`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "перенос не отменён")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "Активного переноса нет")
		return
	}
	h.setPanelFreeze(r, false)
	audit(ctx, h.dbOf(ctx), claims.UserID, "panel_transfer.cancel", "panel", nil)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) PostAdminPanelTransferFreeze(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.panelTransferOwner(w, r)
	if !ok {
		return
	}
	var body struct {
		On bool `json:"on"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	h.setPanelFreeze(r, body.On)
	audit(r.Context(), h.dbOf(r.Context()), claims.UserID, "panel_transfer.freeze", "panel",
		map[string]any{"on": body.On})
	writeJSON(w, http.StatusOK, map[string]any{"frozen": body.On})
}

func (h *Handler) setPanelFreeze(r *http.Request, on bool) {
	ctx := r.Context()
	value, _ := json.Marshal(on)
	since, _ := json.Marshal("")
	if on {
		since, _ = json.Marshal(time.Now().UTC().Format(time.RFC3339))
	}
	for key, raw := range map[string][]byte{settingPanelFreeze: value, settingPanelFreezeSince: since} {
		_, _ = h.dbOf(ctx).Exec(ctx, `
			INSERT INTO core.tenant_settings (key, value) VALUES ($1, $2::jsonb)
			ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = now()
		`, key, raw)
	}
	resetFreezeCache()
}

func (h *Handler) dbSchemaVersion(r *http.Request) string {
	var version string
	if h.dbOf(r.Context()).QueryRow(r.Context(),
		`SELECT version FROM core.schema_migrations ORDER BY version DESC LIMIT 1`).Scan(&version) != nil {
		return ""
	}
	return version
}

func (h *Handler) PostAdminPanelTransferExport(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.panelTransferOwner(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	var body struct {
		Password string `json:"password"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if len(body.Password) < 12 {
		writeError(w, http.StatusUnprocessableEntity, "Пароль архива должен быть не короче 12 символов")
		return
	}

	upd := updates.NewUpdater()
	probe, err := upd.TransferProbe(ctx)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	source, err := upd.TransferExport(ctx)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	defer source.Close()

	created := time.Now().UTC().Format(time.RFC3339)
	name := fmt.Sprintf("vortanix-panel-%s.vxt", time.Now().UTC().Format("20060102-150405"))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
	w.WriteHeader(http.StatusOK)

	sealed, err := paneltransfer.Seal(w, body.Password, paneltransfer.Head{
		PanelVersion: buildinfo.Current(),
		CreatedAt:    created,
	})
	if err != nil {
		return
	}
	out := paneltransfer.NewWriter(sealed)
	manifest := &paneltransfer.Manifest{
		PanelVersion:    buildinfo.Current(),
		DBSchemaVersion: h.dbSchemaVersion(r),
		CreatedAt:       created,
		SourceAddress:   probe.SourceAddress,
		Mode:            "export",
		DBBytes:         probe.DBBytes,
		UploadsBytes:    probe.UploadsBytes,
		Contents:        []string{paneltransfer.PartEnv, paneltransfer.PartDB, paneltransfer.PartUploads},
	}
	part, err := out.Part(paneltransfer.PartManifest)
	if err != nil || manifest.Encode(part) != nil {
		return
	}
	if err := copyParts(paneltransfer.NewReader(source), out); err != nil {
		return
	}
	if out.Close() != nil {
		return
	}
	if sealed.Close() != nil {
		return
	}
	audit(ctx, h.dbOf(ctx), claims.UserID, "panel_transfer.export", "panel", map[string]any{
		"db_bytes": probe.DBBytes, "uploads_bytes": probe.UploadsBytes,
	})
}

func copyParts(in *paneltransfer.Reader, out *paneltransfer.Writer) error {
	for {
		name, body, err := in.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if name == paneltransfer.PartManifest {
			_, _ = io.Copy(io.Discard, body)
			continue
		}
		part, err := out.Part(name)
		if err != nil {
			return err
		}
		if _, err := io.Copy(part, body); err != nil {
			return err
		}
	}
}

func (h *Handler) panelTransferArchive(w http.ResponseWriter, r *http.Request) (*paneltransfer.Head, io.Reader, string, bool) {
	password := strings.TrimSpace(r.Header.Get("X-Archive-Password"))
	reader, err := r.MultipartReader()
	if err != nil {
		writeError(w, http.StatusBadRequest, "Загрузите файл архива")
		return nil, nil, "", false
	}
	for {
		part, err := reader.NextPart()
		if err != nil {
			writeError(w, http.StatusBadRequest, "В запросе нет файла архива")
			return nil, nil, "", false
		}
		if part.FormName() == "password" {
			raw, _ := io.ReadAll(io.LimitReader(part, 1024))
			password = strings.TrimSpace(string(raw))
			continue
		}
		if part.FormName() != "file" {
			continue
		}
		head, body, err := paneltransfer.Open(part, password)
		if err != nil {
			status := http.StatusUnprocessableEntity
			if errors.Is(err, paneltransfer.ErrPassword) {
				status = http.StatusForbidden
			}
			writeError(w, status, err.Error())
			return nil, nil, "", false
		}
		return head, body, password, true
	}
}

func (h *Handler) PostAdminPanelTransferInspect(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.panelTransferOwner(w, r); !ok {
		return
	}
	head, body, _, ok := h.panelTransferArchive(w, r)
	if !ok {
		return
	}
	in := paneltransfer.NewReader(body)
	name, part, err := in.Next()
	if err != nil || name != paneltransfer.PartManifest {
		writeError(w, http.StatusUnprocessableEntity, "В архиве нет описания")
		return
	}
	manifest, err := paneltransfer.DecodeManifest(part)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	out := map[string]any{"head": head, "manifest": manifest, "version": buildinfo.Current()}
	if err := paneltransfer.CheckCompatible(manifest, buildinfo.Current()); err != nil {
		out["incompatible"] = err.Error()
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) PostAdminPanelTransferImport(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.panelTransferOwner(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	head, body, _, ok := h.panelTransferArchive(w, r)
	if !ok {
		return
	}
	in := paneltransfer.NewReader(body)
	name, part, err := in.Next()
	if err != nil || name != paneltransfer.PartManifest {
		writeError(w, http.StatusUnprocessableEntity, "В архиве нет описания")
		return
	}
	manifest, err := paneltransfer.DecodeManifest(part)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	if err := paneltransfer.CheckCompatible(manifest, buildinfo.Current()); err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	pr, pw := io.Pipe()
	go func() {
		out := paneltransfer.NewWriter(pw)
		err := copyParts(in, out)
		if err == nil {
			err = out.Close()
		}
		_ = pw.CloseWithError(err)
	}()

	stream, err := updates.NewUpdater().TransferImport(ctx, pr)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	defer stream.Close()

	audit(ctx, h.dbOf(ctx), claims.UserID, "panel_transfer.import", "panel", map[string]any{
		"panel_version": head.PanelVersion, "created_at": head.CreatedAt,
	})
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	flusher, _ := w.(http.Flusher)
	buf := make([]byte, 4096)
	for {
		n, err := stream.Read(buf)
		if n > 0 {
			_, _ = w.Write(buf[:n])
			if flusher != nil {
				flusher.Flush()
			}
		}
		if err != nil {
			return
		}
	}
}
