package scenes

import (
	"image/color"
	"math"

	"github.com/janearc/miami/canvas"
	"github.com/janearc/miami/cells"
	"github.com/janearc/miami/geom"
	"github.com/janearc/miami/motifs"
	"github.com/janearc/miami/paint"
	"github.com/janearc/miami/raster"
	"github.com/janearc/miami/typeset"
)

// "let there be lights": a nineteen-fifties roadside church sign, on a pole, at
// dusk, with a light bulb where the cross would be. it is a manpage wide and as
// tall as the descent from sky to road takes.

// the roles that are read, for the legibility gate
var LightsText = []byte{'L', 's', 'b', 'x'}

var finePrint = color.RGBA{202, 194, 182, 255}

// setFit sets a line no wider than maxW and no taller than maxCap.
func setFit(l typeset.Line, cx, base, maxW, maxCap float64) ([]geom.Shape, float64) {
	l.CapHeight = math.Min(maxCap, fitCap(l, maxW))
	return l.Set(cx, base), l.CapHeight
}

// Lights returns the scene and the literal marks laid over its cells.
func Lights(cv canvas.Canvas) (*raster.Scene, []cells.Mark) {
	u := cv.W()
	cx := u / 2
	// horizon, road and page end are set after the board, which decides how
	// far everything below it has to move

	copper := mustFace(supp+"Copperplate.ttc", "Copperplate Light")
	script := mustFace(supp+"SignPainter.ttc", "SignPainter-HouseScript")
	gothic := mustFace(supp+"Futura.ttc", "Futura Condensed Medium")

	// ---- the bulb, on an iron bracket, with its burst
	b := motifs.NewBulb(cx, 20, 52)
	burst := motifs.Sunburst(b.Centre, b.R*1.42,
		[]float64{b.R * 2.25, b.R * 1.62, b.R * 1.9, b.R * 1.62}, 28, -208, 28, b.R*0.07)
	// a plain stem and a diamond finial where it meets the crown. scroll
	// curls at this size read as a moustache.
	bracket := geom.Union(
		geom.Taper{P: []geom.Vec{{X: cx, Y: b.Bottom - 0.5}, {X: cx, Y: 88}}, R: []float64{0.9, 1.3}},
		geom.Poly{P: geom.Pts(geom.Vec{X: cx, Y: 79.5}, geom.Vec{X: cx + 2.6, Y: 83}, geom.Vec{X: cx, Y: 86.5}, geom.Vec{X: cx - 2.6, Y: 83})},
	)

	// ---- the name panel
	panel := motifs.ArchPanel{X0: 7, X1: u - 7, Crown: 86, Spring: 102, Bottom: 196}
	frame := panel.Shape(0)
	face := panel.Shape(2.6)
	var marks []cells.Mark
	bulbGlyph := func(pts []geom.Vec, bg color.RGBA) {
		for _, p := range pts {
			bgc := bg
			marks = append(marks, cells.Mark{X: p.X, Y: p.Y, Glyph: '●', Fg: color.RGBA{255, 222, 140, 255}, Bg: &bgc})
		}
	}
	// spacing of 4 is exactly two rows down the sides; 3 alternated one row
	// and two, and the bulbs clumped in pairs
	bulbGlyph(panel.Border(1.3, 4.0), color.RGBA{22, 88, 100, 255})

	wob := func(seed uint64, tilt float64) typeset.Wobble {
		return typeset.Wobble{Tilt: tilt, Shift: 0.06, Scale: 0.035, Space: 0.05, Seed: seed}
	}
	var name []geom.Shape
	// one word of script, big: a phrase of it at this size was scribble
	welcome, _ := setFit(typeset.Line{Text: "Welcome", Face: script, Arch: 1.5,
		Wobble: typeset.Wobble{Tilt: 1.5, Shift: 0.03, Seed: 3}}, cx, 115, 34, 16)
	// tracking held near the standard's .05em; the wobble supplies the
	// hand-set looseness that wide tracking was faking
	for _, ln := range []struct {
		text     string
		base, w  float64
		cap      float64
		seed     uint64
		tracking float64
		condense float64
	}{
		{"LET", 132, 30, 10.0, 21, 0.05, 1},
		{"THERE", 148, 50, 10.0, 22, 0.05, 1},
		{"BE", 162.5, 20, 9.0, 23, 0.06, 1},
		// the word that matters, condensed so it can stand tallest
		{"LIGHTS", 188.5, 53, 17.5, 24, 0.04, 0.64},
	} {
		s, _ := setFit(typeset.Line{Text: ln.text, Face: copper, Tracking: ln.tracking,
			Condense: ln.condense, Wobble: wob(ln.seed, 3.2)}, cx, ln.base, ln.w, ln.cap)
		name = append(name, s...)
	}
	// rules with a diamond either side of BE: drawn, not a font, and period
	flourish := func(side float64) geom.Shape {
		x := cx + side*12
		return geom.Union(
			geom.Taper{P: []geom.Vec{{X: x, Y: 158}, {X: x + side*14, Y: 158}}, R: []float64{0.55, 0.2}},
			geom.Poly{P: geom.Pts(geom.Vec{X: x, Y: 155.2}, geom.Vec{X: x + side*2.2, Y: 158}, geom.Vec{X: x, Y: 160.8}, geom.Vec{X: x - side*2.2, Y: 158})},
		)
	}
	ornament := geom.Union(flourish(-1), flourish(1))

	// ---- the changeable letter board, hung below on two struts. three
	// short lines rather than two long ones: at two lines the caps were
	// under four rows and a real terminal read ALL BULBS WELCOME as ALL
	// BULIS NELCOME.
	const boardTop, pitch = 203.0, 17.0
	boardLines := []string{"ALL BULBS", "WELCOME", "HUE & NANOLEAF"}
	// the board ends a little below its last rail, not a whole empty line
	// later
	probeCap := 10.5
	boardBot := boardTop + 4 + probeCap + pitch*float64(len(boardLines)-1) + 7
	struts := geom.Union(
		geom.Poly{P: geom.Pts(geom.Vec{X: 19, Y: 195}, geom.Vec{X: 22, Y: 195}, geom.Vec{X: 22, Y: boardTop + 1}, geom.Vec{X: 19, Y: boardTop + 1})},
		geom.Poly{P: geom.Pts(geom.Vec{X: u - 22, Y: 195}, geom.Vec{X: u - 19, Y: 195}, geom.Vec{X: u - 19, Y: boardTop + 1}, geom.Vec{X: u - 22, Y: boardTop + 1})},
	)
	board := geom.Poly{P: geom.Pts(geom.Vec{X: 7, Y: boardTop}, geom.Vec{X: u - 7, Y: boardTop}, geom.Vec{X: u - 7, Y: boardBot}, geom.Vec{X: 7, Y: boardBot})}
	boardFace := geom.Poly{P: geom.Pts(geom.Vec{X: 9.2, Y: boardTop + 2.2}, geom.Vec{X: u - 9.2, Y: boardTop + 2.2}, geom.Vec{X: u - 9.2, Y: boardBot - 2.2}, geom.Vec{X: 9.2, Y: boardBot - 2.2})}
	var rails []geom.Shape
	var boardText []geom.Shape
	// the widest line sets one cap height for all three, as a real board's
	// letters are all one size
	probe := typeset.Line{Text: boardLines[len(boardLines)-1], Face: gothic, Tracking: 0.08}
	boardCap := math.Min(10.5, fitCap(probe, 56))
	for i, t := range boardLines {
		base := boardTop + 4 + boardCap + float64(i)*pitch
		rails = append(rails, geom.Taper{P: []geom.Vec{{X: 10, Y: base + 2.7}, {X: u - 10, Y: base + 2.7}}, R: []float64{0.28, 0.28}})
		l := typeset.Line{Text: t, Face: gothic, Tracking: 0.08, Flat: true, CapHeight: boardCap,
			Wobble: typeset.Wobble{Tilt: 3, Shift: 0.05, Scale: 0.03, Space: 0.06, Seed: uint64(40 + i)}}
		boardText = append(boardText, l.Set(cx, base)...)
	}
	drop := boardBot - 234 // everything below the board moves down by this much

	// ---- pole and arrow
	horizon := 290.0 + drop
	h := 366.0 + drop
	roadEnd := 346.0 + drop
	// the pole runs past the horizon into the near verge. stopping it at
	// the horizon planted the sign at the vanishing point, miles off.
	footY := horizon + 26
	pole := geom.Union(
		geom.Poly{P: geom.Pts(geom.Vec{X: cx - 2.4, Y: boardBot - 1}, geom.Vec{X: cx + 2.4, Y: boardBot - 1}, geom.Vec{X: cx + 3.0, Y: footY}, geom.Vec{X: cx - 3.0, Y: footY})},
		geom.Poly{P: geom.Pts(geom.Vec{X: cx - 6, Y: footY - 1}, geom.Vec{X: cx + 6, Y: footY - 1}, geom.Vec{X: cx + 7, Y: footY + 2.5}, geom.Vec{X: cx - 7, Y: footY + 2.5})},
	)
	// shaft tall enough that the chaser bulbs clear the letters by a full
	// row
	arrow := motifs.Arrow{X0: 10, X1: u - 5, Y: 256 + drop, Shaft: 16, Head: 28, HeadLen: 14}
	bulbGlyph(arrow.Lights(1.6, 4.0), color.RGBA{110, 20, 26, 255})
	exit, _ := setFit(typeset.Line{Text: "NEXT EXIT", Face: gothic, Tracking: 0.08, Weight: 0.13, Flat: true,
		Wobble: typeset.Wobble{Tilt: 2, Shift: 0.04, Space: 0.04, Seed: 51}}, cx-4, 260.2+drop, 40, 7.8)

	// ---- the interstate, from the horizon toward the reader
	vp := geom.Vec{X: cx + 12, Y: horizon}
	roadL, roadR := cx-8, u+36
	road := geom.Poly{P: geom.Pts(geom.Vec{X: vp.X - 1, Y: horizon}, geom.Vec{X: vp.X + 1, Y: horizon}, geom.Vec{X: roadR, Y: roadEnd}, geom.Vec{X: roadL, Y: roadEnd})}
	var dashes []geom.Shape
	for i := 0; i < 9; i++ {
		// perspective: equal steps in depth are unequal steps on screen
		t0 := math.Pow(float64(i)/9, 1.9)
		t1 := math.Pow((float64(i)+0.45)/9, 1.9)
		y0, y1 := horizon+(roadEnd-horizon)*t0, horizon+(roadEnd-horizon)*t1
		xAt := func(y float64) float64 { return vp.X + (y-horizon)/(roadEnd-horizon)*((roadL+roadR)/2-vp.X) }
		dashes = append(dashes, geom.Taper{P: []geom.Vec{{X: xAt(y0), Y: y0}, {X: xAt(y1), Y: y1}},
			R: []float64{0.25 + 0.9*t0, 0.25 + 0.9*t1}})
	}

	// ---- stars thin out as the sky warms toward the horizon
	var stars []geom.Shape
	for _, s := range [][3]float64{{6, 8, 3.2}, {71, 13, 3.6}, {5, 62, 2.4}, {72, 88, 2.4}} {
		stars = append(stars, motifs.Sparkle(geom.Vec{X: s[0], Y: s[1]}, s[2]))
	}
	for _, p := range [][2]float64{{14, 44}, {66, 52}, {68, 3}, {27, 4}, {50, 2.5}, {3, 118}, {74, 136}, {2.5, 160}, {12, 26}, {61, 34}} {
		stars = append(stars, geom.Ellipse{C: geom.Vec{X: p[0], Y: p[1]}, Rx: 0.55, Ry: 0.55})
	}

	sc := &raster.Scene{W: u, H: h}
	sc.Parts = []raster.Part{
		{Role: 'D', Shape: road},
		{Role: 'd', Shape: geom.Union(dashes...)},
		{Role: 'k', Shape: geom.Union(stars...)},
		{Role: 'r', Shape: burst},
		{Role: 'q', Shape: bracket},
		{Role: 'P', Shape: pole},
		{Role: 't', Shape: struts},
		{Role: 'F', Shape: frame},
		{Role: 'n', Shape: face},
		{Role: 's', Shape: geom.Union(welcome...)},
		{Role: 'o', Shape: ornament},
		{Role: 'L', Shape: geom.Union(name...)},
		{Role: 'B', Shape: board},
		{Role: 'W', Shape: boardFace},
		{Role: 'R', Shape: geom.Union(rails...)},
		{Role: 'b', Shape: geom.Union(boardText...)},
		{Role: 'A', Shape: arrow.Shape()},
		{Role: 'x', Shape: geom.Union(exit...)},
		{Role: 'G', Shape: b.Glass},
		{Role: 'w', Shape: b.Wires},
		{Role: 'h', Shape: b.Highlight},
		{Role: 'f', Shape: b.Filament},
		{Role: 'S', Shape: b.Screw},
		{Role: 'g', Shape: b.Grooves},
		{Role: 'I', Shape: b.Insulator},
		{Role: 'T', Shape: b.Tip}}
	sc.Pal = map[byte]color.RGBA{
		// bulb
		'G': {255, 240, 190, 255}, 'w': {214, 150, 60, 255}, 'h': {255, 255, 250, 255},
		'f': {255, 255, 255, 255}, 'S': {178, 168, 150, 255}, 'g': {92, 84, 80, 255},
		'I': {40, 34, 40, 255}, 'T': {120, 110, 100, 255}, 'r': {246, 206, 128, 255},
		'q': {34, 30, 42, 255}, 'k': {232, 228, 255, 255},
		// panel
		'F': {22, 88, 100, 255}, 'n': {238, 226, 196, 255}, 'm': {255, 222, 140, 255},
		'L': {96, 16, 20, 255}, 's': {24, 36, 84, 255}, 'o': {22, 88, 100, 255},
		// board
		't': {30, 28, 40, 255}, 'B': {22, 88, 100, 255}, 'W': {236, 234, 226, 255},
		'R': {170, 168, 170, 255}, 'b': {46, 44, 52, 255},
		// pole, arrow, road
		'P': {30, 28, 40, 255}, 'A': {110, 20, 26, 255}, 'a': {255, 214, 130, 255},
		'x': {250, 244, 230, 255}, 'D': {36, 32, 44, 255}, 'd': {226, 214, 160, 255},
	}
	sc.MinCover = map[byte]float64{
		'f': 0.25, 'g': 0.30, 'w': 0.35, 'r': 0.30, 'k': 0.30, 'q': 0.35,
		'L': 0.30, 's': 0.30, 'b': 0.30, 'x': 0.30, 'R': 0.35, 'o': 0.35, 'd': 0.35,
	}

	glass := paint.Radial(b.Centre.X, b.Centre.Y-0.1*b.R, b.R*1.25, []paint.Stop{
		{At: 0.00, C: color.RGBA{255, 252, 232, 255}},
		{At: 0.55, C: color.RGBA{255, 228, 160, 255}},
		{At: 1.00, C: color.RGBA{246, 176, 84, 255}}})
	rayRamp := []paint.Stop{
		{At: 0.00, C: color.RGBA{255, 232, 170, 255}},
		{At: 1.00, C: color.RGBA{226, 144, 78, 255}}}
	sky := []paint.Stop{
		{At: 0.00, C: color.RGBA{8, 8, 30, 255}},
		{At: 0.35, C: color.RGBA{24, 20, 62, 255}},
		{At: 0.62, C: color.RGBA{52, 32, 90, 255}},
		{At: 0.84, C: color.RGBA{112, 54, 104, 255}},
		{At: 0.95, C: color.RGBA{190, 96, 84, 255}},
		{At: 1.00, C: color.RGBA{228, 138, 84, 255}}}
	ground := []paint.Stop{
		{At: 0.00, C: color.RGBA{62, 38, 58, 255}},
		{At: 0.20, C: color.RGBA{34, 24, 42, 255}},
		{At: 1.00, C: color.RGBA{14, 10, 20, 255}}}
	// fine print: real characters in real cells, sentence case, from the
	// package's own description. held to the band against the ground under
	// it.
	footer := color.RGBA{14, 10, 20, 255}
	for i, line := range []string{
		"lights: one abstraction over Philips Hue and Nanoleaf --",
		"colour, gradient shape and animation intent.",
	} {
		y := 352.0 + drop + float64(i)*cv.RowsToPhys(1)
		for j, r := range line {
			if r == ' ' {
				continue
			}
			bg := footer
			marks = append(marks, cells.Mark{X: 5 + float64(j) + 0.5, Y: y + 1, Glyph: r, Fg: finePrint, Bg: &bg})
		}
	}
	// the plain text palette. the sky is empty, which is the negative
	// space; dark structure against it is a light shade so the pole still
	// holds the sign up; letters are knocked out of full panels.
	//
	// ghostty draws the shade glyphs as flat greys beside full-bright
	// sextants, so a part is ink or paper unless its grey is the point:
	// rays at 0.75 broke into two-tone segments in a real terminal.
	sc.Tone = map[byte]float64{
		'G': 1, 'f': 0, 'w': 0, 'h': 1, 'S': 1, 'g': 0, 'I': 0.25, 'T': 0.5,
		'r': 1, 'q': 0.5, 'k': 1,
		'F': 0.5, 'n': 1, 'L': 0, 's': 0, 'o': 0,
		't': 0.25, 'B': 0.5, 'W': 1, 'R': 0.5, 'b': 0,
		'P': 0.25, 'A': 0.25, 'x': 1, 'D': 0.12, 'd': 1,
	}
	sc.ToneBg = func(x, y float64) float64 {
		if y < horizon {
			// one thin band of tone right at the horizon; a deeper
			// one read as a heavy bar across the page in a real
			// terminal
			return math.Max(0, (y-(horizon-5))/5) * 0.3
		}
		return 0
	}
	sc.Bg = func(fx, fy float64) color.RGBA {
		y := fy * h
		if y < horizon {
			return paint.Ramp(sky, y/horizon)
		}
		return paint.Ramp(ground, (y-horizon)/(h-horizon))
	}
	sc.Shade = func(role byte, x, y float64, c color.RGBA) color.RGBA {
		dx, dy := x-b.Centre.X, y-b.Centre.Y
		d := math.Sqrt(dx*dx+dy*dy) / b.R
		switch role {
		case 'G':
			c = glass(x, y)
		case 'r':
			c = paint.Ramp(rayRamp, (d-1.42)/0.85)
		case 'D':
			// asphalt picks up the last of the sky near the horizon
			c = paint.Lerp(color.RGBA{96, 62, 74, 255}, c, (y-horizon)/18)
		}
		if role != 'G' && role != 'f' && role != 'h' {
			e := math.Max(d-1, 0)
			c = paint.Screen(c, color.RGBA{255, 190, 105, 255}, 0.62/(1+3.2*e*e))
		}
		// the road runs out into the dark rather than being cut off
		if y > roadEnd-22 {
			c = paint.Lerp(c, footer, (y-(roadEnd-22))/22)
		}
		return c
	}
	return sc, marks
}
