package handlers

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vortanix/vortanix/pkg/gamecatalog"
)

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("VORTANIX_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("VORTANIX_TEST_DATABASE_URL не задан — тест против живой БД пропущен")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("подключение: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func newTestTenant(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	ctx := context.Background()
	id := uuid.NewString()
	slug := "test-" + id[:8]
	if _, err := pool.Exec(ctx, `
		INSERT INTO core.tenants (id, slug, name) VALUES ($1::uuid, $2, $3)
	`, id, slug, "Catalog sync test"); err != nil {
		t.Fatalf("создание тенанта: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM core.tenants WHERE id = $1::uuid`, id)
	})
	return id
}

func TestSyncCatalogSeedsEveryGameWithVersion(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	tenantID := newTestTenant(t, pool)

	if err := syncCatalog(ctx, pool, tenantID); err != nil {
		t.Fatalf("syncCatalog: %v", err)
	}

	var games int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM core.games WHERE tenant_id = $1`, tenantID).Scan(&games); err != nil {
		t.Fatalf("подсчёт игр: %v", err)
	}
	if games != len(gamecatalog.All()) {
		t.Errorf("игр в базе %d, в каталоге %d", games, len(gamecatalog.All()))
	}

	var withoutVersion int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM core.games g
		WHERE g.tenant_id = $1
		  AND NOT EXISTS (SELECT 1 FROM core.game_versions v WHERE v.game_id = g.id)
	`, tenantID).Scan(&withoutVersion); err != nil {
		t.Fatalf("подсчёт версий: %v", err)
	}
	if withoutVersion != 0 {
		t.Errorf("игр без версии: %d (должно быть 0)", withoutVersion)
	}

	var steamWithoutApp int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM core.game_versions
		WHERE tenant_id = $1 AND source_type = 'steam' AND COALESCE(steam_app_id, 0) = 0
	`, tenantID).Scan(&steamWithoutApp); err != nil {
		t.Fatalf("проверка steam-версий: %v", err)
	}
	if steamWithoutApp != 0 {
		t.Errorf("steam-версий без App ID: %d", steamWithoutApp)
	}
}

func TestSyncCatalogIsIdempotent(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	tenantID := newTestTenant(t, pool)

	if err := syncCatalog(ctx, pool, tenantID); err != nil {
		t.Fatalf("первый syncCatalog: %v", err)
	}

	if _, err := pool.Exec(ctx, `
		UPDATE core.games SET name = 'Мой Rust', active = true
		WHERE tenant_id = $1 AND slug = 'rust'
	`, tenantID); err != nil {
		t.Fatalf("правка админа: %v", err)
	}

	if err := syncCatalog(ctx, pool, tenantID); err != nil {
		t.Fatalf("повторный syncCatalog: %v", err)
	}

	var games, versions int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM core.games WHERE tenant_id = $1`, tenantID).Scan(&games)
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM core.game_versions WHERE tenant_id = $1`, tenantID).Scan(&versions)
	if games != len(gamecatalog.All()) {
		t.Errorf("после повтора игр %d, ожидалось %d", games, len(gamecatalog.All()))
	}
	if versions != len(gamecatalog.All()) {
		t.Errorf("после повтора версий %d, ожидалось %d", versions, len(gamecatalog.All()))
	}

	var name string
	var active bool
	if err := pool.QueryRow(ctx, `
		SELECT name, active FROM core.games WHERE tenant_id = $1 AND slug = 'rust'
	`, tenantID).Scan(&name, &active); err != nil {
		t.Fatalf("чтение rust: %v", err)
	}
	if name != "Мой Rust" {
		t.Errorf("название админа затёрто: %q", name)
	}
	if !active {
		t.Error("вручную включённая игра снова погашена синхронизацией")
	}
}

func TestSyncCatalogMarksOversizedGamesInactive(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	tenantID := newTestTenant(t, pool)

	if err := syncCatalog(ctx, pool, tenantID); err != nil {
		t.Fatalf("syncCatalog: %v", err)
	}

	rows, err := pool.Query(ctx, `SELECT slug, active FROM core.games WHERE tenant_id = $1`, tenantID)
	if err != nil {
		t.Fatalf("чтение игр: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var slug string
		var active bool
		if rows.Scan(&slug, &active) != nil {
			continue
		}
		if want := gamecatalog.FitsSmallNode(slug); active != want {
			t.Errorf("%s: active=%v, ожидалось %v", slug, active, want)
		}
	}
}

func TestSyncCatalogFillsMissingVersions(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()

	var tenants int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM core.tenants WHERE status = 'active'`).Scan(&tenants); err != nil {
		t.Fatalf("подсчёт тенантов: %v", err)
	}
	if tenants == 0 {
		t.Skip("в тестовой базе нет тенантов")
	}

	if err := SyncCatalog(ctx, pool); err != nil {
		t.Fatalf("SyncCatalog: %v", err)
	}

	var withoutVersion int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM core.games g
		WHERE NOT EXISTS (SELECT 1 FROM core.game_versions v WHERE v.game_id = g.id)
	`).Scan(&withoutVersion); err != nil {
		t.Fatalf("подсчёт игр без версии: %v", err)
	}
	if withoutVersion != 0 {
		t.Errorf("игр без версии после синхронизации: %d", withoutVersion)
	}

	var activeUninstallable int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM core.games g
		WHERE g.active
		  AND NOT EXISTS (
			  SELECT 1 FROM core.game_versions v
			  WHERE v.game_id = g.id AND v.active
		  )
	`).Scan(&activeUninstallable); err != nil {
		t.Fatalf("подсчёт активных игр без версии: %v", err)
	}
	if activeUninstallable != 0 {
		t.Errorf("активных игр без активной версии: %d", activeUninstallable)
	}
}

