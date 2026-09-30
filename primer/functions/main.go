package main

import (
	"errors"
	"fmt"
	"strconv"
)

func add(a, b int) int {
	return a + b
}

// divide returns two values. By convention the last is an error, and nil
// means "nothing went wrong".
func divide(a, b float64) (float64, error) {
	if b == 0 {
		return 0, errors.New("division by zero")
	}
	return a / b, nil
}

// parsePercent turns "75" into 75, or explains why it cannot.
func parsePercent(s string) (int, error) {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("parsePercent %q: %w", s, err)
	}
	if n < 0 || n > 100 {
		return 0, fmt.Errorf("parsePercent: %d is not between 0 and 100", n)
	}
	return n, nil
}

func main() {
	fmt.Println(add(2, 3))

	q, err := divide(1, 4)
	fmt.Println(q, err)

	// The caller must decide what to do with a failure. There is no
	// exception to fall through to.
	if _, err := divide(1, 0); err != nil {
		fmt.Println("error:", err)
	}

	for _, s := range []string{"75", "seventy", "150"} {
		n, err := parsePercent(s)
		if err != nil {
			fmt.Println("error:", err)
			continue
		}
		fmt.Println("percent:", n)
	}
}
