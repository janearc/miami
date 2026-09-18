package cells

import (
	"image/color"
	"math/rand"
	"testing"
)

// mean averages colours, as fit first did it.
func mean(cs []color.RGBA) color.RGBA {
	if len(cs) == 0 {
		return color.RGBA{0, 0, 0, 255}
	}
	var r, g, b int
	for _, c := range cs {
		r += int(c.R)
		g += int(c.G)
		b += int(c.B)
	}
	n := len(cs)
	return color.RGBA{uint8(r / n), uint8(g / n), uint8(b / n), 255}
}

// fitByGroups is fit as it was first written: build the two groups, take
// their means, sum the error. Kept here as the reference the faster one must
// agree with, cell for cell.
func fitByGroups(sub []color.RGBA, glyph func(int) rune) Cell {
	n := len(sub)
	bestErr := -1
	best := Cell{Glyph: ' '}
	for mask := 0; mask < (1 << n); mask++ {
		var on, off []color.RGBA
		for i := 0; i < n; i++ {
			if mask&(1<<i) != 0 {
				on = append(on, sub[i])
			} else {
				off = append(off, sub[i])
			}
		}
		fg, bg := mean(on), mean(off)
		e := 0
		for i := 0; i < n; i++ {
			if mask&(1<<i) != 0 {
				e += dist2(sub[i], fg)
			} else {
				e += dist2(sub[i], bg)
			}
		}
		if bestErr < 0 || e < bestErr {
			bestErr = e
			best = Cell{Glyph: glyph(mask), Fg: fg, Bg: bg}
		}
	}
	return best
}

// The fitter that does not allocate picks the same glyph and the same two
// colours as the one that did, on groups built to tie as well as random ones.
//
// Ties are the case that matters. Random colours almost never tie, and a
// mean rounded a different way only shows up where two masks score the same.
func TestFitAgreesWithTheReference(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	few := []color.RGBA{{0, 0, 0, 255}, {255, 255, 255, 255},
		{17, 200, 3, 255}, {18, 201, 3, 255}}
	for i := 0; i < 200000; i++ {
		n := 6
		glyph := sextant
		if i%4 == 0 {
			n, glyph = 4, quadrant
		}
		sub := make([]color.RGBA, n)
		for j := range sub {
			if i%2 == 0 {
				sub[j] = few[rng.Intn(len(few))]
				continue
			}
			sub[j] = color.RGBA{uint8(rng.Intn(256)),
				uint8(rng.Intn(256)), uint8(rng.Intn(256)), 255}
		}
		got, want := fit(sub, glyph), fitByGroups(sub, glyph)
		if got != want {
			t.Fatalf("group %d %v: got %+v, want %+v",
				i, sub, got, want)
		}
	}
}

// ToCells on every core gives the cells one core gives.
func TestToCellsIsTheSameInParallel(t *testing.T) {
	rng := rand.New(rand.NewSource(11))
	const cols, rows = 97, 41
	px := make([][]color.RGBA, rows*3)
	for y := range px {
		px[y] = make([]color.RGBA, cols*2)
		for x := range px[y] {
			px[y][x] = color.RGBA{uint8(rng.Intn(256)),
				uint8(rng.Intn(256)), uint8(rng.Intn(256)), 255}
		}
	}
	got := ToCells(px, 2, 3)
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			sub := []color.RGBA{}
			for y := 0; y < 3; y++ {
				for x := 0; x < 2; x++ {
					sub = append(sub, px[r*3+y][c*2+x])
				}
			}
			if want := fitByGroups(sub, sextant); got[r][c] != want {
				t.Fatalf("cell %d,%d: got %+v, want %+v",
					r, c, got[r][c], want)
			}
		}
	}
}
