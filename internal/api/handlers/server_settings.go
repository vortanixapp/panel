package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/vortanixapp/panel/pkg/gamecatalog"
	"github.com/vortanixapp/panel/pkg/gamesettings"
)

const secretSentinel = "__vtx_secret__"

type settingsFileOut struct {
	ID        string `json:"id"`
	Path      string `json:"path"`
	Title     string `json:"title"`
	Format    string `json:"format"`
	Exists    bool   `json:"exists"`
	Size      int    `json:"size"`
	Editable  bool   `json:"editable"`
	Truncated bool   `json:"truncated,omitempty"`
	SHA256    string `json:"sha256,omitempty"`
	Error     string `json:"error,omitempty"`
}

type settingsFieldOut struct {
	Key         string                `json:"key"`
	Label       string                `json:"label"`
	Hint        string                `json:"hint,omitempty"`
	Section     string                `json:"section"`
	Kind        string                `json:"kind"`
	Options     []gamesettings.Option `json:"options,omitempty"`
	Default     string                `json:"default,omitempty"`
	Min         *float64              `json:"min,omitempty"`
	Max         *float64              `json:"max,omitempty"`
	MaxLen      int                   `json:"max_len,omitempty"`
	AppliesLive bool                  `json:"applies_live,omitempty"`
	ReadOnly    bool                  `json:"read_only,omitempty"`
	Secret      bool                  `json:"secret,omitempty"`
	Slots       bool                  `json:"slots,omitempty"`
	Clearable   bool                  `json:"clearable,omitempty"`
}

type settingsSectionOut struct {
	ID     string             `json:"id"`
	Title  string             `json:"title"`
	Hint   string             `json:"hint,omitempty"`
	Fields []settingsFieldOut `json:"fields"`
}

type settingsState struct {
	profile  *gamesettings.Profile
	row      *serverGameSettingsRow
	startup  string
	paths    map[string]string
	contents map[string]string
	present  map[string]bool
	nodeSeen bool
	nodeUsed bool
}

func (h *Handler) loadSettingsState(
	ctx context.Context, serverID string,
	profile *gamesettings.Profile, row *serverGameSettingsRow,
) *settingsState {
	st := &settingsState{
		profile:  profile,
		row:      row,
		startup:  effectiveStartupParams(row.Config),
		paths:    map[string]string{},
		contents: map[string]string{},
		present:  map[string]bool{},
	}

	stored := extractStoredGameSettings(row.Config, profile.Key)
	for _, f := range profile.Files {
		if f.IsStartup() {
			st.paths[f.ID] = gamesettings.StartupPath
			st.contents[f.ID] = st.startup
			st.present[f.ID] = true
		}
	}
	resolver := st.pathValues(stored)

	for _, f := range profile.Files {
		if f.IsStartup() {
			continue
		}
		path := resolveSettingsTemplate(f.Path, resolver)
		st.paths[f.ID] = path
		st.nodeUsed = true
		content, ok := h.agentFilesReadQuiet(ctx, serverID, path)
		if ok {
			st.nodeSeen = true
			st.contents[f.ID] = content
			st.present[f.ID] = true
		}
	}
	return st
}

func (st *settingsState) pathValues(stored map[string]any) map[string]string {
	out := map[string]string{}
	for _, field := range st.profile.Fields {
		if v, ok := stored[field.Key]; ok {
			out[field.Key] = toString(v)
		}
	}
	if args, ok := gamesettings.CodecFor(gamesettings.FormatArgs); ok {
		for prop, v := range args.Parse(st.startup) {
			for _, field := range st.profile.Fields {
				if field.Prop == prop && v != "" {
					out[field.Key] = v
				}
			}
		}
	}
	for _, field := range st.profile.Fields {
		if out[field.Key] == "" && field.Default != "" {
			out[field.Key] = field.Default
		}
	}
	return out
}

func resolveSettingsTemplate(path string, values map[string]string) string {
	if !strings.ContainsRune(path, '{') {
		return path
	}
	for key, val := range values {
		if val == "" {
			continue
		}
		path = strings.ReplaceAll(path, "{"+key+"}", val)
	}
	for strings.Contains(path, "{") {
		open := strings.IndexByte(path, '{')
		close := strings.IndexByte(path[open:], '}')
		if close < 0 {
			break
		}
		path = path[:open] + path[open+close+1:]
	}
	return path
}

