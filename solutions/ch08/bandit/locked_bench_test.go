package bandit

import (
	"sync"
	"testing"
)

// rwLocked is Locked with a RWMutex, so snapshots can overlap each other.
type rwLocked struct {
	mu sync.RWMutex
	p  SnapshotPolicy
}

func (l *rwLocked) Update(arm int, reward float64) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.p.Update(arm, reward)
}

func (l *rwLocked) Snapshot() []ArmStat {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.p.Snapshot()
}

type updateSnapshotter interface {
	Update(arm int, reward float64)
	Snapshot() []ArmStat
}

func benchArms(b *testing.B) SnapshotPolicy {
	b.Helper()
	pol, err := NewUCB1(8)
	if err != nil {
		b.Fatal(err)
	}
	return pol
}

// BenchmarkSelectUpdateParallel: every call writes, so a plain Mutex is the
// right tool and the contention is the point.
func BenchmarkLockedSelectUpdate(b *testing.B) {
	l := NewLocked(benchArms(b))
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			l.Update(l.Select(), 1)
		}
	})
}

// BenchmarkSnapshotHeavy: 19 snapshot reads for every update, the shape of a
// service that is polled by monitoring far more often than it is updated.
func BenchmarkSnapshotHeavy(b *testing.B) {
	impls := map[string]func() updateSnapshotter{
		"Mutex":   func() updateSnapshotter { return NewLocked(benchArms(b)) },
		"RWMutex": func() updateSnapshotter { return &rwLocked{p: benchArms(b)} },
	}
	for _, name := range []string{"Mutex", "RWMutex"} {
		b.Run(name, func(b *testing.B) {
			l := impls[name]()
			b.RunParallel(func(pb *testing.PB) {
				i := 0
				for pb.Next() {
					i++
					if i%20 == 0 {
						l.Update(i%8, 1)
					} else {
						l.Snapshot()
					}
				}
			})
		})
	}
}
