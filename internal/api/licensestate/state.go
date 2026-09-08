package licensestate

import (
	"sync/atomic"
	"time"
)

const Unlimited = -1

type Mode string

const (
	ModeActive   Mode = "active"
	ModeGrace    Mode = "grace"
	ModeReadOnly Mode = "read_only"
)

type Limits struct {
	MaxServers int `json:"max_servers"`
	MaxNodes   int `json:"max_nodes"`
	MaxAdmins  int `json:"max_admins"`
	APIRPM     int `json:"api_rpm"`
}

func (l Limits) Allows(used, limit int) bool {
	if limit < 0 {
		return true
	}
	return used < limit
}

type State struct {
	Mode          Mode
	LicenseStatus string
	Plan          string
	KeyHint       string
	Limits        Limits
	Revision      int64

	InstallationID string
	TenantSlug     string
	Domain         string

	TokenExpiresAt   time.Time
	LicenseExpiresAt *time.Time
	GraceUntil       time.Time
	LastVerifiedAt   time.Time
	LastError        string

	Activated   bool
	LimitsKnown bool
	Legacy      bool

	Update UpdateInfo

	Reason string
}

type UpdateInfo struct {
	TargetVersion  string
	MandatoryAfter *time.Time
	Notes          string
	DeferUntil     *time.Time
	// Auto — владелец разрешил ставить обновления без своего участия.
	Auto bool
}

func (u UpdateInfo) Available(currentVersion string) bool {
	return u.TargetVersion != "" && u.TargetVersion != currentVersion
}

func (u UpdateInfo) Overdue(now time.Time) bool {
	return u.MandatoryAfter != nil && !now.Before(*u.MandatoryAfter)
}

func (s *State) BlocksCreation() bool {
	return s.Mode != ModeActive
}

func (s *State) BlocksWrites() bool {
	return s.Mode == ModeReadOnly
}

type Store struct {
	ptr atomic.Pointer[State]
}

func NewStore() *Store {
	s := &Store{}
	s.ptr.Store(&State{
		Mode:          ModeGrace,
		LicenseStatus: "unknown",
		Reason:        "состояние лицензии ещё не загружено",
	})
	return s
}

func (s *Store) Load() *State { return s.ptr.Load() }

func (s *Store) Set(state *State) { s.ptr.Store(state) }
