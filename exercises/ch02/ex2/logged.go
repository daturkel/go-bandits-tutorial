package ex2

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

// TODO: override Select and Update so they record to Log and then call the
// embedded policy. Name needs no code: it is promoted.
