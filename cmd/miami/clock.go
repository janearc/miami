package main

import (
	"fmt"
	"math"
	"os"
	"time"

	"github.com/janearc/miami/app"
	"github.com/janearc/miami/cells"
	"github.com/janearc/miami/grade"
	"github.com/janearc/miami/proof"
	"github.com/janearc/miami/sky"
	"github.com/janearc/miami/view"
)

// A clock, written on miami's own public library.
//
// It is here as the demonstration and it is here as the proof. Every piece of
// it -- the loop, the terminal contract, the keyboard, the scrim, the digits --
// comes out of package app, which is what any other program gets.
//
// If the flagship could only be written by reaching inside, the library would
// not be one.
//
//	q  ctrl-c   stop
//	+  -        the clock's size
//	arrows      where it sits
//	[  ]        yaw
//	{  }        pitch
type clockApp struct {
	stage *app.Stage
	face  *app.Digits
	ball  *view.Ball
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
			if t, err := time.Parse("15:04", args[i+1]); err == nil {
				n := time.Now()
				at = time.Date(n.Year(), n.Month(), n.Day(),
					t.Hour(), t.Minute(), 0, 0, n.Location())
			}
		}
	}

	c := newClock(at)
	if once {
		cs := c.Draw(app.Frame{At: c.now()}, cols, rows)
		proof.Must(write("out/clock.ansi", []byte(cells.ANSI(cs))))
		fmt.Fprintln(os.Stderr, "wrote out/clock.ansi")
		return
	}
	proof.Must(app.Screen{Cols: cols, Rows: rows, FPS: fps, Raw: true}.
		Run(c, c.keys))
}

// newClock builds the scene: a stage with a sky that knows the hour, a lit
// floor for the ball to sit on, and the face over the top of both.
func newClock(at time.Time) *clockApp {
	st, err := app.NewStage("", "sprites/made/starling.css")
	proof.Must(err)
	c := &clockApp{
		stage: st,
		fonts: []string{"mini", "small", "smslant", "chunky"},
		size:  1,
		at:    at,
	}
	c.face = &app.Digits{Font: "small", Row: 2, Col: -1,
		Ink: app.Vaporwave, Scrim: 0.40, Flicker: true, At: c.now}
	st.Add(c.face)
	c.ball = &view.Ball{Radius: 0.34, Spin: view.Vec{X: 0.11, Y: 0.43, Z: 0.07}}
	st.Lit = &view.Lit{
		Eye: view.Eye{At: view.Vec{Y: 0.05}, Lens: 1, Near: 0.05},
		Solids: []view.Solid{
			view.Floor{Y: -0.55, Tile: 0.62, Far: 9.5, A: 'V', B: 'K',
				Shade: map[byte]byte{'V': 'K', 'K': 'D', 'D': 'E'}},
			c.ball,
		},
	}
	// the sky is painted and the floor is traced, so both are told the same
	// horizon: the one the camera's pitch puts it at
	st.Sky = func(sh grade.Sheet, _ time.Time) grade.Stack {
		aspect := (float64(sh.H) * 4) / (float64(sh.W) * 3)
		return sky.At(sh, view.SunAt(c.now()), st.Lit.Eye.HorizonV(aspect))
	}
	return c
}

// now is the clock this is drawn from: the wall, or whatever --at pinned.
func (c *clockApp) now() time.Time {
	if c.at.IsZero() {
		return time.Now()
	}
	return c.at
}

// Draw places the ball for the moment and hands the rest to the stage.
func (c *clockApp) Draw(f app.Frame, cols, rows int) [][]cells.Cell {
	n := c.now()
	sec := float64(n.Second()) + float64(n.Nanosecond())/1e9
	minute := sec / 60
	c.ball.At = view.Vec{
		X: 1.3 * math.Sin(2*math.Pi*minute),
		Y: -0.21 + 0.66*math.Abs(math.Sin(math.Pi*sec/4)),
		Z: 3.6 - 0.7*math.Cos(2*math.Pi*minute),
	}
	// a lamp that is always there, and the sun, which is off at night and
	// so takes a shade off the whole floor when it is
	c.stage.Lit.Lights = []view.Light{
		{At: view.Vec{X: -1.2, Y: 3.0, Z: 1.6}},
		view.SunAt(n).Light(),
	}
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
			c.stage.Lit.Eye.Yaw -= 0.04
		case k.Rune == ']':
			c.stage.Lit.Eye.Yaw += 0.04
		case k.Rune == '{':
			c.stage.Lit.Eye.Pitch -= 0.03
		case k.Rune == '}':
			c.stage.Lit.Eye.Pitch += 0.03
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
