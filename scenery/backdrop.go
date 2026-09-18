package scenery

import (
	"sync"

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

	mu    sync.Mutex
	key   backdropKey
	skies []Sky
}

// backdropKey is what a kept sky depends on.
type backdropKey struct {
	w, h int
	eye  view.Eye
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
	d.skies = nil
	d.mu.Unlock()
}

// skiesFor is the sky in every subpixel's direction, kept for the size and
// the camera.
func (d *Backdrop) skiesFor(w, h int, eye view.Eye) []Sky {
	d.mu.Lock()
	defer d.mu.Unlock()
	key := backdropKey{w: w, h: h, eye: eye}
	if d.skies != nil && d.key == key {
		return d.skies
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
	d.key, d.skies = key, skies
	return skies
}
