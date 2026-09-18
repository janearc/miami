package scenery

import (
	"math"
	"sync"

	"github.com/janearc/miami/internal/split"
	"github.com/janearc/miami/light"
	"github.com/janearc/miami/noise"
	"github.com/janearc/miami/view"
)

// Planet is a world of hexes, turning, lit from one side.
//
// Every subpixel the planet covers is a ray that hit a sphere. Where it hit is
// turned back by however far the planet has spun, so the question "which tile
// is here" is asked in the planet's own frame, where the tiles do not move.
//
// The answer is a biome's colour, raised or shaded by the tile's relief, with
// a dark line where the point is near the tile's edge.
//
// The lines are what make it read as hexes, and they are only drawn while a
// tile is several subpixels across. Smaller than that, a line is a grey
// speckle over the whole disc, so they fade out as the planet gets small and
// the world is left to be read by its colours.
type Planet struct {
	At     view.Vec
	Radius float64

	// Tiles is how finely the icosahedron's faces are split: 10n^2+2
	// tiles. Twelve is big hexes; forty is a world seen from orbit.
	Tiles int
	Seed  uint64
	// Ocean is the share of tiles under the sea, and Plates how many plates
	// the crust is cut into.
	Ocean  float64
	Plates int
	// Cold is added to every tile's cold: above nought is an ice age, and
	// well above it is a frozen world.
	Cold float64

	// Axis is the planet's north, and Day the seconds it takes to turn once
	// about it; negative turns it the other way.
	Axis view.Vec
	Day  float64

	// Sun is the direction the light comes from. Its length is ignored.
	Sun view.Vec
	// Colours is each biome's colour, and Glow each biome's own light:
	// lava, for a molten world.
	Colours [biomes]light.RGB
	Glow    [biomes]float64

	// Lines is how dark the edges between hexes are, nought to one, and
	// Relief how strongly a tile is shaded by the tiles around it.
	Lines, Relief float64
	// Clouds is how much of the sky is cloud, nought to one.
	Clouds float64
	// Air is the atmosphere's colour and how thick it looks at the rim.
	Air      light.RGB
	AirDepth float64
	// Cities is the share of land tiles lit on the night side.
	Cities float64

	mu    sync.Mutex
	world *world
	made  terrain
}

// Draw paints the planet over the frame where it is nearer than what is
// there, with its atmosphere glowing past its edge.
func (p *Planet) Draw(b *light.Buffer, eye view.Eye, t float64) {
	w := p.worldFor()
	sun := unit(p.Sun)
	if length(p.Sun) == 0 {
		sun = unit(view.Vec{X: -1, Y: 0.5, Z: -0.6})
	}
	spin := p.spin(t)
	lens := eye.Lens
	if lens == 0 {
		lens = 1
	}
	pixel := 1 / (lens * float64(b.W)) // radians a subpixel spans
	split.Rows(b.H, func(lo, hi int) {
		for y := lo; y < hi; y++ {
			for x := 0; x < b.W; x++ {
				p.subpixel(b, eye, w, x, y, sun, spin, pixel, t)
			}
		}
	})
}

// turned is the planet's own orientation at a moment: north, and two axes
// square to it that turn with the day.
type turned struct{ north, east, south view.Vec }

// spin is how the planet is turned at t.
func (p *Planet) spin(t float64) turned {
	north := unit(p.Axis)
	if length(p.Axis) == 0 {
		north = view.Vec{Y: 1}
	}
	u, v := basis(north)
	ang := 0.0
	if p.Day != 0 {
		ang = 2 * math.Pi * t / p.Day
	}
	c, s := math.Cos(ang), math.Sin(ang)
	return turned{north: north, east: add(scale(u, c), scale(v, s)),
		south: add(scale(v, c), scale(u, -s))}
}

// body turns a world direction into the planet's own frame.
func (f turned) body(d view.Vec) view.Vec {
	return view.Vec{X: dot(d, f.east), Y: dot(d, f.north),
		Z: dot(d, f.south)}
}

