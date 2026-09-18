package particles

import (
	"math"

	"github.com/janearc/miami/light"
	"github.com/janearc/miami/view"
)

// Drawing is adding light. A particle has no colour of its own to cover
// anything with: it is a small bright thing, and two of them in one place are
// twice as bright, which is what makes a jet read as dense at its root.
//
// A particle smaller than a subpixel still has to show. Its light is shared
// between the four subpixels around where it lands, by how close it is to
// each, so a spark moving slowly across the screen glides rather than stepping
// a subpixel at a time.
//
// Anything farther away than what the frame says is already at that subpixel
// is hidden. That is how a planet in front of a jet hides the jet, and how the
// shadow of a black hole swallows the dust that falls behind it.

// Draw adds every live particle to the frame, as the eye sees it at t.
func (e Emitter) Draw(b *light.Buffer, eye view.Eye, t float64) {
	e.Alive(t, func(p Particle) {
		c := e.Colour.At(p.Age).Scale(e.Light * envelope(p.Age))
		if e.Streak <= 0 {
			e.splat(b, eye, p.At, c)
			return
		}
		e.streak(b, eye, p, t, c)
	})
}

// envelope fades a particle in over the first tenth of its life and out over
// the last third, so nothing appears or vanishes in one frame.
func envelope(age float64) float64 {
	in := math.Min(1, age/0.1)
	out := math.Min(1, (1-age)/0.33)
	return math.Max(0, math.Min(in, out))
}

// streak draws the path a particle took over the last Streak seconds, its
// light shared along it, so a fast mote is a line the way it is on film.
func (e Emitter) streak(b *light.Buffer, eye view.Eye, p Particle,
	t float64, c light.RGB) {
	x0, y0, _, ok0 := eye.Project(p.At, b.W, b.H)
	from, ok := e.Where(p.ID, t-e.Streak)
	x1, y1, _, ok1 := eye.Project(from.At, b.W, b.H)
	if !ok || !ok0 || !ok1 {
		e.splat(b, eye, p.At, c)
		return
	}
	n := int(math.Ceil(math.Hypot(x1-x0, y1-y0)))
	// a mote that wrapped across its box between the two moments would
	// streak across the whole screen; it is drawn as a point instead
	if n > 48 {
		e.splat(b, eye, p.At, c)
		return
	}
	n = max(n, 1)
	share := c.Scale(1 / float64(n))
	for k := range n {
		q, ok := e.Where(p.ID, t-e.Streak*float64(k)/float64(n))
		if !ok {
			continue
		}
		// fading along its length, brightest where it is now
		fade := 1 - float64(k)/float64(n)
		e.splat(b, eye, q.At, share.Scale(2*fade))
	}
}

// splat adds one particle's light at a point in the world.
func (e Emitter) splat(b *light.Buffer, eye view.Eye, at view.Vec,
	c light.RGB) {
	x, y, depth, ok := eye.Project(at, b.W, b.H)
	if !ok {
		return
	}
	ix, iy := int(math.Round(x)), int(math.Round(y))
	if ix < 0 || iy < 0 || ix >= b.W || iy >= b.H {
		return
	}
	if depth > b.Depth(ix, iy) {
		return // something solid is nearer
	}
	lens := eye.Lens
	if lens == 0 {
		lens = 1
	}
	// radius in subpixels across; a subpixel is four thirds taller than
	// it is wide, so the same radius is fewer of them down
	r := e.Size * lens * float64(b.W) / depth
	if r <= 1 {
		// all of its light, in proportion to how much of a subpixel it
		// covers, shared between the four around it
		energy := c.Scale(math.Min(1, math.Max(r*r*math.Pi, 0.25)))
		bilinear(b, x, y, energy)
		return
	}
	energy := c.Scale(math.Pi * r * r)
	ry := r * 3 / 4
	var total float64
	x0, x1 := int(math.Floor(x-2*r)), int(x+2*r)
	y0, y1 := int(math.Floor(y-2*ry)), int(y+2*ry)
	weights := func(do func(px, py int, w float64)) {
		for py := y0; py <= y1; py++ {
			for px := x0; px <= x1; px++ {
				dx, dy := (float64(px)-x)/r, (float64(py)-y)/ry
				if d2 := dx*dx + dy*dy; d2 < 4 {
					do(px, py, math.Exp(-2*d2))
				}
			}
		}
	}
	weights(func(_, _ int, w float64) { total += w })
	if total == 0 {
		return
	}
	weights(func(px, py int, w float64) {
		b.Add(px, py, energy.Scale(w/total))
	})
}

// bilinear shares light between the four subpixels around a point.
func bilinear(b *light.Buffer, x, y float64, c light.RGB) {
	x0, y0 := math.Floor(x), math.Floor(y)
	fx, fy := x-x0, y-y0
	ix, iy := int(x0), int(y0)
	b.Add(ix, iy, c.Scale((1-fx)*(1-fy)))
	b.Add(ix+1, iy, c.Scale(fx*(1-fy)))
	b.Add(ix, iy+1, c.Scale((1-fx)*fy))
	b.Add(ix+1, iy+1, c.Scale(fx*fy))
}
