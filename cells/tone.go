package cells

import (
	"image/color"
	"math"
	"strings"
)

// Tone renders a picture with no colour at all, for documents that ship as
// plain text: shade characters carry fills and quadrant blocks carry edges,
// which is how nfo art held an image together in one colour. every glyph used
// here is in the block elements range, so it survives fonts that lack the
// sextants the colour render depends on.
//
// lit subcells are INK, and ink is drawn light: this assumes a dark
// background, as a terminal usually is. on a light page it reads inverted.

var shades = []struct {
	g rune
	v float64
}{{'░', 0.25}, {'▒', 0.50}, {'▓', 0.75}}

// lightness is CIE L*, 0 to 1: perceptually even steps, so the three shades
// land where the eye expects them rather than all bunching in the darks.
func lightness(c color.RGBA) float64 {
	lin := func(v uint8) float64 {
		s := float64(v) / 255
		if s <= 0.03928 {
			return s / 12.92
		}
		return math.Pow((s+0.055)/1.055, 2.4)
	}
	y := 0.2126*lin(c.R) + 0.7152*lin(c.G) + 0.0722*lin(c.B)
	if y > 0.008856 {
		return (116*math.Cbrt(y) - 16) / 100
	}
	return 9.033 * y
}

// Lightness turns a colour field into a tone field, for scenes with no tone
// palette of their own.
func Lightness(rgb [][]color.RGBA) [][]float64 {
	out := make([][]float64, len(rgb))
	for y, row := range rgb {
		out[y] = make([]float64, len(row))
		for x, c := range row {
			out[y][x] = lightness(c)
		}
	}
	return out
}

// shadeCost is added to every shade's error so a crisp quadrant wins a close
// call. without it the edges of letters picked up half-tone ghost shadows.
const shadeCost = 0.06

// Tone folds a 2 by subH field of lightness into one glyph per cell. with
// subH 3 each cell is fitted with an exact sextant mask, so a letter edge that
// is two lit subcells and four dark is drawn as exactly that; folding three
// subrows into two quadrant rows averaged those edges into grey shades, which
// is what ghosted every letter in the first plain text render. subH 2 is the
// quadrant fallback for fonts without sextants.
func Tone(field [][]float64, subH int) [][]rune {
	n := 2 * subH
	glyph := sextant
	if subH == 2 {
		glyph = quadrant
	}
	rows, cols := len(field)/subH, len(field[0])/2
	out := make([][]rune, rows)
	v := make([]float64, n)
	for r := 0; r < rows; r++ {
		line := make([]rune, cols)
		for c := 0; c < cols; c++ {
			for i := 0; i < n; i++ {
				v[i] = field[r*subH+i/2][c*2+i%2]
			}
			best, bestErr := ' ', math.Inf(1)
			for mask := 0; mask < 1<<n; mask++ {
				e := 0.0
				for i := 0; i < n; i++ {
					t := 0.0
					if mask&(1<<i) != 0 {
						t = 1
					}
					e += (v[i] - t) * (v[i] - t)
				}
				if e < bestErr {
					best, bestErr = glyph(mask), e
				}
			}
			for _, s := range shades {
				e := shadeCost * float64(n) / 4
				for i := 0; i < n; i++ {
					e += (v[i] - s.v) * (v[i] - s.v)
				}
				if e < bestErr {
					best, bestErr = s.g, e
				}
			}
			line[c] = best
		}
		out[r] = line
	}
	return out
}

// ToneText lays marks over a tone render and returns plain text.
func ToneText(t [][]rune, marks []Mark, w, h float64) string {
	rows, cols := len(t), len(t[0])
	for _, m := range marks {
		c := int(m.X / w * float64(cols))
		r := int(m.Y / h * float64(rows))
		if r >= 0 && c >= 0 && r < rows && c < cols {
			t[r][c] = m.Glyph
		}
	}
	var b strings.Builder
	for _, line := range t {
		b.WriteString(strings.TrimRight(string(line), " "))
		b.WriteByte('\n')
	}
	return b.String()
}
