package pricing

import "math"

const HoursPerMonth = 720

func PaymentMode(tariff map[string]any) string {
	if mode, _ := tariff["payment_mode"].(string); mode == "hourly" {
		return "hourly"
	}
	return "prepaid"
}

func Hourly(tariff map[string]any) bool {
	return PaymentMode(tariff) == "hourly"
}

func HourlyRate(tariff map[string]any, order RentOrder) float64 {
	monthly := MonthlyCost(tariff, order)
	if monthly <= 0 {
		return 0
	}
	return math.Round(monthly/HoursPerMonth*10000) / 10000
}

func DailyCost(tariff map[string]any, order RentOrder) float64 {
	return math.Round(HourlyRate(tariff, order)*24*100) / 100
}

func HourlyBreakdown(tariff map[string]any, order RentOrder) []Line {
	lines := Breakdown(tariff, order, 30)
	out := make([]Line, 0, len(lines))
	for _, line := range lines {
		if line.Key == "period_discount" {
			continue
		}
		out = append(out, Line{
			Key:    line.Key,
			Amount: math.Round(line.Amount/HoursPerMonth*10000) / 10000,
			Qty:    line.Qty,
			Unit:   math.Round(line.Unit/HoursPerMonth*10000) / 10000,
		})
	}
	return out
}
