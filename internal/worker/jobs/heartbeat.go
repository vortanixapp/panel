package jobs

import (
	"sync"
	"time"
)

type LoopName string

const (
	LoopProvision      LoopName = "provision"
	LoopBackup         LoopName = "backup"
	LoopNodeSetup      LoopName = "node_setup"
	LoopDaemon         LoopName = "daemon"
	LoopMailing        LoopName = "mailing"
	LoopFTPAccount     LoopName = "ftp_account"
	LoopExpiry         LoopName = "expiry"
	LoopBackupSchedule LoopName = "backup_schedule"
	LoopMigrate        LoopName = "migrate"
	LoopBackupOffsite  LoopName = "backup_offsite"
	LoopNodeBulk       LoopName = "node_bulk"
	LoopWebhook        LoopName = "webhook"
	LoopNotifyDelivery LoopName = "notify_delivery"
)

var loopPeriods = map[LoopName]time.Duration{
	LoopProvision:      30 * time.Second,
	LoopBackup:         30 * time.Second,
	LoopMailing:        30 * time.Second,
	LoopFTPAccount:     30 * time.Second,
	LoopExpiry:         time.Minute,
	LoopBackupSchedule: backupScheduleInterval,
	LoopNodeSetup:      12 * time.Minute,
	LoopDaemon:         12 * time.Minute,
	LoopMigrate:        2 * time.Minute,
	LoopBackupOffsite:  2 * time.Minute,
	LoopNodeBulk:       2 * time.Minute,
	LoopWebhook:        time.Minute,
	LoopNotifyDelivery: time.Minute,
}

const staleFactor = 3

type Heartbeat struct {
	mu    sync.RWMutex
	beats map[LoopName]time.Time
	now   func() time.Time
}

func newHeartbeat() *Heartbeat {
	return &Heartbeat{beats: map[LoopName]time.Time{}, now: time.Now}
}

func (h *Heartbeat) Beat(name LoopName) {
	if h == nil {
		return
	}
	h.mu.Lock()
	h.beats[name] = h.now()
	h.mu.Unlock()
}

type LoopStatus struct {
	Name    LoopName `json:"name"`
	AgeSec  float64  `json:"age_seconds"`
	Stale   bool     `json:"stale"`
	Started bool     `json:"started"`
}

func (h *Heartbeat) Status() ([]LoopStatus, bool) {
	if h == nil {
		return nil, true
	}
	h.mu.RLock()
	defer h.mu.RUnlock()

	now := h.now()
	out := make([]LoopStatus, 0, len(loopPeriods))
	healthy := true
	for name, period := range loopPeriods {
		last, started := h.beats[name]
		st := LoopStatus{Name: name, Started: started}
		if started {
			age := now.Sub(last)
			st.AgeSec = age.Seconds()
			st.Stale = age > period*staleFactor
			if st.Stale {
				healthy = false
			}
		}
		out = append(out, st)
	}
	return out, healthy
}

func (r *Runner) Heartbeat() *Heartbeat { return r.heartbeat }
