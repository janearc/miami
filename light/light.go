// Package light is a frame as light, before it is colours a terminal can show.
//
// A grade stack answers one question per point: what colour is here. That is
// enough for a sky, and it is not enough for anything that glows.
//
// A spark lands on a few subpixels and has to be added to what is there. A
// glow is light that spread from somewhere else, so no point can work it out
// from where it is. And a disk hotter than white has to stay hotter than white
// until the very end, or its glow comes out the colour of paper.
//
// So a frame that glows is drawn into this first: three numbers a subpixel, in
// linear light, with no ceiling. Things are added to it, spread over it, and
// only then brought down to sRGB:
//
//	b := light.New(sh.W, sh.H)
//	b.Paint(sh, stack)          whatever the document painted
//	disk.Draw(b, eye, t)        a black hole, a planet
//	sparks.Draw(b, eye, t)      particles, added
//	b.Bloom(0.8, light.Glow{Radius: 3, Strength: 0.6})
//	px := b.RGBA(1)             the cell fitter's input, as before
//
// Primaries are sRGB's, in linear light, so one is the display's white. The
// numbers are float32: a frame at full screen is a hundred thousand subpixels,
// and half the memory is a quarter of the time spent waiting for it.
package light

import (
	"math"
	"sync/atomic"

	"github.com/janearc/libtheme-css/primitives/functions"
	"github.com/janearc/libtheme-css/primitives/swatch"
	"github.com/janearc/libtheme-css/spaces/srgb"

	"github.com/janearc/miami/grade"
	"github.com/janearc/miami/internal/split"
)

// RGB is an amount of light in sRGB's primaries. It has no ceiling: a value
// above one is brighter than the display can show, and says so only when the
// frame is brought down.
type RGB struct{ R, G, B float64 }

// Scale is the light made brighter or dimmer.
func (c RGB) Scale(k float64) RGB { return RGB{c.R * k, c.G * k, c.B * k} }

// Plus is two lights together.
func (c RGB) Plus(o RGB) RGB { return RGB{c.R + o.R, c.G + o.G, c.B + o.B} }

// Mix is a paint-like blend: t of o over c.
func (c RGB) Mix(o RGB, t float64) RGB {
	return RGB{c.R + (o.R-c.R)*t, c.G + (o.G-c.G)*t, c.B + (o.B-c.B)*t}
}

// Y is the light's luminance.
func (c RGB) Y() float64 { return lumR*c.R + lumG*c.G + lumB*c.B }

// The rows of the matrix from XYZ to these primaries, and the luminance each
// primary carries. Both are read off libtheme's own sRGB rather than typed in:
// its three lamps, each at full, stored back as swatches, are the columns of
// the forward matrix, and the inverse is worked out once here.
var toRGB, lumR, lumG, lumB = primaries()

// primaries works out the matrix and the luminances, once.
func primaries() (m [3][3]float64, r, g, b float64) {
	var fwd [3][3]float64
	for j, lamp := range []srgb.RGB{{R: 1}, {G: 1}, {B: 1}} {
		x, y, z := lamp.Swatch().XYZ()
		fwd[0][j], fwd[1][j], fwd[2][j] = x, y, z
	}
	return invert(fwd), fwd[1][0], fwd[1][1], fwd[1][2]
}

// invert is a three by three matrix's inverse, by cofactors.
func invert(m [3][3]float64) [3][3]float64 {
	c := func(r0, c0, r1, c1 int) float64 {
		return m[r0][c0]*m[r1][c1] - m[r0][c1]*m[r1][c0]
	}
	d := m[0][0]*c(1, 1, 2, 2) - m[0][1]*c(1, 0, 2, 2) +
		m[0][2]*c(1, 0, 2, 1)
	return [3][3]float64{
		{c(1, 1, 2, 2) / d, -c(0, 1, 2, 2) / d, c(0, 1, 1, 2) / d},
		{-c(1, 0, 2, 2) / d, c(0, 0, 2, 2) / d, -c(0, 0, 1, 2) / d},
		{c(1, 0, 2, 1) / d, -c(0, 0, 2, 1) / d, c(0, 0, 1, 1) / d},
	}
}

