package handlers

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/vortanixapp/panel/pkg/buildinfo"
	"github.com/vortanixapp/panel/pkg/updates"
)

const (
	bugReportTemplate     = "panel_report.yml"
	bugReportURLLimit     = 7000
	bugReportLogWindow    = time.Hour
	bugReportJobLimit     = 15
	bugReportActionLimit  = 20
	bugReportSimilarLimit = 5
	bugReportSimilarTTL   = 10 * time.Minute
	bugReportTitleMax     = 200
	bugReportDescMax      = 6000
	bugReportClientMax    = 120
	bugReportErrorMax     = 300
	bugReportAuditAction  = "admin.bug_report.create"
)

const bugReportPastePlaceholder = "Описание не поместилось в ссылку: вставьте его сюда из буфера обмена (Ctrl+V).\n" +
	"The description did not fit into the link: paste it here from the clipboard (Ctrl+V)."

var bugReportSeverities = map[string]string{
	"low":      "Low: cosmetic, does not get in the way",
	"medium":   "Medium: there is a workaround",
	"high":     "High: a feature does not work",
	"critical": "Critical: the panel or nodes are down",
}

var bugReportComponents = map[string]string{
	"panel-ui":        "panel web interface",
	"core-api":        "panel API",
	"worker":          "background jobs",
	"agent":           "node agent",
	"agent-relay":     "agent relay",
	"console-gateway": "server console",
	"metrics-ingest":  "metrics",
	"status-page":     "status page",
	"updater":         "updates",
	"game-image":      "game image",
	"unknown":         "not sure",
}

var bugReportLabels = map[string]string{
	"description": "What happened",
	"severity":    "Severity",
	"component":   "Component",
	"version":     "Panel version",
	"environment": "Environment",
	"logs":        "Logs",
}

var bugReportStopWords = map[string]bool{
	"когда": true, "после": true, "перед": true, "через": true, "может": true, "можно": true,
	"нельзя": true, "нужно": true, "очень": true, "этот": true, "этой": true, "этого": true,
	"этом": true, "если": true, "тоже": true, "также": true, "только": true, "всегда": true,
	"иногда": true, "почему": true, "какой": true, "какая": true, "какие": true, "который": true,
	"которая": true, "которые": true, "есть": true, "было": true, "была": true, "были": true,
	"будет": true, "меня": true, "свой": true, "свою": true, "своей": true, "вообще": true,
	"сейчас": true, "снова": true, "опять": true, "ошибка": true, "ошибки": true, "ошибку": true,
	"ошибкой": true, "проблема": true, "проблемы": true, "работает": true, "работают": true,
	"панель": true, "панели": true, "страница": true, "странице": true, "страницы": true,
	"the": true, "and": true, "for": true, "not": true, "can": true, "but": true, "was": true,
	"are": true, "its": true, "you": true, "has": true, "had": true, "all": true, "any": true,
	"out": true, "off": true, "get": true, "set": true, "new": true, "old": true, "why": true,
	"how": true, "who": true, "use": true, "when": true, "after": true, "before": true,
	"with": true, "from": true, "that": true, "this": true, "does": true, "doesn": true,
	"cannot": true, "into": true, "than": true, "then": true, "there": true, "error": true,
	"errors": true, "issue": true, "problem": true, "panel": true, "page": true, "work": true,
	"works": true, "working": true, "broken": true, "email": true, "hidden": true,
}

