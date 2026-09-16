package motifs

import (
	"math"
	"miami/geom"
)

// a frond launches straight and curls late: the bearing turns by bend*(i/n)^2,
// so little happens over the first third and the tip whips over. distributing
// the turn evenly instead makes every frond start curving at once, they all
// overlap near the crown, and at low resolution the crown fuses into a bar.
func Frond(base geom.Vec, ang0, bend, length, w float64) geom.Taper {
	const n = 8
	step := length / n
	p := base
	pts := []geom.Vec{base}
	rad := []float64{w}
	for i := 1; i <= n; i++ {
		f := float64(i) / n
		r := (ang0 + bend*f*f) * math.Pi / 180
		p = geom.Vec{X: p.X + math.Cos(r)*step, Y: p.Y + math.Sin(r)*step}
		pts = append(pts, p)
		rad = append(rad, w*(1-f*f*0.86))
	}
	return geom.Taper{P: pts, R: rad}
}

// PalmAt builds one palm in the composite's own coordinates. h is trunk
// height; lean is how far the crown sits from the base, positive to the right.
func PalmAt(base geom.Vec, h, lean float64, flip bool, skew float64) []geom.Shape {
	crown := geom.Vec{X: base.X + lean, Y: base.Y - h}
	var tp []geom.Vec
	var tr []float64
	for _, t := range []float64{0, 0.32, 0.60, 0.82, 1.0} {
		tp = append(tp, geom.Vec{X: base.X + lean*math.Pow(t, 1.7), Y: base.Y - h*t})
		tr = append(tr, h*(0.056-0.032*t))
	}
	out := []geom.Shape{geom.Taper{P: tp, R: tr}}

	type fs struct{ ang, bend, frac float64 }
	specs := []fs{
		{-95, -125, 1.00}, {-128, -100, 0.86}, {-158, -66, 0.63},
		{-85, 125, 1.00}, {-52, 100, 0.86}, {-22, 66, 0.63},
	}
	for _, f := range specs {
		ang, bend := f.ang, f.bend
		if flip {
			ang, bend = -180-ang, -bend
		}
		l := f.frac * 0.44 * h
		out = append(out, Frond(crown, ang+skew, bend, l, 0.086*l))
	}
	out = append(out,
		geom.Ellipse{C: geom.Vec{X: crown.X - 0.035*h, Y: crown.Y + 0.055*h}, Rx: 0.032 * h, Ry: 0.028 * h},
		geom.Ellipse{C: geom.Vec{X: crown.X + 0.030*h, Y: crown.Y + 0.062*h}, Rx: 0.030 * h, Ry: 0.026 * h},
	)
	return out
}

// BirdAt is a shallow gull chevron: the cheapest mark that puts empty sky to
// work and earns the title.
func BirdAt(c geom.Vec, span, tilt float64) geom.Shape {
	s := span
	l := []geom.Vec{{X: c.X - 0.50*s, Y: c.Y - 0.11*s}, {X: c.X - 0.24*s, Y: c.Y + 0.06*s}, c}
	r := []geom.Vec{{X: c.X + 0.50*s, Y: c.Y - 0.11*s}, {X: c.X + 0.24*s, Y: c.Y + 0.06*s}, c}
	for i := range l {
		l[i] = geom.Rotate(l[i], c, tilt)
		r[i] = geom.Rotate(r[i], c, tilt)
	}
	w := []float64{0.045 * s, 0.075 * s, 0.090 * s}
	return geom.Union(geom.Taper{P: l, R: w}, geom.Taper{P: r, R: w})
}
