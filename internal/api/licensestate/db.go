package licensestate

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/vortanix/vortanix/internal/api/licensejwt"
)

type Row struct {
	ID               string
	LicenseKey       string
	LicenseToken     string
	TokenExpiresAt   time.Time
	LicenseExpiresAt *time.Time
	LicenseStatus    string
	Plan             string
	Revision         int64
	MaxServers       *int
	MaxNodes         *int
	MaxAdmins        *int
	APIRPM           *int
	PublicKeyPEM     string
	Domain           string
	LastVerifiedAt   time.Time
	LastError        string

	UpdateTargetVersion  string
	UpdateAuto           bool
	UpdateMandatoryAfter *time.Time
	UpdateNotes          string
	UpdateDeferUntil     *time.Time
	UpdateAction         string
	UpdateComponent      string
}

func (r Row) Limits() Limits {
	return Limits{
		MaxServers: derefOr(r.MaxServers, 0),
		MaxNodes:   derefOr(r.MaxNodes, 0),
		MaxAdmins:  derefOr(r.MaxAdmins, 0),
		APIRPM:     derefOr(r.APIRPM, 0),
	}
}

func (r Row) HasLimits() bool {
	return r.MaxServers != nil && r.MaxNodes != nil
}

func derefOr(p *int, fallback int) int {
	if p == nil {
		return fallback
	}
	return *p
}

const rowProjection = `
	SELECT id::text, COALESCE(license_key, ''), COALESCE(license_token, ''),
	       COALESCE(token_expires_at, 'epoch'::timestamptz), license_expires_at,
	       license_status, COALESCE(plan, ''), revision,
	       max_servers, max_nodes, max_admins, api_rpm,
	       COALESCE(license_public_key, ''), COALESCE(domain, ''),
	       COALESCE(last_verified_at, 'epoch'::timestamptz), COALESCE(last_error, ''),
	       COALESCE(update_target_version, ''), update_mandatory_after,
	       COALESCE(update_notes, ''), update_defer_until, COALESCE(update_action, ''),
	       COALESCE(update_component, ''), COALESCE(update_auto, false)
	FROM core.installation WHERE singleton
`

func EnsureRow(ctx context.Context, db *pgxpool.Pool) (Row, error) {
	if _, err := db.Exec(ctx, `
		INSERT INTO core.installation (singleton) VALUES (true) ON CONFLICT DO NOTHING
	`); err != nil {
		return Row{}, err
	}
	return LoadRow(ctx, db)
}

func LoadRow(ctx context.Context, db *pgxpool.Pool) (Row, error) {
	var r Row
	err := db.QueryRow(ctx, rowProjection).Scan(
		&r.ID, &r.LicenseKey, &r.LicenseToken, &r.TokenExpiresAt, &r.LicenseExpiresAt,
		&r.LicenseStatus, &r.Plan, &r.Revision,
		&r.MaxServers, &r.MaxNodes, &r.MaxAdmins, &r.APIRPM,
		&r.PublicKeyPEM, &r.Domain, &r.LastVerifiedAt, &r.LastError,
		&r.UpdateTargetVersion, &r.UpdateMandatoryAfter, &r.UpdateNotes,
		&r.UpdateDeferUntil, &r.UpdateAction, &r.UpdateComponent, &r.UpdateAuto,
	)
	return r, err
}

func SaveKey(ctx context.Context, db *pgxpool.Pool, licenseKey, domain string) error {
	_, err := db.Exec(ctx, `
		UPDATE core.installation SET
			license_key  = $1,
			domain       = COALESCE(NULLIF($2, ''), domain),
			activated_at = COALESCE(activated_at, now()),
			updated_at   = now()
		WHERE singleton
	`, licenseKey, domain)
	return err
}

