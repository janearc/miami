package view

import (
	"math"
	"testing"
)

// A point a subpixel's ray passes through projects back onto that subpixel,
// under a turned camera and at any depth. If Ray and Project disagreed, a
// spark drawn next to a traced ball would sit a few subpixels off it.
func TestProjectUndoesRay(t *testing.T) {
	eyes := []Eye{
		{Lens: 1},
		{At: Vec{0.3, -1, 2}, Lens: 0.7, Yaw: 0.4, Pitch: -0.2},
		{At: Vec{-5, 2, 1}, Lens: 1.6, Yaw: -2.1, Pitch: 0.35},
	}
	const w, h = 180, 72
	for _, e := range eyes {
		for _, px := range [][2]int{{0, 0}, {90, 36}, {179, 71}, {13, 60}} {
			d := e.Ray(px[0], px[1], w, h)
			for _, k := range []float64{0.5, 3, 40} {
				p := Vec{e.At.X + d.X*k, e.At.Y + d.Y*k,
					e.At.Z + d.Z*k}
				x, y, depth, ok := e.Project(p, w, h)
				if !ok {
					t.Fatalf("%+v %v: not projected", e, px)
				}
				if math.Abs(x-float64(px[0])) > 1e-9 ||
					math.Abs(y-float64(px[1])) > 1e-9 {
					t.Fatalf("%+v: subpixel %v came back "+
						"at %.6f,%.6f", e, px, x, y)
				}
				if math.Abs(depth-k) > 1e-9 {
					t.Fatalf("depth %v, want %v", depth, k)
				}
			}
		}
	}
}

// The ray tracer and Ray point the same way, so a solid traced by Lit and a
// spark placed by Project line up.
func TestTraceUsesRay(t *testing.T) {
	e := Eye{At: Vec{0, 0.3, 0}, Lens: 1.2, Yaw: 0.3, Pitch: -0.1}
	ball := &Ball{At: Vec{0.4, 0.1, 3}, Radius: 0.2}
	fr := Lit{Eye: e, Solids: []Solid{ball}}.Trace(0, 40, 14)
	w, h := 80, 42
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			_, hit := HitSphere(Ray{At: e.At, Dir: e.Ray(x, y, w, h)},
				ball.At, ball.Radius, 1e-4, math.Inf(1))
			if hit != (fr.Art[y][x] != '.') {
				t.Fatalf("subpixel %d,%d: tracer %q, Ray hit %v",
					x, y, fr.Art[y][x], hit)
			}
		}
	}
}
