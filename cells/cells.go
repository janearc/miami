package cells

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"sort"
	"strings"

	"github.com/janearc/libtheme-css/spaces/ok"
	"github.com/janearc/libtheme-css/spaces/srgb"
)

// sextants: 2 wide by 3 tall subcells, bit 0 top-left through bit 5
// bottom-right. the set is complete -- every one of the 64 arrangements has a
// character -- so the shape is always exact and only colour is approximated.
func sextant(mask int) rune {
	switch {
	case mask == 0:
		return ' '
	case mask == 63:
		return '█'
	case mask == 21: // left half
		return '▌'
	case mask == 42: // right half
		return '▐'
	}
	idx := mask - 1
	if mask > 21 {
		idx--
	}
	if mask > 42 {
		idx--
	}
	return rune(0x1FB00 + idx)
}

// quadrants: 2 by 2, also complete, for terminals whose font lacks the legacy
// computing block.
func quadrant(mask int) rune {
	t := []rune{' ', '▘', '▝', '▀', '▖', '▌', '▞',
		'▛', '▗', '▚', '▐', '▜', '▄', '▙',
		'▟', '█'}
	return t[mask]
}

type Cell struct {
	Glyph  rune
	Fg, Bg color.RGBA
}

// dist2 is the squared distance between two colours in plain rgb. squared
// because only the ordering is wanted and a square root would not change it.
func dist2(a, b color.RGBA) int {
	dr := int(a.R) - int(b.R)
	dg := int(a.G) - int(b.G)
	db := int(a.B) - int(b.B)
	return dr*dr + dg*dg + db*db
}

// mean averages colours. Right for a photograph, where the group stands for a
// continuum; wrong for flat art, which is what fitFlat is for.
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

