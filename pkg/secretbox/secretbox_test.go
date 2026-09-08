package secretbox

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	box, err := New("test-secrets-key")
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	if !box.Enabled() {
		t.Fatal("expected enabled box")
	}
	const secret = "sk_live_0123456789"
	sealed, err := box.Encrypt(secret)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if !IsEncrypted(sealed) {
		t.Fatalf("expected %q prefix, got %q", Prefix, sealed)
	}
	if strings.Contains(sealed, secret) {
		t.Fatal("plaintext leaked into ciphertext")
	}
	plain, err := box.Decrypt(sealed)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if plain != secret {
		t.Fatalf("expected %q, got %q", secret, plain)
	}
}

func TestNonceIsPerValue(t *testing.T) {
	box, _ := New("test-secrets-key")
	first, _ := box.Encrypt("same")
	second, _ := box.Encrypt("same")
	if first == second {
		t.Fatal("identical ciphertext for the same plaintext — nonce is reused")
	}
}

func TestEncryptIsIdempotent(t *testing.T) {
	box, _ := New("test-secrets-key")
	once, _ := box.Encrypt("token")
	twice, _ := box.Encrypt(once)
	if once != twice {
		t.Fatalf("double encryption changed the value: %q vs %q", once, twice)
	}
}

func TestEmptyStaysEmpty(t *testing.T) {
	box, _ := New("test-secrets-key")
	sealed, err := box.Encrypt("")
	if err != nil || sealed != "" {
		t.Fatalf("expected empty passthrough, got %q (%v)", sealed, err)
	}
}

func TestLegacyPlaintextPassesThrough(t *testing.T) {
	box, _ := New("test-secrets-key")
	plain, err := box.Decrypt("plain-legacy-token")
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if plain != "plain-legacy-token" {
		t.Fatalf("expected passthrough, got %q", plain)
	}
}

func TestDisabledBoxPassesPlaintextThrough(t *testing.T) {
	box, err := New("")
	if err != nil {
		t.Fatalf("empty key should not be an error: %v", err)
	}
	if box.Enabled() {
		t.Fatal("empty key must leave encryption disabled")
	}
	sealed, err := box.Encrypt("token")
	if err != nil || sealed != "token" {
		t.Fatalf("expected passthrough, got %q (%v)", sealed, err)
	}
}

func TestDisabledBoxFailsLoudlyOnCiphertext(t *testing.T) {
	enabled, _ := New("test-secrets-key")
	sealed, _ := enabled.Encrypt("token")

	disabled, _ := New("")
	if _, err := disabled.Decrypt(sealed); !errors.Is(err, ErrNoKey) {
		t.Fatalf("expected ErrNoKey, got %v", err)
	}
	if got := disabled.MustDecrypt(sealed); got != "" {
		t.Fatalf("MustDecrypt must not return ciphertext, got %q", got)
	}
}

func TestWrongKeyFails(t *testing.T) {
	first, _ := New("key-one")
	second, _ := New("key-two")
	sealed, _ := first.Encrypt("token")
	if _, err := second.Decrypt(sealed); err == nil {
		t.Fatal("expected failure with a different key")
	}
}

func TestRawKeyFormatsAreAccepted(t *testing.T) {
	raw := make([]byte, 32)
	for i := range raw {
		raw[i] = byte(i)
	}
	b64, _ := New(base64.StdEncoding.EncodeToString(raw))
	sealed, err := b64.Encrypt("token")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	hexBox, _ := New("000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f")
	plain, err := hexBox.Decrypt(sealed)
	if err != nil {
		t.Fatalf("hex key should derive the same key: %v", err)
	}
	if plain != "token" {
		t.Fatalf("expected token, got %q", plain)
	}
}

func TestJSONEnvelope(t *testing.T) {
	box, _ := New("test-secrets-key")
	original := []byte(`{"secret_key":"sk_live_1","shop_id":"42"}`)

	sealed, err := box.EncryptJSON(original)
	if err != nil {
		t.Fatalf("encrypt json: %v", err)
	}
	if strings.Contains(string(sealed), "sk_live_1") {
		t.Fatalf("secret leaked into the envelope: %s", sealed)
	}
	var probe map[string]any
	if err := json.Unmarshal(sealed, &probe); err != nil {
		t.Fatalf("envelope is not valid json: %v", err)
	}
	if _, ok := probe[EnvelopeKey]; !ok {
		t.Fatalf("expected %q field, got %s", EnvelopeKey, sealed)
	}

	plain, err := box.DecryptJSON(sealed)
	if err != nil {
		t.Fatalf("decrypt json: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(plain, &got); err != nil {
		t.Fatalf("decrypted json: %v", err)
	}
	if got["secret_key"] != "sk_live_1" || got["shop_id"] != "42" {
		t.Fatalf("round trip lost data: %v", got)
	}
}

func TestJSONEnvelopeLeavesPlainConfigAlone(t *testing.T) {
	box, _ := New("test-secrets-key")
	original := []byte(`{"secret_key":"sk_live_1"}`)
	out, err := box.DecryptJSON(original)
	if err != nil {
		t.Fatalf("decrypt json: %v", err)
	}
	if string(out) != string(original) {
		t.Fatalf("plain config should pass through, got %s", out)
	}
}

func TestJSONEnvelopeIsIdempotent(t *testing.T) {
	box, _ := New("test-secrets-key")
	once, _ := box.EncryptJSON([]byte(`{"a":"b"}`))
	twice, _ := box.EncryptJSON(once)
	if string(once) != string(twice) {
		t.Fatalf("double envelope: %s vs %s", once, twice)
	}
}

func TestJSONEnvelopeWithoutKeyFailsLoudly(t *testing.T) {
	enabled, _ := New("test-secrets-key")
	sealed, _ := enabled.EncryptJSON([]byte(`{"a":"b"}`))

	disabled, _ := New("")
	if _, err := disabled.DecryptJSON(sealed); !errors.Is(err, ErrNoKey) {
		t.Fatalf("expected ErrNoKey, got %v", err)
	}
}
