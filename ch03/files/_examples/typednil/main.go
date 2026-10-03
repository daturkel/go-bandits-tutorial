package main

import (
	"errors"
	"fmt"
)

type Policy interface{ Select() int }

type Greedy struct{}

func (*Greedy) Select() int { return 0 }

func newGreedy(ok bool) (*Greedy, error) {
	if !ok {
		return nil, errors.New("bad configuration")
	}
	return &Greedy{}, nil
}

// direct passes the constructor's results straight through.
func direct(ok bool) (Policy, error) { return newGreedy(ok) }

// careful returns a true nil on failure.
func careful(ok bool) (Policy, error) {
	g, err := newGreedy(ok)
	if err != nil {
		return nil, err
	}
	return g, nil
}

func main() {
	p, err := direct(false)
	fmt.Printf("direct:  err=%v  p == nil: %-5v  p is %T\n", err, p == nil, p)
	p, err = careful(false)
	fmt.Printf("careful: err=%v  p == nil: %-5v  p is %T\n", err, p == nil, p)
}
