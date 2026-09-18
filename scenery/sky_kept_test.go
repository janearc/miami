package scenery

import (
	"testing"
	"time"

	"github.com/janearc/miami/light"
	"github.com/janearc/miami/view"
)

// A camera that moves without turning sees the sky it already had.
//
// The sky is a function of direction: where the eye is cannot change it, and
// a camera drifting through space is drawing the same stars. The whole Eye
// used to be the key, so a drift of a nanometre threw the bake away and paid
// for it again -- 0.04ms became 1.92ms, every frame, for a picture that did
// not change by one subpixel.
//
// opezart found it by suspecting his own corridor's drift was doing exactly
// this, and asking rather than working around it.
func TestASkyIsKeptWhileTheCameraOnlyMoves(t *testing.T) {
	if testing.Short() {
		t.Skip("timing")
	}
	d := &Backdrop{Space: Night()}
	b := light.New(200, 90)
	eye := view.Eye{At: view.Vec{Z: -10}, Lens: 1}
	d.Draw(b, eye, 0) // the bake, untimed

	took := func(e view.Eye) time.Duration {
		start := time.Now()
		d.Draw(b, e, 0)
		return time.Since(start)
	}
	kept := took(eye)

	// moved, a long way, and moved by almost nothing
	for _, at := range []view.Vec{{X: 900, Y: -400, Z: 12345},
		{X: 1e-9, Z: -10}} {
		moved := eye
		moved.At = at
		if d := took(moved); d > 8*kept+time.Millisecond {
			t.Errorf("a camera at %v paid %v for a sky it "+
				"already had; standing still costs %v", at, d,
				kept)
		}
	}

	// and turning is a different sky, which must be worked out
	turned := eye
	turned.Yaw = 0.3
	if d := took(turned); d <= kept {
		t.Errorf("a camera that turned cost %v and standing still "+
			"costs %v: the sky it is looking at is not being "+
			"worked out", d, kept)
	}
}

// The sky a camera sees is the sky it draws: a moved camera and a still one
// draw the same subpixels, which is why the one above may be kept for the
// other.
func TestAMovedCameraDrawsTheSameSky(t *testing.T) {
	sp := Night()
	draw := func(at view.Vec) *light.Buffer {
		b := light.New(90, 45)
		(&Backdrop{Space: sp}).Draw(b,
			view.Eye{At: at, Lens: 1}, 0)
		return b
	}
	here, far := draw(view.Vec{Z: -10}), draw(view.Vec{X: 900, Z: 12345})
	for y := range 45 {
		for x := range 90 {
			if here.At(x, y) != far.At(x, y) {
				t.Fatalf("at %d,%d the sky differs by where "+
					"the camera stands: %v against %v",
					x, y, here.At(x, y), far.At(x, y))
			}
		}
	}
}