// fit picks the two colours and the subcell mask that minimise squared error
// over the group. exhaustive over all 2^n masks, which is 64 at worst.
func fit(sub []color.RGBA, glyph func(int) rune) Cell {
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

// ToCells folds the pixel field into terminal cells. sub is 2x3 or 2x2.
func ToCells(rgb [][]color.RGBA, subW, subH int) [][]Cell {
	glyph := sextant
	if subH == 2 {
		glyph = quadrant
	}
	rows := len(rgb) / subH
	cols := len(rgb[0]) / subW
	out := make([][]Cell, rows)
	for r := 0; r < rows; r++ {
		line := make([]Cell, cols)
		for c := 0; c < cols; c++ {
			sub := make([]color.RGBA, 0, subW*subH)
			for y := 0; y < subH; y++ {
				for x := 0; x < subW; x++ {
					sub = append(
						sub, rgb[r*subH+y][c*subW+x],
					)
				}
			}
			line[c] = fit(sub, glyph)
		}
		out[r] = line
	}
	return out
}

// ANSI is the cells as escape sequences: colour and glyphs, nothing that moves
// the cursor, so the output can be cat'd inside a readme.
//
// Every cell gets both colours stated, which means a transparent background is
// written as black rather than left to the terminal.
//
// That is the trap in this function and it has been fallen into: a caller that
// set a background transparent to mean "leave what is behind it" got opaque
// boxes stamped over a picture. If transparency is meant, use ANSIAlpha.
func ANSI(cells [][]Cell) string {
	var b strings.Builder
	for _, row := range cells {
		for _, c := range row {
			fmt.Fprintf(
				&b, "\x1b[38;2;%d;%d;%dm\x1b[48;2;%d;%d;%dm%c",
				c.Fg.R, c.Fg.G, c.Fg.B, c.Bg.R, c.Bg.G,
				c.Bg.B, c.Glyph)
		}
		b.WriteString("\x1b[0m\n")
	}
	return b.String()
}

// CellsToPixels expands cells back into a pixel field so the terminal render
// can be looked at as an image rather than asserted about.
func CellsToPixels(cells [][]Cell, subW, subH int) [][]color.RGBA {
	h := len(cells) * subH
	w := len(cells[0]) * subW
	out := make([][]color.RGBA, h)
	for i := range out {
		out[i] = make([]color.RGBA, w)
	}
	for r, row := range cells {
		for c, cell := range row {
			mask := maskOf(cell.Glyph, subW*subH)
			for i := 0; i < subW*subH; i++ {
				y := r*subH + i/subW
				x := c*subW + i%subW
				if mask&(1<<i) != 0 {
					out[y][x] = cell.Fg
				} else {
					out[y][x] = cell.Bg
				}
			}
		}
	}
	return out
}

// maskOf is the subcell mask a block glyph stands for, or -1 if it is not one.
func maskOf(g rune, n int) int {
	if n == 4 {
		for m := 0; m < 16; m++ {
			if quadrant(m) == g {
				return m
			}
		}
		return 0
	}
	if n == 6 {
		for m := 0; m < 64; m++ {
			if sextant(m) == g {
				return m
			}
		}
	}
	for m := 0; m < 16; m++ {
		if quadrant(m) == g {
			return m
		}
	}
	return 0
}

// Mark is a literal character placed in a cell by physical position. anything
// smaller than a cell -- a marquee bulb, a chaser light, fine print -- aliases
// into a different block glyph wherever it lands on the grid, so it is handed
// to the terminal's own font instead, which draws it the same way every time.
type Mark struct {
	X, Y  float64 // physical units
	Glyph rune
	Fg    color.RGBA
	Bg    *color.RGBA // nil keeps the colour that dominates the cell
}

// Place writes marks into a grid of cells covering a w by h physical page. two
// marks landing in one cell: the first wins.
func Place(cs [][]Cell, marks []Mark, w, h float64) {
	rows, cols := len(cs), len(cs[0])
	taken := map[[2]int]bool{}
	for _, m := range marks {
		c := int(m.X / w * float64(cols))
		r := int(m.Y / h * float64(rows))
		key := [2]int{r, c}
		if r < 0 || c < 0 || r >= rows || c >= cols || taken[key] {
			continue
		}
		taken[key] = true
		cell := cs[r][c]
		bg := cell.Bg
		if m.Bg != nil {
			bg = *m.Bg
		} else if n := ones(maskOf(cell.Glyph, 6)); n >= 3 {
			bg = cell.Fg
		}
		cs[r][c] = Cell{Glyph: m.Glyph, Fg: m.Fg, Bg: bg}
	}
}

// ones counts the set bits of a mask, which is how many subcells a glyph fills.
func ones(m int) int {
	n := 0
	for ; m > 0; m >>= 1 {
		n += m & 1
	}
	return n
}

// IsBlock reports whether a cell holds a block glyph this package drew, as
// opposed to a literal character.
func IsBlock(g rune) bool {
	return g == ' ' || g == 0x2588 || g == 0x258C || g == 0x2590 ||
		(g >= 0x1FB00 && g <= 0x1FB3B) || IsQuadrant(g)
}

// IsQuadrant reports a 2 by 2 block glyph, which previews must draw as such.
func IsQuadrant(g rune) bool {
	return (g >= 0x2596 && g <= 0x259F) || g == 0x2580 || g == 0x2584
}

// MaskOf is exported for previews: which subcells of a block glyph are lit.
func MaskOf(g rune, n int) int { return maskOf(g, n) }

// Sample reduces a picture to a grid of subpixels, ready for the cell fitter.
//
// Two details decide whether the edges look right. The average is of
// premultiplied colour: an emoji is mostly transparent, and averaging plain
// RGBA drags the zeroes behind those transparent pixels into every edge, which
// puts a dark fringe round the whole shape.
//
// And the result is composited onto the ground rather than left with an alpha,
// because a cell has a foreground and a background and nowhere to put a third
// thing.
func Sample(src image.Image, w, h int, ground color.RGBA) [][]color.RGBA {
	b := src.Bounds()
	out := make([][]color.RGBA, h)
	for y := 0; y < h; y++ {
		out[y] = make([]color.RGBA, w)
		y0 := b.Min.Y + b.Dy()*y/h
		y1 := b.Min.Y + b.Dy()*(y+1)/h
		if y1 == y0 {
			y1++
		}
		for x := 0; x < w; x++ {
			x0 := b.Min.X + b.Dx()*x/w
			x1 := b.Min.X + b.Dx()*(x+1)/w
			if x1 == x0 {
				x1++
			}
			var sr, sg, sb, sa, n float64
			for py := y0; py < y1; py++ {
				for px := x0; px < x1; px++ {
					r, g, bl, a := src.At(
						px, py,
					).RGBA() // already premultiplied
					sr += float64(r)
					sg += float64(g)
					sb += float64(bl)
					sa += float64(a)
					n++
				}
			}
			if n == 0 {
				out[y][x] = ground
				continue
			}
			a := sa / n / 65535
			mix := func(c float64, g uint8) uint8 {
				v := c/n/65535 + (1-a)*float64(g)/255
				if v > 1 {
					v = 1
				}
				return uint8(v*255 + 0.5)
			}
			out[y][x] = color.RGBA{
				mix(sr, ground.R), mix(sg, ground.G),
				mix(sb, ground.B), 255,
			}
		}
	}
	return out
}

// Transparent marks a cell's background as the terminal's own. A zero alpha is
// the signal rather than a new field: a cell with no background is exactly a
// cell whose background is not a colour.
var Transparent = color.RGBA{}

// SampleNearest reduces a picture by point-sampling, keeping alpha.
//
// Point-sampling and not averaging, on purpose. An emoji is flat colour with
// hard edges -- that is what it is drawn as -- and averaging is anti-aliasing:
//
// it turns every flat region into a slight gradient and every edge into a smear
// of intermediate colours, which the two-colour cell fitter then has to choose
// between. Taking the middle of each region keeps the flats flat and the edges
// hard, which is what the picture was drawn with in the first place.
func SampleNearest(src image.Image, w, h int) [][]color.RGBA {
	b := src.Bounds()
	out := make([][]color.RGBA, h)
	for y := 0; y < h; y++ {
		out[y] = make([]color.RGBA, w)
		sy := b.Min.Y + (2*y+1)*b.Dy()/(2*h)
		for x := 0; x < w; x++ {
			sx := b.Min.X + (2*x+1)*b.Dx()/(2*w)
			r, g, bl, a := src.At(sx, sy).RGBA()
			if a == 0 {
				out[y][x] = Transparent
				continue
			}
			// un-premultiply, or a half-covered edge pixel reads as
			// a dark version of its own colour
			un := func(c uint32) uint8 { return uint8(c * 255 / a) }
			out[y][x] = color.RGBA{
				un(r), un(g), un(bl), uint8(a >> 8),
			}
		}
	}
	return out
}

// ToCellsAlpha fits cells where some subpixels have no ink at all.
//
// Sextants are a complete set: all sixty-four arrangements of six subcells
// exist as characters.
//
// So a cell that is partly transparent does not need a compromise -- the glyph
// for its exact ink mask is always available, the background is left as the
// terminal's, and the silhouette comes out with the edge the artwork actually
// has.
//
// A cell that is wholly opaque goes through the ordinary two-colour fit, which
// is what keeps colour inside the shape.
//
// cut is the alpha at which a subpixel counts as ink. It is a hard threshold
// and meant to be: a soft one is the anti-aliasing this avoids.
func ToCellsAlpha(rgb [][]color.RGBA, subW, subH int, cut uint8) [][]Cell {
	glyph := sextant
	if subH == 2 {
		glyph = quadrant
	}
	rows, cols := len(rgb)/subH, len(rgb[0])/subW
	out := make([][]Cell, rows)
	for r := 0; r < rows; r++ {
		line := make([]Cell, cols)
		for c := 0; c < cols; c++ {
			sub := make([]color.RGBA, 0, subW*subH)
			mask, ink := 0, 0
			for y := 0; y < subH; y++ {
				for x := 0; x < subW; x++ {
					p := rgb[r*subH+y][c*subW+x]
					sub = append(sub, p)
					if p.A >= cut {
						mask |= 1 << uint(y*subW+x)
						ink++
					}
				}
			}
			switch {
			case ink == 0:
				line[c] = Cell{
					Glyph: ' ', Fg: Transparent,
					Bg: Transparent,
				}
			case ink == subW*subH:
				line[c] = fitFlat(sub, glyph)
			default:
				line[c] = Cell{
					Glyph: glyph(mask),
					Fg:    dominant(sub, cut),
					Bg:    Transparent,
				}
			}
		}
		out[r] = line
	}
	return out
}

// dominant is the commonest ink colour in a cell, which for flat artwork is the
// colour the cell actually is. A mean would invent one that is in none of its
// subpixels, and on a two-tone edge that invented colour is the fringe.
func dominant(sub []color.RGBA, cut uint8) color.RGBA {
	count := map[color.RGBA]int{}
	best, bestN := color.RGBA{}, 0
	for _, p := range sub {
		if p.A < cut {
			continue
		}
		p.A = 255
		count[p]++
		if count[p] > bestN {
			best, bestN = p, count[p]
		}
	}
	return best
}

// ANSIAlpha is ANSI with transparent cells left to the terminal's own
// background, which is what makes a header sit on the reader's colours instead
// of stamping a rectangle of ours over them.
func ANSIAlpha(cells [][]Cell) string {
	var b strings.Builder
	for _, row := range cells {
		for _, c := range row {
			if c.Bg == Transparent && c.Glyph == ' ' {
				b.WriteString("\x1b[0m ")
				continue
			}
			fmt.Fprintf(
				&b, "\x1b[38;2;%d;%d;%dm", c.Fg.R, c.Fg.G,
				c.Fg.B,
			)
			if c.Bg == Transparent {
				b.WriteString("\x1b[49m")
			} else {
				fmt.Fprintf(
					&b, "\x1b[48;2;%d;%d;%dm", c.Bg.R,
					c.Bg.G, c.Bg.B,
				)
			}
			b.WriteRune(c.Glyph)
		}
		b.WriteString("\x1b[0m\n")
	}
	return b.String()
}

// OnGround expands cells into pixels for previewing, painting transparent cells
// with the ground the reader's terminal would supply.
func OnGround(
	cells [][]Cell, subW, subH int, ground color.RGBA,
) [][]color.RGBA {
	out := CellsToPixels(cells, subW, subH)
	for r, row := range cells {
		for c, cell := range row {
			if cell.Bg != Transparent {
				continue
			}
			for y := 0; y < subH; y++ {
				dst := out[r*subH+y]
				for x := 0; x < subW; x++ {
					if dst[c*subW+x] == Transparent {
						dst[c*subW+x] = ground
					}
				}
			}
		}
	}
	return out
}

// Posterise reduces a picture to at most n colours, chosen by how much of the
// picture they cover and merged by eye rather than by arithmetic.
//
// This is the other half of not anti-aliasing.
//
// An emoji is drawn with a gradient across its face, and reproducing that
// gradient faithfully gives three hundred near-identical colours, a cell fitter
// thrashing between them, and a picture that reads as a photograph of an emoji
// rather than as the thing itself.
//
// Banding is the honest look at this size, and it makes the escape sequence a
// fraction of the length.
//
// The merge is in oklab, where a fixed distance means the same amount of
// visible difference everywhere, which is not true of rgb. Colours are taken in
// order of how many subpixels they cover, so the palette is the picture's own:
//
// the flats win and the edge colours, which are few and spread thin, collapse
// into whichever flat they are nearest.
func Posterise(px [][]color.RGBA, n int, tol float64) [][]color.RGBA {
	pal := PaletteOf(px, n, tol)
	if len(pal) == 0 {
		return px
	}
	labs := make([]ok.OKLab, len(pal))
	for i, c := range pal {
		labs[i] = labOf(c)
	}
	snap := map[color.RGBA]color.RGBA{}
	out := make([][]color.RGBA, len(px))
	for y, row := range px {
		out[y] = make([]color.RGBA, len(row))
		for x, p := range row {
			if p.A == 0 {
				out[y][x] = Transparent
				continue
			}
			if s, ok2 := snap[p]; ok2 {
				out[y][x] = s
				continue
			}
			l := labOf(p)
			best, bestD := pal[0], math.Inf(1)
			for i, q := range labs {
				if d := ok.Distance(l, q); d < bestD {
					best, bestD = pal[i], d
				}
			}
			best.A = p.A
			snap[p] = best
			out[y][x] = best
		}
	}
	return out
}

// PaletteOf is the picture's own palette: the colours that cover most of it,
// merged by eye. Exposed because a wordmark set beside a picture should be
// coloured out of that picture rather than out of somebody's taste.
func PaletteOf(px [][]color.RGBA, n int, tol float64) []color.RGBA {
	count := map[color.RGBA]int{}
	for _, row := range px {
		for _, p := range row {
			if p.A != 0 {
				count[p]++
			}
		}
	}
	type seen struct {
		c color.RGBA
		n int
	}
	all := make([]seen, 0, len(count))
	for c, k := range count {
		all = append(all, seen{c, k})
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].n != all[j].n {
			return all[i].n > all[j].n
		}
		return lessRGBA(
			all[i].c, all[j].c,
		) // ties broken by value, so it is repeatable
	})

	var pal []ok.OKLab
	var palRGBA []color.RGBA
	for _, s := range all {
		l := labOf(s.c)
		near := false
		for _, p := range pal {
			if ok.Distance(l, p) < tol {
				near = true
				break
			}
		}
		if !near {
			pal = append(pal, l)
			palRGBA = append(palRGBA, s.c)
			if len(pal) >= n {
				break
			}
		}
	}
	return palRGBA
}

