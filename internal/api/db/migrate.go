package db

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Migrate(ctx context.Context, pool *pgxpool.Pool, dir string) error {
	if _, err := pool.Exec(ctx, `
		CREATE SCHEMA IF NOT EXISTS core;
		CREATE TABLE IF NOT EXISTS core.schema_migrations (
		    version    TEXT PRIMARY KEY,
		    applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
	`); err != nil {
		return fmt.Errorf("журнал миграций не создан: %w", err)
	}

	files, err := filepath.Glob(filepath.Join(dir, "*.sql"))
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return fmt.Errorf("в %s нет миграций: укажите каталог в CORE_MIGRATIONS_DIR", dir)
	}
	sort.Strings(files)

	for _, file := range files {
		version := filepath.Base(file)

		var applied bool
		if err := pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM core.schema_migrations WHERE version = $1)`, version,
		).Scan(&applied); err != nil {
			return err
		}
		if applied {
			continue
		}

		body, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		if _, err := pool.Exec(ctx, string(body)); err != nil {
			return fmt.Errorf("миграция %s не применилась: %w", version, err)
		}
		if _, err := pool.Exec(ctx,
			`INSERT INTO core.schema_migrations (version) VALUES ($1) ON CONFLICT DO NOTHING`, version,
		); err != nil {
			return err
		}
	}
	return nil
}
