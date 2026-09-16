package proof

import (
	"image"
	"image/color"
	"sync"

	"miami/cells"
	"miami/geom"
	"miami/typeset"
)

// CellImage draws cells the way a terminal would: block glyphs as exact
// subcells, and literal characters with a real monospace font. without the
// second half, a preview shows every marquee bulb as an empty cell and the
// judgement made from it is wrong.
type CellImage struct {
	Face *typeset.Face
	// Ground stands in for a cell whose background is the terminal's own. A
	// preview has to paint something there, and painting transparent black
	// is how a transparent header comes out looking like a black rectangle.
	Ground color.RGBA
	cap    float64 // cap height at which one advance is exactly one cell wide; glyphs are drawn at it
	W, H   int     // pixels per cell
	glyphs map[rune]geom.Path
}

// NewCellImage sets up a drawer at a cell size, working out the cap height at
// which one glyph advance is exactly one cell wide.
func NewCellImage(face *typeset.Face, w, h int) *CellImage {
	_, adv, err := face.Glyph('M', geom.Vec{}, 1)
	if err != nil || adv == 0 {
		adv = 0.6
	}
	return &CellImage{Face: face, cap: float64(w) / adv, W: w, H: h,
		Ground: color.RGBA{0, 0, 0, 255}, glyphs: map[rune]geom.Path{}}
}

// NewCellImageOn is a cell image drawn over a given terminal background.
func NewCellImageOn(face *typeset.Face, w, h int, ground color.RGBA) *CellImage {
	ci := NewCellImage(face, w, h)
	ci.Ground = ground
	return ci
}

// Mono is Menlo, the face every preview of a terminal render is drawn with.
func Mono() *typeset.Face {
	monoOnce.Do(func() {
		monoFace, monoErr = typeset.LoadFace("/System/Library/Fonts/Menlo.ttc", "Menlo Regular")
	})
	Must(monoErr)
	return monoFace
}

var (
	monoOnce sync.Once
	monoFace *typeset.Face
	monoErr  error
)

// Draw renders cells as an image: block glyphs as their masks, shades as the
// flat tones ghostty actually draws, and everything else as real type.
func (ci *CellImage) Draw(cs [][]cells.Cell) *image.RGBA {
	rows, cols := len(cs), len(cs[0])
	img := image.NewRGBA(image.Rect(0, 0, cols*ci.W, rows*ci.H))
	for r, row := range cs {
		for c, cell := range row {
			ox, oy := c*ci.W, r*ci.H
			if cell.Bg.A == 0 {
				cell.Bg = ci.Ground
			}
			if cell.Fg.A == 0 {
				cell.Fg = ci.Ground
			}
			if cells.IsBlock(cell.Glyph) {
				// quadrants are 2 by 2, everything else here is
				// 2 by 3
				n, sh := 6, 3
				if cells.IsQuadrant(cell.Glyph) {
					n, sh = 4, 2
				}
				mask := cells.MaskOf(cell.Glyph, n)
				for y := 0; y < ci.H; y++ {
					sy := y * sh / ci.H
					for x := 0; x < ci.W; x++ {
						sx := x * 2 / ci.W
						col := cell.Bg
						if mask&(1<<(sy*2+sx)) != 0 {
							col = cell.Fg
						}
						img.SetRGBA(ox+x, oy+y, col)
					}
				}
				continue
			}
			// ghostty draws the shade glyphs as flat tones, not
			// dither patterns: seen in real captures on 2026-09-14.
			// a preview that dithered them would hide the two-tone
			// breakup those flat greys cause.
			if t, ok := flatShade[cell.Glyph]; ok {
				col := mix(cell.Bg, cell.Fg, t)
				for y := 0; y < ci.H; y++ {
					for x := 0; x < ci.W; x++ {
						img.SetRGBA(ox+x, oy+y, col)
					}
				}
				continue
			}
			p, ok := ci.glyphs[cell.Glyph]
			if !ok {
				// baseline a little below three quarters of the
				// cell, as monospace terminal fonts sit
				p, _, _ = ci.Face.Glyph(cell.Glyph, geom.Vec{X: 0, Y: float64(ci.H) * 0.77}, ci.cap)
				ci.glyphs[cell.Glyph] = p
			}
			for y := 0; y < ci.H; y++ {
				for x := 0; x < ci.W; x++ {
					// 2x2 supersample for a fair edge
					n := 0
					for _, d := range [4][2]float64{{.25, .25}, {.75, .25}, {.25, .75}, {.75, .75}} {
						if p.In(geom.Vec{X: float64(x) + d[0], Y: float64(y) + d[1]}) {
							n++
						}
					}
					img.SetRGBA(ox+x, oy+y, mix(cell.Bg, cell.Fg, float64(n)/4))
				}
			}
		}
	}
	return img
}

var flatShade = map[rune]float64{'░': 0.25, '▒': 0.5, '▓': 0.75}

// mix is two colours blended, for a glyph's edge against its background.
func mix(a, b color.RGBA, t float64) color.RGBA {
	f := func(x, y uint8) uint8 { return uint8(float64(x) + (float64(y)-float64(x))*t) }
	return color.RGBA{f(a.R, b.R), f(a.G, b.G), f(a.B, b.B), 255}
}
