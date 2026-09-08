package licensestate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/vortanix/vortanix/internal/api/licenseclient"
	"github.com/vortanix/vortanix/internal/api/licensejwt"
)

const refreshBefore = 6 * time.Hour

const advisoryLockKey int64 = 8021977312041

type Config struct {
	PollInterval    time.Duration
	Grace           time.Duration
	Version         string
	AllowExpiredFor time.Duration
}

type UsageFunc func(ctx context.Context) (servers, nodes, admins, cpu int)

type Refresher struct {
	db     *pgxpool.Pool
	store  *Store
	client *licenseclient.Client
	verify *licensejwt.Verifier
	cfg    Config
	usage  UsageFunc

	backoff time.Duration
	kick    chan struct{}
	pools   PoolsFunc

	// onUpdate вызывается, когда сервис лицензий назвал новую целевую версию.
	// Само оповещение живёт в handlers: пакету состояния лицензии незачем
	// знать ни о почте, ни о получателях.
	onUpdate UpdateNoticeFunc
}

// UpdateNoticeFunc — «появилась новая версия панели»: база арендатора, версия
// и заметки к релизу.
type UpdateNoticeFunc func(ctx context.Context, db *pgxpool.Pool, target, notes string)

func New(db *pgxpool.Pool, store *Store, client *licenseclient.Client, verify *licensejwt.Verifier, cfg Config, usage UsageFunc) *Refresher {
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = time.Minute
	}
	if cfg.Grace <= 0 {
		cfg.Grace = 24 * time.Hour
	}
	if cfg.AllowExpiredFor <= 0 {
		cfg.AllowExpiredFor = 30 * 24 * time.Hour
	}
	return &Refresher{
		db:     db,
		store:  store,
		client: client,
		verify: verify,
		cfg:    cfg,
		usage:  usage,
		kick:   make(chan struct{}, 1),
	}
}

func (r *Refresher) Start(ctx context.Context) {
	r.tick(ctx)

	go func() {
		for {
			delay := r.cfg.PollInterval
			if r.backoff > 0 {
				delay = r.backoff
			}
			delay += time.Duration(rand.Int63n(int64(delay / 10)))

			select {
			case <-ctx.Done():
				return
			case <-time.After(delay):
			case <-r.kick:
			}
			r.tick(ctx)
		}
	}()
}

func (r *Refresher) RefreshNow(ctx context.Context) {
	select {
	case r.kick <- struct{}{}:
	default:
	}
}

// Pools отдаёт базы арендаторов. Задаётся снаружи, чтобы обновлятель не знал
// про реестр баз: ему достаточно списка.
type PoolsFunc func(ctx context.Context) []*pgxpool.Pool

func (r *Refresher) SetTenantPools(f PoolsFunc) { r.pools = f }

// SetUpdateNotice задаёт обработчик появления новой версии панели.
func (r *Refresher) SetUpdateNotice(f UpdateNoticeFunc) { r.onUpdate = f }

func (r *Refresher) tick(ctx context.Context) {
	r.refresh(ctx, r.db, true)

	// Панели клиентов живут в своих базах, и состояние лицензии у каждой своё.
	// Без этого обхода их строка остаётся такой, какой была в момент активации:
	// панель никогда не узнает ни о новом релизе, ни об изменении статуса.
	if r.pools == nil {
		return
	}
	for _, pool := range r.pools(ctx) {
		r.refresh(ctx, pool, false)
	}
}

// RefreshTenant опрашивает сервис лицензий для одной базы и ждёт результата:
// кнопка «проверить сейчас» должна показывать свежие данные, а не обещание.
func (r *Refresher) RefreshTenant(ctx context.Context, db *pgxpool.Pool) {
	r.refresh(ctx, db, db == r.db)
}

func (r *Refresher) refresh(ctx context.Context, db *pgxpool.Pool, publish bool) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	row, err := LoadRow(ctx, db)
	if err != nil {
		log.Printf("license: не удалось прочитать состояние установки: %v", err)
		return
	}

	if row.LicenseKey == "" || row.LicenseStatus == "legacy" {
		r.publishIf(publish, row, nil)
		return
	}

	claims := r.parseStored(row)

	release, leader := r.acquireLeader(ctx)
	if !leader {
		r.publishIf(publish, row, claims)
		return
	}
	defer release()

	if err := r.probe(ctx, db, &row, claims); err != nil {
		// Установка без активации — обычное дело после переустановки панели:
		// идентификатор сменился, а прежняя запись снята. Активируемся заново
		// сохранённым ключом, иначе панель осталась бы с мёртвой лицензией и
		// ждала бы человека, который зайдёт на страницу активации.
		if r.reactivate(ctx, db, row, err) {
			if fresh, err := LoadRow(ctx, db); err == nil {
				row = fresh
				claims = r.parseStored(row)
				r.backoff = 0
				r.publishIf(publish, row, claims)
				return
			}
		}
		r.noteFailure(ctx, db, err)
		r.publishIf(publish, row, claims)
		return
	}

	r.backoff = 0
	claims = r.parseStored(row)
	r.publishIf(publish, row, claims)
}

