package docker

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/vortanixapp/panel/pkg/dockerapi"
)

var (
	helperMu    sync.Mutex
	helperImage string
	helperAt    time.Time
)

func selfImage(ctx context.Context) string {
	helperMu.Lock()
	defer helperMu.Unlock()
	if helperImage != "" && time.Since(helperAt) < 10*time.Minute {
		return helperImage
	}
	id := dockerapi.SelfContainerID()
	if id == "" {
		return ""
	}
	c, err := dockerapi.New().Inspect(ctx, id)
	if err != nil || c.Image == "" {
		return ""
	}
	helperImage, helperAt = c.Image, time.Now()
	return helperImage
}

func HostShell(ctx context.Context, script string) (string, error) {
	image := selfImage(ctx)
	var cmd *exec.Cmd
	if image == "" {
		cmd = exec.CommandContext(ctx, "sh", "-c", script)
	} else {
		cmd = exec.CommandContext(ctx, "docker", "run", "--rm",
			"--privileged", "--pid", "host", "--network", "none",
			"--label", "vortanix.helper=host",
			"--entrypoint", "nsenter", image,
			"-t", "1", "-m", "-n", "--", "sh", "-c", script)
	}
	out, err := cmd.CombinedOutput()
	text := strings.TrimSpace(string(out))
	if err != nil {
		if text == "" {
			return "", err
		}
		return text, fmt.Errorf("%w: %s", err, lastLine(text))
	}
	return text, nil
}

func lastLine(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	return strings.TrimSpace(lines[len(lines)-1])
}
