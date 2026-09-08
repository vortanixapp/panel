package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/vortanix/vortanix/pkg/gamecatalog"
)

type catalogDB interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func syncCatalog(ctx context.Context, db catalogDB, tenantID string) error {
	for _, g := range gamecatalog.All() {
		gameID, err := upsertGame(ctx, db, tenantID, g)
		if err != nil {
			return fmt.Errorf("игра %s: %w", g.Key, err)
		}
		if err := upsertDefaultVersion(ctx, db, tenantID, gameID, g); err != nil {
			return fmt.Errorf("версия игры %s: %w", g.Key, err)
		}
		if err := upsertTariffs(ctx, db, tenantID, gameID, g); err != nil {
			return fmt.Errorf("тарифы игры %s: %w", g.Key, err)
		}
	}
	return seedDefaultPaymentProviders(ctx, db, tenantID)
}

func upsertGame(ctx context.Context, db catalogDB, tenantID string, g gamecatalog.Game) (string, error) {
	meta := map[string]any{}
	if g.MaxSlots > 0 {
		meta["max_slots"] = g.MaxSlots
	}
	if g.Install.SourceType == gamecatalog.SourceSteam {
		meta["steam_updatable"] = true
	}
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return "", err
	}

	var id string
	err = db.QueryRow(ctx, `
		INSERT INTO core.games (
			tenant_id, slug, name, description, image_url, code, query_protocol,
			min_port, max_port, active, meta
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11::jsonb)
		ON CONFLICT (tenant_id, slug) DO UPDATE SET
			query_protocol = EXCLUDED.query_protocol,
			-- Игру, которая физически не запускается на линуксовой ноде,
			-- гасим и у существующих тенантов. Обычный active админ
			-- контролирует сам (см. комментарий выше), но здесь выбора нет:
			-- до аудита 2026-08-24 у десяти игр стоял App ID клиентской
			-- игры или вовсе чужого приложения, они были включены и их
			-- можно было арендовать — оплаченный сервер не запускался
			-- никогда. Появятся Windows-ноды — админ включит вручную.
			active = CASE WHEN $12 THEN core.games.active ELSE false END
		RETURNING id::text
	`,
		tenantID, g.Key, g.Name, g.Description, gamecatalog.PosterURL(g.Key),
		g.Code, g.QueryProto, g.MinPort, g.MaxPort,
		gamecatalog.Enabled(g.Key), metaJSON, gamecatalog.LinuxSupported(g.Key),
	).Scan(&id)
	if err != nil {
		return "", err
	}
	return id, nil
}

func upsertDefaultVersion(ctx context.Context, db catalogDB, tenantID, gameID string, g gamecatalog.Game) error {
	in := g.Install

	var archiveURL, steamBranch, modConfig *string
	var steamAppID *int64
	if in.ArchiveURL != "" {
		url := in.ArchiveURL
		archiveURL = &url
	}
	if in.SteamBranch != "" {
		branch := in.SteamBranch
		steamBranch = &branch
	}
	if in.SteamModConfig != "" {
		mod := in.SteamModConfig
		modConfig = &mod
	}
	if in.SteamAppID > 0 {
		appID := in.SteamAppID
		steamAppID = &appID
	}

	meta := map[string]any{"managed_by": "gamecatalog"}
	if note := gamecatalog.InstallNote(g); note != "" {
		meta["note"] = note
	}
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return err
	}

	image := gamecatalog.Image(g.Key)
	_, err = db.Exec(ctx, `
		INSERT INTO core.game_versions (
			tenant_id, game_id, version, source_type, archive_url, docker_image,
			steam_app_id, steam_branch, steam_mod_config, active, sort_order, meta
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, true, 0, $10::jsonb)
		ON CONFLICT (game_id, version) DO UPDATE SET
			source_type = EXCLUDED.source_type,
			archive_url = EXCLUDED.archive_url,
			docker_image = EXCLUDED.docker_image,
			steam_app_id = EXCLUDED.steam_app_id,
			steam_branch = EXCLUDED.steam_branch,
			steam_mod_config = EXCLUDED.steam_mod_config,
			meta = EXCLUDED.meta
		WHERE core.game_versions.meta->>'managed_by' = 'gamecatalog'
	`, tenantID, gameID, in.Version, in.SourceType, archiveURL, image,
		steamAppID, steamBranch, modConfig, metaJSON)
	if err != nil {
		return err
	}

	_, err = db.Exec(ctx, `
		UPDATE core.game_versions
		SET active = false
		WHERE tenant_id = $1 AND game_id = $2 AND version <> $3
		  AND meta->>'managed_by' = 'gamecatalog' AND active = true
	`, tenantID, gameID, in.Version)
	return err
}

func seedDefaultCatalog(ctx context.Context, tx pgx.Tx, tenantID string) error {
	return syncCatalog(ctx, tx, tenantID)
}

func SyncCatalogForAllTenants(ctx context.Context, db catalogDB) error {
	rows, err := db.Query(ctx, `SELECT id::text FROM core.tenants WHERE status = 'active'`)
	if err != nil {
		return fmt.Errorf("список тенантов: %w", err)
	}
	var tenants []string
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			tenants = append(tenants, id)
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("список тенантов: %w", err)
	}

	for _, tenantID := range tenants {
		if err := syncCatalog(ctx, db, tenantID); err != nil {
			return fmt.Errorf("тенант %s: %w", tenantID, err)
		}
	}
	return nil
}

func seedDefaultPaymentProviders(ctx context.Context, db catalogDB, tenantID string) error {
	providers := []string{"yookassa", "freekassa", "robokassa", "stripe"}
	for _, p := range providers {
		_, err := db.Exec(ctx, `
			INSERT INTO core.payment_providers (tenant_id, provider, enabled, config)
			VALUES ($1, $2, false, '{}'::jsonb)
			ON CONFLICT (tenant_id, provider) DO NOTHING
		`, tenantID, p)
		if err != nil {
			return err
		}
	}
	return nil
}
