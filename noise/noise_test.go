package noise

import (
	"math"
	"testing"
)

// The same inputs give the same number, and different seeds give different
// ones.
func TestHashIsRepeatableAndSeeded(t *testing.T) {
	if Hash(1, 42) != Hash(1, 42) || Hash3(9, 1, 2, 3) != Hash3(9, 1, 2, 3) {
		t.Fatal("a hash changed between two asks")
	}
	if Hash(1, 42) == Hash(2, 42) {
		t.Fatal("two seeds gave one number")
	}
}

// Neighbouring indices give numbers spread evenly over nought to one: each of
// ten bins gets its tenth of a hundred thousand, within a few percent. The
// sine trick fails this for neighbours, which is why it is not used here.
func TestHashSpreadsNeighbours(t *testing.T) {
	var bins [10]int
	const n = 100000
	for i := range int64(n) {
		h := Hash2(3, i, i+1)
		if h < 0 || h >= 1 {
			t.Fatalf("hash %v out of range", h)
		}
		bins[int(h*10)]++
	}
	for b, c := range bins {
		if math.Abs(float64(c)-n/10) > n/10*0.04 {
			t.Errorf("bin %d holds %d of %d", b, c, n)
		}
	}
}

// Gradient noise is nought on the lattice, bounded, and continuous: a step of
// a thousandth moves it by a small amount everywhere it is tried.
func TestGradientIsSmoothAndBounded(t *testing.T) {
	for i := range int64(2000) {
		x := Hash(1, i)*40 - 20
		y := Hash(2, i)*40 - 20
		z := Hash(3, i)*40 - 20
		v := Gradient(5, x, y, z)
		if math.Abs(v) > 1.05 {
			t.Fatalf("noise %v at %v,%v,%v", v, x, y, z)
		}
		if d := math.Abs(Gradient(5, x+1e-3, y, z) - v); d > 0.01 {
			t.Fatalf("a step of 1e-3 moved it by %v", d)
		}
		ix, iy, iz := math.Floor(x), math.Floor(y), math.Floor(z)
		if g := Gradient(5, ix, iy, iz); g != 0 {
			t.Fatalf("not nought on the lattice: %v", g)
		}
	}
}
