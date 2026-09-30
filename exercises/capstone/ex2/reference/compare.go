package ex2

import "math"

// Verdict says what repeated measurements of two policies support.
type Verdict int

const (
	Inconclusive Verdict = iota // the difference is within what noise could explain
	AIsLower                    // a's mean is lower, clearly
	BIsLower                    // b's mean is lower, clearly
)

// Compare decides which of two sets of measurements (say, regret per 1000
// selections from repeated runs of policy A and policy B) has the lower mean,
// if that is clear. The difference of the means is clear when it is more than
// z standard errors of the difference; the standard error of a difference of
// independent means is sqrt(se_a^2 + se_b^2), where se is s/sqrt(n) and s is
// the sample standard deviation (n-1 in the denominator).
//
// With fewer than two measurements in either set the noise cannot be
// estimated, so the answer is Inconclusive.
func Compare(a, b []float64, z float64) Verdict {
	if len(a) < 2 || len(b) < 2 {
		return Inconclusive
	}
	ma, va := meanVar(a)
	mb, vb := meanVar(b)
	se := math.Sqrt(va/float64(len(a)) + vb/float64(len(b)))
	diff := ma - mb
	switch {
	case diff == 0:
		return Inconclusive
	case math.Abs(diff) <= z*se:
		return Inconclusive
	case diff < 0:
		return AIsLower
	default:
		return BIsLower
	}
}

// meanVar returns the mean and the sample variance (n-1 in the denominator).
func meanVar(x []float64) (mean, variance float64) {
	for _, v := range x {
		mean += v
	}
	mean /= float64(len(x))
	for _, v := range x {
		variance += (v - mean) * (v - mean)
	}
	return mean, variance / float64(len(x)-1)
}