// subpixel draws one subpixel of the planet, or of its air.
func (p *Planet) subpixel(b *light.Buffer, eye view.Eye, w *world,
	x, y int, sun view.Vec, f turned, pixel, t float64) {
	dir := eye.Ray(x, y, b.W, b.H)
	oc := sub(eye.At, p.At)
	a := dot(dir, dir)
	half := dot(oc, dir)
	c := dot(oc, oc) - p.Radius*p.Radius
	disc := half*half - a*c
	// how close the ray passes to the middle, as a share of the radius
	near := math.Sqrt(math.Max(0, dot(oc, oc)-half*half/a)) / p.Radius
	depthMid := -half / a // depth of the closest approach
	if depthMid <= 0 {
		return // behind the camera
	}
	if disc < 0 {
		p.halo(b, x, y, eye, dir, near, depthMid, sun)
		return
	}
	// where the ray enters the sphere, which is always nearer than its
	// closest approach to the middle. Culling on the closest approach hid
	// surface that was in front of what culled it: with a planet at 4.2 of
	// radius 1 the surface is entered at 3.2, and a wall at 3.7 hid it.
	depth := (-half - math.Sqrt(disc)) / a
	if depth <= 0 || depth > b.Depth(x, y) {
		return
	}
	hit := add(eye.At, scale(dir, depth))
	n := scale(sub(hit, p.At), 1/p.Radius)
	view_ := scale(unit(dir), -1)
	col := p.surface(w, n, view_, sun, f, depth, pixel, t)
	// the limb, softened over about a subpixel so the disc is round
	// rather than stepped
	edge := p.Radius * (1 - near) / (depth * pixel)
	cover := clamp01(edge + 0.5)
	b.Set(x, y, b.At(x, y).Mix(col, cover))
	b.SetDepth(x, y, depth)
	p.halo(b, x, y, eye, dir, near, depthMid, sun)
}

// surface is the light leaving the planet at unit normal n, toward the eye.
func (p *Planet) surface(w *world, n, toEye, sun view.Vec, f turned,
	depth, pixel, t float64) light.RGB {
	local := f.body(n)
	tile := w.hexes.Nearest(local)
	biome := w.biome[tile]
	base := p.Colours[biome]

	// relief: the tile's slope tips its normal, so a raised tile is lit on
	// the side facing the sun
	slope := w.slope[tile]
	worldSlope := add(add(scale(f.east, slope.X), scale(f.north, slope.Y)),
		scale(f.south, slope.Z))
	shaded := unit(add(n, scale(worldSlope, p.Relief)))
	lit := clamp01((dot(shaded, sun) + 0.08) / 1.08)
	col := base.Scale(0.03 + lit)
	if g := p.Glow[biome]; g > 0 {
		// what glows is its own light, and the sun on it hardly shows
		col = base.Scale(g + 0.1*lit)
	}

	// the lines between tiles, while a tile is big enough to hold one
	across := w.hexes.Spacing * p.Radius / (depth * pixel)
	if p.Lines > 0 && across > 4 {
		// a subpixel, measured on the sphere
		thin := 0.6 * depth * pixel / p.Radius
		width := math.Max(w.hexes.Spacing*0.06, thin)
		line := 1 - smooth(0, width, w.hexes.Edge(local, tile))
		col = col.Scale(1 - p.Lines*line*smooth(4, 9, across))
	}

	// water glints where the sun would reflect straight at the eye
	if biome <= Sea && p.Glow[biome] == 0 {
		half := unit(add(sun, toEye))
		spec := 0.8 * math.Pow(math.Max(0, dot(n, half)), 70)
		col = col.Plus(light.RGB{R: 1, G: 0.95, B: 0.85}.Scale(spec))
	}

	// on the night side, cities
	night := 1 - smooth(-0.15, 0.1, dot(n, sun))
	if w.lights[tile] > 0 && night > 0 {
		d := math.Acos(math.Min(1, dot(local, w.hexes.Centres[tile])))
		spot := 1 - smooth(0, w.hexes.Spacing*0.3, d)
		col = col.Plus(light.RGB{R: 1, G: 0.72, B: 0.35}.Scale(
			1.4 * w.lights[tile] * spot * night))
	}

	if p.Clouds > 0 {
		col = p.cloud(col, w, local, n, sun, across, t)
	}

	// the air, thickest looking at the rim
	rim := math.Pow(1-math.Max(0, dot(n, toEye)), 3)
	sky := clamp01(dot(n, sun) + 0.35)
	return col.Plus(p.Air.Scale(p.AirDepth * rim * sky))
}

