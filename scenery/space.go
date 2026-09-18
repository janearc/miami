package scenery

import (
	"math"

	"github.com/janearc/miami/light"
	"github.com/janearc/miami/noise"
	"github.com/janearc/miami/view"
)

// Space is the sky behind everything: stars, gas, dust and the band of a
// galaxy seen edge on.
//
// It is a function of direction and not of where on the screen, which is the
// one decision everything else here leans on.
//
// A black hole bends the ray for a subpixel and asks the sky what is in the
// direction the ray ended up going, so the stars behind a hole are smeared
// round it for nothing. A camera that turns sees the same stars move the way
// they should, and a sky that knew about the screen could do neither.
type Space struct {
	Seed uint64

	// Stars is how many, where one is a dark site on a clear night.
	Stars float64
	// Twinkle is how much the bright ones flicker, nought to one.
	Twinkle float64

	// Nebula is how bright the gas is, and Gas its colours across the
	// gas's own variation. Nought is no gas at all.
	Nebula float64
	Gas    light.Ramp
	// Dust is how much the dark lanes hide of what is behind them.
	Dust float64

	// Ambient is light from every direction alike: a sky rather than the
	// dark between stars.
	//
	// It is what a black hole against daylight needs. Being the same
	// whichever way a ray leaves, it is not bent into anything, so the
	// shadow stays a black disc cut out of it while the stars and the gas
	// around it are still smeared by the lens.
	Ambient light.RGB

	// Band is the axis of a galaxy's plane, so the band of it runs round
	// the sky square to this. A zero axis is no band. Width is in radians
	// either side of the plane, and Glow is the band's own light.
	Band  view.Vec
	Width float64
	Glow  float64
}

// Sky is what is in a direction, for a sky pixel pixel radians across, apart
// from the twinkle: the light that does not change, and how much of it is a
// star that should flicker, and at what phase.
type Sky struct {
	Light   light.RGB
	Star    float64 // luminance of it that is star, for the twinkle
	Phase   float64
	Twinkle float64
}

// At is the sky's light at t, flicker included.
func (k Sky) At(t float64) light.RGB {
	if k.Star <= 0 || k.Twinkle <= 0 {
		return k.Light
	}
	f := k.Twinkle * 0.5 * math.Sin(t*(2.3+k.Phase)+k.Phase*40)
	return k.Light.Plus(k.Light.Scale(f * k.Star / math.Max(k.Light.Y(),
		1e-9)))
}

// Look is the sky in direction dir. pixel is how many radians one subpixel
// spans, so a star can be kept at least a subpixel across however far the
// camera zooms: a star smaller than that flickers in and out as it moves.
func (s Space) Look(dir view.Vec, pixel float64) Sky {
	d := unit(dir)
	band := s.band(d)
	stars, star, phase := s.stars(d, pixel, band)
	var gas light.RGB
	dust := 0.0
	if s.Nebula > 0 || s.Glow > 0 || s.Dust > 0 {
		gas, dust = s.gas(d, band)
	}
	keep := 1 - dust
	lit := s.Ambient.Plus(gas).Plus(stars.Scale(keep))
	return Sky{Light: lit, Star: star * keep,
		Phase: phase, Twinkle: s.Twinkle}
}

// band is how deep in the galaxy's plane a direction is: one on it, falling
// off either side, nought everywhere with no band.
func (s Space) band(d view.Vec) float64 {
	if length(s.Band) == 0 {
		return 0
	}
	w := s.Width
	if w <= 0 {
		w = 0.25
	}
	off := math.Asin(math.Max(-1, math.Min(1, dot(d, unit(s.Band)))))
	return math.Exp(-off * off / (2 * w * w))
}

// gas is the nebula's light in a direction, and how much dust there hides
// what is behind.
//
// Warped noise: the position the noise is read at is itself moved by
// another noise first. Unwarped fractal noise reads as cloud seen from below;
// warping it is what makes the filaments and folds of gas lit from inside.
func (s Space) gas(d view.Vec, band float64) (light.RGB, float64) {
	seed := s.Seed*0x9e3779b97f4a7c15 + 17
	wx := noise.Fractal(seed+1, d.X*1.7, d.Y*1.7, d.Z*1.7, 2)
	wy := noise.Fractal(seed+2, d.X*1.7+5, d.Y*1.7, d.Z*1.7, 2)
	wz := noise.Fractal(seed+3, d.X*1.7, d.Y*1.7+9, d.Z*1.7, 2)
	x, y, z := d.X*2.3+0.9*wx, d.Y*2.3+0.9*wy, d.Z*2.3+0.9*wz

	var out light.RGB
	dust := 0.0
	if s.Nebula > 0 {
		n := noise.Fractal(seed+4, x, y, z, 5)
		dense := smooth(-0.05+0.25*(1-band), 0.55, n+0.3*band)
		dense = math.Pow(dense, 1.6)
		tone := 0.5 + 0.9*noise.Fractal(seed+5, x*1.4, y*1.4, z*1.4, 3)
		out = s.Gas.At(tone).Scale(s.Nebula * dense)
	}
	if s.Glow > 0 && band > 0 {
		n := noise.Fractal(seed+6, x*3, y*3, z*3, 3)
		out = out.Plus(light.Kelvin(5200).Scale(s.Glow * band *
			(0.55 + 0.45*n)))
	}
	if s.Dust > 0 {
		lane := noise.Fractal(seed+7, x*2.2, y*2.2, z*2.2, 4)
		dust = s.Dust * smooth(0.05, 0.45, lane) * (0.35 + 0.65*band)
		out = out.Scale(1 - dust)
	}
	return out, dust
}

