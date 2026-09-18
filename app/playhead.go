package app

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/janearc/miami/scene"
)

// Music, as far as a scene is concerned: four numbers a document can bind to.
//
//	@beat    where inside the current beat, nought up to one
//	@bar     where inside the current bar, nought up to one
//	@beats   beats since the first sample; only grows, for things that roll
//	@level   how loud the mix is around now, nought to one
//
// A document never learns what made them. A bounce that lands on every beat is
// four times beat times one less beat, written with the arithmetic bindings
// already have -- and that parabola is not a picture of a bounce, it is the
// path a thrown thing takes under gravity:
//
//	{"mul": [4, "@beat", {"add": [1, {"mul": [-1, "@beat"]}]}]}
//
// The music side knows none of this. A Playhead answers where a piece is at an
// offset from its first sample, and it is a function of that offset alone: no
// clock, no audio device, testable with a number.
//
// The clock is here, because this is what starts the sound, and it is an
// estimate. afplay cannot say where it has got to, so the offset is our own
// clock started when afplay was, less a latency nobody has measured.
//
// That is good enough for a ball landing on the beat and not for anything
// that has to be exact to the sample. Do not build on accuracy it lacks.

// Playhead says where a piece of music is, at an offset from its first sample.
//
// Four floats and a bool rather than a struct, so that satisfying it needs no
// import of this package and this package needs no import of the music. When
// ok is false -- before the start, or past the end of a piece that does not
// loop -- the four numbers are nought.
type Playhead interface {
	At(offset time.Duration) (beat, bar, beats, level float64, ok bool)
}

// musicNames are the constants a playhead supplies, and the only ones a
// document may read that need one attached.
var musicNames = []string{"bar", "beat", "beats", "level"}

// clockNames are the constants the wall clock supplies, always.
var clockNames = []string{"daylight", "hour", "sun"}

// supplied is every name something can hand a document: its own builtins, the
// wall clock, and a playhead if one is attached.
func supplied() [][]string {
	return [][]string{scene.Builtins, clockNames, musicNames}
}

// musicConsts are the four numbers for the frame being drawn.
//
// When the piece is not playing, beat, bar and level go to nought -- a ball on
// nought is resting on the floor, and silence is silence -- but beats holds
// where it got to. Beats is for things that roll, and a helix that snapped back
// to its start the moment the music ended would look like a fault.
//
// Called with s.mu held: it reads what Score writes, and it writes Started and
// heldBeats itself.
func (s *Stage) musicConsts(f Frame) map[string]float64 {
	if s.Started.IsZero() {
		s.Started = f.At // the first frame is when it began
	}
	offset := f.At.Sub(s.Started) - s.Latency
	beat, bar, beats, level, ok := s.Music.At(offset)
	if ok {
		s.heldBeats = beats
	} else {
		beat, bar, level = 0, 0, 0
		beats = s.heldBeats
	}
	return map[string]float64{
		"beat": beat, "bar": bar, "beats": beats, "level": level,
	}
}

// checkRefs refuses a document that reads a name nothing could supply.
//
// A playhead's names are allowed here because one might be attached after the
// stage is built. Whether one was is Validate's question, asked before the
// first frame.
func (s *Stage) checkRefs(d *scene.Doc) error {
	known := map[string]bool{}
	for _, list := range supplied() {
		for _, n := range list {
			known[n] = true
		}
	}
	// a const that takes a name the runtime supplies is refused rather
	// than allowed to lose quietly. the runtime wins a collision, so a
	// document declaring "beat" or "daylight" would have its own number
	// replaced the moment music or the clock arrived, and say nothing.
	var shadows []string
	for n := range d.Const {
		if known[n] {
			shadows = append(shadows, n)
		}
	}
	if len(shadows) > 0 {
		sort.Strings(shadows)
		return fmt.Errorf("declares const %s, which the runtime "+
			"supplies and would replace; name it something else",
			strings.Join(shadows, ", "))
	}
	for n := range d.Const {
		known[n] = true
	}
	var unknown []string
	for _, n := range d.Refs() {
		if !known[n] {
			unknown = append(unknown, "@"+n)
		}
	}
	if len(unknown) > 0 {
		return fmt.Errorf("reads %s, which nothing supplies; "+
			"a document can read its own const, %s",
			strings.Join(unknown, ", "), sources())
	}
	s.refs = d.Refs()
	return nil
}

// sources names what a document can read, for an error somebody has to act on.
func sources() string {
	var all []string
	for _, list := range supplied() {
		for _, n := range list {
			all = append(all, "@"+n)
		}
	}
	sort.Strings(all)
	return strings.Join(all, " ")
}

