// Пакет выдаёт подключение к базе конкретного арендатора.
//
// У каждого арендатора своя база: панели живут на машинах клиентов, но данные
// держим мы, и смешивать их в одной базе нельзя — клиент должен получать доступ
// к своей и только к своей. Реестр «арендатор → база» лежит в центральной базе,
// в core.tenant_databases: отдельно от core.tenants, потому что база заводится
// при установке панели, а запись об арендаторе появляется позже, при активации.
package tenantdb

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Имя базы собирается из слага, поэтому слаг обязан быть безобидным: он попадёт
// в CREATE DATABASE, который параметры не принимает.
var safeSlug = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,48}[a-z0-9]$`)

type Registry struct {
	central       *pgxpool.Pool
	centralDSN    string
	migrationsDir string

	mu     sync.RWMutex
	pools  map[string]*pgxpool.Pool
	bySlug map[string]string
}

func New(central *pgxpool.Pool, centralDSN, migrationsDir string) *Registry {
	return &Registry{
		central:       central,
		centralDSN:    centralDSN,
		migrationsDir: migrationsDir,
		pools:         map[string]*pgxpool.Pool{},
		bySlug:        map[string]string{},
	}
}

// PoolForTenantID выбирает базу по идентификатору арендатора из его токена.
//
// Заголовку X-Tenant-Slug доверять для этого нельзя: панель может его не
// прислать — и тогда запрос молча уходил в центральную базу, где арендатора
// нет. Именно так терялись настройки: INSERT падал на внешнем ключе, а ошибка
// проглатывалась. Личность в токене известна всегда, поэтому идём от неё.
func (r *Registry) PoolForTenantID(ctx context.Context, tenantID string) *pgxpool.Pool {
	if strings.TrimSpace(tenantID) == "" {
		return nil
	}

	r.mu.RLock()
	slug, ok := r.bySlug[tenantID]
	r.mu.RUnlock()
	if ok {
		pool, err := r.Pool(ctx, slug)
		if err != nil {
			return nil
		}
		return pool
	}

	slugs, err := r.Slugs(ctx)
	if err != nil {
		return nil
	}
	for _, s := range slugs {
		pool, err := r.Pool(ctx, s)
		if err != nil {
			continue
		}
		var exists bool
		if pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM core.tenants WHERE id = $1)`,
			tenantID).Scan(&exists) != nil || !exists {
			continue
		}
		r.mu.Lock()
		r.bySlug[tenantID] = s
		r.mu.Unlock()
		return pool
	}
	return nil
}

var ErrUnknownTenant = fmt.Errorf("арендатор не найден")

