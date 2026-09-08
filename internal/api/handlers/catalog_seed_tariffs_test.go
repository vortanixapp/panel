package handlers

import (
	"testing"

	"github.com/vortanixapp/panel/pkg/gamecatalog"
)

func TestEveryTariffMeetsGameMinimum(t *testing.T) {
	for _, g := range gamecatalog.All() {
		specs := buildTariffSpecs(g)
		if len(specs) != len(tariffTiers) {
			t.Fatalf("%s: ступеней %d, ожидалось %d", g.Key, len(specs), len(tariffTiers))
		}
		for _, s := range specs {
			if g.MinRAMMB > 0 && s.RAMMB < g.MinRAMMB {
				t.Errorf("%s/%s: памяти %d МБ, минимум игры %d МБ", g.Key, s.Slug, s.RAMMB, g.MinRAMMB)
			}
			if g.MinDiskMB > 0 && s.DiskMB < g.MinDiskMB {
				t.Errorf("%s/%s: диска %d МБ, минимум игры %d МБ", g.Key, s.Slug, s.DiskMB, g.MinDiskMB)
			}
			if g.MinCPU > 0 && s.CPUCores < g.MinCPU {
				t.Errorf("%s/%s: ядер %.1f, минимум игры %.1f", g.Key, s.Slug, s.CPUCores, g.MinCPU)
			}
			if s.Price <= 0 {
				t.Errorf("%s/%s: цена %.0f", g.Key, s.Slug, s.Price)
			}
			if s.SlotsMax <= 0 {
				t.Errorf("%s/%s: слотов %d", g.Key, s.Slug, s.SlotsMax)
			}
			if g.MaxSlots > 0 && s.SlotsMax > g.MaxSlots {
				t.Errorf("%s/%s: слотов %d, потолок игры %d", g.Key, s.Slug, s.SlotsMax, g.MaxSlots)
			}
		}
	}
}

func TestTiersGrowMonotonically(t *testing.T) {
	for _, g := range gamecatalog.All() {
		specs := buildTariffSpecs(g)
		for i := 1; i < len(specs); i++ {
			prev, cur := specs[i-1], specs[i]
			if cur.Price < prev.Price {
				t.Errorf("%s: %s (%.0f) дешевле %s (%.0f)", g.Key, cur.Slug, cur.Price, prev.Slug, prev.Price)
			}
			if cur.RAMMB < prev.RAMMB {
				t.Errorf("%s: %s даёт меньше памяти, чем %s", g.Key, cur.Slug, prev.Slug)
			}
			if cur.CPUCores < prev.CPUCores {
				t.Errorf("%s: %s даёт меньше ядер, чем %s", g.Key, cur.Slug, prev.Slug)
			}
			if cur.SlotsMax < prev.SlotsMax {
				t.Errorf("%s: %s даёт меньше слотов, чем %s", g.Key, cur.Slug, prev.Slug)
			}
		}
	}
}

func TestEntryPriceStaysReasonable(t *testing.T) {
	mc, ok := gamecatalog.Resolve("mcjava")
	if !ok {
		t.Fatal("mcjava не найдена в каталоге")
	}
	start := buildTariffSpecs(mc)[0]
	if start.Price > 500 {
		t.Errorf("вход в Minecraft стоит %.0f ₽ — слишком дорого для 1 ГБ", start.Price)
	}
}

func TestTariffSlugsAreUnique(t *testing.T) {
	seen := map[string]string{}
	for _, g := range gamecatalog.All() {
		for _, s := range buildTariffSpecs(g) {
			if owner, dup := seen[s.Slug]; dup {
				t.Fatalf("слаг %s занят и игрой %s, и %s", s.Slug, owner, g.Key)
			}
			seen[s.Slug] = g.Key
		}
	}
}

func TestPriceFormula(t *testing.T) {
	got := tariffPrice(2048, 4096, 2)
	want := 50.0 + 2*90 + 2*120 + 4*3
	if got < want || got > want+tariffPriceRounding {
		t.Errorf("цена %.0f, ожидалась около %.0f", got, want)
	}
}
