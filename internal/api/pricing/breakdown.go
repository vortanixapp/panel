package pricing

import "math"

type Line struct {
	Key     string  `json:"key"`
	Amount  float64 `json:"amount"`
	Qty     int     `json:"qty,omitempty"`
	Unit    float64 `json:"unit,omitempty"`
	Percent float64 `json:"percent,omitempty"`
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

func Breakdown(tariff map[string]any, order RentOrder, periodDays int) []Line {
	if periodDays <= 0 {
		periodDays = 30
	}
	scale := float64(periodDays) / 30.0
	lines := []Line{}
	add := func(key string, monthly float64, qty int, unit float64) {
		if monthly <= 0 && qty == 0 {
			return
		}
		lines = append(lines, Line{
			Key:    key,
			Amount: round2(monthly * scale),
			Qty:    qty,
			Unit:   round2(unit * scale),
		})
	}

	switch billingType(tariff) {
	case "slots":
		slots := positive(order.Slots, positive(intVal(tariff["min_slots"]), 1))
		perSlot := floatVal(tariff["price_per_slot"])
		add("slots", float64(slots)*perSlot, slots, perSlot)
	case "resources":
		if base := floatVal(tariff["base_price_monthly"]); base > 0 {
			add("base", base, 0, 0)
		}
		cpu := positive(order.CPUCores, positive(intVal(tariff["cpu_cores"]), 1))
		ram := positive(order.RAMGb, positive(intVal(tariff["ram_gb"]), 1))
		disk := positive(order.DiskGb, positive(intVal(tariff["disk_gb"]), 10))
		add("cpu", float64(cpu)*floatVal(tariff["price_per_cpu_core"]), cpu, floatVal(tariff["price_per_cpu_core"]))
		add("ram", float64(ram)*floatVal(tariff["price_per_ram_gb"]), ram, floatVal(tariff["price_per_ram_gb"]))
		add("disk", float64(disk)*floatVal(tariff["price_per_disk_gb"]), disk, floatVal(tariff["price_per_disk_gb"]))
	default:
		add("tariff", floatVal(tariff["price_monthly"]), 0, 0)
	}

	if order.AntiddosEnabled && boolVal(tariff["allow_antiddos"]) {
		add("antiddos", floatVal(tariff["antiddos_price"]), 0, 0)
	}

	subtotal := 0.0
	for _, line := range lines {
		subtotal += line.Amount
	}
	if percent := PeriodDiscount(tariff, periodDays); percent > 0 {
		lines = append(lines, Line{
			Key:     "period_discount",
			Amount:  -round2(subtotal * percent / 100),
			Percent: percent,
		})
	}
	return lines
}
