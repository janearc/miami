package scenery

import (
	"math"
	"sync"

	"github.com/janearc/miami/internal/split"
	"github.com/janearc/miami/light"
	"github.com/janearc/miami/noise"
	"github.com/janearc/miami/view"
)

// BlackHole is a black hole with a disk of hot gas round it, in front of a sky
// it bends.
//
// Two costs, kept apart. Where every subpixel's ray goes -- which way it
// leaves, where it crosses the disk, how blue the light from there arrives --
// depends on the camera and the hole and nothing else.
//
// That is worked out once and kept, and it is the expensive part: a few
// hundred steps of an equation per ray.
//
// What the gas looks like at each of those crossings depends on the time,
// because the gas is going round. That is worked out every frame, and it is
// cheap: some noise at a place already known.
//
// So a black hole that stays put costs a frame what a noisy texture costs. One
// whose camera moves costs a bake a frame, which is tens of milliseconds at
// full screen on every core, and is still a thing that can be done.
type BlackHole struct {
	At view.Vec // where the hole is
	// Mass is world units to one mass. The shadow is about 10.4 across.
	Mass float64

	// Axis is the disk's axis. A disk seen almost edge on, the look
	// everyone knows, has an axis nearly straight up and tipped a little
	// toward the camera.
	Axis view.Vec
	// Inner and Outer are where the disk starts and stops, in masses. Six
	// is the innermost orbit gas can keep round a hole that does not spin.
	Inner, Outer float64

	// Turn is how many seconds the inner edge takes to go round; negative
	// turns it the other way. Gas further out takes longer, as orbits do.
	Turn float64
	// Beaming is how much brighter and bluer the side coming toward the
	// camera is: one is what physics says, nought is none at all, and
	// films choose somewhere between.
	Beaming float64
	// Heat is where on the ramp the hottest gas sits before any shift, and
	// Bright how bright the disk is. Neither needs a new bake, so both can
	// move with the music.
	Heat, Bright float64
	// Disk is the gas's colour, from cold at nought to hottest at one.
	Disk light.Ramp
	// Opacity is how much a thick patch of gas hides what is behind it.
	Opacity float64
	// Grain is how fine the streaks are; one is the default.
	Grain float64
	Seed  uint64

	// Space is the sky behind, as the hole bends it. It is baked into the
	// rays, so a change to it is seen at the next bake: a new size, a new
	// camera, or Forget.
	Space Space

	// Samples is how many rays a side a subpixel on an edge gets: the rim
	// of the shadow and the thin lensed rings. Nought is three.
	Samples int

	mu    sync.Mutex
	baked *bake
}

// bake is everything about the rays that does not change with time.
type bake struct {
	key   bakeKey
	first []int32 // per subpixel, its first sample
	count []uint8 // per subpixel, how many samples
	s     []sample
	hits  []hit
	depth float64 // the hole's depth, for hiding what is behind the shadow
}

// bakeKey is what a bake depends on. A bake whose key still matches is kept.
type bakeKey struct {
	w, h               int
	eye                view.Eye
	at, axis           view.Vec
	mass, inner, outer float64
	beaming, spin      float64
	samples            int
}

// sample is one ray: what it ended up seeing past the disk, and the disk
// crossings in front of that.
type sample struct {
	caught bool
	sky    Sky
	hit0   int32
	nhit   uint8
}

// hit is one crossing of the disk, with what the heat and brightness there
// are apart from the gas's own texture: the temperature profile and the
// Doppler shift, both fixed by where the ray crossed.
type hit struct {
	r, phi     float32
	heat, glow float32
}

// maxHits is how many crossings a ray keeps. The fourth image of the disk is a
// ring far thinner than a subpixel.
const maxHits = 3