// labOf is a terminal colour in oklab, where a fixed distance means the same
// amount of visible difference everywhere.
func labOf(c color.RGBA) ok.OKLab {
	rgb := srgb.RGB{
		R: float64(c.R) / 255,
		G: float64(c.G) / 255,
		B: float64(c.B) / 255,
	}
	return ok.FromSwatch(rgb.Swatch())
}

// lessRGBA orders colours by value, so a tie in a count breaks the same way
// every run and the output is repeatable.
func lessRGBA(a, b color.RGBA) bool {
	if a.R != b.R {
		return a.R < b.R
	}
	if a.G != b.G {
		return a.G < b.G
	}
	return a.B < b.B
}

// Letters stamps characters into cells as themselves.
//
// This is the other way to use a wordmark, and at a small size it is the only
// one that works. The block-glyph fitter approximates a shape with two colours;
// a letter is not a shape to be approximated, it is a letter, and at six
// subpixels of cap height an approximated N is an H.
//
// Here the character goes into the cell unchanged and only its colour is
// chosen, so the letterform is exactly what the font drew and the palette is
// still ours.
//
// The background is whatever the cell already had. Setting it transparent was
// wrong and it took a card over a bright gradient to show it: ANSI writes a
// zero background as a literal black, so every letter arrived in an opaque
// black box punched through the artwork.
//
// On a dark card that is invisible; on six colour ramps it reads as damage.
//
// Keeping the existing background means a word laid over a picture is type on
// the picture, and a word laid over nothing is still type on nothing, because
// nothing is what was already there.
func Letters(
	cs [][]Cell, art []string, col, row int, fg func(x, y int) color.RGBA,
) {
	for y, line := range art {
		for x, ch := range line {
			if ch == ' ' {
				continue
			}
			r, c := row+y, col+x
			if r < 0 || r >= len(cs) || c < 0 || c >= len(cs[r]) {
				continue
			}
			cs[r][c] = Cell{
				Glyph: ch, Fg: fg(x, y), Bg: cs[r][c].Bg,
			}
		}
	}
}

