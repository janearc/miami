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
	{"emoji", "<hex> [cols] [colours] [word] [font]", "an emoji from the system font, as cells"},
	{"scene", "<file.json> [cols] [rows]", "render a scene document, and diff it against its go twin"},
	{"fly", "[cols rows secs fps]", "a starling crossing a dusk, to out/fly.frames"},
	{"flock", "[cols rows secs]", "nine birds at nine depths, to out/flock.frames"},
	{"ball", "[cols rows secs]", "a checkered ball spinning, scaling and moving"},
	{"dusk", "[cols rows]", "six samples across the twenty seconds"},
	{"sunset", "[cols rows]", "the vaporwave stack, plus every layer on its own"},
	{"sunsetanim", "[cols rows n]", "the same with its effect layers running"},
	{"anim", "[dir]", "rebuild every sprite animation, from the stencils"},
	{"clock", "[cols rows fps]", "a vaporwave clock, generated as it is shown"},
	{"bench", "", "what a size costs, and which ceiling it hits first"},
	{"play", "<file.frames> [loop]", "play a frames file to this terminal"},
	{"check", "<file>", "say what a document holds and what is wrong with it"},
	{"docs", "[page] [--png|--go]", "the documentation, painted; --png so an agent can see it"},
	{"png", "<file.ansi> [out.png]", "render an ansi file to a png, so it can be looked at"},
	{"sprite", "[cols]", "emit the dolphin stencil"},
	{"bulbsprite", "[cols]", "emit the bulb stencil, cold"},
	{"word", "<text> [cols]", "set a word as a stencil"},
	{"bb", "<cols> <rows>", "the big.birds negative-space piece"},
	{"lights", "", "the roadside sign"},
	{"miami", "[term]", "the composite scene"},
	{"<scene>", "<png|sil|roles|term> [cols]", "macaw, palm, car, cassowary: one drawing"},
}

// usage prints what miami is and every verb, and is the only place a verb is
// described.
func usage() {
	fmt.Println(`miami draws for terminals: stills, motion, effects and lighting.

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
way to tell whether a picture works.`)
	fmt.Println("\nverbs:")
	for _, v := range verbs {
		fmt.Printf("  %-12s %-38s %s\n", v.name, v.args, v.what)
	}
	os.Exit(2)
}
