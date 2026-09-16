// Package effect lights a sprite with another sprite.
//
// An effect here is not a filter and not a mode. It is a second sprite,
// carrying no picture at all -- only how much light is at each pixel -- and a
// compositor that mixes the two.
//
// The still thing is authored once and never edited again; the light is
// authored once; and either can be replaced without touching the other.
//
// The effect sprite is a real sprite. It is in the eleven characters, it plays
// on its own through the player, and daffy can read it. What makes it an alpha
// channel is only how this package reads it: five roles, in a fixed ladder,
// standing for five levels of coverage.
//
//	.  0.00    nothing
//	D  0.25    the far halo
//	K  0.50
//	V  0.75
//	W  1.00    the source itself
//
// Five and not a continuum on purpose. Banding is the look -- a roadside sign
// at night has hard rings around every bulb -- and quantising bounds what the
// compositor can produce, so a lit frame is still a stencil in eleven
// characters and still takes any colourway written later.
//
// The mixing is in oklab lightness, never in the role characters. A pixel's lit
// lightness is its own lifted toward white by the alpha, and the output role is
// whichever role the colourway binds nearest to that. A role is never made
// darker than it started, because light does not subtract.
//
// Fields name roles, never pixels. Glow is told which role the light comes out
// of; on a bulb that is the filament, on a wordmark it is the letters, and the
// field is a distance transform out of whatever it was handed. One number
// genuinely has to move between objects, and it is extent:
//
// a bulb's halo is nine pixels because the globe is, a letter's is two and a
// half because its counters are. At nine the glow fills the counters and the
// word becomes a lozenge.
package effect

import (
	"fmt"
	"math"
	"os"
	"strings"

	sprite "github.com/janearc/libdaffy-sprite"
	"github.com/janearc/libtheme-css/css"
	"github.com/janearc/libtheme-css/spaces/ok"

	"github.com/janearc/miami/motion"
)

// Ladder is the alpha ramp, dimmest first. One place; change it here and every
// effect re-reads.
const Ladder = ".DKVW"

// Lit is what light can turn a pixel into, dimmest first. The warm roles are in
// and wing is out: the near field of a tungsten bulb is amber, so a pixel with
// real light on it should go gold rather than lilac. This is the one place that
// knows the light is warm.
const Lit = ".DKYBRX"

// roles maps a colourway's custom-property names onto the alphabet. It is the
// mirror of the role table the sprite emitter writes, and it lives here because
// a colourway is written in words and a sprite in characters.
var roles = map[string]byte{
	"none": '.', "dark": 'K', "light": 'W', "wing": 'V', "stripe": 'D',
	"beak-base": 'B', "beak-band": 'Y', "beak-tip": 'R', "feet": 'O',
	"pupil": 'E', "eye-ring": 'X',
}

