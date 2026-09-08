package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vortanixapp/panel/pkg/oauth"
)

func TestVerifyTelegramHash(t *testing.T) {
	botToken := "123456:ABC-DEF"
	data := map[string]string{
		"id":         "42",
		"first_name": "Test",
		"auth_date":  "1600000000",
		"hash":       "",
	}
	if verifyTelegramHash(data, botToken) {
		t.Fatal("expected invalid without hash")
	}
}

func TestEmailVerificationHash(t *testing.T) {
	h1 := emailVerificationHash("User@Example.com")
	h2 := emailVerificationHash("user@example.com")
	if h1 != h2 {
		t.Fatalf("hash should be case-insensitive: %s vs %s", h1, h2)
	}
	if len(h1) != 40 {
		t.Fatalf("expected sha1 hex length 40, got %d", len(h1))
	}
}

func TestPublicProviderKeyMapsVKontakte(t *testing.T) {
	if got := publicProviderKey("vkontakte"); got != "vk" {
		t.Fatalf("vkontakte should map to vk, got %q", got)
	}
	if got := publicProviderKey("VKontakte"); got != "vk" {
		t.Fatalf("mapping should be case-insensitive, got %q", got)
	}
	for _, key := range []string{"google", "discord", "telegram"} {
		if got := publicProviderKey(key); got != key {
			t.Fatalf("%s should pass through, got %q", key, got)
		}
	}
	if got := publicProviderKey(" Google "); got != "google" {
		t.Fatalf("expected trimmed lowercase, got %q", got)
	}
}

func TestSocialProvidersListsOnlyConfigured(t *testing.T) {
	h := &Handler{oauth: oauth.NewRegistry("http://localhost:3000", "gid", "gsecret", "", "", "vkid", "vksecret")}

	rec := httptest.NewRecorder()
	h.SocialProviders(rec, httptest.NewRequest(http.MethodGet, "/v1/auth/social/providers", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var body struct {
		Providers []string `json:"providers"`
		Telegram  struct {
			Enabled     bool   `json:"enabled"`
			BotUsername string `json:"bot_username"`
		} `json:"telegram"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Providers) != 2 || body.Providers[0] != "google" || body.Providers[1] != "vk" {
		t.Fatalf("expected [google vk], got %v", body.Providers)
	}
	for _, p := range body.Providers {
		if p == "discord" {
			t.Fatal("unconfigured discord leaked into the list")
		}
	}
	if body.Telegram.Enabled || body.Telegram.BotUsername != "" {
		t.Fatalf("telegram should be disabled without a token, got %+v", body.Telegram)
	}
}

func TestSocialProvidersWithoutRegistry(t *testing.T) {
	rec := httptest.NewRecorder()
	(&Handler{}).SocialProviders(rec, httptest.NewRequest(http.MethodGet, "/v1/auth/social/providers", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if body := rec.Body.String(); !strings.Contains(body, `"providers":[]`) {
		t.Fatalf("expected empty provider list, got %s", body)
	}
}
