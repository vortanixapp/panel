package docker

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/vortanixapp/panel/pkg/settingsreg"
)

const (
	cacheSteamDir   = "steam"
	cacheArchiveDir = "archives"
	cacheMetaSuffix = ".meta"
	cacheUsedSuffix = ".used"
	cacheBinSuffix  = ".bin"
)

var cacheLocks sync.Map

func lockCache(path string) *sync.Mutex {
	v, _ := cacheLocks.LoadOrStore(path, &sync.Mutex{})
	return v.(*sync.Mutex)
}

func installCacheEnabled() bool {
	return settingsreg.AgentInstallCache.Bool()
}

func installCacheRoot() string {
	if dir := strings.TrimSpace(os.Getenv("VORTANIX_CACHE_DIR")); dir != "" {
		return dir
	}
	return filepath.Join(filepath.Dir(serverDataDir("x")), "..", "install-cache")
}

func touchFile(path string) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	_ = os.WriteFile(path, []byte(time.Now().UTC().Format(time.RFC3339)), 0o644)
}

func fileAge(path string) (time.Duration, bool) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, false
	}
	return time.Since(info.ModTime()), true
}

func cacheKey(parts ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(sum[:])[:24]
}

func steamCacheKey(spec InstallSpec) string {
	return fmt.Sprintf("%d-%s", spec.SteamAppID, cacheKey(spec.SteamBranch, spec.SteamModConfig))
}

func cloneTree(ctx context.Context, src, dst string) error {
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	out, err := exec.CommandContext(ctx, "cp", "-a", "--reflink=auto", src+"/.", dst+"/").CombinedOutput()
	if err != nil {
		return fmt.Errorf("копирование из кэша: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func installSteamFromCache(ctx context.Context, dataDir string, spec InstallSpec, report ProgressFunc) bool {
	dir := filepath.Join(installCacheRoot(), cacheSteamDir, steamCacheKey(spec))
	mu := lockCache(dir)
	mu.Lock()
	defer mu.Unlock()

	age, filled := fileAge(dir + cacheMetaSuffix)
	switch {
	case !filled:
		_ = os.RemoveAll(dir)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return false
		}
		report(StageSteamCMD, 0, "Загрузка файлов игры в кэш узла", "")
		if err := runSteamcmd(ctx, dir, spec, true, report); err != nil {
			_ = os.RemoveAll(dir)
			return false
		}
		touchFile(dir + cacheMetaSuffix)
	case age > settingsreg.AgentInstallCacheTTL.Duration():
		report(StageSteamCMD, 0, "Обновление кэша игры на узле", "")
		if err := runSteamcmd(ctx, dir, spec, false, report); err != nil {
			return false
		}
		touchFile(dir + cacheMetaSuffix)
	}
	touchFile(dir + cacheUsedSuffix)

	report(StageSteamCMD, 95, "Копирование файлов из кэша узла", "")
	if err := cloneTree(ctx, dir, dataDir); err != nil {
		report(StageSteamCMD, -1, "", err.Error())
		return false
	}
	go pruneInstallCache(dir)
	return true
}

type cachedArchive struct {
	path string
	meta downloadMeta
	mu   *sync.Mutex
}

func archiveCachePath(url string) string {
	return filepath.Join(installCacheRoot(), cacheArchiveDir, cacheKey(url)+cacheBinSuffix)
}

func lockedCachedArchive(url string) (*cachedArchive, bool) {
	bin := archiveCachePath(url)
	mu := lockCache(bin)
	mu.Lock()
	age, ok := fileAge(bin + cacheMetaSuffix)
	if !ok || age > settingsreg.AgentInstallCacheTTL.Duration() {
		mu.Unlock()
		return nil, false
	}
	raw, err := os.ReadFile(bin + cacheMetaSuffix)
	if err != nil {
		mu.Unlock()
		return nil, false
	}
	var meta downloadMeta
	if json.Unmarshal(raw, &meta) != nil {
		mu.Unlock()
		return nil, false
	}
	if _, err := os.Stat(bin); err != nil {
		mu.Unlock()
		return nil, false
	}
	touchFile(bin + cacheUsedSuffix)
	return &cachedArchive{path: bin, meta: meta, mu: mu}, true
}

func lockedStoreArchive(url, downloaded string, meta downloadMeta) (*cachedArchive, bool) {
	bin := archiveCachePath(url)
	mu := lockCache(bin)
	mu.Lock()
	_ = os.Remove(bin + cacheMetaSuffix)
	if err := os.MkdirAll(filepath.Dir(bin), 0o755); err != nil {
		mu.Unlock()
		return nil, false
	}
	if err := moveFile(downloaded, bin); err != nil {
		_ = os.Remove(bin)
		mu.Unlock()
		return nil, false
	}
	raw, err := json.Marshal(meta)
	if err != nil {
		mu.Unlock()
		return nil, false
	}
	if err := os.WriteFile(bin+cacheMetaSuffix, raw, 0o644); err != nil {
		mu.Unlock()
		return nil, false
	}
	touchFile(bin + cacheUsedSuffix)
	return &cachedArchive{path: bin, meta: meta, mu: mu}, true
}

type cacheEntry struct {
	path string
	used time.Time
	size int64
}

func entrySize(path string) int64 {
	var total int64
	_ = filepath.WalkDir(path, func(_ string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if info, err := d.Info(); err == nil {
			total += info.Size()
		}
		return nil
	})
	return total
}

func listCacheEntries(root string) []cacheEntry {
	var paths []string
	if dirs, err := os.ReadDir(filepath.Join(root, cacheSteamDir)); err == nil {
		for _, d := range dirs {
			if d.IsDir() {
				paths = append(paths, filepath.Join(root, cacheSteamDir, d.Name()))
			}
		}
	}
	if files, err := os.ReadDir(filepath.Join(root, cacheArchiveDir)); err == nil {
		for _, f := range files {
			if strings.HasSuffix(f.Name(), cacheBinSuffix) {
				paths = append(paths, filepath.Join(root, cacheArchiveDir, f.Name()))
			}
		}
	}
	entries := make([]cacheEntry, 0, len(paths))
	for _, p := range paths {
		used := time.Time{}
		if info, err := os.Stat(p + cacheUsedSuffix); err == nil {
			used = info.ModTime()
		} else if info, err := os.Stat(p); err == nil {
			used = info.ModTime()
		}
		entries = append(entries, cacheEntry{path: p, used: used, size: entrySize(p)})
	}
	return entries
}

func pruneInstallCache(keep string) {
	pruneInstallCacheTo(installCacheRoot(), settingsreg.AgentInstallCacheMaxMB.Int()<<20, keep)
}

func pruneInstallCacheTo(root string, limit int64, keep string) {
	entries := listCacheEntries(root)
	var total int64
	for _, e := range entries {
		total += e.size
	}
	if total <= limit {
		return
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].used.Before(entries[j].used) })
	for _, e := range entries {
		if total <= limit {
			return
		}
		if e.path == keep {
			continue
		}
		mu := lockCache(e.path)
		if !mu.TryLock() {
			continue
		}
		_ = os.RemoveAll(e.path)
		_ = os.Remove(e.path + cacheMetaSuffix)
		_ = os.Remove(e.path + cacheUsedSuffix)
		mu.Unlock()
		total -= e.size
	}
}
