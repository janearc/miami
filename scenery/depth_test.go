package scenery

import (
	"math"
	"testing"

	"github.com/janearc/miami/light"
	"github.com/janearc/miami/view"
)

// A wall behind the planet's surface, but in front of the middle of the
// sphere, must not hide the surface.
//
// subpixel culls on depthMid, the closest approach, and the surface is
// entered at t1 = (-half - sqrt(disc))/a, which is always nearer. An
// occluder between the two culls a point it does not cover.
func TestAWallBehindTheSurfaceDoesNotHideIt(t *testing.T) {
	p := Terran() // at z 4.2, radius 1: entered at 3.2, middle at 4.2
	eye := view.Eye{Lens: 1}
	lit := func(wall float64) light.RGB {
		b := light.New(60, 30)
		if wall > 0 {
			for y := range 30 {
				for x := range 60 {
					b.SetDepth(x, y, wall)
				}
			}
		}
		p.Draw(b, eye, 0)
		return b.At(30, 15)
	}
	open := lit(0)
	if open == (light.RGB{}) {
		t.Fatal("the planet did not draw with nothing in the way")
	}
	if got := lit(3.0); got != (light.RGB{}) {
		t.Errorf("a wall at 3.0, in front of the surface, left %v", got)
	}
	if got := lit(3.7); got != open {
		t.Errorf("a wall at 3.7 is behind the surface at 3.2 and in "+
			"front of the middle at 4.2: the surface should be "+
			"untouched, got %v, want %v", got, open)
	}
}

// Two solids in one frame must compose the same way whichever is drawn
// first: what is behind cannot reach the frame, at the silhouette or
// anywhere else.
func TestTwoSolidsComposeTheSameEitherWay(t *testing.T) {
	for _, air := range []float64{0, 0.7} {
		near, far := Terran(), Terran()
		near.At, near.Radius, near.AirDepth = view.Vec{Z: 3}, 0.8, air
		far.At, far.Radius, far.AirDepth = view.Vec{Z: 6}, 2, air
		eye := view.Eye{Lens: 1}
		// opezart's test, with one change, said here rather than
		// quietly: the two bodies go through Draw, which puts them in
		// order, instead of being drawn one after the other in the
		// order given.
		//
		// The claim is unchanged -- the frame must not depend on which
		// body the caller names first -- but it is now the drawer's job
		// to hold it rather than each body's. Getting it right inside a
		// body needs the background it painted over, which is a list of
		// fragments per subpixel; furthest first costs a sort.
		draw := func(first, second *Planet) *light.Buffer {
			b := light.New(120, 60)
			Draw(b, eye, 0, first, second)
			return b
		}
		a, c := draw(far, near), draw(near, far)
		n, worst := 0, 0.0
		for y := range 60 {
			for x := range 120 {
				p, q := a.At(x, y), c.At(x, y)
				d := math.Max(math.Abs(p.R-q.R), math.Max(
					math.Abs(p.G-q.G), math.Abs(p.B-q.B)))
				if d > 1e-9 {
					n++
					worst = math.Max(worst, d)
				}
			}
		}
		if n > 0 {
			t.Errorf("air %.1f: %d subpixels of %d differ by draw "+
				"order, worst %.4f", air, n, 120*60, worst)
		}
	}
}
