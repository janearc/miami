package proof

import (
	"fmt"
	"image/color"
	"os"
	"strings"

	"miami/canvas"
	"miami/cells"
	"miami/raster"
	"miami/typeset"
)

// Page renders a tall scene at a locked width and writes what is needed to
// actually look at it: the whole page small, the page in readable windows, and
// the same scene at half and quarter width to see whether the masses hold.
func Page(sc *raster.Scene, cols int, name string, text []byte, marks []cells.Mark) ([][]cells.Cell, *raster.Grid, [][]color.RGBA) {
	monoFace, err := typeset.LoadFace("/System/Library/Fonts/Menlo.ttc", "Menlo Regular")
	Must(err)
	mono := monoFace
	rows := raster.TermRows(sc, cols)
	gw, gh := cols*2, rows*3
	g := raster.Rasterize(sc, gw, gh, 4)
	rgb := raster.Resolve(g, sc, 0, 0, gw, gh)
	cs := cells.ToCells(rgb, 2, 3)
	cells.Place(cs, marks, sc.W, sc.H)

	Must(os.WriteFile("out/"+name+".ansi", []byte(cells.ANSI(cs)), 0o644))
	Must(os.WriteFile("out/"+name+"-roles.txt", []byte(raster.RoleText(g)), 0o644))
	// a cell is 1:2; every preview keeps that, and draws literal glyphs
	// with a real monospace face rather than leaving them blank
	Must(SaveImage("out/"+name+"-whole.png", NewCellImage(mono, 6, 12).Draw(cs)))
	const win = 40
	big := NewCellImage(mono, 12, 24)
	for i, top := 0, 0; top < rows; i, top = i+1, top+win-4 {
		bot := top + win
		if bot > rows {
			bot = rows
		}
		Must(SaveImage(fmt.Sprintf("out/%s-w%d.png", name, i), big.Draw(cs[top:bot])))
		if bot == rows {
			break
		}
	}
	for _, div := range []int{2, 4} {
		c := cols / div
		r := raster.TermRows(sc, c)
		gg := raster.Rasterize(sc, c*2, r*3, 4)
		// no marks here: they are one cell each at any width, so at a
		// quarter the border bulbs swelled into blobs and the fine
		// print overwrote itself. this render judges mass, and marks
		// are not mass.
		cc := cells.ToCells(raster.Resolve(gg, sc, 0, 0, c*2, r*3), 2, 3)
		Must(SaveImage(fmt.Sprintf("out/%s-div%d.png", name, div), NewCellImage(mono, 6*div/2, 12*div/2).Draw(cc)))
	}
	fmt.Fprintf(os.Stderr, "%s: %d x %d cells, %d windows\n", name, cols, rows, (rows+win-5)/(win-4))
	// the plain text render for documents, and what it looks like in a dark
	// terminal with no colour at all
	field, own := raster.ToneField(g, sc)
	if !own {
		field = cells.Lightness(rgb)
	}
	tone := cells.Tone(field, 3)
	txt := cells.ToneText(tone, marks, sc.W, sc.H)
	Must(os.WriteFile("out/"+name+".txt", []byte(txt), 0o644))
	// the quadrant fallback, from its own 2 by 2 raster
	gq := raster.Rasterize(sc, cols*2, rows*2, 4)
	fq, ownq := raster.ToneField(gq, sc)
	if !ownq {
		fq = cells.Lightness(raster.Resolve(gq, sc, 0, 0, cols*2, rows*2))
	}
	Must(os.WriteFile("out/"+name+"-quad.txt", []byte(cells.ToneText(cells.Tone(fq, 2), marks, sc.W, sc.H)), 0o644))
	// the operator's ghostty theme, as it appeared in her screenshots
	ink, paper := color.RGBA{204, 196, 218, 255}, color.RGBA{23, 15, 45, 255}
	var monoCells [][]cells.Cell
	for _, line := range strings.Split(strings.TrimRight(txt, "\n"), "\n") {
		rr := []rune(line)
		row := make([]cells.Cell, cols)
		for i := range row {
			g := ' '
			if i < len(rr) {
				g = rr[i]
			}
			row[i] = cells.Cell{Glyph: g, Fg: ink, Bg: paper}
		}
		monoCells = append(monoCells, row)
	}
	Must(SaveImage("out/"+name+"-text-whole.png", NewCellImage(monoFace, 6, 12).Draw(monoCells)))

	if text != nil {
		Report(os.Stderr, Legible(g, rgb, text))
		LegibleMarks(os.Stderr, cs)
	}
	return cs, g, rgb
}

var _ = canvas.CellAspect
