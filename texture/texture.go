// Package texture is what a surface looks like at a point on it.
//
// A shape says where it is and a texture says what it is made of, and keeping
// those apart is worth a package.
//
// The checks on the ball and the checks on the floor were one thing written
// twice, because one lived in a sphere and one lived in a plane. They are the
// same pattern, and a cylinder wanting checks was the third copy.
//
// So a shape works out where on itself a ray landed, in two numbers, and asks.
// What comes back is a role -- never a colour -- so a pattern is still a
// stencil and a colourway still dresses it.
//
//	u  around, or across       nought to one
//	v  along, or up            nought to one
//
// What u and v mean is the shape's business and it is written down per shape.
// A texture only has to be a function of two numbers, which is why the same
// Checker can be a floor, a beach ball and a barber's pole.
//
// There is no filtering and no mipmapping. At a sextant a subcell is already
// the smallest thing there is, and a pattern finer than one either reads as
// noise or is noise; the answer to a check too small to see is a bigger check,
// not a cleverer sampler.
package texture

import (
	"math"

	"github.com/janearc/miami/grade"
)

// Texture is a pattern over a surface: two coordinates in, a role out.
type Texture interface {
	At(u, v float64) byte
}

// Solid is one role everywhere.
type Solid byte

// At is the role.
func (s Solid) At(_, _ float64) byte { return byte(s) }

// Checker is the classic: two roles alternating in both directions.
//
// U and V are how many squares fit across the whole of each coordinate. The
// phases exist because a shape's coordinates do not always start where a
// pattern wants to: a sphere's latitude runs from a pole rather than from a
// seam, and without a half-square offset the check straddles its equator.
type Checker struct {
	A, B           byte
	U, V           float64
	PhaseU, PhaseV float64
}

// At is the role at a point.
func (c Checker) At(u, v float64) byte {
	i := math.Floor(u*c.U+c.PhaseU) + math.Floor(v*c.V+c.PhaseV)
	if int(i)&1 == 0 {
		return c.A
	}
	return c.B
}

// Bands splits u into equal strips and gives each its own texture.
//
// This is how a ball is told apart from itself. One check over a whole sphere
// says nothing about which way it is turning -- every quarter looks like every
// other quarter -- and four different quarters say it at a glance. It is a
// pattern for reading rotation, and it is why the ball has one.
type Bands struct{ Of []Texture }

// At is the role, from whichever band u falls in.
func (b Bands) At(u, v float64) byte {
	if len(b.Of) == 0 {
		return '.'
	}
	i := int(math.Floor(u * float64(len(b.Of))))
	i = max(i, 0)
	if i >= len(b.Of) {
		i = len(b.Of) - 1
	}
	return b.Of[i].At(u, v)
}

// Noise is roles scattered by a hash: the same every time, and without a seam.
//
// Not random. A surface that draws itself from a random source crawls, because
// every frame is a different surface; this is arithmetic on the coordinate, so
// a point keeps its role for as long as it is that point. Scale is how many
// cells of noise fit across the surface.
type Noise struct {
	Roles []byte
	Scale float64
}

// At is the role at a point.
func (n Noise) At(u, v float64) byte {
	if len(n.Roles) == 0 {
		return '.'
	}
	s := n.Scale
	if s <= 0 {
		s = 8
	}
	h := grade.Hash2(math.Floor(u*s), math.Floor(v*s))
	i := int(h * float64(len(n.Roles)))
	if i >= len(n.Roles) {
		i = len(n.Roles) - 1
	}
	return n.Roles[i]
}

// Stripes is bands, for a surface that wants a direction.
//
// Duty is how much of each period is the first role, so a hairline is a duty
// near nought and an even pair is a half.
//
// Lean is what makes a barber's pole. With no lean the bands run straight
// around a cylinder, which at the height of a progress bar is three stripes
// and reads as a painted rectangle; leaning them makes a helix, and a helix on
// a turning pole appears to travel along it.
//
// That last part is an illusion and it is the useful one: nothing is moving
// lengthways, but a person watching sees the bar going somewhere, which is how
// a bar says it is working while nothing has finished.
type Stripes struct {
	A, B         byte
	Period, Duty float64
	Phase        float64
	Lean         float64
}

// At is the role at a point.
func (s Stripes) At(u, v float64) byte {
	v += s.Lean * u
	p := s.Period
	if p <= 0 {
		p = 0.25
	}
	d := s.Duty
	if d <= 0 {
		d = 0.5
	}
	f := math.Mod(v/p+s.Phase, 1)
	if f < 0 {
		f++
	}
	if f < d {
		return s.A
	}
	return s.B
}

// Split is two textures either side of a line along u, for a surface that shows
// how far along something is.
//
// This is what makes a cylinder a progress bar. The cylinder does not know what
// a percentage is; it asks its texture, and the texture is a line that moves.
type Split struct {
	Done, Rest Texture
	Frac       func() float64 // nought to one; nil is nought
}

// At is the role at a point.
func (s Split) At(u, v float64) byte {
	f := 0.0
	if s.Frac != nil {
		f = s.Frac()
	}
	if u <= f {
		if s.Done != nil {
			return s.Done.At(u, v)
		}
		return '.'
	}
	if s.Rest != nil {
		return s.Rest.At(u, v)
	}
	return '.'
}
