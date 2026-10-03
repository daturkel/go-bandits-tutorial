package extras

import "sync"

// Limit runs every task, with at most n running at the same time, and returns
// when all of them have finished. If n < 1, treat it as 1.
//
// Idiom: a buffered channel of capacity n works as a counting semaphore.
// Send to acquire a slot (blocks when n are in use), receive to release it.
func Limit(n int, tasks []func()) {
	n = max(n, 1)
	sem := make(chan struct{}, n)
	var wg sync.WaitGroup
	for _, task := range tasks {
		sem <- struct{}{} // acquire: waits here while n tasks are running
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }() // release
			task()
		}()
	}
	wg.Wait()
}
