package panelsecret

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	DevJWTSecret  = "dev-secret-change-in-production"
	jwtSettingKey = "security.jwt_secret"
)

func configured(value string) bool {
	value = strings.TrimSpace(value)
	return value != "" && value != DevJWTSecret
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
