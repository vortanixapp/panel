// Package tenantpools раздаёт подключения к базам арендаторов фоновым
// сервисам. Обработчики core-api выбирают базу по заголовку запроса, а у
// воркера и приёмников метрик запроса нет — они идут от данных, поэтому базы
// им нужно перебирать самим.
package tenantpools

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Pools struct {
	central    *pgxpool.Pool
	centralDSN string

	mu     sync.RWMutex
	byName map[string]*pgxpool.Pool
	byUUID map[string]string
}

func New(central *pgxpool.Pool, centralDSN string) *Pools {
	return &Pools{
		central:    central,
		centralDSN: centralDSN,
		byName:     map[string]*pgxpool.Pool{},
		byUUID:     map[string]string{},
	}
}

func (p *Pools) Central() *pgxpool.Pool { return p.central }

// Names перечисляет базы арендаторов из реестра в центральной базе.
func (p *Pools) Names(ctx context.Context) ([]string, error) {
	rows, err := p.central.Query(ctx, `SELECT db_name FROM core.tenant_databases ORDER BY db_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		out = append(out, name)
	}
	return out, rows.Err()
}

// ByName открывает базу арендатора и запоминает пул: параметры подключения у
// всех те же, отличается только имя базы.
func (p *Pools) ByName(ctx context.Context, dbName string) (*pgxpool.Pool, error) {
	p.mu.RLock()
	pool, ok := p.byName[dbName]
	p.mu.RUnlock()
	if ok {
		return pool, nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	if pool, ok := p.byName[dbName]; ok {
		return pool, nil
	}
	pool, err := pgxpool.New(ctx, replaceDatabase(p.centralDSN, dbName))
	if err != nil {
		return nil, err
	}
	p.byName[dbName] = pool
	return pool, nil
}

// All отдаёт центральную базу и базы всех арендаторов. Центральная нужна:
// арендатор, которого ещё не переселили, живёт в ней.
func (p *Pools) All(ctx context.Context) []*pgxpool.Pool {
	out := []*pgxpool.Pool{p.central}
	names, err := p.Names(ctx)
	if err != nil {
		return out
	}
	for _, name := range names {
		pool, err := p.ByName(ctx, name)
		if err != nil {
			continue
		}
		out = append(out, pool)
	}
	return out
}

// ForTenant выбирает базу по идентификатору арендатора: у метрик и релея на
// руках только он, слага в запросе нет.
//
// Возвращает ошибку, а не центральную базу и не nil. Подмена обходилась
// дорого: heartbeat узла записывался туда, где этого узла нет, UPDATE
// отрабатывал успешно, не задевая строк, и нода числилась офлайн при живой
// связи. Вызывающий обязан либо получить базу, либо отказать в операции.
func (p *Pools) ForTenant(ctx context.Context, tenantID string) (*pgxpool.Pool, error) {
	if strings.TrimSpace(tenantID) == "" {
		return nil, fmt.Errorf("пустой идентификатор арендатора")
	}

	p.mu.RLock()
	dbName, ok := p.byUUID[tenantID]
	p.mu.RUnlock()

	if !ok {
		dbName = p.lookupDBName(ctx, tenantID)
		if dbName == "" {
			return nil, fmt.Errorf("база арендатора %s не найдена", tenantID)
		}
		p.mu.Lock()
		p.byUUID[tenantID] = dbName
		p.mu.Unlock()
	}

	pool, err := p.ByName(ctx, dbName)
	if err != nil {
		return nil, fmt.Errorf("база %s недоступна: %w", dbName, err)
	}
	return pool, nil
}

// lookupDBName ищет базу арендатора по его идентификатору.
//
// В центральной базе строки арендатора может не быть вовсе: при переезде она
// уезжает в его собственную базу, а в реестре остаётся только слаг. Раньше
// поиск шёл соединением с центральной core.tenants и молча не находил
// ничего — вызывающий получал центральный пул, писал в него, и запись
// уходила в никуда: UPDATE выполнялся успешно, не задевая ни одной строки.
func (p *Pools) lookupDBName(ctx context.Context, tenantID string) string {
	var dbName string
	err := p.central.QueryRow(ctx, `
		SELECT d.db_name
		FROM core.tenant_databases d
		JOIN core.tenants t ON t.slug = d.slug
		WHERE t.id = $1
	`, tenantID).Scan(&dbName)
	if err == nil {
		return dbName
	}

	names, err := p.Names(ctx)
	if err != nil {
		return ""
	}
	for _, name := range names {
		pool, err := p.ByName(ctx, name)
		if err != nil {
			continue
		}
		var exists bool
		if pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM core.tenants WHERE id = $1)`,
			tenantID).Scan(&exists) == nil && exists {
			return name
		}
	}
	return ""
}

func (p *Pools) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, pool := range p.byName {
		pool.Close()
	}
	p.byName = map[string]*pgxpool.Pool{}
}

// replaceDatabase меняет в DSN только имя базы: хост, пользователь и параметры
// у арендаторов те же, что у центральной.
func replaceDatabase(dsn, dbName string) string {
	query := ""
	if idx := strings.IndexByte(dsn, '?'); idx >= 0 {
		query = dsn[idx:]
		dsn = dsn[:idx]
	}
	if idx := strings.LastIndexByte(dsn, '/'); idx >= 0 {
		dsn = dsn[:idx+1] + dbName
	}
	return dsn + query
}
