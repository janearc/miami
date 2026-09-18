package light

import (
	"math"
	"sync"
	"testing"

	"github.com/janearc/libtheme-css/primitives/swatch"
)

// What Kelvin keeps, and the thing found while testing that it keeps it.
//
// swatch.Planckian does not answer the same temperature the same way twice:
// asked for 5200K two hundred times it gave two hundred different colours,
// differing in the last bit or two. It is summing over a map, and a map's
// order is randomised per run.
//
// Nothing visible comes of it -- the differences are a part in ten to the
// fourteenth and the frame ends up as eight bits a channel -- but it means two
// things here. A test of this cache cannot ask for bit equality against a
// fresh call, because the fresh call disagrees with itself. And the cache
// makes a temperature answer the same way all run, which the library it
// wraps does not.

// close is whether two colours are the same to a part in a billion, which is
// far below anything a terminal can show and far above Planckian's own wobble.
func close(a, b RGB) bool {
	return math.Abs(a.R-b.R) < 1e-9 && math.Abs(a.G-b.G) < 1e-9 &&
		math.Abs(a.B-b.B) < 1e-9
}

// planckian is Kelvin with nothing kept: roughly what the answer must be.
func planckian(k float64) RGB {
	c := FromSwatch(swatch.Planckian(k))
	if y := c.Y(); y > 0 {
		return c.Scale(1 / y)
	}
	return RGB{}
}

// A kept answer is the answer Planck's law gives.
func TestAKeptTemperatureIsTheColourPlanckGives(t *testing.T) {
	for _, k := range []float64{0, 1, 1000, 2800, 3333.3333, 5200, 5200.0001,
		9876.5, 13800, 40000, 1e6} {
		want := planckian(k)
		for i := range 3 { // first the miss, then the hits
			if got := Kelvin(k); !close(got, want) {
				t.Errorf("%gK asked %d times: %v, want %v", k,
					i+1, got, want)
			}
		}
	}
}

// And it is the same answer every time, which is the property the library
// underneath does not have. A star asked for from two rows of a bake must not
// come out two colours.
func TestATemperatureAnswersTheSameWayEveryTime(t *testing.T) {
	const k = 6543.21
	first := Kelvin(k)
	for i := range 500 {
		if got := Kelvin(k); got != first {
			t.Fatalf("%gK on ask %d: %v, first time %v", k, i+2,
				got, first)
		}
	}
	// and the library it wraps does not manage this, which is why the
	// test above cannot ask for it of planckian
	loose := 0
	for range 200 {
		if planckian(k) != first {
			loose++
		}
	}
	if loose == 0 {
		t.Log("swatch.Planckian is stable now; this cache no longer" +
			" has to make it so")
	}
}

// Two temperatures that land in the same slot must not be given each other's
// colour. The slot holds the temperature it was made for and is only trusted
// on an exact match; without that check this is what breaks, and it breaks as
// a star of the wrong colour rather than as an error.
func TestTwoTemperaturesInOneSlotDoNotMix(t *testing.T) {
	a := 5200.0
	var b float64
	for k := 2800.0; k < 14000; k += 0.5 {
		if k != a && kelvinSlot(k) == kelvinSlot(a) {
			b = k
			break
		}
	}
	if b == 0 {
		t.Skip("no second temperature lands in that slot")
	}
	wantA, wantB := planckian(a), planckian(b)
	if close(wantA, wantB) {
		t.Fatalf("%gK and %gK are the same colour; they prove nothing",
			a, b)
	}
	for range 4 { // one evicts the other, over and over
		if got := Kelvin(a); !close(got, wantA) {
			t.Fatalf("%gK after %gK took its slot: %v, want %v",
				a, b, got, wantA)
		}
		if got := Kelvin(b); !close(got, wantB) {
			t.Fatalf("%gK after %gK took its slot: %v, want %v",
				b, a, got, wantB)
		}
	}
}

// The rows of a bake are drawn in parallel and all of them ask, so the answer
// has to be right when a hundred goroutines are filling the same slots. Run
// with -race, this is also what says the slots are shared safely.
func TestTemperaturesAskedFromEveryRowAtOnce(t *testing.T) {
	const goroutines, each = 64, 200
	var wg sync.WaitGroup
	bad := make([]float64, goroutines)
	for g := range goroutines {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := range each {
				// temperatures a sky actually asks for: a
				// smooth function of a hash, so they arrive in
				// runs and land in a handful of slots
				h := float64((g*each+i)%997) / 997
				k := 2800 + 11000*math.Pow(h, 2)
				if !close(Kelvin(k), planckian(k)) {
					bad[g] = k
					return
				}
			}
		}(g)
	}
	wg.Wait()
	for g, k := range bad {
		if k != 0 {
			t.Fatalf("goroutine %d got the wrong colour for %gK"+
				" while every other row was asking too", g, k)
		}
	}
}
