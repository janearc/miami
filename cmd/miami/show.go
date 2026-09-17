package main

import (
	"fmt"
	"os"
	"time"

	"github.com/janearc/miami/cells"
	"github.com/janearc/miami/light"
	"github.com/janearc/miami/proof"
	"github.com/janearc/miami/scenery"
	"github.com/janearc/miami/view"
)

// show draws a piece of scenery once, to pngs, and says what it cost.
//
//	miami show ember|azure|quasar [COLS ROWS] [--t SECONDS]
func show(args []string) {
	if len(args) < 1 {
		usage()
	}
	cols, rows, at := 160, 48, 0.0
	var rest []string
	for i := 1; i < len(args); i++ {
		if args[i] == "--t" && i+1 < len(args) {
			fmt.Sscan(args[i+1], &at)
			i++
			continue
		}
		rest = append(rest, args[i])
	}
	if len(rest) > 1 {
		fmt.Sscan(rest[0], &cols)
		fmt.Sscan(rest[1], &rows)
	}
	looks := map[string]func() *scenery.BlackHole{
		"ember": scenery.Ember, "azure": scenery.Azure,
		"quasar": scenery.Quasar,
	}
	mk, ok := looks[args[0]]
	if !ok {
		fmt.Fprintln(os.Stderr, "no scenery called", args[0])
		os.Exit(2)
	}
	hole := mk()
	eye := view.Eye{Lens: 1, Near: 0.05}
	w, h := cols*2, rows*3

	start := time.Now()
	b := light.New(w, h)
	hole.Draw(b, eye, at)
	first := time.Since(start)

	start = time.Now()
	b = light.New(w, h)
	hole.Draw(b, eye, at+1.0/24)
	b.Bloom(0.6, light.Glow{Radius: 1.5, Strength: 0.6},
		light.Glow{Radius: 8, Strength: 0.45},
		light.Glow{Radius: 24, Strength: 0.3})
	px := b.RGBA(1)
	frame := time.Since(start)
	cs := cells.ToCells(px, 2, 3)

	name := "out/show-" + args[0]
	proof.Must(os.MkdirAll("out", 0o755))
	proof.Must(proof.WritePNGXY(name+"-px.png", px, 3, 4))
	proof.Must(proof.WritePNGXY(name+".png", cells.CellsToPixels(cs, 2, 3),
		3, 4))
	ms := func(d time.Duration) time.Duration {
		return d.Round(time.Millisecond)
	}
	fmt.Printf("%s at %dx%d: first frame %v (bake included), "+
		"next %v -> %s.png\n", args[0], cols, rows, ms(first),
		ms(frame), name)
}