func TestSyncCatalogDisablesGamesWithoutLinuxServer(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	tenantID := newTestTenant(t, pool)

	if err := syncCatalog(ctx, pool, tenantID); err != nil {
		t.Fatalf("syncCatalog: %v", err)
	}

	var blocked []string
	for _, g := range gamecatalog.All() {
		if g.NoLinuxServer != "" {
			blocked = append(blocked, g.Key)
		}
	}
	if len(blocked) == 0 {
		t.Fatal("в каталоге не осталось игр без Linux-сервера — тест потерял смысл")
	}

	for _, slug := range blocked {
		if _, err := pool.Exec(ctx,
			`UPDATE core.games SET active = true WHERE tenant_id = $1 AND slug = $2`,
			tenantID, slug); err != nil {
			t.Fatalf("%s: включить игру: %v", slug, err)
		}
	}

	if err := syncCatalog(ctx, pool, tenantID); err != nil {
		t.Fatalf("повторный syncCatalog: %v", err)
	}

	for _, slug := range blocked {
		var active bool
		if err := pool.QueryRow(ctx,
			`SELECT active FROM core.games WHERE tenant_id = $1 AND slug = $2`,
			tenantID, slug).Scan(&active); err != nil {
			t.Fatalf("%s: чтение: %v", slug, err)
		}
		if active {
			t.Errorf("%s: игра без Linux-сервера осталась включённой", slug)
		}
	}

	if _, err := pool.Exec(ctx,
		`UPDATE core.games SET active = true WHERE tenant_id = $1 AND slug = 'arksa'`,
		tenantID); err != nil {
		t.Fatalf("включить arksa: %v", err)
	}
	if err := syncCatalog(ctx, pool, tenantID); err != nil {
		t.Fatalf("третий syncCatalog: %v", err)
	}
	var arksaActive bool
	if err := pool.QueryRow(ctx,
		`SELECT active FROM core.games WHERE tenant_id = $1 AND slug = 'arksa'`,
		tenantID).Scan(&arksaActive); err != nil {
		t.Fatalf("arksa: чтение: %v", err)
	}
	if !arksaActive {
		t.Error("arksa: ручное включение админом сброшено синхронизацией")
	}
}

func TestSyncCatalogStoresSteamModConfig(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	tenantID := newTestTenant(t, pool)

	if err := syncCatalog(ctx, pool, tenantID); err != nil {
		t.Fatalf("syncCatalog: %v", err)
	}

	var mod *string
	if err := pool.QueryRow(ctx, `
		SELECT v.steam_mod_config
		FROM core.games g JOIN core.game_versions v ON v.game_id = g.id
		WHERE g.tenant_id = $1 AND g.slug = 'cs16'
		  AND v.meta->>'managed_by' = 'gamecatalog'
	`, tenantID).Scan(&mod); err != nil {
		t.Fatalf("чтение версии cs16: %v", err)
	}
	if mod == nil || *mod != "cstrike" {
		got := "<null>"
		if mod != nil {
			got = *mod
		}
		t.Errorf("cs16: steam_mod_config = %s, ожидалось cstrike", got)
	}
}
