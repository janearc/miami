package scenes

import (
	"github.com/janearc/miami/geom"
	"github.com/janearc/miami/raster"
	"image/color"
)

// scarlet macaw, profile facing left, perched on a diagonal branch. every
// number here is a proportion of the 100x200 canvas; tuning the bird means
// editing one of them, not retyping rows.

// Macaw is the parrot: the piece this library started from, and the reason the
// silhouette test exists.
func Macaw() *raster.Scene {
	// -- skull. sits directly on the shoulders; the crown is a low dome
	// that must visibly overhang the bill base or the bird reads as a
	// hornbill.
	head := geom.Ellipse{C: geom.Vec{X: 38, Y: 26}, Rx: 15.5, Ry: 14.5}
	nape := geom.Taper{P: []geom.Vec{{X: 41, Y: 34}, {X: 50, Y: 50}}, R: []float64{14, 18}}
	// throat runs unbroken from the gape to the breast; without it the head
	// reads as a separate object stuck onto the body.
	throat := geom.Taper{P: []geom.Vec{{X: 31, Y: 39}, {X: 36, Y: 50}, {X: 41, Y: 60}}, R: []float64{8.5, 12, 14}}
	body := geom.Ellipse{C: geom.Vec{X: 55, Y: 62}, Rx: 19.5, Ry: 27, Rot: 10}
	belly := geom.Ellipse{C: geom.Vec{X: 44, Y: 68}, Rx: 13.5, Ry: 17}
	vent := geom.Taper{P: []geom.Vec{{X: 56, Y: 84}, {X: 63, Y: 95}}, R: []float64{12, 7}}
	mass := geom.Union(head, nape, throat, body, belly, vent)

	// -- bare face patch behind the bill, eye inside it. brightest value on
	// the animal and the feature that says macaw rather than parrot.
	face := geom.Ellipse{C: geom.Vec{X: 40, Y: 25.5}, Rx: 10, Ry: 7.5, Rot: -12}
	iris := geom.Ellipse{C: geom.Vec{X: 38.5, Y: 23}, Rx: 2.3, Ry: 2.3}
	pupil := geom.Ellipse{C: geom.Vec{X: 38.5, Y: 23}, Rx: 1.2, Ry: 1.2}

	// -- upper mandible. culmen is a tight quarter circle of about one head
	// height radius; the hook curls back under, so the leading edge is the
	// culmen bulge and the tip sits behind and below it, past the chin.
	culmen := geom.Arc(25.24, 36, 16.24, 280, 162, 14)
	upper := geom.Poly{P: geom.Cat(
		culmen,
		geom.Pts(
			geom.Vec{X: 10.0, Y: 45.0},
			geom.Vec{X: 12.0, Y: 50.0}, // tip. deep enough to survive the cell grid
			geom.Vec{X: 15.5, Y: 47.5},
			geom.Vec{X: 18.0, Y: 41.5},
			geom.Vec{X: 19.5, Y: 36.5}, // notch under the hook
			geom.Vec{X: 23.5, Y: 35.6},
			geom.Vec{X: 29.0, Y: 35.0}, // gape
			geom.Vec{X: 31.5, Y: 27.5},
			geom.Vec{X: 30.0, Y: 20.5},
		),
	)}
	// lower mandible sits BEHIND the hook so the pale tip reads against the
	// background rather than against black.
	lower := geom.Poly{P: geom.Pts(
		geom.Vec{X: 30, Y: 35.5}, geom.Vec{X: 24, Y: 37.5}, geom.Vec{X: 20.5, Y: 39.5}, geom.Vec{X: 19.5, Y: 43.5},
		geom.Vec{X: 22, Y: 46.5}, geom.Vec{X: 27, Y: 46}, geom.Vec{X: 31, Y: 43},
	)}

	// -- folded wing. one outlined shape cut into scarlet coverts, the gold
	// band, a green wash and the blue flights.
	//
	// the cut normal is tilted off the wing axis because real covert tracts
	// run DIAGONALLY across the folded wing; cutting square to the axis
	// gives horizontal stripes and the bird reads as a beach towel.
	sh := geom.Vec{X: 50, Y: 44}
	nrm := geom.Vec{X: 0.423, Y: 0.906}
	const bandR = 70.0
	band := func(t float64) geom.Shape {
		return geom.Ellipse{
			C:  geom.Vec{X: sh.X + nrm.X*(t-bandR), Y: sh.Y + nrm.Y*(t-bandR)},
			Rx: bandR, Ry: bandR,
		}
	}
	wingPath := []geom.Vec{{X: 50, Y: 44}, {X: 56, Y: 58}, {X: 61, Y: 75}, {X: 66, Y: 91}, {X: 69, Y: 103}}
	wingR := []float64{12, 12.5, 11, 7, 2.5}
	wing := geom.Taper{P: wingPath, R: wingR}
	edgeR := make([]float64, len(wingR))
	for i, r := range wingR {
		edgeR[i] = r + 1.0
	}
	wingEdge := geom.Taper{P: wingPath, R: edgeR}
	coverts := geom.Intersect(wing, band(24))
	goldBand := geom.Sub(geom.Intersect(wing, band(35)), band(24))
	greenWash := geom.Sub(geom.Intersect(wing, band(39)), band(35))
	flights := geom.Sub(wing, band(39))

	// -- tail longer than the bird: two central rectrices past the rest.
	tailA := geom.Taper{P: []geom.Vec{{X: 63, Y: 86}, {X: 69, Y: 110}, {X: 72, Y: 137}, {X: 73, Y: 164}}, R: []float64{7, 5, 3, 1.2}}
	tailB := geom.Taper{P: []geom.Vec{{X: 59, Y: 84}, {X: 67, Y: 112}, {X: 73, Y: 147}, {X: 76, Y: 180}}, R: []float64{9, 7.5, 4.5, 1.8}}
	blueTip := geom.Intersect(geom.Union(tailA, tailB),
		geom.HalfPlane{C: geom.Vec{X: 69, Y: 143}, N: geom.Vec{X: -0.174, Y: -0.985}})

	// -- perch and zygodactyl grip.
	perch := geom.Taper{P: []geom.Vec{{X: 0, Y: 112}, {X: 100, Y: 96}}, R: []float64{6, 5}}
	leg := geom.Taper{P: []geom.Vec{{X: 48, Y: 82}, {X: 47, Y: 99}}, R: []float64{5.5, 4.5}}
	toes := geom.Union(
		geom.Taper{P: []geom.Vec{{X: 47, Y: 98}, {X: 40, Y: 103}, {X: 38, Y: 108}}, R: []float64{4, 2.6, 1.8}},
		geom.Taper{P: []geom.Vec{{X: 47, Y: 98}, {X: 54, Y: 103}, {X: 56, Y: 107}}, R: []float64{4, 2.6, 1.8}},
	)

	sc := &raster.Scene{W: 100, H: 200}
	sc.Parts = []raster.Part{
		{Role: 'P', Shape: perch},
		{Role: 'T', Shape: tailA},
		{Role: 'T', Shape: tailB},
		{Role: 'Q', Shape: blueTip},
		{Role: 'L', Shape: leg},
		{Role: 'L', Shape: toes},
		{Role: 'S', Shape: mass},
		{Role: 'M', Shape: geom.Intersect(mass, geom.Ellipse{C: geom.Vec{X: 64, Y: 54}, Rx: 22, Ry: 34, Rot: 10})},
		{Role: 'O', Shape: wingEdge},
		{Role: 'W', Shape: coverts},
		{Role: 'Y', Shape: goldBand},
		{Role: 'G', Shape: greenWash},
		{Role: 'U', Shape: flights},
		{Role: 'F', Shape: face},
		{Role: 'I', Shape: iris},
		{Role: 'E', Shape: pupil},
		{Role: 'N', Shape: lower},
		{Role: 'B', Shape: upper}}
	sc.Pal = map[byte]color.RGBA{
		'S': {236, 58, 50, 255},
		'M': {188, 26, 38, 255},
		'F': {245, 238, 230, 255},
		'I': {238, 214, 130, 255},
		'E': {26, 20, 22, 255},
		'B': {232, 224, 205, 255},
		'N': {36, 30, 32, 255},
		'W': {172, 24, 36, 255},
		'G': {32, 122, 82, 255},
		'Y': {250, 190, 40, 255},
		'U': {36, 78, 190, 255},
		'T': {206, 34, 40, 255},
		'Q': {40, 86, 200, 255},
		'L': {34, 30, 32, 255},
		'P': {124, 86, 56, 255},
		'O': {26, 16, 28, 255},
	}
	sc.Bg = func(x, y float64) color.RGBA { return color.RGBA{200, 205, 210, 255} }
	return sc
}
