package licensestate

import (
	"fmt"
	"time"

	"github.com/vortanix/vortanix/internal/api/licensejwt"
)

func Derive(row Row, claims *licensejwt.Claims, now time.Time, grace time.Duration) *State {
	st := &State{
		InstallationID: row.ID,
		Plan:           row.Plan,
		Domain:         row.Domain,
		LastVerifiedAt: row.LastVerifiedAt,
		LastError:      row.LastError,
		Revision:       row.Revision,
		Activated:      row.LicenseKey != "",
		Update: UpdateInfo{
			TargetVersion:  row.UpdateTargetVersion,
			MandatoryAfter: row.UpdateMandatoryAfter,
			Notes:          row.UpdateNotes,
			DeferUntil:     row.UpdateDeferUntil,
			Auto:           row.UpdateAuto,
		},
	}

	if row.LicenseStatus == "legacy" {
		st.Mode = ModeActive
		st.LicenseStatus = "legacy"
		st.Legacy = true
		st.Limits = row.Limits()
		st.LimitsKnown = row.HasLimits()
		st.Reason = "установка активирована до переноса лицензии на сервер — привяжите ключ в настройках"
		return st
	}

	if !st.Activated {
		st.Mode = ModeGrace
		st.LicenseStatus = "unknown"
		st.Reason = "лицензия не активирована"
		return st
	}

	st.LicenseStatus = row.LicenseStatus
	st.Limits = row.Limits()
	st.LimitsKnown = row.HasLimits()
	st.LicenseExpiresAt = row.LicenseExpiresAt
	st.TokenExpiresAt = row.TokenExpiresAt

	if claims != nil {
		st.TenantSlug = claims.TenantSlug
		st.KeyHint = claims.KeyHint
		if claims.Status != "" {
			st.LicenseStatus = claims.Status
		}
		if claims.Revision > 0 {
			st.Revision = claims.Revision
		}
		if limits := limitsFromClaims(claims); limits.MaxServers != 0 || limits.MaxNodes != 0 {
			st.Limits = limits
			st.LimitsKnown = true
		}
		if claims.ExpiresAt != nil {
			st.TokenExpiresAt = claims.ExpiresAt.Time
		}
		if exp := claims.LicenseExpiresAt(); exp != nil {
			st.LicenseExpiresAt = exp
		}
		if claims.GraceSeconds > 0 {
			grace = time.Duration(claims.GraceSeconds) * time.Second
		}
	}

	if !st.TokenExpiresAt.IsZero() {
		st.GraceUntil = st.TokenExpiresAt.Add(grace)
	}

	switch st.LicenseStatus {
	case "revoked":
		st.Mode = ModeReadOnly
		st.Reason = "лицензия отозвана"
		return st
	case "suspended":
		st.Mode = ModeReadOnly
		st.Reason = "лицензия приостановлена"
		return st
	case "past_due":
		st.Mode = ModeGrace
		st.Reason = "лицензия не оплачена — создание новых ресурсов приостановлено"
		return st
	case "expired":
		st.Mode = ModeGrace
		st.Reason = "срок лицензии истёк"
		if !st.GraceUntil.IsZero() && now.After(st.GraceUntil) {
			st.Mode = ModeReadOnly
			st.Reason = "срок лицензии истёк, льготный период закончился"
		}
		return st
	}

	if !st.TokenExpiresAt.IsZero() && now.After(st.TokenExpiresAt) {
		if now.After(st.GraceUntil) {
			st.Mode = ModeReadOnly
			st.Reason = fmt.Sprintf("лицензия не подтверждена с %s, льготный период закончился",
				st.LastVerifiedAt.Format("02.01.2006 15:04"))
			return st
		}
		st.Mode = ModeGrace
		st.Reason = "не удалось связаться с сервисом лицензий — создание новых ресурсов приостановлено"
		return st
	}

	st.Mode = ModeActive
	st.Reason = ""
	return st
}

func limitsFromClaims(c *licensejwt.Claims) Limits {
	return Limits{
		MaxServers: c.Limits.MaxServers,
		MaxNodes:   c.Limits.MaxNodes,
		MaxAdmins:  c.Limits.MaxAdmins,
		APIRPM:     c.Limits.APIRPM,
	}
}
