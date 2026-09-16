package main

import (
	"github.com/janearc/libdaffy/field"

	"github.com/janearc/miami/cells"
	"github.com/janearc/miami/grade"
	"github.com/janearc/miami/scenes"
)

// fields: the three shapes libdaffy ships, painted side by side through one
// ramp, which is the point -- the ramp does not know which is which.
func fieldsPage() ([][]cells.Cell, string) {
	const cols, rows = 66, 9
	sh := grade.Sextants(cols, rows)
	r := scenes.PageRamp()
	third := func(i int) field.Field {
		switch i {
		case 0:
			return field.Linear{A: field.Point{U: 0.02, V: 0.5}, B: field.Point{U: 0.31, V: 0.5}}
		case 1:
			return field.Radial{Centre: field.Point{U: 0.5, V: 0.5}, Radius: 0.15}
		default:
			return field.Conic{Centre: field.Point{U: 0.82, V: 0.5}}
		}
	}
	st := grade.Stack{}
	for i := 0; i < 3; i++ {
		lo, hi := float64(i)/3, float64(i+1)/3
		st = append(st, grade.Layer{
			Name: "f", Field: third(i), Ramp: r, Gain: 1,
			Alpha: field.Func(func(u, _ float64) float64 {
				if u < lo+0.01 || u >= hi-0.01 {
					return 0
				}
				return 1
			}),
		})
	}
	return cells.ToCells(sh.Paint(st), 2, 3), `
      linear                  radial                  conic

One ramp, three fields, and the ramp never learns which is which. A field is
a rule from a position on the sheet -- u and v in 0..1 -- to a t in 0..1; a
ramp turns that t into a colour. Keeping them apart is what lets one ramp
paint a sun, a sky and a stripe.

In a scene document a field is an object with exactly one key. linear takes
a and b, radial a centre and radius, conic a centre; through wraps any of
them in a matrix so circles become ellipses and axes lean. For an alpha there
are inside, band, smooth, stripes, vband and times, and two primitives that
carry real maths because a document should not: waves and stars.

Fields clamp t at 1. Past the radius and exactly at the radius are the same
number, so an alpha that only reaches 0 at t = 1 is still a quarter open
across the whole rest of the sheet. Make it reach 0 before 1.

A shape none of these describes wants a new primitive in Go, not a cleverer
document. There is deliberately no arithmetic in the file.`
}
