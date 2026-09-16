package app

import (
	"fmt"
	"time"

	"github.com/janearc/miami/grade"
	"github.com/janearc/miami/scene"
	"github.com/janearc/miami/sky"
	"github.com/janearc/miami/texture"
	"github.com/janearc/miami/view"
)

// Turning a document into a stage.
//
// This is the half that makes a scene data rather than code. A program that
// wants a sky knowing the hour, a ball bouncing on a lit floor, a clock and a
// progress bar writes none of it: it names a file, asks for the bar by name,
// and runs its own loop.
//
//	st, _ := app.NewStage("scenes/clock.json", "made/starling.css")
//	bar := st.Bar("build")
//	go work(func(d, n int) { bar.Set(d, n) })
//	app.Screen{Cols: 92}.Run(st, func(f Frame) bool { return !done })
//
// What the document cannot know, the runtime supplies as constants a binding
// reaches with "@name". There are three, and they are the three no file can
// work out for itself:
//
//	@daylight   nought at sunrise, one at sunset, flat outside those
//	@sun        the sun's height, minus one at midnight to one at noon
//	@hour       the hour of the day, with its minutes as a fraction
//
// Which is how a bar's progress becomes the length of the day without a line of
// Go: "value": "@daylight".

// runtimeConsts is what the program knows and the document cannot.
func runtimeConsts(at time.Time) map[string]float64 {
	s := view.SunAt(at)
	h := float64(at.Hour()) + float64(at.Minute())/60 +
		float64(at.Second())/3600

	// sunrise to sunset is six to eighteen in the model the sun uses, so
	// the fraction of the day gone is the hour placed in that span.
	//
	// clamped rather than wrapped: before dawn the day has not started and
	// after dusk it is done, and for a bar both of those are the honest
	// answer. a bar that wrapped at midnight would run backwards.
	day := (h - 6) / 12
	if day < 0 {
		day = 0
	}
	if day > 1 {
		day = 1
	}
	return map[string]float64{"daylight": day, "sun": s.Elevation, "hour": h}
}

// fromDoc builds everything a document describes.
func (s *Stage) fromDoc(d *scene.Doc) error {
	if d.Eye != nil {
		s.eye = view.Eye{At: vecAt(d.Eye.At, scene.Ctx{}), Lens: d.Eye.Lens,
			Near: d.Eye.Near, Yaw: d.Eye.Yaw, Pitch: d.Eye.Pitch}
	}
	for i, sh := range d.Shapes {
		if err := s.addShape(sh); err != nil {
			return fmt.Errorf("shape %d (%s): %w", i, sh.Kind, err)
		}
	}
	if len(s.solids) > 0 || len(s.lights) > 0 {
		s.Lit = &view.Lit{Eye: s.eye, Solids: s.solids, Lights: s.lights}
	}
	return nil
}

// addShape puts one shape where it belongs: in space, in the backdrop, or over
// the top. Which of those it is the only thing its kind decides.
func (s *Stage) addShape(sh scene.Shape) error {
	switch sh.Kind {
	case "sky":
		if sh.Sun {
			s.Sky = func(g grade.Sheet, at time.Time) grade.Stack {
				aspect := (float64(g.H) * 4) / (float64(g.W) * 3)
				return sky.At(g, view.SunAt(at), s.eye.HorizonV(aspect))
			}
		}
		return nil

	case "light":
		s.lights = append(s.lights, view.Light{})
		s.sun = append(s.sun, sh.Sun)
		s.lightAt = append(s.lightAt, sh.At)
		return nil

	case "floor":
		shade, err := sh.ShadeMap()
		if err != nil {
			return err
		}
		tex, err := textureFrom(sh.Texture, s)
		if err != nil {
			return err
		}
		a, b, err := abRoles(sh)
		if err != nil {
			return err
		}
		s.solids = append(s.solids, view.Floor{Y: sh.Y, Tile: sh.Tile,
			Far: sh.Far, A: a, B: b, Shade: shade, Texture: tex})
		return nil

	case "sphere":
		tex, err := textureFrom(sh.Texture, s)
		if err != nil {
			return err
		}
		b := &view.Ball{Radius: sh.Radius, Texture: tex,
			Spin: vecAt(sh.Spin, scene.Ctx{})}
		s.solids = append(s.solids, b)
		s.moving = append(s.moving, mover{at: sh.At, set: func(v view.Vec) {
			b.At = v
		}})
		s.remember(sh.Name, b)
		return nil

	case "cylinder":
		tex, err := textureFrom(sh.Texture, s)
		if err != nil {
			return err
		}
		c := &view.Cylinder{Radius: sh.Radius, Length: sh.Length,
			Caps: sh.Caps, Texture: tex, Axis: vecAt(sh.Axis, scene.Ctx{})}
		s.solids = append(s.solids, c)
		s.moving = append(s.moving, mover{at: sh.At, set: func(v view.Vec) {
			c.At = v
		}})
		s.remember(sh.Name, c)
		return nil

	case "clock":
		d := &Digits{Font: sh.Font, Row: sh.Row, Col: sh.Col,
			Ink: Ramp(sh.Ink), Scrim: sh.Scrim, Format: sh.Format,
			Flicker: sh.Flicker}
		s.Add(d)
		s.remember(sh.Name, d)
		return nil

	case "text":
		t := &Text{Font: sh.Font, Row: sh.Row, Col: sh.Col,
			Ink: Ramp(sh.Ink), Scrim: sh.Scrim}
		t.Set(sh.Text)
		s.Add(t)
		s.remember(sh.Name, t)
		return nil

	case "bar":
		b := &Bar{Row: sh.Row, Col: sh.Col, Width: sh.Width,
			Ink: Ramp(sh.Ink), Label: sh.Label, Scrim: sh.Scrim}
		if sh.Value != nil {
			v := *sh.Value
			b.From = func(f Frame) float64 { return v.At(f.Scene) }
		}
		s.Add(b)
		s.remember(sh.Name, b)
		return nil
	}
	return fmt.Errorf("no way to build it") // the kind was checked on read
}

