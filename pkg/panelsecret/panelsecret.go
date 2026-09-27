package panelsecret

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"os"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	DevJWTSecret      = "dev-secret-change-in-production"
	DevInternalSecret = "dev-internal-secret"
	jwtSettingKey     = "security.jwt_secret"
)

func configured(value string) bool {
	value = strings.TrimSpace(value)
	return value != "" && value != DevJWTSecret
}

func Internal(envValue string) (string, error) {
	value := strings.TrimSpace(envValue)
	if value != "" && value != DevInternalSecret {
		return value, nil
	}
	if DevSecretsAllowed() {
		return DevInternalSecret, nil
	}
	return "", errors.New(
		"INTERNAL_SECRET не задан или оставлен dev-значением: сгенерируйте его " +
			"(openssl rand -base64 32) и пропишите в deploy/.env — иначе внутренние " +
			"API relay и updater открыты всем, кто до них дотянется. " +
			"Для локальной разработки: VORTANIX_DEV=1")
}

func DevSecretsAllowed() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("VORTANIX_DEV"))) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

func JWT(ctx context.Context, db *pgxpool.Pool, envValue string) (string, error) {
	if configured(envValue) {
		return envValue, nil
	}
	if db == nil {
		return "", errors.New("не задан JWT_SECRET и нет доступа к базе, чтобы создать свой")
	}

	if stored, err := readJWT(ctx, db); err != nil {
		return "", err
	} else if stored != "" {
		return stored, nil
	}

	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	generated := base64.RawURLEncoding.EncodeToString(raw)
	if _, err := db.Exec(ctx, `
		INSERT INTO core.tenant_settings (key, value) VALUES ($1, to_jsonb($2::text))
		ON CONFLICT (key) DO NOTHING
	`, jwtSettingKey, generated); err != nil {
		return "", err
	}
	stored, err := readJWT(ctx, db)
	if err != nil {
		return "", err
	}
	if stored == "" {
		return "", errors.New("не удалось сохранить ключ подписи токенов")
	}
	return stored, nil
}

func readJWT(ctx context.Context, db *pgxpool.Pool) (string, error) {
	var value string
	err := db.QueryRow(ctx, `
		SELECT COALESCE(value #>> '{}', '') FROM core.tenant_settings WHERE key = $1
	`, jwtSettingKey).Scan(&value)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(value), nil
}
