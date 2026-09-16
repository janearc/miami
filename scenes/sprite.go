package scenes

import (
	"fmt"
	"image/color"
	"os"
	"strings"

	"github.com/janearc/miami/geom"
	"github.com/janearc/miami/motifs"
	"github.com/janearc/miami/raster"
	"github.com/janearc/miami/typeset"
)

// the role mapping lives HERE, in one place, because the eleven-character
// alphabet is deliberately unfinished: if it changes, this table changes and
// every sprite is re-emitted, rather than every scene being rewritten.
const (
	RoleNone    = '.'
	RoleDark    = 'K'
	RoleLight   = 'W'
	RoleWing    = 'V'
	RoleStripe  = 'D'
	RoleBase    = 'B'
	RoleBand    = 'Y'
	RoleTip     = 'R'
	RoleFeet    = 'O'
	RolePupil   = 'E'
	RoleEyeRing = 'X'
)

// DolphinSprite is the pair drawn as a stencil rather than as a picture: the
// body is the dark mass, the fins sit just off it so a long dark shape is not
// one blob, and the rostrum takes the three protruding bands.
func DolphinSprite() *raster.Scene {
	const w, h = 118.0, 64.0
	sc := &raster.Scene{W: w, H: h}
	add := func(x, y, k float64) {
		at := func(dx, dy float64) geom.Vec { return geom.Vec{X: x + dx*k, Y: y + dy*k} }
		r := func(v float64) float64 { return v * k }
		body := geom.Taper{
			P: []geom.Vec{at(0, 6), at(9, 2.5), at(22, 0.5), at(40, 1.5), at(62, 8),
				at(84, 20), at(102, 32), at(113, 40)},
			R: []float64{r(1.0), r(4.2), r(7.4), r(8.2), r(6.6), r(4.0), r(1.9), r(0.9)},
		}
		melon := geom.Ellipse{C: at(13, 0.2), Rx: r(8.0), Ry: r(5.2), Rot: -12}
		beak := geom.Taper{P: []geom.Vec{at(-1, 7.2), at(8, 4.2)}, R: []float64{r(0.8), r(2.4)}}
		dorsal := geom.Poly{P: geom.Pts(at(38, -4), at(46, -16), at(51, -17.5), at(55, -9), at(58, -4))}
		pectoral := geom.Poly{P: geom.Pts(at(28, 6), at(38, 13), at(42, 19), at(34, 14), at(29, 9))}
		fluke := geom.Poly{P: geom.Pts(
			at(105, 35), at(114, 30.5), at(124, 26.5), at(127, 29), at(120, 35.5),
			at(119, 39.5), at(124, 43.5), at(129, 50.5), at(125, 52), at(117, 45),
			at(108, 42.5))}
		belly := geom.Intersect(geom.Union(body, melon),
			geom.HalfPlane{C: at(0, 5.5), N: geom.Vec{X: 0.17, Y: -0.98}})

		sc.Parts = append(sc.Parts,
			raster.Part{Role: RoleDark, Shape: geom.Union(body, melon)},
			raster.Part{Role: RoleLight, Shape: belly},
			raster.Part{Role: RoleWing, Shape: geom.Union(dorsal, pectoral, fluke)},
			raster.Part{Role: RoleBase, Shape: geom.Intersect(beak, geom.HalfPlane{C: at(5, 0), N: geom.Vec{X: 1}})},
			raster.Part{Role: RoleBand, Shape: geom.Intersect(beak, geom.HalfPlane{C: at(2.5, 0), N: geom.Vec{X: 1}})},
			raster.Part{Role: RoleTip, Shape: geom.Intersect(beak, geom.HalfPlane{C: at(2.5, 0), N: geom.Vec{X: -1}})},
			raster.Part{Role: RoleEyeRing, Shape: geom.Ellipse{C: at(17, 1.5), Rx: r(2.2), Ry: r(2.0)}},
			raster.Part{Role: RolePupil, Shape: geom.Ellipse{C: at(17, 1.5), Rx: r(1.1), Ry: r(1.0)}},
		)
	}
	add(16, 10, 0.70)
	add(9, 38, 0.46)
	sc.Pal = map[byte]color.RGBA{}
	sc.Bg = func(x, y float64) color.RGBA { return color.RGBA{} }
	return sc
}

