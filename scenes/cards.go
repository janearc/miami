package scenes

import (
	"image/color"

	"github.com/janearc/libdaffy/field"
	"github.com/janearc/libtheme-css/primitives/functions"
	"github.com/janearc/libtheme-css/spaces/ok"
	"github.com/janearc/libtheme-css/spaces/srgb"

	"miami/geom"
	"miami/grade"
	"miami/raster"
)

// The cards: one piece of art per repository, seventy-five columns by
// seventeen rows, to sit beside a readme.
//
// Two constraints shape every one of them. They are cat'd inside a readme and
// inside a release tree, so nothing may move the cursor or clear the screen --
// colour escapes and newlines only. And they have to say what the thing is at
// a glance, on a card the size of a business card, which mostly means: pick
// the one feature nothing else has and spend the resolution on that.

// Libtheme is the colour library, and it has no form on purpose.
//
// libtheme-css is authoritative on colour for the whole suite and owns no
// geometry at all -- it stores as css because css is a container, not because
// anybody is in the css business. So its card is colour and nothing else: no
// object, no silhouette, nothing to recognise. Six ramps, each mixed in oklab,
// which is the library's own line between two colours and the reason a
// magenta-to-orange run does not go through mud on the way.
//
// Drawing a paint pot or a palette would have been easy and would have been a
// lie about what the thing does.
// Band is the share of the card the wordmark's ground takes, off the top.
// Type wants a ground, not a gradient: set over the ramps the letters had to
// be punched out in near-black and read as damage to the artwork. Giving them
// their own rows is the ordinary typographic answer and it was porter's.
const Band = 5.0 / 17.0

// LibthemeMarks are the six ramps' top notes, in order. The wordmark is
// coloured from them column by column, so the mark is made of the palette it
// sits above rather than of a colour somebody liked.
var LibthemeMarks = []string{"ff8f4a", "9df7f2", "ff8ad0", "f5c542", "8fc0f2", "c9c2cc"}

func Libtheme(sh grade.Sheet) grade.Stack {
	// each band is one ramp, and the bands are the whole picture
	runs := [][]string{
		{"1b1040", "5a2071", "b8446b", "ff8f4a", "ffd08a"},
		{"04202a", "0e5a66", "2de2e6", "9df7f2", "e8fffd"},
		{"2a0b44", "6a2f8f", "c44fb0", "ff8ad0", "ffd6ee"},
		{"3a1a06", "8a4410", "e08a22", "f5c542", "fff0b8"},
		{"07142a", "16386e", "3f7ad6", "8fc0f2", "dcebff"},
		{"140f14", "3a3038", "6e6068", "aaa0a6", "eae4e8"},
	}
	// the sheet centres its unit square in the long direction, so on a card
	// this wide v runs from about a fifth to about four fifths and never
	// touches nought or one. bands cut at 0..1 lose the top and bottom one.
	_, v0 := sh.UV(0, 0)
	_, v1 := sh.UV(0, sh.H-1)
	span := v1 - v0
	// the ramps start below the wordmark's band
	v0 += span * Band
	span = v1 - v0

	var st grade.Stack
	n := float64(len(runs))
	for i, run := range runs {
		lo := v0 + span*float64(i)/n
		hi := v0 + span*float64(i+1)/n
		if i == len(runs)-1 {
			hi = v1 + 1 // the last band owns the last row outright
		}
		stops := make([]any, 0, len(run)*2)
		for k, c := range run {
			stops = append(stops, float64(k)/float64(len(run)-1), c)
		}
		st = append(st, grade.Layer{
			Name:  "band",
			Field: field.Linear{A: field.Point{U: 0.03}, B: field.Point{U: 0.97}},
			Ramp:  ramp(stops...),
			// a hard band: the edge between two ramps is the only line in the
			// picture, and a soft one would read as a seventh colour
			Alpha: grade.Times(
				field.Func(func(_, v float64) float64 {
					if v < lo || v >= hi {
						return 0
					}
					return 1
				}),
				grade.Flat(1)),
			Gain: 1,
		})
	}
	return st
}

