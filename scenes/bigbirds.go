package scenes

import (
	_ "embed"
	"image/color"
	"math"
	"miami/canvas"
	"miami/geom"
	"miami/motifs"
	"miami/paint"
	"miami/raster"
	"miami/typeset"
)

// one dark silhouette family in front of one graded sky. no contours, no third
// value: a two-mass picture is the only kind that still reads after being
// halved twice.

var bbSky = []paint.Stop{
	// the horizon must stay DARKER than the sun. ramping the sky to gold
	// made the sky brighter than the disc and the sun lost its edge
	// entirely.
	{At: 0.00, C: color.RGBA{24, 8, 60, 255}},
	{At: 0.30, C: color.RGBA{86, 18, 104, 255}},
	{At: 0.55, C: color.RGBA{170, 34, 108, 255}},
	{At: 0.78, C: color.RGBA{216, 56, 96, 255}},
	{At: 0.92, C: color.RGBA{238, 92, 84, 255}},
	{At: 1.00, C: color.RGBA{246, 124, 80, 255}}}

var bbGround = []paint.Stop{
	{At: 0.00, C: color.RGBA{96, 50, 90, 255}},
	{At: 0.22, C: color.RGBA{66, 32, 70, 255}},
	{At: 1.00, C: color.RGBA{22, 10, 32, 255}}}

// tunables swept during acceptance, kept as knobs because the right values
// differ between a 144 column frame and an 80 column one.
var (
	BBWordFrac  = 0.620
	BBErodeUp   = 8
	BBErodeIter = 1
	bbSunFrac   = 0.420
)

// BigBirds is the negative-space piece: birds cut out of a sunset, with the
// wordmark left as empty space rather than drawn.
func BigBirds(cols, rows int) *raster.Scene {
	u := float64(cols)
	v := float64(rows) * canvas.CellAspect
	hy := 0.850 * v
	// scale the round and tall things off the SMALLER of the two
	// dimensions. keying them to height alone made the sun eat 84 percent
	// of a square frame and pushed both palm crowns off the edge.
	s := math.Min(v, 0.70*u)

	sunC := geom.Vec{X: 0.465 * u, Y: 0.800 * v}
	sunR := bbSunFrac * s

	sc := &raster.Scene{W: u, H: v}

	// skyline: sparse, low, right of centre only. a full width one fills
	// the negative space with small holes. two palms, not three. a middle
	// one tangles with the left and fills the open space the composition is
	// built around. both palms lean OUT of frame.
	//
	// leaning the left one inward ran its trunk straight through the
	// wordmark, and two dark shapes of the same value that overlap stop
	// being two shapes.
	palms := geom.Union(append(
		motifs.PalmAt(geom.Vec{X: 0.135 * u, Y: hy + 0.006*v}, 0.680*s, -0.046*u, false, -4),
		motifs.PalmAt(geom.Vec{X: 0.885 * u, Y: hy + 0.004*v}, 0.510*s, -0.034*u, true, 3)...)...)

	raw, _, _ := typeset.ParseInk(bbFigs)
	ink := typeset.Erode(raw, BBErodeUp, BBErodeIter)
	tc, tr := len(ink[0]), len(ink)
	// low on the disc, so the sun's top arc stays clear above it
	word := typeset.FitText(ink, tc, tr, 0.465*u, hy-0.022*v, BBWordFrac*u)

	// three, and large enough to be marks rather than dust. pushed into the
	// open right, which is the negative space the asymmetry buys; three
	// marks, sized to read rather than to speckle. five, spread up the open
	// right so a square frame's tall sky is not a hole.
	//
	// they are the only thing living in the negative space.
	birds := geom.Union(
		motifs.BirdAt(geom.Vec{X: 0.560 * u, Y: 0.148 * v}, 0.110*u, -7),
		motifs.BirdAt(geom.Vec{X: 0.668 * u, Y: 0.258 * v}, 0.078*u, 5),
		motifs.BirdAt(geom.Vec{X: 0.726 * u, Y: 0.100 * v}, 0.060*u, -4),
		motifs.BirdAt(geom.Vec{X: 0.430 * u, Y: 0.062 * v}, 0.070*u, 6),
		motifs.BirdAt(geom.Vec{X: 0.615 * u, Y: 0.058 * v}, 0.048*u, -5),
	)

	sc.Parts = []raster.Part{
		{Role: 'p', Shape: palms},
		{Role: 'g', Shape: word},
		{Role: 'v', Shape: birds}}
	sc.Pal = map[byte]color.RGBA{
		'p': {32, 14, 42, 255},
		'g': {20, 7, 30, 255},
		'v': {26, 11, 36, 255},
	}
	sc.Bg = func(fx, fy float64) color.RGBA {
		px, py := fx*u, fy*v
		if py >= hy {
			t := (py - hy) / (v - hy)
			g := paint.Ramp(bbGround, t)
			d := math.Abs(px-sunC.X) / (0.30 * u)
			if d < 1 {
				g = paint.Lerp(g, color.RGBA{250, 168, 122, 255},
					(1-d)*(1-t)*(1-t)*0.55)
			}
			if w := 0.055 * v; py < hy+w {
				return paint.Lerp(color.RGBA{58, 166, 162, 255}, g, (py-hy)/w)
			}
			return g
		}
		c := paint.Ramp(bbSky, py/hy)
		dx, dy := px-sunC.X, py-sunC.Y
		r := math.Sqrt(dx*dx + dy*dy)
		core := color.RGBA{255, 250, 228, 255}
		rim := color.RGBA{253, 172, 96, 255}
		switch {
		case r < sunR:
			c = paint.Lerp(core, rim, math.Pow(r/sunR, 2.1))
		case r < sunR+0.028*v:
			c = paint.Lerp(rim, c, (r-sunR)/(0.028*v))
		}
		return c
	}
	return sc
}

// the big birds banner, copied in from frontier-art so a rename over there
// cannot break a render here. it moved once already, on 2026-09-13.
//
//go:embed assets/bb.txt
var bbFigs []byte
