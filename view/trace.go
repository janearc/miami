package view

import (
	"math"
	"time"

	sprite "github.com/janearc/libdaffy-sprite"
)

// Rays, and the one thing they are for here: asking whether something is in the
// way.
//
// A shadow is not a lighting model. It is the answer to "does the straight line
// from this point to the light meet anything", and for a sphere that is a
// quadratic with no iteration in it.
//
// That is the whole of why tracing is cheap in a scene made of objects that can
// answer questions about themselves, rather than one made of triangles that
// cannot.
//
// Three rays are cast per subpixel and no more:
//
//	the view ray     what is seen here -- the ball, the floor, or nothing
//	the shadow ray   from the floor toward the light: blocked, or not
//	nothing else     no bounces, no ambient occlusion, no second generation
//
// Colour never enters. Every answer comes back as a role, so the picture is
// still a stencil and a colourway still dresses it -- including the shadow,
// which is a role remap rather than a darkening, and therefore survives a
// sixteen-colour target and a theme written next year.

// Ray is a point and a direction. The direction is not required to be unit
// length; every intersection below divides it out.
type Ray struct{ At, Dir Vec }

// dot is the dot product.
func dot(a, b Vec) float64 { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }

// sub is one vector less another.
func sub(a, b Vec) Vec { return Vec{a.X - b.X, a.Y - b.Y, a.Z - b.Z} }

// add is two vectors summed.
func add(a, b Vec) Vec { return Vec{a.X + b.X, a.Y + b.Y, a.Z + b.Z} }

// scale is a vector lengthened or shortened.
func scale(a Vec, s float64) Vec {
	return Vec{a.X * s, a.Y * s, a.Z * s}
}

// unit is a vector of length one, or the vector itself if it has no length.
func unit(a Vec) Vec {
	l := math.Sqrt(dot(a, a))
	if l == 0 {
		return a
	}
	return scale(a, 1/l)
}

// HitSphere is the nearest distance along a ray at which it meets a sphere, or
// false. The near limit exists to stop a shadow ray leaving a surface and
// immediately striking the surface it left, which is the oldest bug in ray
// tracing and shows up as the object shadowing itself in a speckle.
func HitSphere(r Ray, centre Vec, radius, near, far float64) (float64, bool) {
	oc := sub(r.At, centre)
	a := dot(r.Dir, r.Dir)
	b := 2 * dot(oc, r.Dir)
	c := dot(oc, oc) - radius*radius
	disc := b*b - 4*a*c
	if disc < 0 {
		return 0, false
	}
	root := math.Sqrt(disc)
	for _, t := range [2]float64{(-b - root) / (2 * a), (-b + root) / (2 * a)} {
		if t > near && t < far {
			return t, true
		}
	}
	return 0, false
}

// HitPlane is where a ray meets the horizontal plane at height y.
func HitPlane(r Ray, y, near, far float64) (float64, bool) {
	if math.Abs(r.Dir.Y) < 1e-9 {
		return 0, false // parallel to the floor, and so never on it
	}
	t := (y - r.At.Y) / r.Dir.Y
	if t <= near || t >= far {
		return 0, false
	}
	return t, true
}

// Floor is a checkered horizontal plane, in two roles.
type Floor struct {
	Y    float64 // its height in world units
	Tile float64 // how many world units a square is
	A, B byte    // the two roles it is checked in
	// Far is where the floor stops. A plane really does go on for ever, and
	// drawn that way its squares fall below one subpixel long before the
	// horizon and turn into a band of moire -- the same failure the
	// sunset's grid had, and the same answer:
	//
	// stop drawing what cannot be resolved.
	Far   float64
	Shade map[byte]byte
}

// Shadowed is the role a floor square takes when the light cannot reach it.
//
// A remap and not a darkening. The alphabet has no midpoint between two roles
// and inventing one would put the shadow outside whatever the colourway chose;
// sending each role to a darker role keeps every colour in the picture one the
// theme already picked.
func (f Floor) Shadowed(role byte) byte {
	if f.Shade != nil {
		if r, ok := f.Shade[role]; ok {
			return r
		}
	}
	return role
}

// Lit is a scene a ray tracer can answer questions about: one ball, one floor,
// one light. Deliberately not a general scene graph -- this is the smallest
// arrangement in which a shadow means anything, and the shape of the code does
// not change when there are nine balls.
type Lit struct {
	Ball  Ball
	Floor Floor
	Light Vec // where the lamp is, in world units
	Eye   Eye
}

// Trace renders a frame of roles by casting rays.
func (s Lit) Trace(t time.Duration, cols, rows int) sprite.Frame {
	w, h := cols*2, rows*3
	// the picture plane is one unit out and as wide as the sheet. a subcell
	// is half a column by a third of a row, so on a cell twice as tall as
	// it is wide the sheet is four rows high for every three columns across
	// -- the same correction as everywhere else, applied once, here.
	aspect := (float64(h) * 4) / (float64(w) * 3)
	lens := s.Eye.Lens
	if lens == 0 {
		lens = 1
	}

	ballAt, _ := s.Ball.Extent(t)
	spin := s.Ball.Spin_(t)

	art := make([]string, h)
	for y := 0; y < h; y++ {
		row := make([]byte, w)
		py := (0.5 - (float64(y)+0.5)/float64(h)) * aspect
		for x := 0; x < w; x++ {
			px := (float64(x)+0.5)/float64(w) - 0.5
			eye := Ray{At: s.Eye.At, Dir: Vec{px / lens, py / lens, 1}}

			tb, hitBall := HitSphere(eye, ballAt, s.Ball.Radius, 1e-4, math.Inf(1))
			tf, hitFloor := HitPlane(eye, s.Floor.Y, 1e-4, math.Inf(1))

			switch {
			case hitBall && (!hitFloor || tb < tf):
				p := add(eye.At, scale(eye.Dir, tb))
				row[x] = surfaceAt(spin, unit(sub(p, ballAt)))

			case hitFloor:
				p := add(eye.At, scale(eye.Dir, tf))
				if s.Floor.Far > 0 && p.Z > s.Floor.Far {
					row[x] = '.'
					break
				}
				role := s.Floor.A
				if (int(math.Floor(p.X/s.Floor.Tile))+
					int(math.Floor(p.Z/s.Floor.Tile)))&1 == 0 {
					role = s.Floor.B
				}
				// the shadow ray. it stops at the light rather
				// than running on for ever: something behind
				// the lamp is not in the way.
				toLight := sub(s.Light, p)
				if _, blocked := HitSphere(
					Ray{At: p, Dir: toLight}, ballAt, s.Ball.Radius, 1e-3, 1); blocked {
					role = s.Floor.Shadowed(role)
				}
				row[x] = role

			default:
				row[x] = '.' // the sky, which is the terminal's own background
			}
		}
		art[y] = string(row)
	}
	return sprite.Frame{Art: art}
}
