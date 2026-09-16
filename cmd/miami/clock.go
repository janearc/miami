package main

import (
	"fmt"
	"image/color"
	"math"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"miami/cells"
	"miami/compose"
	"miami/proof"
	"miami/scenes"
	"miami/typeset"
	"miami/view"
)

// A clock, generated as it is shown.
//
// Everything else here renders to a file and plays it back. This does not:
// each frame is built from the wall clock at the moment it is due and written
// straight out, so there is nothing on disk and nothing to get stale.
//
// That is the shape a stream wants anyway -- frames arrive and are drawn as
// they arrive, and nothing waits for the whole file.
//
// It also happens to exercise most of the library at once: a traced ball over
// a floor with a cast shadow, type set from a figlet font and coloured from a
// ramp, and the two composited by alpha into cells. If one of those is broken
// this is where it shows.
func clock(args []string) {
	cols, rows, fps := 96, 28, 12
	if len(args) > 1 {
		fmt.Sscan(args[0], &cols)
		fmt.Sscan(args[1], &rows)
	}
	if len(args) > 2 && args[2] != "--once" {
		fmt.Sscan(args[2], &fps)
	}
	// one frame and out, so a clock can be looked at. a thing that only
	// exists while it is running cannot be checked, and this one is made of
	// six other things that can each be wrong.
	once := false
	for _, a := range args {
		if a == "--once" {
			once = true
		}
	}

	col, err := compose.Colours("sprites/made/starling.css")
	proof.Must(err)
	fig, err := typeset.LoadFig("small")
	proof.Must(err)

	// vaporwave: the digits run hot pink through magenta into cyan, across
	// the face rather than down it, so the hour and the seconds are
	// different colours and the thing reads as one object moving through a
	// spectrum rather than as eight separate numbers.
	marks := []string{"ff2a9d", "c44fb0", "8a5ad8", "3f7ad6", "2de2e6"}

	// the terminal is put back the way it was found however this ends
	restore := func() { fmt.Print("\x1b[?25h\x1b[0m\n") }
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	go func() { <-sig; restore(); os.Exit(0) }()
	defer restore()

	if once {
		cs := clockFrame(time.Now(), cols, rows, col, fig, marks)
		proof.Must(write("out/clock.ansi", []byte(cells.ANSI(cs))))
		fmt.Fprintln(os.Stderr, "wrote out/clock.ansi")
		return
	}

	fmt.Print("\x1b[?25l\x1b[2J")
	var prev [][]cells.Cell
	tick := time.Duration(float64(time.Second) / float64(fps))
	for {
		start := time.Now()
		cs := clockFrame(start, cols, rows, col, fig, marks)

		fmt.Print("\x1b[?2026h")
		if prev == nil {
			fmt.Print("\x1b[H")
			fmt.Print(cells.ANSI(cs))
		} else {
			fmt.Print(cells.Delta(prev, cs))
		}
		fmt.Print("\x1b[?2026l")
		prev = cs

		if d := tick - time.Since(start); d > 0 {
			time.Sleep(d)
		}
	}
}

// clockFrame is the whole picture at one instant.
func clockFrame(now time.Time, cols, rows int, col map[byte]color.RGBA,
	fig *typeset.FigFont, marks []string) [][]cells.Cell {
	// the ball is driven by the clock rather than by a frame counter, so it
	// is in the same place at the same second whenever this is started
	sec := float64(now.Second()) + float64(now.Nanosecond())/1e9
	minute := sec / 60
	sc := view.Lit{
		Eye: view.Eye{At: view.Vec{Y: 0.05}, Lens: 1.0, Near: 0.05},
		Floor: view.Floor{Y: -0.55, Tile: 0.62, Far: 9.5, A: 'V', B: 'K',
			Shade: map[byte]byte{'V': 'K', 'K': 'D'}},
		Light: view.Vec{X: -1.2, Y: 3.0, Z: 1.6},
		Ball: view.Ball{
			At: view.Vec{
				// once across the floor a minute, and a bounce
				// every four seconds
				X: 1.3 * math.Sin(2*math.Pi*minute),
				Y: -0.21 + 0.66*math.Abs(math.Sin(math.Pi*sec/4)),
				Z: 3.6 - 0.7*math.Cos(2*math.Pi*minute),
			},
			Radius: 0.34,
			Spin:   view.Vec{X: 0.11, Y: 0.43, Z: 0.07},
		},
	}
	f := sc.Trace(time.Duration(sec*float64(time.Second)), cols, rows)

	px := make([][]color.RGBA, rows*3)
	for y := range px {
		px[y] = make([]color.RGBA, cols*2)
		for x := range px[y] {
			px[y][x] = color.RGBA{10, 9, 16, 255}
		}
	}
	compose.Blit(px, f, col, 0, 0)
	cs := cells.ToCells(px, 2, 3)

	// the colons are off for the second half of every second. rendered
	// character by character rather than as one string, because a kerned
	// render gives no way to say which columns a colon occupies, and
	// blanking the wrong ones eats a digit.
	on := now.Nanosecond() < 5e8
	art, spans := setDigits(fig, now.Format("15:04:05"))
	if !on {
		art = blankSpans(art, spans)
	}
	wide := 0
	for _, l := range art {
		if len(l) > wide {
			wide = len(l)
		}
	}
	cells.Letters(cs, art, (cols-wide)/2, 2, func(x, _ int) color.RGBA {
		t := float64(x) / math.Max(1, float64(wide-1)) * float64(len(marks)-1)
		i := int(t)
		if i >= len(marks)-1 {
			return scenes.Hex(marks[len(marks)-1])
		}
		return scenes.Blend(marks[i], marks[i+1], t-float64(i))
	})
	return cs
}

// setDigits lays characters side by side without kerning, and says which
// columns each colon took. Without the spans a colon cannot be turned off:
// the only way to find one in a kerned render is to guess.
func setDigits(f *typeset.FigFont, s string) ([]string, [][2]int) {
	out := make([]string, f.Height)
	var spans [][2]int
	for _, r := range s {
		g := typeset.Trim(f.Render(string(r)))
		w := 0
		for _, l := range g {
			if len(l) > w {
				w = len(l)
			}
		}
		if r == ':' {
			spans = append(spans, [2]int{len(out[0]), len(out[0]) + w})
		}
		for i := range out {
			line := ""
			if i < len(g) {
				line = g[i]
			}
			out[i] += line + strings.Repeat(" ", w-len(line)) + " "
		}
	}
	return out, spans
}

// blankSpans clears the columns a set of characters occupied.
func blankSpans(art []string, spans [][2]int) []string {
	out := make([]string, len(art))
	for i, l := range art {
		b := []byte(l)
		for _, s := range spans {
			for x := s[0]; x < s[1] && x < len(b); x++ {
				b[x] = ' '
			}
		}
		out[i] = string(b)
	}
	return out
}
