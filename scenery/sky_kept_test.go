package scenery

import (
	"testing"

	"github.com/janearc/miami/light"
	"github.com/janearc/miami/view"
)

// A camera that moves without turning sees the sky it already had.
//
// Counted rather than timed. The first version of this decided with a clock --
// a moved camera must not cost much more than a still one -- which passed
// twenty times out of twenty and is still a test that can be wrong on a busy
// machine. keep.Cache knows how many skies it is holding, so one bake and
// then two is the same claim with nothing to be unlucky about. porter's note.
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
	d := &Backdrop{Space: Night()}
	b := light.New(120, 60)
	eye := view.Eye{At: view.Vec{Z: -10}, Lens: 1}
	d.Draw(b, eye, 0)
	if n := d.kept.Len(); n != 1 {
		t.Fatalf("one camera made %d skies", n)
	}
	// moved, a long way and by almost nothing: the same sky each time
	for _, at := range []view.Vec{{X: 900, Y: -400, Z: 12345},
		{X: 1e-9, Z: -10}, {Z: 4000}} {
		moved := eye
		moved.At = at
		d.Draw(b, moved, 0)
		if n := d.kept.Len(); n != 1 {
			t.Errorf("a camera at %v made a second sky: %d kept",
				at, n)
		}
	}
	// turning is a different sky, and has to be worked out
	turned := eye
	turned.Yaw = 0.3
	d.Draw(b, turned, 0)
	if n := d.kept.Len(); n != 2 {
		t.Errorf("a camera that turned left %d skies kept, want 2:"+
			" the sky it is looking at is not being worked out", n)
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
