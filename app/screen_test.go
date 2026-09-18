package app

import (
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/janearc/miami/cells"
)

// flat is a drawer that fills every cell, so a frame is exactly as tall as
// the screen it is drawn on.
type flat struct{ glyph rune }

func (f flat) Draw(_ Frame, cols, rows int) [][]cells.Cell {
	cs := make([][]cells.Cell, rows)
	for r := range cs {
		cs[r] = make([]cells.Cell, cols)
		for c := range cs[r] {
			cs[r][c] = cells.Cell{Glyph: f.glyph}
		}
	}
	return cs
}

// A frame as tall as the terminal must not scroll it.
//
// cells.ANSI ends every row with a newline, which is right for a file and
// wrong for a screen: the newline after the last row scrolls the terminal by
// one, the top row is pushed off, and because every frame after the first is
// only the cells that changed, it never comes back. opezart found it in a
// 43-row tmux pane, where row 0 was clipped and row 1 held what belonged
// below it.
//
// So the loop places every cell where it goes, first frame included, and the
// bytes it writes carry no newline at all.
func TestAFullHeightFrameDoesNotScrollTheTerminal(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "frame")
	if err != nil {
		t.Fatal(err)
	}
	sc := Screen{Cols: 6, Rows: 4, FPS: 60, Out: f}
	if err := sc.Run(flat{'x'}, func(Frame) bool { return false }); err != nil {
		t.Fatal(err)
	}
	f.Close()
	b, err := os.ReadFile(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	// the only newline written is the one that puts the shell's prompt
	// back on a line of its own, after the picture is finished with
	wrote := string(b)
	if n := strings.Count(wrote, "\n"); n != 1 {
		t.Errorf("a %dx%d frame wrote %d newlines; every one of them"+
			" scrolls a terminal that is exactly that tall",
			sc.Cols, sc.Rows, n)
	}
	if !strings.HasSuffix(wrote, "\n") {
		t.Error("the newline is not the last thing written, so it is" +
			" inside the picture")
	}
	// and it did draw: every row placed where it goes
	for r := 1; r <= sc.Rows; r++ {
		if !strings.Contains(wrote, "\x1b["+strconv.Itoa(r)+";1H") {
			t.Errorf("row %d was never placed", r)
		}
	}
}
