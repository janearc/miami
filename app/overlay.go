package app

import (
	"image/color"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/janearc/miami/cells"
	"github.com/janearc/miami/paint"
	"github.com/janearc/miami/typeset"
)

// The things drawn over a scene.
//
// Three of them, because three cover what a program doing something else
// actually wants to say: what it is, how far along it is, and what time it is.
// Each is themed from a ramp rather than from a colour, so a caller picks a
// palette and not a sequence of hex codes.
//
// All of them are safe to set from another goroutine, because that is the whole
// point: the work happens on its own clock and the loop picks up whatever the
// state is when a frame falls due.

// Ramp is a set of hex colours a thing is coloured across.
type Ramp []string

// Vaporwave is the house ramp: hot pink through magenta into cyan.
var Vaporwave = Ramp{"ff2a9d", "c44fb0", "8a5ad8", "3f7ad6", "2de2e6"}

// At is the colour t of the way along, blended in oklab.
func (r Ramp) At(t float64) color.RGBA {
	if len(r) == 0 {
		return color.RGBA{255, 255, 255, 255}
	}
	if len(r) == 1 {
		return paint.Hex(r[0])
	}
	t = math.Max(0, math.Min(1, t)) * float64(len(r)-1)
	i := int(t)
	if i >= len(r)-1 {
		return paint.Hex(r[len(r)-1])
	}
	return paint.Blend(r[i], r[i+1], t-float64(i))
}

// Text is a line of figlet type at a place on the screen.
type Text struct {
	Font   string // a figlet font name; "small" if empty
	Row    int    // cells from the top; negative counts from the bottom
	Col    int    // cells from the left; -1 centres it
	Ink    Ramp
	Scrim  float64 // how far to pull the cells behind it toward dark; 0 is none
	mu     sync.Mutex
	text   string
	cached string
	art    []string
}

// Set changes what it says. Safe from any goroutine.
func (t *Text) Set(s string) {
	t.mu.Lock()
	t.text = s
	t.mu.Unlock()
}

// Over draws it.
func (t *Text) Over(cs [][]cells.Cell, f Frame) {
	t.mu.Lock()
	s := t.text
	t.mu.Unlock()
	if s == "" {
		return
	}
	art := t.render(s)
	drawArt(cs, art, t.Row, t.Col, t.Ink, t.Scrim)
}

// render is the figlet art for a string, kept from last time if the string has
// not changed. Loading a font and setting a line is not expensive, but it is
// the same answer every frame for as long as the text stands still, and a
// clock's text stands still for a second at a time.
func (t *Text) render(s string) []string {
	t.mu.Lock()
	defer t.mu.Unlock()
	if s == t.cached && t.art != nil {
		return t.art
	}
	name := t.Font
	if name == "" {
		name = "small"
	}
	f, err := typeset.LoadFig(name)
	if err != nil {
		return nil
	}
	t.art, t.cached = typeset.Trim(f.Render(s)), s
	return t.art
}

// Digits is a clock face: figlet type whose colons go out for half of every
// second.
//
// Its own overlay rather than a Text with a trick, because turning a colon off
// needs to know which columns it took, and that needs the characters set one at
// a time instead of kerned as a string. A kerned render gives no way to find a
// colon, and blanking the wrong columns eats a digit.
type Digits struct {
	Font    string
	Row     int
	Col     int
	Ink     Ramp
	Scrim   float64
	Format  string // a go time layout; "15:04:05" if empty
	Flicker bool   // colons out for the back half of each second
	At      func() time.Time
}

// Over draws the time.
func (d *Digits) Over(cs [][]cells.Cell, f Frame) {
	now := f.At
	if d.At != nil {
		now = d.At()
	}
	layout := d.Format
	if layout == "" {
		layout = "15:04:05"
	}
	name := d.Font
	if name == "" {
		name = "small"
	}
	fig, err := typeset.LoadFig(name)
	if err != nil {
		return
	}
	art, spans := setSeparately(fig, now.Format(layout), ':')
	if d.Flicker && now.Nanosecond() >= 5e8 {
		art = blankSpans(art, spans)
	}
	drawArt(cs, art, d.Row, d.Col, d.Ink, d.Scrim)
}

