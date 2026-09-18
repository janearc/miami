package scene

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/janearc/libtheme-css/primitives/functions"
	"github.com/janearc/libtheme-css/spaces/ok"
)

// Scenery, as a document says it: space, a black hole, a planet, particles.
//
// Each of these has thirty-odd numbers, and almost none of them are shared
// with a ball or a clock, so they are not more fields on Shape. A shape of
// kind "blackhole" carries a "hole" block, a "planet" a "world" block, and
// so on, and a block on the wrong kind is refused.
//
// Every block starts from a look -- a named set of numbers someone already
// made worth looking at -- and says only what it changes:
//
//	{ "kind": "blackhole", "hole": { "look": "ember", "bright": "@glow" } }
//
// So a field left out means "as the look has it", and that is why every
// number here is a pointer: nought is a real value for most of them, and a
// sentinel that collides with a real value is not a sentinel.
//
// Which looks exist is the app's business, not this package's; a look nobody
// has is refused when the stage is built, before anything is drawn.

// Hole is a black hole's block.
type Hole struct {
	Look    string `json:"look"` // ember, azure, quasar
	Mass    *Value `json:"mass"`
	Axis    *Vec   `json:"axis"`
	Inner   *Value `json:"inner"`
	Outer   *Value `json:"outer"`
	Turn    *Value `json:"turn"`
	Beaming *Value `json:"beaming"`
	Heat    *Value `json:"heat"`
	Bright  *Value `json:"bright"`
	Opacity *Value `json:"opacity"`
	Grain   *Value `json:"grain"`
	Seed    *Value `json:"seed"`
	Samples int    `json:"samples"`
	Disk    []Stop `json:"disk"`
	Space   *Space `json:"space"`
}

// Space is the sky's block: for a "space" shape, or inside a hole.
type Space struct {
	Look    string `json:"look"` // night; a hole's own sky if empty
	Seed    *Value `json:"seed"`
	Stars   *Value `json:"stars"`
	Twinkle *Value `json:"twinkle"`
	Nebula  *Value `json:"nebula"`
	Dust    *Value `json:"dust"`
	Band    *Vec   `json:"band"`
	Width   *Value `json:"width"`
	Glow    *Value `json:"glow"`
	Ambient string `json:"ambient"` // a hex colour: a lit sky
	Gas     []Stop `json:"gas"`
}

// World is a planet's block.
type World struct {
	Look     string `json:"look"` // terran, arid, frozen, molten
	Tiles    *Value `json:"tiles"`
	Seed     *Value `json:"seed"`
	Ocean    *Value `json:"ocean"`
	Plates   *Value `json:"plates"`
	Cold     *Value `json:"cold"`
	Axis     *Vec   `json:"axis"`
	Day      *Value `json:"day"`
	Sun      *Vec   `json:"sun"`
	Lines    *Value `json:"lines"`
	Relief   *Value `json:"relief"`
	Clouds   *Value `json:"clouds"`
	Air      string `json:"air"` // a hex colour
	AirDepth *Value `json:"airdepth"`
	Cities   *Value `json:"cities"`

	// Colours and Glow are by biome name: deep, sea, ice, beach, desert,
	// grass, forest, jungle, tundra, rock, snow.
	Colours map[string]string  `json:"colours"`
	Glow    map[string]float64 `json:"glow"`
}

// Emit is a particle emitter's block. Unlike the others it has no look:
// there is nothing to start from that is more use than the numbers.
type Emit struct {
	Kind    string `json:"kind"` // fountain, jet, spiral, drift
	Seed    *Value `json:"seed"`
	Axis    *Vec   `json:"axis"`
	Rate    *Value `json:"rate"`
	Life    *Value `json:"life"`
	Speed   *Value `json:"speed"`
	Spread  *Value `json:"spread"`
	Gravity *Vec   `json:"gravity"`
	Drag    *Value `json:"drag"`
	Radius  *Value `json:"radius"`
	Inner   *Value `json:"inner"`
	Wobble  *Value `json:"wobble"`
	Size    *Value `json:"size"`
	Light   *Value `json:"light"`
	Streak  *Value `json:"streak"`
	Ramp    []Stop `json:"ramp"`
}

// Glow is how a frame with scenery in it is brought down to the screen: how
// much the light is scaled, and how it spreads.
type Glow struct {
	Exposure  float64     `json:"exposure"`
	Threshold float64     `json:"threshold"`
	Spread    []GlowScale `json:"spread"`
}

// GlowScale is one radius of glow, in columns, and how strong it is.
type GlowScale struct {
	Radius   float64 `json:"radius"`
	Strength float64 `json:"strength"`
}

