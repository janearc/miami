package app

import (
	"image/color"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/janearc/miami/cells"
	"github.com/janearc/miami/emoji"
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

// Faces are the figlet faces this library picks between when a face is not
// named, smallest first, with how wide each sets a clock.
//
// Chosen by reading them, which is the third time that lesson has been paid
// for here. Chunky is the widest face in the set and is not in it: its digits
// are bars with gaps between, and they come apart at the size they are drawn.
//
// Big is the large one, six rows rather than four. It is large by being
// taller, which is what large means.
//
// A figlet glyph is made of characters and a character is one cell, so type
// does not scale with the window the way the scene does: the same face in a
// terminal twice as wide is half the size relative to everything around it.
// Choosing a bigger face is the only scaling type has.
var Faces = []struct {
	Name  string
	Clock int // columns "00:00:00" takes in it
}{
	{"mini", 20}, {"small", 38}, {"smslant", 42},
	{"standard", 48}, {"big", 48},
}

// faceFor is the largest face whose clock fits in half the width.
//
// Half rather than a third, which was the first try and put mini on an
// ordinary eighty-column terminal.
//
// Mini's digits are slashes and underscores, and it is the face this library
// has been caught using once already. A rule that picks it on a normal window
// is worse than one that sets type a little large.
func faceFor(cols int) string {
	name := Faces[0].Name
	for _, f := range Faces {
		if f.Clock <= cols/2 {
			name = f.Name
		}
	}
	return name
}

// Text is a line of figlet type at a place on the screen.
type Text struct {
	Font   string // a figlet font name; "small" if empty
	Row    int    // cells from the top; negative counts from the bottom
	Col    int    // cells from the left; -1 centres it
	Ink    Ramp
	Scrim  float64 // how far to darken the cells behind it; 0 is none
	mu     sync.Mutex
	text   string
	cached string
	wide   int // the width it was last set for; a resize resets it
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
	if s == "" || len(cs) == 0 {
		return
	}
	art := t.render(s, len(cs[0]))
	drawArt(cs, art, t.Row, t.Col, t.Ink, t.Scrim)
}

// render is the figlet art for a string, kept from last time if the string has
// not changed. Loading a font and setting a line is not expensive, but it is
// the same answer every frame for as long as the text stands still, and a
// clock's text stands still for a second at a time.
func (t *Text) render(s string, cols int) []string {
	t.mu.Lock()
	defer t.mu.Unlock()
	if s == t.cached && t.art != nil && cols == t.wide {
		return t.art
	}
	t.wide = cols
	name := t.Font
	if name == "" || name == "auto" {
		name = faceFor(cols)
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
	if name == "" || name == "auto" {
		name = faceFor(len(cs[0]))
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

	// From drives the bar from a function instead of from Set: a document
	// binding, or anything else a caller wants to read per frame. A bar
	// with one ignores Set, because two sources for one number is how a
	// bar ends up showing neither.
	From func(Frame) float64

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
	switch {
	case b.From != nil:
		frac = math.Max(0, math.Min(1, b.From(f)))
	case total > 0:
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
			bg := cs[row][x].Bg
			cs[row][x] = cells.Cell{Glyph: '▄', Fg: dim(bg, 1.6),
				Bg: bg}
		}
	}
	if b.Label != "" || note != "" {
		s := b.Label
		if note != "" {
			s = b.Label + "  " + note
		}
		if total > 0 || b.From != nil {
			s += "  " + itoa(int(frac*100)) + "%"
		}
		writeText(cs, row-1, col, s, ink.At(0.15))
	}
}

// drawArt stamps figlet art, placed and themed.
func drawArt(
	cs [][]cells.Cell, art []string, row, col int, ink Ramp, scrim float64,
) {
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
func setSeparately(f *typeset.FigFont, s string, mark rune) (
	[]string, [][2]int,
) {
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
			spans = append(
				spans, [2]int{len(out[0]), len(out[0]) + w},
			)
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

// Emoji is a picture from the system's emoji font, placed on the scene.
//
// A shape because it is the most convenient picture there is: a whole drawing
// per codepoint, already anti-aliased by somebody who could draw, in a file
// every machine already has. Nothing else in this library gives that much
// picture for one field of a document.
//
// It is drawn once and kept. An emoji is a fixed image and re-quantising it
// sixty times a second would be the most expensive thing in the frame, for a
// picture that never changes.
type Emoji struct {
	Rune    rune
	Cols    int // cells wide; twenty if unset
	Colours int // how many colours to reduce it to; twelve if unset
	Row     int // negative counts from the bottom
	Col     int // -1 centres it

	once sync.Once
	art  [][]cells.Cell
}

// Over draws it, leaving the scene showing through wherever the emoji is
// transparent -- which is most of the square, for most emoji.
func (e *Emoji) Over(cs [][]cells.Cell, _ Frame) {
	e.once.Do(func() {
		art, err := emoji.Cells(e.Rune, e.Cols, e.Colours)
		if err != nil {
			return // no emoji font, or no such glyph: draw the rest
		}
		e.art = art
	})
	if len(e.art) == 0 || len(cs) == 0 {
		return
	}
	row, col := e.Row, e.Col
	if row < 0 {
		row += len(cs) - len(e.art)
	}
	if col < 0 {
		col = (len(cs[0]) - len(e.art[0])) / 2
	}
	for y, line := range e.art {
		for x, c := range line {
			ty, tx := row+y, col+x
			outside := ty < 0 || ty >= len(cs) || tx < 0 ||
				tx >= len(cs[ty])
			if outside {
				continue
			}
			if c.Bg == cells.Transparent && c.Glyph == ' ' {
				continue // nothing here: leave the scene alone
			}
			if c.Bg == cells.Transparent {
				c.Bg = cs[ty][tx].Bg
			}
			cs[ty][tx] = c
		}
	}
}
