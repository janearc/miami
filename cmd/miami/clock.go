package main

import (
	"fmt"
	"os"
	"time"

	"github.com/janearc/miami/app"
	"github.com/janearc/miami/cells"
	"github.com/janearc/miami/proof"
)

// A clock, and nothing of it written here.
//
// It is the demonstration and it is the proof, and what it proves has changed.
// It used to prove that the flagship could be built on the public library
// rather than by reaching inside it.
//
// It proves something stronger now. The scene is app/builtin/clock.json, and
// this file has no sky in it, no ball, no floor, no lights and no digits.
// What is left is the four things that are genuinely this program's: which
// file, which clock, what the keys do, and when to stop.
//
//	q  ctrl-c   stop
//	+  -        the clock's size
//	arrows      where it sits
//	[  ]        yaw
//	{  }        pitch
type clockApp struct {
	stage *app.Stage
	face  *app.Digits
	fonts []string
	size  int
	at    time.Time // fixed, for --at; zero means the wall clock
}

// clock reads the arguments and either writes one frame or runs the loop.
//
//	miami clock [COLS ROWS [FPS]] [--once] [--at HH:MM]
func clock(args []string) {
	cols, rows, fps := 96, 28, 12
	if len(args) > 1 {
		fmt.Sscan(args[0], &cols)
		fmt.Sscan(args[1], &rows)
	}
	if len(args) > 2 && args[2] != "--once" && args[2] != "--at" {
		fmt.Sscan(args[2], &fps)
	}

	// --once writes a single frame and --at sets the clock it is drawn
	// from. a picture that changes with the hour cannot be checked at the
	// hour it happens to be; --at is how dawn gets looked at in the
	// afternoon.
	once, at := false, time.Time{}
	for i, a := range args {
		if a == "--once" {
			once = true
		}
		if a == "--at" && i+1 < len(args) {
			if t, err := time.Parse(
				"15:04", args[i+1],
			); err == nil {
				n := time.Now()
				at = time.Date(n.Year(), n.Month(), n.Day(),
					t.Hour(), t.Minute(), 0, 0,
					n.Location())
			}
		}
	}

	c, err := newClock(at)
	proof.Must(err)
	if once {
		cs := c.Draw(app.Frame{At: c.now()}, cols, rows)
		proof.Must(write("out/clock.ansi", []byte(cells.ANSI(cs))))
		fmt.Fprintln(os.Stderr, "wrote out/clock.ansi")
		return
	}
	proof.Must(app.Screen{Cols: cols, Rows: rows, FPS: fps, Raw: true}.
		Run(c, c.keys))
}

// newClock loads the scene and finds the one shape the keyboard steers.
func newClock(at time.Time) (*clockApp, error) {
	st, err := app.NewStage("@clock", "@starling")
	if err != nil {
		return nil, err
	}
	c := &clockApp{
		stage: st,
		face:  st.Digits("face"),
		fonts: []string{"mini", "small", "smslant", "chunky"},
		size:  1,
		at:    at,
	}
	if c.face == nil {
		return nil, fmt.Errorf(
			"the clock scene has no clock named face")
	}
	return c, nil
}

// now is the clock this is drawn from: the wall, or whatever --at pinned.
func (c *clockApp) now() time.Time {
	if c.at.IsZero() {
		return time.Now()
	}
	return c.at
}

// Draw hands the moment to the stage, on the wall clock rather than the loop's.
//
// The scene's own time is the second hand, so the ball's minute is the clock's
// minute. A ball on the loop's clock drifts against the time it is standing in
// front of, and after a while it reads as two scenes sharing a screen.
func (c *clockApp) Draw(f app.Frame, cols, rows int) [][]cells.Cell {
	n := c.now()
	sec := float64(n.Second()) + float64(n.Nanosecond())/1e9
	f.At = n
	f.Since = time.Duration(sec * float64(time.Second))
	return c.stage.Draw(f, cols, rows)
}

// keys steers it, and says when to stop.
func (c *clockApp) keys(f app.Frame) bool {
	for _, k := range f.Keys {
		switch {
		case k.Rune == 'q' || k.Rune == 3:
			return false
		case k.Rune == '+' || k.Rune == '=':
			c.resize(1)
		case k.Rune == '-' || k.Rune == '_':
			c.resize(-1)
		case k.Rune == '[':
			c.stage.Eye().Yaw -= 0.04
		case k.Rune == ']':
			c.stage.Eye().Yaw += 0.04
		case k.Rune == '{':
			c.stage.Eye().Pitch -= 0.03
		case k.Rune == '}':
			c.stage.Eye().Pitch += 0.03
		case k.Name == "up":
			c.face.Row--
		case k.Name == "down":
			c.face.Row++
		case k.Name == "left":
			c.nudge(-1)
		case k.Name == "right":
			c.nudge(1)
		}
	}
	return true
}

// resize steps through the faces.
//
// The size is a choice of font rather than a scale factor: a figlet glyph is
// made of characters and there is no half of one, so doubling a small face
// gives a blocky copy of it rather than a large face.
func (c *clockApp) resize(d int) {
	c.size += d
	if c.size < 0 {
		c.size = 0
	}
	if c.size >= len(c.fonts) {
		c.size = len(c.fonts) - 1
	}
	c.face.Font = c.fonts[c.size]
}

// nudge moves the face sideways. A Col of -1 means centred, so the first nudge
// has to leave that behind rather than treat it as a coordinate.
func (c *clockApp) nudge(d int) {
	if c.face.Col < 0 {
		c.face.Col = 0
	}
	c.face.Col += d * 2
}
