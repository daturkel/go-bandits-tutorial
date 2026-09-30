package main

import "fmt"

func main() {
	rewards := []int{1, 0, 0, 1, 1}

	// range over a slice gives the index and the value.
	total := 0
	for i, r := range rewards {
		if r == 1 {
			fmt.Println("win at pull", i)
		}
		total += r
	}
	fmt.Println("total:", total)

	// The classic three-part loop.
	for i := 0; i < 3; i++ {
		fmt.Print(i, " ")
	}
	fmt.Println()

	// range over an integer counts from 0 up to (not including) n.
	for n := range 3 {
		fmt.Print(n, " ")
	}
	fmt.Println()

	// There is no while. A for with only a condition is one.
	n := 1
	for n < 100 {
		n *= 3
	}
	fmt.Println("first power of 3 over 100:", n)

	// An if can start with a short statement; its variable lives only in the if.
	if avg := float64(total) / float64(len(rewards)); avg > 0.5 {
		fmt.Printf("average %.1f: mostly wins\n", avg)
	} else {
		fmt.Printf("average %.1f: mostly losses\n", avg)
	}

	// switch has no fall-through and no break; the first matching case runs.
	switch total {
	case 0:
		fmt.Println("no wins")
	case 1, 2:
		fmt.Println("a few wins")
	default:
		fmt.Println("many wins")
	}

	// A switch with no value is a tidier if / else if chain.
	switch {
	case total > 4:
		fmt.Println("perfect")
	case total > 2:
		fmt.Println("good")
	default:
		fmt.Println("keep going")
	}
}
