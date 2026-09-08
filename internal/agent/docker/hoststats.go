package docker

import (
	"os"
	"strconv"
	"strings"
)

type HostStats struct {
	CPUPercent  float64 `json:"cpu_percent"`
	RAMPercent  float64 `json:"ram_percent"`
	RAMTotalMB  int64   `json:"ram_total_mb"`
	RAMUsedMB   int64   `json:"ram_used_mb"`
	DiskPercent float64 `json:"disk_percent"`
	DiskTotalMB int64   `json:"disk_total_mb"`
	DiskUsedMB  int64   `json:"disk_used_mb"`
	// DiskQuota — ограничивается ли место серверов на этой ноде. Админ, который
	// продаёт тариф с местом, обязан видеть, держит ли локация это обещание.
	DiskQuota bool `json:"disk_quota"`
}

var prevCPU struct {
	total, idle uint64
	valid       bool
}

func CollectHostStats() HostStats {
	var out HostStats
	out.CPUPercent = cpuPercent()
	out.RAMTotalMB, out.RAMUsedMB, out.RAMPercent = memInfo()
	out.DiskTotalMB, out.DiskUsedMB, out.DiskPercent = diskInfo(hostDiskPath())
	out.DiskQuota = QuotaSupported()
	return out
}

func hostDiskPath() string {
	if v := strings.TrimSpace(os.Getenv("VORTANIX_DATA_DIR")); v != "" {
		return v
	}
	return "/"
}

func cpuPercent() float64 {
	raw, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0
	}
	line := ""
	for _, l := range strings.Split(string(raw), "\n") {
		if strings.HasPrefix(l, "cpu ") {
			line = l
			break
		}
	}
	if line == "" {
		return 0
	}
	fields := strings.Fields(line)[1:]
	var total, idle uint64
	for i, f := range fields {
		v, err := strconv.ParseUint(f, 10, 64)
		if err != nil {
			continue
		}
		total += v
		if i == 3 || i == 4 {
			idle += v
		}
	}
	if total == 0 {
		return 0
	}

	prev := prevCPU
	prevCPU.total, prevCPU.idle, prevCPU.valid = total, idle, true
	if !prev.valid || total <= prev.total {
		return 0
	}
	dTotal := float64(total - prev.total)
	dIdle := float64(idle - prev.idle)
	if dTotal <= 0 {
		return 0
	}
	pct := (dTotal - dIdle) / dTotal * 100
	return clampPercent(pct)
}

func memInfo() (totalMB, usedMB int64, percent float64) {
	raw, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, 0, 0
	}
	vals := map[string]int64{}
	for _, line := range strings.Split(string(raw), "\n") {
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		key := strings.TrimSuffix(parts[0], ":")
		v, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			continue
		}
		vals[key] = v
	}
	total := vals["MemTotal"]
	if total == 0 {
		return 0, 0, 0
	}
	avail := vals["MemAvailable"]
	if avail == 0 {
		avail = vals["MemFree"] + vals["Buffers"] + vals["Cached"]
	}
	used := total - avail
	if used < 0 {
		used = 0
	}
	return total / 1024, used / 1024, clampPercent(float64(used) / float64(total) * 100)
}

func clampPercent(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return float64(int(v*100+0.5)) / 100
}
