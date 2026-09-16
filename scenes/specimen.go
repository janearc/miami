package scenes

import (
	"image/color"

	"miami/canvas"
	"miami/geom"
	"miami/raster"
	"miami/typeset"
)

const (
	fonts = "/System/Library/Fonts/"
	supp  = fonts + "Supplemental/"
)

func mustFace(path, name string) *typeset.Face {
	f, err := typeset.LoadFace(path, name)
	if err != nil {
		panic(err)
	}
	return f
}

// fitCap finds the cap height at which a line sets to a given width.
func fitCap(l typeset.Line, width float64) float64 {
	l.CapHeight = 10
	return 10 * width / l.Width()
}

// Specimen sets the candidate faces identically, so the choice is made from
// a render at the real width rather than from a description.
func Specimen(cv canvas.Canvas) *raster.Scene {
	faces := []*typeset.Face{
		mustFace(supp+"Copperplate.ttc", "Copperplate Light"),
		mustFace(supp+"Futura.ttc", "Futura Medium"),
		mustFace(fonts+"Optima.ttc", "Optima Regular"),
	}
	u := cv.W()
	band := 44.0
	var letters []geom.Shape
	var panels []geom.Shape
	for i, f := range faces {
		top := float64(i) * band
		panels = append(panels, geom.Poly{P: geom.Pts(
			geom.Vec{X: 3, Y: top + 3}, geom.Vec{X: u - 3, Y: top + 3},
			geom.Vec{X: u - 3, Y: top + band - 3}, geom.Vec{X: 3, Y: top + band - 3})})

		small := typeset.Line{Text: "LET THERE BE", Face: f, Tracking: 0.12,
			Wobble: typeset.Wobble{Tilt: 3, Shift: 0.05, Scale: 0.03, Space: 0.04, Seed: 7}}
		small.CapHeight = fitCap(small, 0.62*u)
		big := typeset.Line{Text: "LIGHTS", Face: f, Tracking: 0.10,
			Wobble: typeset.Wobble{Tilt: 3, Shift: 0.04, Scale: 0.03, Space: 0.04, Seed: 11}}
		big.CapHeight = fitCap(big, 0.80*u)
		letters = append(letters, small.Set(u/2, top+8+small.CapHeight)...)
		letters = append(letters, big.Set(u/2, top+band-7)...)
	}
	sc := &raster.Scene{W: u, H: band * float64(len(faces))}
	sc.Parts = []raster.Part{
		{Role: 'n', Shape: geom.Union(panels...)},
		{Role: 'L', Shape: geom.Union(letters...)}}
	sc.Pal = map[byte]color.RGBA{
		'n': {238, 226, 196, 255},
		'L': {96, 16, 20, 255},
	}
	sc.MinCover = map[byte]float64{'L': 0.30}
	sc.Bg = func(x, y float64) color.RGBA { return color.RGBA{28, 30, 64, 255} }
	return sc
}
