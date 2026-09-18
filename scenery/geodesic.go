package scenery

import (
	"math"

	"github.com/janearc/miami/view"
)

// How light moves near a black hole that does not spin.
//
// Everything here is in units of the hole's mass, with the speed of light and
// the gravitational constant both one. In those units the numbers that matter
// are small and fixed:
//
//	r = 2         the horizon; light that gets this close is gone
//	r = 3         the photon sphere, where light can circle
//	b = 3*sqrt(3) the widest a ray can pass and still fall in: the shadow
//	r = 6         the innermost stable orbit, where a disk usually ends
//
// A ray near a mass that does not spin stays in one plane: the plane holding
// the hole, the camera and the direction it set out in. So a ray is traced as
// a curve in that plane, in the radius r and the angle phi around the hole,
// and it obeys one equation in u = 1/r:
//
//	u'' = 3u^2 - u      (a prime is d/dphi)
//
// That is Binet's equation for light with the relativistic term in it. Drop
// the 3u^2 and it is a straight line. Keep it and the line bends, circles or
// falls in.
//
// It is traced backwards, from the camera outward, which is the usual trick:
// only rays that reach the eye are ever followed. Where the curve crosses the
// plane of the disk, the ray has seen the disk there. Where u reaches nought,
// it has left for good and sees whatever is in that direction.

// fate is how a traced ray ended.
type fate uint8

const (
	escaped  fate = iota // left for infinity, and sees the sky there
	captured             // crossed the horizon, or circled until it did
)

// crossing is a place a ray passed through the plane of the disk.
type crossing struct {
	r     float64 // radius, in masses
	phi   float64 // angle around the disk's own axis
	order int     // how many times the ray had already crossed
}

// traced is what one ray met.
type traced struct {
	fate  fate
	away  view.Vec   // escaped: the direction it left in
	cross []crossing // every crossing, nearest the camera first
	// lambda is the ray's angular momentum about the disk's axis, per unit
	// energy, as the light that reaches the camera carries it. It is what
	// says how fast the disk was coming toward the camera where the light
	// left it.
	lambda float64
	// at is the camera's radius, for the redshift of light arriving there.
	at float64
}

// maxTurn is how far round a ray is followed before it is counted as caught.
// A ray that has circled the hole four times has an impact parameter within
// about a millionth of the critical one, and what it would see is a ring
// thinner than a subpixel.
const maxTurn = 8 * math.Pi

// trace follows a ray from a camera at cam, in masses from the hole, setting
// out in direction dir, and reports where it crossed the plane square to
// normal and where it went. dir and normal need not be unit length.
func trace(cam, dir, normal view.Vec, out *traced) {
	out.cross = out.cross[:0]
	r0 := length(cam)
	e1 := scale(cam, 1/r0)
	d := unit(dir)
	a := dot(d, e1)
	perp := sub(d, scale(e1, a))
	c := length(perp)
	out.at = r0
	if c < 1e-12 {
		// straight at the hole or straight away from it
		if a < 0 {
			out.fate = captured
		} else {
			out.fate, out.away = escaped, d
		}
		return
	}
	e2 := scale(perp, 1/c)
	n := unit(normal)

	// the impact parameter as a static camera at r0 measures it, and the
	// angular momentum about the disk axis the arriving light carries:
	// opposite in sense to the backward ray's own
	b := r0 * c / math.Sqrt(1-2/r0)
	out.lambda = -b * dot(cross(e1, e2), n)

	u := 1 / r0
	up := -a * u * math.Sqrt(1-2*u) / c
	phi := 0.0

	// height above the disk plane, as a function of phi, is
	// r (A cos phi + B sin phi); only its sign matters
	A, B := dot(e1, n), dot(e2, n)
	side := A
	dx, dy := basis(n)

	for phi < maxTurn {
		h := step(u)
		u1, up1 := rk4(u, up, h)
		phi1 := phi + h
		if u1 <= 0 {
			// left: where u crossed nought is the direction it went
			f := u / (u - u1)
			away := phi + f*h
			out.fate = escaped
			out.away = add(scale(e1, math.Cos(away)),
				scale(e2, math.Sin(away)))
			return
		}
		if u1 >= 0.5 {
			out.fate = captured
			return
		}
		side1 := A*math.Cos(phi1) + B*math.Sin(phi1)
		if (side > 0) != (side1 > 0) {
			at := crossingPhi(A, B, phi, phi1)
			uc := hermite(u, up, u1, up1, (at-phi)/h, h)
			if uc > 0 {
				p := add(scale(e1, math.Cos(at)),
					scale(e2, math.Sin(at)))
				around := math.Atan2(dot(p, dy), dot(p, dx))
				out.cross = append(out.cross, crossing{
					r: 1 / uc, phi: around,
					order: len(out.cross)})
			}
		}
		u, up, phi, side = u1, up1, phi1, side1
	}
	out.fate = captured
}

