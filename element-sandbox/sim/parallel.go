package sim

import (
	"runtime"
	"sync"
)

var workers = max(1, min(runtime.NumCPU(), 8))

// parallelRows splits rows [0,n) into contiguous bands and runs f on each
// band concurrently.
func parallelRows(n int, f func(y0, y1 int)) {
	if workers == 1 {
		f(0, n)
		return
	}
	var wg sync.WaitGroup
	band := (n + workers - 1) / workers
	for y0 := 0; y0 < n; y0 += band {
		wg.Add(1)
		go func(y0, y1 int) {
			defer wg.Done()
			f(y0, y1)
		}(y0, min(y0+band, n))
	}
	wg.Wait()
}

// noise is a cheap per-cell, per-frame random value in [0,1) that is safe
// to use from rendering goroutines.
func noise(i, frame int) float32 {
	x := uint32(i)*0x9E3779B1 ^ uint32(frame)*0x85EBCA77
	x ^= x >> 15
	x *= 0x2C1B3C6D
	x ^= x >> 12
	return float32(x>>8) / (1 << 24)
}
