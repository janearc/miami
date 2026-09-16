package grade

import (
	"math"

	"github.com/janearc/libtheme-css/primitives/functions"
	"github.com/janearc/libtheme-css/primitives/swatch"
	"github.com/janearc/libtheme-css/spaces/ok"
	"github.com/janearc/libtheme-css/spaces/srgb"
)

// Saying a colour, and saying a ramp, in the shortest form that is still
// honest.
//
// A stack is mostly stops, and a stop written out longhand is four lines of
// struct for two facts. That is not a readability problem but an accuracy one.
//
// A layer whose stops take a page cannot be read as a single gradient, and a
// gradient that cannot be read as one gets its middle stop wrong unnoticed.
//
// So the two things every layer says get one call each.
//
// What they must not do is skip the conversion: a hex string is sRGB, and sRGB
// is not a space you can average in. Both go through the swatch, which is
// linear light, and that is why a gold-to-blue run does not go through mud.

// Swatch is a hex string as a physical colour. No leading hash and no short
// form -- six hex digits, which is what a theme file writes.
func Swatch(s string) swatch.Swatch {
	var r, g, b uint8
	for i, c := range []byte(s) {
		v := byte(0)
		switch {
		case c >= '0' && c <= '9':
			v = c - '0'
		case c >= 'a' && c <= 'f':
			v = c - 'a' + 10
		}
		switch i / 2 {
		case 0:
			r = r<<4 | v
		case 1:
			g = g<<4 | v
		case 2:
			b = b<<4 | v
		}
	}
	return srgb.RGB{R: float64(r) / 255, G: float64(g) / 255,
		B: float64(b) / 255}.Swatch()
}

// Ramp is a ramp written as alternating positions and hex colours:
//
//	Ramp(0.0, "120631", 0.55, "c63f6b", 1.0, "ffd08a")
//
// The alternation is checked by nothing, which is the price of it reading like
// a gradient. A caller that gets the pairing wrong gets a panic on the type
// assertion, at the line that wrote it.
//
// That is the failure worth having: the other one is a picture that is quietly
// the wrong colour and says nothing about it.
func Ramp(stops ...any) functions.Ramp {
	var s []functions.Stop
	for i := 0; i < len(stops); i += 2 {
		s = append(s, functions.Stop{At: stops[i].(float64),
			Swatch: Swatch(stops[i+1].(string))})
	}
	return functions.New(ok.Mix, s...)
}

// Hash2 is a number between nought and one from a pair of them, the same every
// time it is asked.
//
// Stars, grain and speckle all want something that looks unplanned and is not
// random: a frame that draws its stars from a random source rearranges the sky
// sixty times a second. This is the arithmetic version of that -- no state, no
// seed, and the same answer for the same point for ever.
func Hash2(x, y float64) float64 {
	s := math.Sin(x*127.1+y*311.7) * 43758.5453
	return s - math.Floor(s)
}