// Bar is a progress bar, themed.
//
// This is the one a build tool wants. It is set from wherever the work is, on
// whatever clock the work has, and the loop reads it when a frame falls due --
// miami never learns what the work is.
type Bar struct {
	Row   int
	Col   int // -1 centres it
	Width int // cells; the screen's width less a margin if zero
	Ink   Ramp
	Label string
	Scrim float64

	mu          sync.Mutex
	done, total int
	note        string
}

// Set says how far along the work is. Safe from any goroutine.
func (b *Bar) Set(done, total int) {
	b.mu.Lock()
	b.done, b.total = done, total
	b.mu.Unlock()
}

// Say puts a line of text beside the bar: what is being worked on now.
func (b *Bar) Say(s string) {
	b.mu.Lock()
	b.note = s
	b.mu.Unlock()
}

// Over draws the bar.
func (b *Bar) Over(cs [][]cells.Cell, f Frame) {
	b.mu.Lock()
	done, total, note := b.done, b.total, b.note
	b.mu.Unlock()
	if len(cs) == 0 {
		return
	}
	cols := len(cs[0])
	w := b.Width
	if w == 0 {
		w = cols - 8
	}
	row := b.Row
	if row < 0 {
		row += len(cs)
	}
	col := b.Col
	if col < 0 {
		col = (cols - w) / 2
	}
	if row < 0 || row >= len(cs) {
		return
	}

	frac := 0.0
	if total > 0 {
		frac = math.Max(0, math.Min(1, float64(done)/float64(total)))
	}
	if b.Scrim > 0 {
		Scrim(cs, col, row-1, w, 3, b.Scrim, 2)
	}

	// filled with a half block rather than a full one: a full block in a
	// terminal is a solid slab and reads as a hole punched in the picture,
	// where a half leaves the scene showing through above it
	ink := b.Ink
	if len(ink) == 0 {
		ink = Vaporwave
	}
	for i := 0; i < w; i++ {
		x := col + i
		if x < 0 || x >= cols {
			continue
		}
		t := float64(i) / math.Max(1, float64(w-1))
		if t <= frac {
			cs[row][x] = cells.Cell{Glyph: '▄', Fg: ink.At(t),
				Bg: dim(ink.At(t), 0.22)}
		} else {
			cs[row][x] = cells.Cell{Glyph: '▄', Fg: dim(cs[row][x].Bg, 1.6),
				Bg: cs[row][x].Bg}
		}
	}
	if b.Label != "" || note != "" {
		s := b.Label
		if note != "" {
			s = b.Label + "  " + note
		}
		if total > 0 {
			s += "  " + itoa(int(frac*100)) + "%"
		}
		writeText(cs, row-1, col, s, ink.At(0.15))
	}
}

// drawArt stamps figlet art, placed and themed.
func drawArt(cs [][]cells.Cell, art []string, row, col int, ink Ramp, scrim float64) {
	if len(art) == 0 || len(cs) == 0 {
		return
	}
	wide := 0
	for _, l := range art {
		if len(l) > wide {
			wide = len(l)
		}
	}
	if row < 0 {
		row += len(cs) - len(art)
	}
	if col < 0 {
		col = (len(cs[0]) - wide) / 2
	}
	if scrim > 0 {
		Scrim(cs, col-1, row-1, wide+2, len(art)+2, scrim, 2)
	}
	if len(ink) == 0 {
		ink = Vaporwave
	}
	cells.Letters(cs, art, col, row, func(x, _ int) color.RGBA {
		return ink.At(float64(x) / math.Max(1, float64(wide-1)))
	})
}

// writeText puts plain characters into cells, for a label.
func writeText(cs [][]cells.Cell, row, col int, s string, fg color.RGBA) {
	if row < 0 || row >= len(cs) {
		return
	}
	for i, r := range s {
		x := col + i
		if x < 0 || x >= len(cs[row]) {
			continue
		}
		cs[row][x] = cells.Cell{Glyph: r, Fg: fg, Bg: cs[row][x].Bg}
	}
}

// setSeparately lays characters side by side without kerning, and says which
// columns each of the marked character took.
func setSeparately(f *typeset.FigFont, s string, mark rune) ([]string, [][2]int) {
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
		if r == mark {
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

// itoa is strconv.Itoa, kept local so this file imports nothing for it.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
