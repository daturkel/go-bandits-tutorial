package ex3

// Epsilon returns an exploration rate that halves every halfLife steps:
//
//	start * 0.5^(t/halfLife)
//
// but never drops below floor. t is the number of steps taken so far.
func Epsilon(t int, start, floor, halfLife float64) float64 {
	// TODO
	return 0
}
