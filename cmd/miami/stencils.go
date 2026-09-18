package main

import (
	"fmt"

	"github.com/janearc/miami/proof"
	"github.com/janearc/miami/raster"
	"github.com/janearc/miami/scenes"
)

// The verbs that emit a stencil, and the two studies that are only ever
// looked at: a bulb on its own, and the negative-space piece.

// stencil writes one of the built scenes out as a daffy stencil.
func stencil(sc *raster.Scene, name, path string, cols int,
	args []string) {
	if len(args) > 0 {
		fmt.Sscan(args[0], &cols)
	}
	proof.Must(scenes.WriteSprite(sc, name, path, cols))
	fmt.Println("wrote", path)
}

// dolphinStencil emits the dolphin stencil.
func dolphinStencil(args []string) {
	stencil(scenes.DolphinSprite(), "dolphins",
		"sprites/made/dolphins.sprite", 78, args)
}

// bulbStencil emits the bulb stencil, cold.
func bulbStencil(args []string) {
	stencil(scenes.BulbSprite(), "bulb", "sprites/made/bulb.sprite", 44,
		args)
}

// word sets a word as a stencil.
func word(args []string) {
	text, cols := "LIGHTS", 62
	if len(args) > 0 {
		text = args[0]
	}
	if len(args) > 1 {
		fmt.Sscan(args[1], &cols)
	}
	stencil(scenes.WordSprite(text, 100, 5), "word",
		"sprites/made/word.sprite", cols, nil)
}

// bulb is one bulb at a size, rendered and silhouetted.
func bulb(args []string) {
	if len(args) < 2 {
		usage()
	}
	var c int
	var h float64
	fmt.Sscan(args[0], &c)
	fmt.Sscan(args[1], &h)
	sc := scenes.BulbTest(c, h)
	proof.RenderScene(sc, c, 3, fmt.Sprintf("bulb-%d", c))
	// square samples, or the silhouette lies
	g := raster.Rasterize(sc, c*4, int(h*4), 3)
	proof.Must(proof.WritePNG(fmt.Sprintf("out/bulb-%d-sil.png", c),
		proof.Silhouette(g, 'r'), 2))
}

// dolphins is the pod, as a silhouette, in grey, and as a page.
func dolphins(args []string) {
	sc := scenes.Dolphins()
	gw, gh := int(sc.W*5), int(sc.H*5)
	g := raster.Rasterize(sc, gw, gh, 3)
	proof.Must(proof.WritePNG("out/dolphins-sil.png",
		proof.Silhouette(g), 1))
	proof.Must(proof.WritePNG("out/dolphins-grey.png",
		raster.Resolve(g, sc, 0, 0, gw, gh), 1))
	cols := 76
	if len(args) > 0 {
		fmt.Sscan(args[0], &cols)
	}
	proof.Page(sc, cols, fmt.Sprintf("dolphins-%d", cols), nil, nil)
}

// bigBirds renders the negative-space piece at a size, with its knobs.
func bigBirds(args []string) {
	if len(args) < 2 {
		usage()
	}
	var c, r int
	fmt.Sscan(args[0], &c)
	fmt.Sscan(args[1], &r)
	name := fmt.Sprintf("bb-%dx%d", c, r)
	if len(args) > 2 {
		fmt.Sscan(args[2], &scenes.BBWordFrac)
	}
	if len(args) > 4 {
		fmt.Sscan(args[3], &scenes.BBErodeUp)
		fmt.Sscan(args[4], &scenes.BBErodeIter)
	}
	if len(args) > 5 {
		name = args[5]
	}
	proof.RenderScene(scenes.BigBirds(c, r), c, 3, name)
}