var (
	reportCredentials = regexp.MustCompile(`(?i)\b([a-z][a-z0-9+.\-]*://)[^\s/:@]+:[^\s/@]+@`)
	reportBearer      = regexp.MustCompile(`(?i)\bbearer\s+[A-Za-z0-9._~+/=\-]+`)
	reportSecret      = regexp.MustCompile(`(?i)\b(token|password|passwd|pwd|secret|api[_-]?key|access[_-]?key|private[_-]?key)(\s*[:=]\s*)("[^"]*"|'[^']*'|[^\s,;]+)`)
	reportEmail       = regexp.MustCompile(`[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}`)
	reportIPv4        = regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`)
	reportIPv6        = regexp.MustCompile(`(?i)\b(?:[0-9a-f]{1,4}:){3,7}[0-9a-f]{1,4}\b|\b(?:[0-9a-f]{1,4}:){1,6}:[0-9a-f]{1,4}\b`)
	reportLongToken   = regexp.MustCompile(`[A-Za-z0-9_\-]{32,}|[A-Za-z0-9+/]{40,}={0,2}`)
)

type bugReportField struct {
	ID    string `json:"id"`
	Value string `json:"value"`
}

type bugReportAgent struct {
	Version string `json:"version"`
	Count   int    `json:"count"`
}

type bugReportNodes struct {
	Total  int              `json:"total"`
	Online int              `json:"online"`
	Agents []bugReportAgent `json:"agents"`
}

type bugReportNode struct {
	Version  string
	Online   bool
	Platform string
}

type bugReportClient struct {
	UIVersion string `json:"ui_version"`
	Browser   string `json:"browser"`
	Screen    string `json:"screen"`
	Language  string `json:"language"`
	Time      string `json:"time"`
}

type bugReportRequest struct {
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Severity    string          `json:"severity"`
	Component   string          `json:"component"`
	NodeID      string          `json:"node_id"`
	AttachLogs  bool            `json:"attach_logs"`
	Client      bugReportClient `json:"client"`
}

type bugReport struct {
	Repo      string           `json:"repo"`
	Title     string           `json:"title"`
	URL       string           `json:"url"`
	Fields    []bugReportField `json:"fields"`
	Text      string           `json:"text"`
	Clipboard string           `json:"clipboard"`
	LogsLines int              `json:"logs_lines"`
	LogsTotal int              `json:"logs_total"`
}

func scrubReportText(s string) string {
	s = reportCredentials.ReplaceAllString(s, "${1}[hidden]@")
	s = reportBearer.ReplaceAllString(s, "Bearer [hidden]")
	s = reportSecret.ReplaceAllString(s, "${1}${2}[hidden]")
	s = reportEmail.ReplaceAllString(s, "[email]")
	s = reportIPv4.ReplaceAllString(s, "[ip]")
	s = reportIPv6.ReplaceAllString(s, "[ip]")
	return reportLongToken.ReplaceAllStringFunc(s, func(m string) string {
		if strings.ContainsAny(m, "0123456789") && strings.IndexFunc(m, unicode.IsLetter) >= 0 {
			return "[hidden]"
		}
		return m
	})
}

func bugReportLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func bugReportClamp(s string, max int) string {
	s = strings.TrimSpace(s)
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	return strings.TrimSpace(string([]rune(s)[:max])) + "…"
}

func bugReportClientValue(s string) string {
	return bugReportClamp(scrubReportText(bugReportLine(s)), bugReportClientMax)
}

func bugReportTag(v string) string {
	v = buildinfo.Normalize(v)
	if updates.IsSemver(v) {
		return "v" + v
	}
	return v
}

func bugReportRepo() string {
	if repo := updates.Repo(); repo != "" {
		return repo
	}
	return updates.DefaultRepo
}

func bugReportGitHub(repo, path string) string {
	return "https://github.com/" + repo + path
}

func bugReportStaff(w http.ResponseWriter, r *http.Request) (string, bool) {
	claims, ok := tenantClaims(r.Context())
	if !ok || !isStaffRole(claims.Role) {
		writeError(w, http.StatusForbidden, "forbidden")
		return "", false
	}
	return claims.UserID, true
}

func bugReportWords(query string) []string {
	seen := map[string]bool{}
	words := []string{}
	fields := strings.FieldsFunc(strings.ToLower(scrubReportText(query)), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	for _, word := range fields {
		size := utf8.RuneCountInString(word)
		short := size < 3 || (size == 3 && size != len(word))
		if short || size > 40 || bugReportStopWords[word] || seen[word] {
			continue
		}
		seen[word] = true
		words = append(words, word)
		if len(words) == 5 {
			break
		}
	}
	return words
}

func (h *Handler) bugReportDatabase(ctx context.Context) string {
	var version string
	if err := h.readerOf(ctx).QueryRow(ctx, `SELECT current_setting('server_version')`).Scan(&version); err != nil {
		return ""
	}
	if fields := strings.Fields(version); len(fields) > 0 {
		return "PostgreSQL " + fields[0]
	}
	return ""
}

func (h *Handler) bugReportNodes(ctx context.Context, nodeID string) (bugReportNodes, *bugReportNode) {
	summary := bugReportNodes{Agents: []bugReportAgent{}}
	rows, err := h.readerOf(ctx).Query(ctx, `
		SELECT n.id::text, COALESCE(d.status, 'unknown'), COALESCE(d.version, ''), d.last_seen_at, COALESCE(d.platform, '')
		FROM core.nodes n
		LEFT JOIN core.node_daemons d ON d.node_id = n.id
	`)
	if err != nil {
		return summary, nil
	}
	defer rows.Close()

	counts := map[string]int{}
	var selected *bugReportNode
	for rows.Next() {
		var id, status, version, platform string
		var lastSeen *time.Time
		if rows.Scan(&id, &status, &version, &lastSeen, &platform) != nil {
			continue
		}
		online := agentDaemonOnline(status, lastSeen)
		version = bugReportTag(version)
		summary.Total++
		if online {
			summary.Online++
		}
		if version != "" {
			counts[version]++
		}
		if nodeID != "" && id == nodeID {
			selected = &bugReportNode{Version: version, Online: online, Platform: bugReportClientValue(platform)}
		}
	}
	for version, count := range counts {
		summary.Agents = append(summary.Agents, bugReportAgent{Version: version, Count: count})
	}
	sort.Slice(summary.Agents, func(i, j int) bool {
		if summary.Agents[i].Count != summary.Agents[j].Count {
			return summary.Agents[i].Count > summary.Agents[j].Count
		}
		return summary.Agents[i].Version > summary.Agents[j].Version
	})
	return summary, selected
}

func (h *Handler) bugReportLogs(ctx context.Context, userID string) ([]string, []string) {
	since := time.Now().Add(-bugReportLogWindow)
	jobs := []string{}
	rows, err := h.readerOf(ctx).Query(ctx, `
		SELECT type, attempts, updated_at, COALESCE(result, '{}'::jsonb), COALESCE(error_message, '')
		FROM core.jobs
		WHERE status = 'failed' AND updated_at >= $1
		ORDER BY updated_at DESC
		LIMIT $2
	`, since, bugReportJobLimit)
	if err == nil {
		for rows.Next() {
			var jobType, message string
			var attempts int
			var at time.Time
			var result []byte
			if rows.Scan(&jobType, &attempts, &at, &result, &message) != nil {
				continue
			}
			if text := jobErrorText(result); text != "" {
				message = text
			}
			message = bugReportClamp(scrubReportText(bugReportLine(message)), bugReportErrorMax)
			if message == "" {
				message = "no error text"
			}
			jobs = append(jobs, fmt.Sprintf("%s  %s  attempts %d  %s", at.UTC().Format("15:04:05"), jobType, attempts, message))
		}
		rows.Close()
	}

	actions := []string{}
	rows, err = h.readerOf(ctx).Query(ctx, `
		SELECT action, created_at
		FROM core.audit_logs
		WHERE user_id = $1::uuid AND created_at >= $2 AND action <> $3
		ORDER BY created_at DESC
		LIMIT $4
	`, userID, since, bugReportAuditAction, bugReportActionLimit)
	if err == nil {
		for rows.Next() {
			var action string
			var at time.Time
			if rows.Scan(&action, &at) != nil {
				continue
			}
			actions = append(actions, at.UTC().Format("15:04:05")+"  "+scrubReportText(bugReportLine(action)))
		}
		rows.Close()
	}

	reverseStrings(jobs)
	reverseStrings(actions)
	return jobs, actions
}

func reverseStrings(list []string) {
	for i, j := 0, len(list)-1; i < j; i, j = i+1, j-1 {
		list[i], list[j] = list[j], list[i]
	}
}

func bugReportLogText(jobs, actions []string, keep int) string {
	if keep <= 0 {
		return ""
	}
	window := int(bugReportLogWindow / time.Minute)
	tail := func(list []string) []string {
		if len(list) > keep {
			return list[len(list)-keep:]
		}
		return list
	}
	parts := []string{}
	if len(jobs) > 0 {
		parts = append(parts, fmt.Sprintf("Failed jobs, last %d min (UTC):\n", window)+strings.Join(tail(jobs), "\n"))
	}
	if len(actions) > 0 {
		parts = append(parts, fmt.Sprintf("Admin actions, last %d min (UTC):\n", window)+strings.Join(tail(actions), "\n"))
	}
	return strings.Join(parts, "\n\n")
}

func bugReportEnvironment(client bugReportClient, database string, nodes bugReportNodes, node *bugReportNode) string {
	lines := []string{}
	add := func(label, value string) {
		if value != "" {
			lines = append(lines, "- "+label+": "+value)
		}
	}
	add("Browser", bugReportClientValue(client.Browser))
	add("Screen", bugReportClientValue(client.Screen))
	add("Interface language", bugReportClientValue(client.Language))
	add("Reported at", bugReportClientValue(client.Time))
	add("Database", database)
	add("Nodes", fmt.Sprintf("%d, online %d", nodes.Total, nodes.Online))
	if len(nodes.Agents) > 0 {
		parts := make([]string, 0, len(nodes.Agents))
		for _, agent := range nodes.Agents {
			parts = append(parts, fmt.Sprintf("%s ×%d", agent.Version, agent.Count))
		}
		add("Agents", strings.Join(parts, ", "))
	}
	if node == nil {
		add("Reproduces on", "all nodes")
	} else {
		parts := []string{}
		if node.Version != "" {
			parts = append(parts, "agent "+node.Version)
		}
		if node.Online {
			parts = append(parts, "online")
		} else {
			parts = append(parts, "offline")
		}
		if node.Platform != "" {
			parts = append(parts, node.Platform)
		}
		add("Reproduces on", "one node ("+strings.Join(parts, ", ")+")")
	}
	return strings.Join(lines, "\n")
}

func bugReportIssueURL(repo, title string, fields []bugReportField) string {
	params := url.Values{}
	params.Set("template", bugReportTemplate)
	params.Set("title", title)
	for _, field := range fields {
		if field.Value != "" {
			params.Set(field.ID, field.Value)
		}
	}
	return bugReportGitHub(repo, "/issues/new?"+params.Encode())
}

func bugReportMarkdown(title string, fields []bugReportField) string {
	var b strings.Builder
	b.WriteString("## " + title + "\n")
	for _, field := range fields {
		if field.Value == "" {
			continue
		}
		b.WriteString("\n### " + bugReportLabels[field.ID] + "\n\n")
		if field.ID == "logs" {
			b.WriteString("```text\n" + field.Value + "\n```\n")
		} else {
			b.WriteString(field.Value + "\n")
		}
	}
	return b.String()
}

func buildBugReport(repo, title string, fields []bugReportField, jobs, actions []string) bugReport {
	keepMax := len(jobs)
	if len(actions) > keepMax {
		keepMax = len(actions)
	}
	withLogs := func(base []bugReportField, keep int) []bugReportField {
		out := append([]bugReportField(nil), base...)
		if logs := bugReportLogText(jobs, actions, keep); logs != "" {
			out = append(out, bugReportField{ID: "logs", Value: logs})
		}
		return out
	}

	sent := append([]bugReportField(nil), fields...)
	clipboard := ""
	if len(bugReportIssueURL(repo, title, sent)) > bugReportURLLimit {
		for i := range sent {
			if sent[i].ID == "description" {
				clipboard = sent[i].Value
				sent[i].Value = bugReportPastePlaceholder
			}
		}
	}

	keep := keepMax
	link := bugReportIssueURL(repo, title, withLogs(sent, keep))
	for keep > 0 && len(link) > bugReportURLLimit {
		keep--
		link = bugReportIssueURL(repo, title, withLogs(sent, keep))
	}

	lines := 0
	for _, list := range [][]string{jobs, actions} {
		if len(list) < keep {
			lines += len(list)
		} else {
			lines += keep
		}
	}

	preview := withLogs(fields, keep)
	return bugReport{
		Repo:      repo,
		Title:     title,
		URL:       link,
		Fields:    preview,
		Text:      bugReportMarkdown(title, withLogs(fields, keepMax)),
		Clipboard: clipboard,
		LogsLines: lines,
		LogsTotal: len(jobs) + len(actions),
	}
}

func (h *Handler) AdminBugReportInfo(w http.ResponseWriter, r *http.Request) {
	if _, ok := bugReportStaff(w, r); !ok {
		return
	}
	ctx := r.Context()
	repo := bugReportRepo()
	nodes, _ := h.bugReportNodes(ctx, "")
	writeJSON(w, http.StatusOK, map[string]any{
		"repo":          repo,
		"repo_url":      bugReportGitHub(repo, ""),
		"issues_url":    bugReportGitHub(repo, "/issues"),
		"my_issues_url": bugReportGitHub(repo, "/issues?q="+url.QueryEscape("is:issue author:@me")),
		"version":       bugReportTag(buildinfo.Current()),
		"database":      h.bugReportDatabase(ctx),
		"nodes":         nodes,
	})
}

func (h *Handler) AdminBugReportSimilar(w http.ResponseWriter, r *http.Request) {
	if _, ok := bugReportStaff(w, r); !ok {
		return
	}
	ctx := r.Context()
	repo := bugReportRepo()
	words := bugReportWords(r.URL.Query().Get("q"))
	sum := sha1.Sum([]byte(repo + "\n" + strings.Join(words, " ")))
	key := "bugreport:similar:" + hex.EncodeToString(sum[:])

	issues := []updates.Issue{}
	if found, err := h.cache.GetJSON(ctx, key, &issues); err == nil && found {
		writeJSON(w, http.StatusOK, map[string]any{"issues": issues, "matched": len(words) > 0})
		return
	}
	issues, err := updates.SearchIssues(ctx, repo, words, bugReportSimilarLimit)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	if err := h.cache.SetJSON(ctx, key, issues, bugReportSimilarTTL); err != nil {
		log.Printf("отчёт об ошибке: похожие задачи не закешированы: %v", err)
	}
	writeJSON(w, http.StatusOK, map[string]any{"issues": issues, "matched": len(words) > 0})
}

func (h *Handler) AdminBugReportPrepare(w http.ResponseWriter, r *http.Request) {
	userID, ok := bugReportStaff(w, r)
	if !ok {
		return
	}
	var body bugReportRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Не удалось прочитать отчёт")
		return
	}
	title := bugReportClamp(scrubReportText(bugReportLine(body.Title)), bugReportTitleMax)
	if title == "" {
		writeError(w, http.StatusBadRequest, "Кратко опишите проблему")
		return
	}
	description := bugReportClamp(scrubReportText(strings.TrimSpace(body.Description)), bugReportDescMax)
	if description == "" {
		writeError(w, http.StatusBadRequest, "Опишите, что произошло")
		return
	}
	severity := strings.ToLower(strings.TrimSpace(body.Severity))
	if _, known := bugReportSeverities[severity]; !known {
		severity = "medium"
	}
	component := strings.ToLower(strings.TrimSpace(body.Component))
	if _, known := bugReportComponents[component]; !known {
		component = "unknown"
	}
	nodeID := strings.TrimSpace(body.NodeID)
	if _, err := uuid.Parse(nodeID); err != nil {
		nodeID = ""
	}

	ctx := r.Context()
	repo := bugReportRepo()
	nodes, node := h.bugReportNodes(ctx, nodeID)

	version := bugReportTag(buildinfo.Current())
	if ui := bugReportTag(bugReportClientValue(body.Client.UIVersion)); ui != "" && ui != version {
		version += " (interface " + ui + ")"
	}
	componentText := component + " (" + bugReportComponents[component] + ")"
	if component == "unknown" {
		componentText = bugReportComponents[component]
	}

	fields := []bugReportField{
		{ID: "description", Value: description},
		{ID: "severity", Value: bugReportSeverities[severity]},
		{ID: "component", Value: componentText},
		{ID: "version", Value: version},
		{ID: "environment", Value: bugReportEnvironment(body.Client, h.bugReportDatabase(ctx), nodes, node)},
	}
	var jobs, actions []string
	if body.AttachLogs {
		jobs, actions = h.bugReportLogs(ctx, userID)
	}
	report := buildBugReport(repo, title, fields, jobs, actions)

	audit(ctx, h.dbOf(ctx), userID, bugReportAuditAction, repo, map[string]any{
		"title":     title,
		"severity":  severity,
		"component": component,
		"logs":      report.LogsLines,
	})
	writeJSON(w, http.StatusOK, report)
}
