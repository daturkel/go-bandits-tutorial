package ex2

// EstimateQuantile estimates the q-quantile (0 < q < 1) of a set of
// observations from a histogram, the way Prometheus's histogram_quantile
// does. A histogram does not keep the observations, only how many fell into
// each bucket.
//
// upper[i] is the upper bound of bucket i (ascending; the bucket covers values
// above upper[i-1] and up to upper[i], and the first bucket starts at 0).
// counts[i] is the number of observations in bucket i, NOT cumulative.
//
// Method: find the bucket that contains the observation at rank q*total, then
// assume the observations in that bucket are spread evenly across it and
// interpolate linearly.
//
// Return NaN if there are no observations, q is outside (0, 1), or the two
// slices differ in length.
func EstimateQuantile(q float64, upper []float64, counts []uint64) float64 {
	// TODO
	return 0
}
