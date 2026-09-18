// Package grade stacks gradients.
//
// libdaffy keeps a gradient in two halves on purpose: a field is position to t,
// a ramp is t to a swatch, and neither knows the other exists. That seam is
// what lets one ramp paint a sun, a sky and a stripe.
//
// What it deliberately does not give you is the next thing up -- several of
// those at once, over each other -- because that is not a gradient, it is a
// picture, and neither library is in the picture business.
//
// This is that next thing, and it is small: a layer is a field, a ramp, a
// second field saying how much of it there is, and a rule for putting it over
// what is underneath.
//
//	Field  where am I          -> t
//	Ramp   what colour is t    -> swatch
//	Alpha  how much of it      -> 0..1
//	Blend  and over what
//
// Alpha being a field rather than a number is the whole of it. A sun with a
// hard edge is a ramp with a soft one and an alpha that steps; the slats cut
// through a sunset are an alpha that stripes; and an effect -- a sweep, a
// pulse, a flicker -- is a layer whose alpha has had the time folded into it.
//
// There is no separate effects mechanism, because an effect and a gradient turn
// out to be the same object.
//
// Mixing is in XYZ, which is linear light. That is not a preference: a Swatch
// is a physical colour, so adding two of them is what adding two lights does,
// and no gamma creeps in anywhere between the ramp and the cell.
package grade

import (
	"image/color"
	"math"

	"github.com/janearc/libdaffy/field"
	"github.com/janearc/libtheme-css/primitives/functions"
	"github.com/janearc/libtheme-css/primitives/swatch"
	"github.com/janearc/libtheme-css/spaces/srgb"

	"github.com/janearc/miami/internal/split"
)

// Blend is how a layer meets what is already painted.
type Blend int

const (
	// Over is paint: the layer hides what is under it, in proportion to
	// alpha.
	Over Blend = iota
	// Add is light: the layer is added to what is under it. Two lamps on
	// one wall are brighter than either, and in XYZ that is a sum and
	// nothing else.
	Add
	// Mul is a filter held in front: gels, shadow, vignette.
	Mul
)

// Layer is one gradient and how much of it there is.
type Layer struct {
	Name  string // reported when a layer contributes nothing; a typo shows
	Field field.Field
	Ramp  functions.Ramp
	Alpha field.Field // nil is fully present
	Blend Blend
	// Gain scales the alpha. Nought means absent and one means as authored,
	// and every layer must say which -- there is no "unset".
	//
	// It read the other way round at first, with nought standing for "no
	// gain given, use it whole". That is a trap and it sprang immediately:
	//
	// a layer whose gain is computed, like a sun's glow fading as it sets,
	// reaches nought at exactly the moment it should vanish and instead
	// comes back at full strength. A sentinel that collides with a
	// legitimate value is not a sentinel.
	Gain float64
}

// Stack is layers in painting order, first at the back.
type Stack []Layer

// At is the colour of the sheet at u, v.
func (s Stack) At(u, v float64) swatch.Swatch {
	var x, y, z float64
	for _, l := range s {
		a := 1.0
		if l.Alpha != nil {
			a = l.Alpha.T(u, v)
		}
		a *= l.Gain
		if a <= 0 {
			continue
		}
		if a > 1 {
			a = 1
		}
		lx, ly, lz := l.Ramp.At(l.Field.T(u, v)).XYZ()
		switch l.Blend {
		case Add:
			x, y, z = x+lx*a, y+ly*a, z+lz*a
		case Mul:
			x = x*(1-a) + x*lx*a
			y = y*(1-a) + y*ly*a
			z = z*(1-a) + z*lz*a
		default: // Over
			x = x*(1-a) + lx*a
			y = y*(1-a) + ly*a
			z = z*(1-a) + lz*a
		}
	}
	return swatch.FromXYZ(x, y, z)
}

// Sheet maps a grid of subpixels onto the unit square the fields read.
//
// The correction lives here and nowhere else. A terminal cell is twice as tall
// as it is wide; a sextant subcell is half a cell across and a third of one
// down, so it is four-thirds taller than it is wide.
//
// Fields are written against a square sheet -- the package says so in as many
// words -- and if nobody squares it up, every circle in every scene is an oval.
// One constant, one place.
type Sheet struct {
	W, H           int     // subpixels
	SubW, SubH     float64 // a subpixel's size in cells
	fitU, fitV, o0 float64
	o1             float64
}

// CellAspect is how much taller a terminal cell is than it is wide.
const CellAspect = 2.0

// Sextants is a sheet drawn on sextant subcells, two across and three down.
func Sextants(cols, rows int) Sheet {
	return newSheet(cols*2, rows*3, 0.5, CellAspect/3)
}

// newSheet works out where the unit square sits on a grid of subpixels.
func newSheet(w, h int, subW, subH float64) Sheet {
	s := Sheet{W: w, H: h, SubW: subW, SubH: subH}
	pw, ph := float64(w)*subW, float64(h)*subH
	long := math.Max(pw, ph)
	// the sheet is centred in the long direction, so a square field lands
	// in the middle of a wide canvas instead of in its corner
	s.fitU, s.fitV = pw/long, ph/long
	s.o0, s.o1 = (1-s.fitU)/2, (1-s.fitV)/2
	return s
}

// Du and Dv are how much of the sheet one subpixel covers. A field that wants a
// line of a given width on the screen needs them: width in world units is the
// wrong unit for a thing the viewer measures in pixels, and it is what makes a
// receding grid alias into a smear at one end and a bar at the other.
func (s Sheet) Du() float64 { return s.fitU / float64(s.W) }

