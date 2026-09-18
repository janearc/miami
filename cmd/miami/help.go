package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// write is os.WriteFile with the directory made first. out/ is not in the
// repository -- it is every verb's scratch and the readme says so -- so on a
// fresh clone it does not exist, and every verb that wrote to it panicked.
//
// Found by cloning into /tmp and following the readme, which is the only way
// that kind of thing is found.
func write(path string, b []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

// What miami tells an agent about itself.
//
// This exists because the usage line went three sessions out of date while
// twenty verbs were added to it, which is the ordinary fate of a usage line and
// the reason an agent cannot trust one.
//
// So the verb table below is the only place a verb is described, the bare
// command prints it, and a verb that is not in it does not appear anywhere.
//
// The other half is `explain`, and it carries something a feature list does
// not: the refusals. Every trap in it cost this library a rebuild to find -- a
// sprite that the player will not open, a field that clamps where you did not
// expect, a sentinel that collides with a real value.
//
// An agent that reads them does not have to pay for them again, and that is
// most of what "teach agents how to use it" can mean.

type verb struct{ name, args, what string }

var verbs = []verb{
	{"card", "<name>", "a repository card, 75 by 17, colour escapes only"},
	{"emoji", "<hex> [cols] [n] [word] [font]", "an emoji, as cells"},
	{"scene", "<file.json> [cols rows]", "a document, and its go twin"},
	{"fly", "[cols rows secs fps]", "a starling crossing a dusk"},
	{"flock", "[cols rows secs]", "nine birds at nine depths"},
	{"ball", "[cols rows secs]", "a checkered ball, spinning and moving"},
	{"dusk", "[cols rows]", "six samples across the twenty seconds"},
	{"sunset", "[cols rows]", "the vaporwave stack, and each layer"},
	{"sunsetanim", "[cols rows n]", "the same, with its effects running"},
	{"anim", "[dir]", "rebuild every animation from the stencils"},
	{"clock", "[cols rows fps]", "a vaporwave clock, made as it is shown"},
	{"bench", "", "what a size costs, and which ceiling first"},
	{"icon", "[size] [--dir dist]",
		"the quasar as an app icon, dark and light"},
	{"show", "<scene> [cols rows] [--t s] [--png]",
		"scenery, live; or one frame, to a png"},
	{"play", "<file.frames> [loop]", "play a frames file to this terminal"},
	{"check", "<file>", "what a document holds, and what is wrong"},
	{"docs", "[page] [--png|--go]", "the documentation, painted"},
	{"png", "<file.ansi> [out.png]", "an ansi file as a png, to look at"},
	{"sprite", "[cols]", "emit the dolphin stencil"},
	{"bulbsprite", "[cols]", "emit the bulb stencil, cold"},
	{"word", "<text> [cols]", "set a word as a stencil"},
	{"bb", "<cols> <rows>", "the big.birds negative-space piece"},
	{"lights", "", "the roadside sign"},
	{"miami", "[term]", "the composite scene"},
	{"<scene>", "<png|sil|roles|term> [cols]", "macaw, palm, car"},
}

// about is what miami is, in the words the bare command prints.
//
// At the margin rather than indented inside the function: a raw string
// keeps every tab in it, so prose nested two levels deep is eighty columns
// of sentence plus sixteen of indent, and the lint counts both.
const about = `miami draws for terminals: stills, motion, effects and lighting.

It paints with fields and ramps, moves stencils in daffy's eleven-role
alphabet, and composites the two by alpha. Output is cells -- sextant glyphs,
two colours each -- and every verb below writes files rather than needing a
window.

Start with "miami docs" if you are about to write a document, and
"miami check <file>" before you render one.

If you are an agent: you cannot see an escape sequence, so do not try. Every
verb here writes a png beside its output, and "miami docs PAGE --png" and
"miami png FILE.ansi" render anything else. Open the png, look at it, and
change one thing. That loop is how this library was built and it is the only
way to tell whether a picture works.`

// usage prints what miami is and every verb, and is the only place a verb is
// described.
func usage() {
	fmt.Println(about)
	fmt.Println("\nverbs:")
	for _, v := range verbs {
		fmt.Printf("  %-12s %-38s %s\n", v.name, v.args, v.what)
	}
	fmt.Println(`
This command is one caller of the library and not the only one it is for.
Package app draws from another program's event loop -- a build tool wanting a
themed progress bar over a sunset while it compiles -- and "miami docs library"
paints what that gets. example/builder is a working one in forty lines.`)
	os.Exit(2)
}
