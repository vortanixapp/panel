package portalloc

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/vortanixapp/panel/pkg/gamecatalog"
)

type Querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type Port struct {
	Port     int
	Protocol string
	Purpose  string
}

type Allocation struct {
	PrimaryPort int
	Ports       []Port
}

var ErrNoFreePort = fmt.Errorf("на ноде нет свободного порта в диапазоне игры")

const (
	defaultMinPort = 27000
	defaultMaxPort = 28999
)

func Allocate(ctx context.Context, q Querier, tenantID, nodeID, gameSlug string) (Allocation, error) {
	return AllocateOn(ctx, q, tenantID, nodeID, gameSlug, "")
}

func AllocateOn(ctx context.Context, q Querier, tenantID, nodeID, gameSlug, address string) (Allocation, error) {
	minPort, maxPort := gameRange(ctx, q, tenantID, gameSlug)
	span := gamecatalog.PortSpan(gameSlug)

	busy, err := busyPorts(ctx, q, tenantID, nodeID, address)
	if err != nil {
		return Allocation{}, err
	}

	base, ok := pickBase(minPort, maxPort, span, busy)
	if !ok {
		return Allocation{}, ErrNoFreePort
	}
	return Allocation{PrimaryPort: base, Ports: portsFor(base, gamecatalog.PortLayout(gameSlug))}, nil
}

func pickBase(minPort, maxPort, span int, busy map[int]bool) (int, bool) {
	if span < 1 {
		span = 1
	}
	for base := minPort; base+span-1 <= maxPort; base += span {
		if blockFree(base, span, busy) {
			return base, true
		}
	}
	return 0, false
}

func PortsFor(gameSlug string, primaryPort int) []Port {
	return portsFor(primaryPort, gamecatalog.PortLayout(gameSlug))
}

func portsFor(base int, layout []gamecatalog.PortSpec) []Port {
	out := make([]Port, 0, len(layout))
	for _, spec := range layout {
		port := base + spec.Offset
		if port < 1 || port > 65535 {
			continue
		}
		out = append(out, Port{Port: port, Protocol: spec.Protocol, Purpose: spec.Purpose})
	}
	return out
}

func blockFree(base, span int, busy map[int]bool) bool {
	for p := base; p < base+span; p++ {
		if busy[p] {
			return false
		}
	}
	return true
}

func gameRange(ctx context.Context, q Querier, tenantID, gameSlug string) (int, int) {
	minPort, maxPort := defaultMinPort, defaultMaxPort
	slugs := gamecatalog.Slugs(gameSlug)
	var lo, hi int
	err := q.QueryRow(ctx, `
		SELECT COALESCE(min_port, 0), COALESCE(max_port, 0)
		FROM core.games
		WHERE tenant_id = $1 AND slug = ANY($2)
		ORDER BY slug = $3 DESC
		LIMIT 1
	`, tenantID, slugs, gamecatalog.Normalize(gameSlug)).Scan(&lo, &hi)
	if err == nil && lo > 0 && hi > lo {
		minPort, maxPort = lo, hi
	}
	if minPort < 1024 {
		minPort = 1024
	}
	if maxPort > 65535 {
		maxPort = 65535
	}
	return minPort, maxPort
}

func busyPorts(ctx context.Context, q Querier, tenantID, nodeID, address string) (map[int]bool, error) {
	busy := map[int]bool{}

	rows, err := q.Query(ctx, `
		SELECT sp.port
		FROM core.server_ports sp
		JOIN core.servers s ON s.id = sp.server_id
		WHERE s.tenant_id = $1 AND s.node_id = $2::uuid
		  AND ($3 = '' OR COALESCE(s.ip_address, '') = $3)
	`, tenantID, nodeID, address)
	if err != nil {
		return nil, fmt.Errorf("занятые порты: %w", err)
	}
	for rows.Next() {
		var port int
		if rows.Scan(&port) == nil && port > 0 {
			busy[port] = true
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("занятые порты: %w", err)
	}

	legacy, err := q.Query(ctx, `
		SELECT game_id, COALESCE(primary_port, 0)
		FROM core.servers
		WHERE tenant_id = $1 AND node_id = $2::uuid AND COALESCE(primary_port, 0) > 0
		  AND ($3 = '' OR COALESCE(ip_address, '') = $3)
	`, tenantID, nodeID, address)
	if err != nil {
		return nil, fmt.Errorf("занятые порты серверов: %w", err)
	}
	defer legacy.Close()
	for legacy.Next() {
		var slug string
		var port int
		if legacy.Scan(&slug, &port) != nil || port <= 0 {
			continue
		}
		for i := 0; i < gamecatalog.PortSpan(slug); i++ {
			busy[port+i] = true
		}
	}
	return busy, legacy.Err()
}

type Execer interface {
	Querier
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

func Assign(ctx context.Context, db Execer, tenantID, nodeID, serverID, gameSlug string) (int, error) {
	return AssignOn(ctx, db, tenantID, nodeID, serverID, gameSlug, "")
}

func AssignOn(ctx context.Context, db Execer, tenantID, nodeID, serverID, gameSlug, address string) (int, error) {
	var existing int
	_ = db.QueryRow(ctx, `
		SELECT COALESCE(primary_port, 0) FROM core.servers
		WHERE id = $1::uuid AND tenant_id = $2
	`, serverID, tenantID).Scan(&existing)

	primary := existing
	if primary <= 0 {
		alloc, err := AllocateOn(ctx, db, tenantID, nodeID, gameSlug, address)
		if err != nil {
			return 0, err
		}
		primary = alloc.PrimaryPort
		if _, err := db.Exec(ctx, `
			UPDATE core.servers SET primary_port = $2
			WHERE id = $1::uuid AND tenant_id = $3
		`, serverID, primary, tenantID); err != nil {
			return 0, fmt.Errorf("сохранение порта: %w", err)
		}
	}

	for _, p := range PortsFor(gameSlug, primary) {
		if _, err := db.Exec(ctx, `
			INSERT INTO core.server_ports (tenant_id, server_id, port, protocol, purpose)
			VALUES ($1, $2::uuid, $3, $4, $5)
			ON CONFLICT (server_id, port, protocol) DO NOTHING
		`, tenantID, serverID, p.Port, p.Protocol, p.Purpose); err != nil {
			return 0, fmt.Errorf("запись портов сервера: %w", err)
		}
	}
	return primary, nil
}
