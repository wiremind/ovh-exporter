// Package parallel runs a function over a slice with bounded concurrency.
package parallel

import "sync"

// Map runs fn on every item, at most limit at a time, and returns the results
// in input order. A limit below 1 runs every item at once.
func Map[T, R any](items []T, limit int, fn func(T) R) []R {
	if limit < 1 {
		limit = max(len(items), 1)
	}

	out := make([]R, len(items))
	slots := make(chan struct{}, limit)

	var wg sync.WaitGroup

	for i, item := range items {
		slots <- struct{}{}

		wg.Go(func() {
			defer func() { <-slots }()

			out[i] = fn(item)
		})
	}

	wg.Wait()

	return out
}
