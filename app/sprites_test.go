package app

import (
	"testing"
	"time"

	sprite "github.com/janearc/libdaffy-sprite"

	"github.com/janearc/miami/cells"
	"github.com/janearc/miami/view"
)

// blob is a small solid sprite standing at one place, for asking whether it
// was drawn.
type blob struct {
	at     view.Vec
	height float64
}

func (b blob) Extent(time.Duration) (view.Vec, float64) { return b.at, b.height }

func (b blob) Render(_ view.Eye, _ time.Duration, rows int) sprite.Frame {
	art := make([]string, max(rows, 1))
	for i := range art {
		art[i] = "WWWW"
	}
	return sprite.Frame{Art: art}
}

// A sprite behind the hole is hidden by it, and the same sprite in front of it
// is not. This is the whole point of sprites going into the light buffer: a
// starling can fly behind the disk.
func TestASpriteGoesBehindTheHole(t *testing.T) {
	hole := `{"shapes": [{"kind": "blackhole", "at": {"z": 40},
		"hole": {"look": "ember", "outer": 12}}]}`
	frame := func(at *view.Vec) [][]cells.Cell {
		st, err := stageFrom(t, hole)
		if err != nil {
			t.Fatal(err)
		}
		if at != nil {
			st.Objects = []view.Object{blob{at: *at, height: 3}}
		}
		return st.Draw(Frame{At: time.Unix(0, 0)}, 60, 20)
	}
	bare := frame(nil)
	behind := frame(&view.Vec{Z: 70})
	front := frame(&view.Vec{Z: 12})
	if !same(bare, behind) {
		t.Error("a sprite behind the hole was drawn over its shadow")
	}
	if same(bare, front) {
		t.Error("a sprite in front of the hole was not drawn")
	}
}

// The stage draws sprites through its own camera, so turning the camera moves
// the sprite with the scene rather than leaving it nailed to the origin.
//
// The sky here is empty on purpose. Against stars the frames differ whatever
// the sprite does, because the stars turn with the camera, and the test would
// pass with the sprite nailed down.
func TestTheCameraTurnsTheSpriteToo(t *testing.T) {
	st, err := stageFrom(t, `{"shapes": [{"kind": "space", "space":
		{"stars": 0, "nebula": 0, "dust": 0, "glow": 0}}]}`)
	if err != nil {
		t.Fatal(err)
	}
	st.Objects = []view.Object{blob{at: view.Vec{Z: 6}, height: 1}}
	still := st.Draw(Frame{At: time.Unix(0, 0)}, 60, 20)
	st.Eye().Yaw = 0.25
	turned := st.Draw(Frame{At: time.Unix(0, 0)}, 60, 20)
	if same(still, turned) {
		t.Fatal("the camera turned and the sprite did not move")
	}
}

// same is whether two frames of cells are the same picture.
func same(a, b [][]cells.Cell) bool {
	for y := range a {
		for x := range a[y] {
			if a[y][x] != b[y][x] {
				return false
			}
		}
	}
	return true
}
