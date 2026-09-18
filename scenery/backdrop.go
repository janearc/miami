package scenery

import (
	"math"
	"sync"

	"github.com/janearc/miami/internal/keep"
	"github.com/janearc/miami/internal/split"
	"github.com/janearc/miami/light"
	"github.com/janearc/miami/view"
)

// Backdrop is Space drawn straight, with nothing bending it: the sky behind a
// planet, or behind anything else that is not a black hole.
//
// The sky in each subpixel's direction does not change while the camera
// stays put, apart from the twinkle, so it is worked out once for a size and
// a camera and kept, as a black hole's rays are.
type Backdrop struct {
	Space Space

	mu   sync.Mutex
	kept *keep.Cache[backdropKey, []Sky]
}

// backdropKey is what a kept sky depends on: the size of the frame, and which
// way the eye is pointed.
//
// Not where the eye is. The sky is a function of direction, so a camera that
// moves without turning sees the same sky, and one that moves twelve thousand
// units sees exactly the same sky -- measured, nought subpixels different.
//
// The whole Eye was in here, so any movement at all threw the bake away and
// paid for it again: a still camera cost 0.04ms and one drifting by a
// nanometre cost 1.92ms. opezart found it by suspecting his own drift was
// doing this and asking, which it was.
type backdropKey struct {
	w, h             int
	yaw, pitch, lens float64
}

// skyKey is what a sky is kept under, for a frame this size seen this way.
func skyKey(w, h int, eye view.Eye) backdropKey {
	lens := eye.Lens
	if lens == 0 {
		lens = 1
	}
	return backdropKey{w: w, h: h, yaw: eye.Yaw, pitch: eye.Pitch,
		lens: lens}
}

// Draw fills the frame with the sky, replacing what was there.
func (d *Backdrop) Draw(b *light.Buffer, eye view.Eye, t float64) {
	skies := d.skiesFor(b.W, b.H, eye)
	split.Rows(b.H, func(lo, hi int) {
		for y := lo; y < hi; y++ {
			for x := 0; x < b.W; x++ {
				b.Set(x, y, skies[y*b.W+x].At(t))
			}
		}
	})
}

// Forget drops the kept sky, for a Space changed while it was on screen.
func (d *Backdrop) Forget() {
	d.mu.Lock()
	if d.kept != nil {
		d.kept.Clear()
	}
	d.mu.Unlock()
}

// skiesFor is the sky in every subpixel's direction, kept for the size and
// the camera.
func (d *Backdrop) skiesFor(w, h int, eye view.Eye) []Sky {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.kept == nil {
		d.kept = keep.New[backdropKey, []Sky](keptBakes)
	}
	key := skyKey(w, h, eye)
	if s, ok := d.kept.Get(key); ok {
		return s
	}
	lens := eye.Lens
	if lens == 0 {
		lens = 1
	}
	pixel := 1 / (lens * float64(w))
	skies := make([]Sky, w*h)
	split.Rows(h, func(lo, hi int) {
		for y := lo; y < hi; y++ {
			for x := 0; x < w; x++ {
				skies[y*w+x] = d.Space.Look(eye.Ray(x, y, w, h),
					pixel)
			}
		}
	})
	d.kept.Put(key, skies)
	return skies
}

// Away is how far the sky is: further than anything, since it is the sky.
func (d *Backdrop) Away(view.Eye) float64 { return math.Inf(1) }
