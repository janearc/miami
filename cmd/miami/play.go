package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/janearc/miami/proof"
	"github.com/janearc/miami/scenes"
)

// play plays a frames file to stdout.
//
// Synchronised output, so a frame is presented whole rather than torn, the
// cursor hidden, and the screen put back the way it was found on the way out.
func play(args []string) {
	if len(args) < 1 {
		usage()
	}
	raw, err := os.ReadFile(args[0])
	proof.Must(err)
	loop := len(args) > 1 && args[1] == "loop"
	type frame struct {
		hold time.Duration
		body string
	}
	var frames []frame
	for _, block := range strings.Split(string(raw), "\nframe ")[1:] {
		nl := strings.Index(block, "\n")
		ms := 100
		fmt.Sscanf(block[:nl], "%dms", &ms)
		frames = append(frames, frame{
			time.Duration(ms) * time.Millisecond, block[nl+1:]})
	}
	fmt.Print("\x1b[?25l\x1b[2J")
	defer fmt.Print("\x1b[?25h\x1b[0m\n")
	for {
		for i, f := range frames {
			fmt.Print("\x1b[?2026h")
			if i == 0 {
				fmt.Print("\x1b[H")
			}
			fmt.Print(f.body)
			fmt.Print("\x1b[?2026l")
			time.Sleep(f.hold)
		}
		if !loop {
			return
		}
	}
}

// anim rebuilds every animation from the stencils.
func anim(args []string) {
	dir := "out/anim/"
	if len(args) > 0 {
		dir = args[0]
	}
	proof.Must(os.MkdirAll(dir, 0o755))
	rs, err := scenes.Animations("sprites/made/")
	proof.Must(err)
	lit, err := scenes.Lit("sprites/made/", "sprites/made/starling.css")
	proof.Must(err)
	said, err := scenes.WriteAll(dir, append(rs, lit...))
	proof.Must(err)
	for _, line := range said {
		fmt.Println(line)
	}
}

// asPNG renders an ansi file as a png, to look at. Anything that is not an
// ansi file is not this verb, and says so the way an unknown scene does.
func asPNG(args []string) {
	if len(args) < 1 {
		usage()
	}
	if !strings.HasSuffix(args[0], ".ansi") {
		unknown("png")
	}
	out := "out/" + strings.TrimSuffix(filepath.Base(args[0]), ".ansi") +
		".png"
	if len(args) > 1 {
		out = args[1]
	}
	ansiToPNG(args[0], out)
}
