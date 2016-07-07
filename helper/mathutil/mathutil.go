package mathutil

import "math"

// Credits to https://gist.github.com/pelegm/c48cff315cd223f7cf7b
// Rounds a value val starting at roundOn, and keep places decimal
// Round(123.555555, .5, 3) -> 123.556
// Round(123.558, .5, 2) -> 123.56
// Round(123.00001, 0, 0) -> 124
func Round(val float64, roundOn float64, places int ) (newVal float64) {
	var round float64
	pow := math.Pow(10, float64(places))
	digit := pow * val
	_, div := math.Modf(digit)
	_div := math.Copysign(div, val)
	_roundOn := math.Copysign(roundOn, val)
	if _div >= _roundOn {
		round = math.Ceil(digit)
	} else {
		round = math.Floor(digit)
	}
	newVal = round / pow
	return
}
