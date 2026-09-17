// Package app is miami with an event loop around it, for programs that are
// doing something else.
//
// Everything else here draws a picture and stops, or draws a picture per frame
// and owns the terminal while it does. Neither is any use to a build tool that
// wants a themed progress bar over a sunset while it compiles: that program
// owns the events, and miami has to be the thing it calls.
//
// So the split is: the caller owns what happens, and this owns what a frame is.
// It does not know or ask what the events are. It knows how long a frame lasts,
// it knows how to compose one, and it draws until the caller says to stop.
//
//	sc := app.Screen{Cols: 90, Rows: 24, FPS: 12}
//	st, _ := app.NewStage("scenes/dusk.json", "made/starling.css")
//	bar := &app.Bar{Row: 20, Label: "building"}
//	st.Add(bar)
//
//	go doTheBuild(func(done, total int) { bar.Set(done, total) })
//	sc.Run(st, func(f app.Frame) bool { return !finished })
//
// Nothing in that reaches into miami's drawing. The bar is set from wherever
// the work happens, on whatever clock the work has, and the loop picks up
// whatever the state is when a frame falls due.
package app

import (
	"fmt"
	"image/color"
	"math"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/janearc/miami/cells"
	"github.com/janearc/miami/compose"
	"github.com/janearc/miami/grade"
	"github.com/janearc/miami/scene"
	"github.com/janearc/miami/view"
)

// Frame is what the caller is told about the moment being drawn.
type Frame struct {
	At    time.Time
	Since time.Duration // since the loop started
	Keys  []Key
	N     int // how many frames have been drawn

	// Scene is the same moment as a document sees it: the scene's own
	// clock, its constants, and the few things only the program knows --
	// the hour, the sun's height, how much of the day is gone.
	Scene scene.Ctx
}

// Key is a keypress. Arrows and the like arrive as a Name; everything else as a
// Rune.
type Key struct {
	Rune rune
	Name string // "up" "down" "left" "right" "esc", or empty
}

// Screen is the loop and the terminal contract: raw mode, a hidden cursor,
// synchronised frames, deltas, and putting all of it back afterwards.
type Screen struct {
	Cols, Rows int
	FPS        int
	Raw        bool // take over the keyboard; false leaves stdin alone
	Out        *os.File
}

// Drawer is anything that can produce a frame of cells.
type Drawer interface {
	Draw(f Frame, cols, rows int) [][]cells.Cell
}

// Run draws frames until more returns false.
//
// The caller's function is asked after each frame rather than before, so a
// program that finishes instantly still shows one.
func (s Screen) Run(d Drawer, more func(Frame) bool) error {
	// before anything reaches the terminal: a drawer that knows it cannot
	// draw what it was asked to says so here, with a name, rather than on
	// a screen that has already been cleared.
	if v, ok := d.(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return err
		}
	}
	if s.Cols == 0 || s.Rows == 0 {
		cols, rows := Size()
		if s.Cols == 0 {
			s.Cols = cols
		}
		if s.Rows == 0 {
			s.Rows = rows
		}
	}
	if s.FPS == 0 {
		s.FPS = 12
	}
	out := s.Out
	if out == nil {
		out = os.Stdout
	}

	restore := func() { fmt.Fprint(out, "\x1b[?25h\x1b[0m\n") }
	var unraw func()
	keys := make(chan Key, 32)
	if s.Raw {
		unraw = rawMode()
		go readKeys(keys)
	} else {
		unraw = func() {}
	}
	stop := func() { unraw(); restore() }

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	go func() { <-sig; stop(); os.Exit(0) }()
	defer stop()

	fmt.Fprint(out, "\x1b[?25l\x1b[2J")
	begin := time.Now()
	tick := time.Duration(float64(time.Second) / float64(s.FPS))
	var prev [][]cells.Cell
	for n := 0; ; n++ {
		start := time.Now()
		f := Frame{At: start, Since: start.Sub(begin), N: n}
		for {
			select {
			case k := <-keys:
				f.Keys = append(f.Keys, k)
				continue
			default:
			}
			break
		}

		cs := d.Draw(f, s.Cols, s.Rows)
		fmt.Fprint(out, "\x1b[?2026h")
		if prev == nil {
			fmt.Fprint(out, "\x1b[H")
			fmt.Fprint(out, cells.ANSI(cs))
		} else {
			fmt.Fprint(out, cells.Delta(prev, cs))
		}
		fmt.Fprint(out, "\x1b[?2026l")
		prev = cs

		if !more(f) {
			return nil
		}
		if d := tick - time.Since(start); d > 0 {
			time.Sleep(d)
		}
	}
}

