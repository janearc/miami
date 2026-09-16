package cells

import (
	"image/color"
	"testing"
)

// sample is a couple of rows with the awkward cases in them: a wide sextant, a
// quadrant, colours at both ends of the range, and two transparent backgrounds
// -- transparency being the thing the two writers disagree about.
func sample() [][]Cell {
	return [][]Cell{
		{
			{Glyph: 'a', Fg: color.RGBA{255, 0, 0, 255},
				Bg: color.RGBA{0, 0, 40, 255}},
			{Glyph: '▙', Fg: color.RGBA{1, 2, 3, 255},
				Bg: color.RGBA{250, 251, 252, 255}},
		},
		{
			{Glyph: ' ', Fg: color.RGBA{9, 9, 9, 255}, Bg: Transparent},
			{Glyph: '\U0001FB3B', Fg: color.RGBA{128, 64, 32, 255},
				Bg: Transparent},
		},
	}
}

// TestANSIAlphaRoundTrips writes cells out and reads them back, and asks for
// the same cells.
//
// The claim FromANSI makes is that it is the other half of the writers. This is
// the only thing that can hold it to that: either every glyph and both colours
// survive the trip, or one of the two is lying about what the escapes mean.
//
// ANSIAlpha is the writer used here because it is the one that states the whole
// cell, transparency included, and so is the one a round trip can be exact
// through.
func TestANSIAlphaRoundTrips(t *testing.T) {
	want := sample()
	got := FromANSI(ANSIAlpha(want))
	if len(got) != len(want) {
		t.Fatalf("rows: got %d want %d", len(got), len(want))
	}
	for y := range want {
		if len(got[y]) != len(want[y]) {
			t.Fatalf("row %d: got %d cells want %d",
				y, len(got[y]), len(want[y]))
		}
		for x := range want[y] {
			g, w := got[y][x], want[y][x]
			if g.Glyph != w.Glyph {
				t.Errorf("%d,%d glyph: got %q want %q",
					x, y, g.Glyph, w.Glyph)
			}
			// a transparent cell holding a space is written as a
			// plain reset, which loses a foreground nothing is
			// drawn in. that is not a colour anyone can see.
			if g.Fg != w.Fg && !(w.Bg == Transparent && w.Glyph == ' ') {
				t.Errorf("%d,%d fg: got %v want %v", x, y, g.Fg, w.Fg)
			}
			if g.Bg != w.Bg {
				t.Errorf("%d,%d bg: got %v want %v", x, y, g.Bg, w.Bg)
			}
		}
	}
}

// TestANSIFlattensTransparency pins the difference between the two writers.
//
// ANSI states both colours for every cell, so a transparent background comes
// back as black rather than as transparent. This is not a bug being tolerated
// -- it is what lets an ANSI file be cat'd anywhere and look the same -- but it
// is the trap that once stamped opaque boxes over a card, and a caller reading
// this test should see which writer to reach for.
func TestANSIFlattensTransparency(t *testing.T) {
	got := FromANSI(ANSI(sample()))
	if bg := got[1][0].Bg; bg != (color.RGBA{0, 0, 0, 255}) {
		t.Errorf("ANSI transparent bg: got %v, want opaque black", bg)
	}
	if bg := FromANSI(ANSIAlpha(sample()))[1][0].Bg; bg != Transparent {
		t.Errorf("ANSIAlpha transparent bg: got %v, want transparent", bg)
	}
}

// TestFromANSIPadsShortRows checks that a ragged file comes back square.
// Everything that draws cells indexes by column and would panic on a short row.
func TestFromANSIPadsShortRows(t *testing.T) {
	got := FromANSI("abc\nd\n")
	if len(got) != 2 {
		t.Fatalf("rows: got %d want 2", len(got))
	}
	if len(got[1]) != 3 {
		t.Fatalf("short row: got %d cells want 3", len(got[1]))
	}
	if got[1][2].Glyph != ' ' {
		t.Errorf("padding: got %q want a space", got[1][2].Glyph)
	}
}
