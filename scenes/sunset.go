package scenes

import (
	"math"

	"github.com/janearc/libdaffy/field"
	"github.com/janearc/libtheme-css/primitives/functions"
	"github.com/janearc/libtheme-css/primitives/swatch"
	"github.com/janearc/libtheme-css/spaces/ok"
	"github.com/janearc/libtheme-css/spaces/srgb"

	"miami/grade"
)

// The sunset is six gradients over each other, and it is here because it is
// the picture that cannot be drawn any other way.
//
// Every one of them is a field and a ramp, in libdaffy's sense: a rule from
// position to t, and a rule from t to a colour, kept apart. What stacking
// adds is the third rule -- how much of this layer is here -- and that is
// also a field, which is where the whole thing pays off. The sun is a
// vertical ramp that has never heard of circles, made round by an alpha and
// cut into slats by another alpha multiplied into the first. Change the ramp
// and the slats stay. Change the slats and the colour stays.
//
// The horizon is the one number that several layers share, so it is a
// parameter rather than a constant repeated six times.

func hex(s string) swatch.Swatch {
	var r, g, b uint8
	for i, c := range []byte(s) {
		v := byte(0)
		switch {
		case c >= '0' && c <= '9':
			v = c - '0'
		case c >= 'a' && c <= 'f':
			v = c - 'a' + 10
		}
		switch i / 2 {
		case 0:
			r = r<<4 | v
		case 1:
			g = g<<4 | v
		case 2:
			b = b<<4 | v
		}
	}
	return srgb.RGB{R: float64(r) / 255, G: float64(g) / 255, B: float64(b) / 255}.Swatch()
}

// ramp is stops at given positions, mixed in oklab, which is the mixer that
// keeps a magenta-to-orange run from going through mud on the way.
func ramp(stops ...any) functions.Ramp {
	var s []functions.Stop
	for i := 0; i < len(stops); i += 2 {
		s = append(s, functions.Stop{At: stops[i].(float64), Swatch: hex(stops[i+1].(string))})
	}
	return functions.New(ok.Mix, s...)
}

// Sunset is the stack at time t, in turns. Time is folded into the fields
// when they are built, exactly as libdaffy's field package says to do it:
// there is no clock inside a field and no t coordinate on the sheet.
func Sunset(sh grade.Sheet, horizon, phase float64) grade.Stack {
	sunC := field.Point{U: 0.5, V: horizon - 0.055}

	// the sun's colour runs top to bottom and knows nothing about being round
	sunRamp := ramp(0.0, "fff1a8", 0.34, "ffb03a", 0.62, "ff4d7e", 1.0, "8e2a86")
	sunBody := field.Linear{
		A: field.Point{U: 0.5, V: sunC.V - 0.145},
		B: field.Point{U: 0.5, V: sunC.V + 0.145},
	}
	// the radius is set wider than the sun and the alpha cuts it back inside
	// that. fields clamp t at 1, so "past the radius" and "exactly at the
	// radius" are the same number; an alpha that only reaches 0 at t = 1 is
	// therefore still a quarter open across the entire rest of the sheet,
	// which is how the first attempt got horizontal bands across the sky.
	sunDisc := field.Radial{Centre: sunC, Radius: 0.165}

	// the slats ride upward on the phase, and widen as they fall, which is
	// the only reason flat horizontal bands read as a curved surface
	slats := grade.Stripes(0.020, 0.62, phase*0.020, 2.6)

	return grade.Stack{
		{
			Name:  "sky",
			Field: field.Linear{A: field.Point{U: 0.5}, B: field.Point{U: 0.5, V: horizon}},
			Ramp: ramp(0.0, "120631", 0.30, "3b0f63", 0.58, "7a1b6d",
				0.82, "c63f6b", 1.0, "ff8a4c"),
			Gain: 1,
		},
		{
			Name:  "ground",
			Field: field.Linear{A: field.Point{U: 0.5, V: horizon}, B: field.Point{U: 0.5, V: 1}},
			Ramp:  ramp(0.0, "2a0b44", 1.0, "07021a"),
			// the horizon line itself, as an alpha: nought above it and one
			// below, over the space of a couple of subpixels
			Alpha: field.Linear{
				A: field.Point{U: 0.5, V: horizon - 0.004},
				B: field.Point{U: 0.5, V: horizon + 0.004}},
			Gain: 1,
		},
		{
			// the haze on the horizon is light, so it adds. this is the layer
			// that would look like grey paint in any non-linear space.
			Name:  "haze",
			Field: grade.Flat(0),
			Ramp:  ramp(0.0, "ff6a3c", 1.0, "ff6a3c"),
			// squashed flat: wide across the horizon and thin through it. the
			// matrix magnifies v before the radial reads it, so a circle
			// becomes a long low ellipse -- which is what Through is for.
			Alpha: grade.Inside(field.Through{
				M:     [2][2]float64{{0.55, 0}, {0, 5.5}},
				Field: field.Radial{Centre: field.Point{U: 0.5, V: horizon}, Radius: 0.62}},
				0.02, 0.86),
			Blend: grade.Add,
			Gain:  0.30,
		},
		{
			Name:  "sun",
			Field: sunBody,
			Ramp:  sunRamp,
			// round, and cut into slats: two alphas multiplied. neither knows
			// what the other is for and neither touches the colour.
			Alpha: grade.Times(grade.Inside(sunDisc, 0.85, 0.08), slats),
			Gain:  1,
		},
		{
			Name:  "grid",
			Field: field.Linear{A: field.Point{U: 0.5, V: horizon}, B: field.Point{U: 0.5, V: 1}},
			Ramp:  ramp(0.0, "2de2e6", 0.55, "ff2a9d", 1.0, "ff2a9d"),
			Alpha: perspectiveGrid(sh, horizon, phase),
			Blend: grade.Add,
			Gain:  0.60,
		},
		{
			// the sun's own light on the sky around it, added last so it sits
			// over the grid as well as the sky
			Name:  "bloom",
			Field: grade.Flat(0),
			Ramp:  ramp(0.0, "ffd08a", 1.0, "ffd08a"),
			Alpha: grade.Inside(field.Radial{Centre: sunC, Radius: 0.44}, 0.0, 0.95),
			Blend: grade.Add,
			Gain:  0.16,
		},
	}
}

