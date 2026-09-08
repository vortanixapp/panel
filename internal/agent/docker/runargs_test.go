package docker

import (
	"strings"
	"testing"
)

func argValues(args []string, flag string) []string {
	var out []string
	for i := 0; i < len(args)-1; i++ {
		if args[i] == flag {
			out = append(out, args[i+1])
		}
	}
	return out
}

func hasArgValue(args []string, flag, value string) bool {
	for _, v := range argValues(args, flag) {
		if v == value {
			return true
		}
	}
	return false
}

func TestBuildRunArgsPublishesCatalogPorts(t *testing.T) {
	t.Setenv("VORTANIX_DATA_DIR", t.TempDir())

	cases := []struct {
		game    string
		port    int
		portEnv string
		publish []string
	}{
		{"cs2", 27015, "CS2_PORT=27015", []string{"27015:27015/udp", "27015:27015/tcp"}},
		{"cs16", 27020, "CS16_PORT=27020", []string{"27020:27020/udp", "27020:27020/tcp"}},
		{"palworld", 8211, "GAME_PORT=8211", []string{"8211:8211/udp", "8212:8212/udp", "8221:8221/tcp"}},
		{"valheim", 2456, "GAME_PORT=2456", []string{"2456:2456/udp", "2457:2457/udp", "2458:2458/udp"}},
		{"mcjava", 25565, "SERVER_PORT=25565", []string{"25565:25565/tcp", "25565:25565/udp", "25575:25575/tcp"}},
		{"mcpaper", 25570, "SERVER_PORT=25570", []string{"25570:25570/tcp", "25570:25570/udp", "25580:25580/tcp"}},
		{"rust", 28015, "RUST_PORT=28015", []string{"28015:28015/udp", "28016:28016/udp", "28016:28016/tcp"}},
		{"samp", 7777, "SAMP_PORT=7777", []string{"7777:7777/udp"}},
	}

	for _, tc := range cases {
		args := buildRunArgs("srv-1", tc.game, map[string]any{"memory_mb": 2048, "cpu": 2}, "vortanix/x:latest", tc.port, "")
		if !hasArgValue(args, "-e", tc.portEnv) {
			t.Errorf("%s: не передан %s; args=%v", tc.game, tc.portEnv, args)
		}
		for _, p := range tc.publish {
			if !hasArgValue(args, "-p", p) {
				t.Errorf("%s: не опубликован порт %s; -p=%v", tc.game, p, argValues(args, "-p"))
			}
		}
	}
}

func TestBuildRunArgsAppliesResourceLimits(t *testing.T) {
	t.Setenv("VORTANIX_DATA_DIR", t.TempDir())

	args := buildRunArgs("srv-1", "cs2", map[string]any{"memory_mb": 3072, "cpu": 1.5}, "vortanix/cs2:latest", 27015, "")
	if !hasArgValue(args, "-m", "3072m") {
		t.Errorf("нет лимита памяти; args=%v", args)
	}
	if !hasArgValue(args, "--cpus", "1.5") {
		t.Errorf("нет лимита CPU; args=%v", args)
	}

	bare := buildRunArgs("srv-2", "cs2", nil, "vortanix/cs2:latest", 27015, "")
	if len(argValues(bare, "--cpus")) != 0 {
		t.Errorf("--cpus не должен появляться без лимита; args=%v", bare)
	}
}

func TestBuildRunArgsMinecraftHeap(t *testing.T) {
	t.Setenv("VORTANIX_DATA_DIR", t.TempDir())

	args := buildRunArgs("srv-1", "mcjava", map[string]any{"memory_mb": 2048}, "vortanix/mcjava:latest", 25565, "")
	if !hasArgValue(args, "-e", "MEMORY=1792M") {
		t.Errorf("heap должен быть 2048-256; -e=%v", argValues(args, "-e"))
	}

	cs := buildRunArgs("srv-2", "cs2", map[string]any{"memory_mb": 2048}, "vortanix/cs2:latest", 27015, "")
	for _, v := range argValues(cs, "-e") {
		if strings.HasPrefix(v, "MEMORY=") || v == "EULA=TRUE" {
			t.Errorf("cs2 не должен получать %q", v)
		}
	}
}

func TestBuildRunArgsSkipsPortsWithoutAllocation(t *testing.T) {
	t.Setenv("VORTANIX_DATA_DIR", t.TempDir())

	for _, tc := range []struct {
		game string
		port int
	}{
		{"test", 0},
		{"cs2", 0},
	} {
		args := buildRunArgs("srv-1", tc.game, nil, "alpine:3.20", tc.port, "")
		if got := argValues(args, "-p"); len(got) != 0 {
			t.Errorf("%s/%d: неожиданные публикации портов %v", tc.game, tc.port, got)
		}
	}
}

func TestArchiveKindDetection(t *testing.T) {
	cases := map[string]string{
		"https://example.com/server.zip":              archiveZip,
		"https://example.com/server.tar.gz":           archiveTarGz,
		"https://example.com/server.tgz":              archiveTarGz,
		"https://example.com/server.jar":              archiveJar,
		"https://example.com/server.jar?token=abc":    archiveJar,
		"https://meta.fabricmc.net/v2/.../server/jar": "",
	}
	for url, want := range cases {
		if got := archiveKindFromURL(url); got != want {
			t.Errorf("%s -> %q, ожидалось %q", url, got, want)
		}
	}

	if got := archiveKindFromResponse(downloadMeta{ContentType: "application/java-archive"}); got != archiveJar {
		t.Errorf("java-archive -> %q", got)
	}
	if got := archiveKindFromResponse(downloadMeta{ContentDisposition: `attachment; filename="paper-26.2-112.jar"`}); got != archiveJar {
		t.Errorf("disposition jar -> %q", got)
	}
	if got := archiveKindFromResponse(downloadMeta{ContentType: "application/gzip"}); got != archiveTarGz {
		t.Errorf("gzip -> %q", got)
	}
	if got := archiveKindFromResponse(downloadMeta{ContentType: "text/html"}); got != "" {
		t.Errorf("html не архив, получено %q", got)
	}
}

func TestBuildRunArgsBindsDedicatedIP(t *testing.T) {
	t.Setenv("VORTANIX_DATA_DIR", t.TempDir())

	args := buildRunArgs("srv-1", "mcjava", map[string]any{"memory_mb": 2048},
		"vortanix/mcjava:latest", 25565, "203.0.113.10")
	for _, want := range []string{
		"203.0.113.10:25565:25565/tcp",
		"203.0.113.10:25565:25565/udp",
		"203.0.113.10:25575:25575/tcp",
	} {
		if !hasArgValue(args, "-p", want) {
			t.Errorf("не опубликован %s; -p=%v", want, argValues(args, "-p"))
		}
	}

	bare := buildRunArgs("srv-2", "mcjava", map[string]any{"memory_mb": 2048},
		"vortanix/mcjava:latest", 25565, "")
	if !hasArgValue(bare, "-p", "25565:25565/tcp") {
		t.Errorf("без адреса порт должен публиковаться без префикса; -p=%v", argValues(bare, "-p"))
	}
}
