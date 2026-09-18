package scenes

import (
	"math"
	"testing"
)

// Where the colours of a card's band sit along its ramp.
func TestARunOfOneColourIsAWholeRamp(t *testing.T) {
	if got := spread(0, 1); got != 0 || math.IsNaN(got) {
		t.Errorf("one colour sits at %v; it is the whole ramp, so it"+
			" sits at nought", got)
	}
	for _, n := range []int{2, 3, 7} {
		if got := spread(0, n); got != 0 {
			t.Errorf("the first of %d sits at %v, want 0", n, got)
		}
		if got := spread(n-1, n); got != 1 {
			t.Errorf("the last of %d sits at %v, want 1", n, got)
		}
		for k := range n {
			if got := spread(k, n); math.IsNaN(got) || got < 0 ||
				got > 1 {
				t.Errorf("the %dth of %d sits at %v", k, n,
					got)
			}
		}
	}
	// and nothing at all does not divide by anything
	if got := spread(0, 0); math.IsNaN(got) {
		t.Error("no colours at all gave a NaN")
	}
}