// Widen pads a subpixel field to the right with transparency, making room for
// something that is not painted with pixels.
func Widen(px [][]color.RGBA, w int) [][]color.RGBA {
	out := make([][]color.RGBA, len(px))
	for y, row := range px {
		out[y] = make([]color.RGBA, w)
		copy(out[y], row)
	}
	return out
}

// SampleFlat reduces a picture to flat colour on a subpixel grid.
//
// The order of operations is the whole of it, and the first version had it
// backwards. A colour emoji is not flat art: this one is a face in 1891 shades
// with a dark brow drawn over it, and two fifths of the brow's pixels are the
// anti-aliased ramp at its edge.
//
// Sampling first and reducing after means those ramp pixels are candidates for
// the palette, and the brow comes out as three browns with a fuzzy edge.
//
// So: choose the palette from the picture at full resolution, snap the whole
// picture to it, and only then reduce -- by majority vote, the same way a cell
// and a warped pixel are decided everywhere else here. A vote makes an edge
// pixel pick a side, which is what sharp means.
func SampleFlat(src image.Image, w, h, n int, tol float64) [][]color.RGBA {
	b := src.Bounds()
	pal := paletteOfImage(src, n, tol)
	if len(pal) == 0 {
		return SampleNearest(src, w, h)
	}
	labs := make([]ok.OKLab, len(pal))
	for i, c := range pal {
		labs[i] = labOf(c)
	}
	snap := map[color.RGBA]color.RGBA{}
	nearest := func(c color.RGBA) color.RGBA {
		if s, hit := snap[c]; hit {
			return s
		}
		l := labOf(c)
		best, bestD := pal[0], math.Inf(1)
		for i, q := range labs {
			if d := ok.Distance(l, q); d < bestD {
				best, bestD = pal[i], d
			}
		}
		snap[c] = best
		return best
	}

	out := make([][]color.RGBA, h)
	for y := 0; y < h; y++ {
		out[y] = make([]color.RGBA, w)
		y0, y1 := b.Min.Y+b.Dy()*y/h, b.Min.Y+b.Dy()*(y+1)/h
		if y1 == y0 {
			y1++
		}
		for x := 0; x < w; x++ {
			x0, x1 := b.Min.X+b.Dx()*x/w, b.Min.X+b.Dx()*(x+1)/w
			if x1 == x0 {
				x1++
			}
			votes := map[color.RGBA]int{}
			opaque, total := 0, 0
			best, bestN := Transparent, 0
			for py := y0; py < y1; py++ {
				for px := x0; px < x1; px++ {
					r, g, bl, a := src.At(px, py).RGBA()
					total++
					if a < 0x8000 {
						continue
					}
					opaque++
					un := func(c uint32) uint8 {
						return uint8(c * 255 / a)
					}
					c := nearest(color.RGBA{un(r), un(g),
						un(bl), 255})
					votes[c]++
					if votes[c] > bestN {
						best, bestN = c, votes[c]
					}
				}
			}
			// the silhouette votes too: a subpixel is ink when most
			// of what it covers is ink, so the outline is as hard
			// as the colours inside it
			if opaque*2 <= total {
				out[y][x] = Transparent
			} else {
				out[y][x] = best
			}
		}
	}
	return out
}

