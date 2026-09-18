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

// unaim is Aim undone: a world direction turned into the camera's own frame,
// where it looks down positive Z.
func (e Eye) unaim(d Vec) Vec {
	cy, sy := math.Cos(e.Yaw), math.Sin(e.Yaw)
	d = Vec{d.X*cy - d.Z*sy, d.Y, d.X*sy + d.Z*cy}
	cp, sp := math.Cos(e.Pitch), math.Sin(e.Pitch)
	return Vec{d.X, d.Y*cp + d.Z*sp, -d.Y*sp + d.Z*cp}
}

// Ray is the direction through the middle of subpixel x, y of a sheet w by h
// subpixels. It is not unit length: in the camera's frame its Z is one, so a
// distance along it is a depth.
//
// This is the tracer's own projection, stated once. Everything drawn into
// space -- a traced ball, a black hole, a spark -- goes through this and
// Project, so they agree about where a point lands.
func (e Eye) Ray(x, y, w, h int) Vec {
	return e.Through(float64(x)+0.5, float64(y)+0.5, w, h)
}

// Through is Ray at any point of the sheet rather than a subpixel's middle,
// in subpixels from the top left corner: a subpixel spans x to x+1. It is
// how one subpixel is asked about several times, where something inside it
// is finer than it is.
func (e Eye) Through(x, y float64, w, h int) Vec {
	lens := e.Lens
	if lens == 0 {
		lens = 1
	}
	// a subcell is half a column by a third of a row, so on a cell twice
	// as tall as it is wide the sheet is four rows high for every three
	// columns across
	aspect := (float64(h) * 4) / (float64(w) * 3)
	px := x/float64(w) - 0.5
	py := (0.5 - y/float64(h)) * aspect
	return e.Aim(Vec{px / lens, py / lens, 1})
}

// Project is where a point in the world lands on a sheet w by h subpixels, in
// subpixels, and how deep it is. ok is false for a point behind the near
// plane, which has nowhere to land.
//
// Fractional, so a spark between two subpixels can share its light between
// them. The middle of subpixel x is at x, not x+0.5.
func (e Eye) Project(p Vec, w, h int) (x, y, depth float64, ok bool) {
	c := e.unaim(Vec{p.X - e.At.X, p.Y - e.At.Y, p.Z - e.At.Z})
	near := e.Near
	if near <= 0 {
		near = 1e-3
	}
	if c.Z < near {
		return 0, 0, 0, false
	}
	lens := e.Lens
	if lens == 0 {
		lens = 1
	}
	aspect := (float64(h) * 4) / (float64(w) * 3)
	px, py := lens*c.X/c.Z, lens*c.Y/c.Z
	x = (px+0.5)*float64(w) - 0.5
	y = (0.5-py/aspect)*float64(h) - 0.5
	return x, y, c.Z, true
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
	return Scale(f, wide(f, rows), rows)
}

// wide is how many subcells across a sprite of this shape is, drawn rows
// subcells tall.
//
// A daffy pixel is one column by half a row and a sextant subcell is half a
// column by a third, so a sprite pixel is two subcells across and one and a
// half down. Keeping its proportions is therefore rows times the frame's
// width over its height, times two over one and a half.
//
// That last factor was written as two thirds, which is the reciprocal, so
// every billboard in the library has been drawn half as wide as its own
// stencil since the line was written.
func wide(f sprite.Frame, rows int) int {
	fw, fh := float64(f.Width()), float64(f.Height())
	if fh == 0 {
		return 1
	}
	return max(1, int(math.Round(float64(rows)*fw/fh*4/3)))
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
	w := wide(f, rows)
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
//
// Where a sprite lands and how big it is both come from Project, which is the
// tracer's own projection. They used to come from arithmetic written here that
// read the eye's position but not its yaw or pitch, and divided by Lens where
// the tracer multiplies.
//
// So a turned camera swung the traced ball and left the bird where it was, and
// at any lens the two disagreed vertically, because this had no aspect term at
// all. A sprite cannot fly in front of a black hole drawn by a camera it does
// not share.
func Resolve(objs []Object, e Eye, t time.Duration, subW, subH int) []Placed {
	var out []Placed
	for _, o := range objs {
		at, height := o.Extent(t)
		x0, y0, d, ok := e.Project(at, subW, subH)
		if !ok {
			continue // behind the viewer, or in their eye
		}
		// how tall it is on the sheet is where its top projects against
		// where its middle does, so the same lens, aspect and pitch
		// that place it also size it
		top := Vec{at.X, at.Y + height/2, at.Z}
		_, y1, _, ok := e.Project(top, subW, subH)
		if !ok {
			continue
		}
		h := int(2 * math.Abs(y0-y1))
		if h < 1 {
			continue // further away than the sheet can resolve
		}
		f := o.Render(e, t, h)
		w := f.Width()
		if w < 1 || f.Height() == 0 {
			continue
		}
		x := int(math.Round(x0)) - w/2
		y := int(math.Round(y0)) - f.Height()/2
		if x+w < 0 || x >= subW || y+f.Height() < 0 || y >= subH {
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
