package main

import "fmt"

// A constant is fixed at compile time.
const arms = 3

func main() {
	// var declares a variable. The type comes after the name.
	var pulls int = 10
	var reward float64 = 7.5

	// := declares a variable and infers its type from the value.
	// It only works inside functions.
	name := "epsilon-greedy"
	explore := true

	// A variable you do not set starts at its type's zero value.
	var count int
	var label string
	var done bool
	var mean float64

	fmt.Println(pulls, reward, name, explore, arms)
	fmt.Printf("zero values: %d %q %t %v\n", count, label, done, mean)

	// Go never converts between numeric types for you. You say so.
	average := reward / float64(pulls)
	fmt.Printf("average reward: %.2f\n", average)
	// reward / pulls would not compile: mismatched types float64 and int.

	// Integer division truncates; % is the remainder.
	fmt.Println(7/2, 7.0/2, 7%2)
}
