package app

import (
	"fmt"
	"image/color"
	"sort"
	"strings"

	"github.com/janearc/miami/grade"
	"github.com/janearc/miami/light"
	"github.com/janearc/miami/particles"
	"github.com/janearc/miami/scene"
	"github.com/janearc/miami/scenery"
	"github.com/janearc/miami/view"
)

// Scenery on a stage: space, black holes, planets and particles, from a
// document.
//
// A stage with any of these draws its frame as light first -- the painted
// layers, then the scenery back to front, then glow -- and brings it down to
// colours only at the end. A stage without any draws exactly as it always
// did, so no picture made before scenery existed moves.
//
// Each piece is built once from its look and the numbers the document
// changed, and every frame the numbers are read again, so any of them may be
// a binding: a disk's brightness on @level pulses with the music.
//
// What is expensive to change -- where a black hole's rays go, a planet's
// terrain -- is kept, and only worked out again when a number it depends on
// moves.

// piece is one piece of scenery on a stage.
type piece struct {
	order  int // backdrops, then solids, then light
	draw   func(b *light.Buffer, eye view.Eye, t float64)
	update func(c scene.Ctx) // reads the document's numbers again
}

// holeLooks, worldLooks and skyLooks are the looks a document may start from.
var (
	holeLooks = map[string]func() *scenery.BlackHole{
		"ember": scenery.Ember, "azure": scenery.Azure,
		"quasar": scenery.Quasar,
	}
	worldLooks = map[string]func() *scenery.Planet{
		"terran": scenery.Terran, "arid": scenery.Arid,
		"frozen": scenery.Frozen, "molten": scenery.Molten,
	}
	skyLooks = map[string]func() scenery.Space{"night": scenery.Night}
)

// addScenery builds a scenery shape, or says it is not one.
func (s *Stage) addScenery(sh scene.Shape, c scene.Ctx) (bool, error) {
	var p piece
	var err error
	switch sh.Kind {
	case "space":
		p, err = spacePiece(sh, c)
	case "blackhole":
		p, err = holePiece(sh, c)
	case "planet":
		p, err = planetPiece(sh, c)
	case "emitter":
		p, err = emitterPiece(sh, c)
	default:
		return false, nil
	}
	if err != nil {
		return true, err
	}
	s.scenery = append(s.scenery, p)
	sort.SliceStable(s.scenery, func(i, j int) bool {
		return s.scenery[i].order < s.scenery[j].order
	})
	return true, nil
}

// drawScenery is the frame as light: the stage's painted layers, the
// scenery, and the glow, brought down to subpixels.
func (s *Stage) drawScenery(
	sh grade.Sheet, f Frame, v look,
) [][]color.RGBA {
	b := light.New(sh.W, sh.H)
	switch {
	case s.Sky != nil:
		b.Paint(sh, s.Sky(sh, f.At))
	case s.Doc != nil && len(s.Doc.Layers) > 0:
		if st, err := s.Doc.At(f.Since.Seconds()); err == nil {
			b.Paint(sh, st)
		}
	}
	t := f.Since.Seconds()
	for _, p := range s.scenery {
		p.update(f.Scene)
		p.draw(b, v.eye, t)
	}
	s.traced(b, f)
	s.sprites(b, f, v.eye)
	g := glowOf(s.Doc)
	var spread []light.Glow
	for _, sc := range g.Spread {
		spread = append(spread, light.Glow{Radius: v.radius(sc.Radius),
			Strength: sc.Strength})
	}
	b.Bloom(g.Threshold, spread...)
	return b.RGBA(v.exposed(g.Exposure))
}

// traced puts the ray-traced solids into the frame as light.
//
// They carry no depth: the tracer answers with a role and nothing else, so
// they are drawn over whatever scenery is behind them, which is what they did
// before. A ball that should go behind a planet needs the tracer to report
// how far away it hit, and it does not yet.
func (s *Stage) traced(b *light.Buffer, f Frame) {
	if s.Lit == nil {
		return
	}
	fr := s.Lit.Trace(f.Since, b.W/2, b.H/3)
	for y, row := range fr.Art {
		for x := 0; x < len(row); x++ {
			if c, ok := s.ink(row[x]); ok {
				b.Set(x, y, c)
			}
		}
	}
}

// sprites puts the objects standing in space into the frame as light, nearest
// last, each hidden where something nearer has already been drawn.
//
// This is what lets a bird fly behind a disk. Drawn into the buffer rather
// than over the picture afterwards, a sprite is also subject to the frame's
// exposure and spreads its own glow, like everything else in the scene.
func (s *Stage) sprites(b *light.Buffer, f Frame, eye view.Eye) {
	for _, p := range view.Resolve(s.Objects, eye, f.Since, b.W, b.H) {
		for y, row := range p.Frame.Art {
			py := p.Y + y
			if py < 0 || py >= b.H {
				continue
			}
			for x := 0; x < len(row); x++ {
				px := p.X + x
				if px < 0 || px >= b.W {
					continue
				}
				c, ok := s.ink(row[x])
				if !ok || p.Depth > b.Depth(px, py) {
					continue
				}
				b.Set(px, py, c)
				b.SetDepth(px, py, p.Depth)
			}
		}
	}
}

