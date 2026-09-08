package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	updateCacheKey = "updates:latest"
	updateCacheTTL = time.Hour
)

type releaseInfo struct {
	Version     string `json:"version"`
	Notes       string `json:"notes"`
	URL         string `json:"url"`
	PublishedAt string `json:"published_at"`
	Prerelease  bool   `json:"prerelease"`
	CheckedAt   string `json:"checked_at"`
}

func updateEnv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func updateRepo() string {
	return strings.Trim(updateEnv("UPDATE_REPO", "vortanixapp/panel"), "/")
}

func panelVersion() string {
	return updateEnv("VORTANIX_VERSION", "dev")
}

func (h *Handler) AdminUpdates(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok || !isStaffRole(claims.Role) {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	repo := updateRepo()
	current := panelVersion()

	if repo == "" {
		writeJSON(w, http.StatusOK, map[string]any{
			"current_version":  current,
			"checks_disabled":  true,
			"update_available": false,
		})
		return
	}

	ctx := r.Context()
	var rel releaseInfo
	cached := false
	if r.URL.Query().Get("refresh") != "1" {
		if found, err := h.cache.GetJSON(ctx, updateCacheKey, &rel); err == nil && found {
			cached = true
		}
	}

	if !cached {
		fetched, err := fetchLatestRelease(ctx, repo)
		if err != nil {
			writeJSON(w, http.StatusOK, map[string]any{
				"current_version":  current,
				"repo":             repo,
				"update_available": false,
				"error":            err.Error(),
			})
			return
		}
		rel = *fetched
		if err := h.cache.SetJSON(ctx, updateCacheKey, rel, updateCacheTTL); err != nil {
			log.Printf("проверка обновлений: ответ не закеширован: %v", err)
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"current_version":  current,
		"latest_version":   rel.Version,
		"update_available": isNewerVersion(rel.Version, current),
		"notes":            rel.Notes,
		"release_url":      rel.URL,
		"published_at":     rel.PublishedAt,
		"prerelease":       rel.Prerelease,
		"checked_at":       rel.CheckedAt,
		"from_cache":       cached,
		"repo":             repo,
	})
}

func fetchLatestRelease(ctx context.Context, repo string) (*releaseInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	base := strings.TrimRight(updateEnv("UPDATE_API_BASE", "https://api.github.com"), "/")
	url := base + "/repos/" + repo + "/releases/latest"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "vortanix-panel")
	if token := strings.TrimSpace(os.Getenv("UPDATE_GITHUB_TOKEN")); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("github недоступен: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound:
		return nil, fmt.Errorf("у репозитория %s нет ни одного выпуска", repo)
	case http.StatusForbidden:
		return nil, fmt.Errorf("github ограничил частоту запросов, попробуйте позже")
	default:
		return nil, fmt.Errorf("github ответил %s", resp.Status)
	}

	var payload struct {
		TagName     string `json:"tag_name"`
		Body        string `json:"body"`
		HTMLURL     string `json:"html_url"`
		PublishedAt string `json:"published_at"`
		Prerelease  bool   `json:"prerelease"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	return &releaseInfo{
		Version:     strings.TrimPrefix(strings.TrimSpace(payload.TagName), "v"),
		Notes:       payload.Body,
		URL:         payload.HTMLURL,
		PublishedAt: payload.PublishedAt,
		Prerelease:  payload.Prerelease,
		CheckedAt:   time.Now().UTC().Format(time.RFC3339),
	}, nil
}

// isNewerVersion сравнивает по числовым частям, а не строкой: строкой "0.10.0"
// оказывается меньше "0.9.0", и обновление перестало бы предлагаться ровно на
// десятом выпуске.
func isNewerVersion(latest, current string) bool {
	l, okL := parseVersion(latest)
	c, okC := parseVersion(current)
	if !okL || !okC {
		return false
	}
	for i := 0; i < 3; i++ {
		if l[i] != c[i] {
			return l[i] > c[i]
		}
	}
	return false
}

func parseVersion(v string) ([3]int, bool) {
	var out [3]int
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i]
	}
	parts := strings.Split(v, ".")
	if len(parts) == 0 || len(parts) > 3 {
		return out, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return out, false
		}
		out[i] = n
	}
	return out, true
}