// publishIf обновляет состояние в памяти только для своей установки: чужое там
// ничего не значит и затирало бы наше.
func (r *Refresher) publishIf(own bool, row Row, claims *licensejwt.Claims) {
	if own {
		r.publish(row, claims)
	}
}

func (r *Refresher) probe(ctx context.Context, db *pgxpool.Pool, row *Row, claims *licensejwt.Claims) error {
	stateJWT, err := r.client.State(ctx, row.LicenseToken, r.stateRequest(ctx, *row))
	if err != nil {
		return err
	}

	state, err := r.verify.ParseState(stateJWT)
	if err != nil {
		return err
	}

	if err := SaveProbe(ctx, db, state.Status, state.Revision, state.LicenseExpiresAt(), ""); err != nil {
		log.Printf("license: не удалось сохранить результат опроса: %v", err)
	}
	row.LicenseStatus = state.Status
	row.Revision = state.Revision
	row.LicenseExpiresAt = state.LicenseExpiresAt()
	r.applyUpdateState(ctx, db, row, state.Update)

	needToken := claims == nil ||
		state.Revision != claims.Revision ||
		time.Until(row.TokenExpiresAt) < refreshBefore
	if !needToken {
		return nil
	}

	refreshed, err := r.client.Refresh(ctx, row.LicenseToken)
	if err != nil {
		return err
	}
	fresh, err := r.verify.ParseAllowExpired(refreshed.LicenseToken, r.cfg.AllowExpiredFor)
	if err != nil {
		return err
	}
	if err := SaveToken(ctx, db, refreshed.LicenseToken, fresh); err != nil {
		return err
	}

	row.LicenseToken = refreshed.LicenseToken
	log.Printf("license: получен новый токен, статус %s, ревизия %d", fresh.Status, fresh.Revision)
	return nil
}

func (r *Refresher) stateRequest(ctx context.Context, row Row) licenseclient.StateRequest {
	req := licenseclient.StateRequest{
		RequestID: uuid.NewString(),
		Version:   r.cfg.Version,
		Revision:  row.Revision,
		Domain:    row.Domain,
	}
	if row.UpdateAction != "" {
		req.UpdateAction = row.UpdateAction
		req.UpdateComponent = row.UpdateComponent
		if row.UpdateDeferUntil != nil {
			req.DeferUpdateUntil = row.UpdateDeferUntil.Unix()
		}
	}
	if r.usage != nil {
		servers, nodes, admins, cpu := r.usage(ctx)
		req.ServersUsed = &servers
		req.NodesUsed = &nodes
		req.AdminsUsed = &admins
		req.CPULoadPercent = &cpu
	}
	return req
}

func (r *Refresher) applyUpdateState(ctx context.Context, db *pgxpool.Pool, row *Row, upd *licensejwt.UpdateState) {
	var (
		target    string
		mandatory *time.Time
		notes     string
		auto      bool
	)
	if upd != nil {
		target = upd.TargetVersion
		mandatory = upd.MandatoryAt()
		notes = upd.Notes
		auto = upd.AutoUpdate
	}

	// Сравниваем до записи: после SaveUpdateState прежней цели уже не узнать, а
	// без сравнения оповещение уходило бы на каждой сверке с сервисом лицензий.
	newTarget := target != "" && target != row.UpdateTargetVersion

	if err := SaveUpdateState(ctx, db, target, mandatory, notes, auto); err != nil {
		log.Printf("license: не удалось сохранить состояние обновления: %v", err)
		return
	}

	if newTarget && r.onUpdate != nil {
		r.onUpdate(ctx, db, target, notes)
	}

	row.UpdateTargetVersion = target
	row.UpdateMandatoryAfter = mandatory
	row.UpdateNotes = notes
	row.UpdateAuto = auto
	row.UpdateAction = ""
	row.UpdateComponent = ""
}

func (r *Refresher) parseStored(row Row) *licensejwt.Claims {
	if row.LicenseToken == "" {
		return nil
	}
	claims, err := r.verify.ParseAllowExpired(row.LicenseToken, r.cfg.AllowExpiredFor)
	if err != nil {
		if !errors.Is(err, licensejwt.ErrTokenTooOld) {
			log.Printf("license: сохранённый токен не проходит проверку: %v", err)
		}
		return nil
	}
	return claims
}

func (r *Refresher) publish(row Row, claims *licensejwt.Claims) {
	state := Derive(row, claims, time.Now(), r.cfg.Grace)
	prev := r.store.Load()
	r.store.Set(state)

	if prev == nil || prev.Mode != state.Mode || prev.LicenseStatus != state.LicenseStatus {
		log.Printf("license: режим %s, статус %s%s", state.Mode, state.LicenseStatus, reasonSuffix(state.Reason))
	}
}

func reasonSuffix(reason string) string {
	if reason == "" {
		return ""
	}
	return " (" + reason + ")"
}

