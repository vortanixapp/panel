package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/vortanixapp/panel/pkg/gamecatalog"
)

type catalogDB interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func syncCatalog(ctx context.Context, db catalogDB) error {
	for _, g := range gamecatalog.All() {
		gameID, err := upsertGame(ctx, db, g)
		if err != nil {
			return fmt.Errorf("игра %s: %w", g.Key, err)
		}
		if err := upsertDefaultVersion(ctx, db, gameID, g); err != nil {
			return fmt.Errorf("версия игры %s: %w", g.Key, err)
		}
		if err := upsertTariffs(ctx, db, gameID, g); err != nil {
			return fmt.Errorf("тарифы игры %s: %w", g.Key, err)
		}
	}
	return seedDefaultPaymentProviders(ctx, db)
}

func upsertGame(ctx context.Context, db catalogDB, g gamecatalog.Game) (string, error) {
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
			slug, name, description, image_url, code, query_protocol,
			min_port, max_port, active, meta
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10::jsonb)
		ON CONFLICT (slug) DO UPDATE SET
			query_protocol = EXCLUDED.query_protocol,
			active = CASE WHEN $11 THEN core.games.active ELSE false END
		RETURNING id::text
	`,
		g.Key, g.Name, g.Description, gamecatalog.PosterURL(g.Key),
		g.Code, g.QueryProto, g.MinPort, g.MaxPort,
		gamecatalog.Enabled(g.Key), metaJSON, gamecatalog.LinuxSupported(g.Key),
	).Scan(&id)
	if err != nil {
		return "", err
	}
	return id, nil
}

func upsertDefaultVersion(ctx context.Context, db catalogDB, gameID string, g gamecatalog.Game) error {
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
		INSERT INTO core.game_versions ( game_id, version, source_type, archive_url, docker_image,
			steam_app_id, steam_branch, steam_mod_config, active, sort_order, meta
		)
		VALUES ( $1, $2, $3, $4, $5, $6, $7, $8, true, 0, $9::jsonb)
		ON CONFLICT (game_id, version) DO UPDATE SET
			source_type = EXCLUDED.source_type,
			archive_url = EXCLUDED.archive_url,
			docker_image = EXCLUDED.docker_image,
			steam_app_id = EXCLUDED.steam_app_id,
			steam_branch = EXCLUDED.steam_branch,
			steam_mod_config = EXCLUDED.steam_mod_config,
			meta = EXCLUDED.meta
		WHERE core.game_versions.meta->>'managed_by' = 'gamecatalog'
	`, gameID, in.Version, in.SourceType, archiveURL, image,
		steamAppID, steamBranch, modConfig, metaJSON)
	if err != nil {
		return err
	}

	_, err = db.Exec(ctx, `
		UPDATE core.game_versions
		SET active = false
		WHERE game_id = $1 AND version <> $2
		  AND meta->>'managed_by' = 'gamecatalog' AND active = true
	`, gameID, in.Version)
	return err
}

func seedDefaultCatalog(ctx context.Context, tx pgx.Tx) error {
	return syncCatalog(ctx, tx)
}

func SyncCatalog(ctx context.Context, db catalogDB) error {
	return syncCatalog(ctx, db)
}

func seedDefaultPaymentProviders(ctx context.Context, db catalogDB) error {
	providers := []string{"yookassa", "freekassa", "robokassa", "stripe"}
	for _, p := range providers {
		_, err := db.Exec(ctx, `
			INSERT INTO core.payment_providers (provider, enabled, config)
			VALUES ($1, false, '{}'::jsonb)
			ON CONFLICT (provider) DO NOTHING
		`, p)
		if err != nil {
			return err
		}
	}
	return nil
}
