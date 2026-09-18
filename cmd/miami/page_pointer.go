package main

import (
	"github.com/janearc/miami/app"
	"github.com/janearc/miami/cells"
)

// pointer: what a hand on the picture does, and the contact sheet that says
// it still does it.
//
// The picture is the drag itself -- four frames of a scripted orbit of the
// quasar -- because the question this page answers is "can i touch it", and
// the answer is a thing you look at.
func pointerPage() ([][]cells.Cell, string) {
	const cols, rows = 78, 18
	const each = cols / 2
	px := blank(cols*2, rows*3)
	st, err := app.NewStage("@quasar", "@starling")
	if err != nil {
		return cells.ToCells(px, 2, 3), pointerText
	}
	// the same script the --sheet flag runs, four frames wide: press at
	// the left edge, then across, which is half a turn
	st.Point(app.Frame{Mouse: []app.Mouse{{Kind: app.Press,
		Button: app.Left, Row: rows / 2, SubY: rows / 2 * 3}}},
		each, rows)
	for i := range 2 {
		x := i * each
		f := app.Frame{Mouse: []app.Mouse{{Kind: app.Move,
			Button: app.Left, Col: x / 2, Row: rows / 2,
			SubX: x, SubY: rows / 2 * 3}}}
		st.Point(f, each, rows)
		frame := cells.CellsToPixels(st.Draw(f, each, rows), 2, 3)
		for y, row := range frame {
			copy(px[y][i*each*2:], row)
		}
	}
	return cells.ToCells(px, 2, 3), pointerText
}

// pointerText is the page's words.
const pointerText = `
Drag to turn, the wheel to come in and out, and the camera that moves is the
one drawing where the hand is: two viewports side by side are two things to
take hold of.

    sc := app.Screen{Cols: cols, Rows: rows, FPS: 24, Raw: true, Mouse: true}
    sc.Run(st, func(f app.Frame) bool {
        st.Point(f, sc.Cols, sc.Rows)
        return !done
    })

Mouse needs Raw: the reports arrive up the same stdin the keys do. What a
camera turns about is the nearest body straight ahead, so a drag on a scene
holds the thing in it rather than swinging past it.

WHAT THE TERMINAL HAS TO GIVE. Ghostty outside a multiplexer reports the
pointer in pixels (DEC mode 1016) and says how big a cell is, which is what
makes the half of a cell knowable. Without both, reports are by cell and the
pointer sits in the middle of the one it is in: the picture is still
steerable, a drag just moves a whole cell at a time. Frame.Caps says which
you got and Caps.Missing is a sentence to show a person. "miami probe"
prints it for the terminal you are in.

The modes go off where the cursor comes back, signal path included: a
terminal left reporting every motion prints escapes at the shell for every
twitch of the hand.

A frame carries every press, release and wheel notch in order, and of a run
of moves only the last: a frame that acted on every sample of a drag would
act on where the hand was rather than where it is.

    miami show quasar              take hold of it
    miami show quasar --sheet      the same drag as one picture

The picture above is two frames of that sheet, a third of a turn apart: the
disk has come round and the hole has not moved, because the camera is going
round it rather than turning on the spot.`
