package handlers

import (
	"errors"
	"regexp"
	"strings"

	"github.com/vortanixapp/panel/pkg/gamecatalog"
)

const (
	runtimeVersionsMetaKey = "runtime_versions"
	runtimeVersionsLimit   = 24
	runtimeVersionURLLimit = 500
)

var (
	runtimeVersionPattern = regexp.MustCompile(`^[0-9][0-9A-Za-z._-]{0,23}$`)

	errRuntimeVersionName   = errors.New("Версия среды указана неверно")
	errRuntimeVersionURL    = errors.New("Ссылка на сборку должна начинаться с http:// или https://")
	errRuntimeVersionDouble = errors.New("Версия среды повторяется")
	errRuntimeVersionCount  = errors.New("Слишком много версий среды")
	errRuntimeVersionSource = errors.New("Для этой среды своя ссылка на сборку не поддерживается")
)

type runtimeVersionEntry struct {
	Version string `json:"version"`
	URL     string `json:"url"`
	Enabled bool   `json:"enabled"`
}

func defaultRuntimeVersions(sel gamecatalog.RuntimeSelector) []runtimeVersionEntry {
	out := make([]runtimeVersionEntry, 0, len(sel.Versions))
	for _, v := range sel.Versions {
		out = append(out, runtimeVersionEntry{Version: v, Enabled: true})
	}
	return out
}

func runtimeVersionsOf(meta map[string]any, sel gamecatalog.RuntimeSelector) []runtimeVersionEntry {
	list, ok := meta[runtimeVersionsMetaKey].([]any)
	if !ok || len(list) == 0 {
		return defaultRuntimeVersions(sel)
	}
	out := make([]runtimeVersionEntry, 0, len(list))
	seen := map[string]bool{}
	for _, raw := range list {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		version := strings.TrimSpace(anyString(item["version"]))
		if version == "" || seen[version] || !runtimeVersionPattern.MatchString(version) {
			continue
		}
		seen[version] = true
		entry := runtimeVersionEntry{Version: version, Enabled: true}
		entry.URL = strings.TrimSpace(anyString(item["url"]))
		if sel.URLFile == "" {
			entry.URL = ""
		}
		if on, ok := bodyBoolVal(item["enabled"]); ok {
			entry.Enabled = on
		}
		out = append(out, entry)
	}
	if len(out) == 0 {
		return defaultRuntimeVersions(sel)
	}
	return out
}

func parseRuntimeVersionsBody(raw any, sel gamecatalog.RuntimeSelector) ([]runtimeVersionEntry, error) {
	list, ok := raw.([]any)
	if !ok {
		return nil, errRuntimeVersionName
	}
	if len(list) > runtimeVersionsLimit {
		return nil, errRuntimeVersionCount
	}
	out := make([]runtimeVersionEntry, 0, len(list))
	seen := map[string]bool{}
	for _, item := range list {
		fields, ok := item.(map[string]any)
		if !ok {
			return nil, errRuntimeVersionName
		}
		version := strings.TrimSpace(anyString(fields["version"]))
		if !runtimeVersionPattern.MatchString(version) {
			return nil, errRuntimeVersionName
		}
		if seen[version] {
			return nil, errRuntimeVersionDouble
		}
		seen[version] = true

		entry := runtimeVersionEntry{Version: version, Enabled: true}
		if on, ok := bodyBoolVal(fields["enabled"]); ok {
			entry.Enabled = on
		}
		url := strings.TrimSpace(anyString(fields["url"]))
		if url != "" {
			if sel.URLFile == "" {
				return nil, errRuntimeVersionSource
			}
			if len(url) > runtimeVersionURLLimit ||
				!(strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")) {
				return nil, errRuntimeVersionURL
			}
			entry.URL = url
		}
		out = append(out, entry)
	}
	return out, nil
}

func runtimeVersionsJSON(list []runtimeVersionEntry) []map[string]any {
	out := make([]map[string]any, 0, len(list))
	for _, item := range list {
		out = append(out, map[string]any{
			"version": item.Version,
			"url":     item.URL,
			"enabled": item.Enabled,
		})
	}
	return out
}

func enabledRuntimeVersions(list []runtimeVersionEntry) []string {
	out := make([]string, 0, len(list))
	for _, item := range list {
		if item.Enabled {
			out = append(out, item.Version)
		}
	}
	return out
}

func findRuntimeVersion(list []runtimeVersionEntry, version string) (runtimeVersionEntry, bool) {
	for _, item := range list {
		if item.Version == version && item.Enabled {
			return item, true
		}
	}
	return runtimeVersionEntry{}, false
}

func isRuntimeVersionError(err error) bool {
	for _, known := range []error{
		errRuntimeVersionName, errRuntimeVersionURL,
		errRuntimeVersionDouble, errRuntimeVersionCount, errRuntimeVersionSource,
	} {
		if errors.Is(err, known) {
			return true
		}
	}
	return false
}
