package portalloc

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
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

type testEnv struct {
	pool     *pgxpool.Pool
	tenantID string
	nodeID   string
}

func setupEnv(t *testing.T) testEnv {
	t.Helper()
	pool := testPool(t)
	ctx := context.Background()

	var tenantID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO core.tenants (id, slug, name)
		VALUES (gen_random_uuid(), 'portalloc-' || substr(gen_random_uuid()::text, 1, 8), 'portalloc test')
		RETURNING id::text
	`).Scan(&tenantID); err != nil {
		t.Fatalf("тенант: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM core.tenants WHERE id = $1::uuid`, tenantID)
	})

	var nodeID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO core.nodes (tenant_id, name, fqdn, agent_token)
		VALUES ($1::uuid, 'test-node', 'node.test', 'token')
		RETURNING id::text
	`, tenantID).Scan(&nodeID); err != nil {
		t.Fatalf("нода: %v", err)
	}

	for _, g := range []struct {
		slug           string
		minPort, maxPt int
	}{
		{"cs2", 27015, 27030},
		{"palworld", 8211, 8250},
	} {
		if _, err := pool.Exec(ctx, `
			INSERT INTO core.games (tenant_id, slug, name, min_port, max_port, active)
			VALUES ($1::uuid, $2, $2, $3, $4, true)
		`, tenantID, g.slug, g.minPort, g.maxPt); err != nil {
			t.Fatalf("игра %s: %v", g.slug, err)
		}
	}
	return testEnv{pool: pool, tenantID: tenantID, nodeID: nodeID}
}

func (e testEnv) newServer(t *testing.T, gameSlug string) string {
	t.Helper()
	var id string
	if err := e.pool.QueryRow(context.Background(), `
		INSERT INTO core.servers (tenant_id, node_id, game_id, name)
		VALUES ($1::uuid, $2::uuid, $3, 'srv')
		RETURNING id::text
	`, e.tenantID, e.nodeID, gameSlug).Scan(&id); err != nil {
		t.Fatalf("сервер: %v", err)
	}
	return id
}

func TestAssignGivesDistinctPorts(t *testing.T) {
	env := setupEnv(t)
	ctx := context.Background()

	first := env.newServer(t, "cs2")
	second := env.newServer(t, "cs2")

	p1, err := Assign(ctx, env.pool, env.tenantID, env.nodeID, first, "cs2")
	if err != nil {
		t.Fatalf("первый Assign: %v", err)
	}
	p2, err := Assign(ctx, env.pool, env.tenantID, env.nodeID, second, "cs2")
	if err != nil {
		t.Fatalf("второй Assign: %v", err)
	}
	if p1 == p2 {
		t.Fatalf("оба сервера получили порт %d", p1)
	}
	if p1 != 27015 || p2 != 27016 {
		t.Errorf("порты %d и %d, ожидались 27015 и 27016 из диапазона игры", p1, p2)
	}

	var stored int
	if err := env.pool.QueryRow(ctx, `SELECT primary_port FROM core.servers WHERE id = $1::uuid`, first).Scan(&stored); err != nil {
		t.Fatalf("чтение primary_port: %v", err)
	}
	if stored != p1 {
		t.Errorf("primary_port = %d, ожидалось %d", stored, p1)
	}
}

func TestAssignWritesPortBlock(t *testing.T) {
	env := setupEnv(t)
	ctx := context.Background()

	serverID := env.newServer(t, "palworld")
	primary, err := Assign(ctx, env.pool, env.tenantID, env.nodeID, serverID, "palworld")
	if err != nil {
		t.Fatalf("Assign: %v", err)
	}

	rows, err := env.pool.Query(ctx, `
		SELECT port, protocol, purpose FROM core.server_ports
		WHERE server_id = $1::uuid ORDER BY port, protocol
	`, serverID)
	if err != nil {
		t.Fatalf("чтение портов: %v", err)
	}
	defer rows.Close()

	var got []Port
	for rows.Next() {
		var p Port
		if rows.Scan(&p.Port, &p.Protocol, &p.Purpose) == nil {
			got = append(got, p)
		}
	}
	if len(got) != 3 {
		t.Fatalf("портов %d, ожидалось 3: %+v", len(got), got)
	}
	if got[0].Port != primary || got[1].Port != primary+1 || got[2].Port != primary+10 {
		t.Errorf("блок портов = %+v при основном %d", got, primary)
	}
}

func TestAssignIsIdempotent(t *testing.T) {
	env := setupEnv(t)
	ctx := context.Background()

	serverID := env.newServer(t, "palworld")
	first, err := Assign(ctx, env.pool, env.tenantID, env.nodeID, serverID, "palworld")
	if err != nil {
		t.Fatalf("первый Assign: %v", err)
	}
	second, err := Assign(ctx, env.pool, env.tenantID, env.nodeID, serverID, "palworld")
	if err != nil {
		t.Fatalf("повторный Assign: %v", err)
	}
	if first != second {
		t.Errorf("порт сменился с %d на %d", first, second)
	}

	var count int
	if err := env.pool.QueryRow(ctx, `SELECT count(*) FROM core.server_ports WHERE server_id = $1::uuid`, serverID).Scan(&count); err != nil {
		t.Fatalf("подсчёт портов: %v", err)
	}
	if count != 3 {
		t.Errorf("строк портов %d, ожидалось 3", count)
	}
}

func TestAssignRespectsGameRange(t *testing.T) {
	env := setupEnv(t)
	ctx := context.Background()

	for i := 0; i < 16; i++ {
		id := env.newServer(t, "cs2")
		if _, err := Assign(ctx, env.pool, env.tenantID, env.nodeID, id, "cs2"); err != nil {
			t.Fatalf("сервер %d: %v", i, err)
		}
	}
	overflow := env.newServer(t, "cs2")
	if _, err := Assign(ctx, env.pool, env.tenantID, env.nodeID, overflow, "cs2"); err == nil {
		t.Fatal("17-й сервер должен получить ошибку, а не порт вне диапазона")
	}
}
