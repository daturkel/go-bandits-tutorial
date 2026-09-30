package ex3

import (
	"sync"
	"testing"
)

func TestPickerRotates(t *testing.T) {
	p := NewPicker([]string{"a", "b", "c"})
	var got string
	for range 7 {
		got += p.Next()
	}
	if got != "abcabca" {
		t.Errorf("sequence = %q, want abcabca", got)
	}
}

func TestPickerPanicsOnEmpty(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("NewPicker(nil) should panic")
		}
	}()
	NewPicker(nil)
}

func TestPickerIsSafeForConcurrentUse(t *testing.T) {
	urls := []string{"a", "b", "c", "d"}
	p := NewPicker(urls)
	var mu sync.Mutex
	counts := map[string]int{}
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			local := map[string]int{}
			for range 1000 {
				local[p.Next()]++
			}
			mu.Lock()
			defer mu.Unlock()
			for k, v := range local {
				counts[k] += v
			}
		}()
	}
	wg.Wait()
	for _, u := range urls {
		if counts[u] != 2000 {
			t.Errorf("%s returned %d times, want 2000 (counts: %v)", u, counts[u], counts)
		}
	}
}
