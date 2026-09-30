package main

import (
	"context"
	"fmt"
	"time"
)

func main() {
	// 1. Wait for a value, but not forever.
	slow := make(chan int)
	select {
	case v := <-slow:
		fmt.Println("got", v)
	case <-time.After(30 * time.Millisecond):
		fmt.Println("1. timed out waiting for a value")
	}

	// 2. A default case makes select non-blocking.
	ready := make(chan int, 1)
	select {
	case v := <-ready:
		fmt.Println("got", v)
	default:
		fmt.Println("2. nothing ready, moved on")
	}
	ready <- 7
	select {
	case v := <-ready:
		fmt.Println("2. now ready:", v)
	default:
		fmt.Println("nothing ready")
	}

	// 3. Do periodic work until a deadline: two channels in one select.
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	deadline := time.After(110 * time.Millisecond)
	ticks := 0
loop:
	for {
		select {
		case <-ticker.C:
			ticks++
		case <-deadline:
			break loop
		}
	}
	fmt.Println("3. ticked at least 4 times:", ticks >= 4)

	// 4. A context bundles "stop now" into one channel: ctx.Done().
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	select {
	case <-ctx.Done():
		fmt.Println("4. context ended:", ctx.Err())
	case <-time.After(time.Second):
		fmt.Println("not reached")
	}

	// 5. Cancelling by hand, and recording why.
	ctx2, cancel2 := context.WithCancelCause(context.Background())
	cancel2(fmt.Errorf("operator pressed stop"))
	<-ctx2.Done()
	fmt.Println("5.", ctx2.Err(), "/ cause:", context.Cause(ctx2))
}
