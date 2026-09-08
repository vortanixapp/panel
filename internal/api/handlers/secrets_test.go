package handlers

import (
	"encoding/json"
	"testing"

	"github.com/vortanix/vortanix/pkg/secretbox"
)

func testHandlerWithSecrets(t *testing.T, key string) *Handler {
	t.Helper()
	box, err := secretbox.New(key)
	if err != nil {
		t.Fatalf("secretbox: %v", err)
	}
	return &Handler{secrets: box}
}

func TestSSHPasswordForStorageEncryptsAndClearsMeta(t *testing.T) {
	h := testHandlerWithSecrets(t, "test-secrets-key")
	meta := map[string]any{"code": "eu-1"}

	stored, err := h.sshPasswordForStorage("s3cret", nil, meta)
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	if stored == nil {
		t.Fatal("expected a stored value")
	}
	if !secretbox.IsEncrypted(*stored) {
		t.Fatalf("password stored unencrypted: %q", *stored)
	}
	if _, ok := meta["ssh_password"]; ok {
		t.Fatal("plaintext copy left in meta")
	}
	if plain := h.secrets.MustDecrypt(*stored); plain != "s3cret" {
		t.Fatalf("round trip failed: %q", plain)
	}
}

func TestSSHPasswordForStorageMigratesLegacyMetaValue(t *testing.T) {
	h := testHandlerWithSecrets(t, "test-secrets-key")
	meta := map[string]any{"ssh_password": "legacy-pass"}

	stored, err := h.sshPasswordForStorage(nil, nil, meta)
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	if stored == nil {
		t.Fatal("legacy password lost")
	}
	if plain := h.secrets.MustDecrypt(*stored); plain != "legacy-pass" {
		t.Fatalf("expected legacy-pass, got %q", plain)
	}
	if _, ok := meta["ssh_password"]; ok {
		t.Fatal("legacy plaintext should be removed from meta")
	}
}

func TestSSHPasswordForStorageKeepsCurrentWhenNoInput(t *testing.T) {
	h := testHandlerWithSecrets(t, "test-secrets-key")
	current := "already-set"
	meta := map[string]any{}

	stored, err := h.sshPasswordForStorage(nil, &current, meta)
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	if stored == nil || h.secrets.MustDecrypt(*stored) != "already-set" {
		t.Fatalf("existing password should survive an unrelated edit, got %v", stored)
	}
}

func TestSSHPasswordForStorageIgnoresMissingField(t *testing.T) {
	h := testHandlerWithSecrets(t, "test-secrets-key")
	body := map[string]any{}
	meta := map[string]any{}

	stored, err := h.sshPasswordForStorage(body["ssh_password"], nil, meta)
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	if stored != nil {
		t.Fatalf("expected no password, got %q", *stored)
	}
}

func TestSSHPasswordForStorageWithoutKeyStaysPlaintext(t *testing.T) {
	h := testHandlerWithSecrets(t, "")
	meta := map[string]any{}

	stored, err := h.sshPasswordForStorage("plain", nil, meta)
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	if stored == nil || *stored != "plain" {
		t.Fatalf("expected plaintext passthrough, got %v", stored)
	}
}

func TestProviderConfigDecryptsEnvelope(t *testing.T) {
	h := testHandlerWithSecrets(t, "test-secrets-key")
	sealed, err := h.secrets.EncryptJSON([]byte(`{"secret_key":"sk_live_1"}`))
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	cfg := h.providerConfig(sealed)
	if cfg["secret_key"] != "sk_live_1" {
		t.Fatalf("expected decrypted config, got %v", cfg)
	}
}

func TestProviderConfigReadsLegacyPlaintext(t *testing.T) {
	h := testHandlerWithSecrets(t, "test-secrets-key")
	cfg := h.providerConfig([]byte(`{"secret_key":"sk_live_1"}`))
	if cfg["secret_key"] != "sk_live_1" {
		t.Fatalf("plain config must still be readable, got %v", cfg)
	}
}

func TestProviderConfigWithoutKeyReturnsEmpty(t *testing.T) {
	enabled := testHandlerWithSecrets(t, "test-secrets-key")
	sealed, _ := enabled.secrets.EncryptJSON([]byte(`{"secret_key":"sk_live_1"}`))

	disabled := testHandlerWithSecrets(t, "")
	cfg := disabled.providerConfig(sealed)
	if len(cfg) != 0 {
		t.Fatalf("expected empty config, got %v", cfg)
	}
	raw, _ := json.Marshal(cfg)
	if string(raw) != "{}" {
		t.Fatalf("expected {}, got %s", raw)
	}
}

func TestSecretSettingKeysCoverKnownSecrets(t *testing.T) {
	for formField, settingKey := range map[string]string{
		"mail_password":               "mail.mailers.smtp.password",
		"telegram_bot_token":          "telegram.notifications.bot_token",
		"google_client_secret":        "services.google.client_secret",
		"discord_client_secret":       "services.discord.client_secret",
		"vk_client_secret":            "services.vkontakte.client_secret",
		"recaptcha_secret_key":        "services.recaptcha.secret_key",
		"files_storage_s3_secret":     "files.storage.s3.secret",
		"files_storage_sftp_password": "files.storage.sftp.password",
		"dockerhub_token":             "dockerhub.token",
	} {
		if !secretSettingKeys[settingKey] {
			t.Errorf("%s (%s) is a secret but is not in secretSettingKeys", formField, settingKey)
		}
	}
}
