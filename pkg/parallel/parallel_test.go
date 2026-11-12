package parallel

import (
	"slices"
	"sync/atomic"
	"testing"
	"time"
)

func TestMapKeepsInputOrder(t *testing.T) {
	got := Map([]int{3, 1, 2}, 2, func(n int) int {
		time.Sleep(time.Duration(n) * time.Millisecond)

		return n * 10
	})

	if want := []int{30, 10, 20}; !slices.Equal(got, want) {
		t.Fatalf("Map() = %v, want %v", got, want)
	}
}

func TestMapRespectsLimit(t *testing.T) {
	cases := map[string]struct {
		limit int
		want  int64
	}{
		"limit 2":              {limit: 2, want: 2},
		"limit 0 runs all":     {limit: 0, want: 6},
		"limit above len runs": {limit: 50, want: 6},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var running, peak atomic.Int64

			Map(make([]struct{}, 6), tc.limit, func(struct{}) struct{} {
				now := running.Add(1)
				for {
					old := peak.Load()
					if now <= old || peak.CompareAndSwap(old, now) {
						break
					}
				}
				time.Sleep(20 * time.Millisecond)
				running.Add(-1)

				return struct{}{}
			})

			if got := peak.Load(); got != tc.want {
				t.Fatalf("peak concurrency = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestMapEmpty(t *testing.T) {
	if got := Map(nil, 0, func(n int) int { return n }); len(got) != 0 {
		t.Fatalf("Map(nil) = %v, want empty", got)
	}
}
