package app

import (
	"fmt"
	"testing"

	"github.com/janearc/miami/cells"
	"github.com/janearc/miami/scene"
	"github.com/janearc/miami/view"
)

// hole is a black hole and nothing else, with room left for whatever a test
// wants to say about cameras. Small enough that a bake is quick.
const hole = `{"name":"hole","seconds":10,%s
  "shapes":[{"kind":"blackhole","at":{"x":-4,"y":-1,"z":62},
             "hole":{"look":"quasar"}}]}`

// Two packages each name the main camera because neither imports the other.
// This is the only thing that keeps them the same word.
func TestTheMainCameraIsOneName(t *testing.T) {
	if scene.MainCamera != view.MainCamera {
		t.Fatalf("scene says %q and view says %q",
			scene.MainCamera, view.MainCamera)
	}
}

// Viewports that meet in a document meet on the sheet: no column between
// them that nobody draws, and none that both do, at any terminal width.
func TestViewportsMeetWithNoGap(t *testing.T) {
	for _, parts := range []int{2, 3, 5} {
		for cols := 1; cols <= 200; cols++ {
			at := 0
			for i := 0; i < parts; i++ {
				v := shot{x: float64(i) / float64(parts), y: 0,
					w: 1 / float64(parts), h: 1}
				x0, _, x1, _ := v.cells(cols, 10)
				if x0 != at {
					t.Fatalf("%d parts, %d cols: part %d starts at "+
						"%d, the one before ended at %d",
						parts, cols, i, x0, at)
				}
				at = x1
			}
			if at != cols {
				t.Fatalf("%d parts, %d cols: they end at %d",
					parts, cols, at)
			}
		}
	}
}

// A document from before viewports is one shot filling the sheet, which is
// the path that draws exactly what it drew before.
func TestAnOldDocumentIsOneShot(t *testing.T) {
	st, err := NewStage("@quasar", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(st.shots) != 1 || !st.shots[0].fills() || !st.shots[0].first {
		t.Fatalf("shots = %+v, want one, the first camera's, filling",
			st.shots)
	}
}

// Two cameras on one scene are two different pictures side by side, and
// both have light in them.
func TestTwoCamerasAreTwoPictures(t *testing.T) {
	st, err := stageFrom(t, fmt.Sprintf(hole, `
	  "eye":{"lens":1,"near":0.05},
	  "cameras":[{"name":"aside","lens":1,"near":0.05,"yaw":0.12}],
	  "viewports":[{"camera":"main","x":0,"y":0,"w":0.5,"h":1},
	               {"camera":"aside","x":0.5,"y":0,"w":0.5,"h":1}],`))
	if err != nil {
		t.Fatal(err)
	}
	cs := draw(st, 1)
	left, right := half(cs, 0, 30), half(cs, 30, 60)
	if lightOf(left) < 5000 || lightOf(right) < 5000 {
		t.Fatalf("a dark half: left %v, right %v",
			lightOf(left), lightOf(right))
	}
	if fmt.Sprint(left) == fmt.Sprint(right) {
		t.Fatal("two cameras drew the same picture")
	}
}

// A document that says exposure nought means a black frame. Read as "no
// opinion" it would come back at full strength, which is the trap
// Layer.Gain fell into; the same scene without the word is the control.
func TestNoughtExposureIsBlack(t *testing.T) {
	lit, err := stageFrom(t, fmt.Sprintf(hole,
		`"cameras":[{"name":"c","lens":1,"near":0.05}],`))
	if err != nil {
		t.Fatal(err)
	}
	dark, err := stageFrom(t, fmt.Sprintf(hole,
		`"cameras":[{"name":"c","lens":1,"near":0.05,"exposure":0}],`))
	if err != nil {
		t.Fatal(err)
	}
	if l := lightOf(draw(lit, 1)); l < 20000 {
		t.Fatalf("the control has almost no light: %v", l)
	}
	if l := lightOf(draw(dark, 1)); l != 0 {
		t.Fatalf("exposure nought left %v of light on the screen", l)
	}
}

// A glow keeps its size in the scene: half the sheet, half the columns, and
// never under one column on the way down.
func TestAGlowScalesWithItsViewport(t *testing.T) {
	for _, tc := range []struct{ scale, r, want float64 }{
		{1, 8, 8}, {0.5, 8, 4}, {0.5, 1.5, 1}, {0.25, 24, 6},
		{0.5, 0.5, 0.5},
	} {
		if got := (look{scale: tc.scale}).radius(tc.r); got != tc.want {
			t.Errorf("radius %v at scale %v = %v, want %v",
				tc.r, tc.scale, got, tc.want)
		}
	}
}

// half is the columns from a up to b of every row.
func half(cs [][]cells.Cell, a, b int) [][]cells.Cell {
	out := make([][]cells.Cell, len(cs))
	for y, row := range cs {
		out[y] = row[a:b]
	}
	return out
}
