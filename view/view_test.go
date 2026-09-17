package view

import (
	"math"
	"testing"
	"time"

	sprite "github.com/janearc/libdaffy-sprite"
)

func box(w, h int) sprite.Frame {
	art := make([]string, h)
	for i := range art {
		row := make([]byte, w)
		for j := range row {
			row[j] = 'K'
		}
		art[i] = string(row)
	}
	return sprite.Frame{Art: art}
}

// The same object further away must come back smaller, and in the right order.
// This is the whole claim of the package: distance is handled by asking, not by
// the caller doing arithmetic.
func TestFurtherIsSmallerAndBehind(t *testing.T) {
	f := []sprite.Frame{box(8, 8)}
	near := Billboard{Frames: f, At: Vec{Z: 1}, Height: 0.5}
	far := Billboard{Frames: f, At: Vec{Z: 4}, Height: 0.5}
	got := Resolve([]Object{near, far}, DefaultEye(), 0, 200, 90)
	if len(got) != 2 {
		t.Fatalf("resolved %d objects", len(got))
	}
	// far first: painting order is the whole of occlusion
	if got[0].Depth <= got[1].Depth {
		t.Errorf(
			"not sorted far to near: %v then %v", got[0].Depth,
			got[1].Depth,
		)
	}
	if got[0].H >= got[1].H {
		t.Errorf(
			"the far one is not smaller: %d vs %d", got[0].H,
			got[1].H,
		)
	}
}

// Behind the viewer is not drawn at all, rather than drawn inside out.
func TestBehindIsDropped(t *testing.T) {
	f := []sprite.Frame{box(8, 8)}
	b := Billboard{Frames: f, At: Vec{Z: -2}, Height: 0.5}
	got := Resolve([]Object{b}, DefaultEye(), 0, 200, 90)
	if len(got) != 0 {
		t.Errorf("an object behind the eye was resolved")
	}
}

// Scaling a stencil keeps it a stencil: no role is invented, and the silhouette
// survives being made small.
func TestScaleKeepsTheAlphabet(t *testing.T) {
	small := Scale(box(30, 30), 5, 5)
	if small.Width() != 5 || small.Height() != 5 {
		t.Fatalf("scaled to %dx%d", small.Width(), small.Height())
	}
	for _, row := range small.Art {
		for i := 0; i < len(row); i++ {
			if row[i] != 'K' {
				t.Fatalf("scaling invented %q", row[i])
			}
		}
	}
}

// A turning sprite hands back a different pose when looked at from a different
// side, without the caller saying which.
func TestTurningPicksAPose(t *testing.T) {
	// four poses, and the object faces the camera: heading pi, because
	// nought points towards positive Z and the camera is at lower Z
	poses := [][]sprite.Frame{
		{box(4, 4)}, {box(5, 5)}, {box(6, 6)}, {box(7, 7)},
	}
	s := Turning{Poses: poses, At: Vec{Z: 4}, Height: 1, Heading: math.Pi}

	// the pose index is the assertion, not the frame size: Render scales
	// every pose to the rows it was asked for, so two square poses come
	// back the same width however different they are
	front := Eye{At: Vec{Z: 0}, Lens: 1}
	side := Eye{At: Vec{X: 4, Z: 4}, Lens: 1}
	if got := s.Pose(front); got != 0 {
		t.Errorf("head on should be pose zero, got %d", got)
	}
	if s.Pose(side) == s.Pose(front) {
		t.Errorf("the same pose was returned from two directions")
	}
	if f := s.Render(front, 0, 8); f.Height() != 8 {
		t.Errorf("asked for 8 rows, got %d", f.Height())
	}
	_ = time.Second
}
