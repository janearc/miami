package main

import (
	"image"
	"image/color"
	"math"

	"github.com/janearc/miami/light"
	"testing"
)

// The icon is square whatever shape the sheet was, the shadow is a dark disc
// near the middle of it, and the lit ground is far brighter than the shadow.
//
// The claims are about contrast rather than absolute brightness, because
// brightness depends on the size: a sky of stars too small to resolve is
// brighter per pixel the fewer pixels there are, which is true of the real sky
// and of this one. Two hundred and fifty-six is small enough to be quick and
// large enough to read.
func TestTheIconIsASquareWithAHoleInIt(t *testing.T) {
	const size = 256
	for _, day := range []bool{false, true} {
		img := square(iconLight(size, day), size)
		if b := img.Bounds(); b.Dx() != size || b.Dy() != size {
			t.Fatalf("day %v: %dx%d, want square", day, b.Dx(), b.Dy())
		}
		shadow, mean := math.Inf(1), 0.0
		middle := float64(size) / 2
		for y := range size {
			for x := range size {
				mean += grey(img.RGBAAt(x, y)) / float64(size*size)
				away := math.Hypot(float64(x)-middle,
					float64(y)-middle)
				if away < float64(size)/8 {
					shadow = math.Min(shadow, patch(img, x, y))
				}
			}
		}
		corner := patch(img, 6, 6)
		if shadow > mean/2 {
			t.Errorf("day %v: the darkest place near the middle is "+
				"%.3f against a picture averaging %.3f; the "+
				"shadow should be much darker", day, shadow, mean)
		}
		if day && corner < 0.4 {
			t.Errorf("the lit icon's ground is %.2f, too dark", corner)
		}
		if !day && corner > 0.4 {
			t.Errorf("the dark icon's ground is %.2f, too light", corner)
		}
		if day && shadow > corner/2 {
			t.Errorf("the shadow (%.3f) does not stand out of the lit "+
				"ground (%.3f)", shadow, corner)
		}
	}
}

// patch is how bright a few pixels about a point are, so one stray spark
// cannot answer for the shadow.
func patch(img *image.RGBA, x, y int) float64 {
	sum := 0.0
	for dy := -2; dy <= 2; dy++ {
		for dx := -2; dx <= 2; dx++ {
			sum += grey(img.RGBAAt(x+dx, y+dy))
		}
	}
	return sum / 25
}

// grey is how bright a colour is, nought to one.
func grey(c color.RGBA) float64 {
	return (0.2126*float64(c.R) + 0.7152*float64(c.G) +
		0.0722*float64(c.B)) / 255
}

// Stretching the sheet's rows into the square mixes them rather than
// repeating them: a ramp comes out as a ramp, not as stairs.
//
// A sheet row covers four thirds of what a column does, so three rows become
// four and the odd ones out are between two of the originals. Taking the
// nearer row instead would give a square with only as many distinct values as
// the sheet had rows.
func TestTheSquareMixesRowsRatherThanRepeatingThem(t *testing.T) {
	const size = 64
	b := light.New(size, size*3/4)
	for y := range b.H {
		v := float64(y) / float64(b.H-1)
		for x := range b.W {
			b.Set(x, y, light.RGB{R: v, G: v, B: v})
		}
	}
	img := square(b, size)
	seen := map[uint8]bool{}
	last := -1.0
	for y := range size {
		c := img.RGBAAt(size/2, y)
		seen[c.G] = true
		if g := grey(c); g < last-1e-9 {
			t.Fatalf("row %d went down the ramp: %.3f after %.3f",
				y, g, last)
		}
		last = grey(c)
	}
	if len(seen) <= b.H {
		t.Fatalf("%d distinct values from %d rows: the rows were "+
			"repeated rather than mixed", len(seen), b.H)
	}
}
