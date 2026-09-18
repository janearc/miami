package app

import "fmt"

// The pointer, as the terminal reports it.
//
// A terminal that has been asked for SGR-pixel reporting -- DEC mode 1016 --
// says where the pointer is in pixels rather than in cells.
//
// A cell is two subpixels across and three down here, so pixels are what make
// the half of a cell knowable: without them a picture drawn at sextant
// resolution is steered at a sixth of its own resolution.
//
// Not every terminal has 1016, and one inside a multiplexer usually does not.
// miami is a library other programs call rather than a program that can refuse
// to start, so a terminal without it is told about in a sentence and reports
// cells, which is the driver working less well rather than not at all.

// MouseKind is what the pointer did.
type MouseKind int

const (
	// Press is a button going down.
	Press MouseKind = iota
	// Release is a button coming up.
	Release
	// Move is the pointer moving, with or without a button held.
	Move
	// Wheel is a wheel notch. Wheel says which way.
	Wheel
)

// String is the kind's name, for a readout and for a test that failed.
func (k MouseKind) String() string {
	switch k {
	case Press:
		return "press"
	case Release:
		return "release"
	case Move:
		return "move"
	case Wheel:
		return "wheel"
	}
	return fmt.Sprintf("kind(%d)", int(k))
}

// Button is which button a report is about.
type Button int

const (
	// Left, Middle and Right are the buttons. None is a move with nothing
	// held, and a wheel notch.
	Left Button = iota
	Middle
	Right
	None
)

// Mouse is one report: where the pointer is, what it did, and how well the
// terminal knows where it is.
type Mouse struct {
	Kind   MouseKind
	Button Button
	// Wheel is minus one away from the reader, plus one toward, and nought
	// for anything that is not a wheel notch.
	Wheel int

	// X and Y are pixels from the top left of the window, and are minus
	// one when the terminal gave only cells. Pixels says which.
	X, Y   int
	Pixels bool

	// Col and Row are the cell under the pointer, counted from nought.
	Col, Row int
	// SubX and SubY are the subpixel of the sheet under the pointer, also
	// from nought: a cell is two across and three down, so SubX is
	// Col*2 or Col*2+1.
	SubX, SubY int

	// Shift, Alt and Ctrl are what was held down with it.
	Shift, Alt, Ctrl bool
}

// Held is whether a button was down for this report, which is what tells a
// drag from the pointer merely passing over.
func (m Mouse) Held() bool {
	return m.Button != None && (m.Kind == Press || m.Kind == Move)
}

// String is the report in a line, for a readout and a failed test.
func (m Mouse) String() string {
	where := fmt.Sprintf("cell %d,%d sub %d,%d", m.Col, m.Row, m.SubX,
		m.SubY)
	if m.Pixels {
		where = fmt.Sprintf("px %d,%d %s", m.X, m.Y, where)
	}
	s := fmt.Sprintf("%s button %d %s", m.Kind, int(m.Button), where)
	if m.Kind == Wheel {
		s = fmt.Sprintf("%s wheel %+d %s", m.Kind, m.Wheel, where)
	}
	for _, mod := range []struct {
		on   bool
		name string
	}{{m.Shift, " shift"}, {m.Alt, " alt"}, {m.Ctrl, " ctrl"}} {
		if mod.on {
			s += mod.name
		}
	}
	return s
}

// Caps is what the terminal turned out to be able to do.
type Caps struct {
	// Pixels is whether DEC mode 1016 answered, so reports carry pixels.
	Pixels bool
	// CellW and CellH are a cell's size in pixels, from CSI 16 t. Nought
	// when the terminal did not say, in which case a report's pixels
	// cannot be turned into a subpixel and the middle of the cell is used.
	CellW, CellH int
}

// Missing says in words what the terminal did not give, and is empty when it
// gave everything. It is a sentence for a person, not a flag for a program.
func (c Caps) Missing() string {
	switch {
	case !c.Pixels && c.CellW == 0:
		return "this terminal reports the pointer by cell, not by" +
			" pixel, and did not say how big a cell is: a drag" +
			" moves a whole cell at a time. Ghostty outside a" +
			" multiplexer gives both."
	case !c.Pixels:
		return "this terminal reports the pointer by cell rather than" +
			" by pixel, so a drag moves a whole cell at a time."
	case c.CellW == 0:
		return "this terminal reports pixels but did not say how big" +
			" a cell is, so the pointer is placed in the middle" +
			" of its cell."
	}
	return ""
}

// String is one line for a probe verb.
func (c Caps) String() string {
	s := "pointer: cells"
	if c.Pixels {
		s = "pointer: pixels (DEC 1016)"
	}
	if c.CellW > 0 {
		return fmt.Sprintf("%s, cell %dx%d px", s, c.CellW, c.CellH)
	}
	return s + ", cell size unknown"
}

// What is written to the terminal to start and stop reporting. Starting is
// Caps.modes, because what is asked for depends on what answered; stopping is
// the same three modes off, in the reverse order, whatever was on.
//
// Off goes in the same place the cursor is put back, signal path included. A
// terminal left in any-motion mode after the program is gone prints an escape
// sequence at the shell for every twitch of the hand, and the person has to
// know to type reset.
const (
	mouseOff = "\x1b[?1016l\x1b[?1006l\x1b[?1003l"
	// askCell is CSI 16 t: how big is a cell, in pixels. The answer is
	// CSI 6 ; height ; width t.
	askCell = "\x1b[16t"
	// askPixels asks whether mode 1016 is on (a mode request, CSI ? Ps
	// $ p). The answer is CSI ? 1016 ; ps $ y, and ps of one or three
	// means it is.
	askPixels = "\x1b[?1016$p"
)

// coalesce keeps every press, release and wheel notch in the order they
// happened, and of a run of moves keeps only the last.
//
// A drag sends samples faster than frames can be drawn, so a frame that acted
// on every move would act on where the hand was and not on where it is.
//
// Only the last move in a run says where it is. The ones before it are where
// it has been, and the picture is about to show that anyway.
//
// Presses, releases and notches are not like that. Every one of them is a
// thing the person did, and two clicks are not one click.
func coalesce(in []Mouse) []Mouse {
	out := make([]Mouse, 0, len(in))
	for i, m := range in {
		if m.Kind == Move && i+1 < len(in) && in[i+1].Kind == Move {
			continue // a later move says where the hand is now
		}
		out = append(out, m)
	}
	return out
}
