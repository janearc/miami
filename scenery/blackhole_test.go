package scenery

import (
	"math"
	"testing"

	"github.com/janearc/miami/light"
	"github.com/janearc/miami/view"
)

// frame draws a small ember at t.
func frame(h *BlackHole, t float64) *light.Buffer {
	b := light.New(120, 66)
	h.Draw(b, view.Eye{Lens: 1, Near: 0.05}, t)
	return b
}

// total is all the light in a frame.
func total(b *light.Buffer) float64 {
	sum := 0.0
	for y := range b.H {
		for x := range b.W {
			sum += b.At(x, y).Y()
		}
	}
	return sum
}

// A moment drawn twice is the same picture, and a later moment is a
// different one: the disk turns.
func TestTheDiskTurnsAndAMomentRepeats(t *testing.T) {
	h := Ember()
	a, again, later := frame(h, 3), frame(h, 3), frame(h, 3.5)
	same, moved := true, false
	for y := range a.H {
		for x := range a.W {
			if a.At(x, y) != again.At(x, y) {
				same = false
			}
			if a.At(x, y) != later.At(x, y) {
				moved = true
			}
		}
	}
	if !same {
		t.Fatal("the same moment drew two different pictures")
	}
	if !moved {
		t.Fatal("half a second later nothing had moved")
	}
}

// The disk does not pulse with its crossfade. Its texture is two copies
// faded into each other, and two noises halfway through a fade are flatter
// than either, which dims a disk once every half span unless corrected.
//
// The disk's light also wanders on its own, by ten percent or so, as bright
// clumps turn through the side coming toward the camera. So one run of frames
// cannot tell a pulse from weather.
//
// Frames are averaged instead by where in the fade they fall, over many fades
// and four disks, so the weather cancels. What is left is compared where the
// two copies are equal, in the middle, against where one of them is nearly
// all of it, at the ends. Uncorrected, the middle sags by three percent or
// more; corrected, it is within about one and a half either way.
func TestTheCrossfadeDoesNotPulse(t *testing.T) {
	const fades = 24
	var ends, middle float64
	for _, seed := range []uint64{1, 2, 3, 7} {
		h := Ember()
		h.Seed = seed
		half := math.Abs(h.Turn) * 0.8 / 2
		for f := range fades {
			at := func(k float64) float64 { return (float64(f) + k) * half }
			ends += total(frame(h, at(0.04))) + total(frame(h, at(0.96)))
			middle += total(frame(h, at(0.46))) + total(frame(h, at(0.54)))
		}
	}
	if sag := (ends - middle) / ends; math.Abs(sag) > 0.025 {
		t.Fatalf("the middle of the fade is %.1f%% dimmer than its ends",
			100*sag)
	}
}

// The rays are traced once for a size and a camera, and kept.
func TestTheBakeIsKept(t *testing.T) {
	h := Ember()
	frame(h, 0)
	first := h.baked
	frame(h, 2)
	if h.baked != first {
		t.Fatal("the same size and camera baked again")
	}
	b := light.New(120, 66)
	h.Draw(b, view.Eye{Lens: 1.1, Near: 0.05}, 0)
	if h.baked == first {
		t.Fatal("a new lens used the old bake")
	}
}

// The middle of the shadow is black, and it hides what is behind the hole:
// its depth is the hole's, where nothing was drawn before.
func TestTheShadowIsBlackAndSolid(t *testing.T) {
	h := Ember()
	h.At = view.Vec{Z: 60} // straight ahead, so the middle is the shadow
	h.Axis = view.Vec{Y: 1, Z: -0.05}
	b := frame(h, 1)
	x, y := b.W/2, b.H/2-3 // above the disk's own plane
	if c := b.At(x, y); c.Y() > 1e-3 {
		t.Fatalf("the shadow's middle has light in it: %v", c)
	}
	if d := b.Depth(x, y); math.Abs(d-60) > 1e-6 {
		t.Fatalf("the shadow's depth is %v, want 60", d)
	}
	if d := b.Depth(0, 0); !math.IsInf(d, 1) {
		t.Fatalf("the sky in a corner has a depth of %v", d)
	}
}
