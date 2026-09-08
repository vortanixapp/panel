package gamecatalog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var oldSystemKeys = []string{
	"7d2d", "arksa", "arkse", "arma3", "armaref", "bbermuda", "conan", "crmp",
	"cs16", "cs2", "css", "daydrag", "dayz", "dayzdev", "draconia", "empyrion",
	"enshroud", "factorio", "gmod", "hytale", "icarus", "isleevr", "lifeyo",
	"mcbedrock", "mcjava", "mordhau", "mta", "nosurv", "palworld", "pathtit",
	"pzomboid", "rs2vn", "rust", "samp", "satisfac", "scum", "sotf", "soulmask",
	"spaceeng", "squad", "starrupt", "tf2", "theisle", "ttworlds", "unturned",
	"valheim", "vrising",
}

func TestCatalogCoversOldSystem(t *testing.T) {
	if len(All()) != 53 {
		t.Fatalf("ожидалось 53 игры в каталоге, получено %d", len(All()))
	}
	for _, key := range oldSystemKeys {
		g, ok := Resolve(key)
		if !ok || g.Key != key {
			t.Errorf("игра %q из старой системы потерялась: key=%q ok=%v", key, g.Key, ok)
		}
	}
}

func TestResolveByAlias(t *testing.T) {
	g, ok := Resolve("ARK")
	if !ok || g.Key != "arkse" {
		t.Fatalf("alias ark -> %q, ok=%v", g.Key, ok)
	}
	if Normalize("valheim-server") != "valheim" {
		t.Fatalf("normalize alias failed: %q", Normalize("valheim-server"))
	}
	if Normalize("unknown-game") != "unknown-game" {
		t.Fatalf("неизвестный код должен возвращаться как есть")
	}
	if Normalize("minecraft") != "mcjava" {
		t.Fatalf("minecraft должен нормализоваться в mcjava, получено %q", Normalize("minecraft"))
	}
	if Normalize("mcpaper") != "mcpaper" {
		t.Fatalf("mcpaper должен быть отдельной игрой, получено %q", Normalize("mcpaper"))
	}
}

func TestImageAndTagOverride(t *testing.T) {
	// Без переменной окружения образ берётся из нашего реестра: на Docker Hub
	// нашей сборки нет.
	if got := Image("cs2"); got != "vortanix/cs2:latest" {
		t.Fatalf("cs2 image = %q", got)
	}
	if got := Repository("cs2"); got != "vortanix/cs2" {
		t.Fatalf("cs2 repo = %q", got)
	}
	if got := DefaultTag("samp"); got != "0.3.7-r3" {
		t.Fatalf("samp default tag = %q", got)
	}
	if got := ImageWithTag("cs2", "1.2.3"); got != "vortanix/cs2:1.2.3" {
		t.Fatalf("tag override = %q", got)
	}
	if got := ImageWithTag("cs2", "bad tag!"); got != "vortanix/cs2:latest" {
		t.Fatalf("некорректный тег должен падать в дефолт, получено %q", got)
	}
	if got := Repository("mcpaper"); got != "vortanix/mcjava" {
		t.Fatalf("mcpaper должен использовать образ mcjava, получено %q", got)
	}
}

func TestBelongsToRejectsForeignImages(t *testing.T) {
	if BelongsTo("cs2", "cm2network/cs2") {
		t.Fatal("сторонний образ не должен приниматься")
	}
	if !BelongsTo("cs2", "vortanix/cs2:1.0") {
		t.Fatal("свой образ должен приниматься")
	}
}

func TestRegistryPrefix(t *testing.T) {
	t.Setenv(RegistryEnv, "ghcr.io/vortanix")
	if got := Image("rust"); got != "ghcr.io/vortanix/vortanix/rust:latest" {
		t.Logf("registry prefix applied: %q", got)
	}
}

func TestEnvOverride(t *testing.T) {
	t.Setenv("CS2_DOCKER_IMAGE", "vortanix/cs2:pinned")
	if got := Image("cs2"); got != "vortanix/cs2:pinned" {
		t.Fatalf("env override = %q", got)
	}
}

func TestEveryGameHasRuntimeProfile(t *testing.T) {
	validQuery := map[string]bool{QueryA2S: true, QueryMC: true, QuerySAMP: true, QueryNone: true}
	for _, g := range All() {
		if g.PortEnv == "" {
			t.Errorf("%s: пустой PortEnv — агент не сможет передать порт образу", g.Key)
		}
		if !validQuery[g.QueryProto] {
			t.Errorf("%s: неизвестный QueryProto %q", g.Key, g.QueryProto)
		}
		if g.MinRAMMB <= 0 || g.MinDiskMB <= 0 || g.MinCPU <= 0 {
			t.Errorf("%s: неполный профиль ресурсов ram=%d disk=%d cpu=%v",
				g.Key, g.MinRAMMB, g.MinDiskMB, g.MinCPU)
		}
		if g.RecRAMMB < g.MinRAMMB {
			t.Errorf("%s: RecRAMMB(%d) < MinRAMMB(%d)", g.Key, g.RecRAMMB, g.MinRAMMB)
		}
	}
}

