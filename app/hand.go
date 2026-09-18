package app

import (
	"math"

	"github.com/janearc/miami/view"
)

// A hand on the picture: dragging turns a camera, the wheel moves it in and
// out.
//
// The pointer lands in a viewport, and the camera that turns is the one
// drawing where the hand is. Two viewports side by side are two things to
// take hold of, which is the whole reason this is not a method on one eye.
//
// What it turns about is worked out when the button goes down and kept for the
// whole drag: straight ahead at the distance of the nearest body, or, on a
// stage with nothing in it, the geometry in view.Eye.Pivot. See pivotFor.

// grab is a drag in progress.
//
// have rather than a viewport of minus one, so that a stage nobody has
// pressed on is not holding viewport nought: the zero value has to mean no
// hand, or the first motion report that arrives with a button already down
// turns a camera about a pivot nobody chose.
type grab struct {
	have  bool
	on    int
	pivot view.Vec
	x, y  int // where the pointer was, in subpixels of the whole sheet
}

// Point turns the cameras by what the pointer did this frame.
//
// A stage with no document's viewports has one camera filling the sheet, so a
// drag anywhere turns it. That is the case every program written before this
// is in, and it needs no viewport to be declared to be steerable.
func (s *Stage) Point(f Frame, cols, rows int) {
	for _, m := range f.Mouse {
		s.point(m, cols, rows)
	}
}

// point acts on one report.
func (s *Stage) point(m Mouse, cols, rows int) {
	switch m.Kind {
	case Press:
		if on, ok := s.under(m, cols, rows); ok {
			s.held = grab{have: true, on: on,
				pivot: s.pivotFor(s.eyeOf(on)),
				x:     m.SubX, y: m.SubY}
			return
		}
		s.held = grab{}
	case Release:
		s.held = grab{}
	case Move:
		if !m.Held() || !s.held.have {
			return
		}
		eye := s.eyeOf(s.held.on)
		// a drag across the viewport is half a turn, so how far the
		// hand went is measured against the viewport it went across
		// and not against the terminal
		az, el := view.Drag(m.SubX-s.held.x, m.SubY-s.held.y,
			s.wide(s.held.on, cols))
		*eye = eye.Turned(s.held.pivot, -az, el)
		s.held.x, s.held.y = m.SubX, m.SubY
	case Wheel:
		on, ok := s.under(m, cols, rows)
		if !ok {
			return
		}
		// toward the reader is away from the scene. every notch is the
		// same share of what is left, so the wheel never arrives.
		factor := 1 / notch
		if m.Wheel < 0 {
			factor = notch
		}
		eye := s.eyeOf(on)
		*eye = eye.Dollied(s.pivotFor(eye), factor)
	}
}

// pivotFor is what a camera turns about in this scene.
//
// view.Eye.Pivot is geometry and knows nothing about what is being looked at:
// it takes the point on the line of sight nearest the middle of the scene,
// which is right when the subject is near the middle and wrong otherwise.
//
// The quasar sits sixty-two units out with the camera at the origin. Orbiting
// the middle there is a camera turning on its own nose, while the thing it was
// looking at leaves the frame.
//
// The stage does know: it already asks every body how far away it is, to draw
// them furthest first. So the pivot is straight ahead at the distance of the
// nearest body, and only a stage with nothing in it falls back to geometry.
//
// Straight ahead rather than at the body itself, which is the same point when
// the camera is looking at it and a gentler one when it is not: a pivot off to
// the side would swing the picture as well as turn it.
func (s *Stage) pivotFor(eye *view.Eye) view.Vec {
	near := math.Inf(1)
	for _, p := range s.scenery {
		if d := away(p, *eye); d > 0 && d < near {
			near = d
		}
	}
	if math.IsInf(near, 1) {
		return eye.Pivot()
	}
	f := eye.Forward()
	return view.Vec{X: eye.At.X + f.X*near, Y: eye.At.Y + f.Y*near,
		Z: eye.At.Z + f.Z*near}
}

// notch is how much of the distance to the pivot one turn of the wheel
// closes. A tenth is about as fast as a hand can follow without losing where
// it was.
const notch = 0.9

// under is which viewport the pointer is in, and false when it is in none.
//
// A stage with no viewports of its own has one, filling the sheet, and it is
// viewport nought: the answer is the same shape either way, so nothing that
// acts on it has to know which kind of stage it is.
func (s *Stage) under(m Mouse, cols, rows int) (int, bool) {
	if len(s.shots) == 0 {
		if m.Col < 0 || m.Row < 0 || m.Col >= cols || m.Row >= rows {
			return 0, false
		}
		return 0, true
	}
	for i, v := range s.shots {
		x0, y0, x1, y1 := v.cells(cols, rows)
		if m.Col >= x0 && m.Col < x1 && m.Row >= y0 && m.Row < y1 {
			return i, true
		}
	}
	return 0, false
}

// eyeOf is the eye a viewport draws through, to be turned in place.
func (s *Stage) eyeOf(on int) *view.Eye {
	if len(s.shots) == 0 || on < 0 || on >= len(s.shots) {
		return &s.eye
	}
	return s.shots[on].eye
}

// wide is how many subpixels across a viewport is.
func (s *Stage) wide(on, cols int) int {
	if len(s.shots) == 0 || on < 0 || on >= len(s.shots) {
		return cols * 2
	}
	x0, _, x1, _ := s.shots[on].cells(cols, 1)
	return (x1 - x0) * 2
}
