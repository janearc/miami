package main

import (
	"time"

	"github.com/janearc/miami/app"
	"github.com/janearc/miami/cells"
	"github.com/janearc/miami/grade"
	"github.com/janearc/miami/sky"
	"github.com/janearc/miami/texture"
	"github.com/janearc/miami/view"
)

// library: what another program gets. This page is drawn by calling the public
// library exactly the way a build tool would, and then asking the stage for one
// frame instead of running the loop.
func libraryPage() ([][]cells.Cell, string) {
	const cols, rows = 66, 15

	stage, err := app.NewStage("", "sprites/made/starling.css")
	if err != nil {
		return nil, err.Error()
	}
	stage.Lit = &view.Lit{
		Eye:    view.Eye{At: view.Vec{Y: 0.05}, Lens: 1, Near: 0.05},
		Lights: []view.Light{{At: view.Vec{X: -1.2, Y: 3.0, Z: 1.6}}},
		Solids: []view.Solid{
			view.Floor{
				Y: -0.55, Tile: 0.62, Far: 9.5, A: 'V', B: 'K',
				Shade: floorShade,
			},
			&view.Ball{Radius: 0.3,
				At:   view.Vec{X: 0.45, Y: -0.16, Z: 3.4},
				Spin: view.Vec{X: 0.11, Y: 0.43, Z: 0.07}},
			// the progress bar, which is a cylinder rather than a
			// row of blocks: it stands on the floor, the lamp
			// finds it, and it throws a shadow like anything else
			// in the scene
			&view.Cylinder{
				// resting on the floor: its centre is one
				// radius above it, which is what "on" means
				// and is not a number to guess at
				At:     view.Vec{X: -0.1, Y: -0.48, Z: 3.0},
				Radius: 0.07, Length: 1.9, Caps: true,
				// banded around rather than flat along,
				// because a cylinder reads as round only
				// where its bands crowd at the silhouette.
				// flat roles make a tube look like a plank.
				Texture: texture.Split{
					Done: texture.Stripes{
						A: 'O', B: 'R', Period: 1.0 / 7,
					},
					Rest: texture.Stripes{
						A: 'D', B: 'K', Period: 1.0 / 7,
					},
					Frac: func() float64 { return 0.58 },
				},
			},
		},
	}
	stage.Sky = func(sh grade.Sheet, at time.Time) grade.Stack {
		aspect := (float64(sh.H) * 4) / (float64(sh.W) * 3)
		return sky.At(
			sh, view.SunAt(at), stage.Lit.Eye.HorizonV(aspect),
		)
	}

	title := &app.Text{
		Font: "small", Row: 1, Col: -1, Ink: app.Vaporwave, Scrim: 0.25,
	}
	title.Set("building")
	stage.Add(title)

	// an hour rather than the wall clock, so the page looks the same in the
	// morning as it does at night and a difference in it means a change in
	// the code
	at := time.Date(2026, 9, 16, 17, 20, 0, 0, time.Local)
	return stage.Draw(app.Frame{At: at}, cols, rows), `
That picture was drawn by calling the library, not by miami.

Everything above came from package app. A stage, a sky told what time it is, a
lit floor, and a progress bar sitting at seven of twelve -- composed and handed
back as cells. No terminal was opened to make it.

That split is the point. Screen owns the loop and the terminal contract; Stage
owns what a frame is made of; the caller owns what happens. A build tool starts
its build on its own goroutine, calls bar.Set from wherever the work is, and
calls Run. miami never learns what the work was.

	sc := app.Screen{Cols: 90, Rows: 24, FPS: 12}
	st, _ := app.NewStage("scenes/dusk.json", "made/starling.css")
	bar := &app.Bar{Row: 20, Label: "building"}
	st.Add(bar)

	go build(func(d, n int) { bar.Set(d, n) })
	sc.Run(st, func(f app.Frame) bool { return !done })

Overlays are safe to set from another goroutine, because that is the whole
point: the work runs on its own clock and the loop takes whatever the state is
when a frame falls due. Nothing blocks on a frame boundary.

The clock verb is written on this and nothing else. That is deliberate. If the
flagship could only be built by reaching inside, the library would not be one.`
}

// floorShade is the ladder a shadowed floor square steps down. Named because
// two pages and two programs want the same one, and a shadow that goes to a
// different colour in the documentation than in the program is worse than no
// documentation.
var floorShade = map[byte]byte{'V': 'K', 'K': 'D', 'D': 'E'}
