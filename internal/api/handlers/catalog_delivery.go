package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

const catalogArchiveTokenTTL = 30 * time.Minute

type catalogArchiveToken struct {
	Tenant string `json:"t"`
	Kind   string `json:"k"`
	ID     string `json:"i"`
	Exp    int64  `json:"e"`
}

func (h *Handler) signCatalogArchive(tenantID, kind, id string) (string, error) {
	body, err := json.Marshal(catalogArchiveToken{
		Tenant: tenantID,
		Kind:   kind,
		ID:     id,
		Exp:    time.Now().Add(catalogArchiveTokenTTL).Unix(),
	})
	if err != nil {
		return "", err
	}
	payload := base64.RawURLEncoding.EncodeToString(body)
	mac := hmac.New(sha256.New, []byte(h.jwtSecret))
	mac.Write([]byte(payload))
	return payload + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

func (h *Handler) parseCatalogArchiveToken(raw string) (catalogArchiveToken, error) {
	var out catalogArchiveToken
	payload, sig, ok := strings.Cut(strings.TrimSpace(raw), ".")
	if !ok || payload == "" || sig == "" {
		return out, fmt.Errorf("некорректная ссылка")
	}
	mac := hmac.New(sha256.New, []byte(h.jwtSecret))
	mac.Write([]byte(payload))
	want := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(want), []byte(sig)) {
		return out, fmt.Errorf("подпись не сходится")
	}
	body, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return out, fmt.Errorf("некорректная ссылка")
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return out, fmt.Errorf("некорректная ссылка")
	}
	if out.Exp <= time.Now().Unix() {
		return out, fmt.Errorf("ссылка просрочена")
	}
	if out.Kind != catalogPlugins && out.Kind != catalogMaps {
		return out, fmt.Errorf("неизвестный вид записи")
	}
	return out, nil
}

func (h *Handler) ServeCatalogArchive(w http.ResponseWriter, r *http.Request) {
	claims, err := h.parseCatalogArchiveToken(r.URL.Query().Get("token"))
	if err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}
	pool := h.db
	table := "core.plugins"
	if claims.Kind == catalogMaps {
		table = "core.maps"
	}
	var rel string
	if err := pool.QueryRow(r.Context(),
		//nolint:gosec
		"SELECT COALESCE(archive_path, '') FROM "+table+" WHERE id = $1::uuid AND tenant_id = $2",
		claims.ID, claims.Tenant).Scan(&rel); err != nil || rel == "" {
		writeError(w, http.StatusNotFound, "архив не найден")
		return
	}
	abs, err := h.resolveCatalogPath(rel)
	if err != nil {
		writeError(w, http.StatusNotFound, "архив не найден")
		return
	}
	info, statErr := os.Stat(abs)
	if statErr != nil || info.IsDir() {
		writeError(w, http.StatusNotFound, "архив не найден")
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeFile(w, r, abs)
}

func catalogCachePath(kind, id string, size int64) string {
	return fmt.Sprintf("%s/%s-%d.archive", kind, id, size)
}

func (h *Handler) deliverCatalogArchive(r *http.Request, tenantID, nodeID, kind, itemID string) (string, error) {
	table := "core.plugins"
	if kind == catalogMaps {
		table = "core.maps"
	}
	var rel string
	var size int64
	err := h.dbOf(r.Context()).QueryRow(r.Context(),
		//nolint:gosec
		"SELECT COALESCE(archive_path, ''), COALESCE(archive_size, 0) FROM "+table+" WHERE id = $1::uuid AND tenant_id = $2",
		itemID, tenantID).Scan(&rel, &size)
	if err != nil {
		return "", fmt.Errorf("запись каталога не найдена")
	}
	if rel == "" {
		return "", nil
	}

	token, err := h.signCatalogArchive(tenantID, kind, itemID)
	if err != nil {
		return "", err
	}
	cachePath := catalogCachePath(kind, itemID, size)
	payload := map[string]any{
		"cache_path":   cachePath,
		"download_url": h.publicBaseURL(r) + "/v1/internal/catalog-archive?token=" + token,
		"size":         size,
	}
	if _, err := h.agentCommand(r.Context(), nodeID, "", "archive_cache_fetch", payload); err != nil {
		return "", fmt.Errorf("не удалось доставить архив на локацию: %w", err)
	}

	_, _ = h.dbOf(r.Context()).Exec(r.Context(),
		//nolint:gosec
		"UPDATE "+table+" SET archive_location_id = $3::uuid WHERE id = $1::uuid AND tenant_id = $2",
		itemID, tenantID, nodeID)
	return cachePath, nil
}
