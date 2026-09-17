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

// Play starts a sound file and says when it started, and hands back what stops
// it. Stopping twice is fine; stopping one that already finished is fine.
//
// The start time is taken the moment the process has started, which is before
// it has made a sound: the gap between the two is the latency Stage.Latency
// exists to subtract, and nobody has measured it yet.
func Play(path string) (started time.Time, stop func(), err error) {
	if _, err := os.Stat(path); err != nil {
		// afplay given a missing file starts, fails a moment later, and
		// says so to a stderr nobody is reading. refuse here instead.
		return time.Time{}, nil, err
	}
	bin, err := exec.LookPath(player)
	if err != nil {
		return time.Time{}, nil, ErrNoPlayer
	}
	cmd := exec.Command(bin, path)
	if err := cmd.Start(); err != nil {
		return time.Time{}, nil, err
	}
	started = time.Now()
	var once sync.Once
	stop = func() {
		once.Do(func() {
			cmd.Process.Kill()
			cmd.Wait()
		})
	}
	return started, stop, nil
}

// Score attaches a playhead and starts its sound, in one call: the thing a game
// actually wants. The stop it returns ends the sound.
func (s *Stage) Score(path string, p Playhead) (stop func(), err error) {
	started, stop, err := Play(path)
	if err != nil {
		return nil, err
	}
	s.Music, s.Started = p, started
	return stop, nil
}
