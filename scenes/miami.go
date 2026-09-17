package scenes

import (
	"github.com/janearc/miami/canvas"
	"github.com/janearc/miami/geom"
	"github.com/janearc/miami/paint"
	"github.com/janearc/miami/raster"
	"image/color"
	"math"
)

// the composite is defined in physical units: 96 cells wide by 68 tall, where a
// cell is 1 wide and 2 tall. every sub-picture is rasterised at its own column
// count and blitted in, so the cell aspect is still handled in exactly one
// place -- canvas.CellAspect, up in raster.go.

const (
	compCols = 96
	compRows = 31
	horizonY = 47.0 // physical, out of 62
)

var skyStops = []paint.Stop{
	{At: 0.00, C: color.RGBA{38, 16, 78, 255}},
	{At: 0.34, C: color.RGBA{118, 28, 112, 255}},
	{At: 0.60, C: color.RGBA{206, 56, 108, 255}},
	{At: 0.80, C: color.RGBA{244, 116, 88, 255}},
	{At: 1.00, C: color.RGBA{252, 198, 118, 255}}}

var groundStops = []paint.Stop{
	{At: 0.00, C: color.RGBA{138, 84, 100, 255}},
	{At: 0.18, C: color.RGBA{104, 62, 88, 255}},
	{At: 0.45, C: color.RGBA{72, 42, 70, 255}},
	{At: 1.00, C: color.RGBA{38, 20, 44, 255}}}

// miamiSky answers the colour of the empty role anywhere in the frame: the
// gradient, the sun behind the subject, the cloud bars, the water and the road.
// keeping it a function rather than parts is what lets it be smooth.
func miamiSky(fx, fy float64) color.RGBA {
	px, py := fx*compCols, fy*compRows*canvas.CellAspect

	if py >= horizonY {
		t := (py - horizonY) / (compRows*canvas.CellAspect - horizonY)
		g := paint.Ramp(groundStops, t)
		// wet shine on the road directly under the sun
		d := math.Abs(px-48) / 26
		if d < 1 {
			g = paint.Lerp(
				g, color.RGBA{236, 150, 120, 255},
				(1-d)*(1-t*3.2)*0.5,
			)
		}
		if py < horizonY+2.4 { // waterline
			return paint.Lerp(
				color.RGBA{54, 158, 156, 255}, g,
				(py-horizonY)/2.4,
			)
		}
		return g
	}

	c := paint.Ramp(skyStops, py/horizonY)

	// the sun. large, low, and mostly eclipsed by the bird, which is what
	// pushes the bird forward.
	dx, dy := px-48, py-53
	r := math.Sqrt(dx*dx + dy*dy)
	const sunR = 34.0
	core := color.RGBA{255, 244, 196, 255}
	rim := color.RGBA{252, 138, 84, 255}
	switch {
	case r < sunR:
		c = paint.Lerp(core, rim, math.Pow(r/sunR, 2.2))
	case r < sunR+6:
		c = paint.Lerp(
			rim, c, (r-sunR)/6,
		) // a rim wide enough not to stair-step
	}

	// horizontal cloud bars: period correct, and they stop the sky reading
	// as a plain gradient.
	for _, b := range []struct{ y, h, x0, x1, a float64 }{
		{12.5, 1.1, 8, 62, 0.42},
		{18.0, 0.8, 34, 90, 0.30},
		{25.5, 1.3, 2, 44, 0.38},
		{31.0, 0.9, 52, 94, 0.34},
		{37.5, 1.1, 14, 70, 0.30},
	} {
		if px < b.x0 || px > b.x1 {
			continue
		}
		dv := math.Abs(py-b.y) / b.h
		if dv >= 1 {
			continue
		}
		edge := math.Min(px-b.x0, b.x1-px) / 8
		if edge > 1 {
			edge = 1
		}
		a := b.a * (1 - dv*dv) * edge
		c = paint.Lerp(c, color.RGBA{78, 26, 88, 255}, a)
	}
	return c
}

// Miami is the composite frame: the bough the bird sits on, over the sky.
func Miami() *raster.Scene {
	// thin and near black: a backlit branch, not a telephone pole. the
	// centre line is fixed by where the bird's toes land, so only the
	// radius moved.
	bough := geom.Taper{
		P: []geom.Vec{
			{X: 96, Y: 25.5}, {X: 70, Y: 29.6}, {X: 48, Y: 33.2},
			{X: 32, Y: 36.8}, {X: 25, Y: 39.2},
		},
		R: []float64{1.4, 1.25, 1.15, 0.8, 0.25},
	}

	// a low skyline on the waterline. it is the cheapest thing that turns a
	// gradient into a place.
	var city []geom.Shape
	for _, b := range [][3]float64{
		{1, 5, 41}, {6, 4, 44}, {10, 6, 37}, {16, 3, 43}, {19, 5, 40},
		{24, 4, 33}, {28, 6, 42}, {34, 3, 39}, {37, 5, 44}, {42, 4, 35},
		{46, 6, 41}, {52, 3, 43}, {55, 5, 38}, {60, 4, 44}, {64, 6, 40},
		{70, 3, 42}, {73, 5, 36}, {78, 4, 43}, {82, 6, 41}, {88, 3, 44},
		{91, 5, 39},
	} {
		city = append(city, geom.Poly{P: geom.Pts(
			geom.Vec{X: b[0], Y: b[2]}, geom.Vec{
				X: b[0] + b[1], Y: b[2],
			},
			geom.Vec{X: b[0] + b[1], Y: horizonY + 1}, geom.Vec{
				X: b[0], Y: horizonY + 1,
			},
		)})
	}
	sc := &raster.Scene{W: compCols, H: compRows * canvas.CellAspect}
	sc.Parts = []raster.Part{
		{Role: 'c', Shape: geom.Union(city...)},
		{Role: 'b', Shape: bough},
	}
	sc.Pal = map[byte]color.RGBA{
		'b': {52, 32, 44, 255},
		'c': {58, 28, 70, 255},
		'O': {26, 16, 30, 255},
	}
	sc.Bg = miamiSky
	return sc
}