// FromSwatch is a physical colour as light in these primaries, unclipped.
func FromSwatch(s swatch.Swatch) RGB {
	x, y, z := s.XYZ()
	m := toRGB
	return RGB{m[0][0]*x + m[0][1]*y + m[0][2]*z,
		m[1][0]*x + m[1][1]*y + m[1][2]*z,
		m[2][0]*x + m[2][1]*y + m[2][2]*z}
}

// Hex is a hex colour, six digits and no hash, as light.
func Hex(s string) RGB { return FromSwatch(grade.Swatch(s)) }

// Kelvin is the colour of a black body at a temperature, at a luminance of
// one. A star's colour and a disk's heat are both this, and neither is typed
// in: libtheme integrates Planck's law against the observer.
//
// Answers already worked out are kept, because the integration is expensive
// and a sky asks for the same star's temperature once for every ray that
// lands near it. A profile of a black hole drawn for a camera that had moved
// put a third of the whole frame inside Planck's law.
//
// What is kept is the answer to the exact temperature asked for, so this is a
// speed change and not a picture change: every artefact the gate compares is
// the same bytes with it as without.
func Kelvin(k float64) RGB {
	slot := &kelvins[kelvinSlot(k)]
	if got := slot.Load(); got != nil && got.k == k {
		return got.c
	}
	c := FromSwatch(swatch.Planckian(k))
	if y := c.Y(); y > 0 {
		c = c.Scale(1 / y)
	} else {
		c = RGB{}
	}
	slot.Store(&kelvin{k: k, c: c})
	return c
}

// kelvin is one temperature and the colour it came out as.
type kelvin struct {
	k float64
	c RGB
}

// kelvins holds the last answer for each slot: one temperature per slot, no
// chaining, so a second temperature landing on a slot takes it over.
//
// Direct-mapped rather than a map with a lock, because this is asked from
// every row of a bake at once and the rows are drawn in parallel: a lock here
// serialises them, which a profile of the first attempt showed as threads
// asleep. A stale or lost entry costs one integration and nothing else.
var kelvins [1 << 13]atomic.Pointer[kelvin]

// kelvinSlot is which slot a temperature belongs in. The bits of the float are
// mixed, because temperatures arrive in runs -- a sky's stars are a smooth
// function of a hash -- and the low bits alone would pile up in a few slots.
func kelvinSlot(k float64) uint64 {
	h := math.Float64bits(k)
	h ^= h >> 33
	h *= 0xff51afd7ed558ccd
	h ^= h >> 29
	return h & (uint64(len(kelvins)) - 1)
}

// Ramp is a libtheme ramp sampled into light, for asking a hundred thousand
// times a frame. The ramp mixes in whatever space it was made with; the
// samples are close enough together that a straight line between two of them
// is not a visible difference.
type Ramp struct{ lut []RGB }

// rampSamples is how finely a ramp is sampled.
const rampSamples = 512

// Bake samples a ramp.
func Bake(r functions.Ramp) Ramp {
	lut := make([]RGB, rampSamples)
	for i := range lut {
		lut[i] = FromSwatch(r.At(float64(i) / (rampSamples - 1)))
	}
	return Ramp{lut: lut}
}

// HexRamp is a ramp written as positions and hex colours, mixed in oklab and
// baked, the way grade.Ramp writes one.
func HexRamp(stops ...any) Ramp { return Bake(grade.Ramp(stops...)) }

// At is the light at t, holding the ends outside nought to one.
func (r Ramp) At(t float64) RGB {
	if len(r.lut) == 0 {
		return RGB{}
	}
	f := t * float64(len(r.lut)-1)
	if !(f > 0) { // NaN included: a bad t is the dark end, not a panic
		return r.lut[0]
	}
	i := int(f)
	if i >= len(r.lut)-1 {
		return r.lut[len(r.lut)-1]
	}
	return r.lut[i].Mix(r.lut[i+1], f-float64(i))
}

