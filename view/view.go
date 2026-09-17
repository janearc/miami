// Package view puts objects in space and asks them what they look like.
//
// The idea this rests on is not mine and is worth stating plainly, because it
// is what makes three dimensions cheap here. A sprite already knows what it is.
//
// If it also knows how to present itself from a given direction, then rendering
// a scene is not "intersect geometry, shade fragments" -- it is "ask each
// object what it looks like from here, and compose the answers by alpha".
//
// The alpha channel already does the compositing, and it has been doing it
// since the bulb was lit by a second sprite.
//
// That turns the expensive parts of three dimensions into the cheap parts:
//
//	occlusion   whoever is nearer wins, and alpha says where they are
//	scale       a sprite asked from further away comes back smaller
//	facing      an object with several poses returns the one facing you
//	a ray       walk the objects front to back and stop at the first alpha
//
// The last one is the point. A ray does not need to intersect a surface; it
// needs to find the first object whose alpha is set where it passes. That is a
// lookup, not a solve, and it is why tracing is easier here than in a scene
// made of triangles.
//
// The camera is deliberately the simplest thing that works: the eye sits at the
// origin looking down positive Z, the picture plane is one unit away, and Lens
// is how many picture-plane units across the sheet is. Nothing here rotates
// yet, because nothing has asked to.
package view

import (
	"math"
	"sort"
	"time"

	sprite "github.com/janearc/libdaffy-sprite"
)

// Vec is a position in world units.
type Vec struct{ X, Y, Z float64 }

// Eye is where the picture is seen from.
type Eye struct {
	At   Vec     // where the viewer is
	Lens float64 // sheet width in picture-plane units; smaller is longer
	Near float64 // anything closer than this is behind the viewer
	// Yaw and Pitch turn the view, in radians. Pitch is positive looking
	// up. Only the ray tracer reads them: a billboard has nothing to
	// rotate, and a sprite drawn from eight sides would need its poses
	// re-picked, which is a different job.
	Yaw, Pitch float64
}

// Aim turns a ray direction by the eye's yaw and pitch. Pitch first, about the
// horizontal axis, then yaw about the vertical: the other order rolls the
// horizon, which is the one thing a level camera must not do.
func (e Eye) Aim(d Vec) Vec {
	cp, sp := math.Cos(e.Pitch), math.Sin(e.Pitch)
	d = Vec{d.X, d.Y*cp - d.Z*sp, d.Y*sp + d.Z*cp}
	cy, sy := math.Cos(e.Yaw), math.Sin(e.Yaw)
	return Vec{d.X*cy + d.Z*sy, d.Y, -d.X*sy + d.Z*cy}
}

// HorizonV is where the horizon falls on the sheet, nought at the top and one
// at the bottom, for a sheet of the given aspect.
//
// The sky is painted in sheet coordinates and the floor is traced in world
// ones, so the two have to be told the same thing or the painted horizon and
// the traced one part company the moment the camera tips.
func (e Eye) HorizonV(aspect float64) float64 {
	return 0.5 + math.Tan(e.Pitch)/math.Max(1e-6, aspect)
}

// DefaultEye is a sensible camera for a scene a few units deep.
func DefaultEye() Eye { return Eye{Lens: 1.0, Near: 0.05} }

// Object is anything that can say what it looks like from somewhere.
//
// This is the whole interface, on purpose. A billboard sprite implements it by
// returning its frame; a thing with eight drawn angles implements it by picking
// one; a procedurally drawn thing implements it by drawing. None of them tell
// the renderer how they did it, and the renderer cannot ask.
type Object interface {
	// Extent is where this object is and how tall it stands, in world
	// units.
	Extent(t time.Duration) (at Vec, height float64)

	// Render is what it looks like from the eye, drawn rows subpixels tall.
	// It chooses its own width, because how wide a thing is from a given
	// direction is part of what it looks like from there.
	//
	// The size is asked for rather than fixed because a thing that can draw
	// itself at any size should never be drawn at one size and resampled to
	// another. A billboard has no choice and scales; a ball has the maths
	// in front of it and simply draws smaller.
	Render(e Eye, t time.Duration, rows int) sprite.Frame
}

