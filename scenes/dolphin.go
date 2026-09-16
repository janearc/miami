package scenes

import (
	"image/color"

	"github.com/janearc/miami/geom"
	"github.com/janearc/miami/raster"
)

// two bottlenose dolphins leaving the water, heads to the left, drawn from the
// photograph's pose. traced silhouettes of that photograph read as two dark
// diagonals at any size a manpage can spare:
//
// the features that say dolphin are the beak under the melon, the backswept
// dorsal, and the horizontal fluke, and the first and last of those are lost in
// the source.

// dolphin returns one animal: a spine with girth, its fins, and its fluke.
// scale 1 is the leading animal; x, y place the tip of its rostrum.
func dolphin(x, y, k float64) geom.Shape {
	at := func(dx, dy float64) geom.Vec { return geom.Vec{X: x + dx*k, Y: y + dy*k} }
	r := func(v float64) float64 { return v * k }

	// the arc: rostrum low, back rising to the dorsal, then falling away to
	// a thin peduncle
	body := geom.Taper{
		P: []geom.Vec{at(0, 6), at(9, 2.5), at(22, 0.5), at(40, 1.5), at(62, 8),
			at(84, 20), at(102, 32), at(113, 40)},
		R: []float64{r(1.0), r(4.2), r(7.4), r(8.2), r(6.6), r(4.0), r(1.9), r(0.9)},
	}
	// the melon: a dome over the beak, with the crease between them left as
	// the step where the two shapes meet
	melon := geom.Ellipse{C: at(13, 0.2), Rx: r(8.0), Ry: r(5.2), Rot: -12}
	// the beak itself, slimmer than the head and angled down
	beak := geom.Taper{P: []geom.Vec{at(-1, 7.2), at(8, 4.2)}, R: []float64{r(0.8), r(2.4)}}

	// a clean backswept triangle: the first attempt's trailing edge cut
	// back into the body and left a notch in the animal's back
	dorsal := geom.Poly{P: geom.Pts(
		at(38, -4), at(46, -16), at(51, -17.5), at(55, -9), at(58, -4))}
	pectoral := geom.Poly{P: geom.Pts(
		at(28, 6), at(38, 13), at(42, 19), at(34, 14), at(29, 9))}
	// the fluke is horizontal, which is the whole difference from a fish,
	// so it is drawn as two lobes with a notch between them one crescent
	// with a notch, not two slivers: the lobes meet the peduncle as a
	// single piece, which is how a fluke reads at any size
	fluke := geom.Poly{P: geom.Pts(
		at(105, 35), at(114, 30.5), at(124, 26.5), at(127, 29), at(120, 35.5),
		at(119, 39.5), at(124, 43.5), at(129, 50.5), at(125, 52), at(117, 45),
		at(108, 42.5))}
	return geom.Union(body, melon, beak, dorsal, pectoral, fluke)
}

// Dolphins is the pair for the flipr manpage. each animal is shaded across its
// own girth, dark along the back and light at the belly, because a flat
// silhouette gives the character ramp nothing to work with; the far animal is
// lighter overall, which is the only depth cue at this size.
func Dolphins() *raster.Scene {
	// cropped to the animals, and overlapped as the photograph has them:
	// side by side they stretched the frame and left each one three rows
	// thick
	const w, h = 118.0, 64.0
	type placed struct {
		x, y, k float64
		role    byte
		dark    float64
	}
	set := []placed{{16, 10, 0.70, 'd', 0.22}, {9, 38, 0.46, 'e', 0.45}}
	sc := &raster.Scene{W: w, H: h}
	for _, p := range set {
		sc.Parts = append(sc.Parts, raster.Part{Role: p.role, Shape: dolphin(p.x, p.y, p.k)})
	}
	sc.Pal = map[byte]color.RGBA{'d': {40, 40, 40, 255}, 'e': {90, 90, 90, 255}}
	sc.Bg = func(x, y float64) color.RGBA { return color.RGBA{255, 255, 255, 255} }
	sc.Shade = func(role byte, x, y float64, c color.RGBA) color.RGBA {
		for _, p := range set {
			if p.role != role {
				continue
			}
			// across the animal, measured square to its spine
			const sin, cos = 0.34, 0.94 // the spine runs about 20 degrees down
			d := (-(x-p.x)*sin + (y-p.y)*cos) / (34 * p.k)
			if d < 0 {
				d = 0
			}
			if d > 1 {
				d = 1
			}
			v := uint8(255 * (p.dark + (0.86-p.dark)*d*d))
			return color.RGBA{v, v, v, 255}
		}
		return c
	}
	// the plain text palette: the animals are ink, the sky is the page both
	// animals solid: a mid tone on the smaller one dithered its tail into
	// fragments, and depth is not worth a broken fluke
	sc.Tone = map[byte]float64{'d': 1, 'e': 1}
	sc.ToneBg = func(x, y float64) float64 { return 0 }
	return sc
}
