package scenes

import (
	"github.com/janearc/miami/geom"
	"github.com/janearc/miami/raster"
	"image/color"
)

// mid-eighties italian wedge, side on, facing left. the read is the profile
// line, not the badge: blank low nose, one straight rise to the screen base,
// short flat roof set well back, chopped tail, dark band along the flank where
// the strakes are.

// Car is the sports car, from a real Testarossa's dimensions: the proportions
// are what make it read, and guessing them made it a wedge.
func Car() *raster.Scene {
	// profile scaled from the real thing: 4485 long, 1130 tall, 2550
	// wheelbase, 975 front overhang. eyeballing it produced a car sitting
	// far too low with its wheels cut clean off the body.
	shell := geom.Poly{P: geom.Pts(
		geom.Vec{
			X: 14, Y: 63.5,
		}, // nose tip, low and blank: pop-up lamps are down
		geom.Vec{X: 26, Y: 57},
		geom.Vec{X: 44, Y: 48},
		geom.Vec{X: 64, Y: 45},
		geom.Vec{
			X: 92, Y: 43.5,
		}, // screen base, bonnet almost flat behind the rise
		geom.Vec{X: 118, Y: 36.5},
		geom.Vec{X: 150, Y: 36.5}, // short flat roof set well back
		geom.Vec{X: 168, Y: 38.5},
		geom.Vec{X: 186, Y: 39.5},
		geom.Vec{X: 187, Y: 58}, // chopped tail
		geom.Vec{X: 182, Y: 70},
		geom.Vec{X: 150, Y: 73},
		geom.Vec{X: 60, Y: 74},
		geom.Vec{X: 30, Y: 75},
		geom.Vec{
			X: 15, Y: 75,
		}, // blunt nose face: a knife edge reads as a doorstop
	)}

	fwCentre := geom.Vec{X: 51.5, Y: 67.3}
	rwCentre := geom.Vec{X: 149.5, Y: 67.3}
	archF := geom.Ellipse{C: fwCentre, Rx: 14.0, Ry: 14.0}
	archR := geom.Ellipse{C: rwCentre, Rx: 14.8, Ry: 14.8}
	body := geom.Sub(shell, geom.Union(archF, archR))

	tyres := geom.Union(
		geom.Ellipse{C: fwCentre, Rx: 12.7, Ry: 12.7},
		geom.Ellipse{C: rwCentre, Rx: 13.4, Ry: 13.4},
	)
	hubs := geom.Union(
		geom.Ellipse{C: fwCentre, Rx: 6.6, Ry: 6.6},
		geom.Ellipse{C: rwCentre, Rx: 7.2, Ry: 7.2},
	)

	// the strake bank. individual slats cannot resolve at thirty columns,
	// so the band carries the read and the slats sit inside it for the sake
	// of the high resolution pass.
	bank := geom.Poly{P: geom.Pts(
		geom.Vec{X: 100, Y: 48}, geom.Vec{X: 152, Y: 44},
		geom.Vec{X: 153, Y: 60}, geom.Vec{X: 102, Y: 63},
	)}
	var slats []geom.Shape
	for i := 0; i < 4; i++ {
		y := 50.0 + float64(i)*3.6
		slats = append(slats, geom.Taper{
			P: []geom.Vec{
				{X: 103, Y: y + 0.5}, {X: 149, Y: y - 1.3},
			},
			R: []float64{0.9, 0.9},
		})
	}

	glass := geom.Poly{P: geom.Pts(
		geom.Vec{X: 95, Y: 43}, geom.Vec{X: 120, Y: 37.5},
		geom.Vec{X: 148, Y: 37}, geom.Vec{X: 144, Y: 46},
	)}

	shadow := geom.Ellipse{C: geom.Vec{X: 101, Y: 80.5}, Rx: 84, Ry: 4.0}

	sc := &raster.Scene{W: 200, H: 92}
	sc.Parts = []raster.Part{
		{Role: 'x', Shape: shadow},
		{Role: 'K', Shape: tyres},
		{Role: 'J', Shape: hubs},
		{Role: 'A', Shape: body},
		{Role: 'V', Shape: geom.Intersect(bank, body)},
		{Role: 'k', Shape: geom.Intersect(geom.Union(slats...), body)},
		{Role: 'Z', Shape: glass}}
	sc.Pal = map[byte]color.RGBA{
		'A': {240, 238, 232, 255},
		'Z': {40, 46, 66, 255},
		'K': {28, 26, 30, 255},
		'J': {174, 176, 182, 255},
		'V': {152, 150, 158, 255},
		'k': {80, 78, 88, 255},
		'x': {58, 40, 54, 255},
		'O': {26, 18, 30, 255},
	}
	sc.Bg = func(
		x, y float64,
	) color.RGBA {
		return color.RGBA{240, 170, 120, 255}
	}
	return sc
}
