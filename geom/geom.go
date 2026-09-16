package geom

import (
	"math"
)

// continuous 2d geometry. every shape answers one question: is this point
// inside me. that is all the rasteriser needs, and it means changing a
// proportion is changing one number.

// Vec is written unkeyed throughout the scenes -- Vec{38, 26} -- because a
// scene is mostly coordinates and naming both fields on every one of them
// buries the numbers. that is why this module runs vet with -composites=false.
type Vec struct{ X, Y float64 }

type Shape interface{ In(p Vec) bool }

// ---- primitives ----

type Ellipse struct {
	C      Vec
	Rx, Ry float64
	Rot    float64 // degrees, positive tilts the local +y axis toward -x
}

// In reports whether a point is inside the ellipse, its rotation undone first.
func (e Ellipse) In(p Vec) bool {
	t := e.Rot * math.Pi / 180
	dx, dy := p.X-e.C.X, p.Y-e.C.Y
	u := dx*math.Cos(t) - dy*math.Sin(t)
	v := dx*math.Sin(t) + dy*math.Cos(t)
	return (u*u)/(e.Rx*e.Rx)+(v*v)/(e.Ry*e.Ry) <= 1
}

type Poly struct{ P []Vec }

// In reports whether a point is inside the polygon, by crossing count.
func (g Poly) In(p Vec) bool {
	in := false
	n := len(g.P)
	for i, j := 0, n-1; i < n; j, i = i, i+1 {
		a, b := g.P[i], g.P[j]
		if (a.Y > p.Y) != (b.Y > p.Y) {
			x := (b.X-a.X)*(p.Y-a.Y)/(b.Y-a.Y) + a.X
			if p.X < x {
				in = !in
			}
		}
	}
	return in
}

// Taper is a polyline stroked with a radius that varies per vertex: the
// workhorse for limbs, feathers, trunks and branches.
type Taper struct {
	P []Vec
	R []float64
}

// In reports whether a point is inside the taper: within the swept radius of
// any segment, which is a stroke of varying width.
func (t Taper) In(p Vec) bool {
	for i := 0; i+1 < len(t.P); i++ {
		a, b := t.P[i], t.P[i+1]
		ra, rb := t.R[i], t.R[i+1]
		abx, aby := b.X-a.X, b.Y-a.Y
		l2 := abx*abx + aby*aby
		s := 0.0
		if l2 > 0 {
			s = ((p.X-a.X)*abx + (p.Y-a.Y)*aby) / l2
			s = math.Max(0, math.Min(1, s))
		}
		cx, cy := a.X+s*abx, a.Y+s*aby
		r := ra + s*(rb-ra)
		dx, dy := p.X-cx, p.Y-cy
		if dx*dx+dy*dy <= r*r {
			return true
		}
	}
	return false
}

// HalfPlane keeps the side where n.(p-c) <= 0.
type HalfPlane struct{ C, N Vec }

// In reports whether a point is on the inner side of the half plane.
func (h HalfPlane) In(p Vec) bool {
	return h.N.X*(p.X-h.C.X)+h.N.Y*(p.Y-h.C.Y) <= 0
}

// ---- csg ----

type union struct{ s []Shape }
type isect struct{ s []Shape }
type diff struct{ a, b Shape }

// In reports whether a point is inside any of the shapes.
func (u union) In(p Vec) bool {
	for _, s := range u.s {
		if s.In(p) {
			return true
		}
	}
	return false
}

// In reports whether a point is inside all of the shapes.
func (i isect) In(p Vec) bool {
	for _, s := range i.s {
		if !s.In(p) {
			return false
		}
	}
	return true
}

// In reports whether a point is inside the first shape and not the second.
func (d diff) In(p Vec) bool { return d.a.In(p) && !d.b.In(p) }

// Union is every shape at once.
func Union(s ...Shape) Shape { return union{s} }

// Intersect is where all the shapes overlap.
func Intersect(s ...Shape) Shape { return isect{s} }

// Sub is the first shape with the second taken out of it.
func Sub(a, b Shape) Shape { return diff{a, b} }

// ---- helpers ----

// Arc returns points along a circle, degrees, y-down screen convention.
func Arc(cx, cy, r, a0, a1 float64, n int) []Vec {
	out := make([]Vec, 0, n)
	for i := 0; i < n; i++ {
		a := (a0 + (a1-a0)*float64(i)/float64(n-1)) * math.Pi / 180
		out = append(out, Vec{cx + r*math.Cos(a), cy + r*math.Sin(a)})
	}
	return out
}

// Pts is a slice of points, written inline.
func Pts(v ...Vec) []Vec { return v }

// Cat joins runs of points, for a contour assembled from arcs.
func Cat(groups ...[]Vec) []Vec {
	var out []Vec
	for _, g := range groups {
		out = append(out, g...)
	}
	return out
}

// Rotate turns p about c by deg degrees, screen convention.
func Rotate(p, c Vec, deg float64) Vec {
	a := deg * math.Pi / 180
	dx, dy := p.X-c.X, p.Y-c.Y
	return Vec{c.X + dx*math.Cos(a) - dy*math.Sin(a), c.Y + dx*math.Sin(a) + dy*math.Cos(a)}
}
