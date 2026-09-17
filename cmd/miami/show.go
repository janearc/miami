package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/janearc/miami/cells"
	"github.com/janearc/miami/light"
	"github.com/janearc/miami/proof"
	"github.com/janearc/miami/scenery"
	"github.com/janearc/miami/view"
)

// painter is anything that draws into a frame of light.
type painter interface {
	Draw(b *light.Buffer, eye view.Eye, t float64)
}

// pieces is the scenery show can draw, by name: what to draw, back to front.
var pieces = map[string]func() []painter{
	"ember":  func() []painter { return []painter{scenery.Ember()} },
	"azure":  func() []painter { return []painter{scenery.Azure()} },
	"quasar": func() []painter { return []painter{scenery.Quasar()} },
	"terran": world(scenery.Terran),
	"arid":   world(scenery.Arid),
	"frozen": world(scenery.Frozen),
	"molten": world(scenery.Molten),
}

// world is a planet against the night.
func world(mk func() *scenery.Planet) func() []painter {
	return func() []painter {
		sky := &scenery.Backdrop{Space: scenery.Night()}
		return []painter{sky, mk()}
	}
}

// show draws a piece of scenery once, to pngs, and says what it cost.
//
//	miami show NAME [COLS ROWS] [--t SECONDS]
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
	mk, ok := pieces[args[0]]
	if !ok {
		var names []string
		for n := range pieces {
			names = append(names, n)
		}
		sort.Strings(names)
		fmt.Fprintf(os.Stderr, "no scenery called %s; there is %s\n",
			args[0], strings.Join(names, " "))
		os.Exit(2)
	}
	ps := mk()
	eye := view.Eye{Lens: 1, Near: 0.05}
	w, h := cols*2, rows*3
	draw := func(t float64) *light.Buffer {
		b := light.New(w, h)
		for _, p := range ps {
			p.Draw(b, eye, t)
		}
		return b
	}

	start := time.Now()
	draw(at)
	first := time.Since(start)

	start = time.Now()
	b := draw(at + 1.0/24)
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
