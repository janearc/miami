package view

import (
	"math"
	"time"
)

// Where the sun is, from the time of day.
//
// Not an ephemeris. A real one wants a latitude, a day of the year and an
// equation of time, and would move the sun by a few degrees from what this
// says.
//
// None of that is visible on a floor forty cells wide, and all of it would have
// to be got right to be worth having, so this is the schoolroom version: up at
// six, highest at noon, down at six, and below the horizon in between.
//
// What matters is only that it is continuous, that it crosses the horizon at
// the right hours, and that it is somewhere definite -- because a shadow is
// cast from a position, and a position invented per frame would make the shadow
// jump.

// Sun is where the sun is and how high, at a moment.
type Sun struct {
	At        Vec     // a long way off, in the direction it is shining from
	Elevation float64 // -1 at midnight, 0 at the horizon, 1 at noon
}

// SunAt is the sun at a time of day.
func SunAt(t time.Time) Sun {
	h := float64(t.Hour()) + float64(t.Minute())/60 + float64(t.Second())/3600

	// noon is one, the horizons are nought, midnight is minus one
	el := math.Sin(math.Pi * (h - 6) / 12)

	// east at dawn, west at dusk, swinging through the south. the azimuth
	// runs the whole half circle across the day so the shadow sweeps rather
	// than jumping sides at noon.
	az := math.Pi * (h - 6) / 12

	// far enough away that its rays are near enough parallel, which is what
	// makes the shadow of a sphere the same size as the sphere rather than
	// a cone spreading out behind it
	const far = 60.0
	return Sun{
		Elevation: el,
		At: Vec{
			X: far * math.Cos(az) * -1,
			Y: far * math.Max(el, -0.2),
			Z: far * math.Sin(az) * 0.4,
		},
	}
}

// Light is the sun as the tracer wants it: off when it is below the horizon,
// which costs the floor a shade everywhere and is what night is.
func (s Sun) Light() Light {
	return Light{At: s.At, Off: s.Elevation <= 0}
}

// Up is how far above the horizon the sun is, nought to one, for anything that
// fades with it.
func (s Sun) Up() float64 { return math.Max(0, s.Elevation) }

// Golden is how close the sun is to the horizon while still above it: one at
// the moment it touches, nought by the time it is well up. Sunrise and sunset
// are the same number, which is right -- the light does not know which way it
// is going.
func (s Sun) Golden() float64 {
	if s.Elevation <= 0 {
		return 0
	}
	return math.Max(0, 1-s.Elevation/0.35)
}
