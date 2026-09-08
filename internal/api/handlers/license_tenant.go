package handlers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/vortanix/vortanix/internal/api/licensejwt"
	"github.com/vortanix/vortanix/internal/api/licensestate"
)

// Состояние лицензии в памяти держит только своя установка: обновлять его
// фоном для каждой панели клиента нечем и незачем. Для арендатора состояние
// считается из его базы и живёт недолго — иначе ratelimit ходил бы в базу на
// каждом запросе.
const tenantLicenseTTL = 30 * time.Second

type tenantLicense struct {
	state *licensestate.State
	at    time.Time
}

var tenantLicenses sync.Map

func (h *Handler) licenseFor(ctx context.Context) *licensestate.State {
	pool := h.dbOf(ctx)
	if pool == nil || pool == h.db || h.license == nil {
		return h.licenseNow()
	}

	key := fmt.Sprintf("%p", pool)
	if v, ok := tenantLicenses.Load(key); ok {
		if entry, ok := v.(tenantLicense); ok && time.Since(entry.at) < tenantLicenseTTL {
			return entry.state
		}
	}

	row, err := licensestate.LoadRow(ctx, pool)
	if err != nil {
		return h.licenseNow()
	}
	grace := time.Duration(h.licenseGraceHours) * time.Hour
	var claims *licensejwt.Claims
	if row.LicenseToken != "" {
		claims, _ = h.license.ParseAllowExpired(row.LicenseToken, grace)
	}
	state := licensestate.Derive(row, claims, time.Now(), grace)
	tenantLicenses.Store(key, tenantLicense{state: state, at: time.Now()})
	return state
}

// forgetTenantLicense сбрасывает кэш состояния для базы арендатора. Нужен
// сразу после принудительной проверки: иначе ответ приходит из кэша и кнопка
// показывает те же данные, что и до нажатия.
func (h *Handler) forgetTenantLicense(ctx context.Context) {
	pool := h.dbOf(ctx)
	if pool == nil || pool == h.db {
		return
	}
	tenantLicenses.Delete(fmt.Sprintf("%p", pool))
}