// starLayer is one grid of stars: how fine, how likely a cell is to hold
// one, and how bright the brightest is.
type starLayer struct {
	grid   int
	chance float64
	flux   float64
}

// layers are the faint crowd and the bright few. Two grids and not one,
// because a single grid fine enough for the crowd puts the bright stars on a
// lattice anyone can see.
var layers = []starLayer{
	{grid: 380, chance: 0.10, flux: 0.35},
	{grid: 70, chance: 0.12, flux: 3.5},
}

// stars is the starlight in a direction, how much of it is the brightest
// star there, and that star's twinkle phase.
//
// The sky is cut as a cube: the direction's largest axis picks a face, and the
// other two, divided by it, say where on the face. Each face is a grid, and a
// cell holds a star or not by a hash of where it is. The eight cells around
// are asked too, so a star near a cell's edge is not cut in half.
func (s Space) stars(d view.Vec, pixel, band float64) (light.RGB, float64,
	float64) {
	if s.Stars <= 0 {
		return light.RGB{}, 0, 0
	}
	face, a, b := cube(d)
	sigma := math.Max(pixel*0.55, 1e-5)
	var sum light.RGB
	best, phase := 0.0, 0.0
	for li, l := range layers {
		seed := s.Seed*0x9e3779b97f4a7c15 + uint64(li)*1000 + 99
		n := float64(l.grid)
		ci, cj := int64((a+1)/2*n), int64((b+1)/2*n)
		chance := l.chance * s.Stars * (0.45 + 1.4*band)
		for dj := int64(-1); dj <= 1; dj++ {
			for di := int64(-1); di <= 1; di++ {
				c, glow, ph := l.star(seed, face, ci+di, cj+dj,
					chance, d, pixel, sigma)
				sum = sum.Plus(c)
				if glow > best {
					best, phase = glow, ph
				}
			}
		}
	}
	return sum, best, phase
}

// star is the light a direction gets from the star in one cell of a face, if
// the cell has one: its colour times its glow, the glow alone, and the star's
// twinkle phase.
//
// Its glow is a gaussian sigma radians wide, scaled so the star's total light
// is the same however wide a subpixel is. A star that shrank as the camera
// zoomed out would be dimmer for no reason anyone could see.
func (l starLayer) star(seed uint64, face int, i, j int64, chance float64,
	d view.Vec, pixel, sigma float64) (light.RGB, float64, float64) {
	cell := int64(face)<<40 ^ i<<20 ^ j
	if noise.Hash(seed, cell) >= chance {
		return light.RGB{}, 0, 0
	}
	n := float64(l.grid)
	sa := (float64(i)+noise.Hash(seed+1, cell))/n*2 - 1
	sb := (float64(j)+noise.Hash(seed+2, cell))/n*2 - 1
	gap := length(sub(d, uncube(face, sa, sb)))
	if gap > 4*sigma {
		return light.RGB{}, 0, 0
	}
	flux := l.flux * math.Pow(noise.Hash(seed+3, cell), 6)
	norm := pixel * pixel / (2 * math.Pi * sigma * sigma)
	glow := flux * norm * math.Exp(-gap*gap/(2*sigma*sigma))
	heat := 2800 + 11000*math.Pow(noise.Hash(seed+4, cell), 2)
	tint := light.Kelvin(heat).Mix(light.RGB{R: 1, G: 1, B: 1}, 0.45)
	return tint.Scale(glow), glow, noise.Hash(seed+5, cell) * 6.28
}

// cube is which face of a cube a direction goes through, and where on it,
// each coordinate from minus one to one.
func cube(d view.Vec) (face int, a, b float64) {
	ax, ay, az := math.Abs(d.X), math.Abs(d.Y), math.Abs(d.Z)
	switch {
	case ax >= ay && ax >= az:
		if d.X > 0 {
			return 0, d.Y / ax, d.Z / ax
		}
		return 1, d.Y / ax, d.Z / ax
	case ay >= az:
		if d.Y > 0 {
			return 2, d.X / ay, d.Z / ay
		}
		return 3, d.X / ay, d.Z / ay
	}
	if d.Z > 0 {
		return 4, d.X / az, d.Y / az
	}
	return 5, d.X / az, d.Y / az
}

// uncube is cube backwards: the unit direction through a point on a face.
func uncube(face int, a, b float64) view.Vec {
	var v view.Vec
	switch face {
	case 0:
		v = view.Vec{X: 1, Y: a, Z: b}
	case 1:
		v = view.Vec{X: -1, Y: a, Z: b}
	case 2:
		v = view.Vec{X: a, Y: 1, Z: b}
	case 3:
		v = view.Vec{X: a, Y: -1, Z: b}
	case 4:
		v = view.Vec{X: a, Y: b, Z: 1}
	default:
		v = view.Vec{X: a, Y: b, Z: -1}
	}
	return unit(v)
}

// smooth is nought below a, one above b, and eased between.
func smooth(a, b, x float64) float64 {
	if x <= a {
		return 0
	}
	if x >= b {
		return 1
	}
	t := (x - a) / (b - a)
	return t * t * (3 - 2*t)
}
