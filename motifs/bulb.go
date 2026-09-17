package motifs

import (
	"math"

	"github.com/janearc/miami/geom"
)

// Bulb is an A19 incandescent lamp, upright. proportions come from the real
// part -- 60mm globe, 110mm overall, E26 screw base 26mm across -- because
// eyeballing a manufactured object is how the car went wrong.
type Bulb struct {
	Glass     geom.Shape // globe and neck
	Filament  geom.Shape
	Wires     geom.Shape // the two supports up from the stem
	Highlight geom.Shape // the curved sliver of reflection that says glass
	Screw     geom.Shape // the threaded base, ridges included
	Grooves   geom.Shape // the dark lines between the ridges
	Insulator geom.Shape
	Tip       geom.Shape
	All       geom.Shape // the silhouette

	Centre geom.Vec // globe centre
	R      float64  // globe radius
	Bottom float64  // y of the contact tip
}

// NewBulb places a bulb whose globe top is at top and whose overall height is
// h, centred on x.
func NewBulb(x, top, h float64) Bulb {
	k := h / 110 // millimetres to physical units
	mm := func(dx, dy float64) geom.Vec {
		return geom.Vec{X: x + dx*k, Y: top + dy*k}
	}

	var b Bulb
	b.R = 30 * k
	b.Centre = mm(0, 30)
	globe := geom.Ellipse{C: b.Centre, Rx: b.R, Ry: b.R}
	// the pear: the globe runs into a neck that narrows to the screw the
	// neck starts inside the globe and wider than it is there, or the join
	// shows as a notch on each side
	neck := geom.Taper{
		P: []geom.Vec{mm(0, 38), mm(0, 64), mm(0, 83)},
		R: []float64{27.5 * k, 17.5 * k, 13.2 * k},
	}
	b.Glass = geom.Union(globe, neck)

	// three ridges that stand proud of the base: the silhouette cue that
	// separates a bulb from a balloon or a cone
	screwBody := geom.Poly{P: geom.Pts(
		mm(-13, 82), mm(13, 82), mm(12.2, 103), mm(-12.2, 103))}
	var ridges []geom.Shape
	var grooves []geom.Shape
	for _, y := range []float64{86.5, 92.5, 98.5} {
		ridges = append(ridges, geom.Taper{
			P: []geom.Vec{mm(-13.6, y), mm(13.6, y)},
			R: []float64{2.1 * k, 2.1 * k}})
		grooves = append(grooves, geom.Taper{
			P: []geom.Vec{mm(-12.4, y+3), mm(12.4, y+3)},
			R: []float64{0.7 * k, 0.7 * k}})
	}
	b.Screw = geom.Union(append(ridges, screwBody)...)
	b.Grooves = geom.Intersect(geom.Union(grooves...), screwBody)
	b.Insulator = geom.Poly{P: geom.Pts(
		mm(-9.5, 102.5), mm(9.5, 102.5), mm(7.5, 107), mm(-7.5, 107))}
	b.Tip = geom.Ellipse{C: mm(0, 107), Rx: 4.2 * k, Ry: 3 * k}
	b.Bottom = top + 110*k

	// the filament: a zigzag coil across the globe centre, the shape every
	// drawing of an idea uses, because it reads at any size
	var fp []geom.Vec
	const zig = 7
	for i := 0; i <= zig; i++ {
		dx := -11 + 22*float64(i)/zig
		dy := 30.0
		if i%2 == 1 {
			dy = 24.0
		}
		fp = append(fp, mm(dx, dy))
	}
	fr := make([]float64, len(fp))
	for i := range fr {
		fr[i] = 1.5 * k
	}
	b.Filament = geom.Taper{P: fp, R: fr}
	b.Wires = geom.Union(
		geom.Taper{
			P: []geom.Vec{mm(-4, 76), mm(-11, 30)},
			R: []float64{0.8 * k, 0.8 * k},
		},
		geom.Taper{
			P: []geom.Vec{mm(4, 76), mm(11, 30)},
			R: []float64{0.8 * k, 0.8 * k},
		},
		geom.Taper{
			P: []geom.Vec{mm(-4, 76), mm(4, 76), mm(0, 83)},
			R: []float64{1.2 * k, 1.2 * k, 1.2 * k},
		},
	)

	// highlight: an arc hugging the upper left inside of the globe
	var hp []geom.Vec
	var hr []float64
	for i := 0; i <= 8; i++ {
		a := (200 + 55*float64(i)/8) * math.Pi / 180
		hp = append(hp, geom.Vec{X: b.Centre.X + 0.74*b.R*math.Cos(a),
			Y: b.Centre.Y + 0.74*b.R*math.Sin(a)})
		t := float64(i) / 8
		hr = append(hr, (0.6+2.2*math.Sin(t*math.Pi))*k)
	}
	b.Highlight = geom.Taper{P: hp, R: hr}

	b.All = geom.Union(b.Glass, b.Screw, b.Insulator, b.Tip)
	return b
}

// Sunburst is n thin rays at three repeating lengths around a centre, between
// two angles in screen degrees (0 is right, 90 is down). the period form is
// thin, even and straight; fat alternating wedges read as a cartoon explosion.
func Sunburst(
	c geom.Vec, inner float64, lengths []float64, n int, from, to,
	width float64,
) geom.Shape {
	var rays []geom.Shape
	for i := 0; i < n; i++ {
		t := (float64(i) + 0.5) / float64(n)
		a := (from + (to-from)*t) * math.Pi / 180
		outer := lengths[i%len(lengths)]
		d := geom.Vec{X: math.Cos(a), Y: math.Sin(a)}
		rays = append(rays, geom.Taper{
			P: []geom.Vec{
				{X: c.X + d.X*inner, Y: c.Y + d.Y*inner},
				{X: c.X + d.X*outer, Y: c.Y + d.Y*outer},
			},
			R: []float64{width, width * 0.35},
		})
	}
	return geom.Union(rays...)
}
