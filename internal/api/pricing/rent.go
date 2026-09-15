package pricing

import (
	"math"
	"strconv"
	"strings"
)

type RentOrder struct {
	Slots           int
	CPUCores        int
	RAMGb           int
	DiskGb          int
	AntiddosEnabled bool
}

type Range struct {
	Min     int
	Max     int
	Step    int
	Default int
}

func (r Range) Configurable() bool {
	return r.Max > r.Min
}

func (r Range) Fit(value int) int {
	if value <= 0 {
		value = r.Default
	}
	value = min(max(value, r.Min), r.Max)
	if r.Step > 1 {
		value = r.Min + int(math.Round(float64(value-r.Min)/float64(r.Step)))*r.Step
		for value > r.Max {
			value -= r.Step
		}
		value = max(value, r.Min)
	}
	return value
}

func ResourceRange(tariff map[string]any, resource string) Range {
	fixedKey, fallback := resource+"_gb", 1
	switch resource {
	case "cpu":
		fixedKey = "cpu_cores"
	case "disk":
		fallback = 10
	}
	fixed := positive(intVal(tariff[fixedKey]), fallback)
	r := Range{Min: fixed, Max: fixed, Step: positive(intVal(tariff[resource+"_step"]), 1), Default: fixed}
	lo, hi := intVal(tariff[resource+"_min"]), intVal(tariff[resource+"_max"])
	if billingType(tariff) != "resources" || (lo <= 0 && hi <= 0) {
		return r
	}
	r.Min = positive(lo, fixed)
	r.Max = hi
	if r.Max <= 0 {
		r.Max = max(fixed, r.Min)
	}
	r.Max = max(r.Max, r.Min)
	r.Default = min(max(fixed, r.Min), r.Max)
	return r
}

func SlotRange(tariff map[string]any) Range {
	lo := positive(intVal(tariff["min_slots"]), 1)
	hi := max(intVal(tariff["max_slots"]), lo)
	return Range{Min: lo, Max: hi, Step: 1, Default: lo}
}

func Resolve(tariff map[string]any, order RentOrder) RentOrder {
	out := RentOrder{
		CPUCores:        ResourceRange(tariff, "cpu").Fit(order.CPUCores),
		RAMGb:           ResourceRange(tariff, "ram").Fit(order.RAMGb),
		DiskGb:          ResourceRange(tariff, "disk").Fit(order.DiskGb),
		AntiddosEnabled: order.AntiddosEnabled && boolVal(tariff["allow_antiddos"]),
	}
	if order.Slots > 0 || billingType(tariff) == "slots" {
		out.Slots = SlotRange(tariff).Fit(order.Slots)
	}
	return out
}

func MinimalOrder(tariff map[string]any) RentOrder {
	return RentOrder{
		Slots:    SlotRange(tariff).Min,
		CPUCores: ResourceRange(tariff, "cpu").Min,
		RAMGb:    ResourceRange(tariff, "ram").Min,
		DiskGb:   ResourceRange(tariff, "disk").Min,
	}
}

func PriceFrom(tariff map[string]any) float64 {
	return MonthlyCost(tariff, MinimalOrder(tariff))
}

func MonthlyCost(tariff map[string]any, order RentOrder) float64 {
	total := 0.0
	switch billingType(tariff) {
	case "slots":
		slots := positive(order.Slots, positive(intVal(tariff["min_slots"]), 1))
		total = float64(slots) * floatVal(tariff["price_per_slot"])
	case "resources":
		cpu := positive(order.CPUCores, positive(intVal(tariff["cpu_cores"]), 1))
		ram := positive(order.RAMGb, positive(intVal(tariff["ram_gb"]), 1))
		disk := positive(order.DiskGb, positive(intVal(tariff["disk_gb"]), 10))
		total = floatVal(tariff["base_price_monthly"]) +
			float64(cpu)*floatVal(tariff["price_per_cpu_core"]) +
			float64(ram)*floatVal(tariff["price_per_ram_gb"]) +
			float64(disk)*floatVal(tariff["price_per_disk_gb"])
	default:
		total = floatVal(tariff["price_monthly"])
	}
	if order.AntiddosEnabled && boolVal(tariff["allow_antiddos"]) {
		total += floatVal(tariff["antiddos_price"])
	}
	return math.Round(total*100) / 100
}

func PeriodDiscount(tariff map[string]any, periodDays int) float64 {
	discounts, ok := tariff["discounts"].(map[string]any)
	if !ok {
		return 0
	}
	percent := floatVal(discounts[strconv.Itoa(periodDays)])
	if percent <= 0 {
		return 0
	}
	return math.Min(percent, 100)
}

func CalculateRentCost(tariff map[string]any, order RentOrder, periodDays int) float64 {
	if periodDays <= 0 {
		periodDays = 30
	}
	total := MonthlyCost(tariff, order) * float64(periodDays) / 30.0
	if percent := PeriodDiscount(tariff, periodDays); percent > 0 {
		total -= total * percent / 100
	}
	return math.Round(total*100) / 100
}

func billingType(tariff map[string]any) string {
	s, _ := tariff["billing_type"].(string)
	return s
}

func positive(v, fallback int) int {
	if v > 0 {
		return v
	}
	return fallback
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
	case string:
		f, _ := strconv.ParseFloat(strings.TrimSpace(n), 64)
		return f
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
	case *int:
		if n != nil {
			return *n
		}
		return 0
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
