// Package scene reads a painted scene from JSON.
//
// A grade layer is already data -- a field, a ramp, an alpha, a blend and a
// gain -- so the stack was always a document pretending to be Go. What was not
// obvious is that the layers are the easy half.
//
// Three things in a moving scene resist being written down, and how each is
// handled is the whole of this package:
//
//	a value that changes    the sun's height, the gain of a glow going out
//	a colour that changes   a sky whose every ramp stop travels at dusk
//	a field with maths in   a sum of drifting waves, a field of stars
//
// The first two are solved by letting every number and every colour be a
// binding rather than a constant. The third is solved by naming them: they are
// primitives, not scene logic, and a scene that needs a shape none of the
// primitives describe needs a new primitive, not an escape hatch.
//
// There is deliberately no expression language. The moment a scene file can
// evaluate arithmetic it is a programming language with no debugger, no line
// numbers and no way to test a fragment.
//
// A binding is a shape -- travel from here to there, oscillate about a mid --
// and a shape can be reported, drawn and checked. Anything a binding cannot say
// belongs in Go, behind a name.
//
// The document, in full:
//
//	{
//	  "name": "dusk", "seconds": 20,
//	  "const": { "horizon": 0.62 },
//	  "layers": [ { "name", "field", "ramp", "alpha", "blend", "gain" } ]
//	}
//
// A value is a number, "@name" for a constant, or one of:
//
//	{"from": a, "to": b, "ease": "smooth"}   travels once across the scene
//	{"mid": m, "amp": a, "period": s}        oscillates, seconds per turn
//	{"mul": [v, v]}  {"add": [v, v]}  {"clamp": v}  {"abs": v}
//
// A colour is a hex string or {"from": hex, "to": hex, "ease": "smooth"}.
package scene

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"

	"github.com/janearc/libdaffy/field"
	"github.com/janearc/libtheme-css/primitives/functions"
	"github.com/janearc/libtheme-css/primitives/swatch"
	"github.com/janearc/libtheme-css/spaces/ok"
	"github.com/janearc/libtheme-css/spaces/srgb"

	"github.com/janearc/miami/grade"
	colours "github.com/janearc/miami/internal/hex"
)

// Doc is a whole scene.
type Doc struct {
	Name    string             `json:"name"`
	Seconds float64            `json:"seconds"`
	Const   map[string]float64 `json:"const"`
	Layers  []Layer            `json:"layers"`

	// The rest of the composition. All optional: a document with only
	// layers is the picture this package started as, and still works.
	Eye    *Eye    `json:"eye"`
	Shapes []Shape `json:"shapes"`

	// Cameras and Viewports are more than one picture of the scene at
	// once. A document with neither is one camera filling the sheet, which
	// is what "eye" has always meant; see Shots.
	Cameras   []Camera   `json:"cameras"`
	Viewports []Viewport `json:"viewports"`

	// Glow is how a frame with scenery in it is brought to the screen.
	// Without scenery it is not read.
	Glow *Glow `json:"glow"`

	// src is the document with its comments taken out, kept so the names
	// it reads can be found wherever they are -- see Refs.
	src []byte
}

// Layer is one gradient, exactly as grade means it.
type Layer struct {
	Name  string `json:"name"`
	Field Node   `json:"field"`
	Ramp  []Stop `json:"ramp"`
	Alpha *Node  `json:"alpha,omitempty"`
	Blend string `json:"blend,omitempty"`
	Gain  Value  `json:"gain"`
}

// Stop is a ramp stop, whose colour may itself travel.
type Stop struct {
	At float64 `json:"at"`
	C  Colour  `json:"c"`
}

// ctx is everything a binding is allowed to know: where the scene is in its own
// time, and the constants it declared. Not the sheet, not the pixel -- a
// binding is a function of time and nothing else, which is what keeps it cheap
// enough to evaluate once per frame instead of once per sample.
type ctx struct {
	t, dur, e float64
	c         map[string]float64
}

// Load reads a scene document.
func Load(path string) (*Doc, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Read(path, b)
}

// Read is a document that is already in hand, named only so an error can say
// which one it was. For a caller holding bytes rather than a path: a document
// compiled into a program, or one that arrived down a pipe.
func Read(name string, b []byte) (*Doc, error) {
	trimmed, err := io.ReadAll(newTrimmer(b))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	var d Doc
	dec := json.NewDecoder(bytes.NewReader(trimmed))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&d); err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	if d.Seconds <= 0 {
		d.Seconds = 1
	}
	d.src = trimmed
	// refused here, with the document's name, rather than when a stage is
	// built from it and the name is gone
	if _, _, err := d.Shots(); err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	return &d, nil
}

