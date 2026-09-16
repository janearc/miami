package main

import (
	"image/color"

	"miami/cells"
	"miami/compose"
	"miami/effect"
	"miami/motion"
)

// alpha: the bulb, the effect on its own, and the two composited. Three
// pictures because the claim is about three things, and it is checkable --
// the left one is the stencil, byte for byte, in every lit frame.
func alphaPage() ([][]cells.Cell, string) {
	base, err := motion.ReadSprite("sprites/made/bulb.sprite")
	if err != nil {
		return nil, "no bulb stencil: " + err.Error()
	}
	base = motion.Pad(base, 6, 2, 6, 6)
	L, err := effect.Lightness("sprites/made/starling.css")
	if err != nil {
		return nil, err.Error()
	}
	col, err := compose.Colours("sprites/made/starling.css")
	if err != nil {
		return nil, err.Error()
	}
	glow := effect.Glow(base, effect.GlowOpts{
		Source: "D", Band: 3, Radius: 7, Gain: 1,
		Rays: 10, RayGain: 0.5, RayReach: 0.3, RayInner: 1.3})
	lit := effect.Composite(base, glow, L, 1, 0, 0)

	// three frames side by side, one sprite pixel to one subcell
	w, h := base.Width(), base.Height()
	px := blank(w*3+8, h)
	compose.Blit(px, base, col, 0, 0)
	compose.Blit(px, glow, col, w+4, 0)
	compose.Blit(px, lit, col, 2*w+8, 0)
	return cells.ToCells(px, 2, 3), `
     the stencil            the effect             the two, composited

An effect is a second sprite. It carries no picture at all -- only how much
light is at each pixel -- and a compositor mixes the two. The middle panel is
that alpha, drawn as the sprite it is: it plays on its own, daffy can read it,
and it is in the same eleven characters as everything else.

What makes it an alpha channel is only how the compositor reads it: five
roles standing for five levels of coverage, . D K V W. Five and not a
continuum on purpose -- banding is the look at this size, and quantising
bounds what the compositor can produce, so a lit frame is still a stencil and
still takes any colourway.

The mixing is in oklab lightness, never in the role characters. A pixel's lit
lightness is its own lifted toward white by the alpha, and the output role is
whichever the colourway binds nearest to that. A role is never made darker,
because light does not subtract.

The stencil is not edited. Not one character, in any frame: the unlit frame
is the file it came from and that is asserted, not asserted about. Which is
also why the same halo runs over set type, over a photograph, over anything
carrying the role it is told to come out of.`
}

func blank(w, h int) [][]color.RGBA {
	out := make([][]color.RGBA, h)
	for y := range out {
		out[y] = make([]color.RGBA, w)
	}
	return out
}
