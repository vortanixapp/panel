package gamecatalog

import (
	"os"
	"strings"
	"testing"
)

func TestEveryGameHasLaunch(t *testing.T) {
	for _, g := range All() {
		l, ok := launches[g.Key]
		if !ok {
			t.Errorf("%s: нет записи в таблице запуска", g.Key)
			continue
		}
		if l.Runtime == "" {
			t.Errorf("%s: не задан рантайм", g.Key)
		}
		if _, known := runtimeImages[l.Runtime]; !known {
			t.Errorf("%s: неизвестный рантайм %q", g.Key, l.Runtime)
		}
	}
}

func TestLaunchPortEnvMatchesGame(t *testing.T) {
	for _, g := range All() {
		l, ok := launches[g.Key]
		if !ok {
			continue
		}
		if g.PortEnv == "" {
			continue
		}
		if l.PortEnv != g.PortEnv {
			t.Errorf("%s: PortEnv расходится — в каталоге %q, в запуске %q",
				g.Key, g.PortEnv, l.PortEnv)
		}
	}
}

func TestLaunchBinariesAreUsable(t *testing.T) {
	for key, l := range launches {
		for _, b := range l.Binaries {
			if strings.Contains(b, ":") {
				t.Errorf("%s: путь %q содержит двоеточие — разделитель списка", key, b)
			}
			if !strings.HasPrefix(b, "./") {
				t.Errorf("%s: путь %q должен быть относительным (./…)", key, b)
			}
		}
	}
}

func TestRuntimesAreFewerThanGames(t *testing.T) {
	used := map[Runtime]bool{}
	for _, l := range launches {
		used[l.Runtime] = true
	}
	if len(used) > len(AllRuntimes()) {
		t.Errorf("используется рантаймов %d, объявлено %d", len(used), len(AllRuntimes()))
	}
	if len(used) >= len(All()) {
		t.Fatalf("рантаймов %d при %d играх — смысл общего образа потерян", len(used), len(All()))
	}
	t.Logf("%d игр обслуживаются %d рантаймами", len(All()), len(used))
}

func TestRuntimeEnvIsComplete(t *testing.T) {
	env := RuntimeEnv("arkse")
	if env["VTX_GAME_KEY"] != "arkse" {
		t.Errorf("ключ игры: %q", env["VTX_GAME_KEY"])
	}
	if env["VTX_GAME_NAME"] == "" {
		t.Error("название игры не передаётся")
	}
	bins := env["VTX_SERVER_BINARIES"]
	if !strings.Contains(bins, "ShooterGameServer") {
		t.Errorf("список бинарей не похож на ARK: %q", bins)
	}
	if env["VTX_PORT_ENV"] != "GAME_PORT" {
		t.Errorf("переменная порта: %q", env["VTX_PORT_ENV"])
	}

	if RuntimeEnv("ark-survival-evolved")["VTX_GAME_KEY"] != "arkse" {
		t.Error("синоним игры не разрешается в тот же ключ")
	}
	if RuntimeEnv("такой-игры-нет") != nil {
		t.Error("для неизвестной игры переменных быть не должно")
	}
}

func TestRuntimeImageRespectsOverrides(t *testing.T) {
	if got := RuntimeImage("arkse"); got != "vortanix/runtime-steam:latest" {
		t.Errorf("образ по умолчанию: %q", got)
	}

	t.Setenv("VORTANIX_RUNTIME_STEAM_IMAGE", "ghcr.io/acme/steam:v2")
	if got := RuntimeImage("arkse"); got != "ghcr.io/acme/steam:v2" {
		t.Errorf("переопределение образа не сработало: %q", got)
	}
	_ = os.Unsetenv("VORTANIX_RUNTIME_STEAM_IMAGE")

	t.Setenv(RegistryEnv, "ghcr.io/vortanix")
	if got := RuntimeImage("arkse"); got != "ghcr.io/vortanix/vortanix/runtime-steam:latest" {
		t.Errorf("префикс реестра не применён: %q", got)
	}
}

func TestMinecraftRuntimesDiffer(t *testing.T) {
	java, _ := LaunchOf("mcjava")
	bedrock, _ := LaunchOf("mcbedrock")
	if java.Runtime != RuntimeJava {
		t.Errorf("mcjava: рантайм %q", java.Runtime)
	}
	if bedrock.Runtime == RuntimeJava {
		t.Error("mcbedrock не нуждается в JVM")
	}
}
