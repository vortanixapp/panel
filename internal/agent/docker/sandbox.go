package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/vortanixapp/panel/pkg/settingsreg"
)

const (
	sandboxOff      = "off"
	sandboxSelected = "selected"
	sandboxAll      = "all"
	sandboxCacheTTL = 30 * time.Second
)

var sandboxRuntimeName = regexp.MustCompile(`^[a-z][a-z0-9._-]{0,40}$`)

var (
	sandboxMu        sync.Mutex
	sandboxRuntimes  map[string]bool
	sandboxCheckedAt time.Time
	sandboxInspect   = inspectRuntimes
)

func sandboxMode() string {
	switch raw := strings.ToLower(strings.TrimSpace(os.Getenv("VORTANIX_SANDBOX"))); raw {
	case sandboxOff, sandboxSelected, sandboxAll:
		return raw
	}
	switch mode := settingsreg.AgentSandboxMode.Str(); mode {
	case sandboxSelected, sandboxAll:
		return mode
	}
	return sandboxOff
}

func sandboxGameListed(gameID string) bool {
	want := normalizeGame(gameID)
	for _, item := range strings.FieldsFunc(settingsreg.AgentSandboxGames.Str(), func(r rune) bool {
		return r == ',' || r == ';' || r == ' ' || r == '\n'
	}) {
		if normalizeGame(item) == want {
			return true
		}
	}
	return false
}

func sandboxWanted(gameID string) bool {
	if gameID == "" || gameID == "test" {
		return false
	}
	switch sandboxMode() {
	case sandboxAll:
		return !sandboxGameListed(gameID)
	case sandboxSelected:
		return sandboxGameListed(gameID)
	}
	return false
}

func sandboxRuntime() string {
	name := strings.ToLower(strings.TrimSpace(settingsreg.AgentSandboxRuntime.Str()))
	if !sandboxRuntimeName.MatchString(name) {
		return "runsc"
	}
	return name
}

func inspectRuntimes(ctx context.Context) (map[string]bool, error) {
	out, err := exec.CommandContext(ctx, "docker", "info", "--format", "{{json .Runtimes}}").Output()
	if err != nil {
		return nil, err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, err
	}
	found := make(map[string]bool, len(raw))
	for name := range raw {
		found[name] = true
	}
	return found, nil
}

func installedRuntimes(ctx context.Context) map[string]bool {
	sandboxMu.Lock()
	defer sandboxMu.Unlock()
	if sandboxRuntimes != nil && time.Since(sandboxCheckedAt) < sandboxCacheTTL {
		return sandboxRuntimes
	}
	found, err := sandboxInspect(ctx)
	if err != nil {
		log.Printf("песочница: не удалось получить список окружений Docker: %v", err)
		return sandboxRuntimes
	}
	sandboxRuntimes = found
	sandboxCheckedAt = time.Now()
	return found
}

func resetSandboxCache() {
	sandboxMu.Lock()
	defer sandboxMu.Unlock()
	sandboxRuntimes = nil
	sandboxCheckedAt = time.Time{}
}

func resolveSandbox(ctx context.Context, gameID string) (string, error) {
	if !sandboxWanted(gameID) {
		return "", nil
	}
	name := sandboxRuntime()
	if installedRuntimes(ctx)[name] {
		return name, nil
	}
	if settingsreg.AgentSandboxStrict.Bool() {
		return "", fmt.Errorf("на узле не установлено окружение %s, а песочница обязательна: установите gVisor или отключите обязательность в настройках узлов", name)
	}
	log.Printf("песочница: окружение %s не установлено на узле, сервер %s запускается без неё", name, gameID)
	return "", nil
}

func withSandbox(args []string, runtime string) []string {
	if runtime == "" || len(args) == 0 {
		return args
	}
	out := make([]string, 0, len(args)+4)
	out = append(out, args[0], "--runtime", runtime, "--label", "vortanix.sandbox="+runtime)
	return append(out, args[1:]...)
}