// Billboard is a sprite that always faces the viewer.
//
// Its own clock is its own: Phase offsets it in its loop, so a flock does not
// beat its wings in unison, which is the single thing that would give away that
// they are all the same animation.
type Billboard struct {
	Frames []sprite.Frame
	At     Vec
	Height float64 // world units, tip to tail
	Phase  time.Duration
}

// Extent is where the billboard hangs.
func (b Billboard) Extent(time.Duration) (Vec, float64) {
	return b.At, b.Height
}

// Render scales the frame it is showing. A drawn sprite has no choice: it was
// drawn at one size and everything else is a resample.
func (b Billboard) Render(_ Eye, t time.Duration, rows int) sprite.Frame {
	f := at(b.Frames, t+b.Phase)
	if f.Height() == 0 {
		return f
	}
	// keep its proportions: a daffy pixel is one column by half a row, and
	// a sextant subcell is half a column by a third, so a sprite pixel is
	// two subcells across and one and a half down
	fw, fh := float64(f.Width()), float64(f.Height())
	w := int(float64(rows) * fw / fh * 2 / 3)
	if w < 1 {
		w = 1
	}
	return Scale(f, w, rows)
}

// Turning is a sprite drawn from several directions, which returns whichever
// was drawn closest to the way it is being looked at.
//
// Poses go anticlockwise from facing the camera. This is how a sprite gets a
// third dimension without a model: the artist draws it from eight sides and the
// viewport picks. Doom did it, and it still costs nothing.
type Turning struct {
	Poses  [][]sprite.Frame // one animation per direction
	At     Vec
	Height float64
	// Heading is the world direction the object faces, measured like the
	// world's own angles: nought is towards positive Z. So an object facing
	// a camera that sits at lower Z has a heading of pi.
	//
	// Pose zero is the object seen head on -- that is, seen from the
	// direction it faces -- and the rest follow anticlockwise.
	Heading float64
	Phase   time.Duration
}

// Extent is where the turning sprite stands.
func (s Turning) Extent(time.Duration) (Vec, float64) { return s.At, s.Height }

// Pose is the index of the drawn direction that best matches the line of sight.
// Exposed because which pose was picked is the thing to check when a turning
// sprite looks wrong, and guessing it from the picture is hopeless.
func (s Turning) Pose(e Eye) int {
	if len(s.Poses) == 0 {
		return 0
	}
	toEye := math.Atan2(e.At.X-s.At.X, e.At.Z-s.At.Z)
	rel := norm(toEye - s.Heading)
	n := len(s.Poses)
	return int(math.Round(rel/(2*math.Pi)*float64(n))+float64(n)) % n
}

// Render picks the pose that faces the eye and scales it.
func (s Turning) Render(e Eye, t time.Duration, rows int) sprite.Frame {
	if len(s.Poses) == 0 {
		return sprite.Frame{}
	}
	f := at(s.Poses[s.Pose(e)], t+s.Phase)
	if f.Height() == 0 {
		return f
	}
	fw, fh := float64(f.Width()), float64(f.Height())
	w := int(float64(rows) * fw / fh * 2 / 3)
	if w < 1 {
		w = 1
	}
	return Scale(f, w, rows)
}

// norm wraps an angle into nought to a full turn.
func norm(a float64) float64 {
	for a < 0 {
		a += 2 * math.Pi
	}
	return math.Mod(a, 2*math.Pi)
}

// Placed is an object resolved for one frame: what to draw, where on the sheet,
// how big, and how far away.
type Placed struct {
	Frame sprite.Frame
	X, Y  int // subpixels, top left
	W, H  int // subpixels
	Depth float64
}

