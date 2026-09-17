package scenes

import (
	"fmt"
	"math"
	"strings"

	sprite "github.com/janearc/libdaffy-sprite"

	"github.com/janearc/miami/effect"
	"github.com/janearc/miami/motion"
)

// The animations, as library calls.
//
// What is per-subject is never the effect -- it is the rig. Where the shoulder
// is, which columns are the near leg, which way the fluke points.
//
// The operators in motion and effect name roles and extents and would run on
// any of these stencils; everything specific to one bird lives in its function
// here, and nowhere else.
//
// Each subject gets the one motion its photograph will support. A head study
// blinks because there is nothing else in frame to move; a bird caught with its
// wings spread can fly; a ball sitting off on its own can be caught. None of
// them pretend to a motion the stencil cannot carry.

type run struct {
	name, note string
	frames     []sprite.Frame
}

// note is the header a stream carries, and its lines are that file's lines --
// so they are written here at the margin, where they are the width they will
// be read at. The opening backtick has to sit on a line of its own to make
// that possible, and the newline it leaves behind comes off here.
func note(s string) string { return strings.TrimPrefix(s, "\n") }

// Animations builds every stream. Returning them rather than writing them keeps
// the command in charge of where they land.
func Animations(made string) ([]run, error) {
	var out []run
	for _, fn := range []func(string) (run, error){
		puffinBlink, kingfisherHunt, starlingFlight,
		daffyWalk, gameCatch, dolphinsSwim,
	} {
		r, err := fn(made)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}

// blinkCadence is uneven on purpose: a metronome reads as a fault in the
// terminal, a bird does not. A bird holds its eye open for seconds, shuts it in
// under a fifth of one, and often blinks twice.
var blinkCadence = []struct {
	shut float64
	ms   int
}{
	{0, 2300}, {0.5, 55}, {1, 70}, {0.5, 55},
	{0, 1100}, {0.5, 55}, {1, 70}, {0.5, 55}, {0, 1700},
}

// puffinBlink is the blink, and nothing else: a head study has no limb in
// frame.
func puffinBlink(made string) (run, error) {
	base, err := motion.ReadSprite(made + "puffin.sprite")
	if err != nil {
		return run{}, err
	}
	var fs []sprite.Frame
	for _, c := range blinkCadence {
		fs = append(
			fs,
			motion.Hold(motion.Blink(base, c.shut, "", ""), c.ms),
		)
	}
	return run{"puffin-blink", note(puffinNote), fs}, nil
}

const puffinNote = `
# a puffin blinks, and does nothing else. it is a head study; there is
# no limb in frame to swing. the eye is found by role -- pupil and eye-ring --
# so this cadence drives any stencil carrying an eye. the holds are uneven on
# purpose: a metronome reads as a fault in the terminal, a bird does not.`

// kingfisherHunt is a bob and a blink, the bob on whole pixels because the
// crown is a dither and a resample turns it to mud.
func kingfisherHunt(made string) (run, error) {
	// room below only: the crown is cropped by the top of the photograph,
	// so headroom above it would leave the bird floating in its own frame.
	//
	// the bob is a whole-pixel translation and it has to be. this crown is
	// a dense dither of gold on slate, and rotating it even five degrees
	// runs it through the resampler and turns the speckle to mud -- next to
	// an unrotated frame the texture visibly pops.
	src, err := motion.ReadSprite(made + "kingfisher.sprite")
	if err != nil {
		return run{}, err
	}
	base := motion.Pad(src, 0, 3, 0, 0)
	bob := func(dy int, shut float64) sprite.Frame {
		g := base
		if dy != 0 {
			g = motion.Warp(base,
				motion.Translate(0, float64(dy)),
				motion.All, motion.WarpOpts{Cover: 0.3})
		}
		return motion.Blink(g, shut, "", "")
	}
	steps := []struct {
		ms   int
		dy   int
		shut float64
	}{
		{1500, 0, 0}, {60, 0, 0.6}, {70, 0, 1}, {60, 0, 0.6},
		{700, 0, 0},
		{110, 1, 0}, {150, 2, 0}, {110, 1, 0}, {450, 0, 0},
		{100, 1, 0}, {130, 3, 0}, {300, 3, 0}, {130, 1, 0},
		{1200, 0, 0},
	}
	var fs []sprite.Frame
	for _, s := range steps {
		fs = append(fs, motion.Hold(bob(s.dy, s.shut), s.ms))
	}
	return run{"kingfisher-hunt", note(kingfisherNote), fs}, nil
}

const kingfisherNote = `
# a kingfisher ranging from a perch: a bob, a blink, and a longer bob.
# the bob is a whole-pixel translation on purpose. the crown is a dense dither
# and any warp that is not an exact integer shift runs it through the
# resampler and turns the speckle to mud.`

// starlingFlight is the wingbeat, articulated from the traced photograph.
func starlingFlight(made string) (run, error) {
	// the photograph caught it head-on with both wings fully spread, which
	// is the one pose a wingbeat can be built from. three things make it
	// read:
	//
	// the wings mirror rather than scissor; the span foreshortens at the
	// top and bottom of the stroke, because a wing at the extremes points
	// partly at the viewer; and the tip turns further than the shoulder.
	src, err := motion.ReadSprite(made + "starling.sprite")
	if err != nil {
		return run{}, err
	}
	const padTop = 10
	base := motion.Pad(src, padTop, 2, 0, 0)
	const (
		bodyL, bodyR = 27, 38
		wingBottom   = 21 + padTop
		span         = 28.0
	)
	lpx, lpy := 28.0, 12.0+padTop
	rpx, rpy := 37.0, 12.0+padTop
	lw := motion.And(motion.Left(bodyL), motion.Above(wingBottom))
	rw := motion.And(motion.Right(bodyR), motion.Above(wingBottom))

	pose := func(deg float64, dy int) sprite.Frame {
		sx := 1 - 0.30*(math.Abs(deg)/42)
		opt := motion.WarpOpts{Cover: 0.25}
		g := motion.Compose(base,
			motion.Layer{Frame: motion.Warp(base,
				motion.Bend(lpx, lpy, deg, span, 1.6, sx),
				lw, opt), Owns: lw},
			motion.Layer{Frame: motion.Warp(base,
				motion.Bend(rpx, rpy, -deg, span, 1.6, sx),
				rw, opt), Owns: rw})
		if dy != 0 {
			// the whole bird rises, not just its torso: translating
			// only the body columns opens a seam where they meet
			// the wings
			g = motion.Warp(
				g, motion.Translate(0, float64(dy)), motion.All,
				opt,
			)
		}
		return g
	}
	// the body runs against the wings: lift is made on the downstroke, so
	// the bird sits lowest with its wings raised. the downstroke holds
	// longest.
	beat := []struct {
		deg float64
		dy  int
		ms  int
	}{
		{40, 2, 80}, {26, 2, 55}, {6, 1, 50}, {-18, -1, 60},
		{-36, -2, 70}, {-42, -2, 95}, {-24, -1, 60}, {0, 0, 50}, {
			24, 1, 55,
		},
	}
	var fs []sprite.Frame
	for _, b := range beat {
		fs = append(fs, motion.Hold(pose(b.deg, b.dy), b.ms))
	}
	return run{"starling-flight-full", note(starlingNote), fs}, nil
}

const starlingNote = `
# a starling's wingbeat, articulated from the traced photograph. the wings
# mirror, the span foreshortens at the extremes of the stroke, and the tip
# turns further than the shoulder. the downstroke is the power stroke and
# holds longest; the recovery is quick and unloaded.`

// daffyWalk is a waddle, read entirely from the ankles down.
func daffyWalk(made string) (run, error) {
	// two feet and no bird above them, so the motion has to be a walk read
	// entirely from the ankles down. no padding above: the legs are cropped
	// by the top of the frame in the photograph, so a lifted leg must slide
	// off that edge.
	//
	// the feet are one connected mass -- their inner toes touch at a single
	// pixel -- so the seam goes through that pixel and nowhere else. cut a
	// column either side of it and the walk shears a webbed toe in half.
	src, err := motion.ReadSprite(made + "daffy.sprite")
	if err != nil {
		return run{}, err
	}
	base := motion.Pad(src, 0, 0, 2, 2)
	const split = 27
	lf, rf := motion.Left(split), motion.Right(split)
	opt := motion.WarpOpts{Cover: 0.3}
	step := func(ldy, ldx, rdy, rdx, sway int) sprite.Frame {
		g := motion.Compose(base,
			motion.Layer{Frame: motion.Warp(base,
				motion.Translate(float64(ldx), float64(ldy)),
				lf, opt), Owns: lf},
			motion.Layer{Frame: motion.Warp(base,
				motion.Translate(float64(rdx), float64(rdy)),
				rf, opt), Owns: rf})
		if sway != 0 {
			g = motion.Warp(
				g, motion.Translate(float64(sway), 0),
				motion.All, opt,
			)
		}
		return g
	}
	steps := [][5]int{
		{0, 0, 0, 0, 0}, {-2, 1, 0, 0, 1},
		{-3, 2, 0, 0, 1}, {-1, 1, 0, 0, 0},
		{0, 0, 0, 0, 0}, {0, 0, -2, -1, -1},
		{0, 0, -3, -2, -1}, {0, 0, -1, -1, 0},
	}
	ms := []int{130, 110, 150, 110, 130, 110, 150, 110}
	var fs []sprite.Frame
	for i, s := range steps {
		fs = append(
			fs,
			motion.Hold(step(s[0], s[1], s[2], s[3], s[4]), ms[i]),
		)
	}
	return run{"daffy-walk", note(daffyNote), fs}, nil
}

const daffyNote = `
# a duck's walk, read from the ankles down. the rock onto the standing leg
# comes before the lift, which is what makes a waddle a waddle; the lifted
# foot travels up and a little forward, then slaps down flat.`

// gameCatch is the ball flying into the glove: the one loose component.
func gameCatch(made string) (run, error) {
	// the ball is the only thing in the frame not attached to the fielder,
	// so the grid hands it over as its own component and the picture turns
	// into the thing it was always about: a catch. the traced position is
	// one frame of the flight, not the start of it.
	src, err := motion.ReadSprite(made + "game.sprite")
	if err != nil {
		return run{}, err
	}
	padded := motion.Pad(src, 0, 0, 0, 8)
	comps := motion.Components(padded)
	if len(comps) < 2 {
		return run{}, fmt.Errorf(
			"game: expected a loose ball, found %d components",
			len(comps),
		)
	}
	base, ball := motion.Lift(padded, comps[1])
	glove := motion.Box(38, 20, 57, 35)

	path := [][3]int{{16, -11, 70}, {11, -8, 60}, {7, -5, 60}, {3, -2, 55},
		{0, 0, 55}, {-3, 3, 55}, {-5, 6, 50}}
	var fs []sprite.Frame
	for _, p := range path {
		fs = append(
			fs,
			motion.Hold(motion.Stamp(base, ball, p[0], p[1]), p[2]),
		)
	}
	// taken: the ball is inside the glove, and the glove gives with it
	hit := motion.Compose(base, motion.Layer{
		Frame: motion.Warp(
			base, motion.Translate(1, 1), glove,
			motion.WarpOpts{Cover: 0.3},
		),
		Owns: glove})
	fs = append(
		fs, motion.Hold(hit, 90), motion.Hold(hit, 120),
		motion.Hold(base, 900),
	)
	return run{"game-catch", note(gameNote), fs}, nil
}

const gameNote = `
# a catch. the cricket ball is the one component in the stencil that is not
# touching the fielder, so it lifts out and flies: in along the line of the
# arm, into the glove, and gone. the glove gives a pixel on contact, which is
# the whole of the impact and is enough.`

// dolphinsSwim is a wave travelling down each body.
func dolphinsSwim(made string) (run, error) {
	// a dolphin does not flap its tail, it passes a wave down its body. the
	// amplitude grows from a head that barely moves to a fluke that moves a
	// lot, and because the two animals sit at different points along the
	// same wave they come out of step with each other for free.
	src, err := motion.ReadSprite(made + "dolphins.sprite")
	if err != nil {
		return run{}, err
	}
	base := motion.Pad(src, 3, 3, 0, 0)
	w := float64(base.Width())
	amp := func(x float64) float64 { return 3.6 * math.Pow(x/w, 2.2) }
	const n = 8
	var fs []sprite.Frame
	for k := 0; k < n; k++ {
		ph := 2 * 3.14159265 * float64(k) / n
		fs = append(fs, motion.Hold(motion.Warp(base,
			motion.Undulate(
				amp, 54, ph,
			), motion.All, motion.WarpOpts{Cover: 0.3}), 115))
	}
	return run{"dolphins-swim", note(dolphinNote), fs}, nil
}

const dolphinNote = `
# two dolphins swimming. the motion is a wave travelling down each body,
# not a tail being flapped: the amplitude is near nothing at the rostrum and
# grows toward the fluke. the pair sit at different points along the same wave,
# so they fall out of step with each other without being told to.`

// Lit builds the light work: two still objects, two effects, and every pairing.
// The bulb stencil is emitted cold from miami's own geometry and never edited
// again; the word is set once by the typesetter and never edited again; the
// halo and the sweep are written once each. Lighting the bulb costs a line.
//
// Lighting the word costs the same line.
func Lit(made, css string) ([]run, error) {
	L, err := effect.Lightness(css)
	if err != nil {
		return nil, err
	}
	bulbSrc, err := motion.ReadSprite(made + "bulb.sprite")
	if err != nil {
		return nil, err
	}
	wordSrc, err := motion.ReadSprite(made + "word.sprite")
	if err != nil {
		return nil, err
	}
	bulb := motion.Pad(bulbSrc, 16, 6, 18, 18)
	word := motion.Pad(wordSrc, 3, 3, 3, 3)

	gx0, gy0, gx1, gy1 := bbox(
		bulb, 'V',
	) // the glass; the filament is inside it
	halo := func(gain float64) sprite.Frame {
		if gain <= 0 {
			return blank(bulb)
		}
		return effect.Glow(bulb, effect.GlowOpts{
			Source: "D", Within: motion.Box(
				gx0, gy0, gx1, gy1,
			), Band: 3,
			Radius: 9, Gain: gain, Rays: 12, RayGain: 0.55 * gain,
			RayReach: 0.30, RayInner: 1.25, RayTurn: 0.35})
	}
	// a bulb on an old sign does not come on, it catches. two failed
	// strikes, then the filament takes and floods past where it settles,
	// then it sits and breathes on the mains.
	cadence := []struct {
		gain float64
		ms   int
	}{
		{0, 900}, {0.55, 60}, {0, 140}, {0.75, 45}, {0, 320},
		{0.45, 70}, {0.80, 90}, {1.00, 140}, {0.88, 260},
		{0.94, 700}, {0.88, 900}, {0.94, 800}, {0.90, 1100},
	}
	var bulbFx, bulbLit []sprite.Frame
	for _, c := range cadence {
		a := motion.Hold(halo(c.gain), c.ms)
		bulbFx = append(bulbFx, a)
		lit := effect.Composite(bulb, a, L, 1, 0, 0)
		bulbLit = append(bulbLit, motion.Hold(lit, c.ms))
	}

	ww, wh := word.Width(), word.Height()
	var wordFx, wordLit []sprite.Frame
	for k := 0; k <= 18; k++ {
		at := -0.25 + 1.5*float64(k)/17
		ms := 55
		if k == 18 {
			// dark, and a pause before it runs again
			at, ms = 2.0, 900
		}
		// clipped to the letters: a sweep is a highlight on a surface,
		// and the night behind the sign has no surface to catch it
		sweep := effect.Sweep(ww, wh, -62, 0.16, 0.30, at, 1.15)
		a := motion.Hold(effect.ClipTo(sweep, word, "K", 0), ms)
		wordFx = append(wordFx, a)
		wordLit = append(
			wordLit,
			motion.Hold(effect.Composite(word, a, L, 1, 0, 0), ms),
		)
	}

	// the pairing, swapped over: each effect run over the other object. no
	// effect was edited to make this work and neither stencil was touched.
	bw, bh := bulb.Width(), bulb.Height()
	var bulbSweep []sprite.Frame
	for k := 0; k < 14; k++ {
		a := effect.ClipTo(effect.Sweep(bw, bh, -62, 0.18, 0.28,
			-0.2+1.4*float64(k)/13, 1.15), bulb, "VDKW", 0)
		bulbSweep = append(
			bulbSweep,
			motion.Hold(effect.Composite(bulb, a, L, 1, 0, 0), 60),
		)
	}
	wordBig := motion.Pad(wordSrc, 8, 8, 8, 8)
	var wordGlow []sprite.Frame
	// the word has no filament, so the halo is told to come out of the
	// letters themselves: same effect, a different role named at the call
	// site. the one knob that has to move is extent -- at the bulb's nine
	// pixels the glow fills the counters and the word becomes a lozenge.
	for _, g := range []float64{0.30, 0.55, 0.80, 1.00, 0.80, 0.55} {
		a := effect.Glow(
			wordBig,
			effect.GlowOpts{Source: "K", Radius: 2.6, Gain: g},
		)
		lit := effect.Composite(wordBig, a, L, 1, 0, 0)
		wordGlow = append(wordGlow, motion.Hold(lit, 150))
	}

	const fxNote = "\n#\n# this is the effect on its own:" +
		" an alpha channel drawn as a sprite."
	const sweepNote = "# the word's sweep, run over the bulb." +
		" nothing was edited."
	const glowNote = "# the bulb's halo, run over the word. the light" +
		" is told to come\n" +
		"# out of a different role and nothing else changes."

	return []run{
		{"bulb-lit-fx", note(bulbNote) + fxNote, bulbFx},
		{"bulb-lit", note(bulbNote), bulbLit},
		{"word-sweep-fx", note(wordNote) + fxNote, wordFx},
		{"word-sweep", note(wordNote), wordLit},
		{"bulb-sweep", sweepNote, bulbSweep},
		{"word-glow", glowNote, wordGlow},
	}, nil
}

const bulbNote = `
# a bulb that is not on, lit by a second sprite.
# the stencil never changes -- not one character, in any frame. all of the
# light is an alpha layer composited over it in oklab lightness, and the
# filament is found by role, so the halo comes out of the coil rather than
# out of a circle somebody centred by hand.`

const wordNote = `
# a sweep of light across set type, using the same compositor that lights
# the bulb. the effect names roles and a normalised extent and never a pixel,
# so nothing in it knows it is looking at letters.`

// bbox is the extent of a role in a frame.
func bbox(f sprite.Frame, role byte) (x0, y0, x1, y1 int) {
	x0, y0, x1, y1 = f.Width(), f.Height(), -1, -1
	for y, r := range f.Art {
		for x := 0; x < len(r); x++ {
			if r[x] == role {
				x0, y0 = min(x0, x), min(y0, y)
				x1, y1 = max(x1, x), max(y1, y)
			}
		}
	}
	return x0, y0, x1 + 1, y1 + 1
}

// blank is a frame of the same size with nothing in it.
func blank(f sprite.Frame) sprite.Frame {
	out := make([]string, f.Height())
	for i := range out {
		out[i] = fmt.Sprintf("%*s", f.Width(), "")
	}
	for i := range out {
		b := make([]byte, f.Width())
		for j := range b {
			b[j] = '.'
		}
		out[i] = string(b)
	}
	return sprite.Frame{Art: out}
}

// WriteAll writes a set of runs into a directory.
func WriteAll(dir string, runs []run) ([]string, error) {
	var out []string
	for _, r := range runs {
		if err := motion.WriteStreamFile(
			dir+r.name+".stream", r.name, r.note, r.frames,
		); err != nil {
			return nil, err
		}
		out = append(out, fmt.Sprintf("%-22s %2d frames -> %s%s.stream",
			r.name, len(r.frames), dir, r.name))
	}
	return out, nil
}
