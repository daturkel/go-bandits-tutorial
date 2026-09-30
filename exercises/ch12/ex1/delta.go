package ex1

// Totals is one arm's running summary, as in the banditlab store.
type Totals struct {
	Pulls     int64
	RewardSum float64
}

// Delta returns what was added between two readings of a store's totals:
// newer[i] minus older[i], arm by arm. A replica that pushes deltas instead
// of totals can be merged into shared state without being counted twice.
//
// It returns ok == false, and no slice, if the readings have different
// lengths or any arm's pull count went down (which would mean the store was
// reset, not that new data arrived).
func Delta(newer, older []Totals) (delta []Totals, ok bool) {
	// TODO
	return nil, false
}