// perspectiveGrid is the alpha of a floor of lines running to a vanishing
// point on the horizon. It is a field like any other -- position to t -- and
// it is written inline because the field package says in as many words that a
// shape none of the three shapes describe is still a field.
//
// The whole difficulty is line width. Specify it in world units and the near
// lines are fat bars, the far ones fall between subpixels and flicker, and
// the one line straight down the middle is infinitely wide because its world
// coordinate is nought everywhere along it. So the width is specified on the
// screen, and the world tolerance is derived by how fast the line's own
// coordinate changes per subpixel. That one change is the difference between
// a floor and a smear.
func perspectiveGrid(sh grade.Sheet, horizon, phase float64) field.Field {
	const (
		eye    = 0.055 // how far the camera sits above the floor
		across = 0.014 // spacing of the lines running away from the viewer
		along  = 0.055 // spacing of the lines running across
		width  = 2.6   // line width, in subpixels of the sheet
	)
	du, dv := sh.Du(), sh.Dv()
	return field.Func(func(u, v float64) float64 {
		d := v - horizon
		if d <= 0 {
			return 0
		}
		z := eye / d // depth: the reciprocal is the perspective

		// lines running away: constant world x. x = (u - half) * z, so x moves
		// by z per unit of u, and a line half a pixel wide on screen is half a
		// pixel times that.
		x := (u - 0.5) * z
		latGap := across / (z * du) // how many subpixels apart these lines are
		lat := near(x/across, 0.5*width*du*(z/across)) * resolves(latGap)

		// lines running across: constant depth, marching toward the viewer.
		// z changes by z/d per unit of v.
		lonGap := along * d / (z * dv)
		lon := near((z-phase*0.6)/along, 0.5*width*dv*(z/d)/along) * resolves(lonGap)

		return math.Max(lat, lon)
	})
}

// near is 1 on an integer value and falls to 0 within tol of it.
func near(t, tol float64) float64 {
	q := math.Mod(t, 1)
	if q < 0 {
		q++
	}
	d := math.Min(q, 1-q)
	if tol <= 0 || d >= tol {
		return 0
	}
	return 1 - d/tol
}

// resolves fades a family of lines out as they crowd below the resolution of
// the screen. gap is how many subpixels apart they are; below about two there
// is no gap left to see and drawing them anyway turns the whole area into a
// flat wash of the line colour -- which on this scene was a solid teal band
// laid across the horizon. This is the cheap analytic version of what a
// mipmap does, and the scene needs nothing more.
func resolves(gap float64) float64 {
	const floor, full = 1.0, 3.0
	if gap <= floor {
		return 0
	}
	if gap >= full {
		return 1
	}
	t := (gap - floor) / (full - floor)
	return t * t
}

// Effects are more layers, and that is the entire mechanism.
//
// There is no effects system here because there does not need to be one. An
// effect is a gradient whose alpha has had the time folded into it, which is
// the same sentence as the one describing the sun's slats -- those move too.
// So these append to the stack and the compositor cannot tell them apart from
// the scene, which is the property worth having: an effect can go under a
// layer as easily as over it, just by where it is inserted.
func Effects(sh grade.Sheet, horizon, phase float64) grade.Stack {
	return grade.Stack{
		{
			// scanlines. a raster gap is not a grey wash over the picture, it
			// is the absence of light, so this multiplies rather than paints.
			// the period is in sheet units and works out at one dark line per
			// sextant row, which is the finest a terminal can carry.
			Name:  "scan",
			Field: grade.Flat(0),
			Ramp:  ramp(0.0, "000000", 1.0, "000000"),
			Alpha: grade.Stripes(2*sh.Dv(), 0.5, phase*0.5*sh.Dv(), 0),
			Blend: grade.Mul,
			Gain:  0.30,
		},
		{
			// a glare crossing the tube, the reflection of a window behind the
			// viewer. it is a Band over a Linear, which is to say the same two
			// objects as everything else on the stack.
			Name:  "glare",
			Field: grade.Flat(0),
			Ramp:  ramp(0.0, "b9e6ff", 1.0, "b9e6ff"),
			Alpha: grade.Band(
				field.Linear{A: field.Point{U: 0, V: 0.25}, B: field.Point{U: 1, V: 0.75}},
				mod1(phase*0.5+0.15), 0.06, 0.20),
			Blend: grade.Add,
			Gain:  0.10,
		},
	}
}

// mod1 wraps into 0..1, so a phase that runs on for ever still lands on the
// sheet.
func mod1(t float64) float64 {
	t = math.Mod(t, 1)
	if t < 0 {
		t++
	}
	return t
}
