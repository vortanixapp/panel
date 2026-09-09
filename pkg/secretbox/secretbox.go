package secretbox

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

const Prefix = "enc:v1:"

const EnvelopeKey = "__enc"

var ErrNoKey = errors.New("secretbox: encrypted value found but SECRETS_KEY is not set")

type Box struct {
	aead cipher.AEAD
}

func New(key string) (*Box, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, nil
	}
	raw := deriveKey(key)
	block, err := aes.NewCipher(raw)
	if err != nil {
		return nil, fmt.Errorf("secretbox: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("secretbox: %w", err)
	}
	return &Box{aead: aead}, nil
}

func deriveKey(key string) []byte {
	for _, decode := range []func(string) ([]byte, error){
		base64.StdEncoding.DecodeString,
		base64.RawStdEncoding.DecodeString,
		base64.URLEncoding.DecodeString,
		base64.RawURLEncoding.DecodeString,
		hex.DecodeString,
	} {
		if decoded, err := decode(key); err == nil && len(decoded) == 32 {
			return decoded
		}
	}
	sum := sha256.Sum256([]byte(key))
	return sum[:]
}

func (b *Box) Enabled() bool { return b != nil && b.aead != nil }

func IsEncrypted(value string) bool {
	return strings.HasPrefix(value, Prefix)
}

func (b *Box) Encrypt(plaintext string) (string, error) {
	if !b.Enabled() || plaintext == "" || IsEncrypted(plaintext) {
		return plaintext, nil
	}
	nonce := make([]byte, b.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("secretbox: nonce: %w", err)
	}
	sealed := b.aead.Seal(nonce, nonce, []byte(plaintext), nil)
	return Prefix + base64.RawStdEncoding.EncodeToString(sealed), nil
}

func (b *Box) Decrypt(stored string) (string, error) {
	if !IsEncrypted(stored) {
		return stored, nil
	}
	if !b.Enabled() {
		return "", ErrNoKey
	}
	sealed, err := base64.RawStdEncoding.DecodeString(strings.TrimPrefix(stored, Prefix))
	if err != nil {
		return "", fmt.Errorf("secretbox: decode: %w", err)
	}
	nonceSize := b.aead.NonceSize()
	if len(sealed) < nonceSize {
		return "", errors.New("secretbox: ciphertext too short")
	}
	plain, err := b.aead.Open(nil, sealed[:nonceSize], sealed[nonceSize:], nil)
	if err != nil {
		return "", fmt.Errorf("secretbox: open (wrong SECRETS_KEY?): %w", err)
	}
	return string(plain), nil
}

func (b *Box) MustDecrypt(stored string) string {
	plain, err := b.Decrypt(stored)
	if err != nil {
		return ""
	}
	return plain
}

func (b *Box) EncryptJSON(raw []byte) ([]byte, error) {
	if !b.Enabled() || len(raw) == 0 {
		return raw, nil
	}
	if _, ok := envelopeOf(raw); ok {
		return raw, nil
	}
	sealed, err := b.Encrypt(string(raw))
	if err != nil {
		return nil, err
	}
	return json.Marshal(map[string]string{EnvelopeKey: sealed})
}

func (b *Box) DecryptJSON(raw []byte) ([]byte, error) {
	sealed, ok := envelopeOf(raw)
	if !ok {
		return raw, nil
	}
	plain, err := b.Decrypt(sealed)
	if err != nil {
		return nil, err
	}
	return []byte(plain), nil
}

func envelopeOf(raw []byte) (string, bool) {
	if len(raw) == 0 {
		return "", false
	}
	var probe map[string]json.RawMessage
	if json.Unmarshal(raw, &probe) != nil {
		return "", false
	}
	value, ok := probe[EnvelopeKey]
	if !ok {
		return "", false
	}
	var sealed string
	if json.Unmarshal(value, &sealed) != nil {
		return "", false
	}
	return sealed, IsEncrypted(sealed)
}
