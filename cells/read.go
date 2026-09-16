package cells

import (
	"image/color"
	"strconv"
	"strings"
)

// Reading an ansi file back into cells.
//
// This is here rather than beside the command that wanted it because it is the
// other half of ANSI. A writer without a reader is a claim nobody can check:
// FromANSI(ANSI(cs)) is a round trip, and a round trip is a test.
//
// It is also the first thing an agent needs. An agent cannot see an escape
// sequence -- handed one it will read the bytes, form a confident opinion and
// be wrong, which happened repeatedly while this library was built.
//
// Parsing the escapes back into cells is what makes a picture out of them, and
// a picture is what can be looked at.

// FromANSI reads SGR colour and text into cells.
//
// Cursor movement is not understood on purpose. A file that moves the cursor is
// a stream rather than a picture -- it says what changed since the last frame,
// so a single frame of it means nothing on its own -- and the verb for those is
// `play`.
func FromANSI(s string) [][]Cell {
	var out [][]Cell
	fg, bg := defaultFg, Transparent
	for _, line := range strings.Split(strings.TrimRight(s, "\n"), "\n") {
		var row []Cell
		for i := 0; i < len(line); {
			if line[i] == 0x1b && i+1 < len(line) && line[i+1] == '[' {
				j := i + 2
				for j < len(line) && (line[j] == ';' ||
					(line[j] >= '0' && line[j] <= '9')) {
					j++
				}
				if j < len(line) && line[j] == 'm' {
					fg, bg = applySGR(line[i+2:j], fg, bg)
				}
				i = j + 1
				continue
			}
			r, size := firstRune(line[i:])
			row = append(row, Cell{Glyph: r, Fg: fg, Bg: bg})
			i += size
		}
		out = append(out, row)
	}

	// square it up: anything that draws these indexes by column, and a
	// picture whose rows are different lengths is not a picture yet
	w := 0
	for _, r := range out {
		if len(r) > w {
			w = len(r)
		}
	}
	for i, r := range out {
		for len(r) < w {
			r = append(r, Cell{Glyph: ' ', Bg: Transparent})
		}
		out[i] = r
	}
	return out
}

// defaultFg is what a cell is drawn in before anything has said otherwise, and
// what a reset returns it to. Not white: a terminal's own default foreground is
// off-white, and starting at full white makes an unstyled glyph brighter than
// anything the picture chose.
var defaultFg = color.RGBA{220, 220, 220, 255}

// applySGR reads one escape's parameters and returns the colours after it.
//
// Only the parameters this library writes are understood: reset, default
// background, and the two truecolour forms. Anything else is passed over rather
// than refused, so a file with a bold or an underline in it still reads.
func applySGR(args string, fg, bg color.RGBA) (color.RGBA, color.RGBA) {
	parts := strings.Split(args, ";")
	for k := 0; k < len(parts); k++ {
		switch parts[k] {
		case "", "0":
			fg, bg = defaultFg, Transparent
		case "49":
			bg = Transparent
		case "38", "48":
			if k+4 < len(parts) && parts[k+1] == "2" {
				n := func(s string) uint8 {
					v, _ := strconv.Atoi(s)
					return uint8(v)
				}
				c := color.RGBA{n(parts[k+2]), n(parts[k+3]),
					n(parts[k+4]), 255}
				if parts[k] == "38" {
					fg = c
				} else {
					bg = c
				}
				k += 4
			}
		}
	}
	return fg, bg
}

// firstRune is the first rune of a string, and how many bytes it took.
func firstRune(s string) (rune, int) {
	for _, r := range s {
		return r, len(string(r))
	}
	return ' ', 1
}
