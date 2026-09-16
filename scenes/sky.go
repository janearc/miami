package scenes

import (
	"math"

	"github.com/janearc/libdaffy/field"
	"github.com/janearc/libtheme-css/primitives/functions"
	"github.com/janearc/libtheme-css/spaces/ok"

	"miami/grade"
	"miami/view"
)

// The sky at a time of day.
//
// Three skies, mixed rather than chosen between. Night and day are the ends and
// the sun's height slides between them; the gold of sunrise and sunset is mixed
// in on top of that, strongest where the sun touches the horizon.
//
// Mixed and not switched because a sky that changes state pops. Every stop
// travels on its own line in oklab, which is the same rule the dusk scene uses
// and the reason a magenta-to-orange run does not go through mud.
var (
	skyNight = []string{"03040f", "090a24", "141038", "241a47", "3a2452"}
	skyDay   = []string{"1d4fa8", "3f7ad6", "6fa6e8", "a8cbf2", "d6e7fb"}
	skyGold  = []string{"2a1040", "7a2a6a", "d1496a", "ff8f4a", "ffd08a"}
)

// SkyAt is the sky, and the sun in it, for a sun at a given height.
func SkyAt(sh grade.Sheet, sun view.Sun, horizon float64) grade.Stack {
	up, gold := sun.Up(), sun.Golden()

	stops := make([]functions.Stop, len(skyNight))
	for i := range skyNight {
		at := float64(i) / float64(len(skyNight)-1)
		c := ok.Mix.Mix(hex(skyNight[i]), hex(skyDay[i]), up)
		c = ok.Mix.Mix(c, hex(skyGold[i]), gold)
		stops[i] = functions.Stop{At: at, Swatch: c}
	}

	// the sun's place on the sheet. the height is its elevation against the
	// horizon; the side is its azimuth, east at dawn and west at dusk. it
	// is only ever seen near the horizon, because the camera looks level
	// and a noon sun is above the top of the frame.
	sunV := horizon - sun.Elevation*0.9
	sunU := 0.5 + 0.42*(sun.At.X/60)

	st := grade.Stack{{
		Name:  "sky",
		Field: field.Linear{A: field.Point{U: 0.5}, B: field.Point{U: 0.5, V: horizon}},
		Ramp:  functions.New(ok.Mix, stops...),
		Gain:  1,
	}, {
		// stars, out only when the sky is dark enough to hold them
		Name:  "stars",
		Field: grade.Flat(0),
		Ramp:  ramp(0.0, "e8ecff", 1.0, "e8ecff"),
		Alpha: field.Func(func(u, v float64) float64 {
			if v > horizon-0.02 {
				return 0
			}
			return starAt(u, v)
		}),
		Blend: grade.Add,
		Gain:  math.Max(0, 1-up*6),
	}}

	// the disc, while it is anywhere near the horizon. above that it is out
	// of frame and drawing it would put a sun in the middle of a blue sky
	// at the wrong height.
	if sun.Elevation > -0.25 && sun.Elevation < 0.55 {
		st = append(st, grade.Layer{
			Name: "sun",
			Field: field.Linear{
				A: field.Point{U: sunU, V: sunV - 0.07},
				B: field.Point{U: sunU, V: sunV + 0.07}},
			Ramp: ramp(0.0, "fff6cc", 0.5, "ffc24a", 1.0, "ff6a5a"),
			Alpha: grade.Inside(field.Radial{
				Centre: field.Point{U: sunU, V: sunV}, Radius: 0.095}, 0.78, 0.14),
			Gain: 1,
		}, grade.Layer{
			Name:  "bloom",
			Field: grade.Flat(0),
			Ramp:  ramp(0.0, "ffcf8a", 1.0, "ffcf8a"),
			Alpha: grade.Inside(field.Radial{
				Centre: field.Point{U: sunU, V: sunV}, Radius: 0.5}, 0.0, 0.95),
			Blend: grade.Add,
			Gain:  0.22,
		})
	}
	return st
}

// starAt is one fixed field of points, shared by every sky, so the stars do not
// rearrange themselves between one frame and the next.
func starAt(u, v float64) float64 {
	const grid = 42.0
	cu, cv := math.Floor(u*grid), math.Floor(v*grid)
	h := hash2(cu, cv)
	if h < 0.9 {
		return 0
	}
	ou, ov := hash2(cu+13, cv), hash2(cu, cv+7)
	d := math.Hypot(u*grid-cu-ou, v*grid-cv-ov)
	if d > 0.16 {
		return 0
	}
	return 1 - d/0.16
}
