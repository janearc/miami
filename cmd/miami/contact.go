package main

import (
	"fmt"
	"image/color"
	"os"
	"time"

	"github.com/janearc/miami/app"
	"github.com/janearc/miami/cells"
	"github.com/janearc/miami/proof"
)

// A drag, as one picture.
//
// A png cannot show a hand turning a scene, and a person cannot be asked to
// take hold of a terminal to find out whether the last change broke it.
//
// So the pointer is driven from a script -- press, then a dozen moves -- and
// every frame is laid out in a grid: one picture, which an agent can look at
// and a person can glance at.
//
// The reports are the ones the parser would have made of what a terminal
// sends, so what is exercised is the drag, not a shortcut into the camera.

// contact draws a scripted orbit as a contact sheet.
//
//	miami show scene --sheet [cols rows]
//
// Each frame is the same scene a little further round, and the last is half a
// turn from the first, because a drag across the picture is half a turn.
func contact(st *app.Stage, name string, cols, rows int) {
	if cols == 0 || rows == 0 {
		cols, rows = 60, 18
	}
	const across, down = 4, 3
	const frames = across * down
	// the hand starts in the middle and crosses the whole picture, which
	// is half a turn, in frames-1 equal steps
	mid := rows / 2
	step := cols * 2 / (frames - 1)
	base := time.Now()
	var shots [][][]color.RGBA
	for i := range frames {
		var mice []app.Mouse
		switch {
		case i == 0:
			mice = []app.Mouse{{Kind: app.Press, Button: app.Left,
				Col: 0, Row: mid, SubX: 0, SubY: mid * 3}}
		default:
			x := i * step
			mice = []app.Mouse{{Kind: app.Move, Button: app.Left,
				Col: x / 2, Row: mid, SubX: x, SubY: mid * 3}}
		}
		f := app.Frame{At: base, Since: 0, Mouse: mice}
		st.Point(f, cols, rows)
		shots = append(shots, cells.CellsToPixels(
			st.Draw(f, cols, rows), 2, 3))
	}

	w, h := cols*2, rows*3
	const gap = 2
	sheet := make([][]color.RGBA, down*(h+gap)-gap)
	for y := range sheet {
		sheet[y] = make([]color.RGBA, across*(w+gap)-gap)
		for x := range sheet[y] {
			sheet[y][x] = color.RGBA{24, 22, 32, 255}
		}
	}
	for i, px := range shots {
		x0, y0 := (i%across)*(w+gap), (i/across)*(h+gap)
		for y, row := range px {
			copy(sheet[y0+y][x0:], row)
		}
	}
	out := "out/drag-" + name + ".png"
	proof.Must(os.MkdirAll("out", 0o755))
	proof.Must(proof.WritePNGXY(out, sheet, 2, 2))
	fmt.Printf("%d frames of a drag across %s, %dx%d cells each -> %s\n",
		frames, name, cols, rows, out)
}
