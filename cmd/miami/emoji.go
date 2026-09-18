package main

import (
	"bytes"
	"fmt"
	"image/color"
	"image/png"
	"math"

	"github.com/janearc/miami/cells"
	"github.com/janearc/miami/emoji"
	"github.com/janearc/miami/paint"
	"github.com/janearc/miami/proof"
	"github.com/janearc/miami/typeset"
)

// emojiCells draws an emoji as cells, with a wordmark beside it if one is
// asked for.
//
// The rune is given as a hex codepoint, because argv and a terminal between
// them will not reliably deliver an astral character.
func emojiCells(args []string) {
	if len(args) < 1 {
		usage()
	}
	var cp int64
	fmt.Sscanf(args[0], "%x", &cp)
	cols, n, word, fig := 44, 8, "", "smslant"
	if len(args) > 1 {
		fmt.Sscan(args[1], &cols)
	}
	if len(args) > 2 {
		fmt.Sscan(args[2], &n)
	}
	if len(args) > 3 {
		word = args[3]
	}
	if len(args) > 4 {
		fig = args[4]
	}
	bm, err := emoji.PNG(rune(cp))
	proof.Must(err)
	img, err := png.Decode(bytes.NewReader(bm.PNG))
	proof.Must(err)
	// keep the picture square: a cell is twice as tall as it is wide, so
	// the row count is the column count scaled by the image's own aspect
	// and then halved. that is the only place the aspect is applied.
	ib := img.Bounds()
	iw, ih := float64(ib.Dx()), float64(ib.Dy())
	rows := int(float64(cols)*ih/iw/2 + 0.5)
	// palette first, then snap, then vote down: reducing after sampling
	// lets the source's anti-aliased edges into the palette
	px := cells.SampleFlat(img, cols*2, rows*3, n, 0.055)

	// the wordmark is figlet letterforms placed as characters, not type
	// rasterised into the block-glyph fitter: at this size the fitter
	// cannot hold a letterform and the N comes out as an H
	var art []string
	if word != "" {
		ff, err := typeset.LoadFig(fig)
		proof.Must(err)
		art = typeset.Trim(ff.Render(word))
		wide := 0
		for _, l := range art {
			wide = max(wide, len(l))
		}
		px = cells.Widen(px, (cols+2+wide+1)*2)
	}
	cs := cells.ToCellsAlpha(px, 2, 3, 128)
	if art != nil {
		ramp := paint.ByLightness(cells.PaletteOf(px, n, 0.055))
		top := (len(cs) - len(art)) / 2
		last := math.Max(1, float64(len(art)-1))
		cells.Letters(cs, art, cols+2, top,
			func(x, y int) color.RGBA {
				return paint.Hot(ramp, float64(y)/last)
			})
	}
	name := fmt.Sprintf("out/emoji-%X", cp)
	proof.Must(write(name+".ansi", []byte(cells.ANSIAlpha(cs))))
	// the same cells on two grounds, because "the background is
	// transparent" is a claim about what the terminal supplies and the only
	// way to see it is to supply two different things.
	//
	// drawn through the cell-image renderer rather than by expanding the
	// block masks: a literal letter is not a sextant, so the mask path
	// draws nothing where the wordmark is and the preview quietly lies
	// about what the file contains.
	for _, g := range []struct {
		suffix string
		ground color.RGBA
	}{
		{"", color.RGBA{18, 16, 28, 255}},
		{"-light", color.RGBA{238, 236, 230, 255}},
	} {
		proof.Must(proof.SaveImage(name+g.suffix+".png",
			proof.NewCellImageOn(
				proof.Mono(), 14, 28, g.ground,
			).Draw(cs)))
	}
	fmt.Printf("%U at %dx%d cells (ppem %d) -> %s.ansi\n",
		cp, len(cs[0]), len(cs), bm.PPEM, name)
}
