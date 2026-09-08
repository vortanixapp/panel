package docker

import "testing"

func TestCPUPercentNeedsTwoSamples(t *testing.T) {
	prevCPU.valid = false
	if got := cpuPercent(); got != 0 {
		t.Errorf("первый снимок сравнивать не с чем, ожидался 0, получено %v", got)
	}
}

func TestClampPercent(t *testing.T) {
	cases := map[float64]float64{
		-5:      0,
		0:       0,
		42.4567: 42.46,
		100:     100,
		140:     100,
	}
	for in, want := range cases {
		if got := clampPercent(in); got != want {
			t.Errorf("clampPercent(%v) = %v, ожидалось %v", in, got, want)
		}
	}
}

func TestCollectHostStatsStaysInRange(t *testing.T) {
	s := CollectHostStats()
	for name, v := range map[string]float64{
		"cpu": s.CPUPercent, "ram": s.RAMPercent, "disk": s.DiskPercent,
	} {
		if v < 0 || v > 100 {
			t.Errorf("%s = %v вне диапазона 0..100", name, v)
		}
	}
	if s.RAMUsedMB > s.RAMTotalMB && s.RAMTotalMB > 0 {
		t.Errorf("занято памяти %d МБ при всего %d МБ", s.RAMUsedMB, s.RAMTotalMB)
	}
}

func TestHostDiskPathFollowsDataDir(t *testing.T) {
	t.Setenv("VORTANIX_DATA_DIR", "/var/lib/vortanix/servers")
	if got := hostDiskPath(); got != "/var/lib/vortanix/servers" {
		t.Errorf("hostDiskPath() = %q", got)
	}
	t.Setenv("VORTANIX_DATA_DIR", "")
	if got := hostDiskPath(); got != "/" {
		t.Errorf("без переменной ожидался корень, получено %q", got)
	}
}
