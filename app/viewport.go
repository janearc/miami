package app

import (
	"math"

	"github.com/janearc/miami/scene"
	"github.com/janearc/miami/view"
)

// A viewport is a camera and a destination. The stage draws the whole scene
// once for each, at the size of its rectangle, and lays the pictures on the
// sheet in the order the document wrote them.
//
// Each is a complete picture: its own light, its own depth, its own glow.
// Nothing is shared between two viewports but the scene, so what is in front
// of what is decided per camera, which is the only way it can be right.

// shot is one viewport as the stage holds it.
type shot struct {
	cam        view.Camera
	first      bool // its eye is the stage's own, which a caller may turn
	x, y, w, h float64
	exposure   *scene.Value // the document's, read again every frame
}

// shoot takes a document's cameras and viewports as the stage's own.
//
// The first camera is the one Eye hands out to be steered. scene.MainCamera
// and view.MainCamera are two constants because neither package imports the
// other; they must agree, and a test here holds them to it.
func (s *Stage) shoot(cams []scene.Camera, views []scene.Viewport) {
	built := map[string]view.Camera{}
	for i, c := range cams {
		vc := view.Camera{Eye: eyeOf(c.Eye), Name: c.Name}
		if c.Default {
			vc = view.DefaultCamera()
		}
		built[c.Name] = vc
		if i == 0 {
			s.eye = vc.Eye
		}
	}
	docs := map[string]*scene.Value{}
	for _, c := range cams {
		docs[c.Name] = c.Exposure
	}
	s.shots = s.shots[:0]
	for _, v := range views {
		s.shots = append(s.shots, shot{
			cam: built[v.Camera], first: v.Camera == cams[0].Name,
			x: v.X, y: v.Y, w: v.W, h: v.H,
			exposure: docs[v.Camera],
		})
	}
}

// eyeOf is a document's eye as the one shapes are drawn through. Its place is
// read at the scene's start: a camera that travels is a later job.
func eyeOf(e scene.Eye) view.Eye {
	return view.Eye{
		At: vecAt(e.At, scene.Ctx{}), Lens: e.Lens, Near: e.Near,
		Yaw: e.Yaw, Pitch: e.Pitch,
	}
}

// fills says a viewport is the whole sheet, which is every document written
// before viewports and most written after.
func (v shot) fills() bool {
	return v.x == 0 && v.y == 0 && v.w == 1 && v.h == 1
}

// cells is a viewport in whole cells of a sheet this size.
//
// The edges are rounded, not the widths. Two viewports that meet at a
// fraction then meet at the same cell, with no gap between them and no cell
// drawn twice, whatever the terminal's width happens to be.
func (v shot) cells(cols, rows int) (x0, y0, x1, y1 int) {
	edge := func(f float64, n int) int {
		return min(n, max(0, int(math.Round(f*float64(n)))))
	}
	return edge(v.x, cols), edge(v.y, rows),
		edge(v.x+v.w, cols), edge(v.y+v.h, rows)
}
