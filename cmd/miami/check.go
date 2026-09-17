package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	sprite "github.com/janearc/libdaffy-sprite"

	"github.com/janearc/miami/effect"
	"github.com/janearc/miami/motion"
	"github.com/janearc/miami/scene"
)

// check: what a document holds, and what is wrong with it.
//
// This is the verb to call before committing to a render, and it is why a role
// that matches nothing is a report rather than a crash. It is also
// deterministic -- same file, same output -- so it can be diffed, tested, and
// put behind an agent that will run it two hundred times without watching.
func checkFile(path string) {
	if strings.HasSuffix(path, ".json") {
		// a scene is known by its top-level keys, not by the words in
		// it. searching the text sent a sprite document that mentioned
		// layers in a comment to the scene reader, and a scene made
		// only of shapes to the sprite reader.
		if b, err := os.ReadFile(path); err == nil && scene.IsScene(b) {
			checkScene(path)
			return
		}
		frames, err := motion.ReadDoc(path)
		if err != nil {
			fmt.Println("REFUSED:", err)
			os.Exit(1)
		}
		reportFrames(path, frames)
		return
	}
	frames, err := motion.ReadStream(path)
	if err != nil {
		f, err2 := motion.ReadSprite(path)
		if err2 != nil {
			fmt.Println("REFUSED:", err)
			os.Exit(1)
		}
		frames = []sprite.Frame{f}
	}
	reportFrames(path, frames)
}

// checkScene reports a scene document: its layers, and each layer's gain at
// three moments, which is where a gain written as nought shows up.
func checkScene(path string) {
	d, err := scene.Load(path)
	if err != nil {
		fmt.Println("REFUSED:", err)
		os.Exit(1)
	}
	fmt.Printf("scene %q: %g seconds, %d layers, %d shapes\n",
		d.Name, d.Seconds, len(d.Layers), len(d.Shapes))
	for _, sh := range d.Shapes {
		name := sh.Name
		if name == "" {
			name = "--"
		}
		what := ""
		if sh.Texture != nil {
			what = " in " + sh.Texture.Kind
		}
		fmt.Printf("  %-9s %-8s%s\n", sh.Kind, name, what)
	}
	// the names it reads, because a name nothing supplies is what a stage
	// refuses, and @beat and friends need a playhead attached to draw
	if refs := d.Refs(); len(refs) > 0 {
		fmt.Printf("  reads: @%s\n", strings.Join(refs, " @"))
	}
	for _, at := range []float64{0, d.Seconds / 2, d.Seconds} {
		st, err := d.At(at)
		if err != nil {
			fmt.Printf("  at %gs: REFUSED: %v\n", at, err)
			os.Exit(1)
		}
		fmt.Printf("  %5.1fs  ", at)
		for _, l := range st {
			fmt.Printf("%s=%.2f ", l.Name, l.Gain)
		}
		fmt.Println()
	}
	// a layer that is never present at any sample is almost always a gain
	// written as nought meaning "unset", which is the trap this library
	// already fell into once
	for _, l := range d.Layers {
		on := false
		for _, at := range []float64{0, d.Seconds / 2, d.Seconds} {
			st, _ := d.At(at)
			for _, x := range st {
				if x.Name == l.Name && x.Gain > 0 {
					on = true
				}
			}
		}
		if !on {
			fmt.Printf("  WARN layer %q has gain 0 at every"+
				" sample and will never appear\n", l.Name)
		}
		if len(l.Ramp) == 0 {
			fmt.Printf("  WARN layer %q has no ramp stops"+
				" and will paint black\n", l.Name)
		}
	}
	fmt.Println("ok -- render it with: miami scene", path)
}

// reportFrames reports a stencil or a stream: its size, which roles it carries,
// and which of the eleven it does not.
func reportFrames(path string, frames []sprite.Frame) {
	if len(frames) == 0 {
		fmt.Println("REFUSED: no frames")
		os.Exit(1)
	}
	var total int64
	for _, f := range frames {
		total += f.Hold.Milliseconds()
	}
	fmt.Printf("%s: %d frames, %d by %d pixels, %dms\n", path, len(frames),
		frames[0].Width(), frames[0].Height(), total)

	used := map[byte]int{}
	for _, f := range frames {
		for _, row := range f.Art {
			for i := 0; i < len(row); i++ {
				used[row[i]]++
			}
		}
	}
	const alphabet = ".KWVDBYROEX"
	var have, missing []string
	for i := 0; i < len(alphabet); i++ {
		c := alphabet[i]
		if used[c] > 0 {
			have = append(have, fmt.Sprintf("%c:%d", c, used[c]))
		} else {
			missing = append(missing, string(c))
		}
		delete(used, c)
	}
	fmt.Println("  roles:", strings.Join(have, " "))
	if len(missing) > 0 {
		fmt.Println("  unused:", strings.Join(missing, ""),
			"-- an effect naming one of these matches nothing"+
				" and says so")
	}
	if len(used) > 0 {
		var bad []string
		for c := range used {
			bad = append(bad, fmt.Sprintf("%q", string(c)))
		}
		sort.Strings(bad)
		fmt.Println(
			"  REFUSED: outside the alphabet:",
			strings.Join(bad, " "),
		)
		os.Exit(1)
	}
	w, h := frames[0].Width(), frames[0].Height()
	for i, f := range frames {
		if f.Width() == w && f.Height() == h {
			continue
		}
		fmt.Printf("  REFUSED: frame %d is %dx%d, frame 0 is %dx%d\n",
			i, f.Width(), f.Height(), w, h)
		os.Exit(1)
	}
	if _, err := effect.Lightness("sprites/made/starling.css"); err == nil {
		fmt.Println("  play:",
			"bin/bandersnatch play --colourway made/starling.css",
			strings.TrimPrefix(path, "sprites/"))
	}
	fmt.Println("  look: render it to a png before you judge it")
	fmt.Println("ok")
}