// Dv is the vertical counterpart of Du.
func (s Sheet) Dv() float64 { return s.fitV / float64(s.H) }

// UV is the position of subpixel x, y on the unit sheet.
func (s Sheet) UV(x, y int) (u, v float64) {
	return s.o0 + s.fitU*(float64(x)+0.5)/float64(s.W),
		s.o1 + s.fitV*(float64(y)+0.5)/float64(s.H)
}

// Paint evaluates the stack over the sheet and returns sRGB subpixels, ready
// for the cell fitter. Colours outside sRGB are pulled to the edge of the gamut
// rather than clipped per channel, which would swing the hue.
//
// Rows are painted on every core at once, so a field MUST be safe to ask from
// several goroutines. Every field here is a function of where it is asked
// and of values fixed before painting starts, which is the rule to keep.
func (s Sheet) Paint(st Stack) [][]color.RGBA {
	out := make([][]color.RGBA, s.H)
	split.Rows(s.H, func(lo, hi int) {
		for y := lo; y < hi; y++ {
			row := make([]color.RGBA, s.W)
			for x := 0; x < s.W; x++ {
				u, v := s.UV(x, y)
				row[x] = ToRGBA(st.At(u, v))
			}
			out[y] = row
		}
	})
	return out
}

// ToRGBA is a swatch as a terminal colour, fitted into sRGB.
func ToRGBA(sw swatch.Swatch) color.RGBA {
	c, ok := srgb.FromSwatch(sw)
	if !ok {
		xy := srgb.Gamut.Fit(sw.XY())
		_, luminance, _ := sw.XYZ()
		lum := math.Min(1, luminance)
		c, _ = srgb.FromSwatch(swatch.FromXY(xy, lum))
	}
	q := func(v float64) uint8 {
		return uint8(math.Round(math.Max(0, math.Min(1, v)) * 255))
	}
	return color.RGBA{q(c.R), q(c.G), q(c.B), 255}
}

// Stripes is an alpha that cuts regular gaps, the slats every airbrushed sun of
// 1984 was cut with. Period and duty are in sheet units; lean widens the gaps
// toward the bottom, which is what makes flat slats read as perspective.
func Stripes(period, duty, phase, lean float64) field.Field {
	return field.Func(func(u, v float64) float64 {
		p := period * (1 + lean*v)
		t := math.Mod((v-phase)/math.Max(1e-6, p), 1)
		if t < 0 {
			t++
		}
		return step(t, duty)
	})
}

// Band is an alpha that is 1 inside a soft-edged band of a field's t.
func Band(f field.Field, at, width, soft float64) field.Field {
	return field.Func(func(u, v float64) float64 {
		d := math.Abs(f.T(u, v) - at)
		if d <= width/2 {
			return 1
		}
		if d >= width/2+soft {
			return 0
		}
		return 1 - (d-width/2)/soft
	})
}

// Inside is an alpha that is 1 where a field is below a threshold and falls off
// over soft. A disc is Inside(Radial, 1, small).
func Inside(f field.Field, at, soft float64) field.Field {
	return field.Func(func(u, v float64) float64 {
		t := f.T(u, v)
		if t <= at {
			return 1
		}
		if soft <= 0 || t >= at+soft {
			return 0
		}
		return 1 - (t-at)/soft
	})
}

// Times multiplies alphas: a sun with slats through it is a disc and a stripe.
func Times(fs ...field.Field) field.Field {
	return field.Func(func(u, v float64) float64 {
		t := 1.0
		for _, f := range fs {
			t *= f.T(u, v)
		}
		return t
	})
}

// Flat is a constant alpha, for a layer that is simply present.
func Flat(a float64) field.Field {
	return field.Func(func(u, v float64) float64 { return a })
}

// step is one cycle of a stripe: on for the duty, off for the rest, with a soft
// edge either side so a band does not alias.
func step(t, duty float64) float64 {
	const soft = 0.12
	if t < duty-soft {
		return 1
	}
	if t < duty {
		return (duty - t) / soft
	}
	if t < 1-soft {
		return 0
	}
	return 1 - (1-t)/soft
}

// Contact renders each layer on its own, and the alpha that admits it, as a
// strip of images. Stacking is the one part of this work where looking at the
// result tells you nothing about which layer is wrong: six things add up to a
// wash and the wash does not say who washed it.
//
// So the inspector is part of the package rather than a thing rebuilt by hand
// each time.
func (s Sheet) Contact(st Stack) (
	colour, alpha [][][]color.RGBA, names []string,
) {
	for _, l := range st {
		c := make([][]color.RGBA, s.H)
		a := make([][]color.RGBA, s.H)
		for y := 0; y < s.H; y++ {
			c[y] = make([]color.RGBA, s.W)
			a[y] = make([]color.RGBA, s.W)
			for x := 0; x < s.W; x++ {
				u, v := s.UV(x, y)
				c[y][x] = ToRGBA(l.Ramp.At(l.Field.T(u, v)))
				t := 1.0
				if l.Alpha != nil {
					t = l.Alpha.T(u, v)
				}
				t *= l.Gain
				clamped := math.Max(0, math.Min(1, t))
				g := uint8(math.Round(clamped * 255))
				a[y][x] = color.RGBA{g, g, g, 255}
			}
		}
		colour = append(colour, c)
		alpha = append(alpha, a)
		names = append(names, l.Name)
	}
	return
}
