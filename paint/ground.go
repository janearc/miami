package paint

import (
	"image/color"
	"math"

	"github.com/janearc/libtheme-css/spaces/ok"
)

// the ground is the terminal's own background, and it decides what a picture
// can say. black plumage on a dark terminal is not dark, it is absent; the
// same colour on a light one is the strongest thing on the page. so a palette
// is adapted to the ground rather than written for one and hoped over.

// Ground is the colour a picture is drawn on.
type Ground struct{ C color.RGBA }

// Dark reports whether the ground is the darker half of the range.
func (g Ground) Dark() bool { return Lab(g.C).L < 0.5 }

// MinSeparation is how far in oklab lightness a colour must sit from the
// ground to be seen at all. 0.18 was measured the hard way: a bill at 0.06
// vanished into a dark terminal.
const MinSeparation = 0.18

// Adapt pushes every entry away from the ground until it separates, keeping
// hue and chroma, so a palette written for one terminal reads on the other.
func (g Ground) Adapt(p *Palette) *Palette {
	gl := Lab(g.C).L
	out := make([]Entry, len(p.Entries))
	for i, e := range p.Entries {
		out[i] = Entry{e.Name, g.push(e.Light, gl), g.push(e.Deep, gl)}
	}
	return NewPalette(out...)
}

func (g Ground) push(c color.RGBA, gl float64) color.RGBA {
	lab := Lab(c)
	if math.Abs(lab.L-gl) >= MinSeparation {
		return c
	}
	l := gl + MinSeparation
	if !g.Dark() {
		l = gl - MinSeparation
	}
	polar := lab.Polar()
	polar.L = math.Max(0, math.Min(1, l))
	return RGBA(polar.Rect())
}

// Ink is the colour text should take on this ground: the far end of the
// range, so a one-colour render knows which way round it is.
func (g Ground) Ink() color.RGBA {
	if g.Dark() {
		return RGBA(ok.OKLab{L: 0.92})
	}
	return RGBA(ok.OKLab{L: 0.18})
}