// Lightness is each role's oklab lightness under a colourway, read with
// libtheme's own colourway reader rather than by looking for hashes.
func Lightness(path string) (map[byte]float64, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cw := css.Read(string(src))
	out := map[byte]float64{}
	for name, r := range roles {
		if sw, ok2 := cw.Roles.Get(name); ok2 {
			out[r] = ok.FromSwatch(sw).L
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%s: no roles of the alphabet found", path)
	}
	return out, nil
}

// Alpha is the coverage a ladder character stands for.
func Alpha(c byte) float64 {
	i := strings.IndexByte(Ladder, c)
	if i < 0 {
		return 0
	}
	return float64(i) / float64(len(Ladder)-1)
}

// Quantise is a scalar as a ladder character.
func Quantise(v float64) byte {
	i := int(math.Round(math.Max(0, math.Min(1, v)) * float64(len(Ladder)-1)))
	return Ladder[i]
}

// Composite lights a base frame through an alpha frame. The result is a
// stencil, not a picture: it goes back out as roles and is dressed by whatever
// colourway plays it.
func Composite(base, glow sprite.Frame, L map[byte]float64, strength float64, ox, oy int) sprite.Frame {
	if strength == 0 {
		strength = 1
	}
	type rung struct {
		l float64
		c byte
	}
	var ladder []rung
	for i := 0; i < len(Lit); i++ {
		if l, ok2 := L[Lit[i]]; ok2 {
			ladder = append(ladder, rung{l, Lit[i]})
		}
	}
	ground := L['.']
	out := make([]string, base.Height())
	for y := 0; y < base.Height(); y++ {
		row := make([]byte, base.Width())
		for x := 0; x < base.Width(); x++ {
			ch := base.Art[y][x]
			a := Alpha(motion.At(glow, x-ox, y-oy)) * strength
			if a <= 0 {
				row[x] = ch
				continue
			}
			lb, ok2 := L[ch]
			if !ok2 {
				lb = ground
			}
			// quantise the lit lightness to the nearest rung, then
			// accept it only if it is an increase.
			//
			// picking the nearest rung that is brighter
			// instead forces a change wherever there is any
			// alpha at all.
			//
			// so the faintest halo lifts every pixel it
			// touches a whole step, and the effect reads at
			// full strength from its first frame.
			target := lb + a*(1-lb) // light adds; it never subtracts
			best, bestD := byte(0), math.Inf(1)
			for _, r := range ladder {
				if d := math.Abs(r.l - target); d < bestD {
					best, bestD = r.c, d
				}
			}
			if bestD < math.Inf(1) && L[best] > lb {
				row[x] = best
			} else {
				row[x] = ch
			}
		}
		out[y] = string(row)
	}
	return sprite.Frame{Art: out, Hold: base.Hold}
}

// GlowOpts shapes a halo.
type GlowOpts struct {
	Source   string // roles the light comes out of; required
	Within   motion.Mask
	Band     int     // keep only rows within this many of the densest one; 0 keeps all
	Radius   float64 // extent, in pixels: the one genuine per-object knob
	Gain     float64
	Rays     int
	RayGain  float64
	RayReach float64
	RayInner float64 // rays start this many radii out, so they do not spine the object
	RayTurn  float64
}

// Glow is the field a light source makes, as an alpha sprite.
//
// The light comes from wherever a named role is, not from a point. For a bulb
// that is the filament, and the field is a distance transform out of its actual
// zigzag -- so the hottest pixels are the wire itself and the glass around it
// falls away, instead of a circle washing the whole globe to one flat role.
func Glow(f sprite.Frame, o GlowOpts) sprite.Frame {
	h, w := f.Height(), f.Width()
	var src []motion.Pt
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if strings.IndexByte(o.Source, f.Art[y][x]) >= 0 &&
				(o.Within == nil || o.Within(x, y)) {
				src = append(src, motion.Pt{X: x, Y: y})
			}
		}
	}
	if o.Band > 0 && len(src) > 0 {
		// keep only the rows around the densest one. a filament's lead
		// wires carry the same role as its coil, and a distance
		// transform that includes them makes a glowing rectangle
		// instead of a glowing coil.
		//
		// the coil is the densest band of the role; that is derivable,
		// so it is derived rather than written down as a row number.
		count := map[int]int{}
		peak, peakN := 0, 0
		for _, p := range src {
			count[p.Y]++
			if count[p.Y] > peakN {
				peak, peakN = p.Y, count[p.Y]
			}
		}
		var keep []motion.Pt
		for _, p := range src {
			if abs(p.Y-peak) <= o.Band {
				keep = append(keep, p)
			}
		}
		src = keep
	}
	blank := sprite.Frame{Art: make([]string, h)}
	for i := range blank.Art {
		blank.Art[i] = strings.Repeat(".", w)
	}
	if len(src) == 0 {
		// a role the object does not carry is a no-op, and says so. it
		// must not be an error or a library of effects is unusable
		// across a mixed set of objects, and it must not be silent or a
		// misspelled role looks like a subtle effect.
		fmt.Fprintf(os.Stderr, "glow: roles %q matched 0 pixels\n", o.Source)
		return blank
	}

	var cx, cy float64
	for _, p := range src {
		cx, cy = cx+float64(p.X), cy+float64(p.Y)
	}
	cx, cy = cx/float64(len(src)), cy/float64(len(src))

	gain := o.Gain
	if gain == 0 {
		gain = 1
	}
	out := make([]string, h)
	for y := 0; y < h; y++ {
		row := make([]byte, w)
		for x := 0; x < w; x++ {
			// a daffy pixel is one column by half a row, which is
			// square, so no aspect correction belongs anywhere in
			// here
			d := math.Inf(1)
			for _, p := range src {
				d = math.Min(d, math.Hypot(float64(x-p.X), float64(y-p.Y)))
			}
			d /= o.Radius
			v := gain / (1 + 2.4*d*d)
			if o.Rays > 0 && o.RayGain > 0 {
				rd := math.Hypot(float64(x)-cx, float64(y)-cy) / o.Radius
				if rd >= o.RayInner {
					a := math.Atan2(float64(y)-cy, float64(x)-cx)
					lobe := math.Pow(math.Abs(math.Cos(float64(o.Rays)*0.5*(a-o.RayTurn))), 9)
					v += o.RayGain * gain * lobe *
						math.Min(1, (rd-o.RayInner)*1.5) / (1 + o.RayReach*rd*rd)
				}
			}
			row[x] = Quantise(v)
		}
		out[y] = string(row)
	}
	return sprite.Frame{Art: out}
}

