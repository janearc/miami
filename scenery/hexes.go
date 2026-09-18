package scenery

import (
	"math"
	"sort"

	"github.com/janearc/miami/view"
)

// Hexes cover a sphere in hexagons, and twelve pentagons.
//
// A sphere cannot be tiled in hexagons alone -- Euler says so, and a football
// shows it. The tiling that comes closest starts from an icosahedron: split
// each of its twenty triangles into n rows of smaller triangles, push every
// corner out onto the sphere, and take each corner as the middle of a tile.
//
// A corner where six small triangles meet becomes a hexagon. The icosahedron's
// own twelve corners, where five meet, become the pentagons. There are 10n^2
// + 2 tiles, and every one is nearly the same size.
//
// That is a Goldberg polyhedron, and it is the tiling hex-grid planets in
// games are built on.
//
// The tiles are never stored as outlines. A point on the sphere is in
// whichever tile's middle is nearest, and it is near an edge where the nearest
// two are nearly as near as each other; both are questions a renderer can ask
// a subpixel at a time.
type Hexes struct {
	N          int        // rows each icosahedron face was split into
	Centres    []view.Vec // unit vectors, one a tile
	Neighbours [][]int32  // tiles sharing an edge, five or six of them
	Spacing    float64    // radians between neighbouring middles, typically

	grid int     // cube cells a face side, for finding a start
	seed []int32 // per cube cell, the tile nearest its middle
}

// NewHexes tiles a sphere with each icosahedron face split n times.
func NewHexes(n int) *Hexes {
	n = max(n, 1)
	h := &Hexes{N: n}
	verts, faces := icosahedron()
	index := map[[3]int64]int32{}
	add := func(p view.Vec) int32 {
		p = unit(p)
		round := func(v float64) int64 {
			return int64(math.Round(v * 1e9))
		}
		k := [3]int64{round(p.X), round(p.Y), round(p.Z)}
		if i, ok := index[k]; ok {
			return i
		}
		i := int32(len(h.Centres))
		index[k] = i
		h.Centres = append(h.Centres, p)
		return i
	}
	edges := map[[2]int32]bool{}
	link := func(a, b int32) {
		if a > b {
			a, b = b, a
		}
		edges[[2]int32{a, b}] = true
	}
	for _, f := range faces {
		a, b, c := verts[f[0]], verts[f[1]], verts[f[2]]
		at := func(i, j int) int32 {
			fi, fj := float64(i)/float64(n), float64(j)/float64(n)
			return add(add3(scale(a, 1-fi-fj), scale(b, fi),
				scale(c, fj)))
		}
		// every small triangle pointing up links its three corners;
		// the ones pointing down share all their edges with those
		for i := 0; i < n; i++ {
			for j := 0; i+j < n; j++ {
				p, q, r := at(i, j), at(i+1, j), at(i, j+1)
				link(p, q)
				link(q, r)
				link(r, p)
			}
		}
	}
	h.Neighbours = make([][]int32, len(h.Centres))
	for e := range edges {
		h.Neighbours[e[0]] = append(h.Neighbours[e[0]], e[1])
		h.Neighbours[e[1]] = append(h.Neighbours[e[1]], e[0])
	}
	for i := range h.Neighbours {
		ns := h.Neighbours[i]
		sort.Slice(ns, func(a, b int) bool { return ns[a] < ns[b] })
	}
	h.Spacing = math.Acos(math.Min(1, dot(h.Centres[0],
		h.Centres[h.Neighbours[0][0]])))
	h.buildSeeds()
	return h
}

// icosahedron is the twelve corners and twenty faces.
func icosahedron() ([]view.Vec, [][3]int) {
	g := (1 + math.Sqrt(5)) / 2
	v := []view.Vec{
		{X: -1, Y: g}, {X: 1, Y: g}, {X: -1, Y: -g}, {X: 1, Y: -g},
		{Y: -1, Z: g}, {Y: 1, Z: g}, {Y: -1, Z: -g}, {Y: 1, Z: -g},
		{X: g, Z: -1}, {X: g, Z: 1}, {X: -g, Z: -1}, {X: -g, Z: 1},
	}
	f := [][3]int{
		{0, 11, 5}, {0, 5, 1}, {0, 1, 7}, {0, 7, 10}, {0, 10, 11},
		{1, 5, 9}, {5, 11, 4}, {11, 10, 2}, {10, 7, 6}, {7, 1, 8},
		{3, 9, 4}, {3, 4, 2}, {3, 2, 6}, {3, 6, 8}, {3, 8, 9},
		{4, 9, 5}, {2, 4, 11}, {6, 2, 10}, {8, 6, 7}, {9, 8, 1},
	}
	return v, f
}

// buildSeeds finds, for each cell of a cube laid over the sphere, the tile
// nearest the cell's middle. A search that starts there has a step or two to
// go at most.
func (h *Hexes) buildSeeds() {
	h.grid = max(4, h.N*2)
	h.seed = make([]int32, 6*h.grid*h.grid)
	for i := range h.seed {
		h.seed[i] = -1
	}
	// every tile claims the cell it is in; cells no tile landed in are
	// then filled by searching from a neighbouring cell's tile
	for i, c := range h.Centres {
		h.seed[h.cell(c)] = int32(i)
	}
	for face := range 6 {
		for j := range h.grid {
			for i := range h.grid {
				k := (face*h.grid+j)*h.grid + i
				if h.seed[k] >= 0 {
					continue
				}
				a := (float64(i)+0.5)/float64(h.grid)*2 - 1
				b := (float64(j)+0.5)/float64(h.grid)*2 - 1
				h.seed[k] = int32(h.walk(uncube(face, a, b), 0))
			}
		}
	}
}

// cell is which cube cell a unit vector falls in.
func (h *Hexes) cell(p view.Vec) int {
	face, a, b := cube(p)
	g := float64(h.grid)
	i := min(h.grid-1, max(0, int((a+1)/2*g)))
	j := min(h.grid-1, max(0, int((b+1)/2*g)))
	return (face*h.grid+j)*h.grid + i
}

// Nearest is the tile a unit vector is in.
func (h *Hexes) Nearest(p view.Vec) int {
	return h.walk(p, int(h.seed[h.cell(p)]))
}

// walk goes from tile to neighbouring tile while a neighbour is nearer p, and
// stops at the nearest.
//
// On a triangulation this even, stepping to a nearer neighbour always ends at
// the true nearest; the test holds it to a brute-force search over every
// tile. Near a tie the two are the same tile to within a hair.
func (h *Hexes) walk(p view.Vec, from int) int {
	at := from
	best := dot(p, h.Centres[at])
	for {
		next := -1
		for _, n := range h.Neighbours[at] {
			if d := dot(p, h.Centres[n]); d > best {
				best, next = d, int(n)
			}
		}
		if next < 0 {
			return at
		}
		at = next
	}
}

// Edge is how far p is from the edge of tile i, in radians: nought on the
// edge, growing toward the middle. The edge with a neighbour is the plane
// square to the chord between the two middles, halfway along it.
func (h *Hexes) Edge(p view.Vec, i int) float64 {
	c := h.Centres[i]
	best := math.Inf(1)
	for _, n := range h.Neighbours[i] {
		d := sub(c, h.Centres[n])
		l := length(d)
		mid := scale(add(c, h.Centres[n]), 0.5)
		best = math.Min(best, dot(sub(p, mid), d)/l)
	}
	return best
}

// add3 is three vectors summed.
func add3(a, b, c view.Vec) view.Vec { return add(add(a, b), c) }
