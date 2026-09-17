package scene

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// The rest of a scene: what stands in it, and what is written over it.
//
// Layers describe a painted backdrop and nothing else, which was enough while a
// document was a picture. It is not enough now that a document is what another
// program runs.
//
// A build tool asking miami for a scene wants a sky that knows the hour, a ball
// on a floor, a clock and a bar showing how far along it is, and it should not
// have to write any of that in Go to get it.
//
// So a document is a list of shapes. One list, not four: a sphere, a floor, a
// sky and a clock are all things in a scene with attributes, and sorting them
// by how they happen to be drawn is the library's problem rather than the
// author's.
//
//	sky       a backdrop that knows the time of day
//	floor     the plane everything stands on
//	sphere    a ball
//	cylinder  a tube, which is what a progress bar is
//	light     a lamp, or the sun
//	clock     the time, with separators that blink
//	text      a line of type
//	bar       a flat progress bar, for a scene with no room for a cylinder
//	emoji     a picture from the system font
//	log       a box a program talks into, in plain glyphs
//
// What this package deliberately does not do is know how any of it is drawn.
// There are no view or app types here, only numbers and names: this package
// parses and evaluates, and app translates. That is why a document cannot
// smuggle drawing logic in, and why the dependency points one way.
//
// The bindings are the ones layers already had. A ball's path is three values
// that change with time, which is exactly what a gain or a horizon was, so
// nothing new was needed to make it move:
//
//	"at": { "x": {"mid": 0, "amp": 1.3, "period": 60},
//	        "y": {"abs": {"mid": 0, "amp": 0.66, "period": 4}},
//	        "z": {"mid": 3.6, "amp": 0.7, "period": 60, "phase": 0.25} }

// Ctx is a moment, for a caller outside this package evaluating a binding.
//
// T is seconds into the scene, Dur its length and E the fraction of it done.
// Const is what "@name" looks up, and holds both the document's own constants
// and whatever the runtime supplies -- see Doc.Ctx.
type Ctx struct {
	T, Dur, E float64
	Const     map[string]float64
}

// At is what a value is at a moment.
func (v Value) At(c Ctx) float64 {
	return v.at(ctx{t: c.T, dur: c.Dur, e: c.E, c: c.Const})
}

// Vec is a point whose every coordinate may change with time.
type Vec struct {
	X Value `json:"x"`
	Y Value `json:"y"`
	Z Value `json:"z"`
}

// At resolves the point.
func (v Vec) At(c Ctx) (x, y, z float64) {
	return v.X.At(c), v.Y.At(c), v.Z.At(c)
}

// Eye is the camera.
type Eye struct {
	At    Vec     `json:"at"`
	Lens  float64 `json:"lens"`
	Near  float64 `json:"near"`
	Yaw   float64 `json:"yaw"`
	Pitch float64 `json:"pitch"`
}

// Shape is one thing in a scene.
//
// One struct with a Kind rather than a struct per kind. The fields overlap
// almost entirely -- where it is, how big, what it is made of -- and eight
// near-identical types would drift apart.
//
// What keeps it honest is that an unknown kind is refused when the document is
// read, rather than ignored and missing from the picture.
type Shape struct {
	Kind string `json:"kind"`

	// Name is what a caller asks for it by. A bar the program drives has to
	// be findable from the program, and a document with two of anything
	// needs them told apart.
	Name string `json:"name"`

	// In space.
	At      Vec               `json:"at"`
	Spin    Vec               `json:"spin"`
	Axis    Vec               `json:"axis"`
	Radius  float64           `json:"radius"`
	Length  float64           `json:"length"`
	Caps    bool              `json:"caps"`
	Y       float64           `json:"y"`    // floor: its height
	Tile    float64           `json:"tile"` // floor: world units to a square
	Far     float64           `json:"far"`  // floor: where it stops being drawn
	Texture *Texture          `json:"texture"`
	Shade   map[string]string `json:"shade"`

	// Painted.
	Sun bool `json:"sun"` // sky: know the time of day. light: be the sun.

	// Written over the top.
	Row     int      `json:"row"` // negative counts from the bottom
	Col     int      `json:"col"` // -1 centres it
	Ink     []string `json:"ink"` // hex colours; the house ramp if empty
	Scrim   float64  `json:"scrim"`
	Font    string   `json:"font"`
	Format  string   `json:"format"`  // clock: a go time layout
	Flicker bool     `json:"flicker"` // clock: separators out for half a second
	Text    string   `json:"text"`
	Label   string   `json:"label"`
	Width   int      `json:"width"`

	// Rune is an emoji's codepoint, as hex: "1F32B" and not the glyph.
	//
	// A document is read and edited in a terminal, and an emoji pasted into
	// one is a character nobody can tell from its neighbours -- there are
	// four different clouds and they look alike at this size.
	Rune    string `json:"rune"`
	Colours int    `json:"colours"`
	Cols    int    `json:"cols"`

	// Rows is a log's height in lines. It shows the last that many, so a
	// taller box is more history rather than the same view padded.
	Rows int `json:"rows"`

	// Value drives a bar from the document instead of from the program: a
	// binding evaluated every frame, nought to one. A bar without it is the
	// program's to Set, which is the usual case.
	Value *Value `json:"value"`
}

