package docker

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

const projectIDBase = 10000

var (
	quotaOnce    sync.Once
	quotaMount   string
	quotaWarned  sync.Once
	projectIDMux sync.Mutex
)

func quotaMountpoint() string {
	quotaOnce.Do(func() {
		raw, err := os.ReadFile("/proc/mounts")
		if err != nil {
			return
		}
		quotaMount = quotaMountFrom(string(raw), serversRoot())
	})
	return quotaMount
}

func quotaMountFrom(mounts, dir string) string {
	target, opts := "", ""
	for _, line := range strings.Split(mounts, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		point := strings.ReplaceAll(fields[1], `\040`, " ")
		if !underMount(dir, point) {
			continue
		}
		if len(point) < len(target) {
			continue
		}
		target, opts = point, fields[3]
	}

	if target == "" {
		return ""
	}
	for _, opt := range strings.Split(opts, ",") {
		if opt == "prjquota" {
			return target
		}
	}
	return ""
}

func underMount(dir, target string) bool {
	if target == "/" {
		return true
	}
	if dir == target {
		return true
	}
	return strings.HasPrefix(dir, strings.TrimSuffix(target, "/")+"/")
}

func serversRoot() string {
	base := os.Getenv("VORTANIX_DATA_DIR")
	if base == "" {
		base = "/var/lib/vortanix/servers"
	}
	return base
}

func QuotaSupported() bool { return quotaMountpoint() != "" }

func ApplyDiskQuota(ctx context.Context, serverID string, diskMB int) {
	if diskMB <= 0 {
		return
	}
	mount := quotaMountpoint()
	if mount == "" {
		quotaWarned.Do(func() {
			log.Printf("дисковые квоты недоступны: %s смонтирован без prjquota — место сервера не ограничивается",
				serversRoot())
		})
		return
	}

	dir := serverDataDir(serverID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Printf("квота %s: каталог недоступен: %v", serverID, err)
		return
	}

	projID, err := ensureProjectID(ctx, dir)
	if err != nil {
		log.Printf("квота %s: не удалось назначить проект: %v", serverID, err)
		return
	}

	blocks := strconv.Itoa(diskMB * 1024)
	cmd := exec.CommandContext(ctx, "setquota", "-P", strconv.FormatUint(uint64(projID), 10),
		blocks, blocks, "0", "0", mount)
	if out, err := cmd.CombinedOutput(); err != nil {
		log.Printf("квота %s: setquota не отработал: %v: %s", serverID, err, strings.TrimSpace(string(out)))
		return
	}
}

func ensureProjectID(ctx context.Context, dir string) (uint32, error) {
	projectIDMux.Lock()
	defer projectIDMux.Unlock()

	if id := readProjectID(ctx, dir); id != 0 {
		return id, nil
	}

	next := uint32(projectIDBase)
	entries, err := os.ReadDir(serversRoot())
	if err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			if id := readProjectID(ctx, filepath.Join(serversRoot(), e.Name())); id >= next {
				next = id + 1
			}
		}
	}

	id := strconv.FormatUint(uint64(next), 10)

	if out, err := exec.CommandContext(ctx, "chattr", "-p", id, "+P", dir).CombinedOutput(); err != nil {
		return 0, fmt.Errorf("chattr +P: %w: %s", err, strings.TrimSpace(string(out)))
	}

	if out, err := exec.CommandContext(ctx, "chattr", "-R", "-p", id, dir).CombinedOutput(); err != nil {
		log.Printf("квота: не весь %s помечен проектом %s: %v: %s",
			dir, id, err, strings.TrimSpace(string(out)))
	}
	return next, nil
}

func readProjectID(ctx context.Context, dir string) uint32 {
	out, err := exec.CommandContext(ctx, "lsattr", "-pd", dir).Output()
	if err != nil {
		return 0
	}
	fields := strings.Fields(strings.TrimSpace(string(out)))
	if len(fields) == 0 {
		return 0
	}
	id, err := strconv.ParseUint(fields[0], 10, 32)
	if err != nil {
		return 0
	}
	return uint32(id)
}

func DiskQuotaUsedMB(ctx context.Context, serverID string) int {
	mount := quotaMountpoint()
	if mount == "" {
		return 0
	}
	dir := serverDataDir(serverID)
	projID := readProjectID(ctx, dir)
	if projID == 0 {
		return 0
	}
	out, err := exec.CommandContext(ctx, "repquota", "-P", "-O", "csv", mount).Output()
	if err != nil {
		return 0
	}
	return parseRepquotaUsedMB(string(out), projID)
}

func parseRepquotaUsedMB(out string, projID uint32) int {
	const (
		colProject   = 0
		colBlockUsed = 3
	)
	want := strconv.FormatUint(uint64(projID), 10)
	for _, line := range strings.Split(out, "\n") {
		cols := strings.Split(strings.TrimSpace(line), ",")
		if len(cols) <= colBlockUsed {
			continue
		}
		if strings.TrimSpace(cols[colProject]) != want {
			continue
		}
		kb, err := strconv.Atoi(strings.TrimSpace(cols[colBlockUsed]))
		if err != nil || kb < 0 {
			return 0
		}
		return kb / 1024
	}
	return 0
}
