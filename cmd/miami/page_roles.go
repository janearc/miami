package main

import (
	"image/color"

	"miami/cells"
	"miami/compose"
)

// roles: the alphabet, painted in a colourway so the names have colours
// attached rather than descriptions.
func rolesPage() ([][]cells.Cell, string) {
	const alphabet = ".KWVDBYROEX"
	col, err := compose.Colours("sprites/made/starling.css")
	if err != nil {
		return nil, "no colourway on the path: " + err.Error()
	}
	cs := swatchRow(66, 3, len(alphabet), func(i int) color.RGBA {
		return col[alphabet[i]]
	})
	return cs, `
  .     K     W     V     D     B     Y     R     O     E     X
 none  dark light wing stripe base  band   tip  feet pupil  ring

The alphabet. Eleven characters, and nothing else may appear in an art block:
a character outside it is refused by name rather than guessed at. A sprite
holds roles, never colours, so one stencil takes any colourway there is,
including ones written after it. The row above is one colourway's answer;
another would repaint it entirely without the sprite changing a byte.

Address an effect by role and never by coordinate. An effect written against
roles transfers to any stencil carrying them; one written against pixels
transfers to nothing. A role an object does not carry is a no-op and says so
on stderr -- not an error, or a library of effects is unusable across a mixed
set of objects, and not silent, or a misspelled role looks like a subtle
effect.

Two roles that look alike in most sheets: beak-base and beak-band are both
gold. A checkered ball drawn in that pair came out a flat orange patch with
no check in it at all. Check contrast within a pair, not just between them.`
}
