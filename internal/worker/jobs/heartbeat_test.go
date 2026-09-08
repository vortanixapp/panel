package jobs

import (
	"testing"
	"time"
)

func TestHeartbeatFreshLoopIsHealthy(t *testing.T) {
	now := time.Now()
	h := newHeartbeat()
	h.now = func() time.Time { return now }

	for name := range loopPeriods {
		h.Beat(name)
	}
	_, healthy := h.Status()
	if !healthy {
		t.Fatal("только что отметившиеся циклы должны считаться живыми")
	}
}

func TestHeartbeatDetectsStuckLoop(t *testing.T) {
	now := time.Now()
	h := newHeartbeat()
	h.now = func() time.Time { return now }
	for name := range loopPeriods {
		h.Beat(name)
	}

	now = now.Add(loopPeriods[LoopProvision]*staleFactor + time.Second)
	for name := range loopPeriods {
		if name != LoopProvision {
			h.Beat(name)
		}
	}

	statuses, healthy := h.Status()
	if healthy {
		t.Fatal("зависший цикл должен делать воркер нездоровым")
	}
	var found bool
	for _, st := range statuses {
		if st.Name == LoopProvision {
			found = true
			if !st.Stale {
				t.Error("provision должен быть помечен как зависший")
			}
		} else if st.Stale {
			t.Errorf("%s зависшим не является", st.Name)
		}
	}
	if !found {
		t.Error("provision отсутствует в отчёте")
	}
}

func TestHeartbeatToleratesOneSlowPass(t *testing.T) {
	now := time.Now()
	h := newHeartbeat()
	h.now = func() time.Time { return now }
	for name := range loopPeriods {
		h.Beat(name)
	}

	now = now.Add(loopPeriods[LoopProvision] * (staleFactor - 1))
	if _, healthy := h.Status(); !healthy {
		t.Fatal("пропуск меньше порога не должен считаться зависанием")
	}
}

func TestHeartbeatNotStartedIsHealthy(t *testing.T) {
	h := newHeartbeat()
	statuses, healthy := h.Status()
	if !healthy {
		t.Fatal("не начавшие работу циклы не должны считаться зависшими")
	}
	for _, st := range statuses {
		if st.Started {
			t.Errorf("%s не должен числиться запущенным", st.Name)
		}
	}
}

func TestEveryLoopHasPeriod(t *testing.T) {
	expected := []LoopName{
		LoopProvision, LoopBackup, LoopNodeSetup, LoopDaemon,
		LoopMailing, LoopFTPAccount, LoopExpiry, LoopBackupSchedule,
		LoopMigrate, LoopBackupOffsite, LoopNodeBulk, LoopWebhook,
		LoopNotifyDelivery,
	}
	if len(loopPeriods) != len(expected) {
		t.Fatalf("циклов в loopPeriods %d, перечислено %d — добавьте новый цикл в оба места",
			len(loopPeriods), len(expected))
	}
	for _, name := range expected {
		if _, ok := loopPeriods[name]; !ok {
			t.Errorf("цикл %s без периода — его зависание не заметят", name)
		}
	}
}