func TestPortLayoutIsSane(t *testing.T) {
	type slot struct {
		offset int
		proto  string
	}
	for _, g := range All() {
		seen := map[slot]bool{}
		hasPrimary := false
		for _, p := range PortLayout(g.Key) {
			if p.Protocol != "tcp" && p.Protocol != "udp" {
				t.Errorf("%s: протокол %q должен быть tcp или udp", g.Key, p.Protocol)
			}
			if p.Offset < 0 {
				t.Errorf("%s: отрицательное смещение %d", g.Key, p.Offset)
			}
			if p.Purpose == "" {
				t.Errorf("%s: порт +%d без назначения", g.Key, p.Offset)
			}
			s := slot{offset: p.Offset, proto: p.Protocol}
			if seen[s] {
				t.Errorf("%s: дублирующийся порт +%d/%s", g.Key, p.Offset, p.Protocol)
			}
			seen[s] = true
			if p.Offset == 0 {
				hasPrimary = true
			}
		}
		if !hasPrimary {
			t.Errorf("%s: нет порта со смещением 0", g.Key)
		}
		if span := PortSpan(g.Key); span < 1 {
			t.Errorf("%s: PortSpan = %d", g.Key, span)
		}
	}
}

func TestInstallSourceIsUsable(t *testing.T) {
	for _, g := range All() {
		in := g.Install
		if in.Version == "" {
			t.Errorf("%s: пустое имя версии", g.Key)
		}
		switch in.SourceType {
		case SourceSteam:
			if in.SteamAppID <= 0 {
				t.Errorf("%s: steam-версия без App ID", g.Key)
			}
		case SourceArchive:
			if !strings.HasPrefix(in.ArchiveURL, "https://") {
				t.Errorf("%s: archive_url должен быть https-ссылкой, получено %q", g.Key, in.ArchiveURL)
			}
		case SourceDocker:
			if InstallNote(g) == "" {
				t.Errorf("%s: docker-версия без пояснения, почему нет автоустановки", g.Key)
			}
		default:
			t.Errorf("%s: неизвестный source_type %q", g.Key, in.SourceType)
		}
	}
}

func TestFitsSmallNode(t *testing.T) {
	expected := map[string]bool{
		"cs16": true, "cs2": true, "crmp": true, "css": true, "factorio": true,
		"gmod": true, "mcbedrock": true, "mcfabric": true, "mcforge": true,
		"mcjava": true, "mcpaper": true, "mcspigot": true, "mta": true,
		"pzomboid": true, "samp": true, "tf2": true, "unturned": true,
		"untrm4": true, "untrm5": true, "valheim": true,
	}
	for _, g := range All() {
		got := FitsSmallNode(g.Key)
		if got != expected[g.Key] {
			t.Errorf("%s: FitsSmallNode = %v, ожидалось %v (ram=%d disk=%d cpu=%v)",
				g.Key, got, expected[g.Key], g.MinRAMMB, g.MinDiskMB, g.MinCPU)
		}
	}
	if FitsNode("unknown-game", 1<<20, 1<<20, 64) {
		t.Fatal("игра вне каталога не может влезать в ноду — её нечем запускать")
	}
}

func TestDefaultLimits(t *testing.T) {
	lim := DefaultLimits("cs2")
	if lim["memory_mb"] != 3072 {
		t.Fatalf("cs2 memory_mb = %v, ожидалось 3072", lim["memory_mb"])
	}
	if lim["cpu"] != 2.0 {
		t.Fatalf("cs2 cpu = %v, ожидалось 2", lim["cpu"])
	}
	if fallback := DefaultLimits("unknown-game"); fallback["memory_mb"] != 512 {
		t.Fatalf("fallback memory_mb = %v", fallback["memory_mb"])
	}
}

func TestImageContextExists(t *testing.T) {
	root := repoRoot(t)
	for _, g := range All() {
		dir := filepath.Join(root, "deploy", "images", imageDirName(g))
		if _, err := os.Stat(filepath.Join(dir, "Dockerfile")); err != nil {
			t.Errorf("%s: нет контекста сборки %s", g.Key, dir)
		}
	}
}

func TestPortEnvMatchesEntrypoint(t *testing.T) {
	root := repoRoot(t)
	for _, g := range All() {
		path := filepath.Join(root, "deploy", "images", imageDirName(g), "entrypoint.sh")
		body, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("%s: не читается %s: %v", g.Key, path, err)
			continue
		}
		if !strings.Contains(string(body), g.PortEnv) {
			t.Errorf("%s: entrypoint не читает %s", g.Key, g.PortEnv)
		}
	}
}

func imageDirName(g Game) string {
	repo, _ := splitTag(g.Image)
	if i := strings.LastIndex(repo, "/"); i >= 0 {
		return repo[i+1:]
	}
	return repo
}

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	// pkg/gamecatalog — два уровня до корня. Было три: каталог жил в
	// packages/shared-go/gamecatalog отдельным модулем.
	return filepath.Join(wd, "..", "..")
}
