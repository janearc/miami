package main

import (
	"fmt"
	"image/color"
	"math"
	"os"
	"time"

	"github.com/janearc/miami/compose"
	"github.com/janearc/miami/grade"
	"github.com/janearc/miami/motion"
	"github.com/janearc/miami/proof"
	"github.com/janearc/miami/view"

	sprite "github.com/janearc/libdaffy-sprite"
)

// ball is a checkered ball on nothing at all: spinning, moving, and coming
// toward the viewer. No background, so what is being looked at is the object
// and not the picture around it.
func ball(args []string) {
	cols, rows, secs := 90, 26, 12.0
	size(args, &cols, &rows, &secs)
	const fps = 15
	sh := grade.Sextants(cols, rows)
	col, err := compose.Colours("sprites/made/starling.css")
	proof.Must(err)
	eye := view.DefaultEye()

	r := reel{name: "ball", secs: secs, fps: fps, shots: 8,
		note: []string{
			"a checkered ball: spinning, scaling and moving."}}
	n := r.run(func(t float64) [][]color.RGBA {
		e := t / secs
		px := ground(sh.W, sh.H, color.RGBA{14, 13, 22, 255})
		b := view.Ball{
			// across, and toward the viewer and away again, so the
			// same object is seen at several sizes without anything
			// resizing it
			At: view.Vec{
				X: 0.62 * math.Sin(2*math.Pi*e),
				Y: 0.22 * math.Sin(4*math.Pi*e),
				Z: 2.9 - 1.25*math.Sin(math.Pi*e),
			},
			Radius: 0.42,
			Spin:   view.Vec{X: 0.13, Y: 0.37, Z: 0.05},
		}
		at := time.Duration(t * float64(time.Second))
		for _, p := range view.Resolve(
			[]view.Object{b}, eye, at, sh.W, sh.H,
		) {
			compose.Blit(px, p.Frame, col, p.X, p.Y)
		}
		return px
	})
	fmt.Printf("%d frames -> out/ball.frames\n", n)
	baked(eye, n, fps)
}

// baked writes the same spin out as a daffy document, both ways, so the trade
// between size and compute is a measurement rather than a claim.
func baked(eye view.Eye, n, fps int) {
	hold := time.Duration(float64(time.Second) / float64(fps))
	spin := view.Ball{At: view.Vec{Z: 2.4}, Radius: 0.42,
		Spin: view.Vec{X: 0.13, Y: 0.37, Z: 0.05}}
	var frames []sprite.Frame
	for i := 0; i < n; i++ {
		ns := float64(i) / float64(fps) * float64(time.Second)
		f := spin.Render(eye, time.Duration(ns), 24)
		f.Hold = hold
		frames = append(frames, f)
	}
	proof.Must(motion.WriteDoc("out/ball-full.json", "ball", frames, false))
	proof.Must(motion.WriteDoc("out/ball.json", "ball", frames, true))
	back, err := motion.ReadDoc("out/ball.json")
	proof.Must(err)
	for i := range frames {
		for y := range frames[i].Art {
			if back[i].Art[y] == frames[i].Art[y] {
				continue
			}
			proof.Must(fmt.Errorf("frame %d row %d did not replay",
				i, y))
		}
	}
	full, _ := os.Stat("out/ball-full.json")
	deltas, _ := os.Stat("out/ball.json")
	pct := 100 * float64(deltas.Size()) / float64(full.Size())
	fmt.Printf("baked %d frames: full %d bytes,"+
		" deltas %d bytes (%.0f%%), replayed exactly\n",
		len(frames), full.Size(), deltas.Size(), pct)
}
