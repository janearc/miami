package scenery

import (
	"github.com/janearc/miami/light"
	"github.com/janearc/miami/view"
)

// Worlds: planets dressed four ways. As with the black holes, every number
// here could be written in a scene document; these are the ones to start from.

// palette is eleven hex colours in biome order: deep, sea, ice, beach,
// desert, grass, forest, jungle, tundra, rock, snow.
func palette(hex ...string) [biomes]light.RGB {
	var out [biomes]light.RGB
	for i, h := range hex {
		out[i] = light.Hex(h)
	}
	return out
}

// Terran is a blue and green world with white poles, weather and cities.
func Terran() *Planet {
	return &Planet{
		At: view.Vec{Z: 4.2}, Radius: 1, Tiles: 14, Seed: 5,
		Ocean: 0.6, Plates: 11, Axis: view.Vec{X: 0.2, Y: 1}, Day: 90,
		Sun: view.Vec{X: -1, Y: 0.3, Z: -0.15},
		Colours: palette("0a2452", "1d5c9a", "d6e6f0", "d8c48c",
			"d0a45e", "7da744", "3b7838", "27662c", "8c9a7c",
			"7a6e62", "f2f5f8"),
		Lines: 0.6, Relief: 0.45, Clouds: 0.3, Cities: 0.08,
		Air: light.Hex("5aa0ff"), AirDepth: 0.7,
	}
}

// Arid is a rust-red world of dry basins and a thin, warm sky.
func Arid() *Planet {
	p := Terran()
	p.Seed, p.Ocean, p.Plates, p.Clouds, p.Cities = 21, 0.3, 8, 0.08, 0
	p.Colours = palette("3a1c12", "5c2c1a", "e6d6c4", "b86a3c",
		"c56a36", "a85c32", "8c4c2c", "7a4024", "9a7a6a", "5e4234",
		"e8dcd0")
	p.Air, p.AirDepth = light.Hex("ffae7a"), 0.4
	return p
}

// Frozen is an ice world: pale continents, dark cold seas, and a blue rim.
func Frozen() *Planet {
	p := Terran()
	p.Seed, p.Cold, p.Clouds, p.Cities = 33, 0.55, 0.3, 0
	p.Colours = palette("0c263e", "24506e", "cfe4f2", "aebfcc",
		"c6d6de", "9ab4bc", "5c7884", "4c6874", "a8bec6", "6a7680",
		"f4faff")
	p.Air, p.AirDepth = light.Hex("a8d8ff"), 0.8
	return p
}

// Molten is a world whose seas are lava: dark crust cracked into hexes over
// light of its own.
func Molten() *Planet {
	p := Terran()
	p.Seed, p.Ocean, p.Clouds, p.Cities, p.Lines = 47, 0.45, 0, 0, 0.8
	p.Colours = palette("ff5a0a", "e8400a", "4a3a34", "3a2a24",
		"2c201c", "342622", "2a1e1a", "241a16", "3a302a", "1e1612",
		"4a3e38")
	p.Glow[Deep], p.Glow[Sea] = 1.1, 0.7
	p.Air, p.AirDepth = light.Hex("ff6a2a"), 0.5
	return p
}

// Night is the sky a world is shown against: stars and a little gas.
func Night() Space {
	return Space{Seed: 9, Stars: 1, Twinkle: 0.4, Nebula: 0.08,
		Gas: light.HexRamp(0.0, "0a0c1c", 1.0, "2a2248"), Dust: 0.2}
}