func SaveToken(ctx context.Context, db *pgxpool.Pool, token string, claims *licensejwt.Claims) error {
	var tokenExpires time.Time
	if claims.ExpiresAt != nil {
		tokenExpires = claims.ExpiresAt.Time
	}

	_, err := db.Exec(ctx, `
		UPDATE core.installation SET
			license_token      = $1,
			token_expires_at   = $2,
			license_expires_at = $3,
			license_status     = $4,
			plan               = $5,
			revision           = $6,
			max_servers        = $7,
			max_nodes          = $8,
			max_admins         = $9,
			api_rpm            = $10,
			last_verified_at   = now(),
			last_attempt_at    = now(),
			last_error         = NULL,
			updated_at         = now()
		WHERE singleton
	`, token, tokenExpires, claims.LicenseExpiresAt(), statusOrActive(claims.Status),
		claims.Plan, claims.Revision,
		claims.Limits.MaxServers, claims.Limits.MaxNodes,
		claims.Limits.MaxAdmins, claims.Limits.APIRPM)
	return err
}

func SaveProbe(ctx context.Context, db *pgxpool.Pool, status string, revision int64, licenseExpires *time.Time, probeErr string) error {
	_, err := db.Exec(ctx, `
		UPDATE core.installation SET
			license_status     = COALESCE(NULLIF($1, ''), license_status),
			revision           = CASE WHEN $2 > 0 THEN $2 ELSE revision END,
			license_expires_at = COALESCE($3, license_expires_at),
			last_verified_at   = CASE WHEN $4 = '' THEN now() ELSE last_verified_at END,
			last_attempt_at    = now(),
			last_error         = NULLIF($4, ''),
			updated_at         = now()
		WHERE singleton
	`, status, revision, licenseExpires, probeErr)
	return err
}

func SaveUpdateState(ctx context.Context, db *pgxpool.Pool, target string, mandatoryAfter *time.Time, notes string, auto bool) error {
	_, err := db.Exec(ctx, `
		UPDATE core.installation SET
			update_target_version  = NULLIF($1, ''),
			update_mandatory_after = $2,
			update_notes           = NULLIF($3, ''),
			update_auto            = $4,
			update_action          = NULL,
			updated_at             = now()
		WHERE singleton
	`, target, mandatoryAfter, notes, auto)
	return err
}

// SaveUpdateAuto записывает выбор владельца сразу, не дожидаясь сверки с
// сервисом лицензий.
//
// Без этого переключатель показывал прежнее состояние: страница берёт значение
// из состояния лицензии в памяти, а оно обновляется только после ответа
// сервиса. Между нажатием и ответом экран уверенно рисовал противоположное
// тому, что владелец только что выбрал. Сверка это значение затем подтвердит.
func SaveUpdateAuto(ctx context.Context, db *pgxpool.Pool, enabled bool) error {
	_, err := db.Exec(ctx, `
		UPDATE core.installation SET update_auto = $1, updated_at = now()
		WHERE singleton
	`, enabled)
	return err
}

// LoadUpdateAuto читает выбор владельца из базы.
func LoadUpdateAuto(ctx context.Context, db *pgxpool.Pool) bool {
	var enabled bool
	if err := db.QueryRow(ctx,
		`SELECT COALESCE(update_auto, false) FROM core.installation WHERE singleton`,
	).Scan(&enabled); err != nil {
		return false
	}
	return enabled
}

func SaveUpdateDecision(ctx context.Context, db *pgxpool.Pool, action, component string, deferUntil *time.Time) error {
	_, err := db.Exec(ctx, `
		UPDATE core.installation SET
			update_action      = $1,
			update_defer_until = $2,
			update_component   = $3,
			updated_at         = now()
		WHERE singleton
	`, action, deferUntil, component)
	return err
}

func SavePublicKey(ctx context.Context, db *pgxpool.Pool, pemText string) error {
	_, err := db.Exec(ctx, `
		UPDATE core.installation SET license_public_key = $1, updated_at = now()
		WHERE singleton AND COALESCE(license_public_key, '') <> $1
	`, pemText)
	return err
}

func statusOrActive(status string) string {
	if status == "" {
		return "active"
	}
	return status
}