// abRoles is a floor's two roles, defaulting to the pair every floor in the
// library used before a floor could be given a texture instead.
func abRoles(sh scene.Shape) (byte, byte, error) {
	a, b := byte('V'), byte('K')
	if sh.Texture != nil {
		return a, b, nil // the texture decides; these are never asked
	}
	return a, b, nil
}

// textureFrom builds a pattern, or nil for a shape that did not name one.
func textureFrom(t *scene.Texture, s *Stage) (texture.Texture, error) {
	if t == nil {
		return nil, nil
	}
	role := func(x string) (byte, error) { return scene.Role(x) }
	switch t.Kind {
	case "quadrants":
		return view.Quadrants(), nil

	case "solid":
		a, err := role(t.A)
		if err != nil {
			return nil, err
		}
		return texture.Solid(a), nil

	case "checker":
		a, b, err := twoRoles(t)
		if err != nil {
			return nil, err
		}
		return texture.Checker{A: a, B: b, U: t.U, V: t.V,
			PhaseU: t.PhaseU, PhaseV: t.PhaseV}, nil

	case "stripes":
		a, b, err := twoRoles(t)
		if err != nil {
			return nil, err
		}
		return texture.Stripes{A: a, B: b, Period: t.Period,
			Duty: t.Duty, Phase: t.Phase}, nil

	case "noise":
		rs := make([]byte, 0, len(t.Roles))
		for _, r := range t.Roles {
			b, err := role(r)
			if err != nil {
				return nil, err
			}
			rs = append(rs, b)
		}
		return texture.Noise{Roles: rs, Scale: t.Scale}, nil

	case "bands":
		of := make([]texture.Texture, 0, len(t.Of))
		for i := range t.Of {
			sub, err := textureFrom(&t.Of[i], s)
			if err != nil {
				return nil, fmt.Errorf("band %d: %w", i, err)
			}
			of = append(of, sub)
		}
		return texture.Bands{Of: of}, nil

	case "split":
		done, err := textureFrom(t.Done, s)
		if err != nil {
			return nil, fmt.Errorf("done: %w", err)
		}
		rest, err := textureFrom(t.Rest, s)
		if err != nil {
			return nil, fmt.Errorf("rest: %w", err)
		}
		sp := texture.Split{Done: done, Rest: rest}
		if t.Frac != nil {
			v := *t.Frac
			// read through the stage rather than captured, because
			// a split is asked per subpixel and the moment it is
			// asked about is the one the frame is being drawn at
			sp.Frac = func() float64 { return v.At(s.now()) }
		}
		return sp, nil
	}
	return nil, fmt.Errorf("unknown texture %q", t.Kind)
}

// twoRoles is a texture's pair.
func twoRoles(t *scene.Texture) (byte, byte, error) {
	a, err := scene.Role(t.A)
	if err != nil {
		return 0, 0, fmt.Errorf("a: %w", err)
	}
	b, err := scene.Role(t.B)
	if err != nil {
		return 0, 0, fmt.Errorf("b: %w", err)
	}
	return a, b, nil
}

// vecAt resolves a document point.
func vecAt(v scene.Vec, c scene.Ctx) view.Vec {
	x, y, z := v.At(c)
	return view.Vec{X: x, Y: y, Z: z}
}

// remember files a shape under its name, so a program can find it.
func (s *Stage) remember(name string, v any) {
	if name == "" {
		return
	}
	s.mu.Lock()
	if s.named == nil {
		s.named = map[string]any{}
	}
	s.named[name] = v
	s.mu.Unlock()
}

// Bar is the bar of that name, or nil.
//
// nil rather than an error, because a program that asks for a bar the scene
// does not have should carry on drawing the scene. The alternative is a build
// tool that will not build because its wallpaper is missing a widget.
func (s *Stage) Bar(name string) *Bar { b, _ := s.find(name).(*Bar); return b }

// Text is the line of type of that name, or nil.
func (s *Stage) Text(name string) *Text { t, _ := s.find(name).(*Text); return t }

// Sphere is the ball of that name, or nil.
func (s *Stage) Sphere(name string) *view.Ball {
	b, _ := s.find(name).(*view.Ball)
	return b
}

// Cylinder is the tube of that name, or nil.
func (s *Stage) Cylinder(name string) *view.Cylinder {
	c, _ := s.find(name).(*view.Cylinder)
	return c
}

// find is a named shape, whatever kind it is.
func (s *Stage) find(name string) any {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.named[name]
}

// place puts everything that moves where this moment says, before the frame is
// traced.
func (s *Stage) place(f Frame) {
	for _, m := range s.moving {
		m.set(vecAt(m.at, f.Scene))
	}
	for i := range s.lights {
		if s.sun[i] {
			s.lights[i] = view.SunAt(f.At).Light()
			continue
		}
		s.lights[i] = view.Light{At: vecAt(s.lightAt[i], f.Scene)}
	}
	if s.Lit != nil {
		s.Lit.Lights = s.lights
		s.Lit.Eye = s.eye
	}
}

// Digits is the clock of that name, or nil.
func (s *Stage) Digits(name string) *Digits {
	d, _ := s.find(name).(*Digits)
	return d
}

// Eye is the camera, to be steered. A pointer because steering it is the point:
// a caller nudging yaw wants the scene to turn, not a copy of it.
func (s *Stage) Eye() *view.Eye { return &s.eye }
