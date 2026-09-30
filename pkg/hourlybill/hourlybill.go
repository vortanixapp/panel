package hourlybill

import "math"

func Split(rate, carry float64) (charge, newCarry float64) {
	owed := math.Round((rate+carry)*1e6) / 1e6
	cents := math.Floor(owed*100 + 1e-6)
	charge = cents / 100
	newCarry = math.Round((owed-charge)*1e6) / 1e6
	if newCarry < 0 {
		newCarry = 0
	}
	return charge, newCarry
}
