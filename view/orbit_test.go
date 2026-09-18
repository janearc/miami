package view

import (
	"math"
	"testing"
)

// looking is an eye a distance from the origin, looking at it.
func looking(r float64) Eye {
	return Eye{At: Vec{Z: -r}, Lens: 1}
}

// A full turn comes home.
//
// Three hundred and sixty steps of a degree, and the camera is where it
// started. Nothing here is allowed to be nearly right: an orbit that loses a
// thousandth of a degree a step is an orbit that slides while a hand holds it
// still, which is exactly the fault that is hard to see and impossible to
// unsee.
func TestAFullTurnComesHome(t *testing.T) {
	start := looking(12)
	pivot := start.Pivot()
	e := start
	step := 2 * math.Pi / 360
	for range 360 {
		e = e.Turned(pivot, step, 0)
	}
	if d := far(sub(e.At, start.At)); d > 1e-9 {
		t.Errorf("after a full turn the camera is %g from where it"+
			" started: %v, was %v", d, e.At, start.At)
	}
	if d := math.Abs(e.Yaw - start.Yaw - 2*math.Pi); d > 1e-12 {
		t.Errorf("yaw came home to %g, want %g", e.Yaw,
			start.Yaw+2*math.Pi)
	}
}

// The distance to the pivot does not drift over a thousand small steps.
//
// A drag is a thousand small steps: a hand crossing the screen sends a report
// every few milliseconds for as long as it takes. If each step rebuilt the
// position by adding to it, the distance would wander, and a picture would
// creep toward or away from the thing being turned while the hand went round
// in circles.
func TestTheDistanceDoesNotDriftOverAThousandSteps(t *testing.T) {
	e := looking(7.5)
	pivot := e.Pivot()
	want := e.Away(pivot)
	worst := 0.0
	for i := range 1000 {
		// a hand does not drag in a straight line
		az := 0.004 * math.Cos(float64(i)/37)
		el := 0.003 * math.Sin(float64(i)/23)
		e = e.Turned(pivot, az, el)
		worst = math.Max(worst, math.Abs(e.Away(pivot)-want))
	}
	if worst > 1e-9 {
		t.Errorf("the distance wandered by %g over a thousand steps,"+
			" from %g", worst, want)
	}
}

// And the camera is still looking at the pivot afterwards, which is the other
// half of the same claim: a turn that keeps the distance but loses the aim is
// a camera sliding round a wall.
func TestATurnedCameraStillLooksAtThePivot(t *testing.T) {
	e := looking(4)
	pivot := e.Pivot()
	for i := range 200 {
		e = e.Turned(pivot, 0.05, 0.02*math.Cos(float64(i)/11))
		toward := unit(sub(pivot, e.At))
		f := e.Forward()
		if d := dot(toward, f); d < 1-1e-12 {
			t.Fatalf("step %d: the camera looks %v and the pivot"+
				" is %v away from it: dot %g", i, f, toward, d)
		}
	}
}

// Dragged to the top, the camera stops short of the pole rather than going
// over it.
func TestAnElevationStopsShortOfThePole(t *testing.T) {
	for _, way := range []float64{1, -1} {
		e := looking(3)
		pivot := e.Pivot()
		for range 500 {
			e = e.Turned(pivot, 0, way*0.05)
		}
		if math.Abs(e.Pitch) >= math.Pi/2 {
			t.Errorf("dragged %g, the pitch reached %g", way,
				e.Pitch)
		}
		// and it is still a camera: still that far away, still aimed
		if d := math.Abs(e.Away(pivot) - 3); d > 1e-9 {
			t.Errorf("at the pole the distance became %g",
				e.Away(pivot))
		}
	}
}

// The wheel never reaches the pivot, however long it is held.
func TestTheWheelNeverArrives(t *testing.T) {
	e := looking(9)
	pivot := e.Pivot()
	before := e.Forward()
	for range 2000 {
		e = e.Dollied(pivot, 0.9)
	}
	r := e.Away(pivot)
	if r <= 0 {
		t.Fatalf("the camera arrived at the pivot: %g away", r)
	}
	if r != closest {
		t.Errorf("two thousand notches left it %g away; it should be"+
			" sitting on the floor at %g", r, closest)
	}
	// and it did not pass through and come out the other side facing
	// backwards, which is what an add rather than a multiply does
	if d := dot(e.Forward(), before); d < 1-1e-12 {
		t.Errorf("the camera turned round on the way in: %g", d)
	}
}