// Draw puts the hole, its disk and the bent sky into the frame.
//
// It replaces what is there, because everything it draws is either the hole or
// the sky behind it -- except where something nearer than the hole has already
// been drawn, which it leaves alone.
//
// Without that, a planet drawn first was erased by a hole forty units behind
// it, and only the order of the calls saved it.
func (h *BlackHole) Draw(b *light.Buffer, eye view.Eye, t float64) {
	bk := h.bakeFor(b.W, b.H, eye)
	split.Rows(b.H, func(lo, hi int) {
		for y := lo; y < hi; y++ {
			for x := 0; x < b.W; x++ {
				if b.Depth(x, y) < bk.depth {
					continue
				}
				c, caught := h.shade(bk, y*b.W+x, t)
				b.Set(x, y, c)
				if caught {
					b.SetDepth(x, y, bk.depth)
				}
			}
		}
	})
}

// Forget drops the rays kept from the last bake, so the next frame traces
// them again: for a sky changed while the hole was on screen.
func (h *BlackHole) Forget() {
	h.mu.Lock()
	h.baked = nil
	h.mu.Unlock()
}

// shade is the light for one subpixel at t, and whether any of its rays fell
// in.
func (h *BlackHole) shade(bk *bake, p int, t float64) (light.RGB, bool) {
	n := int(bk.count[p])
	if n == 0 {
		return light.RGB{}, false
	}
	var sum light.RGB
	caught := false
	for i := range n {
		s := bk.s[int(bk.first[p])+i]
		hits := bk.hits[s.hit0 : int(s.hit0)+int(s.nhit)]
		col, keep := h.through(hits, t)
		if s.caught {
			caught = true
		} else {
			col = col.Plus(s.sky.At(t).Scale(keep))
		}
		sum = sum.Plus(col)
	}
	return sum.Scale(1 / float64(n)), caught
}

// through is the light a ray picks up crossing the disk at its hits, nearest
// first, and how much of what is behind the disk still shows.
func (h *BlackHole) through(hits []hit, t float64) (light.RGB, float64) {
	var col light.RGB
	keep := 1.0
	for _, ht := range hits {
		dense := h.density(float64(ht.r), float64(ht.phi), t)
		if dense <= 0 {
			continue
		}
		alpha := 1 - math.Exp(-dense*h.opacity())
		heat := h.Heat * float64(ht.heat) * (0.85 + 0.25*dense)
		glow := h.Bright * float64(ht.glow) * (0.35 + 0.65*dense)
		col = col.Plus(h.Disk.At(heat).Scale(glow * alpha * keep))
		keep *= 1 - alpha
	}
	return col, keep
}

// opacity is Opacity, or a default for a disk that did not say.
func (h *BlackHole) opacity() float64 {
	if h.Opacity <= 0 {
		return 2.5
	}
	return h.Opacity
}

// edges is Inner and Outer, defaulted.
func (h *BlackHole) edges() (inner, outer float64) {
	inner, outer = h.Inner, h.Outer
	if inner <= 0 {
		inner = 6
	}
	if outer <= inner {
		outer = inner * 4
	}
	return inner, outer
}

// density is how much gas there is at a place on the disk at t, nought to one.
//
// The gas is streaks of noise that go round at the orbital speed of their own
// radius. Left to do that for ever, the inner streaks lap the outer ones until
// the disk is wound into rings too fine to draw.
//
// So the pattern is two copies, each carried round for a short while and then
// swapped for a fresh one while it is faded out: the old flow-map trick. At
// any moment it moves as an orbiting disk does, and it never winds up.
func (h *BlackHole) density(r, phi, t float64) float64 {
	inner, outer := h.edges()
	edge := smooth(inner, inner*1.12, r) *
		(1 - smooth(outer*0.55, outer, r))
	if edge <= 0 {
		return 0
	}
	turn := h.Turn
	if turn == 0 {
		turn = 6
	}
	omega := 2 * math.Pi / turn * math.Pow(inner/r, 1.5)
	span := math.Abs(turn) * 0.8
	var sum, weights float64
	for layer := range 2 {
		tau := t + float64(layer)*span/2
		cycle := math.Floor(tau / span)
		local := tau - cycle*span
		w := 1 - math.Abs(2*local/span-1)
		seed := h.Seed + uint64(int64(cycle))*0x9e3779b97f4a7c15 +
			uint64(layer)
		sum += w * h.streaks(seed, r, phi-omega*local)
		weights += w * w
	}
	// two uncorrelated noises halfway through a crossfade are flatter
	// than either, so the blend is scaled back up by its spread
	n := sum / math.Sqrt(math.Max(weights, 1e-6))
	return edge * math.Pow(clamp01(0.45+1.15*n), 1.6)
}

