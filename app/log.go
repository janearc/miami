package app

import (
	"image"
	"strings"
	"sync"

	"github.com/janearc/libdaffy/canvas"

	"github.com/janearc/miami/cells"
)

// A box that a program talks into.
//
// The other overlays here are type: figlet letterforms, set large, for a word
// somebody chose. That is the wrong instrument for output.
//
// A build says "github.com/janearc/miami/view" and figlet turns it into rows
// of punctuation nobody can read -- which is what "done" in mini looked like.
//
// Output wants glyphs. One character per character, wrapped to a box, and as
// many lines as will fit -- which is a text box, and libdaffy already has one.
//
// What libdaffy's does not do is scroll. Lines() wraps and takes the first
// however-many fit, which is right for a caption somebody typed and wrong for
// a stream: the first thing a build said would sit there while the rest went
// unseen.
//
// So the wrapping is libdaffy's and the tail is here.
type Log struct {
	Row, Col int // negative Row counts from the bottom
	Cols     int // width in cells; the screen less a margin if zero
	Rows     int // how many lines show at once
	Ink      Ramp
	Scrim    float64

	mu    sync.Mutex
	lines []string
}

// how much a log keeps. more than it can show, so a taller box shows more
// history rather than the same view with gaps, and bounded so a build that
// prints for an hour does not grow without end.
const logKeeps = 400

// Say adds a line. Safe from any goroutine, which is the point of it: the work
// is on its own clock and says things when it has something to say.
//
// A line arriving with newlines in it becomes several, because a program
// writing output does not think in lines the way a box does, and one that
// prints a paragraph should not have it laid out as one long line.
func (l *Log) Say(s string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.lines = append(l.lines, strings.Split(strings.TrimRight(s, "\n"), "\n")...)
	if n := len(l.lines) - logKeeps; n > 0 {
		l.lines = append([]string(nil), l.lines[n:]...)
	}
}

// Clear empties it.
func (l *Log) Clear() {
	l.mu.Lock()
	l.lines = nil
	l.mu.Unlock()
}

// Over draws the tail of what it has been told.
func (l *Log) Over(cs [][]cells.Cell, _ Frame) {
	if len(cs) == 0 {
		return
	}
	l.mu.Lock()
	text := strings.Join(l.lines, "\n")
	l.mu.Unlock()
	if text == "" {
		return
	}

	// nought is "as much as there is": a box pinned to a column count is
	// the same box in a window twice the size, which is the half of a
	// scene that does not scale.
	w, h := l.Cols, l.Rows
	if w <= 0 {
		// a share of the window, and not all of it. build output is
		// long -- a go import path is most of eighty columns -- but a
		// box the width of a wide terminal is a wall.
		w = min(len(cs[0])-8, 96)
	}
	if h <= 0 {
		h = max(3, len(cs)/6)
	}
	row, col := l.Row, l.Col
	if row < 0 {
		row += len(cs) - h
	}
	if col < 0 {
		col = (len(cs[0]) - w) / 2
	}

	// wrapped by libdaffy, with the box told it is as tall as it needs to
	// be: its Lines stops at the height it is given, and a box asked for
	// three would wrap only the first three lines and never see the rest.
	box := canvas.TextBox{
		Box:  image.Rect(0, 0, w, logKeeps*4),
		Text: text,
	}
	lines := box.Lines()
	if len(lines) > h {
		lines = lines[len(lines)-h:] // the tail: the newest fills the box
	}

	// darkened over what is written rather than over the box it may one
	// day fill. an empty box with a scrim on it is a black panel laid on
	// the scene, and for the first second of a build that is all it is.
	if l.Scrim > 0 {
		wide := 0
		for _, line := range lines {
			wide = max(wide, len(line))
		}
		Scrim(cs, col-1, row-1, wide+2, len(lines)+2, l.Scrim, 2)
	}
	ink := l.Ink
	if len(ink) == 0 {
		ink = Vaporwave
	}
	for i, line := range lines {
		y := row + i
		if y < 0 || y >= len(cs) {
			continue
		}
		// older lines sit further down the ramp, so the newest reads
		// brightest and the box has a direction to it
		fg := ink.At(float64(i) / float64(max(1, h-1)))
		writeText(cs, y, col, line, fg)
	}
}
