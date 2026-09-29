package bandit

import (
	"math"
	"math/rand/v2"
)

// SampleGamma draws from Gamma(shape, 1) using the Marsaglia-Tsang method.
// math/rand/v2 has normal and exponential variates but no gamma or beta.
func SampleGamma(shape float64, rng *rand.Rand) float64 {
	if shape < 1 {
		// Boost: Gamma(a) = Gamma(a+1) * U^(1/a).
		return SampleGamma(shape+1, rng) * math.Pow(rng.Float64(), 1/shape)
	}
	d := shape - 1.0/3
	c := 1 / math.Sqrt(9*d)
	for {
		x := rng.NormFloat64()
		v := 1 + c*x
		if v <= 0 {
			continue
		}
		v = v * v * v
		u := rng.Float64()
		if math.Log(u) < 0.5*x*x+d-d*v+d*math.Log(v) {
			return d * v
		}
	}
}

// SampleBeta draws from Beta(a, b) as X/(X+Y) with X~Gamma(a), Y~Gamma(b).
func SampleBeta(a, b float64, rng *rand.Rand) float64 {
	x := SampleGamma(a, rng)
	y := SampleGamma(b, rng)
	return x / (x + y)
}
