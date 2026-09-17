package particles

import (
	"math"
	"testing"

	"github.com/janearc/miami/light"
	"github.com/janearc/miami/view"
)

// A moment is the same whenever it is asked about, and whatever was asked
// before it. This is the property the whole package is built for: a stepped
// system passes the first half and fails the second.
func TestAMomentIsAFunctionOfTime(t *testing.T) {
	for _, k := range []Kind{Fountain, Jet, Spiral, Drift} {
		e := Emitter{Kind: k, Seed: 3, Axis: view.Vec{Y: 1}, Rate: 40,
			Life: 2, Speed: 1.5, Spread: 0.4, Radius: 2, Inner: 0.3,
			Gravity: view.Vec{Y: -2}, Wobble: 0.2}
		snap := func(at float64) []Particle {
			var ps []Particle
			e.Alive(at, func(p Particle) { ps = append(ps, p) })
			return ps
		}
		first := snap(5.25)
		snap(1)
		snap(9.5)
		again := snap(5.25)
		if len(first) == 0 || len(first) != len(again) {
			t.Fatalf("kind %d: %d particles, then %d", k, len(first),
				len(again))
		}
		for i := range first {
			if first[i] != again[i] {
				t.Fatalf("kind %d: particle %d moved between "+
					"asks: %+v then %+v", k, i, first[i], again[i])
			}
		}
	}
}

// An emitter holds rate times life particles at once, give or take the ones
// being born and dying this instant.
func TestAnEmitterHoldsRateTimesLife(t *testing.T) {
	e := Emitter{Kind: Fountain, Axis: view.Vec{Y: 1}, Rate: 120,
		Life: 1.5, Speed: 1}
	for _, at := range []float64{0, 0.7, 30} {
		n := 0
		e.Alive(at, func(Particle) { n++ })
		if math.Abs(float64(n)-180) > 3 {
			t.Errorf("t=%v: %d alive, want about 180", at, n)
		}
	}
}

// Straight up with no drag, a particle is as high as it will ever be when
// gravity has taken all its speed, and that height is v squared over 2g.
func TestAFountainPeaksWhereBallisticsSays(t *testing.T) {
	e := Emitter{Kind: Fountain, Axis: view.Vec{Y: 1}, Rate: 10, Life: 5,
		Speed: 4, Gravity: view.Vec{Y: -9.8}}
	id := int64(7)
	speed := e.Speed * (0.75 + 0.5*e.hash(id, 3))
	peak := speed / 9.8
	p, ok := e.Where(id, e.born(id)+peak)
	if !ok {
		t.Fatal("not alive at its peak")
	}
	if want := speed * speed / (2 * 9.8); math.Abs(p.At.Y-want) > 1e-9 {
		t.Fatalf("peak %v, want %v", p.At.Y, want)
	}
}

// With drag, the closed form agrees with stepping the same motion in small
// steps, so the formula is the physics it claims to be.
func TestDragMatchesStepping(t *testing.T) {
	e := Emitter{Kind: Fountain, Axis: view.Vec{X: 1, Y: 2}, Rate: 10,
		Life: 5, Speed: 3, Spread: 0.3, Drag: 0.8,
		Gravity: view.Vec{Y: -4}}
	id := int64(3)
	dir := cone(e.Axis, e.Spread, e.hash(id, 1), e.hash(id, 2))
	v := scale(dir, e.Speed*(0.75+0.5*e.hash(id, 3)))
	var p view.Vec
	const dt = 1e-5
	for s := 0.0; s < 2; s += dt {
		acc := add(e.Gravity, scale(v, -e.Drag))
		v = add(v, scale(acc, dt))
		p = add(p, scale(v, dt))
	}
	got, _ := e.Where(id, e.born(id)+2)
	if d := math.Hypot(got.At.X-p.X, got.At.Y-p.Y); d > 1e-3 {
		t.Fatalf("closed form %v, stepped %v", got.At, p)
	}
}

// A spiral's particle only ever falls inward, and goes round faster as it
// does.
func TestASpiralFallsInAndSpeedsUp(t *testing.T) {
	e := Emitter{Kind: Spiral, Axis: view.Vec{Y: 1}, Rate: 10, Life: 4,
		Speed: 1, Radius: 5, Inner: 0.5}
	id := int64(2)
	born := e.born(id)
	lastR, lastA, lastStep := math.Inf(1), 0.0, 0.0
	for i := range 20 {
		p, ok := e.Where(id, born+float64(i)*0.19)
		if !ok {
			t.Fatalf("step %d: dead", i)
		}
		r := math.Hypot(p.At.X, p.At.Z)
		a := math.Atan2(p.At.Z, p.At.X)
		if r > lastR+1e-9 {
			t.Fatalf("step %d: radius rose from %v to %v", i, lastR, r)
		}
		if i > 0 {
			// how far round, whichever way round it goes
			step := math.Abs(math.Remainder(a-lastA, 2*math.Pi))
			if i > 1 && step < lastStep-1e-9 {
				t.Fatalf("step %d: turned %v after %v", i, step,
					lastStep)
			}
			lastStep = step
		}
		lastR, lastA = r, a
	}
}

// Dust stays in its box, and a jet sends to both poles.
func TestDriftStaysInItsBoxAndJetsGoBothWays(t *testing.T) {
	d := Emitter{Kind: Drift, At: view.Vec{Z: 5}, Axis: view.Vec{X: 1},
		Rate: 100, Life: 3, Speed: 2, Radius: 1.5}
	d.Alive(123.4, func(p Particle) {
		if math.Abs(p.At.X) > 1.5 || math.Abs(p.At.Y) > 1.5 ||
			math.Abs(p.At.Z-5) > 1.5 {
			t.Fatalf("mote %d outside its box: %v", p.ID, p.At)
		}
	})
	j := Emitter{Kind: Jet, Axis: view.Vec{Y: 1}, Rate: 200, Life: 1,
		Speed: 3, Spread: 0.05}
	up, down := 0, 0
	j.Alive(10, func(p Particle) {
		if p.At.Y > 0 {
			up++
		} else {
			down++
		}
	})
	if up < 60 || down < 60 {
		t.Fatalf("jet: %d up and %d down", up, down)
	}
}

// Drawing adds light where the particles are, and nothing where something
// nearer is already drawn.
func TestDrawAddsLightAndRespectsDepth(t *testing.T) {
	eye := view.Eye{Lens: 1}
	e := Emitter{Kind: Drift, At: view.Vec{Z: 4}, Axis: view.Vec{X: 1},
		Rate: 400, Life: 1, Speed: 0.1, Radius: 1, Size: 0.01,
		Light: 1, Colour: light.HexRamp(0.0, "ffffff", 1.0, "ffffff")}
	open := light.New(120, 90)
	e.Draw(open, eye, 2)
	blocked := light.New(120, 90)
	for y := range 90 {
		for x := range 120 {
			blocked.SetDepth(x, y, 1) // a wall, nearer than the dust
		}
	}
	e.Draw(blocked, eye, 2)
	var lit, hidden float64
	for y := range 90 {
		for x := range 120 {
			lit += open.At(x, y).Y()
			hidden += blocked.At(x, y).Y()
		}
	}
	if lit <= 0 {
		t.Fatal("drawing added no light")
	}
	if hidden != 0 {
		t.Fatalf("dust behind a wall added %v", hidden)
	}
}
