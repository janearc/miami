// Package particles is sparks, jets, dust and matter falling in.
//
// A particle system is usually a list that is stepped each frame: move every
// particle a little, kill the old, spawn the new. That keeps state, it depends
// on the frame rate, and a frame cannot be drawn without drawing every frame
// before it.
//
// None of that is needed. A particle is born at a time, from a place, in a
// direction, and all three come from a hash of its number. Where it is at
// any later moment is then a formula, so an emitter is a function of time
// and nothing else:
//
//	particle i is born at        (i + jitter) / rate
//	it is alive at t if          t - born is between nought and life
//	so the ones alive at t are   ceil((t-life)*rate) .. floor(t*rate)
//
// Nothing is stored and nothing is stepped. A game can jump to a moment, a
// frame is the same frame however many times it is drawn, and a test can ask
// where particle 40 is at eleven seconds.
//
// Negative numbers are particles born before nought, which is why a fountain
// is already full on the first frame instead of starting dry.
package particles

import (
	"math"

	"github.com/janearc/miami/light"
	"github.com/janearc/miami/noise"
	"github.com/janearc/miami/view"
)

// Kind is how an emitter throws what it throws.
type Kind int

const (
	// Fountain throws particles out in a cone about Axis, and they fall
	// under Gravity, slowed by Drag.
	Fountain Kind = iota
	// Jet throws them both ways along Axis, fast and nearly straight, and
	// the axis turns slowly about itself by Wobble: what comes out of the
	// poles of a black hole.
	Jet
	// Spiral starts them on a ring of Radius in the plane Axis is square
	// to, and they fall inward to Inner, going round faster as they go.
	Spiral
	// Drift fills a box of half-size Radius with particles that all move
	// along Axis and wrap at its sides: dust the camera is flying through.
	Drift
)

// Emitter is a source of particles, described completely by its numbers.
type Emitter struct {
	Kind Kind
	Seed uint64

	At   view.Vec // where they come from, or the middle of the box
	Axis view.Vec // which way; need not be unit length

	Rate   float64 // born a second; a drift holds Rate*Life at once
	Life   float64 // seconds each lives
	Speed  float64 // world units a second; a spiral's at Radius
	Spread float64 // radians either side of Axis

	Gravity view.Vec // fountain
	Drag    float64  // fountain: how fast speed decays, a second
	Radius  float64  // spiral: the ring they start on; drift: the box
	Inner   float64  // spiral: where they vanish
	Wobble  float64  // jet: radians the axis leans as it turns

	Size   float64    // world units, one particle's radius
	Light  float64    // how bright one is at its core
	Colour light.Ramp // over its life, nought at birth
	Streak float64    // seconds of the path drawn behind it
}

// Particle is one particle at a moment.
type Particle struct {
	ID  int64
	At  view.Vec
	Age float64 // nought at birth, one at death
}

// Alive calls do with every particle alive at t, in the order they were born.
//
// An emitter that cannot throw anything throws nothing: no rate, no life, or
// -- for the two kinds whose size is what they are -- no radius. A spiral of
// no radius used to divide by it, which made every particle's place a NaN and
// drew nothing while saying nothing.
func (e Emitter) Alive(t float64, do func(Particle)) {
	if e.Life <= 0 || e.Rate <= 0 {
		return
	}
	if (e.Kind == Spiral || e.Kind == Drift) && e.Radius <= 0 {
		return // a ring of no radius, a box of no size
	}
	if e.Kind == Drift {
		n := int64(e.Rate * e.Life)
		for id := range n {
			if p, ok := e.Where(id, t); ok {
				do(p)
			}
		}
		return
	}
	first := int64(math.Floor((t-e.Life)*e.Rate)) - 1
	last := int64(math.Floor(t * e.Rate))
	for id := first; id <= last; id++ {
		if p, ok := e.Where(id, t); ok {
			do(p)
		}
	}
}

// born is when particle id was born.
func (e Emitter) born(id int64) float64 {
	return (float64(id) + 0.9*e.hash(id, 0)) / e.Rate
}

// hash is the k-th number particle id was given.
func (e Emitter) hash(id int64, k int64) float64 {
	return noise.Hash2(e.Seed, id, k)
}

// Where is particle id at t, and false if it is not alive then.
func (e Emitter) Where(id int64, t float64) (Particle, bool) {
	if e.Kind == Drift {
		return e.drift(id, t), true
	}
	age := t - e.born(id)
	if age < 0 || age >= e.Life {
		return Particle{}, false
	}
	p := Particle{ID: id, Age: age / e.Life}
	switch e.Kind {
	case Jet:
		p.At = e.jet(id, age)
	case Spiral:
		p.At = e.spiral(id, age)
	default:
		p.At = e.fountain(id, age)
	}
	return p, true
}

// fountain is a thrown particle: a cone, then gravity and drag.
func (e Emitter) fountain(id int64, age float64) view.Vec {
	dir := cone(e.Axis, e.Spread, e.hash(id, 1), e.hash(id, 2))
	v := scale(dir, e.Speed*(0.75+0.5*e.hash(id, 3)))
	if e.Drag <= 0 {
		fall := scale(e.Gravity, age*age/2)
		return add(e.At, add(scale(v, age), fall))
	}
	// with drag k the velocity decays as exp(-k t), and the closed form
	// is what integrating that twice gives
	k := e.Drag
	gone := (1 - math.Exp(-k*age)) / k
	fall := (age - gone) / k
	return add(e.At, add(scale(v, gone), scale(e.Gravity, fall)))
}

