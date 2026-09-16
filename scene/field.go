package scene

import (
	"encoding/json"
	"fmt"
	"math"

	"github.com/janearc/libdaffy/field"

	"miami/grade"
)

// Node is one field, named by its single key.
//
// Three of these are libdaffy's own -- linear, radial, conic -- and through is
// its matrix wrapper. The rest are either combinators, which build an alpha out
// of other fields, or primitives that carry real maths: waves and stars.
//
// Those two are the interesting admission. A scene document cannot express a
// sum of drifting sines or a field of twinkling points, and it should not try:
// the alternative is an expression language, and the cost of that is a
// programming language nobody can debug.
//
// So they are named, their parameters are data, and a shape none of them
// describes is a reason to write a new primitive in Go rather than to make the
// file cleverer.
type Node struct {
	raw map[string]json.RawMessage
}

// UnmarshalJSON reads a field, which is an object with exactly one name in it.
func (n *Node) UnmarshalJSON(b []byte) error {
	n.raw = map[string]json.RawMessage{}
	if err := json.Unmarshal(b, &n.raw); err != nil {
		return fmt.Errorf("field: %w", err)
	}
	if len(n.raw) != 1 {
		return fmt.Errorf("a field has exactly one name, got %d", len(n.raw))
	}
	return nil
}

// Pt is a position on the sheet, either coordinate of which may move.
type Pt [2]Value

// at is the point this position resolves to at a moment in the scene.
func (p Pt) at(c ctx) field.Point {
	return field.Point{U: p[0].at(c), V: p[1].at(c)}
}

// field builds the field this node names, at a moment in the scene.
func (n Node) field(c ctx) (field.Field, error) {
	for name, body := range n.raw {
		switch name {

		case "linear":
			var a struct{ A, B Pt }
			if err := json.Unmarshal(body, &a); err != nil {
				return nil, err
			}
			return field.Linear{A: a.A.at(c), B: a.B.at(c)}, nil

		case "radial":
			var a struct {
				Centre Pt
				Radius Value
			}
			if err := json.Unmarshal(body, &a); err != nil {
				return nil, err
			}
			return field.Radial{Centre: a.Centre.at(c), Radius: a.Radius.at(c)}, nil

		case "conic":
			var a struct{ Centre Pt }
			if err := json.Unmarshal(body, &a); err != nil {
				return nil, err
			}
			return field.Conic{Centre: a.Centre.at(c)}, nil

		case "through":
			var a struct {
				M     [2][2]Value
				Field Node
			}
			if err := json.Unmarshal(body, &a); err != nil {
				return nil, err
			}
			inner, err := a.Field.field(c)
			if err != nil {
				return nil, err
			}
			var m [2][2]float64
			for i := range m {
				for j := range m[i] {
					m[i][j] = a.M[i][j].at(c)
				}
			}
			return field.Through{M: m, Field: inner}, nil

		case "flat":
			var v Value
			if err := json.Unmarshal(body, &v); err != nil {
				return nil, err
			}
			return grade.Flat(v.at(c)), nil

		case "inside":
			var a struct {
				Field    Node
				At, Soft Value
			}
			if err := json.Unmarshal(body, &a); err != nil {
				return nil, err
			}
			inner, err := a.Field.field(c)
			if err != nil {
				return nil, err
			}
			return grade.Inside(inner, a.At.at(c), a.Soft.at(c)), nil

		case "band":
			var a struct {
				Field           Node
				At, Width, Soft Value
			}
			if err := json.Unmarshal(body, &a); err != nil {
				return nil, err
			}
			inner, err := a.Field.field(c)
			if err != nil {
				return nil, err
			}
			return grade.Band(inner, a.At.at(c), a.Width.at(c), a.Soft.at(c)), nil

		case "stripes":
			var a struct{ Period, Duty, Phase, Lean Value }
			if err := json.Unmarshal(body, &a); err != nil {
				return nil, err
			}
			return grade.Stripes(a.Period.at(c), a.Duty.at(c),
				a.Phase.at(c), a.Lean.at(c)), nil

		case "times":
			var ns []Node
			if err := json.Unmarshal(body, &ns); err != nil {
				return nil, err
			}
			fs := make([]field.Field, len(ns))
			for i, x := range ns {
				f, err := x.field(c)
				if err != nil {
					return nil, err
				}
				fs[i] = f
			}
			return grade.Times(fs...), nil

		case "smooth":
			// a smoothstep across a field's own t: what turns a
			// continuous field into a band with a soft edge
			var a struct {
				Field Node
				A, B  Value
			}
			if err := json.Unmarshal(body, &a); err != nil {
				return nil, err
			}
			inner, err := a.Field.field(c)
			if err != nil {
				return nil, err
			}
			lo, hi := a.A.at(c), a.B.at(c)
			return field.Func(func(u, v float64) float64 {
				return smoothstep(inner.T(u, v), lo, hi)
			}), nil

		case "vband":
			// a gate on height alone: a cloud does not appear below
			// the horizon and a star does not come out over the
			// ground
			var a struct{ Min, Max Value }
			if err := json.Unmarshal(body, &a); err != nil {
				return nil, err
			}
			lo, hi := a.Min.at(c), a.Max.at(c)
			if hi == 0 {
				hi = 1
			}
			return field.Func(func(_, v float64) float64 {
				if v < lo || v > hi {
					return 0
				}
				return 1
			}), nil

		case "waves":
			return waves(body, c)

		case "stars":
			return stars(body, c)
		}
		return nil, fmt.Errorf("unknown field %q", name)
	}
	return nil, fmt.Errorf("empty field")
}