// emitKinds are the particle emitters a document may name.
var emitKinds = map[string]bool{
	"fountain": true, "jet": true, "spiral": true, "drift": true,
}

// UnmarshalJSON reads an emitter and refuses a kind that is not one.
func (e *Emit) UnmarshalJSON(b []byte) error {
	type plain Emit
	var p plain
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&p); err != nil {
		return fmt.Errorf("emit: %w", err)
	}
	if !emitKinds[p.Kind] {
		return fmt.Errorf("emit: unknown kind %q; want fountain, jet, "+
			"spiral or drift", p.Kind)
	}
	*e = Emit(p)
	return nil
}

// sceneryBlocks checks a shape carries the block its kind needs and no block
// that belongs to another kind.
func (s *Shape) sceneryBlocks() error {
	has := map[string]bool{"hole": s.Hole != nil, "world": s.World != nil,
		"space": s.Space != nil, "emit": s.Emit != nil}
	want := map[string]string{"blackhole": "hole", "planet": "world",
		"space": "space", "emitter": "emit"}[s.Kind]
	for block, there := range has {
		if there && block != want {
			return fmt.Errorf("a %s cannot have a %q block", s.Kind,
				block)
		}
	}
	if s.Kind == "emitter" && s.Emit == nil {
		return fmt.Errorf(`an emitter needs an "emit" block saying ` +
			`what kind: fountain, jet, spiral or drift`)
	}
	return s.sceneryNumbers()
}

// sceneryNumbers refuses a number that cannot draw anything.
//
// Two kinds of wrong, and both were accepted before. A spiral with no radius
// divided by nought, so every particle's place became a NaN and the emitter
// drew nothing while saying nothing.
//
// And an explicit minus sign was thrown away, because the code asked whether
// a number was above nought, which a document that said nothing also answers.
//
// So: absent stays absent and means the look's, and a number the document
// actually wrote has to be one that works.
func (s *Shape) sceneryNumbers() error {
	if s.Kind == "planet" && s.Radius < 0 {
		return fmt.Errorf("a planet's radius is %g; leave it out for "+
			"the look's own", s.Radius)
	}
	if s.Hole != nil && s.Hole.Samples < 0 {
		return fmt.Errorf("a hole samples %d rays; it is how many "+
			"a side an edge gets", s.Hole.Samples)
	}
	if s.World != nil {
		for _, f := range []struct {
			name string
			v    *Value
		}{{"tiles", s.World.Tiles}, {"plates", s.World.Plates}} {
			if err := above(f.name, f.v, 0); err != nil {
				return err
			}
		}
	}
	if s.Emit != nil {
		return s.Emit.numbers()
	}
	return nil
}

// numbers refuses an emitter that cannot throw anything.
func (e *Emit) numbers() error {
	for _, f := range []struct {
		name string
		v    *Value
	}{{"rate", e.Rate}, {"life", e.Life}, {"radius", e.Radius}} {
		if err := above(f.name, f.v, 0); err != nil {
			return err
		}
	}
	// a spiral turns about a ring and a drift fills a box, and neither has
	// a sensible size to fall back on
	needs := e.Kind == "spiral" || e.Kind == "drift"
	if needs && e.Radius == nil {
		return fmt.Errorf("a %s needs a radius: the ring it starts "+
			"on, or the box it fills", e.Kind)
	}
	return nil
}

// above refuses a number the document wrote that is not above a floor. A
// binding is left alone: what it is worth cannot be known until it is drawn.
func above(name string, v *Value, floor float64) error {
	if v == nil {
		return nil
	}
	if n, ok := v.Fixed(); ok && n <= floor {
		return fmt.Errorf("%s is %g; it must be more than %g", name, n,
			floor)
	}
	return nil
}

// Moves is whether a colour travels over the scene, so a ramp made of fixed
// colours can be made once instead of every frame.
func (c Colour) Moves() bool { return c.fixed == "" }

// RampAt is a ramp's stops as libtheme's ramp at a moment, mixed in oklab.
func RampAt(stops []Stop, c Ctx) functions.Ramp {
	x := ctx{t: c.T, dur: c.Dur, e: c.E, c: c.Const}
	out := make([]functions.Stop, len(stops))
	for i, s := range stops {
		out[i] = functions.Stop{At: s.At, Swatch: s.C.at(x)}
	}
	return functions.New(ok.Mix, out...)
}

// RampMoves is whether any stop of a ramp travels.
func RampMoves(stops []Stop) bool {
	for _, s := range stops {
		if s.C.Moves() {
			return true
		}
	}
	return false
}
