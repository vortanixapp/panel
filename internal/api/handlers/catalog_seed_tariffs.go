package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"math"

	"github.com/vortanix/vortanix/pkg/gamecatalog"
)

const (
	tariffBaseRUB       = 50.0
	tariffPerRAMGbRUB   = 90.0
	tariffPerCPUCoreRUB = 120.0
	tariffPerDiskGbRUB  = 3.0
	tariffPriceRounding = 10.0
)

type tariffTier struct {
	slugSuffix     string
	name           string
	useRecommended bool
	ramFactor      float64
	diskFactor     float64
	cpuExtra       float64
	slotsFactor    float64
	position       int
}

var tariffTiers = []tariffTier{
	{slugSuffix: "start", name: "Старт", useRecommended: false, ramFactor: 1.0, diskFactor: 1.0, cpuExtra: 0, slotsFactor: 0.25, position: 10},
	{slugSuffix: "optimal", name: "Оптимальный", useRecommended: true, ramFactor: 1.0, diskFactor: 1.5, cpuExtra: 1, slotsFactor: 0.5, position: 20},
	{slugSuffix: "max", name: "Максимум", useRecommended: true, ramFactor: 1.5, diskFactor: 2.0, cpuExtra: 2, slotsFactor: 1.0, position: 30},
}

const (
	fallbackRAMMB   = 2048
	fallbackDiskMB  = 8192
	fallbackCPU     = 1.0
	fallbackMaxSlot = 128
)

type tariffSpec struct {
	Slug     string
	Name     string
	RAMMB    int
	DiskMB   int
	CPUCores float64
	SlotsMin int
	SlotsMax int
	Price    float64
	Position int
}

func buildTariffSpecs(g gamecatalog.Game) []tariffSpec {
	minRAM := g.MinRAMMB
	if minRAM <= 0 {
		minRAM = fallbackRAMMB
	}
	recRAM := g.RecRAMMB
	if recRAM < minRAM {
		recRAM = minRAM
	}
	baseDisk := g.MinDiskMB
	if baseDisk <= 0 {
		baseDisk = fallbackDiskMB
	}
	baseCPU := g.MinCPU
	if baseCPU <= 0 {
		baseCPU = fallbackCPU
	}
	maxSlots := g.MaxSlots
	if maxSlots <= 0 {
		maxSlots = fallbackMaxSlot
	}

	specs := make([]tariffSpec, 0, len(tariffTiers))
	for _, tier := range tariffTiers {
		base := minRAM
		if tier.useRecommended {
			base = recRAM
		}
		ram := roundToStep(float64(base)*tier.ramFactor, 1024)
		disk := roundToStep(float64(baseDisk)*tier.diskFactor, 1024)
		cpu := baseCPU + tier.cpuExtra

		if ram < g.MinRAMMB {
			ram = g.MinRAMMB
		}
		if disk < g.MinDiskMB {
			disk = g.MinDiskMB
		}

		slotsMax := int(math.Round(float64(maxSlots) * tier.slotsFactor))
		if slotsMax < 4 {
			slotsMax = 4
		}
		if slotsMax > maxSlots {
			slotsMax = maxSlots
		}

		specs = append(specs, tariffSpec{
			Slug:     fmt.Sprintf("%s-%s", g.Key, tier.slugSuffix),
			Name:     fmt.Sprintf("%s · %s", g.Name, tier.name),
			RAMMB:    ram,
			DiskMB:   disk,
			CPUCores: cpu,
			SlotsMin: 1,
			SlotsMax: slotsMax,
			Price:    tariffPrice(ram, disk, cpu),
			Position: tier.position,
		})
	}
	return specs
}

func tariffPrice(ramMB, diskMB int, cpuCores float64) float64 {
	price := tariffBaseRUB +
		float64(ramMB)/1024*tariffPerRAMGbRUB +
		cpuCores*tariffPerCPUCoreRUB +
		float64(diskMB)/1024*tariffPerDiskGbRUB
	return math.Ceil(price/tariffPriceRounding) * tariffPriceRounding
}

func roundToStep(value, step float64) int {
	if step <= 0 {
		return int(math.Round(value))
	}
	return int(math.Ceil(value/step) * step)
}

func upsertTariffs(ctx context.Context, db catalogDB, tenantID, gameID string, g gamecatalog.Game) error {
	for _, spec := range buildTariffSpecs(g) {
		meta, err := json.Marshal(map[string]any{
			"generated_from": "gamecatalog",
			"game_key":       g.Key,
		})
		if err != nil {
			return err
		}

		if _, err := db.Exec(ctx, `
			INSERT INTO core.tariffs (
				tenant_id, game_id, name, slug, billing_type, price_monthly, currency,
				slots_min, slots_max, cpu_cores, ram_mb, disk_mb, position, active, meta
			)
			VALUES ($1, $2, $3, $4, 'resources', $5, 'RUB', $6, $7, $8, $9, $10, $11, true, $12::jsonb)
			ON CONFLICT (tenant_id, slug) DO UPDATE SET
				price_monthly = EXCLUDED.price_monthly,
				cpu_cores     = EXCLUDED.cpu_cores,
				ram_mb        = EXCLUDED.ram_mb,
				disk_mb       = EXCLUDED.disk_mb,
				slots_max     = EXCLUDED.slots_max
		`,
			tenantID, gameID, spec.Name, spec.Slug, spec.Price,
			spec.SlotsMin, spec.SlotsMax, spec.CPUCores, spec.RAMMB, spec.DiskMB,
			spec.Position, meta,
		); err != nil {
			return err
		}
	}
	return nil
}
