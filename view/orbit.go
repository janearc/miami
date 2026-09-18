package view

import "math"

// Turning a camera about a point, which is what a hand dragging on a picture
// means.
//
// A camera here is a place and two angles, and the angles already say which
// way it looks. So an orbit does not integrate a position: it moves the
// angles, keeps the distance, and works the place out from those.
//
// The camera is aimed at the pivot by construction rather than re-aimed
// after, and the distance cannot drift because nothing ever adds to it.
//
// The pivot is the point on the camera's line of sight nearest the middle of
// the scene. A camera looking at a black hole orbits the black hole; one
// looking past it orbits the point it is looking past, which is the thing the
// hand is holding.

// pole is how near the straight-up and straight-down a camera may be turned.
// At the pole the way the picture is turned stops being defined -- every
// direction is the same direction -- and a drag through it flips the horizon.
const pole = math.Pi/2 - 1e-3

// closest is how near a camera may be dollied to its pivot. A dolly is a
// multiply, so it can never reach the pivot by itself; this is for the hand
// that keeps turning the wheel.
const closest = 1e-3

// Forward is the direction the eye looks, as a unit vector.
func (e Eye) Forward() Vec { return e.Aim(Vec{Z: 1}) }

// far is a vector's length.
func far(a Vec) float64 { return math.Sqrt(dot(a, a)) }

// Pivot is the point this eye turns about: the point on its line of sight
// nearest the middle of the scene.
//
// A camera looking away from the middle has no such point in front of it, and
// would otherwise orbit something behind its own shoulder. It is given a pivot
// the distance it already is from the middle, straight ahead, so a drag turns
// it about what it can see.
func (e Eye) Pivot() Vec {
	f := e.Forward()
	// how far along the line of sight the nearest point to the origin is
	t := -dot(e.At, f)
	if t <= closest {
		t = far(e.At)
		if t <= closest {
			t = 1
		}
	}
	return add(e.At, scale(f, t))
}

// Away is how far the eye is from a pivot.
func (e Eye) Away(pivot Vec) float64 { return far(sub(e.At, pivot)) }

// Turned is the eye turned about a pivot: az radians round, el radians up,
// keeping its distance and still looking at the pivot.
//
// The elevation is held short of the poles. A camera taken over the top comes
// down the other side with the horizon upside down, and a hand that drags too
// far expects to stop, not to be turned over.
func (e Eye) Turned(pivot Vec, az, el float64) Eye {
	r := e.Away(pivot)
	if r <= closest {
		r = closest
	}
	e.Yaw += az
	e.Pitch = math.Max(-pole, math.Min(pole, e.Pitch+el))
	e.At = sub(pivot, scale(e.Forward(), r))
	return e
}

// Dollied is the eye moved along its line of sight, by a factor: less than one
// is toward the pivot and more is away.
//
// Multiplied rather than added, so every notch of the wheel covers the same
// share of what is left and the camera never arrives at the pivot. A wheel
// held down for a minute ends up very close and still outside.
func (e Eye) Dollied(pivot Vec, factor float64) Eye {
	if !(factor > 0) {
		return e
	}
	r := e.Away(pivot) * factor
	if r < closest {
		r = closest
	}
	e.At = sub(pivot, scale(e.Forward(), r))
	return e
}

// Drag is a hand moving across the picture, in subpixels, turned into an
// orbit: how far round and how far up the camera goes for it.
//
// A drag across the whole width is half a turn, which is the figure that makes
// a hand feel as though it is holding the thing rather than nudging it.
//
// Up and down is the same radians per subpixel rather than the same fraction
// of the height: a cell is taller than it is wide, and the alternative is a
// drag that climbs faster than it turns.
func Drag(dx, dy, w int) (az, el float64) {
	if w <= 0 {
		return 0, 0
	}
	per := math.Pi / float64(w)
	return float64(dx) * per, float64(dy) * per
}
