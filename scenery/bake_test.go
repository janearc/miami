package scenery

import (
	"fmt"
	"testing"
	"time"

	"github.com/janearc/miami/light"
	"github.com/janearc/miami/view"
)

// What a moved eye costs.
//
// A black hole's rays are baked for an eye, and the eye is part of the key, so
// a camera that has not moved draws from the bake it made last frame and one
// that has moved pays for a new one. Every sample of a drag is a moved eye,
// which is why this number decides what a drag may do rather than being
// interesting on its own.

// sizes are a full screen and a third of one, as subpixels: a cell is two
// across and three down, so 240 by 68 cells is 480 by 204.
var sizes = []struct {
	what string
	w, h int
}{
	{"full 240x68 cells", 480, 204},
	{"third 80x22 cells", 160, 66},
}

// TestWhatAMovedEyeCosts prints what a resting eye and a moved one cost at
// each size. It asserts only the thing that must be true -- that a resting
// eye is much cheaper, which is the cache working -- and prints the rest,
// because the numbers belong to the machine and not to the test.
func TestWhatAMovedEyeCosts(t *testing.T) {
	if testing.Short() {
		t.Skip("timing")
	}
	for _, s := range sizes {
		h := Quasar()
		b := light.New(s.w, s.h)
		eye := view.Eye{At: view.Vec{Z: -14}, Lens: 1}
		h.Draw(b, eye, 0) // the first bake, which is not timed
		resting := timed(func() { h.Draw(b, eye, 0) })
		moved := timed(func() {
			// a drag's eye: a little further round, never seen before
			e := eye
			e.At.X += 0.017
			h.Draw(b, e, 0)
		})
		fmt.Printf("%-20s resting %7.1fms   moved %7.1fms\n", s.what,
			ms(resting), ms(moved))
		if resting >= moved {
			t.Errorf("%s: a resting eye cost %v and a moved one %v;"+
				" the bake is not being kept", s.what, resting,
				moved)
		}
	}
}

// timed is how long a thing took.
func timed(do func()) time.Duration {
	start := time.Now()
	do()
	return time.Since(start)
}

// ms is a duration in milliseconds.
func ms(d time.Duration) float64 {
	return float64(d) / float64(time.Millisecond)
}

// A drag must not cost the camera that is standing still.
//
// Two viewports of one hole share that hole's bakes, and there are four
// slots. A drag makes a new eye every sample, so the question porter's brief
// asks is whether those push the resting camera's bake out: if they do, the
// still picture is re-baked every frame for the whole drag, and the frame
// costs twice what the drag alone would.
//
// It survives because the slots are least-recently-used and the resting eye is
// used every frame: what the drag evicts is its own older samples.
func TestADragDoesNotEvictTheCameraStandingStill(t *testing.T) {
	if testing.Short() {
		t.Skip("timing")
	}
	h := Quasar()
	b := light.New(320, 136)
	resting := view.Eye{At: view.Vec{Z: -14}, Lens: 1}
	dragged := view.Eye{At: view.Vec{X: 3, Z: -14}, Lens: 1}
	h.Draw(b, resting, 0) // both baked once, and not timed
	h.Draw(b, dragged, 0)
	first := timed(func() { h.Draw(b, resting, 0) })
	worst := time.Duration(0)
	for i := range 12 { // a drag: a new eye every frame
		e := dragged
		e.At.X += 0.03 * float64(i+1)
		h.Draw(b, e, 0)
		if d := timed(func() { h.Draw(b, resting, 0) }); d > worst {
			worst = d
		}
	}
	fmt.Printf("resting camera during a drag: %.1fms, and %.1fms before "+
		"the drag began\n", ms(worst), ms(first))
	if worst > 8*first+time.Millisecond {
		t.Errorf("the resting camera cost %v during the drag and %v"+
			" before it: its bake is being evicted", worst, first)
	}
}
