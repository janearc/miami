package scenery

import (
	"testing"

	"github.com/janearc/miami/light"
	"github.com/janearc/miami/particles"
	"github.com/janearc/miami/view"
)

// A body and a light in one buffer.
//
// The rule this pins down was derived by opezart, in compose, by writing the
// smallest scene that has one of each and watching it fail both ways. It is
// what this library already did everywhere, and had written down nowhere,
// which is why he had to find it by failing rather than by reading -- and why
// the advice I gave him was the opposite of my own code.
//
//	a solid owns its subpixel; a light does not.
//	a light reads the depth and never writes it.
//	a shared buffer is drawn bodies first, then lights.
//
// There is no fourth arrangement. A light that writes depth culls the body
// behind it, and a body is opaque where it draws -- Mix at full cover
// replaces the subpixel, light and all -- so a light drawn first is simply
// lost. Both halves below are that, as tests.

// lamp is a light at a place: one particle, alive for the whole test, bright
// enough to see against a lit planet.
func lamp(at view.Vec) particles.Emitter {
	return particles.Emitter{Kind: particles.Drift, Seed: 3,
		At: at, Axis: view.Vec{Z: 1}, Rate: 1, Life: 1000,
		Speed: 0, Radius: 1e-9, Size: 0.5, Light: 40,
		Colour: light.HexRamp(0.0, "ff2200", 1.0, "ff2200")}
}

// world is a planet filling the middle of the frame at a distance.
func globe(z float64) *Planet {
	p := Terran()
	p.At, p.Radius, p.AirDepth = view.Vec{Z: z}, 1.4, 0
	return p
}

// lit is the light in the middle of the frame after drawing bodies, then
// lights, in that order.
func lit(t *testing.T, body *Planet, e particles.Emitter) light.RGB {
	t.Helper()
	b := light.New(80, 40)
	eye := view.Eye{Lens: 1}
	Draw(b, eye, 0, body)
	e.Draw(b, eye, 0)
	return b.At(40, 20)
}

// A light in front of a body adds to it: the body is still there and the
// light is on top of it. Nothing is taken away -- no channel goes down.
//
// This one does not fail when a light writes depth, and an earlier version of
// this comment said it did. porter ran the mutation: with the bodies drawn
// first there is nothing left behind the light to cull, so the third test is
// the one that catches it. A comment that names the wrong test is worse than
// no comment, because the next person stops looking.
func TestALightInFrontOfABodyAddsToIt(t *testing.T) {
	body := globe(6)
	alone := lit(t, body, particles.Emitter{})
	if alone == (light.RGB{}) {
		t.Fatal("the body did not draw at all")
	}
	both := lit(t, body, lamp(view.Vec{Z: 4}))
	if both == alone {
		t.Fatal("a light in front of the body changed nothing")
	}
	for what, pair := range map[string][2]float64{
		"red":   {both.R, alone.R},
		"green": {both.G, alone.G},
		"blue":  {both.B, alone.B},
	} {
		if pair[0] < pair[1] {
			t.Errorf("%s went down, from %g to %g: the light took"+
				" the body away rather than adding to it",
				what, pair[1], pair[0])
		}
	}
	if both.R <= alone.R {
		t.Errorf("a red light in front of the body left red at %g,"+
			" was %g", both.R, alone.R)
	}
}

// A light behind a body is hidden by it, exactly: the subpixel is the body
// and nothing else.
func TestALightBehindABodyIsHiddenByIt(t *testing.T) {
	body := globe(6)
	alone := lit(t, body, particles.Emitter{})
	behind := lit(t, body, lamp(view.Vec{Z: 9}))
	if behind != alone {
		t.Errorf("a light behind the body changed the subpixel to %v,"+
			" from %v; a body is opaque where it draws", behind,
			alone)
	}
}

// And the order is the contract, not a preference: drawn first, the light is
// lost, because the body replaces the subpixel it covers.
//
// This is also the test that catches a light writing its own depth, which is
// the mistake I recommended to opezart: the light writes depth 4, the body at
// 6 is then culled where the light is, and what is left is not the body alone.
//
// This is why scenery.Draw takes the bodies as a list rather than trusting
// the caller, and why a stage draws its lights last.
func TestALightDrawnBeforeTheBodyIsLost(t *testing.T) {
	body, e := globe(6), lamp(view.Vec{Z: 4})
	eye := view.Eye{Lens: 1}

	right := light.New(80, 40)
	Draw(right, eye, 0, body)
	e.Draw(right, eye, 0)

	wrong := light.New(80, 40)
	e.Draw(wrong, eye, 0)
	Draw(wrong, eye, 0, body)

	if wrong.At(40, 20) == right.At(40, 20) {
		t.Fatal("drawing the light first made no difference; this " +
			"test is not testing the order")
	}
	only := light.New(80, 40)
	Draw(only, eye, 0, body)
	if wrong.At(40, 20) != only.At(40, 20) {
		t.Errorf("the light drawn first left %v, the body alone is"+
			" %v; they should be the same, the light being gone",
			wrong.At(40, 20), only.At(40, 20))
	}
}
