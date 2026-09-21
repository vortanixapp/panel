package docker

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/vortanixapp/panel/pkg/dockerapi"
)

const (
	CheckOK   = "ok"
	CheckWarn = "warn"
	CheckFail = "fail"
	CheckSkip = "skip"
)

type Check struct {
	ID     string         `json:"id"`
	Status string         `json:"status"`
	Data   map[string]any `json:"data,omitempty"`
	Error  string         `json:"error,omitempty"`
}

type DiagnosticsInput struct {
	ClockSkewMs   int64
	TargetImage   string
	Reconnects    int
	UptimeSec     int64
	RestartPolicy string
}

func writable(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".vtx-probe-")
	if err != nil {
		return err
	}
	name := f.Name()
	_, werr := f.WriteString("ok")
	cerr := f.Close()
	_ = os.Remove(name)
	if werr != nil {
		return werr
	}
	return cerr
}

func abs64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

func RunDiagnostics(ctx context.Context, in DiagnosticsInput, progress func(id string)) []Check {
	var checks []Check
	add := func(c Check) { checks = append(checks, c) }
	cli := dockerapi.New()

	progress("docker")
	dctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	info, infoErr := cli.Info(dctx)
	cancel()
	if infoErr != nil {
		add(Check{ID: "docker", Status: CheckFail, Error: infoErr.Error()})
	} else {
		add(Check{ID: "docker", Status: CheckOK, Data: map[string]any{
			"version": info.ServerVersion, "containers": info.Containers, "running": info.ContainersRunning,
		}})
	}

	progress("disk")
	root := serversRoot()
	total, _, pct := diskInfo(root)
	free, inodes := diskFreeInfo(root)
	st := CheckOK
	switch {
	case total == 0:
		st = CheckSkip
	case free < 2048 || pct > 95:
		st = CheckFail
	case free < 10240 || pct > 90:
		st = CheckWarn
	}
	add(Check{ID: "disk_space", Status: st, Data: map[string]any{"path": root, "free_mb": free, "total_mb": total, "used_percent": pct}})
	st = CheckOK
	switch {
	case total == 0:
		st = CheckSkip
	case inodes > 95:
		st = CheckFail
	case inodes > 85:
		st = CheckWarn
	}
	add(Check{ID: "inodes", Status: st, Data: map[string]any{"used_percent": inodes}})
	if QuotaSupported() {
		add(Check{ID: "quota", Status: CheckOK, Data: map[string]any{"mount": quotaMountpoint()}})
	} else {
		add(Check{ID: "quota", Status: CheckWarn})
	}

	progress("data_dir")
	if err := writable(root); err != nil {
		add(Check{ID: "data_dir_write", Status: CheckFail, Data: map[string]any{"path": root}, Error: err.Error()})
	} else {
		add(Check{ID: "data_dir_write", Status: CheckOK, Data: map[string]any{"path": root}})
	}
	state := envOr("VORTANIX_STATE_DIR", "/opt/vortanix/state")
	if err := writable(state); err != nil {
		add(Check{ID: "state_dir_write", Status: CheckFail, Data: map[string]any{"path": state}, Error: err.Error()})
	} else {
		add(Check{ID: "state_dir_write", Status: CheckOK, Data: map[string]any{"path": state}})
	}

	progress("firewall")
	if infoErr != nil {
		add(Check{ID: "firewall", Status: CheckSkip})
	} else {
		fctx, cancel := context.WithTimeout(ctx, time.Minute)
		out, err := HostShell(fctx, "iptables -V && iptables -w -S DOCKER-USER >/dev/null")
		cancel()
		if err != nil {
			add(Check{ID: "firewall", Status: CheckFail, Error: err.Error()})
		} else {
			add(Check{ID: "firewall", Status: CheckOK, Data: map[string]any{"iptables": firstLine(out)}})
		}
	}

	progress("clock")
	skew := abs64(in.ClockSkewMs)
	st = CheckOK
	switch {
	case skew > 60_000:
		st = CheckFail
	case skew > 5_000:
		st = CheckWarn
	}
	add(Check{ID: "clock", Status: st, Data: map[string]any{"skew_ms": in.ClockSkewMs}})

	progress("registry")
	if in.TargetImage == "" || infoErr != nil {
		add(Check{ID: "registry", Status: CheckSkip})
	} else {
		rctx, cancel := context.WithTimeout(ctx, 20*time.Second)
		dist, err := cli.DistributionInspect(rctx, in.TargetImage)
		cancel()
		if err != nil {
			add(Check{ID: "registry", Status: CheckFail, Data: map[string]any{"image": in.TargetImage}, Error: err.Error()})
		} else {
			add(Check{ID: "registry", Status: CheckOK, Data: map[string]any{"image": in.TargetImage, "digest": dist.Descriptor.Digest}})
		}
	}

	if info != nil {
		st = CheckOK
		if !info.MemoryLimit {
			st = CheckWarn
		}
		add(Check{ID: "cgroup", Status: st, Data: map[string]any{
			"version": info.CgroupVersion, "driver": info.CgroupDriver, "memory_limit": info.MemoryLimit, "swap_limit": info.SwapLimit,
		}})
		st = CheckOK
		switch strings.ToLower(info.Driver) {
		case "vfs", "devicemapper":
			st = CheckWarn
		}
		add(Check{ID: "storage_driver", Status: st, Data: map[string]any{"driver": info.Driver, "root": info.DockerRootDir}})
	}

	switch in.RestartPolicy {
	case "":
		add(Check{ID: "restart_policy", Status: CheckSkip})
	case "no":
		add(Check{ID: "restart_policy", Status: CheckWarn, Data: map[string]any{"policy": in.RestartPolicy}})
	default:
		add(Check{ID: "restart_policy", Status: CheckOK, Data: map[string]any{"policy": in.RestartPolicy}})
	}

	st = CheckOK
	hours := float64(in.UptimeSec) / 3600
	if hours < 1 {
		hours = 1
	}
	if float64(in.Reconnects)/hours > 6 {
		st = CheckWarn
	}
	add(Check{ID: "reconnects", Status: st, Data: map[string]any{"count": in.Reconnects, "uptime_sec": in.UptimeSec}})
	return checks
}
