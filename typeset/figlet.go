package typeset

import (
	"github.com/janearc/miami/canvas"
	"github.com/janearc/miami/geom"
	"math"
	"os"
	"strings"
)

// a figlet banner used as a stencil. the file is ink and space; we render the
// ink as geometry at subcell resolution rather than pasting its characters, so
// the wordmark scales with the frame instead of being locked to one size.
type TextMask struct {
	Ink    [][]bool
	Origin geom.Vec // top left, physical units
	CW, CH float64  // physical size of one character cell
}

// LoadInk reads a figlet text file as a mask: which cells carry ink.
func LoadInk(path string) ([][]bool, int, int, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, 0, 0, err
	}
	ink, w, h := ParseInk(b)
	return ink, w, h, nil
}

// ParseInk reads a banner held in memory: non-space is ink.
func ParseInk(b []byte) ([][]bool, int, int) {
	lines := strings.Split(strings.TrimRight(string(b), "\n"), "\n")
	w := 0
	for _, l := range lines {
		if len(l) > w {
			w = len(l)
		}
	}
	ink := make([][]bool, len(lines))
	for i, l := range lines {
		row := make([]bool, w)
		for j := 0; j < len(l); j++ {
			row[j] = l[j] != ' ' && l[j] != '\t'
		}
		ink[i] = row
	}
	return ink, w, len(lines)
}

// Erode upscales the stencil and shaves its boundary. the banner font draws
// stems three cells wide and counters one, so at low resolution the counters
// lose the majority vote and every letter fills in. taking a quarter cell off
// each edge costs the stems little and widens the counters by half a cell.
func Erode(ink [][]bool, scale, iters int) [][]bool {
	h, w := len(ink), len(ink[0])
	up := make([][]bool, h*scale)
	for y := range up {
		row := make([]bool, w*scale)
		for x := range row {
			row[x] = ink[y/scale][x/scale]
		}
		up[y] = row
	}
	at := func(g [][]bool, x, y int) bool {
		if y < 0 || y >= len(g) || x < 0 || x >= len(g[0]) {
			return false
		}
		return g[y][x]
	}
	for i := 0; i < iters; i++ {
		next := make([][]bool, len(up))
		for y := range up {
			row := make([]bool, len(up[0]))
			for x := range row {
				row[x] = at(up, x, y) &&
					at(up, x-1, y) && at(up, x+1, y) &&
					at(up, x, y-1) && at(up, x, y+1)
			}
			next[y] = row
		}
		up = next
	}
	return up
}

// In reports whether a point falls on inked text, the mask fitted to the frame.
func (t TextMask) In(p geom.Vec) bool {
	cx := int(math.Floor((p.X - t.Origin.X) / t.CW))
	cy := int(math.Floor((p.Y - t.Origin.Y) / t.CH))
	if cy < 0 || cy >= len(t.Ink) {
		return false
	}
	row := t.Ink[cy]
	if cx < 0 || cx >= len(row) {
		return false
	}
	return row[cx]
}

// FitText places a banner by the width it should occupy, keeping the 1:2
// character cell a terminal font assumes.
func FitText(ink [][]bool, cols, rows int, cx, bottom, width float64) TextMask {
	cw := width / float64(cols)
	ch := cw * canvas.CellAspect
	h := ch * float64(rows)
	return TextMask{
		Ink:    ink,
		Origin: geom.Vec{X: cx - width/2, Y: bottom - h},
		CW:     cw, CH: ch,
	}
}
