package app

import (
	"math"
	"testing"

	"github.com/janearc/miami/scene"
	"github.com/janearc/miami/view"
)

// twoViews is a stage with two viewports side by side, each with its own
// camera, as a document with two cameras gives.
func twoViews() *Stage {
	s := &Stage{}
	s.shoot(
		[]scene.Camera{
			{Name: "main", Eye: scene.Eye{Lens: 1}},
			{Name: "other", Eye: scene.Eye{Lens: 1}},
		},
		[]scene.Viewport{
			{Camera: "main", X: 0, Y: 0, W: 0.5, H: 1},
			{Camera: "other", X: 0.5, Y: 0, W: 0.5, H: 1},
		},
	)
	// both looking at the middle of the scene from a little way back
	for i := range s.shots {
		s.shots[i].eye.At = view.Vec{Z: -10}
	}
	s.eye.At = view.Vec{Z: -10}
	return s
}

// at is a report of a kind, on a cell, with the left button.
func at(kind MouseKind, col, row int) Mouse {
	b := Left
	if kind == Wheel {
		b = None
	}
	return Mouse{Kind: kind, Button: b, Col: col, Row: row,
		SubX: col * 2, SubY: row * 3}
}

// The camera that turns is the one under the hand, and the other does not
// move. This is the whole reason a drag is not a method on one eye.
func TestTheCameraUnderTheHandIsTheOneThatTurns(t *testing.T) {
	const cols, rows = 80, 24
	s := twoViews()
	left, right := *s.shots[0].eye, *s.shots[1].eye
	s.Point(Frame{Mouse: []Mouse{
		at(Press, 10, 12),
		at(Move, 20, 12),
	}}, cols, rows)
	if *s.shots[0].eye == left {
		t.Error("the hand was on the left viewport and its camera did"+
			" not move: still ", left.At)
	}
	if *s.shots[1].eye != right {
		t.Errorf("the hand was on the left viewport and the right"+
			" camera moved: %v, was %v", *s.shots[1].eye, right)
	}

	// and the same on the other side
	s = twoViews()
	left, right = *s.shots[0].eye, *s.shots[1].eye
	s.Point(Frame{Mouse: []Mouse{
		at(Press, 60, 12),
		at(Move, 70, 12),
	}}, cols, rows)
	if *s.shots[1].eye == right {
		t.Error("the hand was on the right viewport and its camera" +
			" did not move")
	}
	if *s.shots[0].eye != left {
		t.Error("the hand was on the right viewport and the left" +
			" camera moved")
	}
}

// The first viewport's camera is the stage's own, so a program holding what
// Eye handed it is holding what the hand turns.
func TestTheHandAndTheCallerSteerTheSameCamera(t *testing.T) {
	s := twoViews()
	s.Point(Frame{Mouse: []Mouse{at(Press, 10, 12), at(Move, 20, 12)}},
		80, 24)
	if *s.Eye() != *s.shots[0].eye {
		t.Errorf("Eye hands out %v and the first viewport draws %v",
			*s.Eye(), *s.shots[0].eye)
	}
}

// A motion report with a button held, arriving with no press before it, must
// turn nothing. The terminal sends these: a button can be down before the
// program started reporting.
func TestAMoveWithNoPressTurnsNothing(t *testing.T) {
	s := twoViews()
	before := *s.shots[0].eye
	s.Point(Frame{Mouse: []Mouse{at(Move, 20, 12), at(Move, 40, 12)}},
		80, 24)
	if *s.shots[0].eye != before {
		t.Errorf("a drag nobody started moved the camera to %v, from"+
			" %v", *s.shots[0].eye, before)
	}
}

