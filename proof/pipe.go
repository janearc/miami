package proof

import (
	"fmt"
	"os"

	"miami/canvas"
	"miami/cells"
	"miami/raster"
)

// RenderScene runs one scene all the way to cells and drops the three review
// artefacts: the picture, the diffable role grid, and the escape sequences.
func RenderScene(sc *raster.Scene, cols, subH int, name string) [][]cells.Cell {
	rows := raster.TermRows(sc, cols)
	gw, gh := cols*2, rows*subH
	g := raster.Rasterize(sc, gw, gh, 4)
	rgb := raster.Resolve(g, sc, 0, 0, gw, gh)
	cs := cells.ToCells(rgb, 2, subH)
	sx := 6
	sy := int(float64(sx) * 2 * canvas.CellAspect / float64(subH))
	Must(WritePNGXY("out/"+name+".png", cells.CellsToPixels(cs, 2, subH), sx, sy))
	Must(os.WriteFile("out/"+name+"-roles.txt", []byte(raster.RoleText(g)), 0o644))
	Must(os.WriteFile("out/"+name+".ansi", []byte(cells.ANSI(cs)), 0o644))
	fmt.Fprintf(os.Stderr, "%-16s %3d x %3d cells\n", name, cols, rows)
	return cs
}

// Must is for the command line and the proof harness only; libraries return
// their errors.
func Must(err error) {
	if err != nil {
		panic(err)
	}
}