// ink is a role's colour as light, and false for the role that is nothing or
// one the colourway did not dress.
func (s *Stage) ink(role byte) (light.RGB, bool) {
	if role == '.' {
		return light.RGB{}, false
	}
	c, ok := s.Colours[role]
	if !ok {
		return light.RGB{}, false
	}
	return light.FromRGBA(c), true
}

// glowOf is a document's glow, or the one the looks were made under.
func glowOf(d *scene.Doc) scene.Glow {
	if d != nil && d.Glow != nil {
		g := *d.Glow
		if g.Exposure == 0 {
			g.Exposure = 1
		}
		return g
	}
	spread := []scene.GlowScale{{Radius: 1.5, Strength: 0.6},
		{Radius: 8, Strength: 0.45}, {Radius: 24, Strength: 0.3}}
	return scene.Glow{Exposure: 1, Threshold: 0.6, Spread: spread}
}

// spacePiece is a sky with nothing bending it. Its numbers are read once:
// the sky in every direction is kept, and a sky that changed every frame
// would be worked out every frame.
func spacePiece(sh scene.Shape, c scene.Ctx) (piece, error) {
	sp, err := spaceFrom(sh.Space, scenery.Night(), c)
	if err != nil {
		return piece{}, err
	}
	d := &scenery.Backdrop{Space: sp}
	return piece{order: 0, draw: d.Draw, update: func(scene.Ctx) {}}, nil
}

// spaceFrom is a sky block laid over a starting sky.
func spaceFrom(b *scene.Space, base scenery.Space,
	c scene.Ctx) (scenery.Space, error) {
	if b == nil {
		return base, nil
	}
	sp := base
	if b.Look != "" {
		mk, ok := skyLooks[b.Look]
		if !ok {
			return sp, fmt.Errorf("no sky look %q; there is %s",
				b.Look, names(skyLooks))
		}
		sp = mk()
	}
	num(&sp.Stars, b.Stars, c)
	num(&sp.Twinkle, b.Twinkle, c)
	num(&sp.Nebula, b.Nebula, c)
	num(&sp.Dust, b.Dust, c)
	num(&sp.Width, b.Width, c)
	num(&sp.Glow, b.Glow, c)
	if b.Seed != nil {
		sp.Seed = uint64(b.Seed.At(c))
	}
	if b.Band != nil {
		sp.Band = vecAt(*b.Band, c)
	}
	if len(b.Gas) > 0 {
		sp.Gas = light.Bake(scene.RampAt(b.Gas, c))
	}
	return sp, nil
}

// holePiece is a black hole.
func holePiece(sh scene.Shape, c scene.Ctx) (piece, error) {
	spec := sh.Hole
	look := "ember"
	if spec != nil && spec.Look != "" {
		look = spec.Look
	}
	mk, ok := holeLooks[look]
	if !ok {
		return piece{}, fmt.Errorf("no black hole look %q; there is %s",
			look, names(holeLooks))
	}
	h := mk()
	if spec == nil {
		spec = &scene.Hole{}
	}
	sky, err := spaceFrom(spec.Space, h.Space, c)
	if err != nil {
		return piece{}, err
	}
	h.Space = sky
	if spec.Samples > 0 {
		h.Samples = spec.Samples
	}
	if len(spec.Disk) > 0 && !scene.RampMoves(spec.Disk) {
		h.Disk = light.Bake(scene.RampAt(spec.Disk, c))
	}
	update := func(c scene.Ctx) {
		if sh.At.Set() {
			h.At = vecAt(sh.At, c)
		}
		num(&h.Mass, spec.Mass, c)
		num(&h.Inner, spec.Inner, c)
		num(&h.Outer, spec.Outer, c)
		num(&h.Turn, spec.Turn, c)
		num(&h.Beaming, spec.Beaming, c)
		num(&h.Heat, spec.Heat, c)
		num(&h.Bright, spec.Bright, c)
		num(&h.Opacity, spec.Opacity, c)
		num(&h.Grain, spec.Grain, c)
		if spec.Seed != nil {
			h.Seed = uint64(spec.Seed.At(c))
		}
		if spec.Axis != nil {
			h.Axis = vecAt(*spec.Axis, c)
		}
		if scene.RampMoves(spec.Disk) {
			h.Disk = light.Bake(scene.RampAt(spec.Disk, c))
		}
	}
	update(c)
	return piece{order: 0, draw: h.Draw, update: update}, nil
}

// biomeNames are the biomes a document may colour, in their order.
var biomeNames = []string{"deep", "sea", "ice", "beach", "desert", "grass",
	"forest", "jungle", "tundra", "rock", "snow"}

