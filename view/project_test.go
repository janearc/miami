package view

import (
	"math"
	"time"

	sprite "github.com/janearc/libdaffy-sprite"
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

// square is a sprite of one role, for asking where it landed.
type square struct {
	at     Vec
	height float64
}

func (s square) Extent(time.Duration) (Vec, float64) { return s.at, s.height }

func (s square) Render(_ Eye, _ time.Duration, rows int) sprite.Frame {
	art := make([]string, max(rows, 1))
	for i := range art {
		art[i] = "WWWW"
	}
	return sprite.Frame{Art: art}
}

// Resolve puts a sprite's middle where Project says its place is, under a
// turned camera and at several lenses.
//
// It used to work this out itself, from the eye's position but not its yaw or
// pitch, dividing by Lens where the tracer multiplies. So a sprite and a
// traced ball at the same point were drawn in different places, which is only
// invisible while the camera is at the origin looking straight ahead with a
// lens of one.
func TestResolveAgreesWithProject(t *testing.T) {
	const w, h = 200, 90
	for _, e := range []Eye{
		{Lens: 1, Near: 0.05},
		{At: Vec{0.4, -0.3, 1}, Lens: 0.6, Near: 0.05, Yaw: 0.3,
			Pitch: -0.15},
		{Lens: 1.8, Near: 0.05, Pitch: 0.2},
	} {
		for _, at := range []Vec{{Z: 6}, {X: 1.2, Y: -0.6, Z: 9},
			{X: -2, Y: 1, Z: 15}} {
			x, y, _, ok := e.Project(at, w, h)
			if !ok || x < 20 || y < 20 || x > w-20 || y > h-20 {
				continue // off the sheet: Resolve drops it, rightly
			}
			got := Resolve([]Object{square{at: at, height: 1}}, e, 0,
				w, h)
			if len(got) != 1 {
				t.Fatalf("%+v at %+v: dropped", e, at)
			}
			p := got[0]
			mx := float64(p.X) + float64(p.W)/2
			my := float64(p.Y) + float64(p.H)/2
			if math.Abs(mx-x) > 1.5 || math.Abs(my-y) > 1.5 {
				t.Errorf("%+v at %+v: middle %.1f,%.1f, "+
					"projected %.1f,%.1f", e, at, mx, my, x, y)
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
