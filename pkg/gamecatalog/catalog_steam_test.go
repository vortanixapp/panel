package gamecatalog

import "testing"

var verifiedSteamApps = map[string]struct {
	appID int64
	name  string
}{
	"cs2":      {740, "Counter-Strike Global Offensive - Dedicated Server"},
	"soulmask": {3017300, "Soulmask Dedicated Server For Linux"},
	"cs16":     {90, "Half-Life Dedicated Server"},
	"css":      {232330, "Counter-Strike: Source Dedicated Server"},
	"tf2":      {232250, "Team Fortress 2 Dedicated Server"},
	"gmod":     {4020, "Garry's Mod Dedicated Server"},
	"rust":     {258550, "Rust Dedicated Server"},
	"dayz":     {223350, "DayZ Server"},
	"valheim":  {896660, "Valheim Dedicated Server"},
}

func TestVerifiedSteamAppIDs(t *testing.T) {
	for key, want := range verifiedSteamApps {
		g, ok := Resolve(key)
		if !ok {
			t.Errorf("%s: нет в каталоге", key)
			continue
		}
		if g.Install.SteamAppID != want.appID {
			t.Errorf("%s: App ID %d, ожидался %d (%s)",
				key, g.Install.SteamAppID, want.appID, want.name)
		}
	}
}

func TestHLDSRequiresModConfig(t *testing.T) {
	g, ok := Resolve("cs16")
	if !ok {
		t.Fatal("cs16 нет в каталоге")
	}
	if g.Install.SteamModConfig != "cstrike" {
		t.Errorf("cs16: SteamModConfig = %q, ожидалось \"cstrike\"", g.Install.SteamModConfig)
	}
}

var noLinuxServer = []string{
	"bbermuda", "daydrag", "draconia", "lifeyo", "nosurv",
	"pathtit", "rs2vn", "scum", "starrupt", "ttworlds",
}

func TestNoLinuxServerGamesAreDisabled(t *testing.T) {
	for _, key := range noLinuxServer {
		g, ok := Resolve(key)
		if !ok {
			t.Errorf("%s: нет в каталоге", key)
			continue
		}
		if g.NoLinuxServer == "" {
			t.Errorf("%s: должна быть указана причина отсутствия Linux-сервера", key)
		}
		if g.Install.SourceType == SourceSteam {
			t.Errorf("%s: source_type=steam при отсутствии Linux-сервера", key)
		}
		if Enabled(key) {
			t.Errorf("%s: игра без Linux-сервера не должна включаться тенанту", key)
		}
		if LinuxSupported(key) {
			t.Errorf("%s: LinuxSupported должен быть false", key)
		}
	}
}

func TestSupportedGamesHaveNoBlockReason(t *testing.T) {
	blocked := map[string]bool{}
	for _, k := range noLinuxServer {
		blocked[k] = true
	}
	for _, g := range All() {
		if blocked[g.Key] {
			continue
		}
		if g.NoLinuxServer != "" {
			t.Errorf("%s: неожиданная причина блокировки %q", g.Key, g.NoLinuxServer)
		}
	}
}
