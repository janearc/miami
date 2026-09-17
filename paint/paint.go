package paint

import (
	"image/color"
	"math"

	"github.com/janearc/libtheme-css/spaces/ok"
)

// Stop is one colour at one position along a ramp.
type Stop struct {
	At float64
	C  color.RGBA
}

// Lerp is two colours mixed, t of the way from the first to the second.
func Lerp(a, b color.RGBA, t float64) color.RGBA {
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	f := func(x, y uint8) uint8 {
		return uint8(float64(x) + (float64(y)-float64(x))*t)
	}
	return color.RGBA{f(a.R, b.R), f(a.G, b.G), f(a.B, b.B), 255}
}

// Ramp is the colour t of the way along a run of stops.
func Ramp(stops []Stop, t float64) color.RGBA {
	if t <= stops[0].At {
		return stops[0].C
	}
	for i := 1; i < len(stops); i++ {
		if t <= stops[i].At {
			s := (t - stops[i-1].At) / (stops[i].At - stops[i-1].At)
			return Lerp(stops[i-1].C, stops[i].C, s)
		}
	}
	return stops[len(stops)-1].C
}

// Radial is a ramp laid out by distance from a centre, as a fraction of r.
func Radial(cx, cy, r float64, stops []Stop) func(x, y float64) color.RGBA {
	return func(x, y float64) color.RGBA {
		dx, dy := x-cx, y-cy
		return Ramp(stops, math.Sqrt(dx*dx+dy*dy)/r)
	}
}

// Screen lays light over a colour: it brightens darks far more than lights and
// can never exceed white, which is how glow behaves on a real surface.
func Screen(c, light color.RGBA, amount float64) color.RGBA {
	if amount <= 0 {
		return c
	}
	if amount > 1 {
		amount = 1
	}
	f := func(a, b uint8) uint8 {
		x, y := float64(a)/255, float64(b)/255*amount
		return uint8(255 * (1 - (1-x)*(1-y)))
	}
	return color.RGBA{
		f(c.R, light.R), f(c.G, light.G), f(c.B, light.B), 255,
	}
}

// Luminance is WCAG 2.1 relative luminance. the 0.03928 threshold is copied
// from gaggle's internal/bench/contrast.go, which ~/.claude/legibility.md names
// as the reference, so these numbers match the ones an auditor there gets.
func Luminance(c color.RGBA) float64 {
	lin := func(v uint8) float64 {
		s := float64(v) / 255
		if s <= 0.03928 {
			return s / 12.92
		}
		return math.Pow((s+0.055)/1.055, 2.4)
	}
	return 0.2126*lin(c.R) + 0.7152*lin(c.G) + 0.0722*lin(c.B)
}

// Contrast is the WCAG ratio between two colours, always >= 1.
func Contrast(a, b color.RGBA) float64 {
	la, lb := Luminance(a), Luminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

// HSL returns hue (0-1), saturation and lightness; same arithmetic as gaggle.
func HSL(c color.RGBA) (h, s, l float64) {
	r, g, b := float64(c.R)/255, float64(c.G)/255, float64(c.B)/255
	mx, mn := math.Max(r, math.Max(g, b)), math.Min(r, math.Min(g, b))
	l = (mx + mn) / 2
	if mx == mn {
		return 0, 0, l
	}
	d := mx - mn
	if l > 0.5 {
		s = d / (2 - mx - mn)
	} else {
		s = d / (mx + mn)
	}
	switch mx {
	case r:
		h = (g - b) / d
		if g < b {
			h += 6
		}
	case g:
		h = (b-r)/d + 2
	default:
		h = (r-g)/d + 4
	}
	return h / 6, s, l
}

// HueApart is the circular distance between two hues in degrees, 0 to 180.
func HueApart(a, b color.RGBA) float64 {
	ha, _, _ := HSL(a)
	hb, _, _ := HSL(b)
	d := math.Abs(ha-hb) * 360
	if d > 180 {
		d = 360 - d
	}
	return d
}

// Hex is a hex colour as a terminal colour. Six digits, no hash.
func Hex(s string) color.RGBA {
	var v [3]uint8
	for i := 0; i < len(s) && i < 6; i++ {
		d := byte(0)
		switch c := s[i]; {
		case c >= '0' && c <= '9':
			d = c - '0'
		case c >= 'a' && c <= 'f':
			d = c - 'a' + 10
		case c >= 'A' && c <= 'F':
			d = c - 'A' + 10
		}
		v[i/2] = v[i/2]<<4 | d
	}
	return color.RGBA{v[0], v[1], v[2], 255}
}

// Blend is two hex colours mixed in oklab, which is the library's own line
// between two colours and the reason a gold-to-blue run does not go through
// mud on the way.
//
// It said this before and did not do it -- it lerped the sRGB bytes, which is
// the mud -- while a second copy of the same function elsewhere did it
// properly. Two implementations of one name is how that survives: nobody
// compares them, because nobody knows there are two.
func Blend(a, b string, t float64) color.RGBA {
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	return RGBA(ok.FromSwatch(ok.Mix.Mix(
		Lab(Hex(a)).Swatch(), Lab(Hex(b)).Swatch(), t)))
}
