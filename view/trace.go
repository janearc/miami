package view

import (
	"math"
	"time"

	sprite "github.com/janearc/libdaffy-sprite"

	"github.com/janearc/miami/internal/split"
	"github.com/janearc/miami/texture"
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
//	nothing else     no bounces, no ambient occlusion, no secondary rays
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
	for _, t := range [2]float64{
		(-b - root) / (2 * a), (-b + root) / (2 * a),
	} {
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

	// Texture is the pattern it wears; the plain A/B check if nil.
	Texture texture.Texture
}

// Light is a lamp, or the sun. Off is not the same as far away: a sun below the
// horizon reaches nothing, and counting it as blocked everywhere is what makes
// the floor go one step darker at night without a second rule for it.
type Light struct {
	At  Vec
	Off bool
}

// Shadowed is the role a floor square takes when one light cannot reach it.
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

// Darken steps a role down once for every light that cannot reach it.
//
// This is what a second light costs, and it is not a bigger boolean. With one
// lamp a point is lit or it is not; with two it is lit by both, by one, or by
// neither, and the middle case has to look different from the other two or the
// second light might as well not be there.
//
// Stepping the role down once per blocked light gives three depths out of the
// ladder a colourway already binds, and the shadows cross where they overlap.
func (f Floor) Darken(role byte, blocked int) byte {
	for i := 0; i < blocked; i++ {
		role = f.Shadowed(role)
	}
	return role
}

// Lit is a scene a ray tracer can answer questions about: some shapes, and
// some lights.
//
// It used to be one ball and one floor, named. Naming them is what stopped
// there ever being two balls, or a cylinder, and the loop below no longer knows
// what any of the shapes are -- it asks each of them the same two questions.
type Lit struct {
	Solids []Solid
	Lights []Light
	Eye    Eye
}

// Trace renders a frame of roles by casting rays.
func (s Lit) Trace(t time.Duration, cols, rows int) sprite.Frame {
	// the picture plane is one unit out and as wide as the sheet; Eye.Ray
	// says how a subpixel maps onto it, for this and everything else
	w, h := cols*2, rows*3
	secs, from := t.Seconds(), s.Eye.At

	// on every core: a ray asks the shapes questions and changes nothing
	// about them, so rows cannot see each other
	art := make([]string, h)
	split.Rows(h, func(lo, hi int) {
		row := make([]byte, w)
		for y := lo; y < hi; y++ {
			for x := 0; x < w; x++ {
				d := s.Eye.Ray(x, y, w, h)
				row[x] = s.shade(Ray{At: from, Dir: d}, secs)
			}
			art[y] = string(row)
		}
	})
	return sprite.Frame{Art: art}
}

// shade is what one ray sees: the nearest shape it meets, and what a light can
// reach there.
func (s Lit) shade(eye Ray, t float64) byte {
	best, role, which := math.Inf(1), byte('.'), -1
	for i, sh := range s.Solids {
		d, r, ok := sh.Hit(eye, t, 1e-4, best)
		if ok && d < best {
			best, role, which = d, r, i
		}
	}
	if which < 0 {
		return '.' // the sky, which is the terminal's own background
	}
	shader, ok := s.Solids[which].(Shader)
	if !ok {
		return role
	}

	// one shadow ray per light, each stopping at its own light rather than
	// running on for ever: something behind a lamp is not in the way of it.
	p := add(eye.At, scale(eye.Dir, best))
	blocked := 0
	for _, l := range s.Lights {
		if l.Off {
			blocked++
			continue
		}
		ray := Ray{At: p, Dir: sub(l.At, p)}
		for i, sh := range s.Solids {
			if i == which {
				continue // nothing shadows itself here
			}
			if sh.Blocks(ray, 1e-3, 1) {
				blocked++
				break
			}
		}
	}
	return shader.Darken(role, blocked)
}

// Hit is where a ray meets the floor, and the role there.
//
// The check is the floor's texture if it has one, and its two roles in a plain
// check if it does not -- the second being what every floor in the library was
// before patterns were a thing they could be given.
func (f Floor) Hit(r Ray, _ float64, near, far float64) (float64, byte, bool) {
	t, ok := HitPlane(r, f.Y, near, far)
	if !ok {
		return 0, 0, false
	}
	p := add(r.At, scale(r.Dir, t))
	if f.Far > 0 && p.Z > f.Far {
		return 0, 0, false // past where a square can still be resolved
	}
	if f.Texture != nil {
		// a plane has no edges, so there is no nought to one to hand
		// a texture. the tile is the unit, and the count is not
		// wrapped: handed only the fraction within a tile, every tile
		// looks identical and a checker comes out a flat colour.
		tile := f.Tile
		if tile == 0 {
			tile = 1
		}
		return t, f.Texture.At(p.X/tile, p.Z/tile), true
	}
	role := f.A
	if (int(math.Floor(p.X/f.Tile))+int(math.Floor(p.Z/f.Tile)))&1 == 0 {
		role = f.B
	}
	return t, role, true
}

// Blocks is false: a floor never shadows anything.
//
// Not an oversight. The floor is the plane every shadow lands on, and a plane
// that blocks light blocks all of it -- every point on it is below the floor
// from the point of view of a light above, so the whole world goes dark.
func (Floor) Blocks(Ray, float64, float64) bool { return false }