// cloud lays cloud over the ground at a point.
//
// Cloud is hexes too, or the planet stops looking like a board: a second
// layer of the same tiles, turning a little slower than the ground so the
// weather drifts across it.
//
// Each cloud tile is in or out by noise read at its middle, and that noise
// slowly changes, so a cloud tile fades in and out rather than popping.
func (p *Planet) cloud(col light.RGB, w *world, local, n, sun view.Vec,
	across, t float64) light.RGB {
	drift := -0.012 * t
	c, s := math.Cos(drift), math.Sin(drift)
	sky := view.Vec{X: local.X*c - local.Z*s, Y: local.Y,
		Z: local.X*s + local.Z*c}
	tile := w.hexes.Nearest(sky)
	m := w.hexes.Centres[tile]
	v := noise.Fractal(p.Seed+90, m.X*2.4, m.Y*2.4+t*0.006, m.Z*2.4, 4)
	q := 0.32 * (0.5 - p.Clouds)
	cover := smooth(q, q+0.12, v)
	if cover <= 0 {
		return col
	}
	lit := clamp01((dot(n, sun) + 0.1) / 1.1)
	white := light.RGB{R: 0.95, G: 0.96, B: 1}.Scale(0.04 + lit)
	if across > 4 {
		// a cloud tile's edge is a little shaded, so it reads as a
		// tile of its own over the tiles below
		edge := w.hexes.Edge(sky, tile)
		line := 1 - smooth(0, w.hexes.Spacing*0.1, edge)
		white = white.Scale(1 - 0.3*line*smooth(4, 9, across))
	}
	return col.Mix(white, cover*0.92)
}

// halo adds the atmosphere seen past the planet's edge: a ray that misses
// but passes close still goes through air, lit if that side faces the sun.
func (p *Planet) halo(b *light.Buffer, x, y int, eye view.Eye, dir view.Vec,
	near, depth float64, sun view.Vec) {
	if p.AirDepth <= 0 || near < 1 {
		return
	}
	thick := 0.08
	if near > 1+thick {
		return
	}
	// air is light on the way to the eye, so anything nearer hides it. It
	// is added rather than laid over, which is only right because what is
	// behind was drawn first: Draw orders bodies furthest away first, and
	// without that a far planet's face erased a near planet's air.
	if depth > b.Depth(x, y) {
		return
	}
	glow := math.Pow(1-(near-1)/thick, 2)
	closest := add(eye.At, scale(dir, depth))
	side := clamp01(dot(unit(sub(closest, p.At)), sun) + 0.4)
	b.Add(x, y, p.Air.Scale(p.AirDepth*glow*side*0.8))
}

// worldFor is the terrain, made on first use and again if what it is made
// from has changed.
func (p *Planet) worldFor() *world {
	p.mu.Lock()
	defer p.mu.Unlock()
	tr := terrain{tiles: max(1, p.Tiles), seed: p.Seed, ocean: p.Ocean,
		plates: p.Plates, axis: view.Vec{Y: 1}, cold: p.Cold,
		cities: p.Cities}
	if p.world == nil || p.made != tr {
		p.world, p.made = newWorld(tr), tr
	}
	return p.world
}

// Away is how far the planet's middle is from the eye, for a drawer putting
// bodies in order.
func (p *Planet) Away(eye view.Eye) float64 {
	return length(sub(p.At, eye.At))
}
