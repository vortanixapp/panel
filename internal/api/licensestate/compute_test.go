package licensestate

import (
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"

	"github.com/vortanix/vortanix/internal/api/licensejwt"
)

const testGrace = 24 * time.Hour

func now() time.Time { return time.Date(2026, 8, 23, 12, 0, 0, 0, time.UTC) }

func activeRow() Row {
	servers, nodes, admins, rpm := 200, 10, 5, 600
	return Row{
		ID:             "11111111-1111-4111-8111-111111111111",
		LicenseKey:     "VRTX-AAAAA-BBBBB-CCCCC-DDDDD",
		LicenseToken:   "token",
		LicenseStatus:  "active",
		Plan:           "pro",
		Revision:       7,
		TokenExpiresAt: now().Add(48 * time.Hour),
		MaxServers:     &servers,
		MaxNodes:       &nodes,
		MaxAdmins:      &admins,
		APIRPM:         &rpm,
	}
}

func claimsFor(status string, expiresAt time.Time) *licensejwt.Claims {
	return &licensejwt.Claims{
		TenantSlug: "acme",
		Status:     status,
		Revision:   7,
		KeyHint:    "DDDDD",
		Limits: licensejwt.PlanLimits{
			MaxServers: 200, MaxNodes: 10, MaxAdmins: 5, APIRPM: 600,
		},
		RegisteredClaims: jwtlib.RegisteredClaims{
			ExpiresAt: jwtlib.NewNumericDate(expiresAt),
		},
	}
}

func TestActiveLicenseAllowsEverything(t *testing.T) {
	st := Derive(activeRow(), claimsFor("active", now().Add(48*time.Hour)), now(), testGrace)

	if st.Mode != ModeActive {
		t.Fatalf("режим %s, ожидался active", st.Mode)
	}
	if st.BlocksCreation() || st.BlocksWrites() {
		t.Fatal("активная лицензия ничего не должна блокировать")
	}
	if st.Limits.MaxServers != 200 || st.Limits.APIRPM != 600 {
		t.Fatalf("лимиты из токена не подхватились: %+v", st.Limits)
	}
	if !st.LimitsKnown {
		t.Fatal("лимиты должны считаться известными")
	}
}

func TestOutageGivesOneDayOfGrace(t *testing.T) {
	row := activeRow()
	row.TokenExpiresAt = now().Add(-time.Hour)
	claims := claimsFor("active", row.TokenExpiresAt)

	st := Derive(row, claims, now(), testGrace)
	if st.Mode != ModeGrace {
		t.Fatalf("через час после истечения режим %s, ожидался grace", st.Mode)
	}
	if !st.BlocksCreation() {
		t.Fatal("в грейсе создание новых ресурсов должно быть заблокировано")
	}
	if st.BlocksWrites() {
		t.Fatal("в грейсе правки и удаления должны работать")
	}
	if !st.LimitsKnown {
		t.Fatal("авария сервиса лицензий не должна стирать лимиты")
	}

	late := Derive(row, claims, now().Add(25*time.Hour), testGrace)
	if late.Mode != ModeReadOnly {
		t.Fatalf("после суток грейса режим %s, ожидался read_only", late.Mode)
	}
}

func TestSuspendedIsReadOnlyImmediately(t *testing.T) {
	row := activeRow()
	row.LicenseStatus = "suspended"

	st := Derive(row, claimsFor("suspended", now().Add(48*time.Hour)), now(), testGrace)
	if st.Mode != ModeReadOnly {
		t.Fatalf("режим %s, ожидался read_only", st.Mode)
	}
	if !st.LimitsKnown {
		t.Fatal("лимиты должны сохраняться и при приостановке")
	}
}

func TestRevokedIsReadOnly(t *testing.T) {
	row := activeRow()
	row.LicenseStatus = "revoked"

	if st := Derive(row, claimsFor("revoked", now().Add(48*time.Hour)), now(), testGrace); st.Mode != ModeReadOnly {
		t.Fatalf("режим %s, ожидался read_only", st.Mode)
	}
}

func TestPastDueBlocksCreationOnly(t *testing.T) {
	row := activeRow()
	row.LicenseStatus = "past_due"

	st := Derive(row, claimsFor("past_due", now().Add(48*time.Hour)), now(), testGrace)
	if st.Mode != ModeGrace {
		t.Fatalf("режим %s, ожидался grace", st.Mode)
	}
	if st.BlocksWrites() {
		t.Fatal("при неоплате правки должны оставаться доступны")
	}
}

func TestExpiredLicenseFallsToReadOnlyAfterGrace(t *testing.T) {
	row := activeRow()
	row.LicenseStatus = "expired"
	claims := claimsFor("expired", now().Add(-2*time.Hour))

	if st := Derive(row, claims, now(), testGrace); st.Mode != ModeGrace {
		t.Fatalf("сразу после истечения режим %s, ожидался grace", st.Mode)
	}
	if st := Derive(row, claims, now().Add(30*time.Hour), testGrace); st.Mode != ModeReadOnly {
		t.Fatalf("после грейса режим %s, ожидался read_only", st.Mode)
	}
}

func TestLegacyInstallationNeverDegrades(t *testing.T) {
	servers, nodes := 500, 20
	row := Row{
		ID:             "22222222-2222-4222-8222-222222222222",
		LicenseStatus:  "legacy",
		Plan:           "pro",
		MaxServers:     &servers,
		MaxNodes:       &nodes,
		TokenExpiresAt: now().Add(-90 * 24 * time.Hour),
	}

	st := Derive(row, nil, now(), testGrace)
	if st.Mode != ModeActive {
		t.Fatalf("legacy-установка получила режим %s", st.Mode)
	}
	if !st.Legacy || st.Reason == "" {
		t.Fatal("legacy-установка должна помечаться и объяснять, что делать")
	}
	if st.BlocksCreation() {
		t.Fatal("legacy-установка не должна блокировать создание")
	}
}

func TestUnactivatedFailsClosedOnCreation(t *testing.T) {
	st := Derive(Row{ID: "x", LicenseStatus: "unknown"}, nil, now(), testGrace)

	if st.Mode != ModeGrace {
		t.Fatalf("режим %s, ожидался grace", st.Mode)
	}
	if st.LimitsKnown {
		t.Fatal("лимиты неактивированной панели не могут быть известны")
	}
	if !st.BlocksCreation() {
		t.Fatal("без лицензии создание ресурсов должно блокироваться")
	}
}

func TestUnlimitedLimits(t *testing.T) {
	l := Limits{MaxServers: Unlimited}
	if !l.Allows(100000, l.MaxServers) {
		t.Fatal("-1 должно означать отсутствие ограничения")
	}
	if l.Allows(5, 5) {
		t.Fatal("на границе лимита создание должно отклоняться")
	}
	if !l.Allows(4, 5) {
		t.Fatal("под лимитом создание должно разрешаться")
	}
}

func TestClaimsStatusOverridesStoredRow(t *testing.T) {
	row := activeRow()
	row.LicenseStatus = "active"

	st := Derive(row, claimsFor("suspended", now().Add(48*time.Hour)), now(), testGrace)
	if st.Mode != ModeReadOnly {
		t.Fatalf("режим %s, ожидался read_only по статусу из токена", st.Mode)
	}
}
