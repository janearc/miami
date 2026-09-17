package light

import (
	"math"
	"testing"

	"github.com/janearc/libdaffy/field"

	"github.com/janearc/miami/grade"
)

// A stack painted as light and brought down gives the picture grade paints
// directly, to within a step, for colours the display can show.
//
// Within a step and not exactly, because the two round at different places:
// grade encodes each colour once from float64, and this goes through float32
// and a table.
func TestPaintMatchesGradeInGamut(t *testing.T) {
	sh := grade.Sextants(40, 12)
	st := grade.Stack{{
		Name:  "sky",
		Field: field.Linear{B: field.Point{U: 1, V: 1}},
		Ramp:  grade.Ramp(0.0, "120631", 0.55, "c63f6b", 1.0, "ffd08a"),
		Gain:  1,
	}, {
		Name:  "sun",
		Field: grade.Flat(0),
		Ramp:  grade.Ramp(0.0, "fff6cc", 1.0, "fff6cc"),
		Alpha: grade.Inside(field.Radial{
			Centre: field.Point{U: 0.5, V: 0.5}, Radius: 0.2}, 0.8, 0.2),
		Gain: 1,
	}}
	want := sh.Paint(st)
	b := New(sh.W, sh.H)
	b.Paint(sh, st)
	got := b.RGBA(1)
	worst := 0
	for y := range want {
		for x := range want[y] {
			for _, d := range []int{
				int(got[y][x].R) - int(want[y][x].R),
				int(got[y][x].G) - int(want[y][x].G),
				int(got[y][x].B) - int(want[y][x].B),
			} {
				worst = max(worst, abs(d))
			}
		}
	}
	if worst > 1 {
		t.Fatalf("light differs from grade by %d steps", worst)
	}
}

// Light too bright to show keeps its hue and its luminance on the way to
// white, rather than clipping a channel at a time and turning yellow.
//
// Hue is checked as (G-B)/(R-B). Moving a colour toward a grey leaves that
// ratio exactly where it was, and clipping one channel moves it, so it tells
// the two apart where "still looks orange" would not.
func TestTooBrightGoesWhiteNotYellow(t *testing.T) {
	orange := Hex("ff7a1a")
	hue := func(c RGB) float64 { return (c.G - c.B) / (c.R - c.B) }
	for _, k := range []float64{1.5, 2.5} { // luminance .53 and .88
		in := orange.Scale(k)
		c := fit(in)
		if math.Abs(hue(c)-hue(orange)) > 1e-9 {
			t.Errorf("x%.1f: hue moved from %.4f to %.4f",
				k, hue(orange), hue(c))
		}
		if math.Abs(c.Y()-in.Y()) > 1e-9 {
			t.Errorf("x%.1f: luminance %v became %v",
				k, in.Y(), c.Y())
		}
		if max(c.R, c.G, c.B) > 1+1e-9 {
			t.Errorf("x%.1f: still out of range: %v", k, c)
		}
	}
	if c := Display(orange.Scale(40)); c.R != 255 || c.G != 255 ||
		c.B != 255 {
		t.Errorf("luminance far past one is not white: %v", c)
	}
}

// Hotter is bluer: blue against red rises from 1850K through 6500K to 30000K,
// and all three come out at a luminance of one.
//
// Blue over red and not red over blue. A candle is a little outside what the
// lamps can mix, so its blue is just below nought, and a ratio with that
// underneath it changes sign.
func TestKelvinRunsRedToBlue(t *testing.T) {
	warm, noon, hot := Kelvin(1850), Kelvin(6500), Kelvin(30000)
	if !(warm.B/warm.R < noon.B/noon.R && noon.B/noon.R < hot.B/hot.R) {
		t.Errorf("not red to blue: %v %v %v", warm, noon, hot)
	}
	for _, c := range []RGB{warm, noon, hot} {
		if math.Abs(c.Y()-1) > 1e-9 {
			t.Errorf("luminance %v, want 1", c.Y())
		}
	}
}

// Bloom spreads light and leaves dark alone: a single bright subpixel lights
// its neighbours, nothing below the threshold spreads, and the far corner of
// the frame stays black.
func TestBloomSpreadsOnlyTheBright(t *testing.T) {
	b := New(80, 60)
	b.Set(40, 30, RGB{50, 50, 50})
	b.Set(10, 10, RGB{0.2, 0.2, 0.2}) // under the threshold
	b.Bloom(1, Glow{Radius: 3, Strength: 1})
	if !(b.At(44, 30).R > 0 && b.At(40, 33).R > 0) {
		t.Errorf("no light reached the neighbours: %v %v",
			b.At(44, 30), b.At(40, 33))
	}
	if b.At(12, 10).R != 0 {
		t.Errorf("light under the threshold spread: %v", b.At(12, 10))
	}
	if c := Display(b.At(79, 59)); c.R != 0 || c.G != 0 || c.B != 0 {
		t.Errorf("the far corner lit up: %v", b.At(79, 59))
	}
}

// abs is an int's size.
func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
