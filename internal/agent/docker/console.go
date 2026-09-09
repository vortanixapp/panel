package docker

import (
	"bufio"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Stats struct {
	CPUPct      float64
	MemUsedMB   int
	MemLimitMB  int
	DiskUsedMB  int
	DiskTotalMB int
	StartedAt   string
	Uptime      string
}

var consoleSessions sync.Map

func StartConsoleStream(ctx context.Context, serverID, sessionID string, onLine func(string)) error {
	if _, loaded := consoleSessions.Load(sessionID); loaded {
		return nil
	}
	cctx, cancel := context.WithCancel(ctx)
	consoleSessions.Store(sessionID, cancel)

	go func() {
		defer func() {
			cancel()
			consoleSessions.Delete(sessionID)
		}()
		cname := ContainerName(serverID)
		cmd := exec.CommandContext(cctx, "docker", "logs", "-f", "--tail", "100", cname)
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			onLine("[console] failed to attach logs: " + err.Error() + "\r\n")
			return
		}
		stderr, _ := cmd.StderrPipe()
		_ = cmd.Start()
		read := func(r io.Reader) {
			sc := bufio.NewScanner(r)
			for sc.Scan() {
				onLine(sc.Text() + "\r\n")
			}
		}
		go read(stdout)
		go read(stderr)
		_ = cmd.Wait()
	}()
	return nil
}

func StopConsoleStream(sessionID string) {
	if v, ok := consoleSessions.LoadAndDelete(sessionID); ok {
		if cancel, ok := v.(context.CancelFunc); ok {
			cancel()
		}
	}
}

func ConsoleInput(ctx context.Context, serverID, data string) error {
	cname := ContainerName(serverID)
	cmd := exec.CommandContext(ctx, "docker", "exec", "-i", cname, "sh", "-c", "cat")
	cmd.Stdin = strings.NewReader(data)
	return cmd.Run()
}

func CollectStats(ctx context.Context, serverID string) (Stats, error) {
	cname := ContainerName(serverID)
	out, err := exec.CommandContext(ctx, "docker", "stats", cname, "--no-stream", "--format", "{{json .}}").Output()
	if err != nil {
		return Stats{}, err
	}
	line := strings.TrimSpace(string(out))
	if line == "" {
		return Stats{}, fmt.Errorf("no stats")
	}
	var raw struct {
		CPUPerc  string `json:"CPUPerc"`
		MemUsage string `json:"MemUsage"`
	}
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		return Stats{}, err
	}
	cpu, _ := strconv.ParseFloat(strings.TrimSuffix(raw.CPUPerc, "%"), 64)
	memUsed, memLimit := parseMemUsage(raw.MemUsage)
	st := Stats{CPUPct: cpu, MemUsedMB: memUsed, MemLimitMB: memLimit}
	if running, _ := isRunning(ctx, cname); running {
		st.StartedAt, st.Uptime = containerStartedAt(ctx, cname)
	}
	return st, nil
}

func CollectStatsExtended(ctx context.Context, serverID string, diskLimitMB int) (Stats, error) {
	st, err := CollectStats(ctx, serverID)
	if err != nil {
		st = Stats{}
	}
	if used := DiskQuotaUsedMB(ctx, serverID); used > 0 {
		st.DiskUsedMB = used
	} else {
		st.DiskUsedMB = serverDiskUsedMB(ctx, serverID)
	}
	if diskLimitMB > 0 {
		st.DiskTotalMB = diskLimitMB
	}
	return st, nil
}

func containerStartedAt(ctx context.Context, cname string) (startedAt, uptime string) {
	out, err := exec.CommandContext(ctx, "docker", "inspect", cname, "--format", "{{.State.StartedAt}}").Output()
	if err != nil {
		return "", ""
	}
	startedAt = strings.TrimSpace(string(out))
	if startedAt == "" || startedAt == "0001-01-01T00:00:00Z" {
		return "", ""
	}
	if t, err := time.Parse(time.RFC3339Nano, startedAt); err == nil {
		uptime = formatAgentUptime(time.Since(t))
		return startedAt, uptime
	}
	if t, err := time.Parse(time.RFC3339, startedAt); err == nil {
		uptime = formatAgentUptime(time.Since(t))
	}
	return startedAt, uptime
}

func serverDiskUsedMB(ctx context.Context, serverID string) int {
	dir := serverDataDir(serverID)
	if out, err := exec.CommandContext(ctx, "du", "-sm", dir).Output(); err == nil {
		fields := strings.Fields(string(out))
		if len(fields) > 0 {
			if mb, err := strconv.Atoi(fields[0]); err == nil {
				return mb
			}
		}
	}
	var total int64
	err := filepath.WalkDir(dir, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if info, err := d.Info(); err == nil {
			total += info.Size()
		}
		return nil
	})
	if err != nil {
		return 0
	}
	return int(total / (1024 * 1024))
}

func formatAgentUptime(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	sec := int(d.Seconds())
	days := sec / 86400
	sec %= 86400
	hours := sec / 3600
	sec %= 3600
	mins := sec / 60
	if days > 0 {
		return fmt.Sprintf("%dd %02dh %02dm", days, hours, mins)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %02dm", hours, mins)
	}
	return fmt.Sprintf("%dm", mins)
}

func parseMemUsage(s string) (used, limit int) {
	parts := strings.Split(s, " / ")
	if len(parts) != 2 {
		return 0, 0
	}
	used = parseMemMB(strings.TrimSpace(parts[0]))
	limit = parseMemMB(strings.TrimSpace(parts[1]))
	return
}

func parseMemMB(s string) int {
	s = strings.TrimSpace(s)
	if strings.HasSuffix(s, "GiB") {
		v, _ := strconv.ParseFloat(strings.TrimSuffix(s, "GiB"), 64)
		return int(v * 1024)
	}
	if strings.HasSuffix(s, "MiB") {
		v, _ := strconv.ParseFloat(strings.TrimSuffix(s, "MiB"), 64)
		return int(v)
	}
	if strings.HasSuffix(s, "KiB") {
		v, _ := strconv.ParseFloat(strings.TrimSuffix(s, "KiB"), 64)
		return int(v / 1024)
	}
	return 0
}

func ListManagedServerIDs(ctx context.Context) ([]string, error) {
	out, err := exec.CommandContext(ctx, "docker", "ps", "-a", "--filter", "label=vortanix.managed=true", "--format", "{{.Label \"vortanix.server_id\"}}").Output()
	if err != nil {
		return nil, err
	}
	seen := map[string]struct{}{}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	ids := make([]string, 0, len(lines)+4)
	for _, l := range lines {
		if l != "" {
			seen[l] = struct{}{}
			ids = append(ids, l)
		}
	}

	nameOut, nameErr := exec.CommandContext(ctx, "docker", "ps", "-a", "--filter", "name=^vortanix-", "--format", "{{.Names}}").Output()
	if nameErr == nil {
		for _, line := range strings.Split(strings.TrimSpace(string(nameOut)), "\n") {
			name := strings.TrimSpace(line)
			if !strings.HasPrefix(name, "vortanix-") {
				continue
			}
			raw := strings.TrimPrefix(name, "vortanix-")
			if len(raw) != 32 {
				continue
			}
			if _, decErr := hex.DecodeString(raw); decErr != nil {
				continue
			}
			id := raw[0:8] + "-" + raw[8:12] + "-" + raw[12:16] + "-" + raw[16:20] + "-" + raw[20:32]
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			ids = append(ids, id)
		}
	}
	return ids, nil
}