// Pool возвращает пул базы арендатора, открывая его при первом обращении.
func (r *Registry) Pool(ctx context.Context, slug string) (*pgxpool.Pool, error) {
	r.mu.RLock()
	pool, ok := r.pools[slug]
	r.mu.RUnlock()
	if ok {
		return pool, nil
	}

	dbName, err := r.dbNameOf(ctx, slug)
	if err != nil {
		return nil, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if pool, ok := r.pools[slug]; ok {
		return pool, nil
	}
	pool, err = pgxpool.New(ctx, r.dsnFor(dbName))
	if err != nil {
		return nil, fmt.Errorf("база арендатора %s недоступна: %w", slug, err)
	}
	r.pools[slug] = pool
	return pool, nil
}

func (r *Registry) dbNameOf(ctx context.Context, slug string) (string, error) {
	var dbName string
	err := r.central.QueryRow(ctx,
		`SELECT db_name FROM core.tenant_databases WHERE slug = $1`, slug).Scan(&dbName)
	if err != nil {
		return "", ErrUnknownTenant
	}
	return dbName, nil
}

// Provision заводит базу арендатора и накатывает в неё схему. Вызывается один
// раз при активации панели.
type Credentials struct {
	DBName   string
	User     string
	Password string
}

func (r *Registry) Provision(ctx context.Context, slug string) (string, error) {
	creds, err := r.ProvisionWithUser(ctx, slug)
	return creds.DBName, err
}

// made помнит, что успел завести именно этот вызов: при обрыве убирать можно
// только своё. База, существовавшая до вызова, может хранить данные клиента.
type made struct {
	db   bool
	role bool
}

// ProvisionWithUser заводит базу и отдельную роль для клиента: доступ к своей
// базе он получает под ней, а не под ролью приложения.
//
// Провизионирование считается состоявшимся только когда база попала в реестр —
// эта запись делается последней. Если её нет, а база и роль уже есть, значит
// прошлая попытка оборвалась на полпути: пароль роли тогда не сохранил никто,
// и повтор обязан выдать новый, иначе биллинг вечно получал пустой и клиент
// оставался без доступа к собственной базе.
func (r *Registry) ProvisionWithUser(ctx context.Context, slug string) (Credentials, error) {
	if !safeSlug.MatchString(slug) {
		return Credentials{}, fmt.Errorf("недопустимое имя арендатора: %q", slug)
	}
	dbName := "vx_" + strings.ReplaceAll(slug, "-", "_")
	role := dbName + "_owner"

	var registered bool
	if err := r.central.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM core.tenant_databases WHERE slug = $1)`, slug,
	).Scan(&registered); err != nil {
		return Credentials{}, fmt.Errorf("реестр баз недоступен: %w", err)
	}

	var done made
	creds, err := r.provision(ctx, slug, dbName, role, registered, &done)
	if err != nil {
		r.undoProvision(ctx, dbName, role, done)
		return Credentials{}, err
	}
	return creds, nil
}

// provision выполняет саму работу и отмечает в done всё, что завёл: снаружи по
// этим отметкам убирают за неудачей.
func (r *Registry) provision(
	ctx context.Context, slug, dbName, role string, registered bool, done *made,
) (Credentials, error) {
	// CREATE DATABASE нельзя выполнить внутри транзакции и нельзя параметризовать,
	// поэтому имя собрано выше и проверено регулярным выражением.
	if _, err := r.central.Exec(ctx, `CREATE DATABASE `+quoteIdent(dbName)); err != nil {
		if !strings.Contains(err.Error(), "already exists") {
			return Credentials{}, fmt.Errorf("не удалось создать базу %s: %w", dbName, err)
		}
	} else {
		done.db = true
	}

	// Отметку ставим до проверки ошибки: роль могла создаться, а споткнуться
	// уже выдача прав — тогда убирать её всё равно нам.
	password, createdRole, err := r.ensureRole(ctx, role, dbName, registered)
	done.role = createdRole
	if err != nil {
		return Credentials{}, err
	}

	pool, err := pgxpool.New(ctx, r.dsnFor(dbName))
	if err != nil {
		return Credentials{}, fmt.Errorf("новая база %s недоступна: %w", dbName, err)
	}
	// Закрыть пул нужно до уборки: DROP DATABASE упрётся в наше же подключение.
	defer pool.Close()

	if err := Migrate(ctx, pool, r.migrationsDir); err != nil {
		return Credentials{}, err
	}
	if _, err := pool.Exec(ctx,
		`GRANT USAGE ON SCHEMA core TO `+quoteIdent(role)+`;
		 GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA core TO `+quoteIdent(role)+`;
		 GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA core TO `+quoteIdent(role)+`;
		 ALTER DEFAULT PRIVILEGES IN SCHEMA core
		     GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO `+quoteIdent(role)+`;`,
	); err != nil {
		return Credentials{}, fmt.Errorf("права роли %s не выданы: %w", role, err)
	}

	if _, err := r.central.Exec(ctx, `
		INSERT INTO core.tenant_databases (slug, db_name) VALUES ($1, $2)
		ON CONFLICT (slug) DO UPDATE SET db_name = EXCLUDED.db_name
	`, slug, dbName); err != nil {
		return Credentials{}, fmt.Errorf("база %s не попала в реестр: %w", dbName, err)
	}
	return Credentials{DBName: dbName, User: role, Password: password}, nil
}

// undoProvision убирает за оборвавшейся попыткой ровно то, что она успела
// завести: недоделанная база с половиной миграций и роль без прав на следующем
// заходе только мешают.
//
// Контекст берём свой: обрыв часто и есть истёкший таймаут вызова, а на таком
// контексте уборка не выполнилась бы вовсе.
func (r *Registry) undoProvision(ctx context.Context, dbName, role string, done made) {
	if !done.db && !done.role {
		return
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
	defer cancel()

	// База — первой: пока она есть, у роли остаётся право CONNECT на неё и
	// DROP ROLE отказывает.
	if done.db {
		if _, err := r.central.Exec(ctx, `DROP DATABASE IF EXISTS `+quoteIdent(dbName)); err != nil {
			log.Printf("недоделанная база %s осталась: %v", dbName, err)
		}
	}
	if done.role {
		if _, err := r.central.Exec(ctx, `DROP ROLE IF EXISTS `+quoteIdent(role)); err != nil {
			log.Printf("роль %s недоделанной базы осталась: %v", role, err)
		}
	}
}

// ensureRole создаёт роль клиента, если её ещё нет, и возвращает пароль вместе
// с признаком «роль завели мы».
//
// У роли состоявшегося арендатора пароль не меняем: его знает тот, кто сохранил
// при первом заведении, а смена оборвала бы живые подключения. Роли, оставшейся
// от неудачной попытки, пароль назначаем заново — прежний не знает никто, и
// подключаться под ней ещё некому.
func (r *Registry) ensureRole(ctx context.Context, role, dbName string, registered bool) (string, bool, error) {
	var exists bool
	if err := r.central.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM pg_roles WHERE rolname = $1)`, role).Scan(&exists); err != nil {
		return "", false, err
	}
	if exists && registered {
		return "", false, nil
	}

	password, err := randomPassword()
	if err != nil {
		return "", false, err
	}
	if exists {
		if _, err := r.central.Exec(ctx,
			`ALTER ROLE `+quoteIdent(role)+` PASSWORD `+quoteLiteral(password)); err != nil {
			return "", false, fmt.Errorf("пароль роли %s не назначен заново: %w", role, err)
		}
	} else if _, err := r.central.Exec(ctx,
		`CREATE ROLE `+quoteIdent(role)+` LOGIN PASSWORD `+quoteLiteral(password)); err != nil {
		return "", false, fmt.Errorf("роль %s не создана: %w", role, err)
	}

	if _, err := r.central.Exec(ctx,
		`GRANT CONNECT ON DATABASE `+quoteIdent(dbName)+` TO `+quoteIdent(role)); err != nil {
		return "", !exists, fmt.Errorf("роли %s не выдан доступ к базе: %w", role, err)
	}
	return password, !exists, nil
}