// BulbSprite is the bulb from the lights sign, drawn cold.
//
// Deliberately cold: no rays, no lit glass, nothing warm. The still sprite is a
// bulb that is not on, because the light is the other sprite. If the stencil
// were already glowing there would be nothing for an alpha layer to add, and
// the effect would be decoration rather than illumination.
//
// The roles therefore run on lightness alone -- clear glass above dark metal,
// with the filament the darkest thing inside the glass so it has somewhere to
// go when the current arrives.
func BulbSprite() *raster.Scene {
	const w, h = 44.0, 74.0
	b := motifs.NewBulb(w/2, 0.03*h, 0.94*h)
	sc := &raster.Scene{W: w, H: h}
	sc.Parts = []raster.Part{
		{Role: RoleWing, Shape: b.Glass},
		{Role: RoleDark, Shape: b.Screw},
		{Role: RolePupil, Shape: b.Grooves},
		{Role: RolePupil, Shape: b.Insulator},
		{Role: RoleWing, Shape: b.Tip},
		{Role: RoleStripe, Shape: b.Wires},
		{Role: RoleStripe, Shape: b.Filament},
		{Role: RoleLight, Shape: b.Highlight}}
	// the filament and the wires are one pixel of tungsten; without a cover
	// override the vote drops them and the bulb reads as an empty jar
	sc.MinCover = map[byte]float64{RoleStripe: 0.22, RoleLight: 0.30}
	sc.Pal = map[byte]color.RGBA{}
	sc.Bg = func(x, y float64) color.RGBA { return color.RGBA{} }
	return sc
}

// WordSprite sets a word as a stencil, so an effect written for a drawing can
// be run over type without knowing the difference.
//
// The letters take one role and one role only. A wordmark has no parts to speak
// of -- there is the type and there is the ground -- and the whole point of
// handing it to the same compositor is that light does not care which it is
// looking at.
//
// Unlit sign letters are dark paint, so they are the dark role, which leaves
// them somewhere to go.
func WordSprite(text string, w, pad float64) *raster.Scene {
	return wordScene(text, w, pad, 0.22,
		typeset.Wobble{Tilt: 1.6, Shift: 0.04, Space: 0.05, Seed: 11})
}

// wordScene is the shared setting. The wobble is a parameter because it is only
// right in one place: a painted sign is set letter by letter and each one sits
// a hair off true, which is what makes it look painted.
//
// A wordmark two cells tall is not painted, it is a logotype, and the same
// wobble there just reads as broken letters.
func wordScene(text string, w, pad, weight float64, wob typeset.Wobble) *raster.Scene {
	gothic := mustFace(supp+"Futura.ttc", "Futura Condensed Medium")
	line := typeset.Line{Text: text, Face: gothic, Tracking: 0.10, Flat: true,
		Weight: weight, Wobble: wob}
	// fitCap is how the sign sets its own lines: ask what cap height fits
	// the measure, rather than picking one and hoping
	line.CapHeight = fitCap(line, w-2*pad)
	h := line.CapHeight + 2*pad

	sc := &raster.Scene{W: w, H: h}
	sc.Parts = []raster.Part{{Role: RoleDark,
		Shape: geom.Union(line.Set(w/2, h-pad)...)}}
	// condensed capitals at this size are hairlines at their thinnest;
	// without a cover floor the vote eats the stems and the word reads as a
	// dotted line
	sc.MinCover = map[byte]float64{RoleDark: 0.28}
	sc.Pal = map[byte]color.RGBA{}
	sc.Bg = func(x, y float64) color.RGBA { return color.RGBA{} }
	return sc
}

// WriteSprite rasterises a scene into a stencil at daffy's pixel, which is one
// column wide and half a row tall, and writes the .sprite file.
func WriteSprite(sc *raster.Scene, name, path string, cols int) error {
	rows := int(float64(cols)*sc.H/sc.W + 0.5) // pixels, not cells: pixels are half as tall
	g := raster.Rasterize(sc, cols, rows, 4)
	var b strings.Builder
	fmt.Fprintf(&b, "# %s, from miami's own geometry: parts, not a photograph.\n", name)
	fmt.Fprintf(&b, "# %d by %d at daffy pixels, which is %d by %d cells.\n",
		cols, rows, cols, (rows+1)/2)
	fmt.Fprintf(&b, "sprite %s\n\nart\n", name)
	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			r := g.Rows[y][x]
			if r == raster.Empty {
				r = RoleNone
			}
			b.WriteByte(r)
		}
		b.WriteByte('\n')
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

// WordMask sets a word and returns which subpixels it inks.
//
// The type is set by the same typesetter that sets the sign, and rasterised
// straight to the subpixel grid rather than to roles, because a wordmark
// composited beside a picture is not a stencil -- it takes its colour from the
// picture it is standing next to.
//
// The height is derived, not given. Type has a size of its own once you fix its
// measure, and the caller asking for a box of the wrong shape would either
// squash the letters or leave them swimming.
func WordMask(text string, w int, aspect float64) [][]bool {
	// set true and a shade heavier: at this size the stems are two
	// subpixels and a hairline drops out of the vote entirely
	sc := wordScene(text, 100, 4, 0.5, typeset.Wobble{})
	h := int(float64(w)*sc.H/sc.W*aspect + 0.5)
	if h < 1 {
		h = 1
	}
	g := raster.Rasterize(sc, w, h, 4)
	out := make([][]bool, h)
	for y := 0; y < h; y++ {
		out[y] = make([]bool, w)
		for x := 0; x < w; x++ {
			out[y][x] = g.Rows[y][x] != raster.Empty
		}
	}
	return out
}