// kinds is every shape a document may name.
var kinds = map[string]bool{
	"sky": true, "floor": true, "sphere": true, "cylinder": true,
	"light": true, "clock": true, "text": true, "bar": true,
	"emoji": true, "log": true,
}

// UnmarshalJSON reads a shape and refuses one whose kind is not a kind.
//
// Refused rather than ignored, because a shape that silently does not appear is
// the worst failure a document has: it looks right, the picture is missing
// something, and nothing says which.
func (s *Shape) UnmarshalJSON(b []byte) error {
	type plain Shape // no method, so no recursion
	var p plain
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&p); err != nil {
		return fmt.Errorf("shape: %w", err)
	}
	if p.Kind == "" {
		return fmt.Errorf("a shape needs a kind; one of %s", kindList())
	}
	if !kinds[p.Kind] {
		return fmt.Errorf("unknown shape %q; want one of %s", p.Kind, kindList())
	}
	if (p.Kind == "bar" || p.Kind == "cylinder") && p.Name == "" && p.Value == nil {
		return fmt.Errorf("a %s that nothing drives needs a name, "+
			"so a program can find it", p.Kind)
	}
	*s = Shape(p)
	return nil
}

// kindList is the kinds, in a fixed order, for an error a person has to read.
func kindList() string {
	return "sky, floor, sphere, cylinder, light, clock, text, bar, emoji, log"
}

// Texture is a pattern, as a document says it.
type Texture struct {
	Kind string `json:"kind"`

	A string `json:"a"` // a role, one character
	B string `json:"b"`

	U      float64 `json:"u"` // checker: squares across
	V      float64 `json:"v"`
	PhaseU float64 `json:"phaseu"`
	PhaseV float64 `json:"phasev"`

	Roles []string `json:"roles"` // noise
	Scale float64  `json:"scale"`

	Period float64 `json:"period"` // stripes
	Duty   float64 `json:"duty"`
	Phase  float64 `json:"phase"`

	Of []Texture `json:"of"` // bands

	Done *Texture `json:"done"` // split
	Rest *Texture `json:"rest"`
	Frac *Value   `json:"frac"`
}

// textureKinds is every pattern a document may name.
var textureKinds = map[string]bool{
	"solid": true, "checker": true, "bands": true,
	"noise": true, "stripes": true, "split": true, "quadrants": true,
}

// UnmarshalJSON reads a texture, refusing an unknown pattern for the same
// reason a shape does.
func (t *Texture) UnmarshalJSON(b []byte) error {
	type plain Texture
	var p plain
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&p); err != nil {
		return fmt.Errorf("texture: %w", err)
	}
	if !textureKinds[p.Kind] {
		return fmt.Errorf("unknown texture %q; want solid, checker, "+
			"bands, noise, stripes, split or quadrants", p.Kind)
	}
	*t = Texture(p)
	return nil
}

// Role is a role as a single byte, refusing anything that is not one.
func Role(s string) (byte, error) {
	if len(s) != 1 {
		return 0, fmt.Errorf("a role is one character, got %q", s)
	}
	return s[0], nil
}

// ShadeMap is a shape's shade ladder as the tracer wants it.
func (s Shape) ShadeMap() (map[byte]byte, error) {
	if len(s.Shade) == 0 {
		return nil, nil
	}
	out := make(map[byte]byte, len(s.Shade))
	for k, v := range s.Shade {
		kb, err := Role(k)
		if err != nil {
			return nil, fmt.Errorf("shade key: %w", err)
		}
		vb, err := Role(v)
		if err != nil {
			return nil, fmt.Errorf("shade %q: %w", k, err)
		}
		out[kb] = vb
	}
	return out, nil
}

// Ctx builds the moment to evaluate against: the scene's own clock, the
// document's constants, and what the runtime knows that a document cannot.
//
// The runtime's values win a collision. A document can declare "daylight" and
// get a fixed one for testing, but the program's is the answer when it has one.
//
// A document cannot know the hour, and a stale constant quietly overriding the
// sun would be a picture that is wrong about the time and says nothing.
func (d *Doc) Ctx(t float64, runtime map[string]float64) Ctx {
	c := make(map[string]float64, len(d.Const)+len(runtime))
	for k, v := range d.Const {
		c[k] = v
	}
	for k, v := range runtime {
		c[k] = v
	}
	return Ctx{T: t, Dur: d.Seconds, E: clamp01(t / d.Seconds), Const: c}
}
