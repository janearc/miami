package app

import (
	"image/color"
	"math"

	"github.com/janearc/miami/compose"
	"github.com/janearc/miami/grade"
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
	cam view.Camera
	// eye is the eye this viewport draws through, and it can be turned in
	// place: a hand dragging on a viewport turns the camera drawing it.
	// The first viewport's is the stage's own, so a caller holding what
	// Eye handed it steers the same camera a hand does.
	eye        *view.Eye
	first      bool
	x, y, w, h float64
	exposure   *scene.Value // the document's, read again every frame
}

// look is how one viewport sees one frame.
type look struct {
	eye      view.Eye
	exposure float64
	decided  bool    // somebody had an opinion; nought then means black
	scale    float64 // the viewport's width over the sheet's
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
			s.eye, s.seen = vc.Eye, vc.Eye
		}
	}
	docs := map[string]*scene.Value{}
	for _, c := range cams {
		docs[c.Name] = c.Exposure
	}
	s.shots = s.shots[:0]
	for _, v := range views {
		first := v.Camera == cams[0].Name
		eye := &s.eye
		if !first {
			// its own, so that turning this viewport does not turn
			// another that happens to name the same camera
			own := built[v.Camera].Eye
			eye = &own
		}
		s.shots = append(s.shots, shot{
			cam: built[v.Camera], eye: eye, first: first,
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

// lookOf is a viewport's look for this frame.
//
// A document's exposure is read here, every frame, and used as it is: a
// binding that reaches nought on the beat is a black frame, not a camera
// with no opinion. Only a camera built by hand uses nought to say nothing.
func (s *Stage) lookOf(v shot, f Frame, scale float64) look {
	l := look{eye: v.cam.Eye, scale: scale}
	if v.eye != nil {
		l.eye = *v.eye
	}
	switch {
	case v.exposure != nil:
		l.exposure, l.decided = v.exposure.At(f.Scene), true
	case v.cam.Exposure > 0:
		l.exposure, l.decided = v.cam.Exposure, true
	}
	return l
}

// exposed is the exposure a frame is brought to the screen with.
func (l look) exposed(scene float64) float64 {
	if l.decided {
		return l.exposure
	}
	return scene
}

// radius is a glow's radius in this viewport.
//
// A glow belongs to the picture and not to the terminal, so half a sheet's
// worth of quasar glows half as far in columns and the same in the scene.
// It stops at one column on the way down, or a small viewport would
// silently lose its glow; a radius already under one is left as it was.
func (l look) radius(r float64) float64 {
	if l.scale >= 1 || r <= 1 {
		return r
	}
	return math.Max(1, r*l.scale)
}

// pictures is the frame's subpixels: every viewport's picture, in its place.
func (s *Stage) pictures(f Frame, cols, rows int) [][]color.RGBA {
	if len(s.shots) == 0 {
		return s.picture(cols, rows, f, look{eye: s.eye, scale: 1})
	}
	if len(s.shots) == 1 && s.shots[0].fills() {
		return s.picture(cols, rows, f, s.lookOf(s.shots[0], f, 1))
	}
	px := dark(cols*2, rows*3)
	for _, v := range s.shots {
		x0, y0, x1, y1 := v.cells(cols, rows)
		if x1 <= x0 || y1 <= y0 {
			continue // too small to be a cell on a sheet this size
		}
		scale := float64(x1-x0) / float64(cols)
		sub := s.picture(x1-x0, y1-y0, f, s.lookOf(v, f, scale))
		for y, row := range sub {
			copy(px[y0*3+y][x0*2:], row)
		}
	}
	return px
}

// picture is one whole picture of the scene, this many cells, seen this way.
//
// This is what Draw did before there were viewports, with the eye taken as
// an argument instead of read from the stage.
func (s *Stage) picture(cols, rows int, f Frame, v look) [][]color.RGBA {
	sh := grade.Sextants(cols, rows)
	s.seen = v.eye
	if s.Lit != nil {
		s.Lit.Eye = v.eye
	}
	var px [][]color.RGBA
	switch {
	case len(s.scenery) > 0:
		px = s.drawScenery(sh, f, v)
	case s.Sky != nil:
		px = sh.Paint(s.Sky(sh, f.At))
	case s.Doc != nil:
		st, err := s.Doc.At(f.Since.Seconds())
		if err == nil {
			px = sh.Paint(st)
		}
	}
	if px == nil {
		px = dark(cols*2, rows*3)
	}

	// with scenery, the solids and the sprites were drawn into the light
	// the frame is made of, before the glow and the exposure. Without it
	// they go over the picture, as they always have -- and there is no
	// light to expose, so a camera's exposure does nothing to such a scene.
	if len(s.scenery) == 0 {
		if s.Lit != nil {
			compose.Blit(
				px, s.Lit.Trace(f.Since, cols, rows), s.Colours,
				0, 0,
			)
		}
		for _, p := range view.Resolve(
			s.Objects, v.eye, f.Since, sh.W, sh.H,
		) {
			compose.Blit(px, p.Frame, s.Colours, p.X, p.Y)
		}
	}
	return px
}

// dark is a sheet of the colour behind everything.
func dark(w, h int) [][]color.RGBA {
	px := make([][]color.RGBA, h)
	for y := range px {
		px[y] = make([]color.RGBA, w)
		for x := range px[y] {
			px[y][x] = color.RGBA{10, 9, 16, 255}
		}
	}
	return px
}
