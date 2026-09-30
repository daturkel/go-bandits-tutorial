package ex2

import (
	"testing"
	"time"
)

type clock struct{ t time.Time }

func (c *clock) Now() time.Time { return c.t }

func newClock() *clock { return &clock{t: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)} }

func TestGetBeforeAndAfterDeadline(t *testing.T) {
	c := newClock()
	e := NewExpiring[string, int](c.Now)
	e.Set("a", 42, time.Minute)

	if v, ok := e.Get("a"); !ok || v != 42 {
		t.Fatalf("Get = (%d, %v), want (42, true)", v, ok)
	}
	c.t = c.t.Add(59 * time.Second)
	if _, ok := e.Get("a"); !ok {
		t.Error("expired too early")
	}
	c.t = c.t.Add(time.Second) // exactly at the deadline
	if v, ok := e.Get("a"); ok || v != 0 {
		t.Errorf("at the deadline Get = (%d, %v), want (0, false)", v, ok)
	}
}

func TestMissingKey(t *testing.T) {
	e := NewExpiring[string, string](newClock().Now)
	if v, ok := e.Get("nope"); ok || v != "" {
		t.Errorf("Get(nope) = (%q, %v)", v, ok)
	}
}

func TestSetReplacesValueAndDeadline(t *testing.T) {
	c := newClock()
	e := NewExpiring[string, int](c.Now)
	e.Set("a", 1, time.Minute)
	c.t = c.t.Add(50 * time.Second)
	e.Set("a", 2, time.Minute) // fresh deadline: 60s from now
	c.t = c.t.Add(50 * time.Second)
	if v, ok := e.Get("a"); !ok || v != 2 {
		t.Errorf("Get = (%d, %v), want (2, true)", v, ok)
	}
	if e.Len() != 1 {
		t.Errorf("Len = %d, want 1", e.Len())
	}
}

func TestSweepAndLen(t *testing.T) {
	c := newClock()
	e := NewExpiring[int, string](c.Now)
	e.Set(1, "short", 10*time.Second)
	e.Set(2, "short", 20*time.Second)
	e.Set(3, "long", time.Hour)

	c.t = c.t.Add(30 * time.Second)
	if e.Len() != 3 {
		t.Errorf("Len before sweep = %d, want 3 (expired entries linger until swept)", e.Len())
	}
	if n := e.Sweep(); n != 2 {
		t.Errorf("Sweep = %d, want 2", n)
	}
	if e.Len() != 1 {
		t.Errorf("Len after sweep = %d, want 1", e.Len())
	}
	if n := e.Sweep(); n != 0 {
		t.Errorf("second Sweep = %d, want 0", n)
	}
	if v, ok := e.Get(3); !ok || v != "long" {
		t.Errorf("the live entry was lost: (%q, %v)", v, ok)
	}
}
