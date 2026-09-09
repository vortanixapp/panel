package pricing

import "math"

type RentOrder struct {
	Slots           int
	CPUCores        int
	RAMGb           int
	DiskGb          int
	AntiddosEnabled bool
}

func CalculateRentCost(tariff map[string]any, order RentOrder, periodDays int) float64 {
	if periodDays <= 0 {
		periodDays = 30
	}
	factor := float64(periodDays) / 30.0

	billingType, _ := tariff["billing_type"].(string)
	if billingType == "" {
		billingType = "fixed"
	}

	base := 0.0
	extra := 0.0
	addons := 0.0

	switch billingType {
	case "slots":
		pricePerSlot := floatVal(tariff["price_per_slot"])
		if pricePerSlot <= 0 {
			pricePerSlot = floatVal(tariff["price_monthly"])
		}
		slots := order.Slots
		if slots <= 0 {
			slots = intVal(tariff["min_slots"])
			if slots <= 0 {
				slots = 10
			}
		}
		extra = float64(slots) * pricePerSlot * factor
	default:
		priceMonthly := floatVal(tariff["price_monthly"])
		if billingType == "resources" {
			baseMonthly := floatVal(tariff["base_price_monthly"])
			if baseMonthly > 0 {
				base = baseMonthly * factor
			}
			cpu := order.CPUCores
			if cpu <= 0 {
				cpu = intVal(tariff["cpu_cores"])
				if cpu <= 0 {
					cpu = 1
				}
			}
			ram := order.RAMGb
			if ram <= 0 {
				ram = intVal(tariff["ram_gb"])
				if ram <= 0 {
					ram = 1
				}
			}
			disk := order.DiskGb
			if disk <= 0 {
				disk = intVal(tariff["disk_gb"])
				if disk <= 0 {
					disk = 10
				}
			}
			extra = (float64(cpu)*floatVal(tariff["price_per_cpu_core"]) +
				float64(ram)*floatVal(tariff["price_per_ram_gb"]) +
				float64(disk)*floatVal(tariff["price_per_disk_gb"])) * factor
			if base == 0 && extra == 0 {
				base = priceMonthly * factor
			}
		} else {
			base = priceMonthly * factor
		}
	}

	if order.AntiddosEnabled && boolVal(tariff["allow_antiddos"]) {
		addons = floatVal(tariff["antiddos_price"]) * factor
	}

	total := base + extra + addons
	return math.Round(total*100) / 100
}

func floatVal(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int64:
		return float64(n)
	default:
		return 0
	}
}

func intVal(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	default:
		return 0
	}
}

func boolVal(v any) bool {
	switch b := v.(type) {
	case bool:
		return b
	case float64:
		return b != 0
	case string:
		return b == "1" || b == "true"
	default:
		return false
	}
}
