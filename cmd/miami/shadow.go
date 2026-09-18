package main

import (
	"fmt"
	"image/color"
	"math"
	"time"

	"github.com/janearc/miami/compose"
	"github.com/janearc/miami/proof"
	"github.com/janearc/miami/view"
)

// shadow is a ball over a floor, with a light somewhere in the scene and a
// shadow that is cast rather than drawn: for every point of floor, one ray
// toward the lamp, and whether it arrives.
func shadow(args []string) {
	cols, rows, secs := 90, 26, 8.0
	size(args, &cols, &rows, &secs)
	col, err := compose.Colours("sprites/made/starling.css")
	proof.Must(err)

	r := reel{name: "shadow", secs: secs, fps: 15, shots: 6,
		note: []string{"a ball over a floor, and a shadow cast by a" +
			" located light."}}
	n := r.run(func(t float64) [][]color.RGBA {
		e := t / secs
		// the floor sits below the eye and the eye looks along it
		eye := view.Eye{At: view.Vec{Y: 0.30}, Lens: 1.0, Near: 0.05}
		floor := view.Floor{
			Y: -0.55, Tile: 0.62, Far: 9.5, A: 'V', B: 'K',
			// a shadow is a remap, so every colour in it was
			// already chosen by the colourway
			Shade: map[byte]byte{'V': 'K', 'K': 'D'},
		}
		// a bounce: the height is a sine's absolute, so it arrives at
		// the floor rather than easing into it
		bounce := math.Abs(math.Sin(3 * math.Pi * e))
		ball := &view.Ball{
			At: view.Vec{
				X: 1.30 * math.Sin(2*math.Pi*e),
				Y: -0.21 + 0.66*bounce,
				Z: 3.6 - 0.7*math.Cos(2*math.Pi*e),
			},
			Radius: 0.34,
			Spin:   view.Vec{X: 0.11, Y: 0.43, Z: 0.07},
		}
		// the lamp: high, and behind the ball as well as above it, so
		// its shadow is thrown toward the viewer.
		//
		// with the lamp in front, every shadow fell away from the
		// camera and hid behind the thing casting it.
		lamp := view.Light{At: view.Vec{X: -1.2, Y: 3.0, Z: 1.6}}
		sc := view.Lit{Eye: eye, Lights: []view.Light{lamp},
			Solids: []view.Solid{floor, ball}}
		f := sc.Trace(time.Duration(t*float64(time.Second)), cols, rows)
		px := ground(cols*2, rows*3, color.RGBA{10, 9, 16, 255})
		compose.Blit(px, f, col, 0, 0)
		return px
	})
	fmt.Printf("%d frames -> out/shadow.frames\n", n)
}
