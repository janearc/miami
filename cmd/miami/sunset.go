package main

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/janearc/miami/cells"
	"github.com/janearc/miami/grade"
	"github.com/janearc/miami/proof"
	"github.com/janearc/miami/scene"
	"github.com/janearc/miami/scenes"
)

// The vaporwave stack: as one still with every layer beside it, as an
// animation with its effects running, as six samples across its twenty
// seconds, and as a document checked against the go scene it replaced.

// sunset paints the stack once, and each of its layers on its own.
func sunset(args []string) {
	cols, rows := 100, 30
	if len(args) > 1 {
		fmt.Sscan(args[0], &cols)
		fmt.Sscan(args[1], &rows)
	}
	sh := grade.Sextants(cols, rows)
	px := sh.Paint(scenes.Sunset(sh, 0.52, 0))
	cs := cells.ToCells(px, 2, 3)
	proof.Must(write("out/sunset.ansi", []byte(cells.ANSI(cs))))
	proof.Must(proof.WritePNG("out/sunset-px.png", px, 4))
	proof.Must(proof.WritePNG("out/sunset.png",
		cells.CellsToPixels(cs, 2, 3), 4))
	cols2, als, names := sh.Contact(scenes.Sunset(sh, 0.52, 0))
	for i, n := range names {
		proof.Must(proof.WritePNG("out/sunset-l-"+n+".png",
			cols2[i], 2))
		proof.Must(proof.WritePNG("out/sunset-a-"+n+".png",
			als[i], 2))
	}
	fmt.Println("wrote out/sunset.ansi and out/sunset*.png; layers:", names)
}

// sunsetAnim is the same stack with its effects running, as frames.
func sunsetAnim(args []string) {
	cols, rows, n := 100, 30, 16
	if len(args) > 2 {
		fmt.Sscan(args[0], &cols)
		fmt.Sscan(args[1], &rows)
		fmt.Sscan(args[2], &n)
	}
	sh := grade.Sextants(cols, rows)
	var frames []string
	for i := 0; i < n; i++ {
		ph := float64(i) / float64(n)
		st := append(scenes.Sunset(sh, 0.52, ph),
			scenes.Effects(sh, 0.52, ph)...)
		cs := cells.ToCells(sh.Paint(st), 2, 3)
		frames = append(frames, cells.ANSI(cs))
		proof.Must(proof.WritePNG(fmt.Sprintf("out/anim-%02d.png", i),
			cells.CellsToPixels(cs, 2, 3), 2))
	}
	proof.Must(write("out/sunset-frames.ansi",
		[]byte(strings.Join(frames, "\x1b[H"))))
	fmt.Println("wrote", n, "frames")
}

// dusk is six samples across the twenty seconds.
func dusk(args []string) {
	cols, rows := 100, 30
	if len(args) > 1 {
		fmt.Sscan(args[0], &cols)
		fmt.Sscan(args[1], &rows)
	}
	sh := grade.Sextants(cols, rows)
	for i, t := range []float64{0, 4, 8, 12, 16, 20} {
		cs := cells.ToCells(sh.Paint(scenes.Dusk(sh, t, 20)), 2, 3)
		proof.Must(proof.WritePNGXY(fmt.Sprintf("out/dusk-%d.png", i),
			cells.CellsToPixels(cs, 2, 3), 3, 4))
	}
	fmt.Println("wrote six dusk samples")
}

// document renders a json scene, and if a go scene of the same name exists,
// compares them: the only proof a document expresses a scene is that it paints
// the same pixels as the code it replaced.
func document(args []string) {
	if len(args) < 1 {
		usage()
	}
	doc, err := scene.Load(args[0])
	proof.Must(err)
	cols, rows := 100, 30
	if len(args) > 2 {
		fmt.Sscan(args[1], &cols)
		fmt.Sscan(args[2], &rows)
	}
	sh := grade.Sextants(cols, rows)
	same := true
	for i, t := range []float64{0, 4, 8, 12, 16, 20} {
		st, err := doc.At(t)
		proof.Must(err)
		got := sh.Paint(st)
		cs := cells.ToCells(got, 2, 3)
		proof.Must(proof.WritePNGXY(fmt.Sprintf("out/scene-%d.png", i),
			cells.CellsToPixels(cs, 2, 3), 3, 4))
		want := sh.Paint(scenes.Dusk(sh, t, doc.Seconds))
		if x, y, ok := firstDiff(want, got); !ok {
			fmt.Printf("t=%.0f differs first at %d,%d:"+
				" go %v json %v\n", t, x, y, want[y][x],
				got[y][x])
			same = false
		}
	}
	if same {
		fmt.Println("json scene paints the same pixels" +
			" as the go scene, at every sample")
	}
}

// firstDiff is where two paintings first disagree, and true if they do not.
func firstDiff(want, got [][]color.RGBA) (int, int, bool) {
	for y := range want {
		for x := range want[y] {
			if want[y][x] != got[y][x] {
				return x, y, false
			}
		}
	}
	return 0, 0, true
}