// At builds the stack for a moment in the scene.
func (d *Doc) At(t float64) (grade.Stack, error) {
	c := ctx{t: t, dur: d.Seconds, e: clamp01(t / d.Seconds), c: d.Const}
	out := make(grade.Stack, 0, len(d.Layers))
	for _, l := range d.Layers {
		f, err := l.Field.field(c)
		if err != nil {
			return nil, fmt.Errorf(
				"layer %q field: %w", l.Name, err,
			)
		}
		var a field.Field
		if l.Alpha != nil {
			if a, err = l.Alpha.field(c); err != nil {
				return nil, fmt.Errorf(
					"layer %q alpha: %w", l.Name, err,
				)
			}
		}
		blend, err := blendOf(l.Blend)
		if err != nil {
			return nil, fmt.Errorf("layer %q: %w", l.Name, err)
		}
		stops := make([]functions.Stop, len(l.Ramp))
		for i, s := range l.Ramp {
			stops[i] = functions.Stop{At: s.At, Swatch: s.C.at(c)}
		}
		out = append(out, grade.Layer{
			Name: l.Name, Field: f, Alpha: a, Blend: blend,
			Gain: l.Gain.at(c), Ramp: functions.New(
				ok.Mix, stops...,
			),
		})
	}
	return out, nil
}

// blendOf is the blend a document names, and refuses one it does not know.
func blendOf(s string) (grade.Blend, error) {
	switch s {
	case "", "over":
		return grade.Over, nil
	case "add":
		return grade.Add, nil
	case "mul":
		return grade.Mul, nil
	}
	return 0, fmt.Errorf("unknown blend %q", s)
}

// ---------------------------------------------------------------- values

// Value is a number that may change with time.
type Value struct {
	fixed  *float64
	ref    string
	from   *Value
	to     *Value
	ease   string
	mid    *Value
	amp    *Value
	period *Value
	phase  *Value
	mul    []Value
	add    []Value
	clamp  *Value
	abs    *Value
}

// UnmarshalJSON reads a value: a number, an @constant, or a binding.
func (v *Value) UnmarshalJSON(b []byte) error {
	var n float64
	if json.Unmarshal(b, &n) == nil {
		v.fixed = &n
		return nil
	}
	var s string
	if json.Unmarshal(b, &s) == nil {
		if len(s) < 2 || s[0] != '@' {
			return fmt.Errorf(
				"a value string must be \"@constant\", got %q",
				s,
			)
		}
		v.ref = s[1:]
		return nil
	}
	var raw struct {
		From, To, Mid, Amp, Period, Phase, Clamp, Abs *Value
		Ease                                          string
		Mul, Add                                      []Value
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return fmt.Errorf("value: %w", err)
	}
	v.from, v.to, v.ease = raw.From, raw.To, raw.Ease
	v.mid, v.amp = raw.Mid, raw.Amp
	v.period, v.phase = raw.Period, raw.Phase
	v.mul, v.add, v.clamp, v.abs = raw.Mul, raw.Add, raw.Clamp, raw.Abs
	return nil
}

// at is what this value is at a moment in the scene.
func (v Value) at(c ctx) float64 {
	switch {
	case v.fixed != nil:
		return *v.fixed
	case v.ref != "":
		return c.lookup(v.ref)
	case v.from != nil && v.to != nil:
		return v.from.at(c) + (v.to.at(c)-v.from.at(c))*curve(
			v.ease, c.e,
		)
	case v.period != nil:
		p := v.period.at(c)
		if p == 0 {
			p = 1
		}
		ph := 0.0
		if v.phase != nil {
			ph = v.phase.at(c)
		}
		a, m := 1.0, 0.0
		if v.amp != nil {
			a = v.amp.at(c)
		}
		if v.mid != nil {
			m = v.mid.at(c)
		}
		return m + a*math.Sin(2*math.Pi*(c.t/p+ph))
	case len(v.mul) > 0:
		r := 1.0
		for _, x := range v.mul {
			r *= x.at(c)
		}
		return r
	case len(v.add) > 0:
		r := 0.0
		for _, x := range v.add {
			r += x.at(c)
		}
		return r
	case v.clamp != nil:
		return clamp01(v.clamp.at(c))
	case v.abs != nil:
		// a bounce. a sine goes below the floor and comes back, and
		// the half below is the ball underground; folding it up is
		// what turns an oscillation into something landing.
		return math.Abs(v.abs.at(c))
	}
	return 0
}

