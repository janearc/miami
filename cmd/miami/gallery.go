package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/janearc/miami/canvas"
	"github.com/janearc/miami/cells"
	"github.com/janearc/miami/proof"
	"github.com/janearc/miami/raster"
	"github.com/janearc/miami/scenes"
	"github.com/janearc/miami/typeset"
)

// The built scenes: the ones with a verb of their own, and the ones named on
// the command line and rendered whichever way is asked for.

// single are the scenes a name on the command line may ask for.
var single = map[string]func() *raster.Scene{
	"macaw":     scenes.Macaw,
	"palm":      scenes.Palm,
	"car":       scenes.Car,
	"miami":     scenes.Miami,
	"cassowary": scenes.Cassowary,
}

// lights is the roadside sign.
func lights([]string) {
	sc, marks := scenes.Lights(canvas.Manpage)
	proof.Page(sc, canvas.Manpage.Cols, "lights", scenes.LightsText, marks)
}

// specimen is the type specimen, at manpage width.
func specimen([]string) {
	proof.RenderScene(scenes.Specimen(canvas.Manpage), canvas.Manpage.Cols,
		3, "specimen")
}

// faces lists what letterforms a figlet file holds.
func faces(args []string) {
	for _, f := range args {
		names, err := typeset.Faces(f)
		proof.Must(err)
		fmt.Printf("%s\n  %s\n", f, strings.Join(names, "\n  "))
	}
}

// miamiQuad is the composite scene at quarter height.
func miamiQuad([]string) {
	fmt.Print(cells.ANSI(scenes.BuildMiami("miami-quad", 2)))
}

// miami is the composite scene, or the scene of that name rendered whichever
// way was asked for.
func miami(args []string) {
	if len(args) == 0 || args[0] == "term" {
		fmt.Print(cells.ANSI(scenes.BuildMiami("miami", 3)))
		return
	}
	renderSingle("miami", args)
}

// renderSingle draws one built scene as a png, a silhouette, its roles, or
// cells for a terminal.
func renderSingle(name string, args []string) {
	mk, ok := single[name]
	if !ok {
		unknown(name)
	}
	if len(args) < 1 {
		usage()
	}
	sc := mk()
	cols := 34
	if len(args) > 1 {
		fmt.Sscan(args[1], &cols)
	}
	switch mode := args[0]; mode {
	case "png", "sil", "roles":
		gw, gh := int(sc.W*2), int(sc.H*2)
		g := raster.Rasterize(sc, gw, gh, 3)
		switch mode {
		case "sil":
			proof.Must(proof.WritePNG("out/"+name+"-sil.png",
				proof.Silhouette(g, 'P', 'x'), 2))
		case "roles":
			fmt.Print(raster.RoleText(g))
		default:
			raster.Contour(g, 'O', 1)
			proof.Must(proof.WritePNG("out/"+name+".png",
				raster.Resolve(g, sc, 0, 0, gw, gh), 2))
		}
	case "term":
		rows := raster.TermRows(sc, cols)
		gw, gh := cols*2, rows*3
		g := raster.Rasterize(sc, gw, gh, 4)
		raster.Contour(g, 'O', 1)
		cs := cells.ToCells(raster.Resolve(g, sc, 0, 0, gw, gh), 2, 3)
		fmt.Print(cells.ANSI(cs))
		proof.Must(proof.WritePNGXY("out/"+name+"-term.png",
			cells.CellsToPixels(cs, 2, 3), 7, 9))
		fmt.Fprintf(os.Stderr, "%d cols x %d rows\n", cols, rows)
	}
}

// unknown says a name is not a scene and stops, which is what a mistyped verb
// arrives as.
func unknown(name string) {
	fmt.Println("unknown scene", name)
	os.Exit(2)
}
