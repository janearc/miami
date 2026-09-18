package main

import (
	"fmt"
	"image/color"
	"strings"
	"time"

	"github.com/janearc/miami/cells"
	"github.com/janearc/miami/proof"
)

// A reel is frames written to a file and a few pngs beside them.
//
// Four verbs -- shadow, ball, flock, fly -- draw a moving picture and write it
// out the same way: a header, then every frame as a delta on the one before,
// with a png every so often because an agent cannot see an escape sequence.
//
// That loop was written four times and differed in four small ways, none of
// them on purpose.

// reel is one of those sequences, and how it is written out.
type reel struct {
	name  string  // out/<name>.frames, and out/<name>-<k>.png beside it
	secs  float64 // how long it runs
	fps   int
	shots int      // how many pngs to leave across it
	note  []string // the header, a line at a time
}

// count is how many frames the reel holds.
func (r reel) count() int { return int(r.secs*float64(r.fps) + 0.5) }

// run draws every frame and writes the reel, and answers how many it held.
//
// The delta encoder is checked as it goes: the delta from the last frame is
// replayed onto it and must give this frame back exactly. It is cheap next to
// drawing, and a delta that does not replay is the kind of fault that reaches
// a terminal as a smear rather than an error.
func (r reel) run(draw func(t float64) [][]color.RGBA) int {
	hold := time.Duration(float64(time.Second) / float64(r.fps))
	n := r.count()
	step := 0
	if r.shots > 0 {
		step = n / r.shots
	}
	var out strings.Builder
	for _, line := range r.note {
		fmt.Fprintf(&out, "# %s\n", line)
	}
	var prev [][]cells.Cell
	for i := 0; i < n; i++ {
		cs := cells.ToCells(draw(float64(i)/float64(r.fps)), 2, 3)
		fmt.Fprintf(&out, "\nframe %dms\n", hold.Milliseconds())
		if prev == nil {
			out.WriteString(cells.ANSI(cs))
		} else {
			out.WriteString(cells.Delta(prev, cs))
			if row, col, ok := replayed(prev, cs); !ok {
				proof.Must(fmt.Errorf("frame %d: delta does"+
					" not replay at %d,%d", i, row, col))
			}
		}
		prev = cs
		if step > 0 && i%step == 0 && i/step < r.shots {
			proof.Must(proof.WritePNGXY(
				fmt.Sprintf("out/%s-%d.png", r.name, i/step),
				cells.CellsToPixels(cs, 2, 3), 3, 4))
		}
	}
	proof.Must(write("out/"+r.name+".frames", []byte(out.String())))
	return n
}

// replayed applies the delta from prev and reports whether it gives cs back
// exactly. On a difference it names the first cell that disagrees, because
// "the delta is wrong" is not a thing anyone can act on at four in the
// morning.
func replayed(prev, cs [][]cells.Cell) (int, int, bool) {
	echo := make([][]cells.Cell, len(prev))
	for i := range prev {
		echo[i] = append([]cells.Cell(nil), prev[i]...)
	}
	cells.Apply(echo, cells.Delta(prev, cs))
	for r := range echo {
		for c := range echo[r] {
			if echo[r][c] != cs[r][c] {
				return r, c, false
			}
		}
	}
	return 0, 0, true
}

// ground is a frame of one colour, for a verb that paints no background.
func ground(w, h int, c color.RGBA) [][]color.RGBA {
	px := make([][]color.RGBA, h)
	for y := range px {
		px[y] = make([]color.RGBA, w)
		for x := range px[y] {
			px[y][x] = c
		}
	}
	return px
}

// size reads cols, rows and seconds from a verb's arguments, keeping whatever
// it is not given.
func size(args []string, cols, rows *int, secs *float64) {
	if len(args) > 2 {
		fmt.Sscan(args[0], cols)
		fmt.Sscan(args[1], rows)
		fmt.Sscan(args[2], secs)
	}
}
