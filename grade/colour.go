package grade

import (
	"fmt"
	"github.com/janearc/miami/internal/hex"
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
	r, g, b, _ := hex.Bytes(s)
	return srgb.RGB{R: float64(r) / 255, G: float64(g) / 255,
		B: float64(b) / 255}.Swatch()
}

// IsColour is whether a string is six hex digits, however they are written.
//
// For whoever reads a document, so that a colour nobody can draw is refused
// by name rather than drawn as something else. Swatch itself still answers,
// because a library call in the middle of a frame is no place to stop.
func IsColour(s string) bool { return hex.Is(s) }

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
	if len(stops)%2 != 0 {
		panic(fmt.Sprintf("grade.Ramp: %d things, which is a place "+
			"and a colour over and over and one left over; the "+
			"last is %v", len(stops), stops[len(stops)-1]))
	}
	var s []functions.Stop
	for i := 0; i+1 < len(stops); i += 2 {
		s = append(s, functions.Stop{At: at(stops[i]),
			Swatch: Swatch(colour(stops[i+1]))})
	}
	return functions.New(ok.Mix, s...)
}

// colour is a stop's colour, which is six hex digits and no hash.
//
// Guarded for the same reason the number is, and because leaving one of the
// two would be this change made in one field and not the one beside it:
// Ramp(0, 0xff2200, ...) panicked with "interface {} is int, not string",
// which is the message this set out to retire.
func colour(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	panic(fmt.Sprintf("grade.Ramp: a stop is a colour as six hex digits, "+
		"and %v is a %T", v, v))
}

// at is where a stop sits, from whatever number was written.
//
// An untyped nought or one in a call reaches here as an int, and asserting
// float64 on it panicked with "interface {} is int, not float64", which says
// nothing about the ramp it came from. Nought and one are the ends of every
// ramp, so they are the two most likely things anybody writes.
func at(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	}
	panic(fmt.Sprintf("grade.Ramp: a stop sits at a number, and %v is a "+
		"%T; the call is a place and a colour, over and over", v, v))
	// a ramp has no name to give, so these say the function and the
	// value: it is the call that is wrong, and the value is what finds it
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