func randomPassword() (string, error) {
	buf := make([]byte, 18)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func quoteLiteral(v string) string {
	return "'" + strings.ReplaceAll(v, "'", "''") + "'"
}

// Migrate накатывает миграции core в порядке имён файлов и ведёт тот же журнал,
// что и scripts/migrate-core-remote.sh, чтобы обе дороги сходились.
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
		return fmt.Errorf("в %s нет миграций", dir)
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

func (r *Registry) dsnFor(dbName string) string {
	return replaceDatabase(r.centralDSN, dbName)
}

// replaceDatabase меняет имя базы в DSN, не трогая всё остальное: параметры
// подключения у арендаторов те же, отличается только база.
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

func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func (r *Registry) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, pool := range r.pools {
		pool.Close()
	}
	r.pools = map[string]*pgxpool.Pool{}
}

// Slugs перечисляет арендаторов, у которых заведена база.
func (r *Registry) Slugs(ctx context.Context) ([]string, error) {
	rows, err := r.central.Query(ctx, `SELECT slug FROM core.tenant_databases ORDER BY slug`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var slug string
		if err := rows.Scan(&slug); err != nil {
			return nil, err
		}
		out = append(out, slug)
	}
	return out, rows.Err()
}

// MigrateAll докатывает схему в базы всех арендаторов. Нужен на старте: схема
// меняется вместе с кодом, а базы клиентов заведены раньше и сами о правках не
// узнают.
func (r *Registry) MigrateAll(ctx context.Context) {
	slugs, err := r.Slugs(ctx)
	if err != nil {
		log.Printf("список баз арендаторов не получен: %v", err)
		return
	}
	for _, slug := range slugs {
		pool, err := r.Pool(ctx, slug)
		if err != nil {
			log.Printf("база арендатора %s недоступна: %v", slug, err)
			continue
		}
		if err := Migrate(ctx, pool, r.migrationsDir); err != nil {
			log.Printf("схема арендатора %s не обновлена: %v", slug, err)
		}
	}
}

// Drop удаляет базу арендатора вместе с его ролью. Операция необратима:
// восстановить данные будет нечем.
func (r *Registry) Drop(ctx context.Context, slug string) error {
	dbName, err := r.dbNameOf(ctx, slug)
	if err != nil {
		return err
	}

	// Свой пул закрываем первым, иначе DROP DATABASE упрётся в наши же
	// подключения.
	r.mu.Lock()
	if pool, ok := r.pools[slug]; ok {
		pool.Close()
		delete(r.pools, slug)
	}
	r.mu.Unlock()

	if _, err := r.central.Exec(ctx, `
		SELECT pg_terminate_backend(pid) FROM pg_stat_activity
		WHERE datname = $1 AND pid <> pg_backend_pid()
	`, dbName); err != nil {
		return fmt.Errorf("подключения к базе %s не разорваны: %w", dbName, err)
	}

	if _, err := r.central.Exec(ctx, `DROP DATABASE IF EXISTS `+quoteIdent(dbName)); err != nil {
		return fmt.Errorf("база %s не удалена: %w", dbName, err)
	}
	if _, err := r.central.Exec(ctx, `DROP ROLE IF EXISTS `+quoteIdent(dbName+"_owner")); err != nil {
		log.Printf("роль %s_owner осталась: %v", dbName, err)
	}
	if _, err := r.central.Exec(ctx,
		`DELETE FROM core.tenant_databases WHERE slug = $1`, slug); err != nil {
		return fmt.Errorf("запись о базе %s осталась в реестре: %w", dbName, err)
	}
	return nil
}

// Pools отдаёт базы всех арендаторов, открывая недостающие подключения.
func (r *Registry) Pools(ctx context.Context) []*pgxpool.Pool {
	slugs, err := r.Slugs(ctx)
	if err != nil {
		log.Printf("список баз арендаторов не получен: %v", err)
		return nil
	}

	out := make([]*pgxpool.Pool, 0, len(slugs))
	for _, slug := range slugs {
		pool, err := r.Pool(ctx, slug)
		if err != nil {
			log.Printf("база арендатора %s недоступна: %v", slug, err)
			continue
		}
		out = append(out, pool)
	}
	return out
}