// rawMode puts the terminal into raw mode through stty and hands back the undo.
// stty rather than a termios binding because it is on every machine this runs
// on, and a keyboard is not worth a dependency.
func rawMode() func() {
	saved, err := exec.Command("stty", "-g").Output()
	if err != nil {
		// not a terminal: it still draws, it just cannot be steered
		return func() {}
	}
	raw := exec.Command("stty", "raw", "-echo")
	raw.Stdin = os.Stdin
	raw.Run()
	return func() {
		c := exec.Command("stty", strings.TrimSpace(string(saved)))
		c.Stdin = os.Stdin
		c.Run()
	}
}

// readKeys turns bytes into keys. Arrows arrive as three bytes and are named
// rather than passed through, so a caller never has to know that.
func readKeys(out chan<- Key) {
	buf := make([]byte, 32)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil || n == 0 {
			return
		}
		for i := 0; i < n; i++ {
			if buf[i] == 0x1b && i+2 < n && buf[i+1] == '[' {
				name := map[byte]string{'A': "up", 'B': "down",
					'C': "right", 'D': "left"}[buf[i+2]]
				if name != "" {
					out <- Key{Name: name}
					i += 2
					continue
				}
			}
			out <- Key{Rune: rune(buf[i])}
		}
	}
}

// Overlay is something drawn over the scene: type, a bar, a readout.
type Overlay interface {
	Over(cs [][]cells.Cell, f Frame)
}

// Stage composes a frame: a painted scene from a document, anything standing in
// space, and overlays on top.
//
// All three are optional. A program that only wants a progress bar over a
// gradient sets a scene and one overlay and never touches the rest.
type Stage struct {
	Doc      *scene.Doc
	Sky      func(sh grade.Sheet, at time.Time) grade.Stack // overrides Doc
	Lit      *view.Lit
	Objects  []view.Object
	Colours  map[byte]color.RGBA
	Overlays []Overlay

	// Music is where a piece of music is, for a document that reads @beat,
	// @bar, @beats or @level. Started is when its first sample went out,
	// and the first frame if left zero.
	//
	// Latency comes off the offset: how long the player takes to make a
	// sound once started. Score sets all three; see playhead.go.
	//
	// Set these directly only before Run. While Run is drawing, attach
	// music with Score, which takes the lock. A stage drawn once already --
	// for a png, say -- should be given a Started of its own, or the first
	// frame it ever drew becomes the start of the piece.
	Music   Playhead
	Started time.Time
	Latency time.Duration

	// What a document built, and what it needs to move each frame.
	eye     view.Eye
	solids  []view.Solid
	lights  []view.Light
	sun     []bool      // per light: it is the sun, so the hour places it
	lightAt []scene.Vec // per light: where a lamp is, as the document says
	moving  []mover
	named   map[string]any
	at      scene.Ctx // the moment being drawn, for a texture that asks

	refs      []string // every @name the document reads
	heldBeats float64  // where @beats got to, kept once the music stops
	musicGen  int      // which Score the music came from
	nextGen   int      // the last generation handed out

	mu sync.Mutex
}

// mover is one thing whose place is a binding: where it should be, and how to
// put it there. A closure rather than an index, because the shapes it moves are
// different types and what they have in common is only that they have a place.
type mover struct {
	at  scene.Vec
	set func(view.Vec)
}

// now is the moment being drawn, for a texture evaluated per subpixel.
func (s *Stage) now() scene.Ctx { return s.at }

// NewStage loads a scene document and a colourway.
func NewStage(doc, colourway string) (*Stage, error) {
	st := &Stage{}
	if doc != "" {
		d, err := loadScene(doc)
		if err != nil {
			return nil, err
		}
		st.Doc = d
	}
	if colourway != "" {
		c, err := loadColours(colourway)
		if err != nil {
			return nil, err
		}
		st.Colours = c
	}
	if st.Doc != nil {
		if err := st.checkRefs(st.Doc); err != nil {
			return nil, fmt.Errorf("%s: %w", doc, err)
		}
		if err := st.fromDoc(st.Doc); err != nil {
			return nil, fmt.Errorf("%s: %w", doc, err)
		}
	}
	return st, nil
}

// Add puts an overlay on top of whatever is already there.
func (s *Stage) Add(o Overlay) {
	s.mu.Lock()
	s.Overlays = append(s.Overlays, o)
	s.mu.Unlock()
}

