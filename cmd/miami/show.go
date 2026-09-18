package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/janearc/miami/app"
	"github.com/janearc/miami/cells"
	"github.com/janearc/miami/proof"
)

// show runs a scene: scenery at full screen, until q. With --png it draws one
// frame to out/ instead, and says what it cost, which is how an agent looks at
// scenery without a terminal to watch it in.
//
//	miami show scene [cols rows] [--t seconds] [--png]
//
// The scene is a built-in -- blackhole, quasar, hexworld, with or without the
// @ -- or a document on disk.
func show(args []string) {
	if len(args) < 1 {
		usage()
	}
	cols, rows, at, png := 0, 0, 0.0, false
	var rest []string
	for i := 1; i < len(args); i++ {
		switch {
		case args[i] == "--t" && i+1 < len(args):
			fmt.Sscan(args[i+1], &at)
			i++
		case args[i] == "--png":
			png = true
		default:
			rest = append(rest, args[i])
		}
	}
	if len(rest) > 1 {
		fmt.Sscan(rest[0], &cols)
		fmt.Sscan(rest[1], &rows)
	}
	name := args[0]
	if !strings.HasPrefix(name, "@") && !strings.Contains(name, ".") {
		name = "@" + name
	}

	start := time.Now()
	st, err := app.NewStage(name, "@starling")
	proof.Must(err)
	built := time.Since(start)

	if !png {
		sc := app.Screen{Cols: cols, Rows: rows, FPS: 24, Raw: true}
		proof.Must(sc.Run(st, func(f app.Frame) bool {
			for _, k := range f.Keys {
				if k.Rune == 'q' || k.Rune == 3 {
					return false
				}
			}
			return true
		}))
		return
	}

	if cols == 0 || rows == 0 {
		cols, rows = 160, 48
	}
	base := time.Now()
	frame := func(t float64) [][]cells.Cell {
		d := time.Duration(t * float64(time.Second))
		return st.Draw(app.Frame{At: base.Add(d), Since: d}, cols, rows)
	}
	start = time.Now()
	frame(at)
	first := time.Since(start)
	start = time.Now()
	cs := frame(at + 1.0/24)
	next := time.Since(start)

	// what the terminal has to take: a second of frames at 24 a second,
	// as the differences the loop would send
	wire, prev := 0, cs
	for i := 2; i < 26; i++ {
		c := frame(at + float64(i)/24)
		wire += len(cells.Delta(prev, c))
		prev = c
	}

	stem := strings.TrimSuffix(filepath.Base(strings.TrimPrefix(name, "@")),
		".json")
	out := "out/show-" + stem + ".png"
	proof.Must(os.MkdirAll("out", 0o755))
	proof.Must(proof.WritePNGXY(out, cells.CellsToPixels(cs, 2, 3), 3, 4))
	ms := func(d time.Duration) time.Duration {
		return d.Round(time.Millisecond)
	}
	fmt.Printf("%s at %dx%d: loaded in %v, first frame %v, then %v a "+
		"frame, %s a second at 24 -> %s\n", name, cols, rows, ms(built),
		ms(first), ms(next), kb(wire), out)
}
