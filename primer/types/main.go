package main

import "fmt"

// Arm counts pulls and keeps their total reward. A struct groups related data.
type Arm struct {
	Name  string
	pulls int // lowercase: only code in this package can touch it
	total float64
}

// NewArm is an ordinary function. By convention, a New function returns a
// pointer to a ready-to-use value.
func NewArm(name string) *Arm {
	return &Arm{Name: name}
}

// A method is a function with a receiver. With a pointer receiver (*Arm) the
// method works on the caller's value and can change it.
func (a *Arm) Record(reward float64) {
	a.pulls++
	a.total += reward
}

func (a *Arm) Mean() float64 {
	if a.pulls == 0 {
		return 0
	}
	return a.total / float64(a.pulls)
}

func main() {
	arm := NewArm("left")
	arm.Record(1)
	arm.Record(0)
	arm.Record(1)
	fmt.Printf("%s: %d pulls, mean %.2f\n", arm.Name, arm.pulls, arm.Mean())

	// Assigning a struct copies it. Assigning a pointer copies the pointer,
	// so both names refer to the same struct.
	copyOfArm := *arm
	samePtr := arm
	arm.Record(1)
	fmt.Println("original:", arm.pulls, "copy:", copyOfArm.pulls, "same struct:", samePtr.pulls)

	// A struct literal names its fields; the ones left out are zero.
	right := Arm{Name: "right"}
	fmt.Printf("%+v\n", right)
}
