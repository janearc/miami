package main

import "miami/cells"

// traps: the only page with nothing to paint, because a refusal has no picture.
// Every one of these cost a rebuild to find.
func trapsPage() ([][]cells.Cell, string) {
	return nil, `miami: the refusals

A .sprite is not a stream. The stream grammar wants a "frame" line before its
art and a .sprite has none, so daffy's reader refuses one with "art before any
frame line" -- the same refusal that stops the player opening a .sprite
directly. motion.ReadSprite supplies the missing line.

A colourway is not optional. The built-in sheet binds dark at the ground's own
lightness, and a colour within about 0.18 oklab lightness of the background is
not dark, it is absent: a mostly-dark bird plays as a floating bill and feet.

A gain of nought means nought. It is not "unset". A sentinel that collides
with a legitimate value is not a sentinel, and a glow whose gain is computed
reaches nought exactly when it should vanish -- and came back at full strength.

A transparent background is not black. ANSI writes a zero background as a
literal black, invisible on a dark card and an opaque slab over a gradient.
Keep the background the cell already had, or write 49.

A whole-pixel translation is lossless; nothing else is. Any warp that is not
an exact integer shift runs the stencil through the resampler. Smooth masses
survive it, a dither does not: a dense speckle turns to mud and pops against
an unwarped frame.

Deltas are not always smaller. A wingbeat writes at 59 per cent of full size.
A spinning checkered ball writes at 131 per cent, because a rotating pattern
changes nearly every pixel in short runs and each costs a wrapper heavier than
the character it saves. Choose per frame, not per file.

Do not average colour in flat art. A cell holding three palette colours gets
the mean of two of them; that mean is in none of its subpixels and in no other
cell, and along an edge those invented in-betweens are the fuzz.

Reduce the palette before sampling, not after, and rank it by clusters rather
than by colours. Sampling first lets a source's anti-aliased edges into the
palette. Ranking raw colours lets a face in nineteen hundred shades of orange
outrank a brow drawn in five hundred near-identical browns, and the dark
family gets no entry at all.

Set a wordmark as figlet letterforms placed as characters, never as type
rasterised into the block fitter. The fitter approximates a shape with two
colours; a letter is not a shape to be approximated, and at six subpixels of
cap height an N comes out as an H. Then pick the font for legibility and move
the art to make room -- picking the one that fits the gap is optimising the
wrong thing, and it shipped a card whose wordmark was punctuation.

And do not judge a picture from its bytes. Render it and look: "miami png
FILE.ansi", or --png on any page here.`
}
