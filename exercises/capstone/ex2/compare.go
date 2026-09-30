package ex2

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
	// TODO
	return Inconclusive
}