func (st *settingsState) fieldValues() (map[string]string, map[string]string) {
	stored := extractStoredGameSettings(st.row.Config, st.profile.Key)

	parsed := map[string]map[string]string{}
	for _, f := range st.profile.Files {
		content, ok := st.contents[f.ID]
		if !ok {
			continue
		}
		if vals, err := gamesettings.ParseFile(f, content); err == nil {
			parsed[f.ID] = vals
		}
	}

	values := map[string]string{}
	sources := map[string]string{}
	for _, field := range st.profile.Fields {
		fileID := field.File
		if fileID == "" && len(st.profile.Files) > 0 {
			fileID = st.profile.Files[0].ID
		}
		if vals, ok := parsed[fileID]; ok {
			if v, has := vals[field.Prop]; has {
				values[field.Key] = gamesettings.NormalizeForForm(field, v)
				sources[field.Key] = "file"
				continue
			}
		}
		if v, ok := stored[field.Key]; ok {
			values[field.Key] = gamesettings.NormalizeForForm(field, toString(v))
			sources[field.Key] = "stored"
			continue
		}
		values[field.Key] = field.Default
		sources[field.Key] = "default"
	}
	return values, sources
}

func (st *settingsState) slotsLimit() int {
	if st.row.BillingType == "slots" && st.row.Slots > 0 {
		return st.row.Slots
	}
	return 0
}

func (h *Handler) resolveSettingsProfile(
	w http.ResponseWriter, gameID string,
) (*gamesettings.Profile, bool) {
	profile, ok := gamesettings.For(gameID)
	if ok {
		return profile, true
	}
	reason := "Для этой игры настройки пока не описаны."
	if !gamecatalog.LinuxSupported(gameID) {
		if note := gamecatalog.UnsupportedReason(gameID); note != "" {
			reason = note
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok": true, "game": gameID, "supported": false, "reason": reason,
		"node_online": true, "sections": []any{}, "values": map[string]string{},
		"files": []any{},
	})
	return nil, false
}

func (h *Handler) ServerSettingsSchema(w http.ResponseWriter, r *http.Request) {
	serverID := chi.URLParam(r, "id")
	claims, ok := h.ensureServerForTenant(w, r, serverID)
	if !ok {
		return
	}
	if !h.authorizeServerAction(w, r, claims, serverID, "settings_read") {
		return
	}
	ctx := r.Context()
	row, err := h.loadServerGameSettingsRow(ctx, serverID)
	if err != nil {
		writeError(w, http.StatusNotFound, "server not found")
		return
	}
	profile, ok := h.resolveSettingsProfile(w, row.GameID)
	if !ok {
		return
	}

	st := h.loadSettingsState(ctx, serverID, profile, row)
	values, sources := st.fieldValues()
	slots := st.slotsLimit()

	sections := []settingsSectionOut{}
	for _, group := range profile.GroupBySection() {
		out := settingsSectionOut{
			ID: string(group.Section.ID), Title: group.Section.Title,
			Fields: make([]settingsFieldOut, 0, len(group.Fields)),
		}
		for _, f := range group.Fields {
			item := settingsFieldOut{
				Key: f.Key, Label: f.Label, Hint: f.Hint,
				Section: string(f.Section), Kind: string(f.Kind),
				Options: f.Options, Default: f.Default,
				Min: f.Min, Max: f.Max, MaxLen: f.MaxLen,
				AppliesLive: f.AppliesLive, ReadOnly: f.ReadOnly,
				Secret: f.Secret, Slots: f.Slots, Clearable: f.Clearable,
			}
			if f.Slots && slots > 0 {
				item.ReadOnly = true
				item.Hint = fmt.Sprintf("Ограничено тарифом: %d", slots)
			}
			if f.Secret {
				if values[f.Key] != "" {
					values[f.Key] = secretSentinel
				}
			}
			out.Fields = append(out.Fields, item)
		}
		sections = append(sections, out)
	}
	if slots > 0 {
		for _, f := range profile.Fields {
			if f.Slots {
				values[f.Key] = fmt.Sprint(slots)
			}
		}
	}

	files := make([]settingsFileOut, 0, len(profile.Files))
	for _, f := range profile.Files {
		if f.IsStartup() {
			continue
		}
		content, present := st.contents[f.ID]
		item := settingsFileOut{
			ID: f.ID, Path: st.paths[f.ID], Title: f.Title, Format: string(f.Format),
			Exists: present, Size: len(content), Editable: true,
		}
		if len(content) > f.MaxFileBytes() {
			item.Truncated = true
			item.Editable = false
		}
		if present {
			item.SHA256 = contentDigest(content)
		} else if st.nodeUsed && !st.nodeSeen {
			item.Error = "Нода не на связи"
		}
		files = append(files, item)
	}

	warnings := []string{}
	if st.nodeUsed && !st.nodeSeen {
		warnings = append(warnings,
			"Нода не отвечает: показаны сохранённые значения, а не то, что сейчас в файлах на сервере.")
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok": true, "game": row.GameID, "profile": profile.Key,
		"supported": true, "node_online": !st.nodeUsed || st.nodeSeen,
		"sections": sections, "values": values, "value_sources": sources,
		"files": files, "note": profile.Note, "warnings": warnings,
	})
}

func contentDigest(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}
