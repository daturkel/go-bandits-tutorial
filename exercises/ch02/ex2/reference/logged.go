package ex2

import "fmt"

// Policy is the interface from the chapter.
type Policy interface {
	Name() string
	Select() int
	Update(arm int, reward float64)
}

// Logged wraps a Policy and records every call in Log, as "select 2" and
// "update 1 1" (arm, then reward printed with %g). Everything else is
// delegated to the wrapped policy unchanged.
type Logged struct {
	Policy // embedded: Name, Select and Update are promoted from it
	Log    []string
}

// NewLogged wraps p.
func NewLogged(p Policy) *Logged { return &Logged{Policy: p} }

// Select records the chosen arm.
func (l *Logged) Select() int {
	arm := l.Policy.Select()
	l.Log = append(l.Log, fmt.Sprintf("select %d", arm))
	return arm
}

// Update records the observation, then passes it on.
func (l *Logged) Update(arm int, reward float64) {
	l.Log = append(l.Log, fmt.Sprintf("update %d %g", arm, reward))
	l.Policy.Update(arm, reward)
}