// Resolve asks every object what it looks like and works out where it lands.
//
// Objects behind the viewer, or off the sheet entirely, are dropped here rather
// than drawn and clipped: a flock is mostly off-screen most of the time and
// resampling a sprite nobody will see is the expensive part.
func Resolve(objs []Object, e Eye, t time.Duration, subW, subH int) []Placed {
	var out []Placed
	for _, o := range objs {
		at, height := o.Extent(t)
		d := at.Z - e.At.Z
		if d < e.Near {
			continue // behind the viewer, or in their eye
		}
		// the picture plane is one unit out, so apparent size is size
		// over distance and nothing more
		scale := 1 / (d * e.Lens)
		h := int(height * scale * float64(subH))
		if h < 1 {
			continue // further away than the sheet can resolve
		}
		f := o.Render(e, t, h)
		w := f.Width()
		if w < 1 || f.Height() == 0 {
			continue
		}
		sx := (at.X - e.At.X) * scale
		sy := (at.Y - e.At.Y) * scale
		x := int((0.5+sx)*float64(subW)) - w/2
		y := int((0.5-sy)*float64(subH)) - h/2
		if x+w < 0 || x >= subW || y+h < 0 || y >= subH {
			continue
		}
		out = append(out, Placed{
			Frame: f, X: x, Y: y, W: w, H: f.Height(), Depth: d,
		})
	}
	// far first, so nearer objects paint over them. this is the whole of
	// occlusion when everything composites by alpha.
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Depth > out[j].Depth
	})
	return out
}

// Scale resamples a stencil to a size, by majority vote.
//
// The vote is the same rule as everywhere else here, and for the same reason:
// an alphabet has no midpoint between two roles, so a scaled pixel takes the
// role most of what it covers already was. Averaging would have to invent a
// role, and there is no such thing.
func Scale(f sprite.Frame, w, h int) sprite.Frame {
	sw, sh := f.Width(), f.Height()
	if sw == 0 || sh == 0 || (w == sw && h == sh) {
		return f
	}
	out := make([]string, h)
	for y := 0; y < h; y++ {
		row := make([]byte, w)
		y0, y1 := y*sh/h, (y+1)*sh/h
		if y1 == y0 {
			y1++
		}
		for x := 0; x < w; x++ {
			x0, x1 := x*sw/w, (x+1)*sw/w
			if x1 == x0 {
				x1++
			}
			var votes [256]int
			best, bestN, ink, total := byte('.'), 0, 0, 0
			for py := y0; py < y1 && py < sh; py++ {
				for px := x0; px < x1 && px < sw; px++ {
					total++
					c := f.Art[py][px]
					if c == '.' {
						continue
					}
					ink++
					votes[c]++
					if votes[c] > bestN {
						best, bestN = c, votes[c]
					}
				}
			}
			// a shrinking sprite keeps its silhouette while any of
			// it is ink: requiring a majority would dissolve a thin
			// wing into nothing long before the bird got small
			// enough to stop mattering
			if ink*3 >= total {
				row[x] = best
			} else {
				row[x] = '.'
			}
		}
		out[y] = string(row)
	}
	return sprite.Frame{Art: out, Hold: f.Hold}
}

// at is the frame a looping animation shows at a time.
func at(frames []sprite.Frame, t time.Duration) sprite.Frame {
	if len(frames) == 0 {
		return sprite.Frame{}
	}
	var total time.Duration
	for _, f := range frames {
		total += hold(f)
	}
	if total <= 0 {
		return frames[0]
	}
	t %= total
	if t < 0 {
		t += total
	}
	for _, f := range frames {
		if t < hold(f) {
			return f
		}
		t -= hold(f)
	}
	return frames[len(frames)-1]
}

// hold is a frame's duration, or the default when the stream did not say.
func hold(f sprite.Frame) time.Duration {
	if f.Hold <= 0 {
		return 100 * time.Millisecond
	}
	return f.Hold
}

// seconds turns the tracer's float seconds back into a duration, for the parts
// of the library that were written against one.
func seconds(t float64) time.Duration {
	return time.Duration(t * float64(time.Second))
}
