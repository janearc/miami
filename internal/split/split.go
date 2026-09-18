// Package split does rows of a picture on every core.
//
// A frame is made of rows that do not look at each other: a subpixel's colour
// is a function of where it is, and a cell is fitted from its own six
// subpixels. Work like that parallelises by cutting it into bands and handing
// the bands out, and that is all this does.
//
// The bands are small, several per core, and taken from a shared counter. One
// band per core is the obvious cut and the wrong one.
//
// A black hole's rows do not cost the same -- the rows through the disk
// integrate further than the rows of empty sky -- and the core that drew the
// disk band would still be working while the rest sat idle.
package split

import (
	"runtime"
	"sync"
	"sync/atomic"
)

// Rows calls do over [lo, hi) bands that between them cover [0, n) exactly
// once, on as many goroutines as there are cores, and returns when every band
// is done.
//
// do MUST NOT write anywhere another band reads. Each band writing its own rows
// of an output allocated before the call is the intended use.
func Rows(n int, do func(lo, hi int)) {
	workers := runtime.GOMAXPROCS(0)
	if n <= 0 {
		return
	}
	if workers == 1 || n < 2 {
		do(0, n)
		return
	}
	band := max(1, n/(workers*4))
	var next atomic.Int64
	var wg sync.WaitGroup
	for range min(workers, n) {
		wg.Go(func() {
			for {
				lo := int(next.Add(int64(band))) - band
				if lo >= n {
					return
				}
				do(lo, min(lo+band, n))
			}
		})
	}
	wg.Wait()
}