// Draw composes one frame.
func (s *Stage) Draw(f Frame, cols, rows int) [][]cells.Cell {
	// the moment, as the document sees it, before anything is asked where
	// it is: a texture reads this per subpixel and it has to be the frame
	// being drawn rather than the one before.
	if s.Doc != nil {
		run := runtimeConsts(f.At)
		// under the lock: Score attaches music from whatever goroutine
		// a game starts it on, while this one is drawing
		s.mu.Lock()
		if s.Music != nil {
			for k, v := range s.musicConsts(f) {
				run[k] = v
			}
		}
		s.mu.Unlock()
		f.Scene = s.Doc.Ctx(f.Since.Seconds(), run)
	}
	s.at = f.Scene
	s.place(f)

	sh := grade.Sextants(cols, rows)
	var px [][]color.RGBA
	switch {
	case s.Sky != nil:
		px = sh.Paint(s.Sky(sh, f.At))
	case s.Doc != nil:
		st, err := s.Doc.At(f.Since.Seconds())
		if err == nil {
			px = sh.Paint(st)
		}
	}
	if px == nil {
		px = make([][]color.RGBA, rows*3)
		for y := range px {
			px[y] = make([]color.RGBA, cols*2)
			for x := range px[y] {
				px[y][x] = color.RGBA{10, 9, 16, 255}
			}
		}
	}

	if s.Lit != nil {
		compose.Blit(
			px, s.Lit.Trace(f.Since, cols, rows), s.Colours, 0, 0,
		)
	}
	for _, p := range view.Resolve(
		s.Objects, view.DefaultEye(), f.Since, sh.W, sh.H,
	) {
		compose.Blit(px, p.Frame, s.Colours, p.X, p.Y)
	}

	cs := cells.ToCells(px, 2, 3)
	s.mu.Lock()
	over := append([]Overlay(nil), s.Overlays...)
	s.mu.Unlock()
	for _, o := range over {
		o.Over(cs, f)
	}
	return cs
}

// Scrim pulls cells toward dark so type over them reads, fading at its edges
// rather than stopping.
//
// Feathered because a hard-edged one is a box, and a box laid over a picture
// reads as damage to it. Fading over a couple of cells is enough: the eye reads
// it as the picture being darker there, not as a panel.
func Scrim(cs [][]cells.Cell, x0, y0, w, h int, k, feather float64) {
	if feather < 1 {
		feather = 1
	}
	for y := y0 - int(feather); y < y0+h+int(feather); y++ {
		if y < 0 || y >= len(cs) {
			continue
		}
		for x := x0 - int(feather); x < x0+w+int(feather); x++ {
			if x < 0 || x >= len(cs[y]) {
				continue
			}
			ox := math.Max(
				0, math.Max(float64(x0-x), float64(x-(x0+w-1))),
			)
			oy := math.Max(
				0, math.Max(float64(y0-y), float64(y-(y0+h-1))),
			)
			if out := math.Max(ox, oy); out <= feather {
				f := k + (1-k)*(out/feather)
				cs[y][x].Fg = dim(cs[y][x].Fg, f)
				cs[y][x].Bg = dim(cs[y][x].Bg, f)
			}
		}
	}
}

// dim scales a colour toward black. In sRGB and not in linear light, which is
// wrong for mixing two colours and right for this: the scrim is asking the
// terminal for a fraction of the brightness it is already showing, and that is
// the number the terminal was given.
func dim(c color.RGBA, k float64) color.RGBA {
	return color.RGBA{uint8(float64(c.R) * k), uint8(float64(c.G) * k),
		uint8(float64(c.B) * k), c.A}
}

// Size is the terminal's, in cells, or a sensible pair when there is no
// terminal to ask.
//
// Through stty, for the same reason raw mode is: it is on every machine this
// runs on, and the size of a window is not worth a dependency.
func Size() (cols, rows int) {
	// stty reports the size of whatever is on its standard input, and the
	// terminal worth asking about is the one being drawn to: stdout.
	//
	// asking with no input at all was the first try. it got /dev/null,
	// said "stdin isn't a terminal", and every window however large drew
	// ninety by twenty-four in the corner of itself.
	//
	// stdin second, for a program drawing somewhere else, and the
	// controlling terminal last, for one with both streams redirected.
	for _, in := range []*os.File{os.Stdout, os.Stdin} {
		if c, r, ok := sttySize(in); ok {
			return c, r
		}
	}
	if tty, err := os.Open("/dev/tty"); err == nil {
		defer tty.Close()
		if c, r, ok := sttySize(tty); ok {
			return c, r
		}
	}
	return 90, 24
}

// sttySize asks stty about one file, and says whether the answer was a size.
//
// A row is kept back. A picture drawn to the full height scrolls the moment
// anything is written after it -- the shell's next prompt will do -- and a
// scrolled frame tears every frame after it, because a delta is addressed to
// where the last one was.
func sttySize(in *os.File) (cols, rows int, ok bool) {
	cmd := exec.Command("stty", "size")
	cmd.Stdin = in
	out, err := cmd.Output()
	if err != nil {
		return 0, 0, false
	}
	var r, c int
	if n, _ := fmt.Sscan(string(out), &r, &c); n != 2 || r < 4 || c < 20 {
		return 0, 0, false
	}
	return c, r - 1, true
}
