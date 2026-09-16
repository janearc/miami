package view

import (
	"math"
	"testing"
)

// TestQuadrantsMatchTheBuiltInPattern walks the whole sphere and asks both the
// old code and the texture for the role at every point.
//
// The pattern moved out of the sphere and into a texture. A pattern that is
// nearly the same is worse than one that is different: it looks right, and the
// check sizes and phases are wrong in a way nobody sees until they compare two
// renders side by side months later.
//
// So this is the port's proof, the same one the motion library got. Every point
// or it did not move.
func TestQuadrantsMatchTheBuiltInPattern(t *testing.T) {
	tex := Quadrants()
	const steps = 400
	bad := 0
	for i := 0; i <= steps; i++ {
		// latitude short of the poles: at the pole itself longitude is
		// undefined and both sides are picking from a singularity, which
		// is a property of spheres and not a disagreement
		lat := (float64(i)/steps - 0.5) * math.Pi * 0.998
		for j := 0; j < steps; j++ {
			lon := (float64(j)/steps)*2*math.Pi - math.Pi
			n := Vec{
				X: math.Cos(lat) * math.Sin(lon),
				Y: math.Sin(lat),
				Z: math.Cos(lat) * math.Cos(lon),
			}
			want := surfaceAt(rotation(0, 0, 0), n)
			u, v := SphereUV(n)
			if got := tex.At(u, v); got != want {
				if bad++; bad < 6 {
					t.Errorf("lat %.4f lon %.4f: texture %q, built in %q",
						lat, lon, got, want)
				}
			}
		}
	}
	if bad > 0 {
		t.Fatalf("%d of %d points disagree", bad, (steps+1)*steps)
	}
}