// streaks is one copy of the gas's texture: broad clumps, and filaments three
// times finer across the radius than along the orbit. Gas sheared out by its
// own orbit is long threads, not blobs.
func (h *BlackHole) streaks(seed uint64, r, ang float64) float64 {
	grain := h.Grain
	if grain <= 0 {
		grain = 1
	}
	lr := math.Log(r) * 5 * grain
	c, s := math.Cos(ang)*grain, math.Sin(ang)*grain
	broad := noise.Fractal(seed, lr, c*1.2, s*1.2, 3)
	fine := noise.Fractal(seed+5, lr*3.2, c*2.2, s*2.2, 2)
	return 0.6*broad + 0.55*fine
}

// bakeFor is the bake for a frame of this size seen from this eye, made if
// the one kept does not match.
func (h *BlackHole) bakeFor(w, ht int, eye view.Eye) *bake {
	h.mu.Lock()
	defer h.mu.Unlock()
	inner, outer := h.edges()
	key := bakeKey{w: w, h: ht, eye: eye, at: h.At, axis: h.Axis,
		mass: h.Mass, inner: inner, outer: outer, beaming: h.Beaming,
		spin: math.Copysign(1, h.Turn), samples: h.Samples}
	if h.baked != nil && h.baked.key == key {
		return h.baked
	}
	h.baked = newBaker(h, key).bake()
	return h.baked
}

// baker traces one bake.
type baker struct {
	h     *BlackHole
	key   bakeKey
	cam   view.Vec // the camera, in masses from the hole
	pixel float64  // radians a subpixel spans
	per   int      // rays a side on an edge
	rows  []traceRow
}

// traceRow is what one row of the bake found.
type traceRow struct {
	s     []sample
	hits  []hit
	first []int32
	count []uint8
}

// newBaker works out what every ray of a bake shares.
func newBaker(h *BlackHole, key bakeKey) *baker {
	mass := key.mass
	if mass <= 0 {
		mass = 1
	}
	lens := key.eye.Lens
	if lens == 0 {
		lens = 1
	}
	per := key.samples
	if per <= 0 {
		per = 3
	}
	return &baker{h: h, key: key, per: per,
		cam:   scale(sub(key.eye.At, h.At), 1/mass),
		pixel: 1 / (lens * float64(key.w))}
}

// bake traces every subpixel's rays.
//
// One ray through the middle of each subpixel first. Then a subpixel whose ray
// ended differently from a neighbour's -- one fell in and the other did not,
// or they crossed the disk a different number of times -- is on an edge.
//
// An edge is traced again with a grid of rays across it. The rim of the shadow
// and the lensed rings are those edges, and they are a small part of the frame.
func (bk *baker) bake() *bake {
	w, ht := bk.key.w, bk.key.h
	bk.rows = make([]traceRow, ht)
	split.Rows(ht, func(lo, hi int) {
		var tr traced
		for y := lo; y < hi; y++ {
			for x := 0; x < w; x++ {
				fx, fy := float64(x)+0.5, float64(y)+0.5
				bk.one(fx, fy, &tr, &bk.rows[y])
			}
		}
	})
	final := make([]traceRow, ht)
	split.Rows(ht, func(lo, hi int) {
		var tr traced
		for y := lo; y < hi; y++ {
			bk.again(y, &tr, &final[y])
		}
	})
	return bk.flatten(final)
}

// one traces a ray through a point of the sheet and files what it met.
func (bk *baker) one(fx, fy float64, tr *traced, row *traceRow) {
	key := bk.key
	trace(bk.cam, key.eye.Through(fx, fy, key.w, key.h), key.axis, tr)
	s := sample{caught: tr.fate == captured, hit0: int32(len(row.hits))}
	for _, c := range tr.cross {
		if c.r < key.inner || c.r > key.outer {
			continue
		}
		if s.nhit == maxHits {
			break
		}
		heat, glow := emission(c.r, tr.lambda, tr.at, key)
		row.hits = append(row.hits, hit{r: float32(c.r),
			phi: float32(c.phi), heat: float32(heat),
			glow: float32(glow)})
		s.nhit++
	}
	if !s.caught {
		s.sky = bk.h.Space.Look(tr.away, bk.pixel)
	}
	row.s = append(row.s, s)
}

