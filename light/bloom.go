package light

import (
	"math"

	"github.com/janearc/miami/internal/split"
)

// Glow is light that spread: what a lens and an eye do to anything bright, and
// most of why a picture of a black hole looks like one.
//
// Only the light above a threshold spreads. A glow that took everything would
// be a haze over the whole frame; taking what is over the threshold is what
// keeps the dark dark and lets the disk bleed into it.
//
// Several glows at once is the point of taking a list. A real one has a tight
// core and a wide faint skirt, and no single blur is both.
type Glow struct {
	// Radius is how far it spreads, in terminal columns. A column rather
	// than a subpixel, because a subcell is not square and a glow measured
	// in them comes out oval.
	Radius float64
	// Strength is how much of the spread light is added back.
	Strength float64
}

// Bloom spreads the light above threshold, by each glow in turn, and adds it.
//
// The blur runs at a quarter of the resolution in each direction. A glow wide
// enough to matter is wide enough that nobody can tell, and a sixteenth of the
// subpixels is what lets a full-screen frame afford a wide one.
func (b *Buffer) Bloom(threshold float64, glows ...Glow) {
	if len(glows) == 0 {
		return
	}
	// a quarter pixel is two subpixels across, which is one column, and
	// four down, which is four thirds of a row. a row is two columns tall.
	qw, qh := (b.W+3)/4, (b.H+3)/4
	bright := make([]float32, qw*qh*3)
	for y := 0; y < b.H; y++ {
		for x := 0; x < b.W; x++ {
			c := b.At(x, y)
			l := c.Y()
			if l <= threshold {
				continue
			}
			k := float32((l - threshold) / l / 16)
			i := ((y/4)*qw + x/4) * 3
			bright[i] += float32(c.R) * k
			bright[i+1] += float32(c.G) * k
			bright[i+2] += float32(c.B) * k
		}
	}
	sum := make([]float32, len(bright))
	for _, g := range glows {
		rx := int(math.Round(g.Radius))
		ry := int(math.Round(g.Radius * 3 / 8))
		spread := boxes(bright, qw, qh, max(rx, 1), max(ry, 0))
		for i := range sum {
			sum[i] += spread[i] * float32(g.Strength)
		}
	}
	// back up to full size, bilinear, and added
	split.Rows(b.H, func(lo, hi int) {
		for y := lo; y < hi; y++ {
			fy := (float64(y)+0.5)/4 - 0.5
			y0 := clampInt(int(math.Floor(fy)), 0, qh-1)
			y1 := clampInt(y0+1, 0, qh-1)
			ty := float32(clamp01(fy - float64(y0)))
			for x := 0; x < b.W; x++ {
				fx := (float64(x)+0.5)/4 - 0.5
				x0 := clampInt(int(math.Floor(fx)), 0, qw-1)
				x1 := clampInt(x0+1, 0, qw-1)
				tx := float32(clamp01(fx - float64(x0)))
				o := (y*b.W + x) * 3
				for ch := 0; ch < 3; ch++ {
					a := sum[(y0*qw+x0)*3+ch]*(1-tx) +
						sum[(y0*qw+x1)*3+ch]*tx
					c := sum[(y1*qw+x0)*3+ch]*(1-tx) +
						sum[(y1*qw+x1)*3+ch]*tx
					b.pix[o+ch] += a*(1-ty) + c*ty
				}
			}
		}
	})
}

// boxes is three box blurs one after another, which is close enough to a
// gaussian that the difference is not a thing anyone sees, and each pass
// costs the same however wide it is.
func boxes(src []float32, w, h, rx, ry int) []float32 {
	a := append([]float32(nil), src...)
	tmp := make([]float32, len(src))
	for range 3 {
		blur(a, tmp, w, h, rx, false)
		blur(tmp, a, w, h, ry, true)
	}
	return a
}

// blur is one box pass across or down, from src into dst.
//
// A running sum, so a wide box costs what a narrow one does. Adding and taking
// away in float32 leaves crumbs of about 1e-11 where the sum should be nought,
// and a crumb below nought is light taken away, so nothing below nought is
// kept.
func blur(src, dst []float32, w, h, r int, down bool) {
	if r <= 0 {
		copy(dst, src)
		return
	}
	lines, length := h, w
	if down {
		lines, length = w, h
	}
	split.Rows(lines, func(lo, hi int) {
		for line := lo; line < hi; line++ {
			at := func(i int) int {
				i = clampInt(i, 0, length-1)
				if down {
					return (i*w + line) * 3
				}
				return (line*w + i) * 3
			}
			for ch := range 3 {
				blurLine(src, dst, length, r, ch, at)
			}
		}
	})
}

// blurLine is one channel of one line of a box pass. at is where the i-th
// sample along the line is kept, held to the ends past either edge.
func blurLine(src, dst []float32, length, r, ch int, at func(int) int) {
	norm := 1 / float32(2*r+1)
	var run float32
	for i := -r; i <= r; i++ {
		run += src[at(i)+ch]
	}
	for i := range length {
		dst[at(i)+ch] = max(0, run*norm)
		run += src[at(i+r+1)+ch] - src[at(i-r)+ch]
	}
}

// clampInt holds an int to a range.
func clampInt(v, lo, hi int) int { return max(lo, min(hi, v)) }

// clamp01 holds a number to nought and one.
func clamp01(v float64) float64 { return math.Max(0, math.Min(1, v)) }
