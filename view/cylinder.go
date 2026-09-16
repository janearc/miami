package view

import (
	"math"

	"github.com/janearc/miami/texture"
)

// A cylinder: the shape a progress bar is.
//
// A bar drawn as a row of half blocks is a widget laid on top of a picture, and
// it always looks like one. A cylinder is in the scene -- it stands on the
// floor, the light finds it, it throws a shadow, and the ball can pass behind
// it -- and the thing that makes it a bar is its texture rather than its shape.
//
// It does not know what a percentage is. It works out how far along itself a
// ray landed and asks, which is why the same cylinder is a barber's pole, a
// pipe, or the length of the afternoon.
type Cylinder struct {
	At      Vec     // the centre
	Axis    Vec     // which way it lies; the x axis if zero
	Radius  float64 // world units
	Length  float64 // world units, end to end
	Texture texture.Texture
	Caps    bool // draw the flat ends; an open tube if false
}

// axis is the cylinder's direction, unit length, defaulting to lying across the
// view rather than standing up -- a bar is a horizontal thing.
func (c Cylinder) axis() Vec {
	a := c.Axis
	if a.X == 0 && a.Y == 0 && a.Z == 0 {
		return Vec{X: 1}
	}
	return unit(a)
}

// Hit is where a ray meets the cylinder, and the role there.
//
// The side is the ordinary quadratic, taken perpendicular to the axis: strip
// the axial part out of the ray and what is left is a circle problem in the
// plane. The caps are two discs. Both are checked and the nearer wins, which is
// what stops a ray that enters through an end from painting the far wall.
func (c Cylinder) Hit(r Ray, _ float64, near, far float64) (float64, byte, bool) {
	a := c.axis()
	half := c.Length / 2
	o := sub(r.At, c.At)

	// the ray with its axial component removed
	dPar, oPar := dot(r.Dir, a), dot(o, a)
	dp := sub(r.Dir, scale(a, dPar))
	op := sub(o, scale(a, oPar))

	best, hit := math.Inf(1), false
	var role byte

	qa := dot(dp, dp)
	if qa > 1e-12 {
		qb := 2 * dot(op, dp)
		qc := dot(op, op) - c.Radius*c.Radius
		if disc := qb*qb - 4*qa*qc; disc >= 0 {
			root := math.Sqrt(disc)
			for _, t := range [2]float64{(-qb - root) / (2 * qa),
				(-qb + root) / (2 * qa)} {
				if t <= near || t >= far || t >= best {
					continue
				}
				h := oPar + t*dPar // how far along the axis
				if h < -half || h > half {
					continue
				}
				p := add(o, scale(r.Dir, t))
				radial := sub(p, scale(a, h))
				best, hit, role = t, true,
					c.roleAt(h/c.Length+0.5, aroundOf(radial, a))
			}
		}
	}

	if c.Caps && math.Abs(dPar) > 1e-12 {
		for _, end := range [2]float64{-half, half} {
			t := (end - oPar) / dPar
			if t <= near || t >= far || t >= best {
				continue
			}
			p := add(o, scale(r.Dir, t))
			radial := sub(p, scale(a, end))
			if dot(radial, radial) > c.Radius*c.Radius {
				continue
			}
			best, hit, role = t, true,
				c.roleAt(end/c.Length+0.5, aroundOf(radial, a))
		}
	}
	return best, role, hit
}

// roleAt asks the texture. u runs along the cylinder from one end to the other,
// which is the direction a bar fills; v runs around it.
func (c Cylinder) roleAt(u, v float64) byte {
	if c.Texture == nil {
		return 'W'
	}
	return c.Texture.At(clamp(u, 0, 1), v)
}

// Blocks says whether the cylinder stands in the way of a light.
func (c Cylinder) Blocks(r Ray, near, far float64) bool {
	_, _, ok := c.Hit(r, 0, near, far)
	return ok
}

// aroundOf is how far around the axis a point on the surface is, nought to one.
//
// It needs a fixed zero to measure from, and any direction perpendicular to the
// axis will do as long as it is the same one every time. Picking the world axis
// the cylinder leans on least avoids the degenerate case where the reference
// and the axis are parallel and the cross product is nothing.
func aroundOf(radial, a Vec) float64 {
	ref := Vec{Y: 1}
	if math.Abs(a.Y) > 0.9 {
		ref = Vec{X: 1}
	}
	u := unit(sub(ref, scale(a, dot(ref, a))))
	w := cross(a, u)
	return math.Atan2(dot(radial, w), dot(radial, u))/(2*math.Pi) + 0.5
}

// cross is the vector perpendicular to two others.
func cross(a, b Vec) Vec {
	return Vec{
		X: a.Y*b.Z - a.Z*b.Y,
		Y: a.Z*b.X - a.X*b.Z,
		Z: a.X*b.Y - a.Y*b.X,
	}
}
