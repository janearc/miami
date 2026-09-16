package paint

import (
	"image/color"
	"math"
	"sort"

	"github.com/janearc/libtheme-css/spaces/ok"
	"github.com/janearc/libtheme-css/spaces/srgb"
)

// a picture with many colours in it does not survive being averaged per cell:
// the kingfisher came out as mud that way.
//
// what survives is a few colours, each with a range of tone inside it, and
// every source colour snapped to the nearest of them -- nearest measured where
// distance means something, which is oklab, not the three lamp levels.

// Entry is one colour of a palette: the name a scene uses, and the two ends of
// the range it shades across.
type Entry struct {
	Name        string
	Light, Deep color.RGBA
}

// Palette is a small set of colours a picture is told in.
type Palette struct {
	Entries []Entry
	lab     []ok.OKLab // the midpoint of each entry, for matching
}

// NewPalette prepares a palette for matching.
func NewPalette(entries ...Entry) *Palette {
	p := &Palette{Entries: entries}
	for _, e := range entries {
		mid := color.RGBA{
			uint8((int(e.Light.R) + int(e.Deep.R)) / 2),
			uint8((int(e.Light.G) + int(e.Deep.G)) / 2),
			uint8((int(e.Light.B) + int(e.Deep.B)) / 2), 255}
		p.lab = append(p.lab, Lab(mid))
	}
	return p
}

// Lab is a colour in oklab, through libtheme.
func Lab(c color.RGBA) ok.OKLab {
	return ok.FromSwatch(srgb.RGB{
		R: float64(c.R) / 255, G: float64(c.G) / 255, B: float64(c.B) / 255,
	}.Swatch())
}

// RGBA turns an oklab colour back into what a terminal is sent. out of gamut it
// is the nearest the lamps can do, which libtheme reports honestly.
func RGBA(c ok.OKLab) color.RGBA {
	rgb, _ := srgb.FromSwatch(c.Swatch())
	r, g, b := rgb.Bytes()
	return color.RGBA{r, g, b, 255}
}

// Distance is how different two colours look, 0.02 being the smallest
// difference most people can see.
func Distance(a, b color.RGBA) float64 { return ok.Distance(Lab(a), Lab(b)) }

// lightWeight is how much lightness counts when deciding which colour a sample
// belongs to. plain oklab distance is mostly lightness, so a bright teal
// matched cream; lightness is what the tone parameter carries, and what decides
// the family is hue and chroma.
const lightWeight = 0.3

// Nearest is the entry a colour belongs to, and how dark it sits within that
// entry, 0 at the light end and 1 at the deep end.
func (p *Palette) Nearest(c color.RGBA) (int, float64) {
	want := Lab(c)
	best, bestD := 0, math.Inf(1)
	for i, l := range p.lab {
		dl := (want.L - l.L) * lightWeight
		da, db := want.A-l.A, want.B-l.B
		if d := math.Sqrt(dl*dl + da*da + db*db); d < bestD {
			best, bestD = i, d
		}
	}
	e := p.Entries[best]
	lo, hi := Lab(e.Light).L, Lab(e.Deep).L
	t := 0.0
	if lo != hi {
		t = (want.L - lo) / (hi - lo)
	}
	return best, math.Max(0, math.Min(1, t))
}

// At is the colour of an entry at a tone.
func (p *Palette) At(i int, t float64) color.RGBA {
	e := p.Entries[i]
	a, b := Lab(e.Light), Lab(e.Deep)
	return RGBA(ok.OKLab{
		L: a.L + (b.L-a.L)*t,
		A: a.A + (b.A-a.A)*t,
		B: a.B + (b.B-a.B)*t,
	})
}

// ByLightness is a palette sorted dark to light.
//
// For colouring a wordmark set beside a picture. The palette is the point: a
// wordmark dropped next to a picture in colours somebody liked reads as two
// pictures, and one drawn from the subject's own colours reads as one. Nothing
// here is chosen by hand.
func ByLightness(pal []color.RGBA) []color.RGBA {
	out := append([]color.RGBA(nil), pal...)
	sort.Slice(out, func(i, j int) bool {
		return Lab(out[i]).L < Lab(out[j]).L
	})
	return out
}

// Hot is the colour t of the way down a wordmark, brightest at the top running
// into a hot mid at the bottom.
//
// It runs from the lightest colour to the middle one rather than to the
// darkest, because the darkest colours in a picture are its outlines and would
// put the word out of reach of a dark terminal.
func Hot(ramp []color.RGBA, t float64) color.RGBA {
	if len(ramp) < 2 {
		if len(ramp) == 1 {
			return ramp[0]
		}
		return color.RGBA{255, 255, 255, 255}
	}
	return Lerp(ramp[len(ramp)-1], ramp[len(ramp)/2], t)
}
