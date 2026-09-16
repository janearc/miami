package scenes

import (
	"image/color"

	"miami/geom"
	"miami/raster"
)

// A cassowary, from geometry rather than from a photograph.
//
// There is no photograph of one in this tree, and that turns out to be no loss:
// a cassowary is almost all silhouette. Three things say cassowary and nothing
// else says it -- the casque standing up off the skull like a blade, the bare
// blue neck, and the two red wattles hanging at the throat.
//
// Get those and the bird reads at any size; miss the casque and it is a large
// dark bird of no particular kind.
//
// So it is drawn as a head and neck rather than a whole animal. At seventeen
// rows the whole bird would put the casque at two rows tall, which is the one
// place the detail cannot be spent.
//
// The palette is the bird's own and it happens to suit what the tool is: the
// blue is cold, the wattles are the only warm thing in the frame, and the
// plumage is near-black rather than black so it is a mass and not a hole.
func Cassowary() *raster.Scene {
	const w, h = 100.0, 45.0
	at := func(x, y float64) geom.Vec { return geom.Vec{X: x, Y: y} }

	// the head fills the frame and the neck only enters at the bottom. at
	// seventeen rows every feature is worth three or four of them, so a
	// whole bird would spend the casque -- the one thing that identifies it
	// -- on two rows of tan. so: a portrait.
	skull := geom.Ellipse{C: at(34.0, 20.6), Rx: 7.8, Ry: 7.0, Rot: -10}

	// the casque: a narrow plate standing up off the crown and leaning
	// back, as tall as the skull again. an early version made it round and
	// as wide as the head, which turns a cassowary into a duck. the base
	// sits on the crown and no lower.
	//
	// dipped into the skull it eats the blue face, and a cassowary without
	// a blue face is a helmet.
	casque := geom.Poly{P: geom.Pts(
		at(30.2, 14.6), at(31.0, 8.0), at(32.8, 2.4), at(36.2, 1.6),
		at(39.4, 3.8), at(40.4, 8.8), at(39.6, 13.2), at(38.0, 15.0),
	)}

	// the bill: heavy at the face, tapering forward, turned down at the tip
	bill := geom.Union(
		geom.Taper{P: []geom.Vec{at(28.0, 23.4), at(16.2, 26.4)},
			R: []float64{2.6, 0.9}},
		geom.Poly{P: geom.Pts(at(18.2, 25.6), at(15.2, 26.8), at(16.6, 28.8), at(19.6, 27.4))},
	)

	// the neck, entering from below and running off the frame
	neck := geom.Taper{
		P: []geom.Vec{at(35.0, 24.5), at(34.2, 32.0), at(31.6, 46.5)},
		R: []float64{5.8, 6.8, 8.8},
	}

	// two wattles, hanging separately off the front of the throat. drawn as
	// one shape they read as a bib; the gap between them is what makes them
	// wattles, so it is kept wider than a cell.
	wattles := geom.Union(
		geom.Taper{P: []geom.Vec{at(31.0, 28.5), at(29.0, 38.5)}, R: []float64{2.0, 2.6}},
		geom.Taper{P: []geom.Vec{at(35.6, 29.0), at(35.0, 40.0)}, R: []float64{1.8, 2.3}},
	)

	// the nape: the back of the neck goes violet where the blue runs out
	nape := geom.Intersect(neck, geom.HalfPlane{C: at(36.4, 0), N: geom.Vec{X: -1}})

	// the body, only suggested: a dark shoulder at the corner, so the neck
	// is attached to something
	body := geom.Ellipse{C: at(48, 52), Rx: 20, Ry: 13, Rot: -16}

	sc := &raster.Scene{W: w, H: h}
	sc.Parts = []raster.Part{
		{Role: 'p', Shape: body},
		{Role: 'n', Shape: neck},
		{Role: 'v', Shape: nape},
		{Role: 'w', Shape: wattles},
		{Role: 'n', Shape: skull},
		{Role: 'c', Shape: casque},
		{Role: 'b', Shape: bill},
		{Role: 'r', Shape: geom.Ellipse{C: at(31.0, 19.4), Rx: 2.1, Ry: 1.9}},
		{Role: 'e', Shape: geom.Ellipse{C: at(31.0, 19.4), Rx: 1.0, Ry: 0.9}}}
	sc.Pal = map[byte]color.RGBA{
		'p': {26, 22, 30, 255},   // plumage, near-black so it is a mass not a hole
		'n': {46, 130, 208, 255}, // the bare blue neck
		'v': {74, 62, 150, 255},  // the violet nape
		'w': {206, 46, 58, 255},  // wattles
		'c': {198, 158, 96, 255}, // the casque, horn
		'b': {120, 106, 96, 255}, // bill
		'r': {236, 198, 96, 255}, // the eye's ring
		'e': {18, 14, 16, 255},   // pupil
	}
	sc.MinCover = map[byte]float64{'e': 0.25, 'r': 0.30, 'b': 0.30}
	sc.Bg = func(x, y float64) color.RGBA { return color.RGBA{16, 14, 24, 255} }
	return sc
}