// Letting go ends the drag: the pointer crossing the picture afterwards does
// not drag it.
func TestLettingGoEndsTheDrag(t *testing.T) {
	s := twoViews()
	s.Point(Frame{Mouse: []Mouse{at(Press, 10, 12), at(Move, 20, 12),
		at(Release, 20, 12)}}, 80, 24)
	after := *s.shots[0].eye
	s.Point(Frame{Mouse: []Mouse{at(Move, 30, 12), at(Move, 35, 12)}},
		80, 24)
	if *s.shots[0].eye != after {
		t.Errorf("the camera moved after the button came up: %v, was"+
			" %v", *s.shots[0].eye, after)
	}
}

// The distance to what it is turning about does not change while it turns,
// and the camera keeps looking at it. Dragged right round, it comes home.
func TestADragTurnsWithoutWanderingOrLosingItsAim(t *testing.T) {
	s := twoViews()
	eye := s.shots[0].eye
	pivot := eye.Pivot()
	was := eye.Away(pivot)
	s.Point(Frame{Mouse: []Mouse{at(Press, 0, 12)}}, 80, 24)
	for x := 1; x <= 80; x++ {
		s.Point(Frame{Mouse: []Mouse{{Kind: Move, Button: Left,
			Col: x % 40, Row: 12, SubX: x, SubY: 36}}}, 80, 24)
		if d := math.Abs(eye.Away(pivot) - was); d > 1e-9 {
			t.Fatalf("after %d samples it is %g from the pivot,"+
				" was %g", x, eye.Away(pivot), was)
		}
		toward := view.Vec{X: pivot.X - eye.At.X, Y: pivot.Y - eye.At.Y,
			Z: pivot.Z - eye.At.Z}
		f := eye.Forward()
		dot := (toward.X*f.X + toward.Y*f.Y + toward.Z*f.Z) /
			eye.Away(pivot)
		if dot < 1-1e-9 {
			t.Fatalf("after %d samples it is not looking at the"+
				" pivot: %g", x, dot)
		}
	}
}

// The wheel moves the camera under the pointer, toward the scene one way and
// away the other, and leaves the other camera alone.
func TestTheWheelMovesTheCameraUnderThePointer(t *testing.T) {
	s := twoViews()
	near, far := s.shots[0].eye, s.shots[1].eye
	was := near.Away(near.Pivot())
	other := far.Away(far.Pivot())

	in := at(Wheel, 10, 12)
	in.Wheel = -1
	s.Point(Frame{Mouse: []Mouse{in}}, 80, 24)
	closer := near.Away(near.Pivot())
	if closer >= was {
		t.Errorf("the wheel away from the reader left it %g from the"+
			" pivot, was %g; it should be nearer", closer, was)
	}
	if d := math.Abs(far.Away(far.Pivot()) - other); d > 1e-12 {
		t.Error("the wheel moved the camera the pointer was not on")
	}

	out := at(Wheel, 10, 12)
	out.Wheel = 1
	s.Point(Frame{Mouse: []Mouse{out}}, 80, 24)
	if d := math.Abs(near.Away(near.Pivot()) - was); d > 1e-9 {
		t.Errorf("in and out again left it %g from the pivot, was %g",
			near.Away(near.Pivot()), was)
	}
}

// A stage with no viewports of its own is still steerable: that is every
// program written before viewports existed.
func TestAStageWithNoViewportsIsStillSteerable(t *testing.T) {
	s := &Stage{}
	s.eye = view.Eye{At: view.Vec{Z: -6}, Lens: 1}
	before := s.eye
	s.Point(Frame{Mouse: []Mouse{at(Press, 10, 12), at(Move, 30, 12)}},
		80, 24)
	if s.eye == before {
		t.Error("a drag on a stage with no viewports turned nothing")
	}
	// and a press outside the sheet is not a grab
	s = &Stage{}
	s.eye = view.Eye{At: view.Vec{Z: -6}, Lens: 1}
	before = s.eye
	s.Point(Frame{Mouse: []Mouse{at(Press, 200, 12), at(Move, 30, 12)}},
		80, 24)
	if s.eye != before {
		t.Error("a drag that began off the sheet turned the camera")
	}
}