// jet is a particle leaving along the axis, one pole or the other, from an
// axis that had turned by however long ago it was born.
func (e Emitter) jet(id int64, age float64) view.Vec {
	axis := unit(e.Axis)
	if e.Wobble != 0 {
		u, v := basis(axis)
		// the turn is read at birth: a particle keeps going the way the
		// axis pointed when it left, so the jet draws the turning as a
		// curve instead of swinging like a stick
		turn := 2 * math.Pi * e.born(id) / math.Max(e.Life*6, 1e-6)
		lean := add(scale(u, math.Cos(turn)), scale(v, math.Sin(turn)))
		axis = unit(add(axis, scale(lean, math.Tan(e.Wobble))))
	}
	if e.hash(id, 4) < 0.5 {
		axis = scale(axis, -1)
	}
	dir := cone(axis, e.Spread, e.hash(id, 1), e.hash(id, 2))
	speed := e.Speed * (0.7 + 0.6*e.hash(id, 3))
	return add(e.At, scale(dir, speed*age))
}

// spiral is a particle falling in from a ring, faster as it gets closer, in
// the way a Keplerian orbit is: angular speed goes as radius to the -3/2.
func (e Emitter) spiral(id int64, age float64) view.Vec {
	n := unit(e.Axis)
	u, v := basis(n)
	r0 := e.Radius * (0.55 + 0.45*e.hash(id, 1))
	inner := math.Min(e.Inner, r0)
	radius := func(s float64) float64 {
		return inner + (r0-inner)*math.Sqrt(math.Max(0, 1-s))
	}
	omega := func(r float64) float64 {
		r = math.Max(r, 1e-6)
		return e.Speed / e.Radius * math.Pow(e.Radius/r, 1.5)
	}
	// the angle is the angular speed summed over the life so far, by
	// Simpson's rule: no closed form survives the inner edge
	s := age / e.Life
	const steps = 12
	h := s / steps
	sum := omega(radius(0)) + omega(radius(s))
	for k := 1; k < steps; k++ {
		w := 2.0
		if k%2 == 1 {
			w = 4
		}
		sum += w * omega(radius(float64(k)*h))
	}
	theta := 2*math.Pi*e.hash(id, 2) + sum*h/3*e.Life
	r := radius(s)
	lift := (e.hash(id, 3) - 0.5) * e.Spread * r
	at := add(scale(u, r*math.Cos(theta)), scale(v, r*math.Sin(theta)))
	return add(e.At, add(at, scale(n, lift)))
}

// drift is a mote in a box, moving along the axis and wrapping at the sides.
func (e Emitter) drift(id int64, t float64) Particle {
	r := math.Max(e.Radius, 1e-6)
	base := view.Vec{
		X: (e.hash(id, 1)*2 - 1) * r,
		Y: (e.hash(id, 2)*2 - 1) * r,
		Z: (e.hash(id, 3)*2 - 1) * r,
	}
	v := scale(unit(e.Axis), e.Speed*(0.6+0.8*e.hash(id, 5)))
	wrap := func(x float64) float64 {
		return math.Mod(math.Mod(x+r, 2*r)+2*r, 2*r) - r
	}
	p := add(base, scale(v, t))
	p = view.Vec{X: wrap(p.X), Y: wrap(p.Y), Z: wrap(p.Z)}
	// a mote's age is only for its colour: it cycles, so the dust
	// twinkles through its ramp rather than all being one shade
	age := math.Mod(t/math.Max(e.Life, 1e-6)+e.hash(id, 4), 1)
	return Particle{ID: id, At: add(e.At, p), Age: age}
}

// cone is a direction within spread radians of axis, spread evenly over the
// cap it makes, from two numbers between nought and one.
func cone(axis view.Vec, spread, h1, h2 float64) view.Vec {
	w := unit(axis)
	u, v := basis(w)
	cosT := 1 - h1*(1-math.Cos(spread))
	sinT := math.Sqrt(math.Max(0, 1-cosT*cosT))
	phi := 2 * math.Pi * h2
	return add(scale(w, cosT), add(scale(u, sinT*math.Cos(phi)),
		scale(v, sinT*math.Sin(phi))))
}

// basis is two unit vectors square to n and to each other.
func basis(n view.Vec) (u, v view.Vec) {
	a := view.Vec{X: 1}
	if math.Abs(n.X) > 0.9 {
		a = view.Vec{Y: 1}
	}
	u = unit(cross(n, a))
	return u, cross(n, u)
}

// add is two vectors summed.
func add(a, b view.Vec) view.Vec {
	return view.Vec{X: a.X + b.X, Y: a.Y + b.Y, Z: a.Z + b.Z}
}

// scale is a vector lengthened or shortened.
func scale(a view.Vec, k float64) view.Vec {
	return view.Vec{X: a.X * k, Y: a.Y * k, Z: a.Z * k}
}

// cross is the vector square to both.
func cross(a, b view.Vec) view.Vec {
	return view.Vec{X: a.Y*b.Z - a.Z*b.Y, Y: a.Z*b.X - a.X*b.Z,
		Z: a.X*b.Y - a.Y*b.X}
}

// unit is a vector of length one, or straight ahead if it has no length.
func unit(a view.Vec) view.Vec {
	l := math.Sqrt(a.X*a.X + a.Y*a.Y + a.Z*a.Z)
	if l == 0 {
		return view.Vec{Z: 1}
	}
	return scale(a, 1/l)
}
