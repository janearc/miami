package split

import (
	"runtime"
	"sync/atomic"
	"testing"
)

// Every row is done exactly once, whatever the count and however many cores.
func TestEveryRowOnce(t *testing.T) {
	was := runtime.GOMAXPROCS(0)
	defer runtime.GOMAXPROCS(was)
	for _, procs := range []int{1, 2, 3, 16} {
		runtime.GOMAXPROCS(procs)
		for _, n := range []int{0, 1, 2, 7, 64, 65, 1000} {
			seen := make([]atomic.Int32, n)
			Rows(n, func(lo, hi int) {
				if lo < 0 || hi > n || lo >= hi {
					t.Errorf("band [%d,%d) of %d", lo, hi, n)
					return
				}
				for i := lo; i < hi; i++ {
					seen[i].Add(1)
				}
			})
			for i := range seen {
				if got := seen[i].Load(); got != 1 {
					t.Fatalf("procs %d n %d: row %d done %d "+
						"times", procs, n, i, got)
				}
			}
		}
	}
}
