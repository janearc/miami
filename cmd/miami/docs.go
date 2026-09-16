package main

// The documentation, shown rather than written: one page per idea, painted,
// because this is a library about pictures and the honest way to explain a
// picture is to put it on the screen. The pattern is libtheme's visualdocs,
// and so is the rule that keeps it true -- each page can print the Go that
// painted it, which is its own source file, embedded, so the two cannot drift.
//
// One thing is added for the reader this library expects. An agent cannot see
// an escape sequence: handed one it will describe the bytes and be confident
// and wrong, which happened repeatedly while this was being built. So every
// page renders to a png on request, and so does any ansi file. Look at the
// png, change one thing, look again. That loop is the whole method here and
// nothing else substitutes for it.
//
//	miami docs                every page's name and what it is for
//	miami docs PAGE           painted, to this terminal
//	miami docs PAGE --png     the same, as a png an agent can open
//	miami docs PAGE --go      the go that painted it

import (
	"embed"
	"fmt"
	"image/color"
	"os"
	"strings"

	"miami/cells"
	"miami/proof"
)

//go:embed page_*.go
var pageSources embed.FS

// docPage is one idea: what it is for, the cells that show it, and the prose
// that goes under them. A page that cannot be painted returns no cells and is
// text only; traps is the only one of those and it earns it.
type docPage struct {
	name, what, file string
	draw             func() ([][]cells.Cell, string)
}

var docPages = []docPage{
	{"roles", "the eleven characters a sprite may contain", "page_roles.go", rolesPage},
	{"fields", "the named shapes, painted side by side", "page_fields.go", fieldsPage},
	{"alpha", "how an effect is a second sprite", "page_alpha.go", alphaPage},
	{"viewport", "one sprite asked from three distances", "page_viewport.go", viewportPage},
	{"traps", "the refusals, and what each cost to find", "page_traps.go", trapsPage},
}

const docGround = "\x1b[0m"

func docs(args []string) {
	var name, mode string
	for _, a := range args {
		if strings.HasPrefix(a, "--") {
			mode = a
		} else {
			name = a
		}
	}
	if name == "" {
		fmt.Println("miami docs PAGE [--png|--go]\n\npages:")
		for _, p := range docPages {
			fmt.Printf("  %-10s %s\n", p.name, p.what)
		}
		fmt.Println("\n--png writes out/docs-PAGE.png. Use it: you cannot read an escape\nsequence by looking at the bytes, and guessing at one wastes a rebuild.")
		return
	}
	for _, p := range docPages {
		if p.name != name {
			continue
		}
		if mode == "--go" {
			b, err := pageSources.ReadFile(p.file)
			proof.Must(err)
			fmt.Print(string(b))
			return
		}
		cs, prose := p.draw()
		if cs != nil && mode == "--png" {
			out := "out/docs-" + p.name + ".png"
			proof.Must(proof.SaveImage(out, proof.NewCellImageOn(
				proof.Mono(), 9, 18, color.RGBA{16, 14, 24, 255}).Draw(cs)))
			fmt.Println("wrote", out)
		} else if cs != nil {
			fmt.Print(cells.ANSI(cs))
			fmt.Print(docGround)
		}
		fmt.Println(prose)
		return
	}
	fmt.Printf("no page %q. try: miami docs\n", name)
	os.Exit(2)
}

// swatchRow paints a row of labelled blocks, which most pages want.
func swatchRow(cols, rows int, n int, at func(i int) color.RGBA) [][]cells.Cell {
	px := make([][]color.RGBA, rows*3)
	for y := range px {
		px[y] = make([]color.RGBA, cols*2)
		for x := range px[y] {
			i := x * n / (cols * 2)
			if i >= n {
				i = n - 1
			}
			px[y][x] = at(i)
		}
	}
	return cells.ToCells(px, 2, 3)
}
