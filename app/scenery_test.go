package app

import (
	"strings"
	"testing"
	"time"

	"github.com/janearc/miami/cells"
)

// light is how bright a frame of cells is, summed.
func lightOf(cs [][]cells.Cell) float64 {
	sum := 0.0
	for _, row := range cs {
		for _, c := range row {
			sum += float64(c.Fg.R) + float64(c.Fg.G) + float64(c.Fg.B) +
				float64(c.Bg.R) + float64(c.Bg.G) + float64(c.Bg.B)
		}
	}
	return sum
}

// draw is one frame of a stage at t seconds.
func draw(st *Stage, t float64) [][]cells.Cell {
	d := time.Duration(t * float64(time.Second))
	return st.Draw(Frame{At: time.Unix(0, 0).Add(d), Since: d}, 60, 20)
}

// Every scenery scene that ships loads, passes its checks, and draws a
// picture with light in it.
func TestTheSceneryScenesDraw(t *testing.T) {
	for _, name := range []string{"@blackhole", "@quasar", "@hexworld"} {
		st, err := NewStage(name, "")
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if err := st.Validate(); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if l := lightOf(draw(st, 2)); l < 20000 {
			t.Fatalf("%s: a frame with almost no light: %v", name, l)
		}
	}
}

// A look nobody has is refused when the stage is built, naming the ones
// there are, and a block on the wrong kind of shape is refused on read.
func TestSceneryRefusesWhatItCannotDraw(t *testing.T) {
	_, err := stageFrom(t, `{"shapes": [{"kind": "blackhole",
		"hole": {"look": "gargle"}}]}`)
	if err == nil || !strings.Contains(err.Error(), "ember") {
		t.Fatalf("an unknown look: %v", err)
	}
	_, err = stageFrom(t, `{"shapes": [{"kind": "planet",
		"hole": {"look": "ember"}}]}`)
	if err == nil || !strings.Contains(err.Error(), `"hole"`) {
		t.Fatalf("a hole block on a planet: %v", err)
	}
	_, err = stageFrom(t, `{"shapes": [{"kind": "emitter"}]}`)
	if err == nil || !strings.Contains(err.Error(), "emit") {
		t.Fatalf("an emitter with no block: %v", err)
	}
	_, err = stageFrom(t, `{"shapes": [{"kind": "planet",
		"world": {"colours": {"lava": "ff0000"}}}]}`)
	if err == nil || !strings.Contains(err.Error(), "snow") {
		t.Fatalf("an unknown biome: %v", err)
	}
}

// A number bound to time is read every frame: a disk whose brightness
// swings is dimmer at the bottom of the swing than at the top.
func TestSceneryNumbersAreReadEveryFrame(t *testing.T) {
	st, err := stageFrom(t, `{"seconds": 8, "shapes": [
		{"kind": "blackhole", "at": {"z": 44},
		 "hole": {"look": "ember",
		          "bright": {"mid": 1.5, "amp": 1.4, "period": 8}}}]}`)
	if err != nil {
		t.Fatal(err)
	}
	top, bottom := lightOf(draw(st, 2)), lightOf(draw(st, 6))
	if !(top > bottom*1.2) {
		t.Fatalf("at full brightness %v, at least %v", top, bottom)
	}
}

// A scenery number that reads the music is refused on a stage with none, as
// any other binding is, before a frame is drawn.
func TestSceneryReadingMusicNeedsAPlayhead(t *testing.T) {
	st, err := stageFrom(t, `{"shapes": [{"kind": "blackhole",
		"hole": {"bright": "@level"}}]}`)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Validate(); err == nil ||
		!strings.Contains(err.Error(), "level") {
		t.Fatalf("no playhead, and Validate said %v", err)
	}
}
