package app

import (
	"errors"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
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

// TestScoringWhileRunIsDrawing is a game starting its music on a key press.
//
// Run blocks, so the attach comes from another goroutine while frames are being
// drawn, and the scene reads @beat from the first frame -- which Run refuses
// unless something is attached, so Silence is attached first. Run it under
// -race: that is the half of this test that fails on a stage without a lock.
func TestScoringWhileRunIsDrawing(t *testing.T) {
	was := player
	defer func() { player = was }()
	player = "sh"
	script := filepath.Join(t.TempDir(), "piece")
	if err := os.WriteFile(script, []byte("sleep 5\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := stageFrom(t, bounceDoc)
	if err != nil {
		t.Fatal(err)
	}
	s.Music = Silence // music is coming; the scene reads @beat already
	out, err := os.Create(filepath.Join(t.TempDir(), "terminal"))
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()

	done := make(chan error, 1)
	go func() {
		n := 0
		done <- Screen{Cols: 12, Rows: 4, FPS: 200, Out: out}.Run(s,
			func(Frame) bool { n++; return n < 80 })
	}()
	// attached several times over, not once. the race detector only
	// reports an interleaving that happened, and Draw takes the lock for its
	// overlays a moment after it reads the music, which can order a single
	// attach and hide the race -- once in three runs, measured. five
	// attaches make that a one in two hundred and forty-three.
	for i := 0; i < 5; i++ {
		time.Sleep(12 * time.Millisecond)
		stop, err := s.Score(script, fakeHead{}, 0)
		if err != nil {
			t.Fatal(err)
		}
		defer stop()
	}
	if err := <-done; err != nil {
		t.Fatalf("Run refused a scene with Silence attached: %v", err)
	}
	// the frames after the key press moved to the piece
	if got := s.at.Const["beats"]; got <= 0 {
		t.Errorf("beats after scoring mid-run: %v, want above nought", got)
	}
}

// TestALoopingPiecePlaysAgain checks the sound comes back round, rather than
// playing once under a playhead that counts on into silence.
func TestALoopingPiecePlaysAgain(t *testing.T) {
	was := player
	defer func() { player = was }()
	player = "sh"
	dir := t.TempDir()
	count := filepath.Join(dir, "starts")
	script := filepath.Join(dir, "piece")
	body := "echo x >> " + count + "\nsleep 5\n"
	if err := os.WriteFile(script, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	starts := func() int {
		b, _ := os.ReadFile(count)
		return strings.Count(string(b), "x")
	}

	_, stop, err := playing(script, 80*time.Millisecond, nil)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(300 * time.Millisecond)
	stop()
	n := starts()
	if n < 3 {
		t.Fatalf("a piece looping every 80ms started %d times in 300ms", n)
	}
	time.Sleep(200 * time.Millisecond)
	if again := starts(); again != n {
		t.Errorf("started %d more times after stop", again-n)
	}
}

// TestRestartsDoNotDrift pins the schedule: restarts sit on whole periods after
// the first start however late the wake-ups are, and a wake-up that misses one
// entirely skips it rather than playing twice to catch up.
func TestRestartsDoNotDrift(t *testing.T) {
	period := 2 * time.Second
	for _, c := range []struct{ now, want time.Duration }{
		{0, 2 * time.Second},
		{1500 * time.Millisecond, 2 * time.Second},
		{2 * time.Second, 4 * time.Second},         // exactly on one: the next
		{2001 * time.Millisecond, 4 * time.Second}, // a late wake-up
		{7 * time.Second, 8 * time.Second},         // missed two: skip to the third
	} {
		got := nextStart(t0, period, t0.Add(c.now))
		if want := t0.Add(c.want); !got.Equal(want) {
			t.Errorf("at %v: next %v, want %v",
				c.now, got.Sub(t0), c.want)
		}
	}
}

// TestAConstMayNotShadowTheRuntime checks a document cannot declare a name the
// runtime supplies and then lose it quietly when the runtime arrives.
func TestAConstMayNotShadowTheRuntime(t *testing.T) {
	for _, name := range []string{"beat", "daylight", "t"} {
		doc := `{"name": "x", "const": {"` + name + `": 1},
		  "shapes": [{"kind": "text", "name": "t", "text": "hi"}]}`
		_, err := stageFrom(t, doc)
		if err == nil || !strings.Contains(err.Error(), name) {
			t.Errorf("const %q: got %v, want a refusal naming it",
				name, err)
		}
	}
}

// otherHead is a second playhead, told apart from fakeHead by its type.
type otherHead struct{ fakeHead }

// brokenPlayer is a player that works once and then cannot start again: it is
// deleted the moment the first pass is under way.
func brokenPlayer(t *testing.T) (piece string, breakIt func()) {
	t.Helper()
	dir := t.TempDir()
	bin := filepath.Join(dir, "player")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\nsleep 5\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	piece = filepath.Join(dir, "piece")
	if err := os.WriteFile(piece, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	player = bin
	return piece, func() { os.Remove(bin) }
}

// TestAFailedRestartRestsTheScene is the loop's error path. The music stopping
// silently while its playhead counts on is the fault looping was added to
// prevent, and it must not come back through the door marked "error".
func TestAFailedRestartRestsTheScene(t *testing.T) {
	was := player
	defer func() { player = was }()
	piece, breakIt := brokenPlayer(t)

	s, err := stageFrom(t, bounceDoc)
	if err != nil {
		t.Fatal(err)
	}
	stop, err := s.Score(piece, fakeHead{}, 60*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	breakIt()
	time.Sleep(250 * time.Millisecond)

	s.mu.Lock()
	_, resting := s.Music.(silence)
	s.mu.Unlock()
	if !resting {
		t.Error("a failed restart left the scene moving to stopped music")
	}
	if err := stop(); err == nil {
		t.Error("stop did not say a restart had failed")
	}
}

// TestAFailedRestartLeavesALaterPieceAlone checks the rest only replaces the
// piece that failed. A game that has already scored something else keeps it.
func TestAFailedRestartLeavesALaterPieceAlone(t *testing.T) {
	was := player
	defer func() { player = was }()
	piece, breakIt := brokenPlayer(t)

	s, err := stageFrom(t, bounceDoc)
	if err != nil {
		t.Fatal(err)
	}
	stopA, err := s.Score(piece, fakeHead{}, 60*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	defer stopA()

	player = "sh"
	script := filepath.Join(t.TempDir(), "next")
	if err := os.WriteFile(script, []byte("sleep 5\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stopB, err := s.Score(script, otherHead{}, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer stopB()

	breakIt()
	time.Sleep(250 * time.Millisecond)
	s.mu.Lock()
	_, still := s.Music.(otherHead)
	s.mu.Unlock()
	if !still {
		t.Error("the first piece's failure replaced the piece scored after it")
	}
}

// TestARestartLeavesThePreviousPassPlaying pins the seam. Each pass records its
// process; a restart must leave the earlier ones running, and stop must end
// every one of them.
//
// Killing the previous pass at the restart cut the last latency's worth of
// audio off a piece exactly one period long, mid-waveform. Nobody here can hear
// that click, so the test watches the processes instead.
func TestARestartLeavesThePreviousPassPlaying(t *testing.T) {
	was := player
	defer func() { player = was }()
	player = "sh"
	dir := t.TempDir()
	pids := filepath.Join(dir, "pids")
	script := filepath.Join(dir, "piece")
	body := "echo $$ >> " + pids + "\nsleep 5\n"
	if err := os.WriteFile(script, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	alive := func() (running, total int) {
		b, _ := os.ReadFile(pids)
		for _, f := range strings.Fields(string(b)) {
			pid, err := strconv.Atoi(f)
			if err != nil {
				continue
			}
			total++
			if syscall.Kill(pid, 0) == nil {
				running++
			}
		}
		return running, total
	}

	_, stop, err := playing(script, 60*time.Millisecond, nil)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(220 * time.Millisecond)
	running, total := alive()
	if total < 3 {
		t.Fatalf("only %d passes started in 220ms at a 60ms period", total)
	}
	if running != total {
		t.Errorf("%d of %d passes still playing after restarts; "+
			"a restart ended the pass before it", running, total)
	}

	// stop has to end the passes, not outlast them. it waits for every
	// exit before it returns, so a stop that killed nothing still returns
	// with every pass gone -- five seconds later, when each one has
	// finished by itself. time is the only thing that tells the two apart.
	began := time.Now()
	stop()
	if took := time.Since(began); took > time.Second {
		t.Errorf("stop took %v: it waited for the passes to finish "+
			"rather than ending them", took)
	}
	if running, total := alive(); running != 0 {
		t.Errorf("stop left %d of %d passes playing", running, total)
	}
}

// TestAFailureBeforeTheAttachStillRests puts a restart's failure in the gap
// between the sound starting and Score attaching the piece. No real period is
// short enough to land there, which is exactly why the timing cannot be the
// thing a test relies on: the fake fails before it even returns.
func TestAFailureBeforeTheAttachStillRests(t *testing.T) {
	was := play
	defer func() { play = was }()
	play = func(_ string, _ time.Duration, onFail func(error)) (
		time.Time, func() error, error) {
		onFail(errors.New("gone before the attach"))
		return time.Now(), func() error { return nil }, nil
	}

	s, err := stageFrom(t, bounceDoc)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Score("piece", fakeHead{}, time.Second); err != nil {
		t.Fatal(err)
	}
	s.mu.Lock()
	_, resting := s.Music.(silence)
	s.mu.Unlock()
	if !resting {
		t.Error("a piece that failed before it was attached was attached")
	}
}

// TestAScoreThatCannotStartLeavesThePlayingPieceAbleToRest checks the other
// way reserving the generation could go wrong. A Score that fails to start at
// all must not move the generation on, or the piece already playing could no
// longer be put to rest when its own restart fails.
func TestAScoreThatCannotStartLeavesThePlayingPieceAbleToRest(t *testing.T) {
	was := play
	defer func() { play = was }()

	var failA func(error)
	play = func(_ string, _ time.Duration, onFail func(error)) (
		time.Time, func() error, error) {
		failA = onFail
		return time.Now(), func() error { return nil }, nil
	}
	s, err := stageFrom(t, bounceDoc)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Score("a", fakeHead{}, time.Second); err != nil {
		t.Fatal(err)
	}

	play = func(string, time.Duration, func(error)) (
		time.Time, func() error, error) {
		return time.Time{}, nil, errors.New("no such file")
	}
	if _, err := s.Score("b", otherHead{}, 0); err == nil {
		t.Fatal("a Score that could not start returned no error")
	}

	failA(errors.New("a's restart failed"))
	s.mu.Lock()
	_, resting := s.Music.(silence)
	s.mu.Unlock()
	if !resting {
		t.Error("a failed Score stopped the playing piece from resting")
	}
}

// TestTheLaterScoreWinsARace holds the first Score inside its player until a
// second has attached, which is the one ordering that inverted: the older
// piece attaching last, winding the generation back over the newer.
func TestTheLaterScoreWinsARace(t *testing.T) {
	was := play
	defer func() { play = was }()

	entered, release := make(chan struct{}), make(chan struct{})
	var mu sync.Mutex
	stoppedA := false
	play = func(path string, _ time.Duration, _ func(error)) (
		time.Time, func() error, error) {
		if path != "a" {
			return time.Now(), func() error { return nil }, nil
		}
		close(entered) // a has reserved its generation by now
		<-release
		return time.Now(), func() error {
			mu.Lock()
			stoppedA = true
			mu.Unlock()
			return nil
		}, nil
	}

	s, err := stageFrom(t, bounceDoc)
	if err != nil {
		t.Fatal(err)
	}
	errA := make(chan error, 1)
	go func() {
		_, err := s.Score("a", fakeHead{}, 0)
		errA <- err
	}()
	<-entered
	if _, err := s.Score("b", otherHead{}, 0); err != nil {
		t.Fatal(err)
	}
	close(release)

	if err := <-errA; !errors.Is(err, ErrSuperseded) {
		t.Errorf("the earlier Score returned %v, want ErrSuperseded", err)
	}
	s.mu.Lock()
	_, later := s.Music.(otherHead)
	s.mu.Unlock()
	if !later {
		t.Error("the earlier piece was attached over the later one")
	}
	mu.Lock()
	defer mu.Unlock()
	if !stoppedA {
		t.Error("the superseded piece was left playing")
	}
}