// step is how far round to go in one step: finer near the hole, where the
// curve bends hardest, and coarse far away, where it is nearly straight.
func step(u float64) float64 {
	switch {
	case u > 0.1:
		return 0.02
	case u > 0.03:
		return 0.04
	}
	return 0.08
}

// rk4 steps the orbit equation forward by h, the classical fourth-order way.
func rk4(u, up, h float64) (float64, float64) {
	acc := func(u float64) float64 { return 3*u*u - u }
	k1u, k1p := up, acc(u)
	k2u, k2p := up+h/2*k1p, acc(u+h/2*k1u)
	k3u, k3p := up+h/2*k2p, acc(u+h/2*k2u)
	k4u, k4p := up+h*k3p, acc(u+h*k3u)
	return u + h/6*(k1u+2*k2u+2*k3u+k4u), up + h/6*(k1p+2*k2p+2*k3p+k4p)
}

// crossingPhi is the angle between phi0 and phi1 where A cos + B sin is
// nought. Such angles are a half turn apart, so there is exactly one in a step
// shorter than that.
func crossingPhi(A, B, phi0, phi1 float64) float64 {
	base := math.Atan2(-A, B)
	k := math.Ceil((phi0 - base) / math.Pi)
	at := base + k*math.Pi
	return math.Max(phi0, math.Min(phi1, at))
}

// hermite is u between two steps, from both ends' values and slopes, at a
// fraction s of a step of length h.
func hermite(u0, p0, u1, p1, s, h float64) float64 {
	s2, s3 := s*s, s*s*s
	return (2*s3-3*s2+1)*u0 + (s3-2*s2+s)*h*p0 + (-2*s3+3*s2)*u1 +
		(s3-s2)*h*p1
}

// redshift is how much bluer (above one) or redder (below) light from a disk
// on a circular orbit at radius r arrives at a static camera, for a ray
// carrying angular momentum lambda. spin is one for a disk turning
// anticlockwise about its axis and minus one for the other way.
//
// It is the ratio of the energy the camera measures to the energy the gas
// measured, and it has three things in it: the gas moving toward or away,
// time running slow down at r, and running a little slow up at the camera.
func redshift(r, lambda, at, spin float64) float64 {
	if r <= 3 {
		return 0 // no circular orbit for anything with mass in here
	}
	omega := spin * math.Pow(r, -1.5)
	ut := 1 / math.Sqrt(1-3/r)
	return 1 / (math.Sqrt(1-2/at) * ut * (1 - omega*lambda))
}

// dot is the dot product.
func dot(a, b view.Vec) float64 { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }

// cross is the vector square to both.
func cross(a, b view.Vec) view.Vec {
	return view.Vec{X: a.Y*b.Z - a.Z*b.Y, Y: a.Z*b.X - a.X*b.Z,
		Z: a.X*b.Y - a.Y*b.X}
}

// add is two vectors summed.
func add(a, b view.Vec) view.Vec {
	return view.Vec{X: a.X + b.X, Y: a.Y + b.Y, Z: a.Z + b.Z}
}

// sub is one vector less another.
func sub(a, b view.Vec) view.Vec {
	return view.Vec{X: a.X - b.X, Y: a.Y - b.Y, Z: a.Z - b.Z}
}

// scale is a vector lengthened or shortened.
func scale(a view.Vec, k float64) view.Vec {
	return view.Vec{X: a.X * k, Y: a.Y * k, Z: a.Z * k}
}

// length is a vector's length.
func length(a view.Vec) float64 { return math.Sqrt(dot(a, a)) }

// unit is a vector of length one, or straight ahead if it has no length.
func unit(a view.Vec) view.Vec {
	l := length(a)
	if l == 0 {
		return view.Vec{Z: 1}
	}
	return scale(a, 1/l)
}

// basis is two unit vectors square to n and to each other, the same two every
// time for the same n.
func basis(n view.Vec) (u, v view.Vec) {
	a := view.Vec{X: 1}
	if math.Abs(n.X) > 0.9 {
		a = view.Vec{Y: 1}
	}
	u = unit(cross(n, a))
	return u, cross(unit(n), u)
}
