package scenery

import (
	"math"
	"sort"

	"github.com/janearc/miami/noise"
	"github.com/janearc/miami/view"
)

// A world is a hex tiling with land on it.
//
// Noise alone makes a planet that looks like noise: blobs of land with no
// reason to be where they are. What makes coastlines and mountain ranges look
// like they belong is plates, and the model here is small.
//
// The surface is cut into plates, each drifting its own way. Some plates are
// continent and float high; the rest are ocean floor and sit low.
//
// Where two plates push into each other the ground is thrown up, into
// mountains if both are continent and into a trench beside a coast if one is
// ocean. Where they pull apart it sags into a rift, or rises a little into a
// mid-ocean ridge.
//
// Those effects fade with distance from the boundary, noise roughens
// everything, and the sea is poured in until the right share of tiles is under
// it. Temperature comes from latitude and height, rain from distance to the
// sea, and the two together pick what grows.

// Biome is what a tile is.
type Biome uint8

const (
	Deep Biome = iota
	Sea
	Ice
	Beach
	Desert
	Grass
	Forest
	Jungle
	Tundra
	Rock
	Snow
	biomes
)

// world is the terrain of one tiling, worked out once.
type world struct {
	hexes  *Hexes
	height []float64 // above the sea is positive
	biome  []Biome
	slope  []view.Vec // per tile, which way the ground falls, for relief
	lights []float64  // per tile, how brightly its night side is lit
}

// terrain is how a world is made.
type terrain struct {
	tiles  int     // face splits
	seed   uint64  //
	ocean  float64 // share of tiles under the sea
	plates int
	axis   view.Vec // the world's own north, for latitude
	cold   float64  // added to every tile's cold; a frozen world is high
	cities float64  // share of land tiles lit at night
}

// newWorld makes the terrain.
func newWorld(tr terrain) *world {
	h := NewHexes(tr.tiles)
	n := len(h.Centres)
	w := &world{hexes: h, height: make([]float64, n),
		biome: make([]Biome, n), slope: make([]view.Vec, n),
		lights: make([]float64, n)}
	plate, land, drift := w.plates(tr)
	stress := w.boundaries(plate, land, drift)
	for i, c := range h.Centres {
		base := -0.45
		if land[plate[i]] {
			base = 0.22
		}
		rough := noise.Fractal(tr.seed+40, c.X*2.2, c.Y*2.2, c.Z*2.2, 4)
		w.height[i] = base + 0.5*stress[i] + 0.26*rough
	}
	w.pourSea(tr.ocean)
	w.classify(tr)
	w.relief()
	return w
}

// plates cuts the tiles into plates: each tile goes to the nearest of a few
// scattered middles, measured through noise so the plates are not polygons.
// It returns each tile's plate, which plates are continent, and how each
// plate moves: a turning about an axis through the world's middle.
func (w *world) plates(tr terrain) ([]int, []bool, []view.Vec) {
	h := w.hexes
	count := max(2, tr.plates)
	mids := make([]view.Vec, count)
	land := make([]bool, count)
	drift := make([]view.Vec, count)
	for p := range count {
		k := int64(p)
		mids[p] = unit(view.Vec{X: noise.Hash(tr.seed+1, k)*2 - 1,
			Y: noise.Hash(tr.seed+2, k)*2 - 1,
			Z: noise.Hash(tr.seed+3, k)*2 - 1})
		land[p] = noise.Hash(tr.seed+4, k) < 0.42
		drift[p] = scale(unit(view.Vec{X: noise.Hash(tr.seed+5, k) - .5,
			Y: noise.Hash(tr.seed+6, k) - .5,
			Z: noise.Hash(tr.seed+7, k) - .5}), 1)
	}
	plate := make([]int, len(h.Centres))
	for i, c := range h.Centres {
		warp := noise.Fractal(tr.seed+8, c.X*1.8, c.Y*1.8, c.Z*1.8, 3)
		best := math.Inf(1)
		for p, m := range mids {
			d := math.Acos(math.Max(-1, math.Min(1, dot(c, m))))
			d += 0.55 * warp * noise.Hash(tr.seed+9, int64(p))
			if d < best {
				best, plate[i] = d, p
			}
		}
	}
	return plate, land, drift
}

