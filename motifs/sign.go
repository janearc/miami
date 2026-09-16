package motifs

import (
	"math"

	"github.com/janearc/miami/geom"
)

// ArchPanel is a sign face with a segmental arched top: straight sides from the
// spring line down, and a circular arc above it rising to crown. inset shrinks
// it concentrically, so a frame and its face share one arc centre.
type ArchPanel struct {
	X0, X1, Crown, Spring, Bottom float64
}

// arc is the centre and radius of the panel's crown, from its span and rise.
func (a ArchPanel) arc() (geom.Vec, float64) {
	c := (a.X1 - a.X0) / 2
	s := a.Spring - a.Crown
	r := (c*c + s*s) / (2 * s)
	return geom.Vec{X: (a.X0 + a.X1) / 2, Y: a.Crown + r}, r
}

// Shape returns the panel shrunk inward by inset on every side.
func (a ArchPanel) Shape(inset float64) geom.Shape {
	cen, r := a.arc()
	rect := geom.Poly{P: geom.Pts(
		geom.Vec{X: a.X0 + inset, Y: a.Crown + inset}, geom.Vec{X: a.X1 - inset, Y: a.Crown + inset},
		geom.Vec{X: a.X1 - inset, Y: a.Bottom - inset}, geom.Vec{X: a.X0 + inset, Y: a.Bottom - inset})}
	top := geom.Union(
		geom.Ellipse{C: cen, Rx: r - inset, Ry: r - inset},
		geom.HalfPlane{C: geom.Vec{X: 0, Y: a.Spring}, N: geom.Vec{X: 0, Y: -1}},
	)
	return geom.Intersect(rect, top)
}

// Border returns evenly spaced points running round the panel at inset.
func (a ArchPanel) Border(inset, spacing float64) []geom.Vec {
	cen, r := a.arc()
	ri := r - inset
	x0, x1 := a.X0+inset, a.X1-inset
	// the arc meets the sides where x = x0 and x = x1
	dy := math.Sqrt(math.Max(ri*ri-(x1-cen.X)*(x1-cen.X), 0))
	yJoin := cen.Y - dy
	// screen y runs down and the joins sit above the centre, so both angles
	// are negative and the arc over the top runs from the smaller (left) to
	// the larger (right). adding a turn here walked it round underneath.
	aL := math.Atan2(yJoin-cen.Y, x0-cen.X)
	aR := math.Atan2(yJoin-cen.Y, x1-cen.X)
	var path []geom.Vec
	const n = 60
	for i := 0; i <= n; i++ {
		t := aL + (aR-aL)*float64(i)/n
		path = append(path, geom.Vec{X: cen.X + ri*math.Cos(t), Y: cen.Y + ri*math.Sin(t)})
	}
	yb := a.Bottom - inset
	path = append(path, geom.Vec{X: x1, Y: yb}, geom.Vec{X: x0, Y: yb}, geom.Vec{X: x0, Y: yJoin})
	return Along(path, spacing)
}

// Along samples a polyline at a fixed spacing. points closer than 70 percent of
// the spacing to one already kept are dropped: without that, the end of one
// segment and the start of the next put two bulbs side by side at a corner.
func Along(path []geom.Vec, spacing float64) []geom.Vec {
	raw := along(path, spacing)
	var out []geom.Vec
	for _, p := range raw {
		near := false
		for _, q := range out {
			if math.Hypot(p.X-q.X, p.Y-q.Y) < 0.7*spacing {
				near = true
				break
			}
		}
		if !near {
			out = append(out, p)
		}
	}
	return out
}

// along is points spaced evenly down a path, for lamps around a border.
func along(path []geom.Vec, spacing float64) []geom.Vec {
	var out []geom.Vec
	carry := 0.0
	for i := 0; i+1 < len(path); i++ {
		a, b := path[i], path[i+1]
		l := math.Hypot(b.X-a.X, b.Y-a.Y)
		for d := carry; d < l; d += spacing {
			t := d / l
			out = append(out, geom.Vec{X: a.X + (b.X-a.X)*t, Y: a.Y + (b.Y-a.Y)*t})
		}
		carry = math.Mod(carry-l, spacing)
		if carry < 0 {
			carry += spacing
		}
	}
	return out
}

// Dots is a round mark at every point: marquee bulbs, chaser lights.
func Dots(pts []geom.Vec, r float64) geom.Shape {
	var s []geom.Shape
	for _, p := range pts {
		s = append(s, geom.Ellipse{C: p, Rx: r, Ry: r})
	}
	return geom.Union(s...)
}

// Sparkle is a six-armed star. not four: at terminal size a four-armed
// sparkle's short diagonals vanish and what is left is a plus sign -- a cross,
// which the brief put the bulb in place of precisely to avoid.
func Sparkle(c geom.Vec, arm float64) geom.Shape {
	var s []geom.Shape
	for i := 0; i < 6; i++ {
		a := float64(i)*math.Pi/3 + math.Pi/6
		l := arm
		w := arm * 0.17
		s = append(s, geom.Taper{
			P: []geom.Vec{c, {X: c.X + l*math.Cos(a), Y: c.Y + l*math.Sin(a)}},
			R: []float64{w, w * 0.2},
		})
	}
	return geom.Union(s...)
}

// Arrow is a flat roadside arrow pointing right: a shaft and a head.
type Arrow struct {
	X0, X1, Y, Shaft, Head, HeadLen float64
}

// Shape is the arrow as geometry: a shaft and a head.
func (a Arrow) Shape() geom.Shape {
	hx := a.X1 - a.HeadLen
	return geom.Union(
		geom.Poly{P: geom.Pts(geom.Vec{X: a.X0, Y: a.Y - a.Shaft/2}, geom.Vec{X: hx, Y: a.Y - a.Shaft/2},
			geom.Vec{X: hx, Y: a.Y + a.Shaft/2}, geom.Vec{X: a.X0, Y: a.Y + a.Shaft/2})},
		geom.Poly{P: geom.Pts(geom.Vec{X: hx, Y: a.Y - a.Head/2}, geom.Vec{X: a.X1, Y: a.Y}, geom.Vec{X: hx, Y: a.Y + a.Head/2})},
	)
}

// Lights returns chaser-bulb positions running just inside the arrow outline.
func (a Arrow) Lights(inset, spacing float64) []geom.Vec {
	hx := a.X1 - a.HeadLen
	top, bot := a.Y-a.Shaft/2+inset, a.Y+a.Shaft/2-inset
	// the head's edges, pulled in along their normals by roughly inset
	tip := geom.Vec{X: a.X1 - inset*2.2, Y: a.Y}
	path := []geom.Vec{
		{X: a.X0 + inset, Y: bot}, {X: a.X0 + inset, Y: top}, {X: hx, Y: top},
		{X: hx + inset*0.4, Y: a.Y - a.Head/2 + inset*1.8}, tip,
		{X: hx + inset*0.4, Y: a.Y + a.Head/2 - inset*1.8}, {X: hx, Y: bot}, {X: a.X0 + inset, Y: bot}}
	return Along(path, spacing)
}
