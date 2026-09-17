package scenes

import (
	"math"

	"github.com/janearc/libdaffy/field"
	"github.com/janearc/libtheme-css/primitives/functions"
	"github.com/janearc/libtheme-css/spaces/ok"

	"github.com/janearc/miami/grade"
)

// A dusk that does not repeat, because it is not a loop.
//
// The brief was twenty seconds without repetition, and the honest way to get
// that is not a very long cycle -- it is to have nothing cycling. The sun
// sinks, the sky turns from orange to indigo behind it, the horizon glow goes
// out and the stars come up, and none of those come back.
//
// Twenty seconds in, the picture has arrived somewhere it was not at the start.
//
// The one thing that does oscillate is the cloud drift, and it is three
// oscillators at periods of 7.3, 11.7 and 19.1 seconds. Those do not share a
// factor, so the sum does not return to its starting shape for hours; inside
// twenty seconds it never repeats a frame.
//
// Picking periods that are mutually prime-ish is the whole trick and it is
// cheaper than noise.
//
// t is seconds from the start, and every layer reads it directly. libdaffy's
// field package says to fold the time into the field when it is built rather
// than make time a coordinate of the sheet, and that is what happens here.

// stops is a ramp between two sets of colours, mixed stop by stop in oklab. A
// sky that changes colour is not a crossfade between two pictures -- that would
// go through a grey middle where the two disagree. It is one ramp whose stops
// each travel.
func stops(a, b []string, t float64) functions.Ramp {
	n := len(a)
	out := make([]functions.Stop, n)
	for i := range a {
		at := float64(i) / float64(n-1)
		out[i] = functions.Stop{At: at,
			Swatch: ok.Mix.Mix(
				grade.Swatch(a[i]), grade.Swatch(b[i]), t,
			)}
	}
	return functions.New(ok.Mix, out...)
}

