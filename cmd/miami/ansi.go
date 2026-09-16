package main

import (
	"fmt"
	"image/color"
	"os"

	"miami/cells"
	"miami/proof"
)

// Turning an ansi file into a picture, so it can be looked at.
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
	cs := cells.FromANSI(string(b))
	if len(cs) == 0 {
		fmt.Println("REFUSED: no cells in", in)
		os.Exit(1)
	}
	proof.Must(proof.SaveImage(out, proof.NewCellImageOn(
		proof.Mono(), 9, 18, color.RGBA{16, 14, 24, 255}).Draw(cs)))
	fmt.Printf("%s: %d cols x %d rows -> %s\n", in, len(cs[0]), len(cs), out)
	fmt.Println("open it. do not judge the art from the bytes.")
}
