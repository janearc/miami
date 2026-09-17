package scenery

import (
	"github.com/janearc/miami/light"
	"github.com/janearc/miami/view"
)

// Looks: the same machinery, dressed three ways.
//
// A look is numbers, and every one of them could be written in a scene
// document instead. These exist so that a game wanting a black hole behind it
// asks for one by name and gets something already worth looking at.

// Ember is a black hole with a thin, hot disk seen nearly edge on, in the
// orange of gas at a few thousand degrees.
func Ember() *BlackHole {
	return &BlackHole{
		At: view.Vec{X: 7, Y: 2, Z: 44}, Mass: 1,
		Axis:  view.Vec{X: -0.09, Y: 1, Z: -0.12},
		Inner: 6, Outer: 28, Turn: 5, Beaming: 0.35,
		Heat: 0.95, Bright: 3, Opacity: 2.4, Seed: 7,
		Disk: light.HexRamp(0.0, "0a0200", 0.22, "3f0e02",
			0.48, "a8360a", 0.68, "ec7f33", 0.85, "ffcf98",
			1.0, "fff7ec"),
		Space: Space{Seed: 3, Stars: 1, Twinkle: 0.4, Nebula: 0.04,
			Gas:  light.HexRamp(0.0, "140a06", 1.0, "3a2414"),
			Dust: 0.25},
	}
}

// Azure is the same hole, cooler: a violet and rose disk round a white-blue
// heart, seen from a little higher.
func Azure() *BlackHole {
	h := Ember()
	h.Axis = view.Vec{X: 0.05, Y: 1, Z: -0.3}
	h.At = view.Vec{X: 0, Y: -9, Z: 66}
	h.Beaming, h.Heat, h.Bright = 0.5, 0.78, 2.6
	h.Disk = light.HexRamp(0.0, "07031a", 0.3, "3d1d6e", 0.52, "a8428f",
		0.7, "f08fb8", 0.86, "d8e6ff", 1.0, "ffffff")
	h.Space.Gas = light.HexRamp(0.0, "060818", 1.0, "1c2448")
	h.Space.Nebula = 0.06
	return h
}

// Quasar is a hole feeding hard: a tilted disk, a violet galaxy behind it,
// and the jets that come with it drawn as particles by whoever places it.
func Quasar() *BlackHole {
	h := Ember()
	h.Axis = view.Vec{X: 0.45, Y: 1, Z: -0.55}
	h.At = view.Vec{X: -4, Y: -1, Z: 62}
	h.Inner, h.Outer = 6, 34
	h.Beaming, h.Heat, h.Bright = 0.3, 0.9, 2.2
	h.Disk = light.HexRamp(0.0, "0c0616", 0.3, "4a2266", 0.5, "b0502e",
		0.7, "f39a45", 0.86, "ffd9a0", 1.0, "fffaf0")
	h.Space = Space{Seed: 11, Stars: 1.2, Twinkle: 0.3, Nebula: 0.5,
		Gas: light.HexRamp(0.0, "120a2a", 0.4, "3b2a78",
			0.7, "8a4fb0", 1.0, "d59ad6"),
		Dust: 0.45, Band: view.Vec{X: 0.3, Y: 1, Z: 0.2}, Width: 0.35,
		Glow: 0.12}
	return h
}
