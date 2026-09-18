package main

import (
	"fmt"
	"image/color"
	"math"
	"time"

	"github.com/janearc/miami/compose"
	"github.com/janearc/miami/grade"
	"github.com/janearc/miami/motion"
	"github.com/janearc/miami/proof"
	"github.com/janearc/miami/scene"
	"github.com/janearc/miami/view"
)

// bird is one starling's place in the flock: how far away it is, how high,
// where it started, how fast it crosses, and where its wingbeat is.
type bird struct {
	z, y, x0, speed float64
	phase           time.Duration
}

// flock is nine birds at different depths over the dusk. Every bird is the
// same sprite; what differs is where it is, and the viewport does the rest.
//
// The near ones are big and cross fast, the far ones are small and barely
// move. That is parallax and it comes out of the projection for nothing --
// nothing here is told to make the distant ones slower.
func flock(args []string) {
	cols, rows, secs := 110, 30, 20.0
	size(args, &cols, &rows, &secs)
	sh := grade.Sextants(cols, rows)
	doc, err := scene.Load("scenes/dusk.json")
	proof.Must(err)
	frames, err := motion.ReadStream(starlingStream)
	proof.Must(err)
	col, err := compose.Colours("sprites/made/starling.css")
	proof.Must(err)
	eye := view.DefaultEye()

	var birds []bird
	for i := 0; i < 9; i++ {
		f := float64(i)
		birds = append(birds, bird{
			z:     1.2 + 1.35*f,
			y:     -0.12 + 0.075*math.Sin(f*2.1),
			x0:    -1.8 - 0.55*f,
			speed: 0.55 + 0.05*math.Sin(f*1.7),
			phase: time.Duration(f*137) * time.Millisecond,
		})
	}

	r := reel{name: "flock", secs: secs, fps: 15, shots: 6,
		note: []string{"a flock at nine depths over a dusk." +
			" one sprite, one viewport."}}
	n := r.run(func(t float64) [][]color.RGBA {
		st, err := doc.At(t)
		proof.Must(err)
		px := sh.Paint(st)
		objs := make([]view.Object, 0, len(birds))
		for _, b := range birds {
			// wrapped, so the sky does not empty out:
			//
			// a bird that leaves on the right comes back on the
			// left, and at its own depth nobody can tell it is the
			// same one
			x := math.Mod(b.x0+b.speed*t+4.5, 9) - 4.5
			y := b.y + 0.05*math.Sin(t*0.9+b.z)
			objs = append(objs, view.Billboard{
				Frames: frames, Height: 0.5, Phase: b.phase,
				At: view.Vec{X: x, Y: y, Z: b.z},
			})
		}
		for _, p := range view.Resolve(
			objs, eye, time.Duration(t*float64(time.Second)),
			sh.W, sh.H,
		) {
			compose.Blit(px, p.Frame, col, p.X, p.Y)
		}
		return px
	})
	fmt.Printf("%d frames -> out/flock.frames\n", n)
}

// fly is a dusk that does not repeat, and a starling crossing it.
func fly(args []string) {
	cols, rows, secs, fps := 100, 28, 20.0, 15
	size(args, &cols, &rows, &secs)
	if len(args) > 3 {
		fmt.Sscan(args[3], &fps)
	}
	sh := grade.Sextants(cols, rows)
	doc, err := scene.Load("scenes/dusk.json")
	proof.Must(err)
	secs = doc.Seconds
	frames, err := motion.ReadStream(starlingStream)
	proof.Must(err)
	col, err := compose.Colours("sprites/made/starling.css")
	proof.Must(err)
	bw, bh := compose.Size(frames[0])

	r := reel{name: "fly", secs: secs, fps: fps, shots: 6}
	r.note = []string{
		"a dusk that does not repeat, and a starling crossing it.",
		fmt.Sprintf("%d by %d cells, %d frames at %d per second.",
			cols, rows, r.count(), fps),
	}
	n := r.run(func(t float64) [][]color.RGBA {
		st, err := doc.At(t)
		proof.Must(err)
		px := sh.Paint(st)
		// the bird crosses once, which is its own kind of not
		// repeating, and rises as the light goes. its wingbeat keeps
		// its own clock.
		e := t / secs
		x := int(float64(-bw) + float64(cols*2+2*bw)*e)
		y := int(float64(rows*3)*(0.62-0.34*e) - float64(bh)/2 +
			float64(rows)*0.22*math.Sin(2*math.Pi*t/6.4))
		at := time.Duration(t * float64(time.Second))
		compose.Over(px, compose.At(frames, at), col, x, y)
		return px
	})
	fmt.Printf("%d frames, %.0fs -> out/fly.frames\n", n, secs)
}
