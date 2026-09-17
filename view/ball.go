package view

import (
	"math"
	"time"

	sprite "github.com/janearc/libdaffy-sprite"

	"github.com/janearc/miami/texture"
)

// A ball, drawn by solving for its surface rather than by drawing it.
//
// A billboard has one appearance and a Turning sprite has as many as somebody
// drew. Neither will do for a patterned ball: you cannot draw every facet of a
// sphere, and picking the nearest of eight poses would make the pattern jump.
//
// So this one is not drawn at all.
//
// For every pixel inside the disc, the point on the sphere facing the viewer is
// known exactly -- the z of a unit sphere is what is left of one when you take
// off x squared and y squared -- and that point, turned back through the ball's
// own rotation, says where on the surface you are looking.
//
// The pattern is then a function of latitude and longitude. Every facet is
// known because none of them were stored.
//
// This is also, exactly, the first half of a ray tracer. Finding which point of
// a sphere a pixel sees IS the ray-sphere intersection; the only thing not
// being done here is following the ray onwards after it lands. When tracing
// arrives, this function does not change -- it gets a second caller.
//
// The pattern is four quadrants of longitude, each in its own pair of roles and
// its own check size. One check on a plain ball tells you nothing about which
// way it is turning; four different quarters tell you at a glance, which is the
// whole reason for them.
type Ball struct {
	At     Vec
	Radius float64       // world units
	Spin   Vec           // turns per second about each axis
	Tilt   Vec           // radians at t = 0
	Phase  time.Duration // offset into its own rotation

	// Texture is what it is checked in; the four quadrants if nil.
	Texture texture.Texture
}

// Extent is where the ball is and how tall it stands.
func (b Ball) Extent(time.Duration) (Vec, float64) { return b.At, b.Radius * 2 }

// quadrants are the four longitude quarters: the two roles each is checked in,
// and how many checks it carries. Different colours tell you which face you are
// looking at; different check sizes tell you even in one colourway that binds
// two of them alike.
var quadrants = [4]struct {
	a, b  byte
	steps float64
}{
	// each pair has to contrast *within* the quadrant as well as against
	// the other three. the first try used beak-base and beak-band, which
	// most colourways bind to two shades of the same gold, and that quarter
	// came out as a flat orange patch with no check in it at all.
	{'K', 'W', 4}, // dark against light
	{'D', 'V', 6}, // near-black against lilac
	{'B', 'X', 3}, // gold against cream
	{'O', 'R', 5}, // pink against yellow
}

// Render draws the ball at the size asked for.
func (b Ball) Render(_ Eye, t time.Duration, rows int) sprite.Frame {
	if rows < 1 {
		rows = 1
	}
	// a sprite pixel is one column by half a row and a subcell is half a
	// column by a third, so a circle needs four columns for every three
	// rows. this is the only place the ball knows about the shape of a
	// cell.
	cols := rows * 4 / 3
	if cols < 1 {
		cols = 1
	}

	inv := b.Spin_(t)

	art := make([]string, rows)
	for r := 0; r < rows; r++ {
		row := make([]byte, cols)
		// the pixel's centre, in units of the ball's radius
		ny := (float64(r)+0.5)/float64(rows)*2 - 1
		for c := 0; c < cols; c++ {
			nx := (float64(c)+0.5)/float64(cols)*2 - 1
			d := nx*nx + ny*ny
			if d > 1 {
				// off the disc: the silhouette, for free
				row[c] = '.'
				continue
			}
			// the front of the sphere at this pixel is its own
			// normal
			row[c] = surfaceAt(inv, Vec{nx, -ny, math.Sqrt(1 - d)})
		}
		art[r] = string(row)
	}
	return sprite.Frame{Art: art}
}

// Spin is the inverse of the ball's rotation at a moment, which is what a
// surface lookup needs: the question is never "where does this surface point
// go" but "which surface point is showing here".
func (b Ball) Spin_(t time.Duration) mat {
	s := (t + b.Phase).Seconds()
	return rotation(
		-(b.Tilt.X + 2*math.Pi*b.Spin.X*s),
		-(b.Tilt.Y + 2*math.Pi*b.Spin.Y*s),
		-(b.Tilt.Z + 2*math.Pi*b.Spin.Z*s),
	)
}

// Surface is the role at a point on the ball, given the outward unit normal
// there in world space.
//
// This is the function the doc comment above promised would not change when
// tracing arrived, and it has not: Render calls it with the normal it derives
// from a pixel's place in the disc, and the ray tracer calls it with the normal
// at the point a ray actually struck.
//
// Same maths, two callers, and the pattern cannot disagree between the two
// pictures because there is only one of it.
func (b Ball) Surface(n Vec, t time.Duration) byte {
	return surfaceAt(b.Spin_(t), n)
}

