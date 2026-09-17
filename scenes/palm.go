package scenes

import (
	"github.com/janearc/miami/geom"
	"github.com/janearc/miami/motifs"
	"github.com/janearc/miami/raster"
	"image/color"
)

// Palm is the palm: six fronds with a quadratic bend, because ten merged into a
// solid T and a linear droop read as a spider on a stick.
func Palm() *raster.Scene {
	// leans left, so it frames the left edge of the composite and arches
	// away from the subject rather than into it.
	trunk := geom.Taper{
		P: []geom.Vec{
			{X: 64, Y: 150}, {X: 60, Y: 126}, {X: 53, Y: 100},
			{X: 45, Y: 76}, {X: 40, Y: 60},
		},
		R: []float64{5.0, 4.2, 3.4, 2.8, 2.4},
	}
	shaft := geom.Taper{
		P: []geom.Vec{{X: 41, Y: 66}, {X: 40, Y: 56}},
		R: []float64{2.8, 4.0},
	}
	crown := geom.Vec{X: 40, Y: 56}

	// six fronds, three a side, 35-40 degrees apart at launch. any more and
	// the notches between them fall below one subcell.
	type fs struct{ ang, bend, len, w float64 }
	specs := []fs{
		{-100, -145, 46, 2.8},
		{-140, -105, 38, 2.6},
		{-172, -58, 29, 2.3},
		{-80, 145, 46, 2.8},
		{-40, 105, 38, 2.6},
		{-8, 58, 29, 2.3},
	}
	var fronds []geom.Shape
	for _, f := range specs {
		fronds = append(
			fronds, motifs.Frond(crown, f.ang, f.bend, f.len, f.w),
		)
	}

	nuts := geom.Union(
		geom.Ellipse{C: geom.Vec{X: 37, Y: 64}, Rx: 3.2, Ry: 2.8},
		geom.Ellipse{C: geom.Vec{X: 43, Y: 65}, Rx: 3.0, Ry: 2.6},
	)

	sc := &raster.Scene{W: 84, H: 150}
	sc.Parts = []raster.Part{
		{Role: 'R', Shape: trunk},
		{Role: 'R', Shape: shaft},
		{Role: 'C', Shape: nuts},
		{Role: 'D', Shape: geom.Union(fronds...)}}
	sc.Pal = map[byte]color.RGBA{
		'R': {58, 38, 56, 255},
		'D': {34, 26, 52, 255},
		'C': {46, 32, 50, 255},
		'O': {22, 14, 34, 255},
	}
	sc.Bg = func(
		x, y float64,
	) color.RGBA {
		return color.RGBA{240, 170, 120, 255}
	}
	return sc
}
