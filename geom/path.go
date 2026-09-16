package geom

import "math"

// Path is a set of closed contours filled by the non-zero winding rule. that is
// how outline fonts define holes: the counter of an O is a contour wound the
// other way, not a separate shape subtracted afterwards.
type Path struct {
	Contours [][]Vec
	min, max Vec
}

// NewPath closes over its contours and records their bounds for early reject.
func NewPath(contours [][]Vec) Path {
	p := Path{Contours: contours}
	p.min = Vec{math.Inf(1), math.Inf(1)}
	p.max = Vec{math.Inf(-1), math.Inf(-1)}
	for _, c := range contours {
		for _, v := range c {
			p.min.X = math.Min(p.min.X, v.X)
			p.min.Y = math.Min(p.min.Y, v.Y)
			p.max.X = math.Max(p.max.X, v.X)
			p.max.Y = math.Max(p.max.Y, v.Y)
		}
	}
	return p
}

// Bounds returns the top left and bottom right corners.
func (p Path) Bounds() (Vec, Vec) { return p.min, p.max }

// In reports whether a point is inside the path, by non-zero winding: the rule
// that makes an inner contour a hole rather than another shape.
func (p Path) In(q Vec) bool {
	if q.X < p.min.X || q.X > p.max.X || q.Y < p.min.Y || q.Y > p.max.Y {
		return false
	}
	wn := 0
	for _, c := range p.Contours {
		n := len(c)
		for i := 0; i < n; i++ {
			a, b := c[i], c[(i+1)%n]
			cross := (b.X-a.X)*(q.Y-a.Y) - (q.X-a.X)*(b.Y-a.Y)
			if a.Y <= q.Y {
				if b.Y > q.Y && cross > 0 {
					wn++
				}
			} else if b.Y <= q.Y && cross < 0 {
				wn--
			}
		}
	}
	return wn != 0
}

// Transform maps every point, which is cheaper than inverting per sample.
func (p Path) Transform(m Affine) Path {
	out := make([][]Vec, len(p.Contours))
	for i, c := range p.Contours {
		cc := make([]Vec, len(c))
		for j, v := range c {
			cc[j] = m.Apply(v)
		}
		out[i] = cc
	}
	return NewPath(out)
}

// Affine is x' = A x + B y + C, y' = D x + E y + F.
type Affine struct{ A, B, C, D, E, F float64 }

// Identity is the transform that changes nothing.
func Identity() Affine { return Affine{A: 1, E: 1} }

// Apply moves a point through the transform.
func (m Affine) Apply(p Vec) Vec {
	return Vec{m.A*p.X + m.B*p.Y + m.C, m.D*p.X + m.E*p.Y + m.F}
}

// Then returns the transform that applies m first and n second.
func (m Affine) Then(n Affine) Affine {
	return Affine{
		A: n.A*m.A + n.B*m.D, B: n.A*m.B + n.B*m.E, C: n.A*m.C + n.B*m.F + n.C,
		D: n.D*m.A + n.E*m.D, E: n.D*m.B + n.E*m.E, F: n.D*m.C + n.E*m.F + n.F,
	}
}

// Invert is the transform that undoes this one. A warp is written forwards and
// applied backwards, so the inverse is what actually gets used.
func (m Affine) Invert() Affine {
	det := m.A*m.E - m.B*m.D
	return Affine{
		A: m.E / det, B: -m.B / det, C: (m.B*m.F - m.E*m.C) / det,
		D: -m.D / det, E: m.A / det, F: (m.D*m.C - m.A*m.F) / det,
	}
}

// Translate moves by an offset.
func Translate(dx, dy float64) Affine { return Affine{A: 1, C: dx, E: 1, F: dy} }

// RotateAbout turns by deg degrees about c; positive is clockwise on screen.
func RotateAbout(c Vec, deg float64) Affine {
	a := deg * math.Pi / 180
	s, k := math.Sin(a), math.Cos(a)
	return Translate(-c.X, -c.Y).Then(Affine{A: k, B: -s, D: s, E: k}).Then(Translate(c.X, c.Y))
}

// ScaleAbout scales about a point rather than about the origin.
func ScaleAbout(c Vec, sx, sy float64) Affine {
	return Translate(-c.X, -c.Y).Then(Affine{A: sx, E: sy}).Then(Translate(c.X, c.Y))
}

type transformed struct {
	s   Shape
	inv Affine
}

// In reports whether a point is inside the shape, the transform undone first.
func (t transformed) In(p Vec) bool { return t.s.In(t.inv.Apply(p)) }

// Transformed wraps any shape. prefer Path.Transform for paths.
func Transformed(s Shape, m Affine) Shape { return transformed{s, m.Invert()} }

type weighted struct {
	s    Shape
	r    float64
	ring []Vec
}

// Weight thickens a shape by r (r > 0) or thins it (r < 0), by testing a ring
// of points around each sample. it is an approximation, and it is the stroke
// weight control for type: outline fonts only come in the weights they ship.
func Weight(s Shape, r float64) Shape {
	if r == 0 {
		return s
	}
	const n = 12
	ring := make([]Vec, n)
	for i := range ring {
		a := 2 * math.Pi * float64(i) / n
		ring[i] = Vec{math.Abs(r) * math.Cos(a), math.Abs(r) * math.Sin(a)}
	}
	return weighted{s, r, ring}
}

// In reports whether a point is inside the weighted shape: within r of it,
// sampled around a ring, which dilates a stroke or erodes it when r is
// negative.
func (w weighted) In(p Vec) bool {
	if w.r > 0 {
		if w.s.In(p) {
			return true
		}
		for _, d := range w.ring {
			if w.s.In(Vec{p.X + d.X, p.Y + d.Y}) {
				return true
			}
		}
		return false
	}
	if !w.s.In(p) {
		return false
	}
	for _, d := range w.ring {
		if !w.s.In(Vec{p.X + d.X, p.Y + d.Y}) {
			return false
		}
	}
	return true
}