// In and out again is where it started, as long as the floor was never
// reached. Once it is, the distance is forgotten and cannot come back: that is
// what a floor is, and it is why this is a separate test from the one above
// rather than the same one run further.
func TestTheWheelInAndOutAgainComesHome(t *testing.T) {
	e := looking(9)
	pivot := e.Pivot()
	const notches = 50 // 9 * 0.9^50 is 0.0026, which is above the floor
	for range notches {
		e = e.Dollied(pivot, 0.9)
	}
	if e.Away(pivot) <= closest {
		t.Fatal("the test reached the floor; it proves nothing")
	}
	for range notches {
		e = e.Dollied(pivot, 1/0.9)
	}
	if d := math.Abs(e.Away(pivot) - 9); d > 1e-9 {
		t.Errorf("in and out again left it %g away, not 9",
			e.Away(pivot))
	}
}

// A camera looking at the middle of the scene orbits the middle of the scene.
func TestThePivotOfACameraLookingAtTheOriginIsTheOrigin(t *testing.T) {
	for _, r := range []float64{1, 12, 400} {
		if d := far(looking(r).Pivot()); d > 1e-9 {
			t.Errorf("looking at the origin from %g, the pivot is"+
				" %g away from it", r, d)
		}
	}
}

// A camera off to one side turns about the point it is looking past, which is
// the nearest point on its line of sight to the middle, and not simply the
// middle or its own distance from it.
//
// Every other test here looks straight at the origin, where the nearest point
// on the line of sight and the distance to the middle are the same number, so
// none of them can tell a right answer from a wrong one that agrees there.
func TestThePivotIsTheNearestPointOnTheLineOfSight(t *testing.T) {
	// five to the side, ten back, looking straight down Z: it looks past
	// the origin, and the point it passes is level with it
	e := Eye{At: Vec{X: 5, Z: -10}, Lens: 1}
	pivot := e.Pivot()
	if far(sub(pivot, Vec{X: 5})) > 1e-12 {
		t.Errorf("the pivot is %v, want (5 0 0): the point on the"+
			" line of sight beside the middle", pivot)
	}
	// which is the same as saying it is square to where the camera looks
	if d := math.Abs(dot(pivot, e.Forward())); d > 1e-12 {
		t.Errorf("the pivot is not the nearest point: it lies %g"+
			" along the line of sight from the middle", d)
	}
	// and turning about it keeps the camera ten away from it, not
	// eleven-and-a-fifth
	if d := math.Abs(e.Away(pivot) - 10); d > 1e-12 {
		t.Errorf("the camera is %g from its pivot, want 10",
			e.Away(pivot))
	}
}

// A camera looking away from the middle still has something to turn about,
// rather than orbiting a point behind its own shoulder.
func TestACameraLookingAwayOrbitsWhatItCanSee(t *testing.T) {
	e := Eye{At: Vec{Z: -10}, Lens: 1, Yaw: math.Pi} // turned right round
	pivot := e.Pivot()
	if d := dot(unit(sub(pivot, e.At)), e.Forward()); d < 1-1e-12 {
		t.Fatalf("the pivot is behind the camera: %v, looking %v",
			pivot, e.Forward())
	}
	if r := e.Away(pivot); r <= closest {
		t.Fatalf("the pivot is on top of the camera: %g away", r)
	}
}

// A drag across the whole picture is half a turn, and a drag of nothing turns
// nothing.
func TestADragAcrossThePictureIsHalfATurn(t *testing.T) {
	az, _ := Drag(480, 0, 480)
	if math.Abs(az-math.Pi) > 1e-12 {
		t.Errorf("a drag across 480 subpixels turned %g, want %g", az,
			math.Pi)
	}
	if az, el := Drag(0, 0, 480); az != 0 || el != 0 {
		t.Errorf("a drag of nothing turned %g, %g", az, el)
	}
	if az, el := Drag(10, 10, 0); az != 0 || el != 0 {
		t.Errorf("a picture of no width turned %g, %g", az, el)
	}
}
