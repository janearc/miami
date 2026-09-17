package scenery

import (
	"math"
	"testing"

	"github.com/janearc/miami/view"
)

// aimed is a ray from a camera far up the Z axis that passes the hole at
// impact parameter b, off to the side along +X.
func aimed(r0, b float64) (cam, dir view.Vec) {
	// the static camera's own measure of the angle, from b as the orbit
	// equation defines it
	s := b * math.Sqrt(1-2/r0) / r0
	return view.Vec{Z: r0}, view.Vec{X: s, Z: -math.Sqrt(1 - s*s)}
}

// The shadow's edge is at b = 3 sqrt 3. A ray a hair inside it falls in and a
// hair outside it gets away, from a camera far enough that it is at infinity
// as near as makes no difference.
func TestTheShadowEdgeIsThreeRootThree(t *testing.T) {
	crit := 3 * math.Sqrt(3)
	var out traced
	for _, c := range []struct {
		b    float64
		want fate
	}{{crit - 0.01, captured}, {crit + 0.01, escaped}, {2, captured},
		{9, escaped}} {
		cam, dir := aimed(1e4, c.b)
		trace(cam, dir, view.Vec{Y: 1}, &out)
		if out.fate != c.want {
			t.Errorf("b=%.4f: fate %d, want %d", c.b, out.fate, c.want)
		}
	}
}

// Far from the hole, light bends by 4/b plus a smaller second-order term,
// which is the deflection Eddington measured at the sun.
func TestWeakFieldBendingIsFourOverB(t *testing.T) {
	var out traced
	for _, b := range []float64{100, 400} {
		cam, dir := aimed(1e6, b)
		trace(cam, dir, view.Vec{Y: 1}, &out)
		if out.fate != escaped {
			t.Fatalf("b=%v was caught", b)
		}
		// straight on would be (0,0,-1); the bend turns it back
		// toward the hole's side, which is -X
		bend := math.Atan2(-out.away.X, -out.away.Z) +
			math.Asin(unit(dir).X)
		want := 4/b + 15*math.Pi/4/(b*b)
		if math.Abs(bend-want) > 0.02*want {
			t.Errorf("b=%v: bent %.6f, want %.6f", b, bend, want)
		}
	}
}

// Light from the side of the disk turning toward the camera arrives bluer
// than light from the side turning away. The ray passing on the +X side of
// a disk that turns anticlockwise about +Y sees gas going away from the
// camera, and carries angular momentum -b about the axis.
func TestTheApproachingSideIsBluer(t *testing.T) {
	var out traced
	cam, dir := aimed(1e4, 10)
	trace(cam, dir, view.Vec{Y: 1, Z: 0.3}, &out)
	if math.Abs(out.lambda-(-10)*1/math.Sqrt(1.09)) > 0.01 {
		t.Fatalf("lambda %v, want about %v", out.lambda,
			-10/math.Sqrt(1.09))
	}
	away := redshift(10, -10, 1e4, 1)
	toward := redshift(10, 10, 1e4, 1)
	still := redshift(10, 0, 1e4, 1)
	if !(away < still && still < toward) {
		t.Fatalf("away %v, still %v, toward %v", away, still, toward)
	}
	// with nothing moving along the line of sight what is left is time
	// running slow at r=10: sqrt(1 - 3/10)
	if math.Abs(still-math.Sqrt(0.7)) > 1e-3 {
		t.Fatalf("gravitational redshift at r=10: %v, want %v", still,
			math.Sqrt(0.7))
	}
}

// A ray aimed at a point on the near half of a tilted disk crosses the disk
// very near that point: the ray reaches the disk before it passes the hole,
// and nearly all the bending happens as it passes.
//
// The far half is left out on purpose. A straight line to a point behind the
// hole passes within about ten masses of it and bends by nearly half a
// radian, which is what lifts the back of the disk up over the shadow, and it
// lands nowhere near where it was aimed.
func TestARayAimedAtTheDiskFindsIt(t *testing.T) {
	n := unit(view.Vec{Y: 1, Z: 0.5})
	cam := view.Vec{Y: 0, Z: 300}
	dx, dy := basis(n)
	for _, target := range []float64{25, 40} {
		// the near half, and the two sides, of this disk
		for _, phi := range []float64{1.3, 2.2, 2.9, 3.6, 4.4, 5.0} {
			p := add(scale(dx, target*math.Cos(phi)),
				scale(dy, target*math.Sin(phi)))
			var out traced
			trace(cam, sub(p, cam), n, &out)
			if len(out.cross) == 0 {
				t.Fatalf("r=%v phi=%v: no crossing", target, phi)
			}
			got := out.cross[0]
			if math.Abs(got.r-target) > 0.08*target {
				t.Errorf("r=%v phi=%v: crossed at r=%.3f",
					target, phi, got.r)
			}
			dphi := math.Abs(math.Remainder(got.phi-phi, 2*math.Pi))
			if dphi > 0.08 {
				t.Errorf("r=%v phi=%v: crossed at phi=%.3f",
					target, phi, got.phi)
			}
		}
	}
}