// surfaceAt is the role at a point on the ball, given the normal there and the
// rotation to undo. Render and the ray tracer both call it.
func surfaceAt(inv mat, n Vec) byte {
	q := inv.apply(n)
	lat := math.Asin(clamp(q.Y, -1, 1))
	lon := math.Atan2(q.X, q.Z)
	quad := quadrants[int(math.Floor((lon+math.Pi)/(math.Pi/2)))&3]
	k := quad.steps
	// plain bands of latitude and longitude, which means the checks
	// converge at the poles. that is not an artefact to correct: it is what
	// the seams on a real ball do, and it is the clearest signal of which
	// way the thing is tipped.
	u := math.Floor(lon / (math.Pi / k))
	v := math.Floor(lat / (math.Pi / k))
	if int(u+v)&1 == 0 {
		return quad.a
	}
	return quad.b
}

// mat is a rotation, three rows of three.
type mat [3][3]float64

// apply turns a vector through the matrix.
func (m mat) apply(v Vec) Vec {
	return Vec{
		m[0][0]*v.X + m[0][1]*v.Y + m[0][2]*v.Z,
		m[1][0]*v.X + m[1][1]*v.Y + m[1][2]*v.Z,
		m[2][0]*v.X + m[2][1]*v.Y + m[2][2]*v.Z,
	}
}

// rotation is the three axes applied in a fixed order. Which order hardly
// matters for a ball nobody is aiming, but it has to be written down or the
// same numbers give a different picture between one call and the next.
func rotation(x, y, z float64) mat {
	sx, cx := math.Sin(x), math.Cos(x)
	sy, cy := math.Sin(y), math.Cos(y)
	sz, cz := math.Sin(z), math.Cos(z)
	return mat{
		{cy*cz + sy*sx*sz, cx * sz, -sy*cz + cy*sx*sz},
		{-cy*sz + sy*sx*cz, cx * cz, sy*sz + cy*sx*cz},
		{sy * cx, -sx, cy * cx},
	}
}

// clamp holds a number between two bounds.
func clamp(v, lo, hi float64) float64 { return math.Max(lo, math.Min(hi, v)) }

// Quadrants is the ball's own pattern, written in the texture vocabulary.
//
// The same four quarters, each its own pair of roles and its own check size.
// It is here rather than inside the sphere because it is a pattern and not a
// shape: a cylinder or a floor can wear it, and a sphere can wear something
// else.
//
// The phases are the sphere's angular convention made explicit. Longitude runs
// from a seam and latitude from the equator, so a check of k squares per half
// turn lands at floor(2ku - k) across and floor(kv - k/2) up. Writing those
// offsets down is what let the pattern leave the sphere without moving.
func Quadrants() texture.Texture {
	of := make([]texture.Texture, len(quadrants))
	for i, q := range quadrants {
		k := q.steps
		of[i] = texture.Checker{A: q.a, B: q.b,
			U: 2 * k, V: k, PhaseU: -k, PhaseV: -k / 2}
	}
	return texture.Bands{Of: of}
}

// SphereUV is where a point on a sphere is, as a texture wants it: u around
// from the seam, v up from the south pole, both nought to one.
//
// Exported because it is the contract between this shape and any texture, and a
// pattern written for a sphere has to be able to read it.
func SphereUV(n Vec) (u, v float64) {
	lat := math.Asin(clamp(n.Y, -1, 1))
	lon := math.Atan2(n.X, n.Z)
	return (lon + math.Pi) / (2 * math.Pi), lat/math.Pi + 0.5
}

// ballTexture is the pattern a ball wears unless it is given another. Built
// once: it is the same four checkers every time and a sphere asks it per
// subpixel.
var ballTexture = Quadrants()

// Hit is where a ray meets the ball, and the role of its surface there.
func (b Ball) Hit(r Ray, t float64, near, far float64) (float64, byte, bool) {
	d, ok := HitSphere(r, b.At, b.Radius, near, far)
	if !ok {
		return 0, 0, false
	}
	p := add(r.At, scale(r.Dir, d))
	n := b.Spin_(seconds(t)).apply(unit(sub(p, b.At)))
	tex := b.Texture
	if tex == nil {
		tex = ballTexture
	}
	u, v := SphereUV(n)
	return d, tex.At(u, v), true
}

// Blocks says whether the ball stands in the way of a light. The pattern on it
// does not matter to a shadow, so this asks the cheaper question.
func (b Ball) Blocks(r Ray, near, far float64) bool {
	_, ok := HitSphere(r, b.At, b.Radius, near, far)
	return ok
}