// boundaries is how much each tile is raised or lowered by the plates meeting
// near it.
func (w *world) boundaries(plate []int, land []bool,
	drift []view.Vec) []float64 {
	h := w.hexes
	n := len(h.Centres)
	push := make([]float64, n)
	var front []int32
	for i, c := range h.Centres {
		for _, j := range h.Neighbours[i] {
			pi, pj := plate[i], plate[int(j)]
			if pi == pj {
				continue
			}
			// how fast the two plates close on each other here:
			// each plate's ground moves as its turning axis
			// crossed with where it is
			away := unit(sub(h.Centres[j], c))
			rel := sub(cross(drift[pi], c), cross(drift[pj], c))
			closing := dot(rel, away)
			effect := 0.0
			switch {
			case closing > 0 && land[pi] && land[pj]:
				effect = 1.1 * closing // ranges
			case closing > 0 && land[pi]:
				effect = 0.8 * closing // a coast pushed up
			case closing > 0 && land[pj]:
				effect = -0.6 * closing // the trench beside it
			case closing > 0:
				effect = 0.5 * closing // an island arc
			case land[pi] && land[pj]:
				effect = 0.35 * closing // a rift
			default:
				effect = -0.12 * closing // a ridge, small
			}
			if math.Abs(effect) > math.Abs(push[i]) {
				push[i] = effect
			}
		}
		if push[i] != 0 {
			front = append(front, int32(i))
		}
	}
	// spread inland: each tile takes the effect of the nearest boundary,
	// fading with how many tiles away it is
	reach := make([]int, n)
	for i := range reach {
		reach[i] = -1
	}
	for _, i := range front {
		reach[i] = 0
	}
	fall := math.Max(1.5, float64(h.N)/6)
	out := make([]float64, n)
	for len(front) > 0 {
		var next []int32
		for _, i := range front {
			out[i] = push[i] * math.Exp(-float64(reach[i])/fall)
			for _, j := range h.Neighbours[i] {
				if reach[j] >= 0 {
					continue
				}
				reach[j] = reach[i] + 1
				push[j] = push[i]
				next = append(next, j)
			}
		}
		front = next
	}
	return out
}

// pourSea lowers every tile by the height that leaves the given share under
// water, so a world asked for sixty percent sea gets sixty percent.
//
// Then land is scaled so its highest few percent reach one, and sea floor so
// its deepest few reach minus one. How tall the plates threw the mountains
// depends on how fast they happened to be moving, and a biome rule written as
// "above 0.8 is snow" should mean the same on every world.
func (w *world) pourSea(share float64) {
	sorted := append([]float64(nil), w.height...)
	sort.Float64s(sorted)
	last := float64(len(sorted) - 1)
	level := sorted[int(math.Round(clamp01(share)*last))]
	top := sorted[int(0.985*last)] - level
	bottom := level - sorted[int(0.015*last)]
	for i := range w.height {
		e := w.height[i] - level
		if e > 0 && top > 0 {
			e /= top
		}
		if e < 0 && bottom > 0 {
			e /= bottom
		}
		w.height[i] = e
	}
}

// classify picks each tile's biome from its height, its cold and its rain.
func (w *world) classify(tr terrain) {
	h := w.hexes
	wet := w.distanceToSea()
	north := unit(tr.axis)
	for i, c := range h.Centres {
		e := w.height[i]
		lat := math.Abs(dot(c, north))
		nz := noise.Fractal(tr.seed+60, c.X*4, c.Y*4, c.Z*4, 3)
		cold := lat*1.05 + math.Max(0, e)*0.45 + 0.12*nz + tr.cold
		rain := math.Exp(-float64(wet[i])/math.Max(2, float64(h.N)/5)) +
			0.3*noise.Fractal(tr.seed+61, c.X*3, c.Y*3, c.Z*3, 3)
		w.biome[i] = pick(e, cold, rain, wet[i])
		if w.biome[i] >= Beach && w.biome[i] <= Jungle &&
			noise.Hash(tr.seed+70, int64(i)) < tr.cities {
			w.lights[i] = 0.4 + 0.6*noise.Hash(tr.seed+71, int64(i))
		}
	}
}

// pick is a biome from height, cold and rain.
func pick(e, cold, rain float64, fromSea int) Biome {
	if e < 0 {
		switch {
		case cold > 1.0:
			return Ice
		case e < -0.35:
			return Deep
		}
		return Sea
	}
	switch {
	case cold > 1.08 || e > 0.82:
		return Snow
	case e > 0.62:
		return Rock
	case cold > 0.88:
		return Tundra
	case fromSea == 1 && e < 0.03:
		return Beach
	case rain < 0.28:
		return Desert
	case rain < 0.5:
		return Grass
	case cold < 0.35 && rain > 0.75:
		return Jungle
	}
	return Forest
}

// distanceToSea is, for every tile, how many tiles away the nearest sea is:
// nought for the sea itself.
func (w *world) distanceToSea() []int {
	h := w.hexes
	d := make([]int, len(h.Centres))
	var front []int32
	for i := range d {
		d[i] = -1
		if w.height[i] < 0 {
			d[i] = 0
			front = append(front, int32(i))
		}
	}
	for len(front) > 0 {
		var next []int32
		for _, i := range front {
			for _, j := range h.Neighbours[i] {
				if d[j] < 0 {
					d[j] = d[i] + 1
					next = append(next, j)
				}
			}
		}
		front = next
	}
	for i := range d {
		if d[i] < 0 {
			d[i] = len(d) // a world with no sea at all
		}
	}
	return d
}

// relief is which way each tile's ground falls, from how much higher it is
// than each neighbour: a tile above its neighbours catches light on the side
// facing it and shades on the other, which is what makes a hex look raised.
func (w *world) relief() {
	h := w.hexes
	for i, c := range h.Centres {
		var g view.Vec
		for _, j := range h.Neighbours[i] {
			here, there := w.height[i], w.height[j]
			up := math.Max(0, here) - math.Max(0, there)
			g = add(g, scale(unit(sub(c, h.Centres[j])), up))
		}
		w.slope[i] = g
	}
}
