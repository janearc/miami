package light

import (
	"image/color"
	"math"

	"github.com/janearc/miami/internal/split"
)

// Bringing light down to what a display can show.
//
// Two things are out of reach and they are handled the same way. Light brighter
// than white, and colour more saturated than the three lamps can mix, both
// have their colour pulled toward a grey of the same luminance -- only as far
// as it takes to fit, and never further.
//
// That is what hot things do to the eye and to film. The middle of a flame is
// white and its edge is orange, and a disk drawn in orange with a heart ten
// times too bright comes out white in the middle for free.
//
// Clipping each channel on its own is the other way, and it is wrong. Clip an
// orange ten times too bright one channel at a time and red and green both hit
// one while blue stays near nought: it turns yellow, which no fire is.
//
// Luminance past one is white and cannot be anything else.

// RGBA is the frame as sRGB subpixels for the cell fitter, after multiplying
// every light by exposure.
func (b *Buffer) RGBA(exposure float64) [][]color.RGBA {
	out := make([][]color.RGBA, b.H)
	split.Rows(b.H, func(lo, hi int) {
		for y := lo; y < hi; y++ {
			row := make([]color.RGBA, b.W)
			for x := range row {
				row[x] = Display(b.At(x, y).Scale(exposure))
			}
			out[y] = row
		}
	})
	return out
}

// Display is one light as a terminal colour.
func Display(c RGB) color.RGBA {
	c = fit(c)
	return color.RGBA{encode(c.R), encode(c.G), encode(c.B), 255}
}

// FromRGBA is Display undone: a colour a colourway chose, as light.
//
// A sprite's roles are dressed in colours picked to be looked at, so they are
// display colours, and light is what the frame is made of. Anything drawn into
// the buffer has to arrive in the same units as the rest, or a bird over a
// black hole is lit by a different sun than the disk.
func FromRGBA(c color.RGBA) RGB {
	return RGB{decode[c.R], decode[c.G], decode[c.B]}
}

// decode is the sRGB curve backwards, one entry a byte.
var decode = func() [256]float64 {
	var t [256]float64
	for i := range t {
		v := float64(i) / 255
		if v <= 0.04045 {
			t[i] = v / 12.92
		} else {
			t[i] = math.Pow((v+0.055)/1.055, 2.4)
		}
	}
	return t
}()

// fit pulls a light toward its own grey until every channel is between nought
// and one.
func fit(c RGB) RGB {
	y := c.Y()
	if !(y > 0) { // NaN too: nothing worth showing
		return RGB{}
	}
	if y >= 1 {
		return RGB{1, 1, 1}
	}
	hi := math.Max(c.R, math.Max(c.G, c.B))
	lo := math.Min(c.R, math.Min(c.G, c.B))
	s := 1.0
	if hi > 1 {
		s = math.Min(s, (1-y)/(hi-y))
	}
	if lo < 0 {
		s = math.Min(s, y/(y-lo))
	}
	if s >= 1 {
		return c
	}
	grey := RGB{y, y, y}
	return grey.Mix(c, s)
}

// encode is sRGB's curve, from a table: the power in it is the one expensive
// thing left per subpixel once everything else is arithmetic.
func encode(v float64) uint8 {
	if !(v > 0) {
		return 0
	}
	if v >= 1 {
		return 255
	}
	return curve[int(v*float64(len(curve)-1)+0.5)]
}

// curve is the sRGB transfer, sampled finely enough that neighbouring entries
// never differ by more than one step at the dark end, where it is steepest.
var curve = func() []uint8 {
	const n = 1 << 14
	t := make([]uint8, n)
	for i := range t {
		v := float64(i) / (n - 1)
		if v <= 0.0031308 {
			v *= 12.92
		} else {
			v = 1.055*math.Pow(v, 1/2.4) - 0.055
		}
		t[i] = uint8(math.Round(v * 255))
	}
	return t
}()
