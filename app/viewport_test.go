package app

import (
	"testing"

	"github.com/janearc/miami/scene"
	"github.com/janearc/miami/view"
)

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