// Dusk is the sky at t seconds into a span of dur seconds.
func Dusk(sh grade.Sheet, t, dur float64) grade.Stack {
	const horizon = 0.62
	e := clamp01(t / dur) // how far through the evening

	// the sun sinks past the horizon and takes its light with it
	sunV := horizon - 0.075 + 0.20*ease(e)
	sunC := field.Point{U: 0.42, V: sunV}
	// the disc and the bloom around it are the same circle at two radii
	sunDisc := field.Radial{Centre: sunC, Radius: 0.098}
	sunBloom := field.Radial{Centre: sunC, Radius: 0.42}
	// the haze stays on the horizon while the sun sinks past it
	hazeC := field.Point{U: 0.42, V: horizon}
	// below the horizon it is hidden by the ground layer, so its own alpha
	// only has to fade what is left showing
	sunGain := clamp01(1.25 - 1.5*e)

	// three drifts that do not share a period, so the clouds never arrive
	// back at an arrangement they have already been in
	drift := func(p float64) float64 { return t / p }
	// one turn of a sine, so a sum of three of them still reads as a sum
	wave := func(p float64) float64 { return math.Sin(2 * math.Pi * p) }

	return grade.Stack{
		{
			Name: "sky",
			Field: field.Linear{
				A: field.Point{U: 0.5},
				B: field.Point{U: 0.5, V: horizon},
			},
			Ramp: stops(
				[]string{
					"1b1040", "5a2071", "b8446b", "ff8f4a",
					"ffd08a",
				},
				[]string{
					"03040f", "090a24", "141038", "241a47",
					"3a2452",
				}, ease(e)),
			Gain: 1,
		},
		{
			// stars, come up as the sky goes down. a fixed field of
			// points that does not move: the sky turns dark around
			// them.
			Name:  "stars",
			Field: grade.Flat(0),
			Ramp:  grade.Ramp(0.0, "e8ecff", 1.0, "e8ecff"),
			Alpha: field.Func(func(u, v float64) float64 {
				if v > horizon-0.02 {
					return 0
				}
				return twinkle(u, v, t) * clamp01(e*1.8-0.5)
			}),
			Blend: grade.Add,
			Gain:  1,
		},
		{
			// cloud bars. a sum of three sines across the sky at
			// unrelated periods, thresholded into bands rather than
			// left as a haze, because a terminal cell cannot hold a
			// haze.
			Name: "cloud",
			Field: field.Linear{
				A: field.Point{U: 0.5, V: 0.10},
				B: field.Point{U: 0.5, V: horizon},
			},
			Ramp: stops(
				[]string{"ffb27a", "e2708a", "6c2f63"},
				[]string{
					"2a2340", "1a1730", "0d0b1c",
				}, ease(e)),
			Alpha: field.Func(func(u, v float64) float64 {
				if v > horizon-0.005 {
					return 0
				}
				s := wave(u*1.7-drift(19.1)) +
					0.7*wave(u*3.1+v*2.0-drift(11.7)) +
					0.5*wave(u*5.3-v*1.4-drift(7.3))
				// bands: thin near the horizon, opening out
				// above it
				band := s*0.28 + 0.5 - (horizon-v)*1.1
				return smooth(band, 0.30, 0.52) * 0.85
			}),
			Gain: 1,
		},
		{
			Name:  "haze",
			Field: grade.Flat(0),
			Ramp: stops(
				[]string{"ff7a3c"}, []string{"271a3a"}, ease(e),
			),
			Alpha: grade.Inside(field.Through{
				M: [2][2]float64{{0.5, 0}, {0, 6.0}},
				Field: field.Radial{
					Centre: hazeC, Radius: 0.7}},
				0.02, 0.86),
			Blend: grade.Add,
			Gain:  0.42 * sunGain,
		},
		{
			Name: "sun",
			Field: field.Linear{
				A: field.Point{U: 0.42, V: sunV - 0.075},
				B: field.Point{U: 0.42, V: sunV + 0.075},
			},
			Ramp: grade.Ramp(
				0.0, "fff3bc", 0.5, "ffb03a", 1.0, "ff5a5a",
			),
			Alpha: grade.Times(
				grade.Inside(sunDisc, 0.80, 0.12),
				grade.Flat(clamp01(sunGain*1.4))),
			Gain: 1,
		},
		{
			Name:  "bloom",
			Field: grade.Flat(0),
			Ramp:  grade.Ramp(0.0, "ffcf8a", 1.0, "ffcf8a"),
			Alpha: grade.Inside(sunBloom, 0.0, 0.95),
			Blend: grade.Add,
			Gain:  0.26 * sunGain,
		},
		{
			// last, so the sun sets behind it. painted earlier, the
			// sun went on drawing a neat disc over the dark ground
			// long after it had set, which is the one thing a
			// horizon is for.
			Name: "ground",
			Field: field.Linear{
				A: field.Point{U: 0.5, V: horizon},
				B: field.Point{U: 0.5, V: 1},
			},
			Ramp: stops(
				[]string{"2a1030", "070314"},
				[]string{"0a0616", "020108"}, ease(e),
			),
			Alpha: field.Linear{
				A: field.Point{U: 0.5, V: horizon - 0.004},
				B: field.Point{U: 0.5, V: horizon + 0.004}},
			Gain: 1,
		},
	}
}

// twinkle is a fixed field of stars that vary in brightness but never move. The
// hash is the usual sine-fract trick: it is not good noise, but a star field
// wants points that stay put between frames and nothing more.
func twinkle(u, v, t float64) float64 {
	const grid = 42.0
	cu, cv := math.Floor(u*grid), math.Floor(v*grid)
	h := grade.Hash2(cu, cv)
	if h < 0.90 {
		return 0
	}
	// where in its cell the star sits, so they are not on a lattice
	ou, ov := grade.Hash2(cu+13, cv), grade.Hash2(cu, cv+7)
	du := (u*grid - cu - ou) / 1.0
	dv := (v*grid - cv - ov) / 1.0
	d := math.Hypot(du, dv)
	if d > 0.16 {
		return 0
	}
	bright := 0.55 + 0.45*math.Sin(t*(0.7+2.3*grade.Hash2(cu+3, cv+5))+h*40)
	return (1 - d/0.16) * bright
}

// smooth is nought below a, one above b, and eased between.
func smooth(x, a, b float64) float64 {
	if x <= a {
		return 0
	}
	if x >= b {
		return 1
	}
	t := (x - a) / (b - a)
	return t * t * (3 - 2*t)
}

// clamp01 holds a number to nought and one.
func clamp01(v float64) float64 { return math.Max(0, math.Min(1, v)) }

// ease is slow at both ends, so the evening does not start and stop abruptly.
func ease(t float64) float64 {
	t = clamp01(t)
	return t * t * (3 - 2*t)
}