// waves is a sum of plane waves drifting across the sheet, plus a bias and a
// slope with height.
//
// Cloud. The reason it is one primitive rather than three composable ones is
// that the sum is the point: each term has its own period, and periods that
// share no factor are what stop the sky arriving back at an arrangement it has
// already been in.
//
// Compose them separately and you would have to add them, which needs
// arithmetic, which is the thing being avoided.
func waves(body []byte, c ctx) (field.Field, error) {
	var a struct {
		Terms []struct{ U, V, Period, Amp Value }
		Scale Value
		Bias  Value
		Slope Value
		From  Value
	}
	if err := json.Unmarshal(body, &a); err != nil {
		return nil, err
	}
	type term struct{ fu, fv, period, amp float64 }
	ts := make([]term, len(a.Terms))
	for i, t := range a.Terms {
		ts[i] = term{t.U.at(c), t.V.at(c), t.Period.at(c), t.Amp.at(c)}
	}
	scale, bias := a.Scale.at(c), a.Bias.at(c)
	slope, from, t := a.Slope.at(c), a.From.at(c), c.t
	return field.Func(func(u, v float64) float64 {
		s := 0.0
		for _, w := range ts {
			p := w.period
			if p == 0 {
				p = 1
			}
			s += w.amp * math.Sin(2*math.Pi*(u*w.fu+v*w.fv-t/p))
		}
		return s*scale + bias + slope*(from-v)
	}), nil
}

// stars is a fixed field of points that vary in brightness but never move.
//
// The hash is the usual sine-fract trick. It is not good noise, and it does not
// need to be: a star field wants points that stay exactly put between frames,
// which a hash of the cell gives for nothing, and a lattice broken up by a
// second hash so they are not on a grid.
func stars(body []byte, c ctx) (field.Field, error) {
	var a struct {
		Grid, Threshold, Size, Rate Value
	}
	if err := json.Unmarshal(body, &a); err != nil {
		return nil, err
	}
	grid, thr := a.Grid.at(c), a.Threshold.at(c)
	size, rate, t := a.Size.at(c), a.Rate.at(c), c.t
	if grid <= 0 {
		grid = 40
	}
	if size <= 0 {
		size = 0.16
	}
	return field.Func(func(u, v float64) float64 {
		cu, cv := math.Floor(u*grid), math.Floor(v*grid)
		h := hash2(cu, cv)
		if h < thr {
			return 0
		}
		ou, ov := hash2(cu+13, cv), hash2(cu, cv+7)
		d := math.Hypot(u*grid-cu-ou, v*grid-cv-ov)
		if d > size {
			return 0
		}
		bright := 0.55 + 0.45*math.Sin(t*(0.7+rate*hash2(cu+3, cv+5))+h*40)
		return (1 - d/size) * bright
	}), nil
}

// hash2 is a repeatable number from a pair of coordinates. Not good noise; it
// only has to give a star the same place in every frame.
func hash2(x, y float64) float64 {
	s := math.Sin(x*127.1+y*311.7) * 43758.5453
	return s - math.Floor(s)
}

// smoothstep is nought below a, one above b, and eased between.
func smoothstep(x, a, b float64) float64 {
	if x <= a {
		return 0
	}
	if x >= b {
		return 1
	}
	t := (x - a) / (b - a)
	return t * t * (3 - 2*t)
}