// Sweep is a band of light crossing the canvas, `at` running from below nought
// to above one so it starts and finishes off the object. The chrome sweep every
// bulletin board in the world was running in 1993, and it belongs beside Glow
// because it is the same kind of thing:
//
// a scalar field, quantised to the ladder, applied by the same compositor. What
// is underneath is not consulted.
func Sweep(w, h int, deg, width, soft, at, gain float64) sprite.Frame {
	if gain == 0 {
		gain = 1
	}
	a := deg * math.Pi / 180
	ux, uy := math.Cos(a), math.Sin(a)
	out := make([]string, h)
	for y := 0; y < h; y++ {
		row := make([]byte, w)
		for x := 0; x < w; x++ {
			u := float64(x)/math.Max(1, float64(w-1)) - 0.5
			v := float64(y)/math.Max(1, float64(h-1)) - 0.5
			t := u*ux + v*uy + 0.5
			d := math.Abs(t - at)
			s := 0.0
			switch {
			case d <= width/2:
				s = 1
			case d < width/2+soft:
				s = 1 - (d-width/2)/soft
			}
			row[x] = Quantise(gain * s * s)
		}
		out[y] = string(row)
	}
	return sprite.Frame{Art: out}
}

// ClipTo keeps an effect only where the object carries one of these roles. A
// sweep across type wants to be on the letters, not on the night behind them; a
// halo does not. This is the difference between the two, and it is one line at
// the call site rather than a second effect.
func ClipTo(alpha, base sprite.Frame, keep string, grow int) sprite.Frame {
	h, w := alpha.Height(), alpha.Width()
	on := make([][]bool, h)
	for i := range on {
		on[i] = make([]bool, w)
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if strings.IndexByte(keep, motion.At(base, x, y)) < 0 {
				continue
			}
			for dy := -grow; dy <= grow; dy++ {
				for dx := -grow; dx <= grow; dx++ {
					if y+dy >= 0 && y+dy < h && x+dx >= 0 && x+dx < w {
						on[y+dy][x+dx] = true
					}
				}
			}
		}
	}
	out := make([]string, h)
	for y := 0; y < h; y++ {
		row := make([]byte, w)
		for x := 0; x < w; x++ {
			if on[y][x] {
				row[x] = alpha.Art[y][x]
			} else {
				row[x] = '.'
			}
		}
		out[y] = string(row)
	}
	return sprite.Frame{Art: out, Hold: alpha.Hold}
}

// abs is the absolute value of an int.
func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