// Bender is the distribution: nine repositories shipped as one thing that
// builds itself from within.
//
// A robot, because of the name, but not a drawn character -- a head assembled
// out of nine plates, each in its own colour, with the seams left showing.
// That is the whole claim of the repository in one picture: nine separate
// things, visibly separate, that are nevertheless one head. A smooth robot
// would have said "robot" and nothing else.
func Bender() *raster.Scene {
	const w, h = 100.0, 45.0
	at := func(x, y float64) geom.Vec { return geom.Vec{X: x, Y: y} }

	// the nine plates, laid out as a head: a crown, a brow, two temples, a
	// face, two cheeks, a jaw and a chin. they tile a rectangle with a gap
	// between each, and the gap is the point.
	// the head, chamfered. the plates are cut out of it rather than laid
	// inside it, so the outline is a head and the seams are still seams. laid
	// out as a grid on their own they read as a colour chart with eyes.
	head := geom.Poly{P: geom.Pts(
		at(23, 3), at(61, 3), at(66, 8), at(66, 30),
		at(61, 41), at(23, 41), at(18, 30), at(18, 8),
	)}
	type plate struct {
		role                   byte
		x0, y0, x1, y1, corner float64
	}
	plates := []plate{
		{'1', 16, 1, 40, 11, 0},    // crown, left
		{'2', 41.5, 1, 68, 11, 0},  // crown, right
		{'3', 16, 12.5, 30, 21, 0}, // temple
		{'4', 31.5, 12.5, 68, 21, 0},
		{'5', 16, 22.5, 44, 31, 0}, // face
		{'6', 45.5, 22.5, 68, 31, 0},
		{'7', 16, 32.5, 34, 43, 0}, // jaw
		{'8', 35.5, 32.5, 50, 43, 0},
		{'9', 51.5, 32.5, 68, 43, 0},
	}
	sc := &raster.Scene{W: w, H: h}
	for _, p := range plates {
		sc.Parts = append(sc.Parts, raster.Part{Role: p.role, Shape: geom.Intersect(head,
			geom.Poly{P: geom.Pts(at(p.x0, p.y0), at(p.x1, p.y0), at(p.x1, p.y1), at(p.x0, p.y1))},
		)})
	}
	// two eyes, cut into the face plates so they belong to the head rather
	// than sitting on it
	sc.Parts = append(sc.Parts,
		raster.Part{Role: 'o', Shape: geom.Ellipse{C: at(29, 26.8), Rx: 3.8, Ry: 3.1}},
		raster.Part{Role: 'i', Shape: geom.Ellipse{C: at(29, 26.8), Rx: 1.8, Ry: 1.5}},
		raster.Part{Role: 'o', Shape: geom.Ellipse{C: at(54, 26.8), Rx: 3.8, Ry: 3.1}},
		raster.Part{Role: 'i', Shape: geom.Ellipse{C: at(54, 26.8), Rx: 1.8, Ry: 1.5}},
		// an aerial, because a robot has one
		raster.Part{Role: '4', Shape: geom.Taper{
			P: []geom.Vec{at(57, 3.5), at(62, 0.8)}, R: []float64{0.7, 0.7}}},
		raster.Part{Role: 'o', Shape: geom.Ellipse{C: at(62.6, 0.9), Rx: 1.8, Ry: 1.5}},
	)
	sc.Pal = map[byte]color.RGBA{
		'1': {58, 52, 86, 255},
		'2': {86, 62, 128, 255},
		'3': {44, 92, 140, 255},
		'4': {58, 138, 176, 255},
		'5': {188, 168, 208, 255},
		'6': {150, 134, 180, 255},
		'7': {96, 70, 120, 255},
		'8': {70, 58, 100, 255},
		'9': {120, 88, 150, 255},
		'o': {246, 200, 96, 255},
		'i': {20, 16, 26, 255},
	}
	sc.MinCover = map[byte]float64{'i': 0.25, 'o': 0.30, '4': 0.30}
	sc.Bg = func(x, y float64) color.RGBA { return color.RGBA{16, 14, 24, 255} }
	return sc
}

// PageRamp is a ramp for the documentation to paint with: dark through
// violet and magenta to gold, which shows a mixer doing its job because a
// magenta-to-orange run is exactly where a bad one goes through mud.
func PageRamp() functions.Ramp {
	return ramp(0.0, "120631", 0.3, "5a2071", 0.55, "c63f6b", 0.8, "ff8f4a", 1.0, "ffd08a")
}

// Hex is a hex colour as a terminal colour, for callers outside this package
// that need one of a card's own colours.
func Hex(s string) color.RGBA {
	c, _ := srgb.FromSwatch(hex(s))
	q := func(v float64) uint8 {
		if v < 0 {
			v = 0
		}
		if v > 1 {
			v = 1
		}
		return uint8(v*255 + 0.5)
	}
	return color.RGBA{q(c.R), q(c.G), q(c.B), 255}
}

// Blend is two hex colours mixed in oklab, which is the library's own line
// between two colours and the reason a gold-to-blue run does not go through
// mud on the way.
func Blend(a, b string, t float64) color.RGBA {
	c, _ := srgb.FromSwatch(ok.Mix.Mix(hex(a), hex(b), t))
	q := func(v float64) uint8 {
		if v < 0 {
			v = 0
		}
		if v > 1 {
			v = 1
		}
		return uint8(v*255 + 0.5)
	}
	return color.RGBA{q(c.R), q(c.G), q(c.B), 255}
}
