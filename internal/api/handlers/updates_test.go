package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLatestReleaseIsParsed(t *testing.T) {
	var gotPath, gotAccept string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAccept = r.Header.Get("Accept")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"tag_name": "v1.4.0",
			"body": "Очередь больше не встаёт",
			"html_url": "https://example.test/releases/v1.4.0",
			"published_at": "2026-09-08T10:00:00Z",
			"prerelease": false
		}`))
	}))
	defer srv.Close()
	t.Setenv("UPDATE_API_BASE", srv.URL)

	rel, err := fetchLatestRelease(context.Background(), "owner/repo")
	if err != nil {
		t.Fatalf("выпуск не прочитан: %v", err)
	}
	if gotPath != "/repos/owner/repo/releases/latest" {
		t.Errorf("запрошен неверный адрес: %s", gotPath)
	}
	if !strings.Contains(gotAccept, "github") {
		t.Errorf("не назван формат ответа github: %q", gotAccept)
	}
	if rel.Version != "1.4.0" {
		t.Errorf("префикс v не снят с тега: %q", rel.Version)
	}
	if rel.Notes != "Очередь больше не встаёт" {
		t.Errorf("текст выпуска потерян: %q", rel.Notes)
	}
	if rel.CheckedAt == "" {
		t.Error("не проставлено время проверки — на странице будет прочерк")
	}
}

func TestMissingReleaseIsExplained(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	t.Setenv("UPDATE_API_BASE", srv.URL)

	_, err := fetchLatestRelease(context.Background(), "owner/repo")
	if err == nil {
		t.Fatal("репозиторий без выпусков должен давать ошибку")
	}
	if !strings.Contains(err.Error(), "выпуск") {
		t.Errorf("сообщение не объясняет причину: %v", err)
	}
}

func TestRateLimitIsExplained(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()
	t.Setenv("UPDATE_API_BASE", srv.URL)

	_, err := fetchLatestRelease(context.Background(), "owner/repo")
	if err == nil || !strings.Contains(err.Error(), "частот") {
		t.Errorf("ограничение частоты должно объясняться словами: %v", err)
	}
}

func TestNewerVersionComparesNumbersNotStrings(t *testing.T) {
	newer := [][2]string{
		{"1.0.0", "0.9.9"},
		{"0.10.0", "0.9.0"},
		{"0.2.10", "0.2.9"},
		{"v1.2.0", "1.1.9"},
		{"1.2.3", "1.2"},
	}
	for _, c := range newer {
		if !isNewerVersion(c[0], c[1]) {
			t.Errorf("%s должна считаться новее %s", c[0], c[1])
		}
	}

	notNewer := [][2]string{
		{"1.0.0", "1.0.0"},
		{"0.9.0", "0.10.0"},
		{"1.0.0", "1.0.1"},
		{"1.0.0", "dev"},
		{"edge", "1.0.0"},
		{"", "1.0.0"},
		{"1.0.0", ""},
	}
	for _, c := range notNewer {
		if isNewerVersion(c[0], c[1]) {
			t.Errorf("%s не должна считаться новее %s", c[0], c[1])
		}
	}
}

func TestPrereleaseSuffixIgnoredInComparison(t *testing.T) {
	if !isNewerVersion("1.1.0-rc.1", "1.0.0") {
		t.Error("предвыпуск новой версии всё равно новее прежней")
	}
	if isNewerVersion("1.0.0-rc.1", "1.0.0") {
		t.Error("предвыпуск той же версии не новее её самой")
	}
}

func TestUnsetRepoFallsBackToProject(t *testing.T) {
	t.Setenv("UPDATE_REPO", "")
	if got := updateRepo(); got != "vortanixapp/panel" {
		t.Errorf("без настройки ожидался репозиторий проекта, получено %q", got)
	}
	t.Setenv("UPDATE_REPO", "someone/fork/")
	if got := updateRepo(); got != "someone/fork" {
		t.Errorf("хвостовая косая черта даёт неверный адрес github: %q", got)
	}
}
