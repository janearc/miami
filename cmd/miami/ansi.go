package main

import (
	"fmt"
	"image/color"
	"os"
	"strconv"
	"strings"

	"miami/cells"
	"miami/proof"
)

// Reading an ansi file back, so it can be looked at.
//
// This is the verb that matters most to an agent and it was the last one
// written, which is the wrong way round. An agent cannot see an escape
// sequence:
//
// handed one it will read the bytes, form a confident opinion, and be wrong --
// that happened repeatedly while this library was built, and once shipped a
// card whose wordmark was punctuation because the render was judged from a
// description instead of a picture.
//
// So: parse the escapes back into cells, draw the cells with a real monospace
// face, and write a png. Then look at the png. It is the same loop a person
// does by glancing at the terminal, and it is not optional.
func ansiToPNG(in, out string) {
	b, err := os.ReadFile(in)
	proof.Must(err)
	cs := parseANSI(string(b))
	if len(cs) == 0 {
		fmt.Println("REFUSED: no cells in", in)
		os.Exit(1)
	}
	proof.Must(proof.SaveImage(out, proof.NewCellImageOn(
		proof.Mono(), 9, 18, color.RGBA{16, 14, 24, 255}).Draw(cs)))
	fmt.Printf("%s: %d cols x %d rows -> %s\n", in, len(cs[0]), len(cs), out)
	fmt.Println("open it. do not judge the art from the bytes.")
}

// parseANSI reads SGR colour and text into cells. Cursor movement is not
// understood on purpose: a file that needs it is a stream, not a picture, and
// the verb for those is `play`.
func parseANSI(s string) [][]cells.Cell {
	var out [][]cells.Cell
	fg, bg := color.RGBA{220, 220, 220, 255}, cells.Transparent
	for _, line := range strings.Split(strings.TrimRight(s, "\n"), "\n") {
		var row []cells.Cell
		for i := 0; i < len(line); {
			if line[i] == 0x1b && i+1 < len(line) && line[i+1] == '[' {
				j := i + 2
				for j < len(line) && (line[j] == ';' || (line[j] >= '0' && line[j] <= '9')) {
					j++
				}
				if j < len(line) && line[j] == 'm' {
					fg, bg = applySGR(line[i+2:j], fg, bg)
				}
				i = j + 1
				continue
			}
			r, size := decodeRune(line[i:])
			row = append(row, cells.Cell{Glyph: r, Fg: fg, Bg: bg})
			i += size
		}
		out = append(out, row)
	}
	// square it up: the drawer indexes by column
	w := 0
	for _, r := range out {
		if len(r) > w {
			w = len(r)
		}
	}
	for i, r := range out {
		for len(r) < w {
			r = append(r, cells.Cell{Glyph: ' ', Bg: cells.Transparent})
		}
		out[i] = r
	}
	return out
}

// applySGR reads one escape's parameters and returns the colours after it.
func applySGR(args string, fg, bg color.RGBA) (color.RGBA, color.RGBA) {
	parts := strings.Split(args, ";")
	for k := 0; k < len(parts); k++ {
		switch parts[k] {
		case "", "0":
			fg, bg = color.RGBA{220, 220, 220, 255}, cells.Transparent
		case "49":
			bg = cells.Transparent
		case "38", "48":
			if k+4 < len(parts) && parts[k+1] == "2" {
				n := func(s string) uint8 { v, _ := strconv.Atoi(s); return uint8(v) }
				c := color.RGBA{n(parts[k+2]), n(parts[k+3]), n(parts[k+4]), 255}
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

// decodeRune is the first rune of a string, and how many bytes it took.
func decodeRune(s string) (rune, int) {
	for _, r := range s {
		return r, len(string(r))
	}
	return ' ', 1
}