// planetPiece is a planet.
func planetPiece(sh scene.Shape, at scene.Ctx) (piece, error) {
	spec := sh.World
	look := "terran"
	if spec != nil && spec.Look != "" {
		look = spec.Look
	}
	mk, ok := worldLooks[look]
	if !ok {
		return piece{}, fmt.Errorf("no planet look %q; there is %s",
			look, names(worldLooks))
	}
	p := mk()
	if spec == nil {
		spec = &scene.World{}
	}
	if err := colourBiomes(p, spec); err != nil {
		return piece{}, err
	}
	if sh.Radius > 0 {
		p.Radius = sh.Radius
	}
	update := func(c scene.Ctx) {
		if sh.At.Set() {
			p.At = vecAt(sh.At, c)
		}
		if spec.Tiles != nil {
			p.Tiles = int(spec.Tiles.At(c))
		}
		if spec.Plates != nil {
			p.Plates = int(spec.Plates.At(c))
		}
		if spec.Seed != nil {
			p.Seed = uint64(spec.Seed.At(c))
		}
		num(&p.Ocean, spec.Ocean, c)
		num(&p.Cold, spec.Cold, c)
		num(&p.Day, spec.Day, c)
		num(&p.Lines, spec.Lines, c)
		num(&p.Relief, spec.Relief, c)
		num(&p.Clouds, spec.Clouds, c)
		num(&p.AirDepth, spec.AirDepth, c)
		num(&p.Cities, spec.Cities, c)
		if spec.Axis != nil {
			p.Axis = vecAt(*spec.Axis, c)
		}
		if spec.Sun != nil {
			p.Sun = vecAt(*spec.Sun, c)
		}
	}
	update(at)
	return piece{order: 1, draw: p.Draw, update: update}, nil
}

// colourBiomes lays a world block's colours over a planet's.
func colourBiomes(p *scenery.Planet, spec *scene.World) error {
	index := map[string]int{}
	for i, n := range biomeNames {
		index[n] = i
	}
	for name, hex := range spec.Colours {
		i, ok := index[name]
		if !ok {
			return fmt.Errorf("no biome %q; there is %s", name,
				strings.Join(biomeNames, ", "))
		}
		p.Colours[i] = light.Hex(hex)
	}
	for name, g := range spec.Glow {
		i, ok := index[name]
		if !ok {
			return fmt.Errorf("no biome %q; there is %s", name,
				strings.Join(biomeNames, ", "))
		}
		p.Glow[i] = g
	}
	if spec.Air != "" {
		p.Air = light.Hex(spec.Air)
	}
	return nil
}

// emitterKinds are the emitters by the name a document gives them.
var emitterKinds = map[string]particles.Kind{
	"fountain": particles.Fountain, "jet": particles.Jet,
	"spiral": particles.Spiral, "drift": particles.Drift,
}

// emitterPiece is a particle emitter.
func emitterPiece(sh scene.Shape, at scene.Ctx) (piece, error) {
	spec := sh.Emit
	e := &particles.Emitter{Kind: emitterKinds[spec.Kind], Size: 0.01,
		Light: 1, Colour: light.HexRamp(0.0, "fff4dc", 1.0, "ff7a2a")}
	if len(spec.Ramp) > 0 && !scene.RampMoves(spec.Ramp) {
		e.Colour = light.Bake(scene.RampAt(spec.Ramp, at))
	}
	update := func(c scene.Ctx) {
		e.At = vecAt(sh.At, c)
		if spec.Axis != nil {
			e.Axis = vecAt(*spec.Axis, c)
		}
		if spec.Gravity != nil {
			e.Gravity = vecAt(*spec.Gravity, c)
		}
		if spec.Seed != nil {
			e.Seed = uint64(spec.Seed.At(c))
		}
		num(&e.Rate, spec.Rate, c)
		num(&e.Life, spec.Life, c)
		num(&e.Speed, spec.Speed, c)
		num(&e.Spread, spec.Spread, c)
		num(&e.Drag, spec.Drag, c)
		num(&e.Radius, spec.Radius, c)
		num(&e.Inner, spec.Inner, c)
		num(&e.Wobble, spec.Wobble, c)
		num(&e.Size, spec.Size, c)
		num(&e.Light, spec.Light, c)
		num(&e.Streak, spec.Streak, c)
		if scene.RampMoves(spec.Ramp) {
			e.Colour = light.Bake(scene.RampAt(spec.Ramp, c))
		}
	}
	update(at)
	return piece{order: 2, draw: e.Draw, update: update}, nil
}

// num sets a number from a document value, if the document gave one.
func num(dst *float64, v *scene.Value, c scene.Ctx) {
	if v != nil {
		*dst = v.At(c)
	}
}

// names is a map's keys, sorted, for an error a person reads.
func names[T any](m map[string]T) string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return strings.Join(out, ", ")
}
