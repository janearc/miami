// builder is what another program does with miami: it has its own work and its
// own events, and miami draws while that happens.
//
// This is the shape a build tool wants. Nothing here reaches inside miami. It
// makes a stage, hangs a themed bar and a line of type on it, starts its own
// work on its own goroutine, and calls Run.
//
// The work sets the bar from wherever it happens to be; the loop picks up
// whatever the state is when a frame falls due. miami never learns what the
// work is.
//
//	go run ./example/builder
package main

import (
	"fmt"
	"math"
	"sync/atomic"
	"time"

	"github.com/janearc/miami/app"
	"github.com/janearc/miami/grade"
	"github.com/janearc/miami/sky"
	"github.com/janearc/miami/view"
)

// main sets a scene up, starts the work, and hands the loop over.
func main() {
	stage, err := app.NewStage("", "sprites/made/starling.css")
	if err != nil {
		fmt.Println(err)
		return
	}

	// a sky that knows the time of day, and a ball on a floor under it
	stage.Sky = func(sh grade.Sheet, at time.Time) grade.Stack {
		aspect := (float64(sh.H) * 4) / (float64(sh.W) * 3)
		return sky.At(sh, view.SunAt(at), stage.Lit.Eye.HorizonV(aspect))
	}
	ball := &view.Ball{Radius: 0.34, Spin: view.Vec{X: 0.11, Y: 0.43, Z: 0.07}}
	stage.Lit = &view.Lit{
		Eye:    view.Eye{At: view.Vec{Y: 0.05}, Lens: 1, Near: 0.05},
		Lights: []view.Light{{At: view.Vec{X: -1.2, Y: 3.0, Z: 1.6}}},
		Solids: []view.Solid{
			view.Floor{Y: -0.55, Tile: 0.62, Far: 9.5, A: 'V', B: 'K',
				Shade: map[byte]byte{'V': 'K', 'K': 'D', 'D': 'E'}},
			ball,
		},
	}

	title := &app.Text{Font: "small", Row: 2, Col: -1, Ink: app.Vaporwave,
		Scrim: 0.40}
	title.Set("building")
	bar := &app.Bar{Row: -3, Col: -1, Ink: app.Vaporwave, Label: "game",
		Scrim: 0.45}
	stage.Add(title)
	stage.Add(bar)

	// the work. it knows nothing about frames and is not on their clock.
	steps := []string{"geom", "raster", "cells", "paint", "typeset", "grade",
		"scene", "motion", "effect", "view", "compose", "app"}
	var done int64
	go func() {
		for i, s := range steps {
			bar.Say(s)
			bar.Set(i+1, len(steps))
			time.Sleep(320 * time.Millisecond)
		}
		title.Set("done")
		atomic.StoreInt64(&done, 1)
	}()

	finish := time.Time{}
	app.Screen{Cols: 92, Rows: 26, FPS: 12, Raw: true}.Run(stage,
		func(f app.Frame) bool {
			for _, k := range f.Keys {
				if k.Rune == 'q' || k.Rune == 3 {
					return false
				}
			}
			// the ball keeps going while the work does
			sec := f.Since.Seconds()
			ball.At = view.Vec{
				X: 1.3 * math.Sin(sec/9),
				Y: -0.21 + 0.66*math.Abs(math.Sin(sec/1.3)),
				Z: 3.6 - 0.7*math.Cos(sec/9),
			}
			if atomic.LoadInt64(&done) == 1 {
				if finish.IsZero() {
					finish = f.At
				}
				return f.At.Sub(finish) < 1500*time.Millisecond
			}
			return true
		})
}
