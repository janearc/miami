package main

import (
	"github.com/janearc/miami/cells"
	"github.com/janearc/miami/light"
	"github.com/janearc/miami/particles"
	"github.com/janearc/miami/scenery"
	"github.com/janearc/miami/view"
)

// scenery: the three things a scene document can put behind a game, drawn
// side by side at the size a page has room for.
func sceneryPage() ([][]cells.Cell, string) {
	const cols, rows = 78, 20
	const wide = cols / 3
	px := blank(cols*2, rows*3)
	for i, draw := range []func(*light.Buffer){holePanel, worldPanel,
		sparkPanel} {
		b := light.New(wide*2, rows*3)
		draw(b)
		b.Bloom(0.7, light.Glow{Radius: 2, Strength: 0.5},
			light.Glow{Radius: 9, Strength: 0.3})
		for y, row := range b.RGBA(1) {
			copy(px[y][i*wide*2:], row)
		}
	}
	return cells.ToCells(px, 2, 3), `
Three kinds, one document. A black hole, a world of hexes, and particles.

    { "kind": "blackhole", "at": {"z": 44}, "hole": {"look": "ember"} }
    { "kind": "planet",    "at": {"z": 4.2}, "world": {"look": "terran"} }
    { "kind": "emitter",   "at": {"z": 6},
      "emit": {"kind": "drift", "rate": 25, "life": 6, "radius": 5} }
    { "kind": "space",     "space": {"look": "night"} }

Every one starts from a look and says only what it changes, so a field left
out is the look's. A look nobody has is refused when the stage is built,
naming the ones there are, and a block on the wrong kind is refused when the
document is read.

The numbers are read again every frame, so any of them may be a binding: a
disk's brightness on "@level" pulses with the music, and a moon's place is
three sines. What is expensive to change is kept -- where a hole's rays go,
a world's terrain -- and worked out again only when a number it depends on
moves.

None of the left-hand picture is drawn. Rays are traced along the orbit
equation for light near a mass, so the far side of the disk lifted over the
shadow, the thin ring inside its rim and the stars smeared round it are what
bent light does. The gas turns at the orbital speed of its own radius.

The middle is a Goldberg polyhedron: hexagons and exactly twelve pentagons,
the tiling a football has. Plates drift and collide, which is where mountains
and coasts come from; cold comes from latitude and height, rain from distance
to the sea, and the two pick what grows.

The right is dust, each mote's place a formula in its number and the time.
Nothing is stored between frames, so a game may jump to any moment.

    miami show blackhole        full screen, until q
    miami show quasar --png     one frame, and what it cost
    miami icon                  the quasar as an app icon
`
}

// holePanel is a black hole, small.
func holePanel(b *light.Buffer) {
	h := scenery.Ember()
	h.At = view.Vec{Z: 30}
	h.Outer, h.Samples = 14, 3
	h.Space.Stars, h.Space.Nebula = 0.2, 0
	h.Draw(b, view.Eye{Lens: 1, Near: 0.05}, 3)
}

// worldPanel is a hex world.
func worldPanel(b *light.Buffer) {
	sky := scenery.Night()
	sky.Nebula, sky.Stars = 0, 0.25
	eye := view.Eye{Lens: 1, Near: 0.05}
	(&scenery.Backdrop{Space: sky}).Draw(b, eye, 0)
	p := scenery.Terran()
	p.At, p.Radius, p.Tiles = view.Vec{Z: 3.1}, 1, 9
	p.Draw(b, eye, 6)
}

// sparkPanel is a fountain of particles.
func sparkPanel(b *light.Buffer) {
	e := particles.Emitter{Kind: particles.Fountain, Seed: 4,
		At: view.Vec{Y: -0.55, Z: 2.6}, Axis: view.Vec{Y: 1},
		Gravity: view.Vec{Y: -1.4}, Rate: 120, Life: 3, Speed: 1.5,
		Spread: 0.45, Size: 0.012, Light: 1.1, Streak: 0.06,
		Colour: light.HexRamp(0.0, "fff6e0", 0.5, "ffb060",
			1.0, "c03010")}
	e.Draw(b, view.Eye{Lens: 1, Near: 0.05}, 9)
}
