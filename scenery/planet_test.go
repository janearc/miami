package scenery

import (
	"math"
	"testing"

	"github.com/janearc/miami/light"
	"github.com/janearc/miami/view"
)

// A world gets the share of sea it asked for, to within a tile, and its land
// is mostly continents rather than a scatter of single tiles.
func TestAWorldHasTheSeaItAskedFor(t *testing.T) {
	for _, share := range []float64{0.3, 0.6, 0.8} {
		w := newWorld(terrain{tiles: 14, seed: 5, ocean: share,
			plates: 11, axis: view.Vec{Y: 1}})
		sea, alone := 0, 0
		for i, b := range w.biome {
			if b <= Ice {
				sea++
				continue
			}
			company := false
			for _, j := range w.hexes.Neighbours[i] {
				company = company || w.biome[j] > Ice
			}
			if !company {
				alone++
			}
		}
		n := len(w.biome)
		if got := float64(sea) / float64(n); math.Abs(got-share) >
			1.5/float64(n) {
			t.Errorf("asked for %.2f sea, got %.4f", share, got)
		}
		if float64(alone) > 0.02*float64(n) {
			t.Errorf("sea %.2f: %d land tiles with no land beside them",
				share, alone)
		}
	}
}

// planetFrame draws a world into a small frame.
func planetFrame(p *Planet, t float64, depth float64) *light.Buffer {
	b := light.New(160, 90)
	if depth > 0 {
		for y := range b.H {
			for x := range b.W {
				b.SetDepth(x, y, depth)
			}
		}
	}
	p.Draw(b, view.Eye{Lens: 1, Near: 0.05}, t)
	return b
}

// The sunlit side is brighter than the far side, the world turns, and a
// moment drawn twice is the same picture.
func TestAPlanetIsLitAndTurns(t *testing.T) {
	p := Terran()
	p.Clouds, p.AirDepth, p.Cities = 0, 0, 0
	p.Sun = view.Vec{X: -1}
	a, again, later := planetFrame(p, 10, 0), planetFrame(p, 10, 0),
		planetFrame(p, 25, 0)
	var left, right float64
	moved := false
	for y := range a.H {
		for x := range a.W {
			if a.At(x, y) != again.At(x, y) {
				t.Fatalf("subpixel %d,%d differs between two draws",
					x, y)
			}
			moved = moved || a.At(x, y) != later.At(x, y)
			if x < a.W/2 {
				left += a.At(x, y).Y()
			} else {
				right += a.At(x, y).Y()
			}
		}
	}
	if !(left > 3*right) {
		t.Errorf("lit from the left, but left %.1f and right %.1f",
			left, right)
	}
	if !moved {
		t.Error("fifteen seconds later the world had not turned")
	}
}

// Something nearer than the planet hides it, and the planet records its own
// depth where it is drawn.
func TestAPlanetRespectsDepth(t *testing.T) {
	p := Terran()
	hidden := planetFrame(p, 0, 1) // a wall at depth one, in front
	for y := range hidden.H {
		for x := range hidden.W {
			if c := hidden.At(x, y); c != (light.RGB{}) {
				t.Fatalf("drawn through a nearer wall at %d,%d: %v",
					x, y, c)
			}
		}
	}
	open := planetFrame(p, 0, 0)
	mid := open.Depth(open.W/2, open.H/2)
	if math.Abs(mid-(p.At.Z-p.Radius)) > 0.01 {
		t.Fatalf("depth at the middle %v, want %v", mid,
			p.At.Z-p.Radius)
	}
}
