package main

import (
	"fmt"
	"image/color"
	"math"
	"strings"

	"github.com/janearc/miami/cells"
	"github.com/janearc/miami/grade"
	"github.com/janearc/miami/paint"
	"github.com/janearc/miami/proof"
	"github.com/janearc/miami/raster"
	"github.com/janearc/miami/scenes"
	"github.com/janearc/miami/typeset"
)

// card is one piece of art per repository, seventy-five by seventeen, with a
// figlet wordmark set as characters beside it. Colour escapes only:
//
// these are cat'd inside a readme and in a release tree, so nothing here may
// move the cursor or clear the screen.
func card(args []string) {
	if len(args) < 1 {
		usage()
	}
	name := args[0]
	const cols, rows = 75, 17
	cs, pal, px := cardArt(name, cols, rows)

	// libtheme-css carries no wordmark at all. the card's whole argument is
	// that the library owns no form, and it sits under a heading that
	// already says its name; a word punched into the ramps was the one
	// thing on it pretending to be an object.
	word := map[string]string{
		"cassowary": "cassowary", "bender": "bender",
		"libtheme-css": "libtheme-css",
	}[name]
	// the art moves over to make room for type that can actually be read.
	// contessa fitted the gap at three rows and held none of its
	// letterforms: cassowary came out as punctuation. fitting the space was
	// the wrong thing to optimise.
	shift := map[string]int{"cassowary": 11, "bender": 12}[name]
	if shift > 0 {
		cells.ShiftLeft(cs, shift, color.RGBA{16, 14, 24, 255})
	}
	wx := map[string]int{
		"cassowary": 26, "bender": 40, "libtheme-css": 2,
	}[name]
	wy := map[string]int{
		"cassowary": 6, "bender": 6, "libtheme-css": 0,
	}[name]
	var art []string
	if word != "" {
		ff, err := typeset.LoadFig("small")
		proof.Must(err)
		art = typeset.Trim(ff.Render(word))
	}
	if art != nil {
		cells.Letters(cs, art, wx, wy, cardInk(name, pal, art))
	}

	proof.Must(write("dist/"+name+".ansi", []byte(cells.ANSI(cs))))
	plain(name, px, art, wx, wy)
	proof.Must(proof.SaveImage("out/card-"+name+".png",
		proof.NewCellImageOn(
			proof.Mono(), 9, 18, color.RGBA{16, 14, 24, 255},
		).Draw(cs)))
	fmt.Printf("dist/%s.ansi  %d cols x %d rows\n", name, cols, rows)
}

// cardArt is the picture: the cells to write, the palette to colour type
// from, and the subpixels the plain companion is made of.
func cardArt(name string, cols, rows int) ([][]cells.Cell, []color.RGBA,
	[][]color.RGBA) {
	if name == "libtheme-css" {
		sh := grade.Sextants(cols, rows)
		px := sh.Paint(scenes.Libtheme(sh))
		pal := cells.PaletteOf(px, 10, 0.05)
		return cells.ToCells(px, 2, 3), pal, px
	}
	sc := scenes.Bender()
	if name == "cassowary" {
		sc = scenes.Cassowary()
	}
	g := raster.Rasterize(sc, cols*2, rows*3, 4)
	px := raster.Resolve(g, sc, 0, 0, cols*2, rows*3)
	// the scene's own declared palette, not the palette of the rendered
	// pixels:
	//
	// most of this card is background, so sampling the render gives a ramp
	// made of ground and the word comes out the colour of the dark it is
	// sitting on
	var pal []color.RGBA
	for r, c := range sc.Pal {
		if r != 'i' && r != 'e' {
			pal = append(pal, c)
		}
	}
	return cells.ToCells(px, 2, 3), pal, px
}

// cardInk is what colour the wordmark is, at a place in it.
func cardInk(name string, pal []color.RGBA,
	art []string) func(x, y int) color.RGBA {
	if name == "libtheme-css" {
		// coloured from the ramps it stands above, column by column, so
		// the mark is the palette: a colour library's wordmark ought
		// to be made of the thing the library does.
		marks := scenes.LibthemeMarks
		wide := 0
		for _, l := range art {
			if len(l) > wide {
				wide = len(l)
			}
		}
		// blended rather than stepped: six hard switches across a word
		// land in the middle of letters, so a single glyph comes out
		// half gold and half blue and reads as a fault rather than a
		// spectrum
		return func(x, _ int) color.RGBA {
			t := float64(x) / math.Max(1, float64(wide-1)) *
				float64(len(marks)-1)
			i := int(t)
			if i >= len(marks)-1 {
				return paint.Hex(marks[len(marks)-1])
			}
			return paint.Blend(marks[i], marks[i+1], t-float64(i))
		}
	}
	// only the bright half of the palette: Hot runs from the lightest
	// colour to the middle one, and on a card that is mostly dark ground
	// the middle one is nearly the ground itself
	ramp := paint.ByLightness(pal)
	if len(ramp) > 3 {
		ramp = ramp[len(ramp)/2:]
	}
	last := math.Max(1, float64(len(art)-1))
	return func(x, y int) color.RGBA {
		return paint.Hot(ramp, float64(y)/last)
	}
}

// plain writes the card's companion without colour, because markdown needs
// four spaces of indent to hold art preformatted and escapes do not survive
// that. The picture has to be carried by lightness alone:
//
// the same subpixels, fitted to sextant masks against how bright each one is.
// The wordmark is stamped after, as the characters it already was.
func plain(name string, px [][]color.RGBA, art []string, wx, wy int) {
	lum := make([][]float64, len(px))
	lo, hi := math.Inf(1), math.Inf(-1)
	for y, row := range px {
		lum[y] = make([]float64, len(row))
		for x, c := range row {
			v := paint.Luminance(c)
			lum[y][x] = v
			lo, hi = math.Min(lo, v), math.Max(hi, v)
		}
	}
	// stretched across the range the picture actually uses. a cassowary
	// painted in mid blues and tans occupies a third of the scale, and left
	// unstretched every one of them fits the same shade:
	//
	// the bird came out as one flat patch of light grey with a word next to
	// it.
	if hi > lo {
		for y := range lum {
			for x := range lum[y] {
				lum[y][x] = math.Pow((lum[y][x]-lo)/(hi-lo),
					0.75)
			}
		}
	}
	tone := cells.Tone(lum, 3)
	for y, line := range art {
		for x, ch := range line {
			if ch == ' ' {
				continue
			}
			r, c := wy+y, wx+x
			if r >= len(tone) || c >= len(tone[r]) {
				continue
			}
			tone[r][c] = ch
		}
	}
	var out strings.Builder
	for _, row := range tone {
		out.WriteString(strings.TrimRight(string(row), " ") + "\n")
	}
	proof.Must(write("dist/"+name+"-plain.txt", []byte(out.String())))
}