// Validate refuses a stage that would draw a picture which cannot do what its
// document says: one that reads a playhead's numbers with no playhead
// attached.
//
// Screen.Run asks this before it writes anything to the terminal, so the
// refusal arrives as an error with a name in it rather than as a ball that
// never bounces.
func (s *Stage) Validate() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Music != nil {
		return nil
	}
	var needs []string
	for _, n := range s.refs {
		for _, m := range musicNames {
			if n == m {
				needs = append(needs, "@"+n)
			}
		}
	}
	if len(needs) > 0 {
		return fmt.Errorf("the scene reads %s, which only a "+
			"playhead supplies, and this stage has none",
			strings.Join(needs, ", "))
	}
	return nil
}

// player is the program that makes a sound from a file. A variable so a test
// can start something that is not a speaker.
var player = "afplay"

// ErrNoPlayer is what Play returns on a machine without afplay, which is every
// machine that is not a Mac.
var ErrNoPlayer = errors.New("no afplay on this machine: sound is macOS only")

// Silence is a playhead that is never playing.
//
// It is what a stage carries before its music starts. A game that starts its
// music on a key press still has a scene that reads @beat from the first
// frame, and with nothing attached Run refuses it before the key is pressed.
//
// Attaching Silence says music is coming: the scene validates, and its
// bindings read the rest values -- a ball on the floor -- until Score puts the
// real piece in its place.
var Silence Playhead = silence{}

type silence struct{}

// At is never playing.
func (silence) At(time.Duration) (beat, bar, beats, level float64, ok bool) {
	return 0, 0, 0, 0, false
}

// Play starts a sound file once and says when it started, and hands back what
// stops it. Stopping twice is fine; stopping one that already finished is
// fine.
//
// The start time is taken the moment the process has started, which is before
// it has made a sound. The gap between the two is the latency Stage.Latency
// exists to subtract, and nobody has measured it yet.
func Play(path string) (started time.Time, stop func() error, err error) {
	return playing(path, 0, nil)
}

// playing is Play, and with a period greater than nought it plays again at
// every multiple of the period after the first start.
//
// afplay has no loop, and a looping piece whose playhead keeps counting after
// its sound has ended moves a scene to music nobody can hear. So the restarts
// are scheduled against the first start, not chained off each other: a late
// wake-up moves one restart, and the next is still where it should be.
//
// Each restart carries afplay's own startup latency again, on top of the
// first. If that latency varies, the picture wobbles against the sound by
// that much once a loop; it does not drift.
//
// A restart that fails is told to onFail, once, and stop returns the error.
// Ending the loop quietly was the first version, and it is the fault this
// exists to prevent, moved to an error path: the sound stops, the playhead
// counts on, and the scene moves to music that is not there.
func playing(path string, period time.Duration, onFail func(error)) (
	time.Time, func() error, error) {
	if _, err := os.Stat(path); err != nil {
		// afplay given a missing file starts, fails a moment later, and
		// says so to a stderr nobody is reading. refuse here instead.
		return time.Time{}, nil, err
	}
	bin, err := exec.LookPath(player)
	if err != nil {
		return time.Time{}, nil, ErrNoPlayer
	}
	snd := &sound{bin: bin, path: path, stopped: make(chan struct{}),
		onFail: onFail, live: map[*exec.Cmd]chan struct{}{}}
	if err := snd.start(); err != nil {
		return time.Time{}, nil, err
	}
	started := time.Now()
	if period > 0 {
		go snd.loop(started, period)
	}
	return started, snd.stop, nil
}

// sound is one piece being played, perhaps more than once.
type sound struct {
	bin, path string
	stopped   chan struct{}
	once      sync.Once
	onFail    func(error)

	mu   sync.Mutex
	live map[*exec.Cmd]chan struct{} // passes still sounding, and exits
	err  error                       // why a restart failed, for stop
}

// start begins a pass of the piece.
//
// It does not end the previous pass, and it used to. A piece exactly one
// period long is still sounding when its restart is due: its sound began one
// latency after its process did, so it ends one latency after the period.
//
// The kill cut that tail off every pass, mid-waveform, which clicks. Left
// alone, a pass ends by itself about when the next begins to sound.
//
// A period shorter than its piece now overlaps rather than truncates. That is
// audible rather than silent, and the period is the music side's to get right.
//
// Every process is waited for by its own goroutine as it exits, so a finished
// pass is never left as a zombie waiting on stop.
func (s *sound) start() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	select {
	case <-s.stopped:
		return nil // stopped while a restart was due: start nothing
	default:
	}
	cmd := exec.Command(s.bin, s.path)
	if err := cmd.Start(); err != nil {
		return err
	}
	exited := make(chan struct{})
	s.live[cmd] = exited
	go func() {
		cmd.Wait()
		s.mu.Lock()
		delete(s.live, cmd)
		s.mu.Unlock()
		close(exited)
	}()
	return nil
}