// paletteOfImage picks a palette from a picture by clustering first and ranking
// after.
//
// Ranking individual colours by how often they occur is what broke this. A face
// in nineteen hundred shades of orange has hundreds of colours occurring a
// hundred times each; a brow drawn in one brown occurs as five hundred
// near-identical browns of fifty each.
//
// By raw count every orange outranks every brown and the palette has no dark in
// it at all -- although the brow is twelve hundred pixels and matters more to
// the picture than any single shade of cheek.
//
// Clustering at the tolerance first, then ranking clusters by their total
// coverage, puts the brow where it belongs: near the top.
func paletteOfImage(src image.Image, n int, tol float64) []color.RGBA {
	b := src.Bounds()
	count := map[color.RGBA]int{}
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, a := src.At(x, y).RGBA()
			if a < 0x8000 {
				continue
			}
			un := func(c uint32) uint8 { return uint8(c * 255 / a) }
			count[color.RGBA{un(r), un(g), un(bl), 255}]++
		}
	}
	type seen struct {
		c color.RGBA
		n int
	}
	all := make([]seen, 0, len(count))
	for c, k := range count {
		all = append(all, seen{c, k})
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].n != all[j].n {
			return all[i].n > all[j].n
		}
		return lessRGBA(all[i].c, all[j].c)
	})

	type cluster struct {
		lab   ok.OKLab
		rep   color.RGBA
		cover int
	}
	var cs []cluster
	for _, s := range all {
		l := labOf(s.c)
		joined := false
		for i := range cs {
			if ok.Distance(l, cs[i].lab) < tol {
				cs[i].cover += s.n
				joined = true
				break
			}
		}
		if !joined {
			cs = append(cs, cluster{l, s.c, s.n})
		}
	}
	sort.SliceStable(
		cs, func(i, j int) bool { return cs[i].cover > cs[j].cover },
	)
	if len(cs) > n {
		cs = cs[:n]
	}
	out := make([]color.RGBA, len(cs))
	for i, c := range cs {
		out[i] = c.rep
	}
	return out
}