func (r *Refresher) noteFailure(ctx context.Context, db *pgxpool.Pool, cause error) {
	var apiErr *licenseclient.Error
	rejected := errors.As(cause, &apiErr) && apiErr.Rejected()

	switch {
	case r.backoff == 0:
		r.backoff = 2 * time.Minute
	case r.backoff < 5*time.Minute:
		r.backoff = 5 * time.Minute
	case r.backoff < 10*time.Minute:
		r.backoff = 10 * time.Minute
	default:
		r.backoff = 15 * time.Minute
	}
	if rejected {
		r.backoff = 15 * time.Minute
	}

	log.Printf("license: опрос не удался (следующая попытка через %s): %v", r.backoff, cause)
	if err := SaveProbe(ctx, db, "", 0, nil, cause.Error()); err != nil {
		log.Printf("license: не удалось сохранить ошибку опроса: %v", err)
	}
}

func (r *Refresher) acquireLeader(ctx context.Context) (release func(), ok bool) {
	conn, err := r.db.Acquire(ctx)
	if err != nil {
		return nil, false
	}

	var locked bool
	if err := conn.QueryRow(ctx, `SELECT pg_try_advisory_lock($1)`, advisoryLockKey).Scan(&locked); err != nil || !locked {
		conn.Release()
		return nil, false
	}

	return func() {
		unlockCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = conn.Exec(unlockCtx, `SELECT pg_advisory_unlock($1)`, advisoryLockKey)
		conn.Release()
	}, true
}

func (r *Refresher) Activate(ctx context.Context, licenseKey, domain string) error {
	return r.ActivateIn(ctx, r.db, licenseKey, domain)
}

// ActivateIn привязывает лицензию к установке в указанной базе. Панель
// каждого клиента активируется в своей: установка там одна, и лицензия
// ложится на неё, а не на нашу собственную в центральной базе.
func (r *Refresher) ActivateIn(ctx context.Context, db *pgxpool.Pool, licenseKey, domain string) error {
	row, err := EnsureRow(ctx, db)
	if err != nil {
		return err
	}

	resp, err := r.client.Activate(ctx, licenseclient.ActivateRequest{
		LicenseKey:     licenseKey,
		InstallationID: row.ID,
		Domain:         domain,
		Fingerprint:    Fingerprint(row.ID, domain),
	})
	if err != nil {
		var apiErr *licenseclient.Error
		if errors.As(err, &apiErr) && apiErr.Rejected() {
			return errors.New("лицензия недействительна: проверьте ключ")
		}
		return err
	}

	claims, err := r.verify.ParseAllowExpired(resp.LicenseToken, r.cfg.AllowExpiredFor)
	if err != nil {
		return fmt.Errorf("сервис лицензий выдал токен, который не проходит проверку: %w", err)
	}

	if err := SaveKey(ctx, db, licenseKey, domain); err != nil {
		return err
	}
	if err := SaveToken(ctx, db, resp.LicenseToken, claims); err != nil {
		return err
	}

	fresh, err := LoadRow(ctx, db)
	if err != nil {
		return err
	}
	// В памяти держим состояние только своей установки: чужое там ничего
	// не значит и затирало бы наше.
	if db == r.db {
		r.publish(fresh, claims)
	}
	return nil
}

func Fingerprint(installationID, domain string) string {
	host, _ := os.Hostname()
	sum := sha256.Sum256([]byte(installationID + "|" + domain + "|" + host))
	return hex.EncodeToString(sum[:])
}

// reactivate пытается заново привязать лицензию к этой установке. Делается
// только при отказе сервиса лицензий: молча переактивировать на каждой ошибке
// нельзя, иначе отозванный ключ оживал бы сам.
func (r *Refresher) reactivate(ctx context.Context, db *pgxpool.Pool, row Row, cause error) bool {
	var apiErr *licenseclient.Error
	if !errors.As(cause, &apiErr) || apiErr.StatusCode != http.StatusForbidden {
		return false
	}
	if row.LicenseKey == "" || row.Domain == "" {
		return false
	}

	if err := r.ActivateIn(ctx, db, row.LicenseKey, row.Domain); err != nil {
		log.Printf("license: повторная активация не удалась: %v", err)
		return false
	}
	log.Printf("license: установка активирована заново по домену %s", row.Domain)
	return true
}

// UpdateEvents отдаёт журнал обновления по ключу лицензии. Пустой компонент —
// все события сразу.
func (r *Refresher) UpdateEvents(ctx context.Context, licenseKey, component string) ([]licenseclient.UpdateEvent, error) {
	return r.client.UpdateEvents(ctx, licenseKey, component)
}

// UpdateReleases отдаёт историю версий компонента.
func (r *Refresher) UpdateReleases(ctx context.Context, licenseKey, component string) (licenseclient.ComponentReleases, error) {
	return r.client.UpdateReleases(ctx, licenseKey, component)
}

// ReportBug передаёт отчёт об ошибке панели разработчику.
func (r *Refresher) ReportBug(ctx context.Context, report licenseclient.BugReport) (int64, error) {
	return r.client.ReportBug(ctx, report)
}
