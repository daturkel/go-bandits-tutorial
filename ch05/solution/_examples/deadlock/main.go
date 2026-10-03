package main

import "fmt"

func main() {
	ch := make(chan int) // unbuffered
	ch <- 1              // nobody is receiving, so this waits forever
	fmt.Println(<-ch)
}
