package scenery

import (
	"math"
	"testing"

	"github.com/janearc/miami/noise"
	"github.com/janearc/miami/view"
)

// Splitting each face n times gives 10n^2+2 tiles: exactly twelve pentagons
// and the rest hexagons. That count is the Goldberg polyhedron's, and a
// tiling off by one tile has a seam somewhere.
func TestTwelvePentagonsAndTheRestHexagons(t *testing.T) {
	for _, n := range []int{1, 2, 5, 16} {
		h := NewHexes(n)
		if want := 10*n*n + 2; len(h.Centres) != want {
			t.Fatalf("n=%d: %d tiles, want %d", n, len(h.Centres), want)
		}
		five, six := 0, 0
		for i, ns := range h.Neighbours {
			switch len(ns) {
			case 5:
				five++
			case 6:
				six++
			default:
				t.Fatalf("n=%d: tile %d has %d neighbours", n, i,
					len(ns))
			}
		}
		if five != 12 {
			t.Fatalf("n=%d: %d pentagons", n, five)
		}
	}
}

// Nearest finds the same tile as asking every tile, for points all over the
// sphere, pentagons included.
func TestNearestIsTheTrueNearest(t *testing.T) {
	h := NewHexes(12)
	for k := range int64(20000) {
		p := unit(view.Vec{X: noise.Hash(1, k)*2 - 1,
			Y: noise.Hash(2, k)*2 - 1, Z: noise.Hash(3, k)*2 - 1})
		got := h.Nearest(p)
		best, want := -2.0, -1
		for i, c := range h.Centres {
			if d := dot(p, c); d > best {
				best, want = d, i
			}
		}
		if got != want && math.Abs(dot(p, h.Centres[got])-best) > 1e-12 {
			t.Fatalf("point %d: tile %d, want %d", k, got, want)
		}
	}
}

// A point is further from its tile's edge the nearer it is to the middle,
// and at the midpoint between two neighbours it is on the edge.
func TestEdgeIsNoughtBetweenNeighbours(t *testing.T) {
	h := NewHexes(8)
	for i := 0; i < len(h.Centres); i += 37 {
		c := h.Centres[i]
		n := h.Centres[h.Neighbours[i][0]]
		mid := unit(add(c, n))
		if e := h.Edge(mid, i); math.Abs(e) > 1e-9 {
			t.Fatalf("tile %d: midpoint is %v from the edge", i, e)
		}
		if h.Edge(c, i) <= h.Edge(unit(add(scale(c, 3), n)), i) {
			t.Fatalf("tile %d: the middle is no further from the edge",
				i)
		}
	}
}