// fitFlat picks two colours for a cell from the colours actually in it.
//
// The difference from fit is that it never averages. Averaging is right for a
// photograph, where the two groups stand for a continuum and their means are
// the best summary of it. It is wrong for flat art:
//
// a cell holding three palette colours gets a mean of two of them, that mean is
// in none of its subpixels and in no other cell, and along an edge those
// invented in-between colours are exactly the fuzz that a flat picture should
// not have.
//
// Exhaustive over pairs of the distinct colours present, which is at most
// fifteen for a sextant, so it is exact rather than approximate.
func fitFlat(sub []color.RGBA, glyph func(int) rune) Cell {
	var distinct []color.RGBA
	for _, c := range sub {
		seen := false
		for _, d := range distinct {
			if d == c {
				seen = true
				break
			}
		}
		if !seen {
			distinct = append(distinct, c)
		}
	}
	if len(distinct) == 1 {
		// a solid cell: everything on, so the glyph is the full block
		return Cell{
			Glyph: glyph((1 << len(sub)) - 1), Fg: distinct[0],
			Bg: distinct[0],
		}
	}
	bestErr, best := -1, Cell{Glyph: ' '}
	for i := 0; i < len(distinct); i++ {
		for j := i + 1; j < len(distinct); j++ {
			fg, bg := distinct[i], distinct[j]
			mask, e := 0, 0
			for k, c := range sub {
				df, db := dist2(c, fg), dist2(c, bg)
				if df <= db {
					mask |= 1 << uint(k)
					e += df
				} else {
					e += db
				}
			}
			if bestErr < 0 || e < bestErr {
				bestErr, best = e, Cell{
					Glyph: glyph(mask), Fg: fg, Bg: bg,
				}
			}
		}
	}
	return best
}

