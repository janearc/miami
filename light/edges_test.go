package light

import (
	"math"
	"testing"
)

// Nothing that reaches past the frame may panic, and nothing may land on a
// cell that is not the one asked for.
//
// Add clipped from the start, and said so: "a spark half off the edge needs
// no clipping by whoever drew it". Nothing else did, so a caller who read that
// and believed it got a panic from Depth and a silent wrong cell from Set.
// opezart hit the panic writing a note's depth at an odd terminal size, in a
// program where every point comes out of a projection.

// off is a spread of points that are not on a 4 by 3 frame, including the
// ones whose index lands inside the slice anyway.
var off = []struct{ x, y int }{
	{-1, 0}, {0, -1}, {-1, -1}, {4, 0}, {0, 3}, {4, 3},
	{-1, 2},    // on row two, an index of 3,1: valid, and the wrong cell
	{5, 1},     // on row one, an index of 1,2: valid, and the wrong cell
	{99, 99},   // far outside, an index past the end
	{-99, -99}, // far outside, a negative index
}

func TestNothingOffTheFrameTouchesTheFrame(t *testing.T) {
	for _, p := range off {
		b := New(4, 3)
		// a frame with something in every cell, so a stray write shows
		for y := range 3 {
			for x := range 4 {
				b.Set(x, y, RGB{R: 1, G: 1, B: 1})
				b.SetDepth(x, y, 7)
			}
		}
		b.Set(p.x, p.y, RGB{})
		b.Add(p.x, p.y, RGB{R: 5})
		b.Over(p.x, p.y, RGB{}, 1)
		b.SetDepth(p.x, p.y, 99)
		for y := range 3 {
			for x := range 4 {
				if got := b.At(x, y); got != (RGB{R: 1, G: 1,
					B: 1}) {
					t.Errorf("drawing at %d,%d, which is "+
						"off a 4x3 frame, changed "+
						"the light at %d,%d to %v",
						p.x, p.y, x, y, got)
				}
				if got := b.Depth(x, y); got != 7 {
					t.Errorf("drawing at %d,%d changed "+
						"the depth at %d,%d to %v",
						p.x, p.y, x, y, got)
				}
			}
		}
	}
}

// And reading off the frame answers rather than panicking: no light, and
// nothing in the way.
func TestReadingOffTheFrameAnswers(t *testing.T) {
	b := New(4, 3)
	for y := range 3 {
		for x := range 4 {
			b.Set(x, y, RGB{R: 1})
			b.SetDepth(x, y, 7)
		}
	}
	for _, p := range off {
		if got := b.At(p.x, p.y); got != (RGB{}) {
			t.Errorf("the light at %d,%d, off the frame, is %v",
				p.x, p.y, got)
		}
		if got := b.Depth(p.x, p.y); !math.IsInf(got, 1) {
			t.Errorf("the depth at %d,%d, off the frame, is %v;"+
				" nothing is there, so nothing is in the way",
				p.x, p.y, got)
		}
	}
}
