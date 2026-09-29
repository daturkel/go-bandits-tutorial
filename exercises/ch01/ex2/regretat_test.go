package ex2

import "testing"

func TestRegretAt(t *testing.T) {
	curve := []float64{0.5, 1.0, 1.0, 2.5}

	got, ok := RegretAt(curve, []int{1, 4, 2})
	if !ok {
		t.Fatal("ok = false, want true")
	}
	want := map[int]float64{1: 0.5, 4: 2.5, 2: 1.0}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("mark %d = %v, want %v", k, got[k], v)
		}
	}

	for _, bad := range [][]int{{0}, {5}, {2, -1}, {1, 99}} {
		if m, ok := RegretAt(curve, bad); ok || m != nil {
			t.Errorf("marks %v: got (%v, %v), want (nil, false)", bad, m, ok)
		}
	}

	if m, ok := RegretAt(curve, nil); !ok || len(m) != 0 {
		t.Errorf("no marks: got (%v, %v), want empty map and true", m, ok)
	}
}
