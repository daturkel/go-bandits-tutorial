package bandit

import "testing"

// selectCounts runs n selections without updating and counts each arm.
func selectCounts(p Policy, nArms, n int) []int {
	counts := make([]int, nArms)
	for range n {
		counts[p.Select()]++
	}
	return counts
}

func TestThompsonRejectsZeroArms(t *testing.T) {
	if _, err := NewThompson(0, NewRNG(1, StreamPolicy)); err == nil {
		t.Error("zero arms should be an error")
	}
}

func TestThompsonSelectReturnsValidArms(t *testing.T) {
	p, _ := NewThompson(5, NewRNG(1, StreamPolicy))
	for i := range 500 {
		if arm := p.Select(); arm < 0 || arm >= 5 {
			t.Fatalf("Select #%d returned arm %d", i, arm)
		}
	}
}

// With no evidence every arm has the same posterior, so every arm should be
// tried about equally often.
func TestThompsonExploresBeforeItLearns(t *testing.T) {
	p, _ := NewThompson(3, NewRNG(1, StreamPolicy))
	for arm, n := range selectCounts(p, 3, 3000) {
		if n < 800 || n > 1200 {
			t.Errorf("arm %d chosen %d/3000 times before any data, want about 1000", arm, n)
		}
	}
}

// After strong evidence for one arm it should be chosen almost every time.
func TestThompsonExploitsAfterLearning(t *testing.T) {
	p, _ := NewThompson(3, NewRNG(2, StreamPolicy))
	for range 50 {
		p.Update(0, 0)
		p.Update(1, 1)
		p.Update(2, 0)
	}
	if counts := selectCounts(p, 3, 1000); counts[1] < 950 {
		t.Errorf("arm 1 had 50 wins and the others 50 losses, but was chosen %d/1000 times", counts[1])
	}
}

// Failures count as evidence too: the arm with a poor record loses ground.
func TestThompsonUpdateRecordsFailures(t *testing.T) {
	p, _ := NewThompson(2, NewRNG(3, StreamPolicy))
	for range 30 {
		p.Update(0, 0)
	}
	if counts := selectCounts(p, 2, 1000); counts[0] > 100 {
		t.Errorf("arm 0 failed 30 times but was chosen %d/1000 times", counts[0])
	}
}
