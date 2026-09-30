package ex2

import (
	"sync/atomic"
	"testing"
	"time"
)

func makeTasks(count int, inFlight, peak, ran *atomic.Int32) []func() {
	tasks := make([]func(), count)
	for i := range tasks {
		tasks[i] = func() {
			n := inFlight.Add(1)
			for {
				p := peak.Load()
				if n <= p || peak.CompareAndSwap(p, n) {
					break
				}
			}
			time.Sleep(5 * time.Millisecond)
			inFlight.Add(-1)
			ran.Add(1)
		}
	}
	return tasks
}

func TestLimitBoundsConcurrency(t *testing.T) {
	for _, n := range []int{1, 3, 8} {
		var inFlight, peak, ran atomic.Int32
		Limit(n, makeTasks(24, &inFlight, &peak, &ran))
		if ran.Load() != 24 {
			t.Errorf("n=%d: %d of 24 tasks ran by the time Limit returned", n, ran.Load())
		}
		if got := peak.Load(); got != int32(n) {
			t.Errorf("n=%d: peak concurrency %d, want exactly %d", n, got, n)
		}
	}
}

func TestLimitNonPositiveMeansOne(t *testing.T) {
	var inFlight, peak, ran atomic.Int32
	Limit(0, makeTasks(6, &inFlight, &peak, &ran))
	if ran.Load() != 6 || peak.Load() != 1 {
		t.Errorf("ran %d tasks with peak %d; want 6 with peak 1", ran.Load(), peak.Load())
	}
}

func TestLimitNoTasks(t *testing.T) {
	Limit(4, nil) // must simply return
}
