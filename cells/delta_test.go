package cells

import (
	"image/color"
	"math/rand"
	"testing"
)

// The delta is what the loop sends sixty times a second and it had no test.
//
// That was found by an agent restructuring Delta for line width: it wanted to
// know its change was safe, found nothing to tell it, wrote a throwaway that
// proved it, and threw the proof away. The throwaway was the right instinct
// and the wrong artefact -- so here it is, kept.
//
// What a delta claims is exactly this: applying it to the previous frame gives
// the current one. Nothing else about the escape sequences matters.

// randomGrid is a frame of cells, some of them repeated so runs of unchanged
// cells occur -- a delta that is never asked to skip anything tests nothing.
func randomGrid(rng *rand.Rand, rows, cols int) [][]Cell {
	ink := []color.RGBA{
		{0, 0, 0, 255}, {255, 255, 255, 255}, {200, 30, 90, 255},
		{30, 200, 90, 255},
	}
	// a background may be transparent and a foreground may not: the glyph
	// has to be drawn in something. the wire says so too -- a background
	// goes out as the terminal's own default, and there is no such escape
	// for a foreground, so one written transparent comes back opaque
	// black. that is the contract, not a defect, and a test that asks for
	// more than it says was the first version of this one.
	grounds := append(append([]color.RGBA{}, ink...), Transparent)
	glyphs := []rune{' ', '▌', '▐', '█', 'a', '\U0001FB3B'}
	g := make([][]Cell, rows)
	for r := range g {
		g[r] = make([]Cell, cols)
		for c := range g[r] {
			g[r][c] = Cell{
				Glyph: glyphs[rng.Intn(len(glyphs))],
				Fg:    ink[rng.Intn(len(ink))],
				Bg:    grounds[rng.Intn(len(grounds))],
			}
		}
	}
	return g
}

// same says whether two grids hold the same cells.
func same(a, b [][]Cell) (int, int, bool) {
	if len(a) != len(b) {
		return -1, -1, false
	}
	for r := range a {
		if len(a[r]) != len(b[r]) {
			return r, -1, false
		}
		for c := range a[r] {
			if a[r][c] != b[r][c] {
				return r, c, false
			}
		}
	}
	return 0, 0, true
}

// TestDeltaAppliesToThePreviousFrame is the whole claim, two hundred times.
func TestDeltaAppliesToThePreviousFrame(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 200; i++ {
		rows, cols := 1+rng.Intn(6), 1+rng.Intn(10)
		prev := randomGrid(rng, rows, cols)
		cur := randomGrid(rng, rows, cols)

		// most frames change a little, not everything: copy some of
		// the old frame into the new one so there is something to skip
		for r := range cur {
			for c := range cur[r] {
				if rng.Intn(3) > 0 {
					cur[r][c] = prev[r][c]
				}
			}
		}

		got := randomGrid(rng, rows, cols)
		copyInto(got, prev)
		Apply(got, Delta(prev, cur))
		if r, c, ok := same(got, cur); !ok {
			t.Fatalf("round %d: differs at %d,%d", i, r, c)
		}
	}
}

// TestDeltaFromNothingIsTheWholeFrame checks the first frame of a loop, where
// there is no previous one to difference against.
func TestDeltaFromNothingIsTheWholeFrame(t *testing.T) {
	rng := rand.New(rand.NewSource(2))
	cur := randomGrid(rng, 4, 7)
	got := make([][]Cell, len(cur))
	for r := range got {
		got[r] = make([]Cell, len(cur[r]))
	}
	Apply(got, Delta(nil, cur))
	if r, c, ok := same(got, cur); !ok {
		t.Fatalf("differs at %d,%d", r, c)
	}
}

// TestDeltaAgainstAShorterPreviousFrame covers a terminal that grew: the old
// frame has fewer rows and shorter rows than the new one, and every cell the
// old frame does not cover has to be sent.
func TestDeltaAgainstAShorterPreviousFrame(t *testing.T) {
	rng := rand.New(rand.NewSource(3))
	prev := randomGrid(rng, 2, 3)
	cur := randomGrid(rng, 5, 9)
	got := make([][]Cell, len(cur))
	for r := range got {
		got[r] = make([]Cell, len(cur[r]))
		if r < len(prev) {
			copy(got[r], prev[r])
		}
	}
	Apply(got, Delta(prev, cur))
	if r, c, ok := same(got, cur); !ok {
		t.Fatalf("differs at %d,%d", r, c)
	}
}

// copyInto fills a grid with another's cells, as far as both reach.
func copyInto(dst, src [][]Cell) {
	for r := range dst {
		if r < len(src) {
			copy(dst[r], src[r])
		}
	}
}