// loop restarts the piece at each multiple of the period after started, until
// it is stopped.
func (s *sound) loop(started time.Time, period time.Duration) {
	for {
		next := nextStart(started, period, time.Now())
		select {
		case <-s.stopped:
			return
		case <-time.After(time.Until(next)):
		}
		if err := s.start(); err != nil {
			s.mu.Lock()
			s.err = err
			s.mu.Unlock()
			if s.onFail != nil {
				s.onFail(err)
			}
			return
		}
	}
}

// nextStart is the first restart after now: the first start plus a whole
// number of periods. A wake-up that missed a restart entirely skips it rather
// than playing twice in a row to catch up.
func nextStart(started time.Time, period time.Duration,
	now time.Time) time.Time {
	k := now.Sub(started)/period + 1
	return started.Add(k * period)
}

// stop ends every pass still sounding and every restart still to come, and
// says why a restart failed if one did.
//
// The exits are collected under the lock and waited on outside it. A pass's
// own waiter takes the lock to say it has gone, so waiting while holding it
// would wait for ever.
func (s *sound) stop() error {
	s.once.Do(func() {
		close(s.stopped)
		s.mu.Lock()
		var exits []chan struct{}
		for cmd, exited := range s.live {
			cmd.Process.Kill()
			exits = append(exits, exited)
		}
		s.mu.Unlock()
		for _, exited := range exits {
			<-exited
		}
	})
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.err
}

// Score starts a piece and attaches its playhead, in one call: the thing a game
// actually wants. The stop it returns ends the sound.
//
//	stop, err := st.Score(wav, piece, piece.LoopPeriod())
//	if err != nil {
//		return err // the scene still draws; it draws at rest
//	}
//	defer func() {
//		if err := stop(); err != nil {
//			log.Println("the music would not stop:", err)
//		}
//	}()
//
// The error stop gives back is worth reading rather than discarding. At the
// end of a program there is little to do with it, but a game that scores a
// second piece while the first will not stop has a sound it cannot end, and
// this is the only place that says so.
//
// period is how long one pass of the piece lasts, for a piece that loops. With
// nought it plays once, and a playhead that keeps counting past the end will
// move the scene to silence -- so a looping piece needs its period, and the
// music side is what knows it.
//
// Safe to call while Run is drawing, from any goroutine: a game starting its
// music on a key press does exactly that.
//
// If a restart fails, the scene is put to rest: Silence goes in where the piece
// was, so nothing moves to a beat that has stopped, and the stop returns why.
//
// Only this piece is replaced; a game that has since scored another keeps it.
// That is what the generation is for: comparing playheads with == panics on
// one that holds a slice.
func (s *Stage) Score(path string, p Playhead, period time.Duration) (
	stop func() error, err error) {
	// the generation is reserved before the sound starts and committed
	// when the piece is attached. reserved rather than committed early:
	// a Score whose file is missing must not make the piece already
	// playing unable to rest.
	//
	// and a failure that arrives before the attach -- a restart due
	// sooner than this goroutine takes the lock -- is written down and
	// read at the attach, so the failed piece is never left moving the
	// scene, whichever of the two gets there first.
	s.mu.Lock()
	s.nextGen++
	gen := s.nextGen
	s.mu.Unlock()
	failed := false // guarded by s.mu
	rest := func(error) {
		s.mu.Lock()
		defer s.mu.Unlock()
		failed = true
		if s.musicGen == gen {
			s.Music = Silence
		}
	}
	started, stop, err := play(path, period, rest)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	if gen < s.musicGen {
		// a Score called after this one has attached already, and the
		// later call wins: attaching this one over it would put the
		// older piece back and leave the newer unable to rest.
		//
		// this sound stops rather than play under a scene following
		// something else: outside the lock, which a failing pass takes.
		s.mu.Unlock()
		stop()
		return nil, ErrSuperseded
	}
	s.musicGen = gen
	s.Music, s.Started, s.heldBeats = p, started, 0
	if failed {
		s.Music = Silence
	}
	s.mu.Unlock()
	return stop, nil
}

// ErrSuperseded is what Score returns when another Score, called after it,
// attached its piece first. The superseded piece has been stopped.
var ErrSuperseded = errors.New("superseded: a later Score attached first")

// play is playing, as Score reaches it: a variable so a test can put a failure
// exactly where the timing of a real one never would.
var play = playing
