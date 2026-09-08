package handlers

import (
	"strings"
	"testing"
)

func TestServerStateBlocksCatalogChanges(t *testing.T) {
	cases := []struct {
		name  string
		state serverOperableState
		want  string
	}{
		{"истёкшая аренда", serverOperableState{Expired: true, Provisioning: "ready"}, "срок аренды"},
		{"идёт установка", serverOperableState{Provisioning: "provisioning"}, "устанавливается"},
		{"ещё не начата", serverOperableState{Provisioning: "pending"}, "устанавливается"},
		{"удаляется", serverOperableState{Provisioning: "deprovisioning"}, "удаляется"},
		{"рабочий сервер", serverOperableState{Provisioning: "ready"}, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.state.ensureAcceptsChanges()
			if c.want == "" {
				if err != nil {
					t.Errorf("рабочий сервер отклонён: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("операция разрешена, хотя не должна")
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Errorf("сообщение %q не содержит %q", err.Error(), c.want)
			}
		})
	}
}

func TestPluginSupportsGame(t *testing.T) {
	all := pluginInstallSpec{}
	star := pluginInstallSpec{SupportedGames: []string{"*"}}
	listed := pluginInstallSpec{SupportedGames: []string{"cs16", "rust"}}

	for _, spec := range []pluginInstallSpec{all, star} {
		if !spec.supportsGame("valheim") {
			t.Error("пустой список и «*» означают «все игры»")
		}
	}
	if !listed.supportsGame("CS16") {
		t.Error("сравнение игр должно быть без учёта регистра")
	}
	if listed.supportsGame("valheim") {
		t.Error("плагин для cs16 и rust не должен ставиться на valheim")
	}
}

func TestPluginInstallGuards(t *testing.T) {
	base := pluginInstallSpec{Active: true, InstallPath: "addons/amxmodx", HasArchive: true}

	if err := base.ensureInstallable("cs16"); err != nil {
		t.Fatalf("обычный плагин отклонён: %v", err)
	}

	off := base
	off.Active = false
	if err := off.ensureInstallable("cs16"); err == nil || !strings.Contains(err.Error(), "отключён") {
		t.Errorf("отключённый плагин должен отвергаться, получено %v", err)
	}

	traversal := base
	traversal.InstallPath = "addons/../../etc"
	if err := traversal.ensureInstallable("cs16"); err == nil {
		t.Error("путь установки с «..» должен отвергаться")
	}

	wrongGame := base
	wrongGame.SupportedGames = []string{"rust"}
	if err := wrongGame.ensureInstallable("cs16"); err == nil || !strings.Contains(err.Error(), "игру") {
		t.Errorf("несовместимая игра должна отвергаться, получено %v", err)
	}

	// Плагин без архива и без действий не изменил бы на сервере ни байта, но
	// установка отчиталась бы об успехе.
	empty := pluginInstallSpec{Active: true, InstallPath: "addons"}
	if err := empty.ensureInstallable("cs16"); err == nil {
		t.Error("пустой плагин должен отвергаться")
	}

	// Плагин из одних только действий с файлами — законный случай.
	actionsOnly := pluginInstallSpec{Active: true, HasFileActions: true}
	if err := actionsOnly.ensureInstallable("cs16"); err != nil {
		t.Errorf("плагин из одних действий должен ставиться: %v", err)
	}
}

// Пустой путь установки означает корень каталога сервера: удаление по нему
// снесло бы сервер целиком.
func TestPluginUninstallRefusesServerRoot(t *testing.T) {
	root := pluginInstallSpec{InstallPath: ""}
	if err := root.ensureUninstallable(); err == nil || !strings.Contains(err.Error(), "корня") {
		t.Errorf("удаление из корня должно отвергаться, получено %v", err)
	}

	rootWithOps := pluginInstallSpec{InstallPath: "", HasUninstallOps: true}
	if err := rootWithOps.ensureUninstallable(); err != nil {
		t.Errorf("плагин со своими действиями удаления снимается и из корня: %v", err)
	}

	normal := pluginInstallSpec{InstallPath: "addons/amxmodx"}
	if err := normal.ensureUninstallable(); err != nil {
		t.Errorf("обычное удаление отклонено: %v", err)
	}
}

func TestMapInstallGuards(t *testing.T) {
	base := mapInstallSpec{Active: true, HasArchive: true, HasFiles: true, GameSlug: "cs16"}

	if err := base.ensureInstallable("cs16"); err != nil {
		t.Fatalf("обычная карта отклонена: %v", err)
	}

	off := base
	off.Active = false
	if err := off.ensureInstallable("cs16"); err == nil || !strings.Contains(err.Error(), "отключена") {
		t.Errorf("отключённая карта должна отвергаться, получено %v", err)
	}

	wrongGame := base
	wrongGame.GameSlug = "rust"
	if err := wrongGame.ensureInstallable("cs16"); err == nil {
		t.Error("карта другой игры должна отвергаться")
	}

	// Карта без игры считается общей и ставится куда угодно.
	anyGame := base
	anyGame.GameSlug = ""
	if err := anyGame.ensureInstallable("valheim"); err != nil {
		t.Errorf("карта без привязки к игре должна ставиться: %v", err)
	}

	noFiles := base
	noFiles.HasFiles = false
	if err := noFiles.ensureInstallable("cs16"); err == nil {
		t.Error("без списка файлов карту потом нечем удалить — установка должна отвергаться")
	}

	noArchive := base
	noArchive.HasArchive = false
	if err := noArchive.ensureInstallable("cs16"); err == nil {
		t.Error("карта без архива не может быть установлена")
	}
}
