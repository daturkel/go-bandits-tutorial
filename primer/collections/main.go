package main

import (
	"fmt"
	"maps"
	"slices"
)

func main() {
	// A slice is a window onto an array. Literals, append and len are the basics.
	a := []int{10, 20, 30}
	a = append(a, 40)
	fmt.Println(a, len(a))

	// make creates a slice of a given length, filled with zero values.
	counts := make([]int, 4)
	counts[2]++
	fmt.Println(counts)

	// Slicing does not copy: b and a share the same elements.
	b := a[:2]
	b[0] = 99
	fmt.Println("a:", a, "b:", b)

	// To get an independent copy, clone.
	c := slices.Clone(a)
	c[0] = -1
	fmt.Println("a:", a, "c:", c)

	// A map is a dictionary. Its zero value for a missing key is the value
	// type's zero value, so counting works without setting anything first.
	wins := map[string]int{"ucb1": 3}
	wins["thompson"]++
	wins["thompson"]++
	fmt.Println(wins["ucb1"], wins["thompson"], wins["nobody"])

	// The comma-ok form tells "missing" apart from "present with value 0".
	if n, ok := wins["nobody"]; !ok {
		fmt.Println("nobody is not in the map:", n, ok)
	}

	// Iteration order of a map is random on purpose, so sort the keys first.
	for _, k := range slices.Sorted(maps.Keys(wins)) {
		fmt.Println(k, wins[k])
	}
	delete(wins, "ucb1")
	fmt.Println(len(wins))
}