// Buffer is a frame of light, and how far away whatever was drawn at each
// subpixel is.
//
// One light a subpixel. A surface replaces what is there, an atmosphere or a
// spark adds to it, and neither asks what was there before. That is only
// right if what is behind is drawn first, which is why bodies go through
// scenery.Draw: it orders them furthest away first.
//
// Who owns a subpixel, for anything sharing a buffer:
//
//	a solid owns it; a light does not.
//	a light reads the depth and never writes it.
//	a buffer is drawn bodies first, then lights.
//
// There is no fourth arrangement, and both halves are tests in scenery. A
// light that writes depth culls the body behind it, which is a glow with a
// body-shaped hole around it. And a body is opaque where it draws, so a
// light drawn before one is simply replaced.
//
// Derived by opezart in compose, by writing the smallest scene with one of
// each in it and watching it fail both ways. This library already did it
// everywhere and said it nowhere.
type Buffer struct {
	W, H  int
	pix   []float32 // three a subpixel
	depth []float32
}

// New is a black frame with nothing in it.
func New(w, h int) *Buffer {
	b := &Buffer{W: w, H: h, pix: make([]float32, w*h*3),
		depth: make([]float32, w*h)}
	for i := range b.depth {
		b.depth[i] = float32(math.Inf(1))
	}
	return b
}

// inside is whether a point is on the frame.
//
// Every way in and out of the buffer asks, because whoever is drawing has
// usually worked the point out from a projection or a radius and cannot
// always know: a spark half off the edge, a glow that reaches past it.
//
// Add said so and clipped from the start. The others indexed the slice and
// took what they found: a panic off the right edge, which opezart hit writing
// a note's depth at an odd terminal size.
//
// Off the left it was worse. An x of minus one on row two is a valid index
// for the end of row one, so it wrote to the wrong cell and said nothing.
func (b *Buffer) inside(x, y int) bool {
	return x >= 0 && y >= 0 && x < b.W && y < b.H
}

// At is the light at a subpixel, and black off the frame.
func (b *Buffer) At(x, y int) RGB {
	if !b.inside(x, y) {
		return RGB{}
	}
	i := (y*b.W + x) * 3
	return RGB{float64(b.pix[i]), float64(b.pix[i+1]),
		float64(b.pix[i+2])}
}

// Set replaces the light at a subpixel. Off the frame it does nothing.
func (b *Buffer) Set(x, y int, c RGB) {
	if !b.inside(x, y) {
		return
	}
	i := (y*b.W + x) * 3
	b.pix[i], b.pix[i+1], b.pix[i+2] = float32(c.R), float32(c.G),
		float32(c.B)
}

// Add puts more light on a subpixel. Outside the frame it does nothing, so a
// spark half off the edge needs no clipping by whoever drew it.
func (b *Buffer) Add(x, y int, c RGB) {
	if !b.inside(x, y) {
		return
	}
	i := (y*b.W + x) * 3
	b.pix[i] += float32(c.R)
	b.pix[i+1] += float32(c.G)
	b.pix[i+2] += float32(c.B)
}

// Over lays light over a subpixel, hiding a of what was there.
func (b *Buffer) Over(x, y int, c RGB, a float64) {
	b.Set(x, y, b.At(x, y).Mix(c, a))
}

// Depth is how far away the nearest thing drawn at a subpixel is, in world
// units, and infinite where nothing solid has been or off the frame.
func (b *Buffer) Depth(x, y int) float64 {
	if !b.inside(x, y) {
		return math.Inf(1)
	}
	return float64(b.depth[y*b.W+x])
}

// SetDepth records how far away the thing just drawn at a subpixel is. Off
// the frame it does nothing.
func (b *Buffer) SetDepth(x, y int, d float64) {
	if !b.inside(x, y) {
		return
	}
	b.depth[y*b.W+x] = float32(d)
}

// Paint puts a grade stack down, as light, over the whole frame.
func (b *Buffer) Paint(sh grade.Sheet, st grade.Stack) {
	split.Rows(b.H, func(lo, hi int) {
		for y := lo; y < hi; y++ {
			for x := 0; x < b.W; x++ {
				b.Set(x, y, FromSwatch(st.At(sh.UV(x, y))))
			}
		}
	})
}