// Delta writes only the cells that changed since the previous frame.
//
// A full repaint of a hundred by twenty-eight cells is about a hundred
// kilobytes of escape sequences, and at fifteen frames a second that is thirty
// megabytes for twenty seconds of sky. Most of those cells did not change: the
// ground does not move, and the bird is a tenth of the frame.
//
// Sending the difference is what makes this a stream rather than a flipbook.
//
// The run of unchanged cells is what costs: jumping the cursor is nine bytes or
// so, and re-sending a cell is about twenty, so a gap of one is not worth
// jumping over. Runs shorter than that are re-sent rather than skipped.
func Delta(prev, cur [][]Cell) string {
	var b strings.Builder
	var lastFg, lastBg color.RGBA
	var haveFg, haveBg bool
	for r, row := range cur {
		col := -1
		var old []Cell
		if prev != nil && r < len(prev) {
			old = prev[r]
		}
		for c, cell := range row {
			if c < len(old) && old[c] == cell {
				continue
			}
			if col != c {
				// a jump resets nothing, but the terminal's
				// idea of the current colour is only safe while
				// we are writing forwards
				fmt.Fprintf(&b, "\x1b[%d;%dH", r+1, c+1)
				haveFg, haveBg = false, false
			}
			if !haveFg || cell.Fg != lastFg {
				fmt.Fprintf(
					&b, "\x1b[38;2;%d;%d;%dm", cell.Fg.R,
					cell.Fg.G, cell.Fg.B,
				)
				lastFg, haveFg = cell.Fg, true
			}
			if !haveBg || cell.Bg != lastBg {
				if cell.Bg == Transparent {
					b.WriteString("\x1b[49m")
				} else {
					fmt.Fprintf(
						&b, "\x1b[48;2;%d;%d;%dm",
						cell.Bg.R, cell.Bg.G, cell.Bg.B,
					)
				}
				lastBg, haveBg = cell.Bg, true
			}
			b.WriteRune(cell.Glyph)
			col = c + 1
		}
	}
	return b.String()
}

