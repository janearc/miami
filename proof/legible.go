package proof

import (
	"fmt"
	"image/color"
	"io"
	"sort"
	"unicode"

	"github.com/janearc/miami/cells"
	"github.com/janearc/miami/paint"
	"github.com/janearc/miami/raster"
)

// the gate from the legibility standard, run on what was rendered: every text
// sample is measured against the non-text samples actually touching it, after
// gradients and glow, not against the colour a palette declared.

const (
	BandLow          = 10.0
	BandHigh         = 12.5
	MinHueSeparation = 40.0
	NeutralEnough    = 0.35
)

// Verdict is one text role's measurement.
type Verdict struct {
	Role  byte
	Pairs int
	// contrast ratios; P90 is the brightest-glare end
	Worst, Median, P90 float64
	HueFailures        int
	Pass               bool
}

// onGrid says whether a neighbour coordinate is on the grid at all.
func onGrid(g *raster.Grid, x, y int) bool {
	return x >= 0 && y >= 0 && x < g.W && y < g.H
}

// sameHue is a pairing with no colour edge: the background has enough
// saturation to carry a hue, and that hue sits too near the ink's.
func sameHue(ink, bg color.RGBA) bool {
	_, s, _ := paint.HSL(bg)
	return s >= NeutralEnough && paint.HueApart(ink, bg) < MinHueSeparation
}

// Legible measures each named text role against its touching background.
//
// a role passes when its median sits in the band, its worst pairing does not
// fall below the floor of the band by more than the noise of an antialiased
// edge (the 10th percentile is used for that, not the single worst sample), and
// no pairing lacks a colour edge.
func Legible(g *raster.Grid, rgb [][]color.RGBA, text []byte) []Verdict {
	isText := map[byte]bool{}
	for _, r := range text {
		isText[r] = true
	}
	var out []Verdict
	for _, role := range text {
		var ratios []float64
		hueBad := 0
		for y := 0; y < g.H; y++ {
			for x := 0; x < g.W; x++ {
				if g.Rows[y][x] != role {
					continue
				}
				ink := rgb[y][x]
				for _, d := range [][2]int{
					{1, 0}, {-1, 0}, {0, 1}, {0, -1},
				} {
					nx, ny := x+d[0], y+d[1]
					if !onGrid(g, nx, ny) {
						continue
					}
					if isText[g.Rows[ny][nx]] {
						continue
					}
					bg := rgb[ny][nx]
					ratios = append(ratios,
						paint.Contrast(ink, bg))
					if sameHue(ink, bg) {
						hueBad++
					}
				}
			}
		}
		v := Verdict{
			Role: role, Pairs: len(ratios), HueFailures: hueBad,
		}
		if len(ratios) > 0 {
			sort.Float64s(ratios)
			v.Worst = ratios[len(ratios)/10]
			v.Median = ratios[len(ratios)/2]
			v.P90 = ratios[len(ratios)*9/10]
			inBand := v.Median >= BandLow && v.Median <= BandHigh
			aboveFloor := v.Worst >= BandLow*0.8
			v.Pass = inBand && aboveFloor && hueBad == 0
		}
		out = append(out, v)
	}
	return out
}

// Report prints verdicts and returns how many failed.
func Report(w io.Writer, vs []Verdict) int {
	const format = "%s role %q  median %.2f:1  p10 %.2f  p90 %.2f  " +
		"pairs %d  no-colour-edge %d  (band %.1f-%.1f)\n"
	bad := 0
	for _, v := range vs {
		mark := "ok  "
		if !v.Pass {
			mark = "FAIL"
			bad++
		}
		fmt.Fprintf(w, format, mark, v.Role, v.Median, v.Worst, v.P90,
			v.Pairs, v.HueFailures, BandLow, BandHigh)
	}
	return bad
}

// LegibleMarks measures literal letters as the terminal will draw them: their
// own foreground on the background they were placed with.
func LegibleMarks(w io.Writer, cs [][]cells.Cell) int {
	var ratios []float64
	for _, row := range cs {
		for _, c := range row {
			g := c.Glyph
			if cells.IsBlock(g) || !unicode.IsLetter(g) {
				continue
			}
			ratios = append(ratios, paint.Contrast(c.Fg, c.Bg))
		}
	}
	if len(ratios) == 0 {
		return 0
	}
	sort.Float64s(ratios)
	lo, hi := ratios[0], ratios[len(ratios)-1]
	ok := lo >= BandLow && hi <= BandHigh
	mark := "ok  "
	if !ok {
		mark = "FAIL"
	}
	fmt.Fprintf(w,
		"%s literal letters  %d cells  %.2f:1 to %.2f:1  "+
			"(band %.1f-%.1f)\n",
		mark, len(ratios), lo, hi, BandLow, BandHigh)
	if ok {
		return 0
	}
	return 1
}
