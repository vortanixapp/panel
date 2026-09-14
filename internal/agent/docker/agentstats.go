package docker

import (
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type AgentStats struct {
	CPUPercent float64 `json:"cpu_percent"`
	RAMPercent float64 `json:"ram_percent"`
	RAMUsedMB  int64   `json:"ram_used_mb"`
}

var prevAgentCPU struct {
	seconds float64
	at      time.Time
	valid   bool
}

func CollectAgentStats() (AgentStats, bool) {
	seconds, ok := agentCPUSeconds()
	if !ok {
		return AgentStats{}, false
	}
	now := time.Now()
	prev := prevAgentCPU
	prevAgentCPU.seconds, prevAgentCPU.at, prevAgentCPU.valid = seconds, now, true
	if !prev.valid || seconds < prev.seconds {
		return AgentStats{}, false
	}
	elapsed := now.Sub(prev.at).Seconds()
	if elapsed <= 0 {
		return AgentStats{}, false
	}
	out := AgentStats{
		CPUPercent: clampPercent((seconds - prev.seconds) / elapsed / float64(runtime.NumCPU()) * 100),
	}
	if used, ok := agentMemoryBytes(); ok {
		out.RAMUsedMB = used >> 20
		if totalMB, _, _ := memInfo(); totalMB > 0 {
			out.RAMPercent = clampPercent(float64(used) / float64(totalMB<<20) * 100)
		}
	}
	return out, true
}

func readIntFile(path string) (int64, bool) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0, false
	}
	v, err := strconv.ParseInt(strings.TrimSpace(string(raw)), 10, 64)
	return v, err == nil
}

func statFileValue(path, key string) (int64, bool) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0, false
	}
	for _, line := range strings.Split(string(raw), "\n") {
		f := strings.Fields(line)
		if len(f) >= 2 && f[0] == key {
			v, err := strconv.ParseInt(f[1], 10, 64)
			return v, err == nil
		}
	}
	return 0, false
}

func agentCPUSeconds() (float64, bool) {
	if v, ok := statFileValue("/sys/fs/cgroup/cpu.stat", "usage_usec"); ok {
		return float64(v) / 1e6, true
	}
	for _, p := range []string{"/sys/fs/cgroup/cpuacct/cpuacct.usage", "/sys/fs/cgroup/cpu,cpuacct/cpuacct.usage"} {
		if v, ok := readIntFile(p); ok {
			return float64(v) / 1e9, true
		}
	}
	raw, err := os.ReadFile("/proc/self/stat")
	if err != nil {
		return 0, false
	}
	s := string(raw)
	i := strings.LastIndex(s, ")")
	if i < 0 {
		return 0, false
	}
	f := strings.Fields(s[i+1:])
	if len(f) < 15 {
		return 0, false
	}
	var ticks int64
	for _, idx := range []int{11, 12, 13, 14} {
		v, err := strconv.ParseInt(f[idx], 10, 64)
		if err != nil {
			return 0, false
		}
		ticks += v
	}
	return float64(ticks) / 100, true
}

func agentMemoryBytes() (int64, bool) {
	if cur, ok := readIntFile("/sys/fs/cgroup/memory.current"); ok {
		inactive, _ := statFileValue("/sys/fs/cgroup/memory.stat", "inactive_file")
		return max(cur-inactive, 0), true
	}
	if cur, ok := readIntFile("/sys/fs/cgroup/memory/memory.usage_in_bytes"); ok {
		inactive, _ := statFileValue("/sys/fs/cgroup/memory/memory.stat", "total_inactive_file")
		return max(cur-inactive, 0), true
	}
	if kb, ok := statFileValue("/proc/self/status", "VmRSS:"); ok {
		return kb << 10, true
	}
	return 0, false
}