// again is one row as it is kept: each subpixel's middle ray copied across,
// or a grid of new rays where the subpixel is on an edge.
func (bk *baker) again(y int, tr *traced, out *traceRow) {
	w, per := bk.key.w, bk.per
	out.first = make([]int32, w)
	out.count = make([]uint8, w)
	for x := 0; x < w; x++ {
		out.first[x] = int32(len(out.s))
		if !bk.edge(x, y) {
			s := bk.rows[y].s[x]
			hs := bk.rows[y].hits[s.hit0 : int(s.hit0)+int(s.nhit)]
			s.hit0 = int32(len(out.hits))
			out.hits = append(out.hits, hs...)
			out.s = append(out.s, s)
			out.count[x] = 1
			continue
		}
		step := 1 / float64(per)
		for j := range per {
			for i := range per {
				fx := float64(x) + (float64(i)+0.5)*step
				fy := float64(y) + (float64(j)+0.5)*step
				bk.one(fx, fy, tr, out)
			}
		}
		out.count[x] = uint8(per * per)
	}
}

// edge is whether a subpixel's middle ray ended differently from one of its
// four neighbours'.
func (bk *baker) edge(x, y int) bool {
	s := bk.rows[y].s[x]
	w, h := bk.key.w, bk.key.h
	for _, n := range [][2]int{{x + 1, y}, {x - 1, y}, {x, y + 1},
		{x, y - 1}} {
		if n[0] < 0 || n[1] < 0 || n[0] >= w || n[1] >= h {
			continue
		}
		o := bk.rows[n[1]].s[n[0]]
		if o.caught != s.caught || o.nhit != s.nhit {
			return true
		}
	}
	return false
}

// flatten joins the rows into one bake, moving each row's offsets to where
// its samples landed.
func (bk *baker) flatten(rows []traceRow) *bake {
	w, ht := bk.key.w, bk.key.h
	out := &bake{key: bk.key, first: make([]int32, w*ht),
		count: make([]uint8, w*ht)}
	for y := range rows {
		row := &rows[y]
		sBase, hBase := int32(len(out.s)), int32(len(out.hits))
		for i := range row.s {
			row.s[i].hit0 += hBase
		}
		out.s = append(out.s, row.s...)
		out.hits = append(out.hits, row.hits...)
		for x := 0; x < w; x++ {
			out.first[y*w+x] = row.first[x] + sBase
			out.count[y*w+x] = row.count[x]
		}
	}
	if _, _, d, ok := bk.key.eye.Project(bk.h.At, w, ht); ok {
		out.depth = d
	}
	return out
}

// emission is what a crossing contributes before the gas's texture: where on
// the ramp its heat falls and how brightly it glows.
//
// The temperature of a thin disk rises from nothing at its inner edge to a
// peak just outside it, and falls away as r to the -3/4 further out.
//
// The light that reaches the camera is that, shifted by g -- hotter by g, and
// brighter by g to the fourth -- where g is the redshift, and Beaming is how
// much of g to believe.
//
// Brightness goes as temperature squared, not the fourth power a black body's
// would. The fourth power is right, and it leaves everything outside the
// brightest ring black on a screen with 256 levels.
func emission(r, lambda, at float64, key bakeKey) (heat, glow float64) {
	inner := key.inner
	profile := func(r float64) float64 {
		return math.Pow(r/inner, -0.75) *
			math.Pow(math.Max(0, 1-math.Sqrt(inner/r)), 0.25)
	}
	temp := profile(r) / profile(inner*49/36)
	g := math.Pow(redshift(r, lambda, at, key.spin), key.beaming)
	return g * temp, math.Pow(g, 4) * temp * temp
}

// clamp01 holds a number to nought and one.
func clamp01(v float64) float64 { return math.Max(0, math.Min(1, v)) }
