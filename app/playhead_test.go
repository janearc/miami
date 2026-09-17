package app

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fakeHead is a piece at a hundred and twenty beats a minute, four to the bar,
// four seconds long and not looping. It is a function of the offset and nothing
// else, which is the property the real one is meant to have.
type fakeHead struct{}

func (fakeHead) At(
	offset time.Duration,
) (beat, bar, beats, level float64, ok bool) {
	if offset < 0 || offset >= 4*time.Second {
		return 0, 0, 0, 0, false
	}
	beats = offset.Seconds() * 2
	return frac(beats), frac(beats / 4), beats, 0.5, true
}

func frac(v float64) float64 { return v - math.Floor(v) }

// bounceDoc is a scene whose only moving part is a bar bound to a bounce on the
// beat, written with the arithmetic bindings already had.
const bounceDoc = `{
  "name": "bounce", "seconds": 60,
  "shapes": [
    { "kind": "bar", "name": "b", "row": 1,
      "value": {"mul": [4, "@beat", {"add": [1, {"mul": [-1, "@beat"]}]}]} }
  ]
}`

// stageFrom writes a document where NewStage can read it.
func stageFrom(t *testing.T, doc string) (*Stage, error) {
	t.Helper()
	p := filepath.Join(t.TempDir(), "scene.json")
	if err := os.WriteFile(p, []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	return NewStage(p, "@starling")
}

// drawAt draws one small frame at an offset from a fixed start.
func drawAt(s *Stage, at time.Time) {
	s.Draw(Frame{At: at}, 12, 4)
}

var t0 = time.Date(2026, 9, 16, 21, 0, 0, 0, time.UTC)

// near is equality for floats that went through arithmetic.
func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

// TestPlayheadDrivesTheScene is the whole seam: an offset goes in, a document
// binding comes out the other side with no Go written for it.
func TestPlayheadDrivesTheScene(t *testing.T) {
	s, err := stageFrom(t, bounceDoc)
	if err != nil {
		t.Fatal(err)
	}
	s.Music, s.Started = fakeHead{}, t0

	drawAt(s, t0.Add(1250*time.Millisecond))
	c := s.at.Const
	if !near(c["beats"], 2.5) || !near(c["beat"], 0.5) ||
		!near(c["bar"], 0.625) || !near(c["level"], 0.5) {
		t.Fatalf("at 1.25s: got %v", c)
	}
	// and the bounce is at the top of its arc, half way through a beat
	if got := s.Bar("b").From(Frame{Scene: s.at}); !near(got, 1) {
		t.Errorf("bounce mid-beat: got %v, want 1", got)
	}
}

// TestLatencyComesOffTheOffset checks the one correction the clock has.
func TestLatencyComesOffTheOffset(t *testing.T) {
	s, err := stageFrom(t, bounceDoc)
	if err != nil {
		t.Fatal(err)
	}
	s.Music, s.Started, s.Latency = fakeHead{}, t0, 250*time.Millisecond
	drawAt(s, t0.Add(1250*time.Millisecond))
	if got := s.at.Const["beats"]; !near(got, 2) {
		t.Fatalf("beats with 250ms latency: got %v, want 2", got)
	}
}

// TestStoppingHoldsBeatsAndRestsTheRest checks what "not playing" means here.
//
// A ball rests on the floor and silence is silence, so beat, bar and level go
// to nought. Beats does not: it is for things that roll, and one that snapped
// back to its start when the music ended would look like a fault.
func TestStoppingHoldsBeatsAndRestsTheRest(t *testing.T) {
	s, err := stageFrom(t, bounceDoc)
	if err != nil {
		t.Fatal(err)
	}
	s.Music, s.Started = fakeHead{}, t0
	drawAt(s, t0.Add(3500*time.Millisecond)) // beats 7, still playing
	drawAt(s, t0.Add(5*time.Second))         // past the end
	c := s.at.Const
	if c["beat"] != 0 || c["bar"] != 0 || c["level"] != 0 {
		t.Errorf("past the end: beat %v bar %v level %v, want noughts",
			c["beat"], c["bar"], c["level"])
	}
	if !near(c["beats"], 7) {
		t.Errorf("past the end: beats %v, want 7 held", c["beats"])
	}
}

// TestZeroStartedMeansTheFirstFrame checks the default a game gets for free.
func TestZeroStartedMeansTheFirstFrame(t *testing.T) {
	s, err := stageFrom(t, bounceDoc)
	if err != nil {
		t.Fatal(err)
	}
	s.Music = fakeHead{}
	drawAt(s, t0)
	if !s.Started.Equal(t0) {
		t.Fatalf("started %v, want the first frame's %v", s.Started, t0)
	}
	drawAt(s, t0.Add(500*time.Millisecond))
	if got := s.at.Const["beats"]; !near(got, 1) {
		t.Errorf("half a second in: beats %v, want 1", got)
	}
}

// TestAnUnknownNameIsRefusedAtLoad is the silent nought, turned into an error
// with the name in it.
func TestAnUnknownNameIsRefusedAtLoad(t *testing.T) {
	_, err := stageFrom(t, `{"name": "x", "shapes": [
	  {"kind": "bar", "name": "b", "value": "@nonsense"}]}`)
	if err == nil || !strings.Contains(err.Error(), "@nonsense") {
		t.Fatalf("got %v, want a refusal naming @nonsense", err)
	}
}

// TestMusicWithoutAPlayheadIsRefusedBeforeDrawing checks the refusal lands
// before the terminal is touched, not on a screen that has been cleared.
func TestMusicWithoutAPlayheadIsRefusedBeforeDrawing(t *testing.T) {
	s, err := stageFrom(t, bounceDoc)
	if err != nil {
		t.Fatalf("a document reading @beat must load: %v", err)
	}
	if err := s.Validate(); err == nil ||
		!strings.Contains(err.Error(), "@beat") {
		t.Fatalf("validate: got %v, want a refusal naming @beat", err)
	}

	out, err := os.Create(filepath.Join(t.TempDir(), "terminal"))
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	err = Screen{Cols: 12, Rows: 4, Out: out}.Run(s,
		func(Frame) bool { return false })
	if err == nil {
		t.Fatal("Run drew a scene that reads @beat with no playhead")
	}
	if fi, _ := out.Stat(); fi.Size() != 0 {
		t.Errorf("Run wrote %d bytes before refusing", fi.Size())
	}

	s.Music = fakeHead{}
	if err := s.Validate(); err != nil {
		t.Errorf("with a playhead attached: %v", err)
	}
}

// TestWordsAreNotNames checks a line of type that happens to start with "@".
func TestWordsAreNotNames(t *testing.T) {
	if _, err := stageFrom(t, `{"name": "x", "shapes": [
	  {"kind": "text", "name": "t", "text": "@home"}]}`); err != nil {
		t.Fatalf("a text saying @home was refused: %v", err)
	}
}

// TestTheBuiltInScenesStillLoad guards the scenes this library ships against
// the check that was added after them.
func TestTheBuiltInScenesStillLoad(t *testing.T) {
	for _, doc := range []string{"@clock", "@build"} {
		if _, err := NewStage(doc, "@starling"); err != nil {
			t.Errorf("%s: %v", doc, err)
		}
	}
}

// TestPlayStartsAndStops runs the player on something that is not a speaker.
func TestPlayStartsAndStops(t *testing.T) {
	was := player
	defer func() { player = was }()
	player = "sh"

	script := filepath.Join(t.TempDir(), "piece")
	if err := os.WriteFile(script, []byte("sleep 5\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	before := time.Now()
	started, stop, err := Play(script)
	if err != nil {
		t.Fatal(err)
	}
	if started.Before(before) {
		t.Errorf("started %v, before Play was called at %v",
			started, before)
	}
	stop()
	stop() // twice is fine

	if _, _, err := Play(filepath.Join(t.TempDir(), "absent")); err == nil {
		t.Error("a missing file started")
	}
}
