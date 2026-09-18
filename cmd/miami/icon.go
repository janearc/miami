package main

import (
	"fmt"
	"image"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/janearc/miami/light"
	"github.com/janearc/miami/particles"
	"github.com/janearc/miami/proof"
	"github.com/janearc/miami/scenery"
	"github.com/janearc/miami/view"
)

// An application's icon, drawn by the renderer rather than by a hand.
//
// Not cells. Everywhere else here the answer is a terminal's cells, because a
// terminal is what is being drawn to; an icon is looked at in a dock at any
// size, so this writes the light the scenery made, at whatever square the
// caller asks for.
//
//	miami icon [SIZE] [--dir dist]
//
// Two of them, because a Mac shows a different one on a dark desktop and a
// light one. They are the same hole from the same camera: what changes is the
// sky behind it. On the dark one that is space; on the light one it is a lit
// sky, which a black hole is a black disc against.
//
// The square is honest rather than fitted afterwards. The sheet is as wide as
// the picture and three quarters as tall, which is the shape that makes the
// field of view square, and the rows are stretched back to the square on the
// way out.
func icon(args []string) {
	size, dir := 1024, "dist"
	if len(args) > 0 && args[0] != "--dir" {
		fmt.Sscan(args[0], &size)
	}
	for i, a := range args {
		if a == "--dir" && i+1 < len(args) {
			dir = args[i+1]
		}
	}
	proof.Must(os.MkdirAll(dir, 0o755))
	for _, ground := range []string{"dark", "light"} {
		start := time.Now()
		px := iconLight(size, ground == "light")
		name := filepath.Join(dir,
			fmt.Sprintf("miami-icon-%s-%d.png", ground, size))
		proof.Must(proof.SaveImage(name, square(px, size)))
		fmt.Printf("%s  %dx%d in %v\n", name, size, size,
			time.Since(start).Round(time.Millisecond))
	}
}

// iconLight draws the hole and its jets into a sheet as wide as the icon and
// three quarters as tall.
func iconLight(size int, day bool) *light.Buffer {
	w, h := size, size*3/4
	hole := scenery.Quasar()
	// in the middle, and near enough that the shadow is about a third of
	// the square: an icon is read at thirty pixels as often as at a
	// thousand, and what has to survive that is the black disc
	hole.At = view.Vec{Z: 38}
	hole.Axis = view.Vec{X: 0.42, Y: 1, Z: -0.5}
	hole.Outer, hole.Bright, hole.Heat = 18, 2.6, 0.95
	hole.Samples = 4
	// a thinner sky than the scene's: an icon wants the hole, not weather
	hole.Space.Stars, hole.Space.Twinkle = 0.45, 0
	hole.Space.Nebula, hole.Space.Dust = 0.05, 0.45
	if day {
		// a lit sky, no stars and no gas: what is left is the shadow,
		// the disk and the jets, which is what an icon can hold
		hole.Space = scenery.Space{Ambient: light.Hex("efecf6").
			Scale(0.62)}
	}
	jets := particles.Emitter{Kind: particles.Jet, Seed: 5,
		At: hole.At, Axis: hole.Axis, Rate: 260, Life: 2.2, Speed: 26,
		Spread: 0.035, Wobble: 0.05, Size: 0.2, Light: 1.5,
		Streak: 0.05, Colour: light.HexRamp(0.0, "f6f2ff",
			0.35, "cba6ff", 1.0, "5a2a8a")}

	b := light.New(w, h)
	eye := view.Eye{Lens: 1, Near: 0.05}
	hole.Draw(b, eye, 7.4)
	jets.Draw(b, eye, 7.4)
	// the wide glow is halved on the lit one: spread over a bright sky it
	// is what turned the shadow from black to grey, and the shadow is the
	// one thing the icon cannot lose
	wide, mid := 0.07, 0.36
	if day {
		wide, mid = 0.05, 0.22
	}
	b.Bloom(0.6, light.Glow{Radius: float64(size) / 120, Strength: 0.6},
		light.Glow{Radius: float64(size) / 24, Strength: mid},
		light.Glow{Radius: float64(size) / 8, Strength: wide})
	return b
}

// square is the sheet as a square image: the rows are stretched back, since a
// sheet row covers four thirds of what a column does.
func square(b *light.Buffer, size int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	for y := range size {
		v := (float64(y)+0.5)*float64(b.H)/float64(size) - 0.5
		y0 := int(math.Floor(v))
		t := v - float64(y0)
		for x := range size {
			c := mixRows(b, x, y0, t)
			img.Set(x, y, light.Display(c))
		}
	}
	return img
}

// mixRows is the light at a column, between two rows.
func mixRows(b *light.Buffer, x, y0 int, t float64) light.RGB {
	at := func(y int) light.RGB {
		return b.At(x, min(b.H-1, max(0, y)))
	}
	return at(y0).Mix(at(y0+1), t)
}
