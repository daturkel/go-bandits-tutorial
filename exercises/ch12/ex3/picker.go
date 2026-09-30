package ex3

// Picker hands out replica URLs in rotation, for a client that spreads its
// requests over several instances. Next is called from many goroutines at
// once and must be safe for that (run the tests with -race).
type Picker struct {
	// add whatever fields you need
}

// NewPicker returns a Picker over urls. It panics if urls is empty.
func NewPicker(urls []string) *Picker {
	// TODO
	return &Picker{}
}

// Next returns the next URL: the first call returns urls[0], the second
// urls[1], and so on, wrapping around. Over any run of len(urls)*k calls,
// including concurrent ones, every URL is returned exactly k times.
func (p *Picker) Next() string {
	// TODO
	return ""
}
