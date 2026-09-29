package ex2

import (
	"slices"
	"testing"
)

type fake struct{ selects, updates int }

func (f *fake) Name() string        { return "fake" }
func (f *fake) Select() int         { f.selects++; return 2 }
func (f *fake) Update(int, float64) { f.updates++ }

func TestLoggedRecordsAndDelegates(t *testing.T) {
	inner := &fake{}
	var p Policy = NewLogged(inner) // *Logged must satisfy Policy

	if p.Name() != "fake" {
		t.Errorf("Name() = %q, want the wrapped policy's name", p.Name())
	}
	if got := p.Select(); got != 2 {
		t.Errorf("Select() = %d, want 2 from the wrapped policy", got)
	}
	p.Update(1, 1)
	p.Update(0, 0.5)

	if inner.selects != 1 || inner.updates != 2 {
		t.Errorf("wrapped policy saw %d selects and %d updates, want 1 and 2", inner.selects, inner.updates)
	}
	want := []string{"select 2", "update 1 1", "update 0 0.5"}
	if got := p.(*Logged).Log; !slices.Equal(got, want) {
		t.Errorf("Log = %q, want %q", got, want)
	}
}
