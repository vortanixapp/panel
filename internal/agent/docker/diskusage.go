package docker

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/vortanixapp/panel/pkg/dockerapi"
)

type ServerDisk struct {
	ServerID string `json:"server_id"`
	UsedMB   int    `json:"used_mb"`
	Source   string `json:"source"`
	Known    bool   `json:"known"`
}

type DockerDisk struct {
	ImagesBytes           int64 `json:"images_bytes"`
	ImagesReclaimable     int64 `json:"images_reclaimable_bytes"`
	ImagesCount           int   `json:"images_count"`
	ContainersBytes       int64 `json:"containers_bytes"`
	ContainersReclaimable int64 `json:"containers_reclaimable_bytes"`
	VolumesBytes          int64 `json:"volumes_bytes"`
	VolumesReclaimable    int64 `json:"volumes_reclaimable_bytes"`
	BuildCacheBytes       int64 `json:"build_cache_bytes"`
	BuildCacheReclaimable int64 `json:"build_cache_reclaimable_bytes"`
	LayersBytes           int64 `json:"layers_bytes"`
}

type DiskReport struct {
	DataDir          string       `json:"data_dir"`
	TotalMB          int64        `json:"total_mb"`
	UsedMB           int64        `json:"used_mb"`
	FreeMB           int64        `json:"free_mb"`
	UsedPercent      float64      `json:"used_percent"`
	InodesPercent    float64      `json:"inodes_percent"`
	Quota            bool         `json:"quota"`
	Servers          []ServerDisk `json:"servers"`
	Docker           *DockerDisk  `json:"docker,omitempty"`
	DockerError      string       `json:"docker_error,omitempty"`
	PluginCacheBytes int64        `json:"plugin_cache_bytes"`
	DurationMs       int64        `json:"duration_ms"`
}

func dirBytes(dir string) int64 {
	var total int64
	_ = filepath.WalkDir(dir, func(_ string, d os.DirEntry, err error) error {
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

func duMB(ctx context.Context, dir string) int {
	out, err := exec.CommandContext(ctx, "du", "-sm", dir).Output()
	if err != nil {
		return int(dirBytes(dir) >> 20)
	}
	fields := strings.Fields(string(out))
	if len(fields) == 0 {
		return 0
	}
	mb, _ := strconv.Atoi(fields[0])
	return mb
}

func repquotaAll(ctx context.Context) map[uint32]int {
	mount := quotaMountpoint()
	if mount == "" {
		return nil
	}
	out, err := exec.CommandContext(ctx, "repquota", "-P", "-O", "csv", mount).Output()
	if err != nil {
		return nil
	}
	used := map[uint32]int{}
	for _, line := range strings.Split(string(out), "\n") {
		cols := strings.Split(strings.TrimSpace(line), ",")
		if len(cols) < 4 {
			continue
		}
		id, err1 := strconv.ParseUint(strings.TrimPrefix(strings.TrimSpace(cols[0]), "#"), 10, 32)
		kb, err2 := strconv.Atoi(strings.TrimSpace(cols[3]))
		if err1 == nil && err2 == nil && kb >= 0 {
			used[uint32(id)] = kb / 1024
		}
	}
	return used
}

func dockerDisk(ctx context.Context) (*DockerDisk, error) {
	df, err := dockerapi.New().SystemDF(ctx)
	if err != nil {
		return nil, err
	}
	d := &DockerDisk{LayersBytes: df.LayersSize, ImagesCount: len(df.Images)}
	for _, img := range df.Images {
		d.ImagesBytes += img.Size - img.SharedSize
		if img.Containers == 0 {
			d.ImagesReclaimable += img.Size - img.SharedSize
		}
	}
	if d.ImagesBytes < df.LayersSize {
		d.ImagesBytes = df.LayersSize
	}
	for _, c := range df.Containers {
		d.ContainersBytes += c.SizeRw
		if c.State != "running" {
			d.ContainersReclaimable += c.SizeRw
		}
	}
	for _, v := range df.Volumes {
		if v.UsageData.Size > 0 {
			d.VolumesBytes += v.UsageData.Size
			if v.UsageData.RefCount == 0 {
				d.VolumesReclaimable += v.UsageData.Size
			}
		}
	}
	for _, b := range df.BuildCache {
		d.BuildCacheBytes += b.Size
		if !b.InUse {
			d.BuildCacheReclaimable += b.Size
		}
	}
	return d, nil
}

func NodeDiskUsage(ctx context.Context, known map[string]bool, progress func(stage string, done, total int)) DiskReport {
	started := time.Now()
	root := serversRoot()
	rep := DiskReport{DataDir: root, Quota: QuotaSupported(), Servers: []ServerDisk{}}
	rep.TotalMB, rep.UsedMB, rep.UsedPercent = diskInfo(root)
	rep.FreeMB, rep.InodesPercent = diskFreeInfo(root)

	progress("docker", 0, 0)
	dctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	if d, err := dockerDisk(dctx); err != nil {
		rep.DockerError = err.Error()
	} else {
		rep.Docker = d
	}
	cancel()

	entries, _ := os.ReadDir(root)
	var dirs []string
	for _, e := range entries {
		if e.IsDir() && safeServerID(e.Name()) {
			dirs = append(dirs, e.Name())
		}
	}
	quotas := map[uint32]int(nil)
	if rep.Quota {
		quotas = repquotaAll(ctx)
	}
	for i, name := range dirs {
		if ctx.Err() != nil {
			break
		}
		progress("servers", i, len(dirs))
		sd := ServerDisk{ServerID: name, Known: known == nil || known[name]}
		dir := filepath.Join(root, name)
		if quotas != nil {
			if id := readProjectID(ctx, dir); id != 0 {
				if mb, ok := quotas[id]; ok {
					sd.UsedMB, sd.Source = mb, "quota"
				}
			}
		}
		if sd.Source == "" {
			sd.UsedMB, sd.Source = duMB(ctx, dir), "du"
		}
		rep.Servers = append(rep.Servers, sd)
	}
	progress("servers", len(dirs), len(dirs))
	sort.Slice(rep.Servers, func(a, b int) bool { return rep.Servers[a].UsedMB > rep.Servers[b].UsedMB })
	rep.PluginCacheBytes = dirBytes(archiveCacheDir())
	rep.DurationMs = time.Since(started).Milliseconds()
	return rep
}