// curve is an easing, by name.
func curve(name string, e float64) float64 {
	e = clamp01(e)
	switch name {
	case "", "linear":
		return e
	case "smooth":
		return e * e * (3 - 2*e)
	case "in":
		return e * e
	case "out":
		return 1 - (1-e)*(1-e)
	}
	return e
}

// Colour is a hex colour that may travel.
type Colour struct {
	fixed    string
	from, to string
	ease     string
}

// Hex is a colour as a document writes one: six hex digits, in whichever
// case, with or without a hash in front.
//
// A type rather than a string, so that a colour nobody can draw is refused
// when the document is read rather than drawn as something else.
//
// Swatch took any string and read anything but a lower-case hex digit as
// nought: "FF6EC7" was nearly black and "#ff6ec7" a different colour, both
// without a word.
type Hex string

// UnmarshalJSON reads a colour and refuses what cannot be one.
func (h *Hex) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return fmt.Errorf("colour: %w", err)
	}
	if !grade.IsColour(s) {
		return fmt.Errorf("%q is not a colour: six hex digits, such "+
			"as ff6ec7, with or without a hash", s)
	}
	*h = Hex(s)
	return nil
}

// UnmarshalJSON reads a colour: a hex string, or one that travels.
func (c *Colour) UnmarshalJSON(b []byte) error {
	var s string
	if json.Unmarshal(b, &s) == nil {
		if !grade.IsColour(s) {
			return fmt.Errorf("%q is not a colour: six hex "+
				"digits, such as ff6ec7, with or without a "+
				"hash", s)
		}
		c.fixed = s
		return nil
	}
	var raw struct{ From, To, Ease string }
	if err := json.Unmarshal(b, &raw); err != nil {
		return fmt.Errorf("colour: %w", err)
	}
	for _, end := range []string{raw.From, raw.To} {
		if !grade.IsColour(end) {
			return fmt.Errorf("%q is not a colour: six hex "+
				"digits, such as ff6ec7, with or without a "+
				"hash", end)
		}
	}
	c.from, c.to, c.ease = raw.From, raw.To, raw.Ease
	return nil
}

// at is what this colour is at a moment in the scene.
func (c Colour) at(x ctx) swatch.Swatch {
	if c.fixed != "" {
		return hex(c.fixed)
	}
	// each stop travels on its own. a sky that changes colour is not a
	// crossfade between two pictures -- that goes through a grey middle
	// where the two disagree -- it is one ramp whose stops each move.
	return ok.Mix.Mix(hex(c.from), hex(c.to), curve(c.ease, x.e))
}

// hex is a colour written as text, as a physical colour. It reads what the
// one parser reads, so a colour Read accepted is the colour Read draws.
func hex(s string) swatch.Swatch {
	r, g, b, _ := colours.Bytes(s)
	return srgb.RGB{R: float64(r) / 255, G: float64(g) / 255,
		B: float64(b) / 255}.Swatch()
}

// clamp01 holds a number to nought and one.
func clamp01(v float64) float64 { return math.Max(0, math.Min(1, v)) }

// newTrimmer strips // line comments outside of strings.
//
// JSON has no comments and a scene file badly wants them: which layer is which,
// why a period is 19.1 and not 19. The house argument against JSON was that
// plain text reads in a terminal and can be hand-edited, and a document nobody
// can annotate loses half of that.
//
// Stripping comments on the way in costs twenty lines and gives it back.
func newTrimmer(b []byte) *bytes.Reader {
	out := make([]byte, 0, len(b))
	inStr, esc := false, false
	for i := 0; i < len(b); i++ {
		switch {
		case esc:
			esc = false
		case inStr && b[i] == '\\':
			esc = true
		case b[i] == '"':
			inStr = !inStr
		case !inStr && b[i] == '/' && i+1 < len(b) && b[i+1] == '/':
			for i < len(b) && b[i] != '\n' {
				i++
			}
			out = append(out, '\n')
			continue
		}
		out = append(out, b[i])
	}
	return bytes.NewReader(out)
}

// built-in constants a binding may name. Scene progress and elapsed seconds are
// the two every moving scene needs and neither belongs in the document.
func (c ctx) lookup(name string) float64 {
	switch name {
	case "e":
		return c.e
	case "t":
		return c.t
	case "dur":
		return c.dur
	}
	return c.c[name]
}