// Apply replays a delta onto a grid, which is how the encoder is checked: a
// stream that cannot be replayed into the frames it was made from is not a
// stream, it is a hope.
func Apply(grid [][]Cell, delta string) {
	r, c := 0, 0
	fg, bg := color.RGBA{}, Transparent
	for i := 0; i < len(delta); {
		if delta[i] != 0x1b {
			ch, sz := decodeRune(delta[i:])
			if r < len(grid) && c < len(grid[r]) {
				grid[r][c] = Cell{Glyph: ch, Fg: fg, Bg: bg}
			}
			c++
			i += sz
			continue
		}
		j := i + 2
		for j < len(delta) && paramByte(delta[j]) {
			j++
		}
		args, verb := delta[i+2:j], delta[j]
		i = j + 1
		switch verb {
		case 'H':
			var rr, cc int
			fmt.Sscanf(args, "%d;%d", &rr, &cc)
			r, c = rr-1, cc-1
		case 'm':
			var a, b2, x, y, z int
			if n, _ := fmt.Sscanf(
				args, "%d;%d;%d;%d;%d", &a, &b2, &x, &y, &z,
			); n == 5 && b2 == 2 {
				if a == 38 {
					fg = color.RGBA{
						uint8(x), uint8(y), uint8(z),
						255,
					}
				} else {
					bg = color.RGBA{
						uint8(x), uint8(y), uint8(z),
						255,
					}
				}
			} else if args == "49" {
				bg = Transparent
			}
		}
	}
}

// paramByte reports whether b belongs to the numeric argument of a control
// sequence rather than the letter that ends it.
func paramByte(b byte) bool { return b == ';' || (b >= '0' && b <= '9') }

// decodeRune is the first rune of a string, and how many bytes it took.
func decodeRune(s string) (rune, int) {
	for i, r := range s {
		_ = i
		return r, len(string(r))
	}
	return ' ', 1
}

// ShiftLeft moves the picture left and fills behind it with a colour, so a
// wordmark can be given room the artwork was using.
func ShiftLeft(cs [][]Cell, n int, fill color.RGBA) {
	if n <= 0 {
		return
	}
	for r := range cs {
		w := len(cs[r])
		for c := 0; c < w; c++ {
			if c+n < w {
				cs[r][c] = cs[r][c+n]
			} else {
				cs[r][c] = Cell{Glyph: ' ', Fg: fill, Bg: fill}
			}
		}
	}
}
