package scenes

import (
	"fmt"
	"image/color"
	"miami/canvas"
	"miami/cells"
	"miami/proof"
	"miami/raster"
	"os"
)

// placement is where a sub-picture goes, in whole terminal cells.
type placement struct {
	name string
	make func() *raster.Scene
	cols int
	ox   int    // column offset
	oy   int    // row offset
	drop []byte // scenery the composite supplies for itself
}

// subH is 3 for sextants and 2 for quadrants. the canvas mapping is uniform, so
// dropping to quadrants needs no other change: aspect is still carried by
// canvas.CellAspect alone.
func BuildMiami(out string, subH int) [][]cells.Cell {
	base := Miami()
	gw, gh := compCols*2, compRows*subH
	g := raster.Rasterize(base, gw, gh, 4)

	plan := []placement{
		{"palm", Palm, 24, 2, 3, nil},
		{"car", Car, 26, 67, 22, nil},
		{"macaw", Macaw, 30, 33, 1, []byte{'P'}},
	}

	pal := map[byte]color.RGBA{}
	for k, v := range base.Pal {
		pal[k] = v
	}
	for _, p := range plan {
		sc := p.make()
		if len(p.drop) > 0 {
			sc.Drop(p.drop...)
		}
		rows := raster.TermRows(sc, p.cols)
		sub := raster.Rasterize(sc, p.cols*2, rows*subH, 4)
		raster.Contour(sub, 'O', 1)
		raster.Blit(g, sub, p.ox*2, p.oy*subH)
		for k, v := range sc.Pal {
			if _, seen := pal[k]; !seen {
				pal[k] = v
			}
		}
		fmt.Fprintf(os.Stderr, "  %-6s %2d x %2d at col %2d row %2d\n",
			p.name, p.cols, rows, p.ox, p.oy)
	}

	merged := &raster.Scene{W: base.W, H: base.H, Pal: pal, Bg: base.Bg}
	rgb := raster.Resolve(g, merged, 0, 0, gw, gh)
	cs := cells.ToCells(rgb, 2, subH)
	// a subcell is (cellW/2) wide by (cellH/subH) tall, so the preview must
	// scale y by 2*canvas.CellAspect/subH times x or it lies about the
	// shape.
	sx := 6
	sy := int(float64(sx) * 2 * canvas.CellAspect / float64(subH))
	proof.Must(proof.WritePNGXY("out/"+out+".png", cells.CellsToPixels(cs, 2, subH), sx, sy))
	proof.Must(os.WriteFile("out/"+out+"-roles.txt", []byte(raster.RoleText(g)), 0o644))
	return cs
}
