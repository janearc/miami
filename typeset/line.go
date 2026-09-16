package typeset

import (
	"math"
	"math/rand/v2"

	"miami/geom"
)

// Wobble is the hand-set imperfection: every letter gets its own small tilt,
// baseline shift, size change and spacing error, drawn from a fixed seed so the
// same line renders the same way every time and role grids still diff.
type Wobble struct {
	Tilt  float64 // degrees, +/-
	Shift float64 // fraction of cap height, +/-
	Scale float64 // fraction, +/-
	Space float64 // fraction of cap height added to an advance, +/-
	Seed  uint64
}

// Line is one run of display type.
type Line struct {
	Text      string
	Face      *Face
	CapHeight float64 // physical units: how tall the capitals actually stand
	Tracking  float64 // extra space after each letter, fraction of cap height
	Weight    float64 // stroke adjustment, physical units; negative thins
	Condense  float64 // horizontal squeeze, 0 or 1 is none; painters condensed the key word to make it tall
	Arch      float64 // baseline rise at the centre, physical units; 0 is flat
	Slope     float64 // degrees the whole baseline tips, positive dips right
	// Flat clips round letters' optical overshoot at the baseline and cap
	// line. at print sizes that overshoot makes an O look level with an H;
	// at one subrow it is a tail, and it turned NANOLEAF's O into a Q.
	Flat   bool
	Wobble Wobble
}

// squeeze is the horizontal condensing, or one when none was asked for.
func (l Line) squeeze() float64 {
	if l.Condense <= 0 {
		return 1
	}
	return l.Condense
}

// Width is the line's set width before wobble, for centring.
func (l Line) Width() float64 {
	k := l.squeeze()
	w := 0.0
	rs := []rune(l.Text)
	for i, r := range rs {
		if r == ' ' {
			w += l.space() * k
			continue
		}
		_, adv, err := l.Face.Glyph(r, geom.Vec{}, l.CapHeight)
		if err != nil {
			continue
		}
		w += adv*k + l.Tracking*l.CapHeight
		if i+1 < len(rs) {
			w += l.Face.Kern(r, rs[i+1], l.CapHeight) * k
		}
	}
	return w - l.Tracking*l.CapHeight
}

// space is the gap after each letter, as a fraction of the cap height.
func (l Line) space() float64 {
	_, adv, err := l.Face.Glyph('n', geom.Vec{}, l.CapHeight)
	if err != nil || adv == 0 {
		return 0.35 * l.CapHeight
	}
	return adv
}

// Set lays the line out centred on cx with its baseline at y, and returns one
// shape per letter so a scene can give letters their own roles if it wants.
func (l Line) Set(cx, y float64) []geom.Shape {
	k := l.squeeze()
	rng := rand.New(rand.NewPCG(l.Wobble.Seed, 0x6c69676874))
	jit := func(amp float64) float64 { return (rng.Float64()*2 - 1) * amp }

	w := l.Width()
	x := cx - w/2
	half := w / 2
	tip := math.Tan(l.Slope * math.Pi / 180)
	rs := []rune(l.Text)
	var out []geom.Shape
	for i, r := range rs {
		if r == ' ' {
			x += l.space() * k
			continue
		}
		_, adv, err := l.Face.Glyph(r, geom.Vec{}, l.CapHeight)
		if err != nil {
			continue
		}
		adv *= k
		mid := x + adv/2
		// arch: a parabola through the line ends, peaking at the centre
		u := (mid - cx) / math.Max(half, 1e-9)
		base := y - l.Arch*(1-u*u) + (mid-cx)*tip
		base += jit(l.Wobble.Shift) * l.CapHeight
		// letters on an arch lean with it, as painted lettering does
		lean := 0.0
		if l.Arch != 0 {
			lean = math.Atan(2*l.Arch*u/math.Max(half, 1e-9)) * 180 / math.Pi
		}
		tilt := lean + l.Slope + jit(l.Wobble.Tilt)
		scale := 1 + jit(l.Wobble.Scale)

		glyph, _, _ := l.Face.Glyph(r, geom.Vec{X: x, Y: base}, l.CapHeight)
		centre := geom.Vec{X: mid, Y: base - l.CapHeight/2}
		// condense about the pen origin first so the glyph lands in its
		// own narrowed advance, then size and tilt about its centre
		m := geom.ScaleAbout(geom.Vec{X: x, Y: base}, k, 1).
			Then(geom.ScaleAbout(centre, scale, scale)).
			Then(geom.RotateAbout(centre, tilt))
		var s geom.Shape = glyph.Transform(m)
		if l.Weight != 0 {
			s = geom.Weight(s, l.Weight)
		}
		if l.Flat {
			// last, in page space: clipping before the tilt let the
			// rotated flat bottom dip at one end, which was the
			// same tail by another route
			band := geom.Intersect(
				geom.HalfPlane{C: geom.Vec{X: 0, Y: base}, N: geom.Vec{X: 0, Y: 1}},
				geom.HalfPlane{C: geom.Vec{X: 0, Y: base - l.CapHeight*scale}, N: geom.Vec{X: 0, Y: -1}},
			)
			s = geom.Intersect(s, band)
		}
		out = append(out, s)

		x += adv + l.Tracking*l.CapHeight + jit(l.Wobble.Space)*l.CapHeight
		if i+1 < len(rs) {
			x += l.Face.Kern(r, rs[i+1], l.CapHeight) * k
		}
	}
	return out
}
