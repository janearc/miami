package view

import (
	"math"
	"testing"

	sprite "github.com/janearc/libdaffy-sprite"
)

// A billboard drawn at any size has the same proportions as its stencil, in
// the only units that matter: columns and rows on the screen.
//
// A daffy pixel is one column by half a row, so a stencil of fw by fh pixels
// is fw columns by fh/2 rows. A sextant subcell is half a column by a third
// of a row, so a drawing w by h subcells is w/2 columns by h/3 rows. The two
// shapes have to match, or a bird's wings are the wrong length.
//
// They did not match. The width was worked out with a factor of two thirds
// where the same comment's reasoning gives four over three, so every
// billboard was drawn half as wide as its stencil.
func TestABillboardKeepsItsProportions(t *testing.T) {
	for _, size := range [][2]int{{12, 6}, {24, 11}, {7, 20}, {31, 9}} {
		art := make([]string, size[1])
		for i := range art {
			art[i] = string(make([]byte, size[0]))
		}
		f := sprite.Frame{Art: art}
		stencil := float64(size[0]) / (float64(size[1]) / 2)
		for _, rows := range []int{6, 30, 90} {
			w := wide(f, rows)
			drawn := (float64(w) / 2) / (float64(rows) / 3)
			if math.Abs(drawn-stencil) > 0.12*stencil {
				t.Errorf("%dx%d at %d rows: drawn %.2f wide "+
					"for every tall, stencil is %.2f",
					size[0], size[1], rows, drawn, stencil)
			}
		}
	}
}

// Billboard and Turning draw at that width, rather than working it out
// separately and drifting apart.
func TestBothSpritesUseTheSameWidth(t *testing.T) {
	art := []string{"WWWWWWWWWWWW", "WWWWWWWWWWWW", "WWWWWWWWWWWW",
		"WWWWWWWWWWWW", "WWWWWWWWWWWW", "WWWWWWWWWWWW"}
	f := sprite.Frame{Art: art}
	want := wide(f, 24)
	b := Billboard{Frames: []sprite.Frame{f}, Height: 1}
	if got := b.Render(Eye{Lens: 1}, 0, 24).Width(); got != want {
		t.Errorf("billboard drew %d wide, want %d", got, want)
	}
	s := Turning{Poses: [][]sprite.Frame{{f}}, Height: 1}
	if got := s.Render(Eye{Lens: 1}, 0, 24).Width(); got != want {
		t.Errorf("turning sprite drew %d wide, want %d", got, want)
	}
}
