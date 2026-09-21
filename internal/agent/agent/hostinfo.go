package agent

import (
	"context"
	"os"
	"runtime"
	"time"

	"github.com/vortanixapp/panel/internal/agent/docker"
	"github.com/vortanixapp/panel/pkg/dockerapi"
	"github.com/vortanixapp/panel/pkg/protocol"
)

var agentCaps = []string{
	protocol.CapLanes,
	protocol.CapOutbox,
	protocol.CapStateSnapshot,
	protocol.CapTasks,
	protocol.CapNodeEvents,
	protocol.CapNodeInfo,
	protocol.CapAgentLogs,
	protocol.CapAgentRestart,
	protocol.CapFirewallHost,
	protocol.CapCronScheduler,
}

type dockerFacts struct {
	Version       string `json:"version,omitempty"`
	API           string `json:"api,omitempty"`
	Cgroup        string `json:"cgroup,omitempty"`
	CgroupDriver  string `json:"cgroup_driver,omitempty"`
	StorageDriver string `json:"storage_driver,omitempty"`
	RootDir       string `json:"root_dir,omitempty"`
	MemoryLimit   bool   `json:"memory_limit"`
	SwapLimit     bool   `json:"swap_limit"`
	Containers    int    `json:"containers"`
	Running       int    `json:"running"`
	Images        int    `json:"images"`
}

type hostFacts struct {
	Hostname    string       `json:"hostname,omitempty"`
	OS          string       `json:"os,omitempty"`
	Kernel      string       `json:"kernel,omitempty"`
	Arch        string       `json:"arch,omitempty"`
	CPUs        int          `json:"cpus,omitempty"`
	CPUModel    string       `json:"cpu_model,omitempty"`
	MemTotalMB  int64        `json:"mem_total_mb,omitempty"`
	UptimeSec   int64        `json:"uptime_sec,omitempty"`
	DataDir     string       `json:"data_dir,omitempty"`
	Quota       bool         `json:"quota"`
	Docker      *dockerFacts `json:"docker,omitempty"`
	DockerError string       `json:"docker_error,omitempty"`
}

type selfFacts struct {
	Version       string `json:"version"`
	Proto         int    `json:"proto"`
	BootID        string `json:"boot_id"`
	StartedAt     string `json:"started_at"`
	Image         string `json:"image,omitempty"`
	ContainerID   string `json:"container_id,omitempty"`
	ContainerName string `json:"container_name,omitempty"`
	RestartPolicy string `json:"restart_policy,omitempty"`
	NetworkMode   string `json:"network_mode,omitempty"`
}

func (a *Agent) collectFacts(parent context.Context) (hostFacts, selfFacts) {
	ctx, cancel := context.WithTimeout(parent, 3*time.Second)
	defer cancel()

	host := hostFacts{
		Arch:      runtime.GOARCH,
		CPUs:      runtime.NumCPU(),
		CPUModel:  docker.HostCPUModel(),
		UptimeSec: docker.HostUptimeSeconds(),
		DataDir:   docker.HostDataDir(),
		Quota:     docker.QuotaSupported(),
	}
	if name, err := os.Hostname(); err == nil {
		host.Hostname = name
	}
	cli := dockerapi.New()
	if info, err := cli.Info(ctx); err == nil {
		if info.Name != "" {
			host.Hostname = info.Name
		}
		host.OS = info.OperatingSystem
		host.Kernel = info.KernelVersion
		if info.Architecture != "" {
			host.Arch = info.Architecture
		}
		if info.NCPU > 0 {
			host.CPUs = info.NCPU
		}
		host.MemTotalMB = info.MemTotal >> 20
		host.Docker = &dockerFacts{
			Version:       info.ServerVersion,
			Cgroup:        info.CgroupVersion,
			CgroupDriver:  info.CgroupDriver,
			StorageDriver: info.Driver,
			RootDir:       info.DockerRootDir,
			MemoryLimit:   info.MemoryLimit,
			SwapLimit:     info.SwapLimit,
			Containers:    info.Containers,
			Running:       info.ContainersRunning,
			Images:        info.Images,
		}
		if v, err := cli.Version(ctx); err == nil {
			host.Docker.API = v.APIVersion
		}
	} else {
		host.DockerError = err.Error()
	}

	self := selfFacts{
		Version:   a.version,
		Proto:     protocol.ProtoVersion,
		BootID:    a.bootID,
		StartedAt: a.startedAt.UTC().Format(time.RFC3339Nano),
	}
	if id := dockerapi.SelfContainerID(); id != "" {
		if c, err := cli.Inspect(ctx, id); err == nil {
			self.ContainerID = c.ID
			self.ContainerName = c.CleanName()
			self.Image = c.ConfigImage()
			if rp, ok := c.HostConfig["RestartPolicy"].(map[string]any); ok {
				self.RestartPolicy, _ = rp["Name"].(string)
			}
			self.NetworkMode, _ = c.HostConfig["NetworkMode"].(string)
		}
	}
	return host, self
}
